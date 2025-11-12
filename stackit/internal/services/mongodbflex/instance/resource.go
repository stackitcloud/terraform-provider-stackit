package mongodbflex

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	mongodbflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mongodbflex/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
	_ resource.ResourceWithModifyPlan  = &instanceResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	InstanceId     types.String `tfsdk:"instance_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	ACL            types.List   `tfsdk:"acl"`
	BackupSchedule types.String `tfsdk:"backup_schedule"`
	Flavor         types.Object `tfsdk:"flavor"`
	Replicas       types.Int64  `tfsdk:"replicas"`
	Storage        types.Object `tfsdk:"storage"`
	Version        types.String `tfsdk:"version"`
	Options        types.Object `tfsdk:"options"`
	Region         types.String `tfsdk:"region"`
}

// Struct corresponding to Model.Flavor
type flavorModel struct {
	Id          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	CPU         types.Int64  `tfsdk:"cpu"`
	RAM         types.Int64  `tfsdk:"ram"`
}

// Types corresponding to flavorModel
var flavorTypes = map[string]attr.Type{
	"id":          basetypes.StringType{},
	"description": basetypes.StringType{},
	"cpu":         basetypes.Int64Type{},
	"ram":         basetypes.Int64Type{},
}

// Struct corresponding to Model.Storage
type storageModel struct {
	Class types.String `tfsdk:"class"`
	Size  types.Int64  `tfsdk:"size"`
}

// Types corresponding to storageModel
var storageTypes = map[string]attr.Type{
	"class": basetypes.StringType{},
	"size":  basetypes.Int64Type{},
}

// Struct corresponding to Model.Options
type optionsModel struct {
	Type                           types.String `tfsdk:"type"`
	SnapshotRetentionDays          types.Int64  `tfsdk:"snapshot_retention_days"`
	PointInTimeWindowHours         types.Int64  `tfsdk:"point_in_time_window_hours"`
	DailySnapshotRetentionDays     types.Int64  `tfsdk:"daily_snapshot_retention_days"`
	WeeklySnapshotRetentionWeeks   types.Int64  `tfsdk:"weekly_snapshot_retention_weeks"`
	MonthlySnapshotRetentionMonths types.Int64  `tfsdk:"monthly_snapshot_retention_months"`
}

// Types corresponding to optionsModel
var optionsTypes = map[string]attr.Type{
	"type":                              basetypes.StringType{},
	"snapshot_retention_days":           basetypes.Int64Type{},
	"point_in_time_window_hours":        basetypes.Int64Type{},
	"daily_snapshot_retention_days":     basetypes.Int64Type{},
	"weekly_snapshot_retention_weeks":   basetypes.Int64Type{},
	"monthly_snapshot_retention_months": basetypes.Int64Type{},
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client       *mongodbflex.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mongodbflex_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := mongodbflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "MongoDB Flex instance client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	typeOptions := []string{"Replica", "Sharded", "Single"}

	descriptions := map[string]string{
		"main":                              "MongoDB Flex instance resource schema. Must have a `region` specified in the provider configuration.",
		"id":                                "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
		"instance_id":                       "ID of the MongoDB Flex instance.",
		"project_id":                        "STACKIT project ID to which the instance is associated.",
		"name":                              "Instance name.",
		"acl":                               "The Access Control List (ACL) for the MongoDB Flex instance.",
		"backup_schedule":                   `The backup schedule. Should follow the cron scheduling system format (e.g. "0 0 * * *").`,
		"options":                           "Custom parameters for the MongoDB Flex instance.",
		"type":                              fmt.Sprintf("Type of the MongoDB Flex instance. %s", utils.FormatPossibleValues(typeOptions...)),
		"snapshot_retention_days":           "The number of days that continuous backups (controlled via the `backup_schedule`) will be retained.",
		"daily_snapshot_retention_days":     "The number of days that daily backups will be retained.",
		"weekly_snapshot_retention_weeks":   "The number of weeks that weekly backups will be retained.",
		"monthly_snapshot_retention_months": "The number of months that monthly backups will be retained.",
		"point_in_time_window_hours":        "The number of hours back in time the point-in-time recovery feature will be able to recover.",
		"region":                            "The resource region. If not defined, the provider region is used.",
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
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
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
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$"),
						"must start with a letter, must have lower case letters, numbers or hyphens, and no hyphen at the end",
					),
				},
			},
			"acl": schema.ListAttribute{
				Description: descriptions["acl"],
				ElementType: types.StringType,
				Required:    true,
			},
			"backup_schedule": schema.StringAttribute{
				Description: descriptions["backup_schedule"],
				Required:    true,
			},
			"flavor": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"description": schema.StringAttribute{
						Computed: true,
					},
					"cpu": schema.Int64Attribute{
						Required: true,
					},
					"ram": schema.Int64Attribute{
						Required: true,
					},
				},
			},
			"replicas": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"storage": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"class": schema.StringAttribute{
						Required: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"size": schema.Int64Attribute{
						Required: true,
					},
				},
			},
			"version": schema.StringAttribute{
				Required: true,
			},
			"options": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: descriptions["type"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"snapshot_retention_days": schema.Int64Attribute{
						Description: descriptions["snapshot_retention_days"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"daily_snapshot_retention_days": schema.Int64Attribute{
						Description: descriptions["daily_snapshot_retention_days"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"weekly_snapshot_retention_weeks": schema.Int64Attribute{
						Description: descriptions["weekly_snapshot_retention_weeks"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"monthly_snapshot_retention_months": schema.Int64Attribute{
						Description: descriptions["monthly_snapshot_retention_months"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"point_in_time_window_hours": schema.Int64Attribute{
						Description: descriptions["point_in_time_window_hours"],
						Required:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
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
func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	var acl []string
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client, &model, flavor, region)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var options = &optionsModel{}
	if !(model.Options.IsNull() || model.Options.IsUnknown()) {
		diags = model.Options.As(ctx, options, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, acl, flavor, storage, options)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new instance
	createResp, err := r.client.CreateInstance(ctx, projectId, region).CreateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if createResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", "API response is empty")
		return
	}
	if createResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", "API response does not contain instance id")
		return
	}
	instanceId := *createResp.Id
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	diags = resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client, projectId, instanceId, region).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, storage, options, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	backupScheduleOptionsPayload, err := toUpdateBackupScheduleOptionsPayload(ctx, &model, options)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	backupScheduleOptions, err := r.client.UpdateBackupSchedule(ctx, projectId, instanceId, region).UpdateBackupSchedulePayload(*backupScheduleOptionsPayload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Updating options: %v", err))
		return
	}

	err = mapOptions(&model, options, backupScheduleOptions)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "MongoDB Flex instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var options = &optionsModel{}
	if !(model.Options.IsNull() || model.Options.IsUnknown()) {
		diags = model.Options.As(ctx, options, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	instanceResp, err := r.client.GetInstance(ctx, projectId, instanceId, region).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", err.Error())
		return
	}

	// Map response body to schema
	err = mapFields(ctx, instanceResp, &model, flavor, storage, options, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "MongoDB Flex instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	var acl []string
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client, &model, flavor, region)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var options = &optionsModel{}
	if !(model.Options.IsNull() || model.Options.IsUnknown()) {
		diags = model.Options.As(ctx, options, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model, acl, flavor, storage, options)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing instance
	_, err = r.client.PartialUpdateInstance(ctx, projectId, instanceId, region).PartialUpdateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", err.Error())
		return
	}
	waitResp, err := wait.UpdateInstanceWaitHandler(ctx, r.client, projectId, instanceId, region).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, storage, options, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	backupScheduleOptionsPayload, err := toUpdateBackupScheduleOptionsPayload(ctx, &model, options)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	backupScheduleOptions, err := r.client.UpdateBackupSchedule(ctx, projectId, instanceId, region).UpdateBackupSchedulePayload(*backupScheduleOptionsPayload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Updating options: %v", err))
		return
	}

	err = mapOptions(&model, options, backupScheduleOptions)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "MongoDB Flex instance updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// Delete existing instance
	err := r.client.DeleteInstance(ctx, projectId, instanceId, region).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client, projectId, instanceId, region).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}

	// This is needed because the waiter is currently not working properly
	// After the get request returns 404 (instance is deleted), creating a new instance with the same name still fails for a short period of time
	time.Sleep(30 * time.Second)

	tflog.Info(ctx, "MongoDB Flex instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	tflog.Info(ctx, "MongoDB Flex instance state imported")
}

func mapFields(ctx context.Context, resp *mongodbflex.InstanceResponse, model *Model, flavor *flavorModel, storage *storageModel, options *optionsModel, region string) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if resp.Item == nil {
		return fmt.Errorf("no instance provided")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	instance := resp.Item

	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if instance.Id != nil {
		instanceId = *instance.Id
	} else {
		return fmt.Errorf("instance id not present")
	}

	var aclList basetypes.ListValue
	var diags diag.Diagnostics
	if instance.Acl == nil || instance.Acl.Items == nil {
		aclList = types.ListNull(types.StringType)
	} else {
		respACL := *instance.Acl.Items
		modelACL, err := utils.ListValuetoStringSlice(model.ACL)
		if err != nil {
			return err
		}

		reconciledACL := utils.ReconcileStringSlices(modelACL, respACL)

		aclList, diags = types.ListValueFrom(ctx, types.StringType, reconciledACL)
		if diags.HasError() {
			return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
		}
	}

	var flavorValues map[string]attr.Value
	if instance.Flavor == nil {
		flavorValues = map[string]attr.Value{
			"id":          flavor.Id,
			"description": flavor.Description,
			"cpu":         flavor.CPU,
			"ram":         flavor.RAM,
		}
	} else {
		flavorValues = map[string]attr.Value{
			"id":          types.StringValue(*instance.Flavor.Id),
			"description": types.StringValue(*instance.Flavor.Description),
			"cpu":         types.Int64PointerValue(instance.Flavor.Cpu),
			"ram":         types.Int64PointerValue(instance.Flavor.Memory),
		}
	}
	flavorObject, diags := types.ObjectValue(flavorTypes, flavorValues)
	if diags.HasError() {
		return fmt.Errorf("creating flavor: %w", core.DiagsToError(diags))
	}

	var storageValues map[string]attr.Value
	if instance.Storage == nil {
		storageValues = map[string]attr.Value{
			"class": storage.Class,
			"size":  storage.Size,
		}
	} else {
		storageValues = map[string]attr.Value{
			"class": types.StringValue(*instance.Storage.Class),
			"size":  types.Int64PointerValue(instance.Storage.Size),
		}
	}
	storageObject, diags := types.ObjectValue(storageTypes, storageValues)
	if diags.HasError() {
		return fmt.Errorf("creating storage: %w", core.DiagsToError(diags))
	}

	var optionsValues map[string]attr.Value
	if instance.Options == nil {
		optionsValues = map[string]attr.Value{
			"type":                              options.Type,
			"snapshot_retention_days":           types.Int64Null(),
			"daily_snapshot_retention_days":     types.Int64Null(),
			"weekly_snapshot_retention_weeks":   types.Int64Null(),
			"monthly_snapshot_retention_months": types.Int64Null(),
			"point_in_time_window_hours":        types.Int64Null(),
		}
	} else {
		snapshotRetentionDaysStr := (*instance.Options)["snapshotRetentionDays"]
		snapshotRetentionDays, err := strconv.ParseInt(snapshotRetentionDaysStr, 10, 64)
		if err != nil {
			return fmt.Errorf("parse snapshot retention days: %w", err)
		}
		dailySnapshotRetentionDaysStr := (*instance.Options)["dailySnapshotRetentionDays"]
		dailySnapshotRetentionDays, err := strconv.ParseInt(dailySnapshotRetentionDaysStr, 10, 64)
		if err != nil {
			return fmt.Errorf("parse daily snapshot retention days: %w", err)
		}
		weeklySnapshotRetentionWeeksStr := (*instance.Options)["weeklySnapshotRetentionWeeks"]
		weeklySnapshotRetentionWeeks, err := strconv.ParseInt(weeklySnapshotRetentionWeeksStr, 10, 64)
		if err != nil {
			return fmt.Errorf("parse weekly snapshot retention weeks: %w", err)
		}
		monthlySnapshotRetentionMonthsStr := (*instance.Options)["monthlySnapshotRetentionMonths"]
		monthlySnapshotRetentionMonths, err := strconv.ParseInt(monthlySnapshotRetentionMonthsStr, 10, 64)
		if err != nil {
			return fmt.Errorf("parse monthly snapshot retention months: %w", err)
		}
		pointInTimeWindowHoursStr := (*instance.Options)["pointInTimeWindowHours"]
		pointInTimeWindowHours, err := strconv.ParseInt(pointInTimeWindowHoursStr, 10, 64)
		if err != nil {
			return fmt.Errorf("parse point in time window hours: %w", err)
		}

		optionsValues = map[string]attr.Value{
			"type":                              types.StringValue((*instance.Options)["type"]),
			"snapshot_retention_days":           types.Int64Value(snapshotRetentionDays),
			"daily_snapshot_retention_days":     types.Int64Value(dailySnapshotRetentionDays),
			"weekly_snapshot_retention_weeks":   types.Int64Value(weeklySnapshotRetentionWeeks),
			"monthly_snapshot_retention_months": types.Int64Value(monthlySnapshotRetentionMonths),
			"point_in_time_window_hours":        types.Int64Value(pointInTimeWindowHours),
		}
	}
	optionsObject, diags := types.ObjectValue(optionsTypes, optionsValues)
	if diags.HasError() {
		return fmt.Errorf("creating options: %w", core.DiagsToError(diags))
	}

	simplifiedModelBackupSchedule := utils.SimplifyBackupSchedule(model.BackupSchedule.ValueString())
	// If the value returned by the API is different from the one in the model after simplification,
	// we update the model so that it causes an error in Terraform
	if simplifiedModelBackupSchedule != types.StringPointerValue(instance.BackupSchedule).ValueString() {
		model.BackupSchedule = types.StringPointerValue(instance.BackupSchedule)
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instanceId)
	model.Region = types.StringValue(region)
	model.InstanceId = types.StringValue(instanceId)
	model.Name = types.StringPointerValue(instance.Name)
	model.ACL = aclList
	model.Flavor = flavorObject
	model.Replicas = types.Int64PointerValue(instance.Replicas)
	model.Storage = storageObject
	model.Version = types.StringPointerValue(instance.Version)
	model.Options = optionsObject
	return nil
}

func mapOptions(model *Model, options *optionsModel, backupScheduleOptions *mongodbflex.BackupSchedule) error {
	var optionsValues map[string]attr.Value
	if backupScheduleOptions == nil {
		optionsValues = map[string]attr.Value{
			"type":                              options.Type,
			"snapshot_retention_days":           types.Int64Null(),
			"daily_snapshot_retention_days":     types.Int64Null(),
			"weekly_snapshot_retention_weeks":   types.Int64Null(),
			"monthly_snapshot_retention_months": types.Int64Null(),
			"point_in_time_window_hours":        types.Int64Null(),
		}
	} else {
		optionsValues = map[string]attr.Value{
			"type":                              options.Type,
			"snapshot_retention_days":           types.Int64Value(*backupScheduleOptions.SnapshotRetentionDays),
			"daily_snapshot_retention_days":     types.Int64Value(*backupScheduleOptions.DailySnapshotRetentionDays),
			"weekly_snapshot_retention_weeks":   types.Int64Value(*backupScheduleOptions.WeeklySnapshotRetentionWeeks),
			"monthly_snapshot_retention_months": types.Int64Value(*backupScheduleOptions.MonthlySnapshotRetentionMonths),
			"point_in_time_window_hours":        types.Int64Value(*backupScheduleOptions.PointInTimeWindowHours),
		}
	}
	optionsTF, diags := types.ObjectValue(optionsTypes, optionsValues)
	if diags.HasError() {
		return fmt.Errorf("creating options: %w", core.DiagsToError(diags))
	}
	model.Options = optionsTF
	return nil
}

func toCreatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel, options *optionsModel) (*mongodbflex.CreateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if acl == nil {
		return nil, fmt.Errorf("nil acl")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}
	if options == nil {
		return nil, fmt.Errorf("nil options")
	}

	payloadOptions := make(map[string]string)
	if options.Type.ValueString() != "" {
		payloadOptions["type"] = options.Type.ValueString()
	}

	return &mongodbflex.CreateInstancePayload{
		Acl: &mongodbflex.CreateInstancePayloadAcl{
			Items: &acl,
		},
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		Replicas:       conversion.Int64ValueToPointer(model.Replicas),
		Storage: &mongodbflex.Storage{
			Class: conversion.StringValueToPointer(storage.Class),
			Size:  conversion.Int64ValueToPointer(storage.Size),
		},
		Version: conversion.StringValueToPointer(model.Version),
		Options: &payloadOptions,
	}, nil
}

func toUpdatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel, options *optionsModel) (*mongodbflex.PartialUpdateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if acl == nil {
		return nil, fmt.Errorf("nil acl")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}
	if options == nil {
		return nil, fmt.Errorf("nil options")
	}

	payloadOptions := make(map[string]string)
	if options.Type.ValueString() != "" {
		payloadOptions["type"] = options.Type.ValueString()
	}

	return &mongodbflex.PartialUpdateInstancePayload{
		Acl: &mongodbflex.ACL{
			Items: &acl,
		},
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		Replicas:       conversion.Int64ValueToPointer(model.Replicas),
		Storage: &mongodbflex.Storage{
			Class: conversion.StringValueToPointer(storage.Class),
			Size:  conversion.Int64ValueToPointer(storage.Size),
		},
		Version: conversion.StringValueToPointer(model.Version),
		Options: &payloadOptions,
	}, nil
}

func toUpdateBackupScheduleOptionsPayload(ctx context.Context, model *Model, configuredOptions *optionsModel) (*mongodbflex.UpdateBackupSchedulePayload, error) {
	if model == nil || configuredOptions == nil {
		return nil, nil
	}

	var currOptions = &optionsModel{}
	if !(model.Options.IsNull() || model.Options.IsUnknown()) {
		diags := model.Options.As(ctx, currOptions, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("map current options: %w", core.DiagsToError(diags))
		}
	}

	backupSchedule := conversion.StringValueToPointer(model.BackupSchedule)

	snapshotRetentionDays := conversion.Int64ValueToPointer(configuredOptions.SnapshotRetentionDays)
	if snapshotRetentionDays == nil {
		snapshotRetentionDays = conversion.Int64ValueToPointer(currOptions.SnapshotRetentionDays)
	}

	dailySnapshotRetentionDays := conversion.Int64ValueToPointer(configuredOptions.DailySnapshotRetentionDays)
	if dailySnapshotRetentionDays == nil {
		dailySnapshotRetentionDays = conversion.Int64ValueToPointer(currOptions.DailySnapshotRetentionDays)
	}

	weeklySnapshotRetentionWeeks := conversion.Int64ValueToPointer(configuredOptions.WeeklySnapshotRetentionWeeks)
	if weeklySnapshotRetentionWeeks == nil {
		weeklySnapshotRetentionWeeks = conversion.Int64ValueToPointer(currOptions.WeeklySnapshotRetentionWeeks)
	}

	monthlySnapshotRetentionMonths := conversion.Int64ValueToPointer(configuredOptions.MonthlySnapshotRetentionMonths)
	if monthlySnapshotRetentionMonths == nil {
		monthlySnapshotRetentionMonths = conversion.Int64ValueToPointer(currOptions.MonthlySnapshotRetentionMonths)
	}

	pointInTimeWindowHours := conversion.Int64ValueToPointer(configuredOptions.PointInTimeWindowHours)
	if pointInTimeWindowHours == nil {
		pointInTimeWindowHours = conversion.Int64ValueToPointer(currOptions.PointInTimeWindowHours)
	}

	return &mongodbflex.UpdateBackupSchedulePayload{
		// This is a PUT endpoint and all fields are required
		BackupSchedule:                 backupSchedule,
		SnapshotRetentionDays:          snapshotRetentionDays,
		DailySnapshotRetentionDays:     dailySnapshotRetentionDays,
		WeeklySnapshotRetentionWeeks:   weeklySnapshotRetentionWeeks,
		MonthlySnapshotRetentionMonths: monthlySnapshotRetentionMonths,
		PointInTimeWindowHours:         pointInTimeWindowHours,
	}, nil
}

type mongoDBFlexClient interface {
	ListFlavorsExecute(ctx context.Context, projectId, region string) (*mongodbflex.ListFlavorsResponse, error)
}

func loadFlavorId(ctx context.Context, client mongoDBFlexClient, model *Model, flavor *flavorModel, region string) error {
	if model == nil {
		return fmt.Errorf("nil model")
	}
	if flavor == nil {
		return fmt.Errorf("nil flavor")
	}
	cpu := conversion.Int64ValueToPointer(flavor.CPU)
	if cpu == nil {
		return fmt.Errorf("nil CPU")
	}
	ram := conversion.Int64ValueToPointer(flavor.RAM)
	if ram == nil {
		return fmt.Errorf("nil RAM")
	}

	projectId := model.ProjectId.ValueString()
	res, err := client.ListFlavorsExecute(ctx, projectId, region)
	if err != nil {
		return fmt.Errorf("listing mongodbflex flavors: %w", err)
	}

	avl := ""
	if res.Flavors == nil {
		return fmt.Errorf("finding flavors for project %s", projectId)
	}
	for _, f := range *res.Flavors {
		if f.Id == nil || f.Cpu == nil || f.Memory == nil {
			continue
		}
		if *f.Cpu == *cpu && *f.Memory == *ram {
			flavor.Id = types.StringValue(*f.Id)
			flavor.Description = types.StringValue(*f.Description)
			break
		}
		avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM", avl, *f.Cpu, *f.Memory)
	}
	if flavor.Id.ValueString() == "" {
		return fmt.Errorf("couldn't find flavor, available specs are:%s", avl)
	}

	return nil
}
