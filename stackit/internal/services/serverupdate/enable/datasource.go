package enable

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	serverupdate "github.com/stackitcloud/stackit-sdk-go/services/serverupdate/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	serverUpdateUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serverupdate/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &serverUpdateEnableDataSource{}
)

type DataModel struct {
	Id        types.String `tfsdk:"id"` // needed by TF
	ProjectId types.String `tfsdk:"project_id"`
	ServerId  types.String `tfsdk:"server_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	Region    types.String `tfsdk:"region"`
}

// NewServerUpdateEnableDataSource is a helper function to simplify the provider implementation.
func NewServerUpdateEnableDataSource() datasource.DataSource {
	return &serverUpdateEnableDataSource{}
}

// serverUpdateEnableDataSource is the data source implementation.
type serverUpdateEnableDataSource struct {
	client       *serverupdate.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *serverUpdateEnableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_update_enable"
}

// Configure adds the provider configured client to the data source.
func (d *serverUpdateEnableDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := serverUpdateUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Server update client client configured")
}

// Schema defines the schema for the resource.
func (d *serverUpdateEnableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":       "Server update enable datasource schema. Must have a `region` specified in the provider configuration.",
		"id":         "Terraform's internal resource identifier. It is structured as \"`project_id`,`server_id`,`region`\".",
		"project_id": "STACKIT Project ID to which the server update enable is associated.",
		"server_id":  "Server ID to which the server update enable is associated.",
		"enabled":    "Set to true if the service is enabled.",
		"region":     "The resource region. If not defined, the provider region is used.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"server_id": schema.StringAttribute{
				Description: descriptions["server_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: descriptions["enabled"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// the region cannot be found automatically, so it has to be passed
				Description: descriptions["region"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *serverUpdateEnableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "region", region)

	serviceResp, err := d.client.DefaultAPI.GetServiceResource(ctx, projectId, serverId, region).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading server update enable",
			fmt.Sprintf("Server update enable does not exist for this server %q.", serverId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q or server with ID %q not found or forbidden access", projectId, serverId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapDataFields(serviceResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server update enable", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server update enable read")
}

func mapDataFields(serviceResp *serverupdate.GetUpdateServiceResponse, model *DataModel, region string) error {
	if serviceResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.ServerId.ValueString(), region)
	model.Region = types.StringValue(region)
	model.Enabled = types.BoolPointerValue(serviceResp.Enabled)

	return nil
}
