package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/git"
	"github.com/stackitcloud/stackit-sdk-go/services/git/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	gitUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/git/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitResource{}
	_ resource.ResourceWithConfigure   = &gitResource{}
	_ resource.ResourceWithImportState = &gitResource{}
)

// Model represents the schema for the git resource.
type Model struct {
	Id                    types.String `tfsdk:"id"` // Required by Terraform
	ACL                   types.List   `tfsdk:"acl"`
	ConsumedDisk          types.String `tfsdk:"consumed_disk"`
	ConsumedObjectStorage types.String `tfsdk:"consumed_object_storage"`
	Created               types.String `tfsdk:"created"`
	Flavor                types.String `tfsdk:"flavor"`
	InstanceId            types.String `tfsdk:"instance_id"`
	Name                  types.String `tfsdk:"name"`
	ProjectId             types.String `tfsdk:"project_id"`
	Url                   types.String `tfsdk:"url"`
	Version               types.String `tfsdk:"version"`
}

// NewGitResource is a helper function to create a new git resource instance.
func NewGitResource() resource.Resource {
	return &gitResource{}
}

// gitResource implements the resource interface for git instances.
type gitResource struct {
	client *git.APIClient
}

// descriptions for the attributes in the Schema
var descriptions = map[string]string{
	"id":                      "Terraform's internal resource ID, structured as \"`project_id`,`instance_id`\".",
	"acl":                     "Restricted ACL for instance access.",
	"consumed_disk":           "How many bytes of disk space is consumed.",
	"consumed_object_storage": "How many bytes of Object Storage is consumed.",
	"created":                 "Instance creation timestamp in RFC3339 format.",
	"flavor":                  "Instance flavor. If not provided, defaults to git-100. For a list of available flavors, refer to our API documentation: `https://docs.api.stackit.cloud/documentation/git/version/v1beta`",
	"instance_id":             "ID linked to the git instance.",
	"name":                    "Unique name linked to the git instance.",
	"project_id":              "STACKIT project ID to which the git instance is associated.",
	"url":                     "Url linked to the git instance.",
	"version":                 "Version linked to the git instance.",
}

// Configure sets up the API client for the git instance resource.
func (g *gitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_git", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := gitUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	g.client = apiClient
	tflog.Info(ctx, "git client configured")
}

// Metadata sets the resource type name for the git instance resource.
func (g *gitResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_git"
}

// Schema defines the schema for the resource.
func (g *gitResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(
			"%s %s",
			features.AddBetaDescription("Git Instance resource schema.", core.Resource),
			"This resource currently does not support updates. Changing the ACLs, flavor, or name will trigger resource recreation. Update functionality will be added soon. In the meantime, please proceed with caution. To update these attributes, please open a support ticket.",
		),
		Description: "Git Instance resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"acl": schema.ListAttribute{
				Description: descriptions["acl"],
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"consumed_disk": schema.StringAttribute{
				Description: descriptions["consumed_disk"],
				Computed:    true,
			},
			"consumed_object_storage": schema.StringAttribute{
				Description: descriptions["consumed_object_storage"],
				Computed:    true,
			},
			"created": schema.StringAttribute{
				Description: descriptions["created"],
				Computed:    true,
			},
			"flavor": schema.StringAttribute{
				Description: descriptions["flavor"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Optional: true,
				Computed: true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(5, 32),
				},
			},
			"url": schema.StringAttribute{
				Description: descriptions["url"],
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state for the git instance.
func (g *gitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the planned values for the resource.
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	// Set logging context with the project ID and instance ID.
	projectId := model.ProjectId.ValueString()
	instanceName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_name", instanceName)

	payload, diags := toCreatePayload(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the new git instance via the API client.
	gitInstanceResp, err := g.client.CreateInstance(ctx, projectId).
		CreateInstancePayload(payload).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	gitInstanceId := *gitInstanceResp.Id
	_, err = wait.CreateGitInstanceWaitHandler(ctx, g.client, projectId, gitInstanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating git instance", fmt.Sprintf("Git instance creation waiting: %v", err))
		return
	}

	err = mapFields(ctx, gitInstanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating git instance", fmt.Sprintf("Mapping fields: %v", err))
		return
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Git Instance created")
}

// Read refreshes the Terraform state with the latest git instance data.
func (g *gitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	// Extract the project ID and instance id of the model
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()

	// Read the current git instance via id
	gitInstanceResp, err := g.client.GetInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, gitInstanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading git instance", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, fmt.Sprintf("read git instance %s", instanceId))
}

// Update attempts to update the resource. In this case, git instances cannot be updated.
// Note: This method is intentionally left without update logic because changes
// to 'project_id' or 'name' require the resource to be entirely replaced.
// As a result, the Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (g *gitResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// git instances cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating git instance", "Git Instance can't be updated")
}

// Delete deletes the git instance and removes it from the Terraform state on success.
func (g *gitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// Call API to delete the existing git instance.
	err := g.client.DeleteInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteGitInstanceWaitHandler(ctx, g.client, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error waiting for instance deletion", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Git instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (g *gitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Split the import identifier to extract project ID and email.
	idParts := strings.Split(req.ID, core.Separator)

	// Ensure the import identifier format is correct.
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing git instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	instanceId := idParts[1]

	// Set the project ID and instance ID attributes in the state.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceId)...)
	tflog.Info(ctx, "Git instance state imported")
}

// mapFields maps a Git response to the model.
func mapFields(ctx context.Context, resp *git.Instance, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if resp.Id == nil {
		return fmt.Errorf("git instance id not present")
	}

	aclList := types.ListNull(types.StringType)
	var diags diag.Diagnostics
	if resp.Acl != nil && len(*resp.Acl) > 0 {
		aclList, diags = types.ListValueFrom(ctx, types.StringType, resp.Acl)
		if diags.HasError() {
			return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
		}
	}

	model.Created = types.StringNull()
	if resp.Created != nil && resp.Created.String() != "" {
		model.Created = types.StringValue(resp.Created.String())
	}

	// Build the ID by combining the project ID and instance id and assign the model's fields.
	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), *resp.Id)
	model.ACL = aclList
	model.ConsumedDisk = types.StringPointerValue(resp.ConsumedDisk)
	model.ConsumedObjectStorage = types.StringPointerValue(resp.ConsumedObjectStorage)
	model.Flavor = types.StringPointerValue(resp.Flavor)
	model.InstanceId = types.StringPointerValue(resp.Id)
	model.Name = types.StringPointerValue(resp.Name)
	model.Url = types.StringPointerValue(resp.Url)
	model.Version = types.StringPointerValue(resp.Version)

	return nil
}

// toCreatePayload creates the payload to create a git instance
func toCreatePayload(ctx context.Context, model *Model) (git.CreateInstancePayload, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if model == nil {
		return git.CreateInstancePayload{}, diags
	}

	payload := git.CreateInstancePayload{
		Name: model.Name.ValueStringPointer(),
	}

	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		var acl []string
		aclDiags := model.ACL.ElementsAs(ctx, &acl, false)
		diags.Append(aclDiags...)
		if !aclDiags.HasError() {
			payload.Acl = &acl
		}
	}

	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		payload.Flavor = git.CreateInstancePayloadGetFlavorAttributeType(model.Flavor.ValueStringPointer())
	}

	return payload, diags
}
