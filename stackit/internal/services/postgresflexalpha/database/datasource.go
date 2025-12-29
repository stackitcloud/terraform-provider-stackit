package postgresflexalpha

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	postgresflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &databaseDataSource{}
)

// NewDatabaseDataSource is a helper function to simplify the provider implementation.
func NewDatabaseDataSource() datasource.DataSource {
	return &databaseDataSource{}
}

// databaseDataSource is the data source implementation.
type databaseDataSource struct {
	client       *postgresflexalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *databaseDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_postgresflexalpha_database"
}

// Configure adds the provider configured client to the data source.
func (r *databaseDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := postgresflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Postgres Flex database client configured")
}

// Schema defines the schema for the data source.
func (r *databaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "Postgres Flex database resource schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`,`database_id`\".",
		"database_id": "Database ID.",
		"instance_id": "ID of the Postgres Flex instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"name":        "Database name.",
		"owner":       "Username of the database owner.",
		"region":      "The resource region. If not defined, the provider region is used.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"database_id": schema.StringAttribute{
				Description: descriptions["database_id"],
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"owner": schema.StringAttribute{
				Description: descriptions["owner"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				// the region cannot be found, so it has to be passed
				Optional:    true,
				Description: descriptions["region"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *databaseDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	databaseId := model.DatabaseId.ValueInt64()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "database_id", databaseId)
	ctx = tflog.SetField(ctx, "region", region)

	databaseResp, err := getDatabase(ctx, r.client, projectId, region, instanceId, databaseId)
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading database",
			fmt.Sprintf(
				"Database with ID %q or instance with ID %q does not exist in project %q.",
				databaseId,
				instanceId,
				projectId,
			),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema and populate Computed attribute values
	err = mapFields(databaseResp, &model, region)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading database",
			fmt.Sprintf("Processing API payload: %v", err),
		)
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex database read")
}
