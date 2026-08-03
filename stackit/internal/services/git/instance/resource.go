package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	git "github.com/stackitcloud/stackit-sdk-go/services/git/v1betaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/git/v1betaapi/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
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

// Default to an open access-control-list unless otherwise specified
var defaultAclValue = types.ListValueMust(
	types.StringType,
	[]attr.Value{types.StringValue("0.0.0.0/0")},
)

// Model represents the schema for the git resource.
type Model struct {
	Id types.String `tfsdk:"id"` // Required by Terraform

	ProjectId  types.String `tfsdk:"project_id"`
	InstanceId types.String `tfsdk:"instance_id"`

	// Requires replacement on change
	Name   types.String `tfsdk:"name"`
	Flavor types.String `tfsdk:"flavor"`

	// Updateable fields
	ACL types.List `tfsdk:"acl"`

	// Read-only fields
	Created               types.String `tfsdk:"created"`
	Url                   types.String `tfsdk:"url"`
	ConsumedDisk          types.String `tfsdk:"consumed_disk"`
	ConsumedObjectStorage types.String `tfsdk:"consumed_object_storage"`
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
	"main":                    "Git Instance resource schema.",
	"id":                      "Terraform's internal resource ID, structured as \"`project_id`,`instance_id`\".",
	"project_id":              "STACKIT project ID to which the git instance is associated.",
	"instance_id":             "ID linked to the git instance.",
	"name":                    "Unique name linked to the git instance.",
	"flavor":                  "Instance flavor. If not provided, defaults to git-100. For a list of available flavors, refer to our API documentation: `https://docs.api.stackit.cloud/documentation/git/version/v1beta`",
	"created":                 "Instance creation timestamp in RFC3339 format.",
	"url":                     "Url linked to the git instance.",
	"acl":                     "Restricted ACL for instance access.",
	"consumed_disk":           "How many bytes of disk space is consumed.",
	"consumed_object_storage": "How many bytes of Object Storage is consumed.",
	"version":                 "Version linked to the git instance.",
}

// Configure sets up the API client for the git instance resource.
func (g *gitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

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
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"acl": schema.ListAttribute{
				Description: descriptions["acl"],
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(defaultAclValue),
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
	gitInstanceResp, err := g.client.DefaultAPI.CreateInstance(ctx, projectId).
		CreateInstancePayload(payload).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	gitInstanceId := gitInstanceResp.Id
	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"instance_id": gitInstanceId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	_, err = wait.CreateGitInstanceWaitHandler(ctx, g.client.DefaultAPI, projectId, gitInstanceId).WaitWithContext(ctx)
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
	if instanceId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}

	// Read the current git instance via id
	gitInstanceResp, err := g.client.DefaultAPI.GetInstance(ctx, projectId, instanceId).Execute()
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

// Updates the git instance.
func (g *gitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the planned values for the resource.
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	payload, diags := toPatchPayload(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "updating instance", map[string]interface{}{
		"project_id": projectId,
		"instanceId": instanceId,
		"payload":    payload,
	})

	gitInstanceResp, err := g.client.DefaultAPI.PatchInstance(ctx, projectId, instanceId).
		PatchInstancePayload(payload).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Wait for update
	_, err = wait.UpdateGitInstanceWaitHandler(ctx, g.client.DefaultAPI, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating git instance", fmt.Sprintf("Git instance update waiting: %v", err))
		return
	}

	err = mapFields(ctx, gitInstanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating git instance", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Git instance updated")
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
	err := g.client.DefaultAPI.DeleteInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteGitInstanceWaitHandler(ctx, g.client.DefaultAPI, projectId, instanceId).WaitWithContext(ctx)
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
	// Set the project ID and instance ID attributes in the state.
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"instance_id": idParts[1],
	})
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

	aclList := types.ListNull(types.StringType)
	var diags diag.Diagnostics
	if len(resp.Acl) > 0 {
		aclList, diags = types.ListValueFrom(ctx, types.StringType, resp.Acl)
		if diags.HasError() {
			return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
		}
	}

	model.Created = types.StringNull()
	if resp.Created.String() != "" {
		model.Created = types.StringValue(resp.Created.String())
	}

	// Build the ID by combining the project ID and instance id and assign the model's fields.
	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), resp.Id)
	model.ACL = aclList
	model.ConsumedDisk = types.StringValue(resp.ConsumedDisk)
	model.ConsumedObjectStorage = types.StringValue(resp.ConsumedObjectStorage)
	model.Flavor = types.StringValue(resp.Flavor)
	model.InstanceId = types.StringValue(resp.Id)
	model.Name = types.StringValue(resp.Name)
	model.Url = types.StringValue(resp.Url)
	model.Version = types.StringValue(resp.Version)

	return nil
}

// toCreatePayload creates the payload to create a git instance
func toCreatePayload(ctx context.Context, model *Model) (git.CreateInstancePayload, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if model == nil {
		return git.CreateInstancePayload{}, diags
	}

	payload := git.CreateInstancePayload{
		Name: model.Name.ValueString(),
	}

	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		var acl []string
		aclDiags := model.ACL.ElementsAs(ctx, &acl, false)
		diags.Append(aclDiags...)
		if !aclDiags.HasError() {
			payload.Acl = acl
		}
	}

	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		payload.Flavor = conversion.StringValueToEnumPointer[git.CreateInstancePayloadFlavor](model.Flavor)
	}

	return payload, diags
}

// toPatchPayload creates the payload to update a git instance
func toPatchPayload(ctx context.Context, model *Model) (git.PatchInstancePayload, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if model == nil {
		return git.PatchInstancePayload{}, diags
	}

	payload := git.PatchInstancePayload{}

	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		var acl []string
		aclDiags := model.ACL.ElementsAs(ctx, &acl, false)
		diags.Append(aclDiags...)
		if !aclDiags.HasError() {
			payload.Acl = acl
		}
	}

	return payload, diags
}
