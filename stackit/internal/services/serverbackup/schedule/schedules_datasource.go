package schedule

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
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
	client *serverbackup.APIClient
}

// Metadata returns the data source type name.
func (r *schedulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_backup_schedules"
}

// Configure adds the provider configured client to the data source.
func (r *schedulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !schedulesDataSourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_server_backup_schedules", "data source")
		if resp.Diagnostics.HasError() {
			return
		}
		schedulesDataSourceBetaCheckDone = true
	}

	var apiClient *serverbackup.APIClient
	var err error
	if providerData.ServerBackupCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "server_backup_custom_endpoint", providerData.ServerBackupCustomEndpoint)
		apiClient, err = serverbackup.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ServerBackupCustomEndpoint),
		)
	} else {
		apiClient, err = serverbackup.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.GetRegion()),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Server backup client configured")
}

// Schema defines the schema for the data source.
func (r *schedulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Server backup schedules datasource schema. Must have a `region` specified in the provider configuration.",
		MarkdownDescription: features.AddBetaDescription("Server backup schedules datasource schema. Must have a `region` specified in the provider configuration."),
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
		},
	}
}

// schedulesDataSourceModel maps the data source schema data.
type schedulesDataSourceModel struct {
	ID        types.String                   `tfsdk:"id"`
	ProjectId types.String                   `tfsdk:"project_id"`
	ServerId  types.String                   `tfsdk:"server_id"`
	Items     []schedulesDatasourceItemModel `tfsdk:"items"`
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
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)

	schedules, err := r.client.ListBackupSchedules(ctx, projectId, serverId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server backup schedules", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapSchedulesDatasourceFields(ctx, schedules, &model)
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

func mapSchedulesDatasourceFields(ctx context.Context, schedules *serverbackup.GetBackupSchedulesResponse, model *schedulesDataSourceModel) error {
	if schedules == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	tflog.Debug(ctx, "response", map[string]any{"schedules": schedules})
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()

	idParts := []string{projectId, serverId}
	model.ID = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

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
