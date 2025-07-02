package schedule

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	serverbackupUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serverbackup/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
)

// scheduleDataSourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var scheduleDataSourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &scheduleDataSource{}
)

// NewScheduleDataSource is a helper function to simplify the provider implementation.
func NewScheduleDataSource() datasource.DataSource {
	return &scheduleDataSource{}
}

// scheduleDataSource is the data source implementation.
type scheduleDataSource struct {
	client       *serverbackup.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *scheduleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_backup_schedule"
}

// Configure adds the provider configured client to the data source.
func (r *scheduleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !scheduleDataSourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_server_backup_schedule", "data source")
		if resp.Diagnostics.HasError() {
			return
		}
		scheduleDataSourceBetaCheckDone = true
	}

	apiClient := serverbackupUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Server backup client configured")
}

// Schema defines the schema for the data source.
func (r *scheduleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Server backup schedule datasource schema. Must have a `region` specified in the provider configuration.",
		MarkdownDescription: features.AddBetaDescription("Server backup schedule datasource schema. Must have a `region` specified in the provider configuration.", core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. It is structured as \"`project_id`,`server_id`,`backup_schedule_id`\".",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The schedule name.",
				Computed:    true,
			},
			"backup_schedule_id": schema.Int64Attribute{
				Description: "Backup schedule ID.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT Project ID to which the server is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"server_id": schema.StringAttribute{
				Description: "Server ID for the backup schedule.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
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
			"region": schema.StringAttribute{
				// the region cannot be found, so it has to be passed
				Optional:    true,
				Description: "The resource region. If not defined, the provider region is used.",
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *scheduleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
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
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading server backup schedule",
			fmt.Sprintf("Backup schedule with ID %q or server with ID %q does not exist in project %q.", strconv.FormatInt(backupScheduleId, 10), serverId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, scheduleResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server backup schedule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server backup schedule read")
}
