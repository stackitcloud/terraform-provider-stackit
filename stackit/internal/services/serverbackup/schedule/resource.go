package schedule

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	serverbackupUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serverbackup/utils"

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

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &scheduleResource{}
	_ resource.ResourceWithConfigure   = &scheduleResource{}
	_ resource.ResourceWithImportState = &scheduleResource{}
	_ resource.ResourceWithModifyPlan  = &scheduleResource{}
)

type Model struct {
	ID               types.String                   `tfsdk:"id"`
	ProjectId        types.String                   `tfsdk:"project_id"`
	ServerId         types.String                   `tfsdk:"server_id"`
	BackupScheduleId types.Int64                    `tfsdk:"backup_schedule_id"`
	Name             types.String                   `tfsdk:"name"`
	Rrule            types.String                   `tfsdk:"rrule"`
	Enabled          types.Bool                     `tfsdk:"enabled"`
	BackupProperties *scheduleBackupPropertiesModel `tfsdk:"backup_properties"`
	Region           types.String                   `tfsdk:"region"`
}

// scheduleBackupPropertiesModel maps schedule backup_properties data
type scheduleBackupPropertiesModel struct {
	BackupName      types.String `tfsdk:"name"`
	RetentionPeriod types.Int64  `tfsdk:"retention_period"`
	VolumeIds       types.List   `tfsdk:"volume_ids"`
}

// NewScheduleResource is a helper function to simplify the provider implementation.
func NewScheduleResource() resource.Resource {
	return &scheduleResource{}
}

// scheduleResource is the resource implementation.
type scheduleResource struct {
	client       *serverbackup.APIClient
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
	resp.TypeName = req.ProviderTypeName + "_server_backup_schedule"
}

// Configure adds the provider configured client to the resource.
func (r *scheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_server_backup_schedule", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := serverbackupUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Server backup client configured.")
}

// Schema defines the schema for the resource.
func (r *scheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Server backup schedule resource schema. Must have a `region` specified in the provider configuration.",
		MarkdownDescription: features.AddBetaDescription("Server backup schedule resource schema. Must have a `region` specified in the provider configuration.", core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`server_id`,`backup_schedule_id`\".",
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
			"backup_schedule_id": schema.Int64Attribute{
				Description: "Backup schedule ID.",
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
				Description: "Server ID for the backup schedule.",
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
				Description: "Backup schedule described in `rrule` (recurrence rule) format.",
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
				Description: "Is the backup schedule enabled or disabled.",
				Required:    true,
			},
			"backup_properties": schema.SingleNestedAttribute{
				Description: "Backup schedule details for the backups.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"volume_ids": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Required: true,
					},
					"retention_period": schema.Int64Attribute{
						Required: true,
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
						},
					},
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
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "region", region)

	// Enable backups if not already enabled
	err := r.enableBackupsService(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server backup schedule", fmt.Sprintf("Enabling server backup project before creation: %v", err))
		return
	}

	// Create new schedule
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server backup schedule", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	scheduleResp, err := r.client.CreateBackupSchedule(ctx, projectId, serverId, region).CreateBackupSchedulePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server backup schedule", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "backup_schedule_id", *scheduleResp.Id)

	// Map response body to schema
	err = mapFields(ctx, scheduleResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server backup schedule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server backup schedule created.")
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
	backupScheduleId := model.BackupScheduleId.ValueInt64()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "backup_schedule_id", backupScheduleId)
	ctx = tflog.SetField(ctx, "region", region)

	scheduleResp, err := r.client.GetBackupSchedule(ctx, projectId, serverId, region, strconv.FormatInt(backupScheduleId, 10)).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading backup schedule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, scheduleResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading backup schedule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server backup schedule read.")
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
	backupScheduleId := model.BackupScheduleId.ValueInt64()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "backup_schedule_id", backupScheduleId)
	ctx = tflog.SetField(ctx, "region", region)

	// Update schedule
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server backup schedule", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	scheduleResp, err := r.client.UpdateBackupSchedule(ctx, projectId, serverId, region, strconv.FormatInt(backupScheduleId, 10)).UpdateBackupSchedulePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server backup schedule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, scheduleResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server backup schedule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server backup schedule updated.")
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
	backupScheduleId := model.BackupScheduleId.ValueInt64()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "backup_schedule_id", backupScheduleId)
	ctx = tflog.SetField(ctx, "region", region)

	err := r.client.DeleteBackupSchedule(ctx, projectId, serverId, region, strconv.FormatInt(backupScheduleId, 10)).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting server backup schedule", fmt.Sprintf("Calling API: %v", err))
		return
	}
	tflog.Info(ctx, "Server backup schedule deleted.")

	// Disable backups service in case there are no backups and no backup schedules.
	err = r.disableBackupsService(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting server backup schedule", fmt.Sprintf("Disabling server backup service after deleting schedule: %v", err))
		return
	}
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: // project_id,server_id,schedule_id
func (r *scheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing server backup schedule",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[server_id],[backup_schedule_id], got %q", req.ID),
		)
		return
	}

	intId, err := strconv.ParseInt(idParts[3], 10, 64)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing server backup schedule",
			fmt.Sprintf("Expected backup_schedule_id to be int64, got %q", idParts[2]),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("backup_schedule_id"), intId)...)
	tflog.Info(ctx, "Server backup schedule state imported.")
}

func mapFields(ctx context.Context, schedule *serverbackup.BackupSchedule, model *Model, region string) error {
	if schedule == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	if schedule.Id == nil {
		return fmt.Errorf("response id is nil")
	}

	model.BackupScheduleId = types.Int64PointerValue(schedule.Id)
	model.ID = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), region, model.ServerId.ValueString(),
		strconv.FormatInt(model.BackupScheduleId.ValueInt64(), 10),
	)
	model.Name = types.StringPointerValue(schedule.Name)
	model.Rrule = types.StringPointerValue(schedule.Rrule)
	model.Enabled = types.BoolPointerValue(schedule.Enabled)
	if schedule.BackupProperties == nil {
		model.BackupProperties = nil
		return nil
	}
	volIds := basetypes.NewListNull(types.StringType)
	if schedule.BackupProperties.VolumeIds != nil {
		modelVolIds, err := utils.ListValuetoStringSlice(model.BackupProperties.VolumeIds)
		if err != nil {
			return err
		}

		respVolIds := *schedule.BackupProperties.VolumeIds
		reconciledVolIds := utils.ReconcileStringSlices(modelVolIds, respVolIds)

		var diags diag.Diagnostics
		volIds, diags = types.ListValueFrom(ctx, types.StringType, reconciledVolIds)
		if diags.HasError() {
			return fmt.Errorf("failed to map volumeIds: %w", core.DiagsToError(diags))
		}
	}
	model.BackupProperties = &scheduleBackupPropertiesModel{
		BackupName:      types.StringValue(*schedule.BackupProperties.Name),
		RetentionPeriod: types.Int64Value(*schedule.BackupProperties.RetentionPeriod),
		VolumeIds:       volIds,
	}
	model.Region = types.StringValue(region)
	return nil
}

// If already enabled, just continues
func (r *scheduleResource) enableBackupsService(ctx context.Context, model *Model) error {
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	tflog.Debug(ctx, "Enabling server backup service")
	request := r.client.EnableServiceResource(ctx, projectId, serverId, region).
		EnableServiceResourcePayload(serverbackup.EnableServiceResourcePayload{})

	if err := request.Execute(); err != nil {
		if strings.Contains(err.Error(), "Tried to activate already active service") {
			tflog.Debug(ctx, "Service for server backup already enabled")
			return nil
		}
		return fmt.Errorf("enable server backup service: %w", err)
	}
	tflog.Info(ctx, "Enabled server backup service")
	return nil
}

// Disables only if no backup schedules are present and no backups are present
func (r *scheduleResource) disableBackupsService(ctx context.Context, model *Model) error {
	tflog.Debug(ctx, "Disabling server backup service (in case there are no backups and no backup schedules)")

	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	tflog.Debug(ctx, "Checking for existing backups")
	backups, err := r.client.ListBackups(ctx, projectId, serverId, region).Execute()
	if err != nil {
		return fmt.Errorf("list backups: %w", err)
	}
	if *backups.Items != nil && len(*backups.Items) > 0 {
		tflog.Debug(ctx, "Backups found - will not disable server backup service")
		return nil
	}

	err = r.client.DisableServiceResourceExecute(ctx, projectId, serverId, region)
	if err != nil {
		return fmt.Errorf("disable server backup service: %w", err)
	}
	tflog.Info(ctx, "Disabled server backup service")
	return nil
}

func toCreatePayload(model *Model) (*serverbackup.CreateBackupSchedulePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	backupProperties := serverbackup.BackupProperties{}
	if model.BackupProperties != nil {
		ids := []string{}
		var err error
		if !(model.BackupProperties.VolumeIds.IsNull() || model.BackupProperties.VolumeIds.IsUnknown()) {
			ids, err = utils.ListValuetoStringSlice(model.BackupProperties.VolumeIds)
			if err != nil {
				return nil, fmt.Errorf("convert volume id: %w", err)
			}
		}
		// we should provide null to the API in case no volumeIds were chosen, else it errors
		if len(ids) == 0 {
			ids = nil
		}
		backupProperties = serverbackup.BackupProperties{
			Name:            conversion.StringValueToPointer(model.BackupProperties.BackupName),
			RetentionPeriod: conversion.Int64ValueToPointer(model.BackupProperties.RetentionPeriod),
			VolumeIds:       &ids,
		}
	}
	return &serverbackup.CreateBackupSchedulePayload{
		Enabled:          conversion.BoolValueToPointer(model.Enabled),
		Name:             conversion.StringValueToPointer(model.Name),
		Rrule:            conversion.StringValueToPointer(model.Rrule),
		BackupProperties: &backupProperties,
	}, nil
}

func toUpdatePayload(model *Model) (*serverbackup.UpdateBackupSchedulePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	backupProperties := serverbackup.BackupProperties{}
	if model.BackupProperties != nil {
		ids := []string{}
		var err error
		if !(model.BackupProperties.VolumeIds.IsNull() || model.BackupProperties.VolumeIds.IsUnknown()) {
			ids, err = utils.ListValuetoStringSlice(model.BackupProperties.VolumeIds)
			if err != nil {
				return nil, fmt.Errorf("convert volume id: %w", err)
			}
		}
		// we should provide null to the API in case no volumeIds were chosen, else it errors
		if len(ids) == 0 {
			ids = nil
		}
		backupProperties = serverbackup.BackupProperties{
			Name:            conversion.StringValueToPointer(model.BackupProperties.BackupName),
			RetentionPeriod: conversion.Int64ValueToPointer(model.BackupProperties.RetentionPeriod),
			VolumeIds:       &ids,
		}
	}

	return &serverbackup.UpdateBackupSchedulePayload{
		Enabled:          conversion.BoolValueToPointer(model.Enabled),
		Name:             conversion.StringValueToPointer(model.Name),
		Rrule:            conversion.StringValueToPointer(model.Rrule),
		BackupProperties: &backupProperties,
	}, nil
}
