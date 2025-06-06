package objectstorage

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	objectstorageUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/utils"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	coreutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &credentialsGroupResource{}
	_ resource.ResourceWithConfigure   = &credentialsGroupResource{}
	_ resource.ResourceWithImportState = &credentialsGroupResource{}
	_ resource.ResourceWithModifyPlan  = &credentialsGroupResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	CredentialsGroupId types.String `tfsdk:"credentials_group_id"`
	Name               types.String `tfsdk:"name"`
	ProjectId          types.String `tfsdk:"project_id"`
	URN                types.String `tfsdk:"urn"`
	Region             types.String `tfsdk:"region"`
}

// NewCredentialsGroupResource is a helper function to simplify the provider implementation.
func NewCredentialsGroupResource() resource.Resource {
	return &credentialsGroupResource{}
}

// credentialsGroupResource is the resource implementation.
type credentialsGroupResource struct {
	client       *objectstorage.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *credentialsGroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	coreutils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Metadata returns the resource type name.
func (r *credentialsGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_credentials_group"
}

// Configure adds the provider configured client to the resource.
func (r *credentialsGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := objectstorageUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "ObjectStorage credentials group client configured")
}

// Schema defines the schema for the resource.
func (r *credentialsGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                 "ObjectStorage credentials group resource schema. Must have a `region` specified in the provider configuration. If you are creating `credentialsgroup` and `bucket` resources simultaneously, please include the `depends_on` field so that they are created sequentially. This prevents errors from concurrent calls to the service enablement that is done in the background.",
		"id":                   "Terraform's internal data source identifier. It is structured as \"`project_id`,`region`,`credentials_group_id`\".",
		"credentials_group_id": "The credentials group ID",
		"name":                 "The credentials group's display name.",
		"project_id":           "Project ID to which the credentials group is associated.",
		"urn":                  "Credentials group uniform resource name (URN)",
		"region":               "The resource region. If not defined, the provider region is used.",
	}

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
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"credentials_group_id": schema.StringAttribute{
				Description: descriptions["credentials_group_id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"urn": schema.StringAttribute{
				Description: descriptions["urn"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *credentialsGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsGroupName := model.Name.ValueString()
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", credentialsGroupName)
	ctx = tflog.SetField(ctx, "region", region)

	createCredentialsGroupPayload := objectstorage.CreateCredentialsGroupPayload{
		DisplayName: utils.Ptr(credentialsGroupName),
	}

	// Handle project init
	err := enableProject(ctx, &model, region, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentials group", fmt.Sprintf("Enabling object storage project before creation: %v", err))
		return
	}

	// Create new credentials group
	got, err := r.client.CreateCredentialsGroup(ctx, projectId, region).CreateCredentialsGroupPayload(createCredentialsGroupPayload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentials group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(got, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentialsGroup", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage credentials group created")
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialsGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_group_id", credentialsGroupId)
	ctx = tflog.SetField(ctx, "region", region)

	found, err := readCredentialsGroups(ctx, &model, region, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credentialsGroup", fmt.Sprintf("getting credential group from list of credentials groups: %v", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	// update the region manually
	model.Region = types.StringValue(region)

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage credentials group read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *credentialsGroupResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating credentials group", "CredentialsGroup can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *credentialsGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_group_id", credentialsGroupId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing credentials group
	_, err := r.client.DeleteCredentialsGroup(ctx, projectId, region, credentialsGroupId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credentials group", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "ObjectStorage credentials group deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id, credentials_group_id
func (r *credentialsGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing credentialsGroup",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[credentials_group_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credentials_group_id"), idParts[2])...)
	tflog.Info(ctx, "ObjectStorage credentials group state imported")
}

func mapFields(credentialsGroupResp *objectstorage.CreateCredentialsGroupResponse, model *Model, region string) error {
	if credentialsGroupResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if credentialsGroupResp.CredentialsGroup == nil {
		return fmt.Errorf("response credentialsGroup is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	credentialsGroup := credentialsGroupResp.CredentialsGroup

	err := mapCredentialsGroup(*credentialsGroup, model, region)
	if err != nil {
		return err
	}
	model.Region = types.StringValue(region)
	return nil
}

func mapCredentialsGroup(credentialsGroup objectstorage.CredentialsGroup, model *Model, region string) error {
	var credentialsGroupId string
	if !coreutils.IsUndefined(model.CredentialsGroupId) {
		credentialsGroupId = model.CredentialsGroupId.ValueString()
	} else if credentialsGroup.CredentialsGroupId != nil {
		credentialsGroupId = *credentialsGroup.CredentialsGroupId
	} else {
		return fmt.Errorf("credential id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		region,
		credentialsGroupId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.CredentialsGroupId = types.StringValue(credentialsGroupId)
	model.URN = types.StringPointerValue(credentialsGroup.Urn)
	model.Name = types.StringPointerValue(credentialsGroup.DisplayName)
	return nil
}

type objectStorageClient interface {
	EnableServiceExecute(ctx context.Context, projectId, region string) (*objectstorage.ProjectStatus, error)
	ListCredentialsGroupsExecute(ctx context.Context, projectId, region string) (*objectstorage.ListCredentialsGroupsResponse, error)
}

// enableProject enables object storage for the specified project. If the project is already enabled, nothing happens
func enableProject(ctx context.Context, model *Model, region string, client objectStorageClient) error {
	projectId := model.ProjectId.ValueString()

	// From the object storage OAS: Creation will also be successful if the project is already enabled, but will not create a duplicate
	_, err := client.EnableServiceExecute(ctx, projectId, region)
	if err != nil {
		return fmt.Errorf("failed to create object storage project: %w", err)
	}
	return nil
}

// readCredentialsGroups gets all the existing credentials groups for the specified project,
// finds the credentials group that is being read and updates the state.
// Returns True if the credential was found, False otherwise.
func readCredentialsGroups(ctx context.Context, model *Model, region string, client objectStorageClient) (bool, error) {
	found := false

	if model.CredentialsGroupId.ValueString() == "" && model.Name.ValueString() == "" {
		return found, fmt.Errorf("missing configuration: either name or credentials group id must be provided")
	}

	credentialsGroupsResp, err := client.ListCredentialsGroupsExecute(ctx, model.ProjectId.ValueString(), region)
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			return found, nil
		}
		return found, fmt.Errorf("getting credentials groups: %w", err)
	}

	if credentialsGroupsResp == nil {
		return found, fmt.Errorf("nil response from GET credentials groups")
	}

	for _, credentialsGroup := range *credentialsGroupsResp.CredentialsGroups {
		if *credentialsGroup.CredentialsGroupId != model.CredentialsGroupId.ValueString() && *credentialsGroup.DisplayName != model.Name.ValueString() {
			continue
		}
		found = true
		err = mapCredentialsGroup(credentialsGroup, model, region)
		if err != nil {
			return found, err
		}
		break
	}

	return found, nil
}
