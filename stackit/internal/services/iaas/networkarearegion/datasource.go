package networkarearegion

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &networkAreaRegionDataSource{}
)

// NewNetworkAreaRegionDataSource is a helper function to simplify the provider implementation.
func NewNetworkAreaRegionDataSource() datasource.DataSource {
	return &networkAreaRegionDataSource{}
}

// networkAreaRegionDataSource is the data source implementation.
type networkAreaRegionDataSource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *networkAreaRegionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_area_region"
}

func (d *networkAreaRegionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (d *networkAreaRegionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Network area region data source schema."

	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`organization_id`,`network_area_id`,`region`\".",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT organization ID to which the network area is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_area_id": schema.StringAttribute{
				Description: "The network area ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				// the region cannot be found, so it has to be passed
				Optional: true,
			},
			"ipv4": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The regional IPv4 config of a network area.",
				Attributes: map[string]schema.Attribute{
					"default_nameservers": schema.ListAttribute{
						Description: "List of DNS Servers/Nameservers.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"network_ranges": schema.ListNestedAttribute{
						Description: "List of Network ranges.",
						Computed:    true,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
							listvalidator.SizeAtMost(64),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"network_range_id": schema.StringAttribute{
									Computed: true,
									Validators: []validator.String{
										validate.UUID(),
										validate.NoSeparator(),
									},
								},
								"prefix": schema.StringAttribute{
									Description: "Classless Inter-Domain Routing (CIDR).",
									Computed:    true,
								},
							},
						},
					},
					"transfer_network": schema.StringAttribute{
						Description: "IPv4 Classless Inter-Domain Routing (CIDR).",
						Computed:    true,
					},
					"default_prefix_length": schema.Int64Attribute{
						Description: "The default prefix length for networks in the network area.",
						Computed:    true,
					},
					"max_prefix_length": schema.Int64Attribute{
						Description: "The maximal prefix length for networks in the network area.",
						Computed:    true,
					},
					"min_prefix_length": schema.Int64Attribute{
						Description: "The minimal prefix length for networks in the network area.",
						Computed:    true,
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *networkAreaRegionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)

	networkAreaRegionResp, err := d.client.GetNetworkAreaRegion(ctx, organizationId, networkAreaId, region).Execute()
	if err != nil {
		utils.LogError(ctx, &resp.Diagnostics, err, "Reading network area region", fmt.Sprintf("Region configuration for %q for network area %q does not exist.", region, networkAreaId), nil)
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, networkAreaRegionResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area region", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area region read")
}
