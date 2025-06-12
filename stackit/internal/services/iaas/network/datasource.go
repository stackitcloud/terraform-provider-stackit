package network

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &networkDataSource{}
)

type DataSourceModel struct {
	Id               types.String `tfsdk:"id"` // needed by TF
	ProjectId        types.String `tfsdk:"project_id"`
	NetworkId        types.String `tfsdk:"network_id"`
	Name             types.String `tfsdk:"name"`
	Nameservers      types.List   `tfsdk:"nameservers"`
	IPv4Gateway      types.String `tfsdk:"ipv4_gateway"`
	IPv4Nameservers  types.List   `tfsdk:"ipv4_nameservers"`
	IPv4Prefix       types.String `tfsdk:"ipv4_prefix"`
	IPv4PrefixLength types.Int64  `tfsdk:"ipv4_prefix_length"`
	Prefixes         types.List   `tfsdk:"prefixes"`
	IPv4Prefixes     types.List   `tfsdk:"ipv4_prefixes"`
	IPv6Gateway      types.String `tfsdk:"ipv6_gateway"`
	IPv6Nameservers  types.List   `tfsdk:"ipv6_nameservers"`
	IPv6Prefix       types.String `tfsdk:"ipv6_prefix"`
	IPv6PrefixLength types.Int64  `tfsdk:"ipv6_prefix_length"`
	IPv6Prefixes     types.List   `tfsdk:"ipv6_prefixes"`
	PublicIP         types.String `tfsdk:"public_ip"`
	Labels           types.Map    `tfsdk:"labels"`
	Routed           types.Bool   `tfsdk:"routed"`
}

// NewNetworkDataSource is a helper function to simplify the provider implementation.
func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

// networkDataSource is the data source implementation.
type networkDataSource struct {
	client *iaas.APIClient
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

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
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
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *networkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	networkResp, err := d.client.GetNetwork(ctx, projectId, networkId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading network",
			fmt.Sprintf("Network with ID %q does not exist in project %q.", networkId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapDataSourceFields(ctx, networkResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network read")
}

func mapDataSourceFields(ctx context.Context, networkResp *iaas.Network, model *DataSourceModel) error {
	if networkResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkId string
	if model.NetworkId.ValueString() != "" {
		networkId = model.NetworkId.ValueString()
	} else if networkResp.NetworkId != nil {
		networkId = *networkResp.NetworkId
	} else {
		return fmt.Errorf("network id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), networkId)

	labels, err := iaasUtils.MapLabels(ctx, networkResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	// IPv4

	if networkResp.Nameservers == nil {
		model.Nameservers = types.ListNull(types.StringType)
		model.IPv4Nameservers = types.ListNull(types.StringType)
	} else {
		respNameservers := *networkResp.Nameservers
		modelNameservers, err := utils.ListValuetoStringSlice(model.Nameservers)
		modelIPv4Nameservers, errIpv4 := utils.ListValuetoStringSlice(model.IPv4Nameservers)
		if err != nil {
			return fmt.Errorf("get current network nameservers from model: %w", err)
		}
		if errIpv4 != nil {
			return fmt.Errorf("get current IPv4 network nameservers from model: %w", errIpv4)
		}

		reconciledNameservers := utils.ReconcileStringSlices(modelNameservers, respNameservers)
		reconciledIPv4Nameservers := utils.ReconcileStringSlices(modelIPv4Nameservers, respNameservers)

		nameserversTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledNameservers)
		ipv4NameserversTF, ipv4Diags := types.ListValueFrom(ctx, types.StringType, reconciledIPv4Nameservers)
		if diags.HasError() {
			return fmt.Errorf("map network nameservers: %w", core.DiagsToError(diags))
		}
		if ipv4Diags.HasError() {
			return fmt.Errorf("map IPv4 network nameservers: %w", core.DiagsToError(ipv4Diags))
		}

		model.Nameservers = nameserversTF
		model.IPv4Nameservers = ipv4NameserversTF
	}

	if networkResp.Prefixes == nil {
		model.Prefixes = types.ListNull(types.StringType)
		model.IPv4Prefixes = types.ListNull(types.StringType)
	} else {
		respPrefixes := *networkResp.Prefixes
		prefixesTF, diags := types.ListValueFrom(ctx, types.StringType, respPrefixes)
		if diags.HasError() {
			return fmt.Errorf("map network prefixes: %w", core.DiagsToError(diags))
		}
		if len(respPrefixes) > 0 {
			model.IPv4Prefix = types.StringValue(respPrefixes[0])
			_, netmask, err := net.ParseCIDR(respPrefixes[0])
			if err != nil {
				// silently ignore parsing error for the netmask
				model.IPv4PrefixLength = types.Int64Null()
			} else {
				ones, _ := netmask.Mask.Size()
				model.IPv4PrefixLength = types.Int64Value(int64(ones))
			}
		}

		model.Prefixes = prefixesTF
		model.IPv4Prefixes = prefixesTF
	}

	if networkResp.Gateway != nil {
		model.IPv4Gateway = types.StringPointerValue(networkResp.GetGateway())
	} else {
		model.IPv4Gateway = types.StringNull()
	}

	// IPv6

	if networkResp.NameserversV6 == nil {
		model.IPv6Nameservers = types.ListNull(types.StringType)
	} else {
		respIPv6Nameservers := *networkResp.NameserversV6
		modelIPv6Nameservers, errIpv6 := utils.ListValuetoStringSlice(model.IPv6Nameservers)
		if errIpv6 != nil {
			return fmt.Errorf("get current IPv6 network nameservers from model: %w", errIpv6)
		}

		reconciledIPv6Nameservers := utils.ReconcileStringSlices(modelIPv6Nameservers, respIPv6Nameservers)

		ipv6NameserversTF, ipv6Diags := types.ListValueFrom(ctx, types.StringType, reconciledIPv6Nameservers)
		if ipv6Diags.HasError() {
			return fmt.Errorf("map IPv6 network nameservers: %w", core.DiagsToError(ipv6Diags))
		}

		model.IPv6Nameservers = ipv6NameserversTF
	}

	if networkResp.PrefixesV6 == nil {
		model.IPv6Prefixes = types.ListNull(types.StringType)
	} else {
		respPrefixesV6 := *networkResp.PrefixesV6
		prefixesV6TF, diags := types.ListValueFrom(ctx, types.StringType, respPrefixesV6)
		if diags.HasError() {
			return fmt.Errorf("map network IPv6 prefixes: %w", core.DiagsToError(diags))
		}
		if len(respPrefixesV6) > 0 {
			model.IPv6Prefix = types.StringValue(respPrefixesV6[0])
			_, netmask, err := net.ParseCIDR(respPrefixesV6[0])
			if err != nil {
				// silently ignore parsing error for the netmask
				model.IPv6PrefixLength = types.Int64Null()
			} else {
				ones, _ := netmask.Mask.Size()
				model.IPv6PrefixLength = types.Int64Value(int64(ones))
			}
		}
		model.IPv6Prefixes = prefixesV6TF
	}

	if networkResp.Gatewayv6 != nil {
		model.IPv6Gateway = types.StringPointerValue(networkResp.GetGatewayv6())
	} else {
		model.IPv6Gateway = types.StringNull()
	}

	model.NetworkId = types.StringValue(networkId)
	model.Name = types.StringPointerValue(networkResp.Name)
	model.PublicIP = types.StringPointerValue(networkResp.PublicIp)
	model.Labels = labels
	model.Routed = types.BoolPointerValue(networkResp.Routed)

	return nil
}
