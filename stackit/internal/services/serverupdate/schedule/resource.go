package schedule

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	serverupdateUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serverupdate/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/serverupdate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &scheduleResource{}
	_ resource.ResourceWithConfigure   = &scheduleResource{}
	_ resource.ResourceWithImportState = &scheduleResource{}
	_ resource.ResourceWithModifyPlan  = &scheduleResource{}
)

type Model struct {
	ID                types.String `tfsdk:"id"`
	ProjectId         types.String `tfsdk:"project_id"`
	ServerId          types.String `tfsdk:"server_id"`
	UpdateScheduleId  types.Int64  `tfsdk:"update_schedule_id"`
	Name              types.String `tfsdk:"name"`
	Rrule             types.String `tfsdk:"rrule"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	MaintenanceWindow types.Int64  `tfsdk:"maintenance_window"`
	Region            types.String `tfsdk:"region"`
}

// NewScheduleResource is a helper function to simplify the provider implementation.
func NewScheduleResource() resource.Resource {
	return &scheduleResource{}
}

// scheduleResource is the resource implementation.
type scheduleResource struct {
	client       *serverupdate.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *scheduleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

// Metadata returns the resource type name.
func (r *scheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_update_schedule"
}

// Configure adds the provider configured client to the resource.
func (r *scheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_server_update_schedule", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := serverupdateUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Server update client configured.")
}

// Schema defines the schema for the resource.
func (r *scheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Server update schedule resource schema. Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on resource level.",
		MarkdownDescription: features.AddBetaDescription("Server update schedule resource schema. Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on resource level.", core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`server_id`,`update_schedule_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The schedule name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"update_schedule_id": schema.Int64Attribute{
				Description: "Update schedule ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT Project ID to which the server is associated.",
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
			"server_id": schema.StringAttribute{
				Description: "Server ID for the update schedule.",
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
			"rrule": schema.StringAttribute{
				Description: "Update schedule described in `rrule` (recurrence rule) format.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.Rrule(),
					validate.NoSeparator(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Is the update schedule enabled or disabled.",
				Required:    true,
			},
			"maintenance_window": schema.Int64Attribute{
				Description: "Maintenance window [1..24].",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AtMost(24),
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
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *scheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "region", region)

	// Enable updates if not already enabled
	err := enableUpdatesService(ctx, &model, r.client, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server update schedule", fmt.Sprintf("Enabling server update project before creation: %v", err))
		return
	}

	// Create new schedule
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server update schedule", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	scheduleResp, err := r.client.CreateUpdateSchedule(ctx, projectId, serverId, region).CreateUpdateSchedulePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server update schedule", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "update_schedule_id", *scheduleResp.Id)

	// Map response body to schema
	err = mapFields(scheduleResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server update schedule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server update schedule created.")
}

// Read refreshes the Terraform state with the latest data.
func (r *scheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	updateScheduleId := model.UpdateScheduleId.ValueInt64()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "update_schedule_id", updateScheduleId)

	scheduleResp, err := r.client.GetUpdateSchedule(ctx, projectId, serverId, strconv.FormatInt(updateScheduleId, 10), region).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading update schedule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(scheduleResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading update schedule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server update schedule read.")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *scheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	updateScheduleId := model.UpdateScheduleId.ValueInt64()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "update_schedule_id", updateScheduleId)

	// Update schedule
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server update schedule", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	scheduleResp, err := r.client.UpdateUpdateSchedule(ctx, projectId, serverId, strconv.FormatInt(updateScheduleId, 10), region).UpdateUpdateSchedulePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server update schedule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(scheduleResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server update schedule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server update schedule updated.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *scheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	updateScheduleId := model.UpdateScheduleId.ValueInt64()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "update_schedule_id", updateScheduleId)

	err := r.client.DeleteUpdateSchedule(ctx, projectId, serverId, strconv.FormatInt(updateScheduleId, 10), region).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting server update schedule", fmt.Sprintf("Calling API: %v", err))
		return
	}
	tflog.Info(ctx, "Server update schedule deleted.")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: // project_id,server_id,schedule_id
func (r *scheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing server update schedule",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[server_id],[update_schedule_id], got %q", req.ID),
		)
		return
	}

	intId, err := strconv.ParseInt(idParts[3], 10, 64)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing server update schedule",
			fmt.Sprintf("Expected update_schedule_id to be int64, got %q", idParts[2]),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("update_schedule_id"), intId)...)
	tflog.Info(ctx, "Server update schedule state imported.")
}

func mapFields(schedule *serverupdate.UpdateSchedule, model *Model, region string) error {
	if schedule == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	if schedule.Id == nil {
		return fmt.Errorf("response id is nil")
	}

	model.UpdateScheduleId = types.Int64PointerValue(schedule.Id)
	model.ID = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), region, model.ServerId.ValueString(),
		strconv.FormatInt(model.UpdateScheduleId.ValueInt64(), 10),
	)
	model.Name = types.StringPointerValue(schedule.Name)
	model.Rrule = types.StringPointerValue(schedule.Rrule)
	model.Enabled = types.BoolPointerValue(schedule.Enabled)
	model.MaintenanceWindow = types.Int64PointerValue(schedule.MaintenanceWindow)
	model.Region = types.StringValue(region)
	return nil
}

// If already enabled, just continues
func enableUpdatesService(ctx context.Context, model *Model, client *serverupdate.APIClient, region string) error {
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	payload := serverupdate.EnableServiceResourcePayload{}

	tflog.Debug(ctx, "Enabling server update service")
	err := client.EnableServiceResource(ctx, projectId, serverId, region).EnableServiceResourcePayload(payload).Execute()
	if err != nil {
		if strings.Contains(err.Error(), "Tried to activate already active service") {
			tflog.Debug(ctx, "Service for server update already enabled")
			return nil
		}
		return fmt.Errorf("enable server update service: %w", err)
	}
	tflog.Info(ctx, "Enabled server update service")
	return nil
}

func toCreatePayload(model *Model) (*serverupdate.CreateUpdateSchedulePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &serverupdate.CreateUpdateSchedulePayload{
		Enabled:           conversion.BoolValueToPointer(model.Enabled),
		Name:              conversion.StringValueToPointer(model.Name),
		Rrule:             conversion.StringValueToPointer(model.Rrule),
		MaintenanceWindow: conversion.Int64ValueToPointer(model.MaintenanceWindow),
	}, nil
}

func toUpdatePayload(model *Model) (*serverupdate.UpdateUpdateSchedulePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &serverupdate.UpdateUpdateSchedulePayload{
		Enabled:           conversion.BoolValueToPointer(model.Enabled),
		Name:              conversion.StringValueToPointer(model.Name),
		Rrule:             conversion.StringValueToPointer(model.Rrule),
		MaintenanceWindow: conversion.Int64ValueToPointer(model.MaintenanceWindow),
	}, nil
}
