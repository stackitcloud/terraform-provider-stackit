package database

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdk "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	sqlserverflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &databaseDataSource{}
	_ datasource.DataSourceWithConfigure = &databaseDataSource{}
)

func NewDatabaseDataSource() datasource.DataSource {
	return &databaseDataSource{}
}

type databaseDataSource struct {
	client       *sdk.APIClient
	providerData core.ProviderData
}

type SharedModel struct {
	Id            types.String `tfsdk:"id"`
	ProjectId     types.String `tfsdk:"project_id"`
	Region        types.String `tfsdk:"region"`
	InstanceId    types.String `tfsdk:"instance_id"`
	Name          types.String `tfsdk:"name"`
	Collation     types.String `tfsdk:"collation"`
	Compatibility types.Int64  `tfsdk:"compatibility"`
	Owner         types.String `tfsdk:"owner"`
	DatabaseId    types.Int64  `tfsdk:"database_id"`
}

type DatasourceModel struct {
	SharedModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *databaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflex_database"
}

func (d *databaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := sqlserverflexUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "SqlserverFlex client configured")
}

func (d *databaseDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: descriptionDatasource,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptionId,
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptionProjectId,
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptionRegion,
				Optional:    true,
				Computed:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: descriptionInstanceId,
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptionName,
				Required:    true,
			},
			"collation": schema.StringAttribute{
				Description: descriptionCollation,
				Computed:    true,
			},
			"compatibility": schema.Int64Attribute{
				Description: descriptionCompatibility,
				Computed:    true,
			},
			"owner": schema.StringAttribute{
				Description: descriptionOwner,
				Computed:    true,
			},
			"database_id": schema.Int64Attribute{
				Description: descriptionDatabaseId,
				Computed:    true,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

func (d *databaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	name := model.Name.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)

	ctx = core.InitProviderContext(ctx)

	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceId,
		"name":        model.Name,
	})

	apiResp, err := d.client.DefaultAPI.GetDatabase(ctx, projectId, region, instanceId, name).Execute()
	if err != nil {
		utils.LogError(ctx, &resp.Diagnostics, err, "read SqlserverFlex database",
			fmt.Sprintf("databse with name %q does not exist in instance %q", name, instanceId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q, or instance with ID %q not found or forbidden access", projectId, instanceId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, apiResp, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading database", fmt.Sprintf("mapping response: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SqlserverFlex database read")
}
