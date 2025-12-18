package resourcepool

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	_ resource.Resource                = &resourcePoolResource{}
	_ resource.ResourceWithImportState = &resourcePoolResource{}
	_ resource.ResourceWithConfigure   = &resourcePoolResource{}
	_ resource.ResourceWithModifyPlan  = &resourcePoolResource{}
)

type Model struct {
	Id                  types.String `tfsdk:"id"` // needed by TF
	ProjectId           types.String `tfsdk:"project_id"`
	ResourcePoolId      types.String `tfsdk:"resource_pool_id"`
	AvailabilityZone    types.String `tfsdk:"availability_zone"`
	IpAcl               types.List   `tfsdk:"ip_acl"`
	Name                types.String `tfsdk:"name"`
	PerformanceClass    types.String `tfsdk:"performance_class"`
	SizeGigabytes       types.Int64  `tfsdk:"size_gigabytes"`
	Region              types.String `tfsdk:"region"`
	SnapshotsAreVisible types.Bool   `tfsdk:"snapshots_are_visible"`
}

// NewResourcePoolResource is a helper function to simplify the provider implementation.
func NewResourcePoolResource() resource.Resource {
	return &resourcePoolResource{}
}

// resourcePoolResource is the resource implementation.
type resourcePoolResource struct {
	client       *sfs.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
func (r *resourcePoolResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { //nolint:gocritic // defined by terraform api
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
func (r *resourcePoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sfs_resource_pool"
}

// Configure adds the provider configured client to the resource.
func (r *resourcePoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_sfs_resource_pool", core.Resource)
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
func (r *resourcePoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Resource-pool resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription(description, core.Resource),
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`resource_pool_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the resource pool is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
			"resource_pool_id": schema.StringAttribute{
				Description: "Resource pool ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"availability_zone": schema.StringAttribute{
				Required:    true,
				Description: "Availability zone.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip_acl": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: `List of IPs that can mount the resource pool in read-only; IPs must have a subnet mask (e.g. "172.16.0.0/24" for a range of IPs, or "172.16.0.250/32" for a specific IP).`,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(validate.CIDR()),
				},
			},
			"performance_class": schema.StringAttribute{
				Required:    true,
				Description: "Name of the performance class.",
			},
			"size_gigabytes": schema.Int64Attribute{
				Required:    true,
				Description: `Size of the resource pool (unit: gigabytes)`,
			},
			"name": schema.StringAttribute{
				Description: "Name of the resource pool.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					// api does not allow to change the name
					stringplanmodifier.RequiresReplace(),
				},
			},
			"snapshots_are_visible": schema.BoolAttribute{
				Description: "If set to true, snapshots are visible and accessible to users. (default: false)",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *resourcePoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating resource pool", fmt.Sprintf("Cannot create payload: %v", err))
		return
	}

	// Create new resourcepool
	resourcePool, err := r.client.CreateResourcePool(ctx, projectId, region).
		CreateResourcePoolPayload(*payload).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating resource pool", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if resourcePool == nil || resourcePool.ResourcePool == nil || resourcePool.ResourcePool.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "error creating resource pool", "Calling API: Incomplete response (id missing)")
		return
	}

	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":       projectId,
		"region":           region,
		"resource_pool_id": *resourcePool.ResourcePool.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := wait.CreateResourcePoolWaitHandler(ctx, r.client, projectId, region, *resourcePool.ResourcePool.Id).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating resource pool", fmt.Sprintf("resource pool creation waiting: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "resource_pool_id", response.ResourcePool.Id)

	// the responses of create and update are not compatible, so we can't use a unified
	// mapFields function. Therefore, we issue a GET request after the create
	// to get a compatible structure
	if response.ResourcePool == nil || response.ResourcePool.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating resource pool", "response did not contain an ID")
		return
	}
	getResponse, err := r.client.GetResourcePoolExecute(ctx, projectId, region, *response.ResourcePool.Id)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating resource pool", fmt.Sprintf("resource pool get: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, region, getResponse.ResourcePool, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating resource pool", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS ResourcePool created")
}

// Read refreshes the Terraform state with the latest data.
func (r *resourcePoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	response, err := r.client.GetResourcePoolExecute(ctx, projectId, region, resourcePoolId)
	if err != nil {
		var openapiError *oapierror.GenericOpenAPIError
		if errors.As(err, &openapiError) {
			if openapiError.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading resource pool", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, region, response.ResourcePool, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading resource pool", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS resource pool read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *resourcePoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)
	ctx = tflog.SetField(ctx, "region", region)

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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Update resource pool", fmt.Sprintf("cannot create payload: %v", err))
		return
	}

	response, err := r.client.UpdateResourcePool(ctx, projectId, region, resourcePoolId).
		UpdateResourcePoolPayload(*payload).
		Execute()
	if err != nil {
		var openapiError *oapierror.GenericOpenAPIError
		if errors.As(err, &openapiError) {
			if openapiError.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating resource pool", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// the responses of create and update are not compatible, so we can't use a unified
	// mapFields function. Therefore, we issue a GET request after the create
	// to get a compatible structure
	if response.ResourcePool == nil || response.ResourcePool.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating resource pool", "response did not contain an ID")
		return
	}

	getResponse, err := wait.UpdateResourcePoolWaitHandler(ctx, r.client, projectId, region, resourcePoolId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating resource pool", fmt.Sprintf("resource pool get: %v", err))
		return
	}
	err = mapFields(ctx, region, getResponse.ResourcePool, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating resource pool", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS resource pool updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *resourcePoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	// Delete existing resource pool
	_, err := r.client.DeleteResourcePoolExecute(ctx, projectId, region, resourcePoolId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting resource pool", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// only delete, if no error occurred
	_, err = wait.DeleteResourcePoolWaitHandler(ctx, r.client, projectId, region, resourcePoolId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting resource pool", fmt.Sprintf("resource pool deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "SFS resource pool deleted")
}

// ImportState implements resource.ResourceWithImportState.
func (r *resourcePoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing resource pool",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[resource_pool_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_pool_id"), idParts[2])...)

	tflog.Info(ctx, "SFS resource pool imported")
}

func mapFields(ctx context.Context, region string, resourcePool *sfs.GetResourcePoolResponseResourcePool, model *Model) error {
	if resourcePool == nil {
		return fmt.Errorf("resource pool empty in response")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if resourcePool.Id == nil {
		return fmt.Errorf("resource pool id not present")
	}
	model.Region = types.StringValue(region)
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		utils.Coalesce(model.ResourcePoolId, types.StringPointerValue(resourcePool.Id)).ValueString(),
	)
	model.AvailabilityZone = types.StringPointerValue(resourcePool.AvailabilityZone)
	model.ResourcePoolId = types.StringPointerValue(resourcePool.Id)
	model.SnapshotsAreVisible = types.BoolPointerValue(resourcePool.SnapshotsAreVisible)

	if resourcePool.IpAcl != nil {
		var diags diag.Diagnostics
		model.IpAcl, diags = types.ListValueFrom(ctx, types.StringType, resourcePool.IpAcl)
		if diags.HasError() {
			return fmt.Errorf("failed to map ip acls: %w", core.DiagsToError(diags))
		}
	} else {
		model.IpAcl = types.ListNull(types.StringType)
	}

	model.Name = types.StringPointerValue(resourcePool.Name)
	if pc := resourcePool.PerformanceClass; pc != nil {
		model.PerformanceClass = types.StringPointerValue(pc.Name)
	}

	if resourcePool.Space != nil {
		model.SizeGigabytes = types.Int64PointerValue(resourcePool.Space.SizeGigabytes)
	}

	return nil
}

func toCreatePayload(model *Model) (*sfs.CreateResourcePoolPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	var (
		aclList *[]string
	)
	if !utils.IsUndefined(model.IpAcl) {
		tmp, err := utils.ListValuetoStringSlice(model.IpAcl)
		if err != nil {
			return nil, fmt.Errorf("cannot get acl ip list from model: %w", err)
		}
		aclList = &tmp
	}

	result := &sfs.CreateResourcePoolPayload{
		AvailabilityZone:    model.AvailabilityZone.ValueStringPointer(),
		IpAcl:               aclList,
		Name:                model.Name.ValueStringPointer(),
		PerformanceClass:    model.PerformanceClass.ValueStringPointer(),
		SizeGigabytes:       model.SizeGigabytes.ValueInt64Pointer(),
		SnapshotsAreVisible: model.SnapshotsAreVisible.ValueBoolPointer(),
	}
	return result, nil
}

func toUpdatePayload(model *Model) (*sfs.UpdateResourcePoolPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	var (
		aclList *[]string
	)
	if !utils.IsUndefined(model.IpAcl) {
		tmp, err := utils.ListValuetoStringSlice(model.IpAcl)
		if err != nil {
			return nil, fmt.Errorf("cannot get acl ip list from model: %w", err)
		}
		aclList = &tmp
	}

	result := &sfs.UpdateResourcePoolPayload{
		IpAcl:               aclList,
		PerformanceClass:    model.PerformanceClass.ValueStringPointer(),
		SizeGigabytes:       model.SizeGigabytes.ValueInt64Pointer(),
		SnapshotsAreVisible: model.SnapshotsAreVisible.ValueBoolPointer(),
	}
	return result, nil
}
