package share

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	sfsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	coreutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource               = &shareResource{}
	_ resource.ResourceWithConfigure  = &shareResource{}
	_ resource.ResourceWithModifyPlan = &shareResource{}
)

type Model struct {
	Id                      types.String `tfsdk:"id"` // needed by TF
	ProjectId               types.String `tfsdk:"project_id"`
	ResourcePoolId          types.String `tfsdk:"resource_pool_id"`
	ShareId                 types.String `tfsdk:"share_id"`
	Name                    types.String `tfsdk:"name"`
	ExportPolicyName        types.String `tfsdk:"export_policy"`
	SpaceHardLimitGigabytes types.Int64  `tfsdk:"space_hard_limit_gigabytes"`
	Region                  types.String `tfsdk:"region"`
	MountPath               types.String `tfsdk:"mount_path"`
}

// NewShareResource is a helper function to simplify the provider implementation.
func NewShareResource() resource.Resource {
	return &shareResource{}
}

// shareResource is the resource implementation.
type shareResource struct {
	client       *sfs.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
func (r *shareResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { //nolint:gocritic // defined by terraform api
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
func (r *shareResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sfs_share"
}

// Configure adds the provider configured client to the resource.
func (r *shareResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_sfs_share", core.Resource)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := sfsUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "SFS client configured")
}

// Schema defines the schema for the resource.
func (r *shareResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "SFS Share schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription(description, core.Resource),
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`resource_pool_id`,`share_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the share is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"resource_pool_id": schema.StringAttribute{
				Description: "The ID of the resource pool for the SFS share.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"share_id": schema.StringAttribute{
				Description: "share ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the share.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					// api does not support changing the name
					stringplanmodifier.RequiresReplace(),
				},
			},
			"export_policy": schema.StringAttribute{
				Description: `Name of the Share Export Policy to use in the Share.
Note that if this is set to an empty string, the Share can only be mounted in read only by 
clients with IPs matching the IP ACL of the Resource Pool hosting this Share. 
You can also assign a Share Export Policy after creating the Share`,
				Required: true,
			},
			"space_hard_limit_gigabytes": schema.Int64Attribute{
				Required: true,
				Description: `Space hard limit for the Share. 
				If zero, the Share will have access to the full space of the Resource Pool it lives in.
				(unit: gigabytes)`,
			},
			"mount_path": schema.StringAttribute{
				Computed:    true,
				Description: "Mount path of the Share, used to mount the Share",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *shareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)

	ctx = core.InitProviderContext(ctx)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Create Resourcepool", fmt.Sprintf("Cannot create payload: %v", err))
		return
	}

	// Create new share
	share, err := r.client.CreateShare(ctx, projectId, region, resourcePoolId).
		CreateSharePayload(payload).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating share", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if share.Share == nil || share.Share.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "error creating share", "Calling API: Incomplete response (id missing)")
		return
	}
	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":       projectId,
		"region":           region,
		"resource_pool_id": resourcePoolId,
		"share_id":         *share.Share.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := wait.CreateShareWaitHandler(ctx, r.client, projectId, region, resourcePoolId, *share.Share.Id).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating share", fmt.Sprintf("share creation waiting: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "share_id", response.Share.Id)

	// the responses of create and update are not compatible, so we can't use a unified
	// mapFields function. Therefore, we issue a GET request after the create
	// to get a compatible structure
	if response.Share == nil || response.Share.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating share", "response did not contain an ID")
		return
	}
	getResponse, err := r.client.GetShareExecute(ctx, projectId, region, resourcePoolId, *response.Share.Id)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating share", fmt.Sprintf("share get: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, getResponse.Share, region, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating share", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS Share created")
}

// Read refreshes the Terraform state with the latest data.
func (r *shareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	shareId := model.ShareId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)
	ctx = tflog.SetField(ctx, "share_id", shareId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	response, err := r.client.GetShareExecute(ctx, projectId, region, resourcePoolId, shareId)
	if err != nil {
		var openapiError *oapierror.GenericOpenAPIError
		if errors.As(err, &openapiError) {
			if openapiError.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading share", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, response.Share, region, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading share", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS share read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *shareResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	shareId := model.ShareId.ValueString()
	region := model.Region.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "share_id", shareId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)

	ctx = core.InitProviderContext(ctx)

	// Retrieve values from state
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Update share", fmt.Sprintf("cannot create payload: %v", err))
		return
	}

	response, err := r.client.UpdateShare(ctx, projectId, region, resourcePoolId, shareId).
		UpdateSharePayload(*payload).
		Execute()
	if err != nil {
		var openapiError *oapierror.GenericOpenAPIError
		if errors.As(err, &openapiError) {
			if openapiError.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating share", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// the responses of create and update are not compatible, so we can't use a unified
	// mapFields function. Therefore, we issue a GET request after the create
	// to get a compatible structure
	if response.Share == nil || response.Share.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating share", "response did not contain an ID")
		return
	}

	getResponse, err := wait.UpdateShareWaitHandler(ctx, r.client, projectId, region, resourcePoolId, shareId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating share", fmt.Sprintf("share get: %v", err))
		return
	}
	err = mapFields(ctx, getResponse.Share, region, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating share", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS share updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *shareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	shareId := model.ShareId.ValueString()
	region := model.Region.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "share_id", shareId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)

	ctx = core.InitProviderContext(ctx)

	// Delete existing share
	_, err := r.client.DeleteShareExecute(ctx, projectId, region, resourcePoolId, shareId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting share", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// only delete, if no error occurred
	_, err = wait.DeleteShareWaitHandler(ctx, r.client, projectId, region, resourcePoolId, shareId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting share", fmt.Sprintf("share deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "SFS share deleted")
}

// ImportState implements resource.ResourceWithImportState.
func (r *shareResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing share",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[resource_pool_id],[share_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_pool_id"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("share_id"), idParts[3])...)

	tflog.Info(ctx, "SFS share imported")
}

func mapFields(_ context.Context, share *sfs.GetShareResponseShare, region string, model *Model) error {
	if share == nil {
		return fmt.Errorf("share empty in response")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if share.Id == nil {
		return fmt.Errorf("share id not present")
	}
	model.ShareId = types.StringPointerValue(share.Id)

	model.Region = types.StringValue(region)
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		model.ResourcePoolId.ValueString(),
		utils.Coalesce(model.ShareId, types.StringPointerValue(share.Id)).ValueString(),
	)
	model.Name = types.StringPointerValue(share.Name)

	if policy := share.ExportPolicy.Get(); policy != nil {
		model.ExportPolicyName = types.StringPointerValue(policy.Name)
	}

	model.SpaceHardLimitGigabytes = types.Int64PointerValue(share.SpaceHardLimitGigabytes)
	model.MountPath = types.StringPointerValue(share.MountPath)

	return nil
}

func toCreatePayload(model *Model) (ret sfs.CreateSharePayload, err error) {
	if model == nil {
		return ret, fmt.Errorf("nil model")
	}
	result := sfs.CreateSharePayload{
		ExportPolicyName:        sfs.NewNullableString(model.ExportPolicyName.ValueStringPointer()),
		Name:                    model.Name.ValueStringPointer(),
		SpaceHardLimitGigabytes: model.SpaceHardLimitGigabytes.ValueInt64Pointer(),
	}
	return result, nil
}

func toUpdatePayload(model *Model) (*sfs.UpdateSharePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	result := &sfs.UpdateSharePayload{
		ExportPolicyName:        sfs.NewNullableString(model.ExportPolicyName.ValueStringPointer()),
		SpaceHardLimitGigabytes: model.SpaceHardLimitGigabytes.ValueInt64Pointer(),
	}
	return result, nil
}
