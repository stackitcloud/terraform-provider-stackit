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
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/serverupdate"
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
	client       *serverupdate.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *schedulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_update_schedules"
}

// Configure adds the provider configured client to the data source.
func (r *schedulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	r.providerData, ok = req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !schedulesDataSourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_server_update_schedules", "data source")
		if resp.Diagnostics.HasError() {
			return
		}
		schedulesDataSourceBetaCheckDone = true
	}

	var apiClient *serverupdate.APIClient
	var err error
	if r.providerData.ServerUpdateCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "server_update_custom_endpoint", r.providerData.ServerUpdateCustomEndpoint)
		apiClient, err = serverupdate.NewAPIClient(
			config.WithCustomAuth(r.providerData.RoundTripper),
			config.WithEndpoint(r.providerData.ServerUpdateCustomEndpoint),
		)
	} else {
		apiClient, err = serverupdate.NewAPIClient(
			config.WithCustomAuth(r.providerData.RoundTripper),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Server update client configured")
}

// Schema defines the schema for the data source.
func (r *schedulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Server update schedules datasource schema. Must have a `region` specified in the provider configuration.",
		MarkdownDescription: features.AddBetaDescription("Server update schedules datasource schema. Must have a `region` specified in the provider configuration."),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source identifier. It is structured as \"`project_id`,`region`,`server_id`\".",
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
				Description: "Server ID (UUID) to which the update schedule is associated.",
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
						"update_schedule_id": schema.Int64Attribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Description: "The update schedule name.",
							Computed:    true,
						},
						"rrule": schema.StringAttribute{
							Description: "Update schedule described in `rrule` (recurrence rule) format.",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Is the update schedule enabled or disabled.",
							Computed:    true,
						},
						"maintenance_window": schema.Int64Attribute{
							Description: "Maintenance window [1..24].",
							Computed:    true,
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
	UpdateScheduleId  types.Int64  `tfsdk:"update_schedule_id"`
	Name              types.String `tfsdk:"name"`
	Rrule             types.String `tfsdk:"rrule"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	MaintenanceWindow types.Int64  `tfsdk:"maintenance_window"`
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
	var region string
	if utils.IsUndefined(model.Region) {
		region = r.providerData.GetRegion()
	} else {
		region = model.Region.ValueString()
	}
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)

	schedules, err := r.client.ListUpdateSchedules(ctx, projectId, serverId, region).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading server update schedules",
			fmt.Sprintf("Server with ID %q does not exists in project %q.", serverId, projectId),
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server update schedules", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server update schedules read")
}

func mapSchedulesDatasourceFields(ctx context.Context, schedules *serverupdate.GetUpdateSchedulesResponse, model *schedulesDataSourceModel, region string) error {
	if schedules == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	tflog.Debug(ctx, "response", map[string]any{"schedules": schedules})
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()

	idParts := []string{projectId, region, serverId}
	model.ID = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.Region = types.StringValue(region)

	for _, schedule := range *schedules.Items {
		scheduleState := schedulesDatasourceItemModel{
			UpdateScheduleId:  types.Int64Value(*schedule.Id),
			Name:              types.StringValue(*schedule.Name),
			Rrule:             types.StringValue(*schedule.Rrule),
			Enabled:           types.BoolValue(*schedule.Enabled),
			MaintenanceWindow: types.Int64Value(*schedule.MaintenanceWindow),
		}
		model.Items = append(model.Items, scheduleState)
	}
	return nil
}
