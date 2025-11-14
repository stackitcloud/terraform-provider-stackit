package schedule

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	serverbackupUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serverbackup/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
)

// scheduleDataSourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var schedulesDataSourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &schedulesDataSource{}
)

// NewSchedulesDataSource is a helper function to simplify the provider implementation.
func NewSchedulesDataSource() datasource.DataSource {
	return &schedulesDataSource{}
}

// schedulesDataSource is the data source implementation.
type schedulesDataSource struct {
	client       *serverbackup.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *schedulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_backup_schedules"
}

// Configure adds the provider configured client to the data source.
func (r *schedulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !schedulesDataSourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_server_backup_schedules", "data source")
		if resp.Diagnostics.HasError() {
			return
		}
		schedulesDataSourceBetaCheckDone = true
	}

	apiClient := serverbackupUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Server backup client configured")
}

// Schema defines the schema for the data source.
func (r *schedulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Server backup schedules datasource schema. Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on resource level.",
		MarkdownDescription: features.AddBetaDescription("Server backup schedules datasource schema. Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on resource level.", core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source identifier. It is structured as \"`project_id`,`server_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT Project ID (UUID) to which the server is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"server_id": schema.StringAttribute{
				Description: "Server ID (UUID) to which the backup schedule is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"backup_schedule_id": schema.Int64Attribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Description: "The backup schedule name.",
							Computed:    true,
						},
						"rrule": schema.StringAttribute{
							Description: "Backup schedule described in `rrule` (recurrence rule) format.",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Is the backup schedule enabled or disabled.",
							Computed:    true,
						},
						"backup_properties": schema.SingleNestedAttribute{
							Description: "Backup schedule details for the backups.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"volume_ids": schema.ListAttribute{
									ElementType: types.StringType,
									Computed:    true,
								},
								"name": schema.StringAttribute{
									Computed: true,
								},
								"retention_period": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
					},
				},
			},
			"region": schema.StringAttribute{
				// the region cannot be found, so it has to be passed
				Optional:    true,
				Description: "The resource region. If not defined, the provider region is used.",
			},
		},
	}
}

// schedulesDataSourceModel maps the data source schema data.
type schedulesDataSourceModel struct {
	ID        types.String                   `tfsdk:"id"`
	ProjectId types.String                   `tfsdk:"project_id"`
	ServerId  types.String                   `tfsdk:"server_id"`
	Items     []schedulesDatasourceItemModel `tfsdk:"items"`
	Region    types.String                   `tfsdk:"region"`
}

// schedulesDatasourceItemModel maps schedule schema data.
type schedulesDatasourceItemModel struct {
	BackupScheduleId types.Int64                    `tfsdk:"backup_schedule_id"`
	Name             types.String                   `tfsdk:"name"`
	Rrule            types.String                   `tfsdk:"rrule"`
	Enabled          types.Bool                     `tfsdk:"enabled"`
	BackupProperties *scheduleBackupPropertiesModel `tfsdk:"backup_properties"`
}

// Read refreshes the Terraform state with the latest data.
func (r *schedulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model schedulesDataSourceModel
	diags := req.Config.Get(ctx, &model)
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

	schedules, err := r.client.ListBackupSchedules(ctx, projectId, serverId, region).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading server backup schedules",
			fmt.Sprintf("Server with ID %q does not exist in project %q.", serverId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapSchedulesDatasourceFields(ctx, schedules, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server backup schedules", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server backup schedules read")
}

func mapSchedulesDatasourceFields(ctx context.Context, schedules *serverbackup.GetBackupSchedulesResponse, model *schedulesDataSourceModel, region string) error {
	if schedules == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	tflog.Debug(ctx, "response", map[string]any{"schedules": schedules})
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()

	model.ID = utils.BuildInternalTerraformId(projectId, region, serverId)
	model.Region = types.StringValue(region)

	for _, schedule := range *schedules.Items {
		scheduleState := schedulesDatasourceItemModel{
			BackupScheduleId: types.Int64Value(*schedule.Id),
			Name:             types.StringValue(*schedule.Name),
			Rrule:            types.StringValue(*schedule.Rrule),
			Enabled:          types.BoolValue(*schedule.Enabled),
		}
		ids, diags := types.ListValueFrom(ctx, types.StringType, schedule.BackupProperties.VolumeIds)
		if diags.HasError() {
			return fmt.Errorf("failed to map hosts: %w", core.DiagsToError(diags))
		}
		scheduleState.BackupProperties = &scheduleBackupPropertiesModel{
			BackupName:      types.StringValue(*schedule.BackupProperties.Name),
			RetentionPeriod: types.Int64Value(*schedule.BackupProperties.RetentionPeriod),
			VolumeIds:       ids,
		}
		model.Items = append(model.Items, scheduleState)
	}
	return nil
}
