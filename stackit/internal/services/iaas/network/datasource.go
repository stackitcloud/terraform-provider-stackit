package network

import (
	"context"

	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/v1network"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/v2network"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &networkDataSource{}
)

// NewNetworkDataSource is a helper function to simplify the provider implementation.
func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

// networkDataSource is the data source implementation.
type networkDataSource struct {
	client *iaas.APIClient
	// alphaClient will be used in case the experimental flag "network" is set
	alphaClient    *iaasalpha.APIClient
	isExperimental bool
	providerData   core.ProviderData
}

// Metadata returns the data source type name.
func (d *networkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (d *networkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.isExperimental = features.CheckExperimentEnabledWithoutError(ctx, &d.providerData, experiment, "stackit_network", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.isExperimental {
		alphaApiClient := iaasAlphaUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		d.alphaClient = alphaApiClient
	} else {
		apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		d.client = apiClient
	}
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the data source.
func (d *networkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`network_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the network is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_id": schema.StringAttribute{
				Description: "The network ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the network.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"nameservers": schema.ListAttribute{
				Description:        "The nameservers of the network. This field is deprecated and will be removed soon, use `ipv4_nameservers` to configure the nameservers for IPv4.",
				DeprecationMessage: "Use `ipv4_nameservers` to configure the nameservers for IPv4.",
				Computed:           true,
				ElementType:        types.StringType,
			},
			"ipv4_gateway": schema.StringAttribute{
				Description: "The IPv4 gateway of a network. If not specified, the first IP of the network will be assigned as the gateway.",
				Computed:    true,
			},
			"ipv4_nameservers": schema.ListAttribute{
				Description: "The IPv4 nameservers of the network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipv4_prefix": schema.StringAttribute{
				Description:        "The IPv4 prefix of the network (CIDR).",
				DeprecationMessage: "The API supports reading multiple prefixes. So using the attribute 'ipv4_prefixes` should be preferred. This attribute will be populated with the first element from the list",
				Computed:           true,
			},
			"ipv4_prefix_length": schema.Int64Attribute{
				Description: "The IPv4 prefix length of the network.",
				Computed:    true,
			},
			"prefixes": schema.ListAttribute{
				Description:        "The prefixes of the network. This field is deprecated and will be removed soon, use `ipv4_prefixes` to read the prefixes of the IPv4 networks.",
				DeprecationMessage: "Use `ipv4_prefixes` to read the prefixes of the IPv4 networks.",
				Computed:           true,
				ElementType:        types.StringType,
			},
			"ipv4_prefixes": schema.ListAttribute{
				Description: "The IPv4 prefixes of the network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipv6_gateway": schema.StringAttribute{
				Description: "The IPv6 gateway of a network. If not specified, the first IP of the network will be assigned as the gateway.",
				Computed:    true,
			},
			"ipv6_nameservers": schema.ListAttribute{
				Description: "The IPv6 nameservers of the network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipv6_prefix": schema.StringAttribute{
				Description:        "The IPv6 prefix of the network (CIDR).",
				DeprecationMessage: "The API supports reading multiple prefixes. So using the attribute 'ipv6_prefixes` should be preferred. This attribute will be populated with the first element from the list",
				Computed:           true,
			},
			"ipv6_prefix_length": schema.Int64Attribute{
				Description: "The IPv6 prefix length of the network.",
				Computed:    true,
			},
			"ipv6_prefixes": schema.ListAttribute{
				Description: "The IPv6 prefixes of the network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"public_ip": schema.StringAttribute{
				Description: "The public IP of the network.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
			"routed": schema.BoolAttribute{
				Description: "Shows if the network is routed and therefore accessible from other networks.",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				// the region cannot be found, so it has to be passed
				Optional:    true,
				Description: "Can only be used when experimental \"network\" is set. This is likely going to undergo significant changes or be removed in the future.\nThe resource region. If not defined, the provider region is used.",
			},
			"routing_table_id": schema.StringAttribute{
				Description: "Can only be used when experimental \"network\" is set. This is likely going to undergo significant changes or be removed in the future. Use it at your own discretion.\nThe ID of the routing table associated with the network.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *networkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	if !d.isExperimental {
		v1network.DatasourceRead(ctx, req, resp, d.client)
	} else {
		v2network.DatasourceRead(ctx, req, resp, d.alphaClient, d.providerData)
	}
}
