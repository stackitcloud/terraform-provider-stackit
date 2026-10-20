package networks

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &networksDataSource{}
)

// NetworksModel represents the model for the plural data source.
type NetworksModel struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	Results   types.List   `tfsdk:"results"`
}

// networkItemModel represents a single network in the list.
type networkItemModel struct {
	NetworkId types.String `tfsdk:"network_id"`
	Name      types.String `tfsdk:"name"`
}

var networkItemModelTypes = map[string]attr.Type{
	"network_id": types.StringType,
	"name":       types.StringType,
}

// NewNetworksDataSource creates a new instance of plural data source.
func NewNetworksDataSource() datasource.DataSource {
	return &networksDataSource{}
}

// networksDataSource is the data source implementation for querying multiple networks.
type networksDataSource struct {
	client       iaas.DefaultAPI
	providerData core.ProviderData
}

func (d *networksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}

func (d *networksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.providerData = providerData
	d.client = clients.IaaSv2Client

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_networks", "datasource")

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "IaaS client configured")
}

func (d *networksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := features.AddBetaDescription(
		"Lists all IaaS networks in a project and region. Results are sorted by image ID.",
		core.Datasource,
	)

	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the networks are associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				Optional:    true,
			},
			"results": schema.ListNestedAttribute{
				Description: "List of networks.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{
							Description: "The network ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the network.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *networksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic
	var model NetworksModel

	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	networksResp, err := d.client.ListNetworks(ctx, projectId, region).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading networks from API",
			fmt.Sprintf("Networks could not be listed for project %q.", projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)

		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, networksResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Mapping networks", err.Error())

		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Networks read")
}

func mapFields(ctx context.Context, networksResp *iaas.NetworkListResponse, model *NetworksModel, region string) error {
	if model == nil {
		return fmt.Errorf("model is nil")
	} else if networksResp == nil {
		return fmt.Errorf("API response is nil")
	} else if networksResp.Items == nil {
		return fmt.Errorf("network items in the API response are nil")
	}

	// sort by ID for stable output across multiple TF runs
	slices.SortFunc(networksResp.Items, func(a, b iaas.Network) int { return strings.Compare(a.Id, b.Id) })

	items := make([]networkItemModel, 0, len(networksResp.Items))
	for _, netResp := range networksResp.Items {
		item := networkItemModel{
			NetworkId: types.StringValue(netResp.Id),
			Name:      types.StringValue(netResp.Name),
		}

		items = append(items, item)
	}

	networksTF, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: networkItemModelTypes}, items)
	if diags.HasError() {
		return fmt.Errorf("converting network results: %w", core.DiagsToError(diags))
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region)
	model.Region = types.StringValue(region)
	model.Results = networksTF

	return nil
}
