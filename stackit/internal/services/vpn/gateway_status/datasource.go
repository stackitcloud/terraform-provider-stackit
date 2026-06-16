package gateway_status

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/vpn/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &vpnGatewayStatusDataSource{}
	_ datasource.DataSourceWithConfigure = &vpnGatewayStatusDataSource{}
)

type vpnGatewayStatusDataSource struct {
	client       *vpn.APIClient
	providerData core.ProviderData
}

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	GatewayId   types.String `tfsdk:"gateway_id"`
	ProjectId   types.String `tfsdk:"project_id"`
	Region      types.String `tfsdk:"region"`
	DisplayName types.String `tfsdk:"display_name"`
	Tunnels     types.List   `tfsdk:"tunnels"`
}

type Tunnel struct {
	InternalNextHopIP types.String `tfsdk:"internal_next_hop_ip"`
	Name              types.String `tfsdk:"name"`
	PublicIP          types.String `tfsdk:"public_ip"`
}

var tunnelsType = map[string]attr.Type{
	"internal_next_hop_ip": basetypes.StringType{},
	"name":                 basetypes.StringType{},
	"public_ip":            basetypes.StringType{},
}

func NewVPNGatewayStatusDataSource() datasource.DataSource {
	return &vpnGatewayStatusDataSource{}
}

func (d *vpnGatewayStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.client = utils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN client configured")
}

func (d *vpnGatewayStatusDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_gateway_status"
}

var schemaDescriptions = map[string]string{
	"id":                          "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`gateway_id`\".",
	"gateway_id":                  "The server-generated UUID of the VPN gateway.",
	"project_id":                  "STACKIT project ID associated with the VPN gateway.",
	"region":                      "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"display_name":                "A user-friendly name for the VPN gateway.",
	"tunnels":                     "List of the VPN tunnels in the gateway.",
	"tunnel_internal_next_hop_ip": "The IPv4 address of the endpoint in the SNA.",
	"tunnel_name":                 fmt.Sprintf("The name of the VPN tunnel. %s", tfutils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(vpn.AllowedVPNTunnelsNameEnumValues)...)),
	"tunnel_public_ip":            "The public IPv4 address of this endpoint.",
}

func (d *vpnGatewayStatusDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("VPN Gateway Status data source schema. %s", core.DatasourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Computed:    true,
			},
			"gateway_id": schema.StringAttribute{
				Description: schemaDescriptions["gateway_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Computed:    true,
			},
			"tunnels": schema.ListNestedAttribute{
				Description: schemaDescriptions["tunnels"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"internal_next_hop_ip": schema.StringAttribute{
							Description: schemaDescriptions["tunnel_internal_next_hop_ip"],
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: schemaDescriptions["tunnel_name"],
							Computed:    true,
						},
						"public_ip": schema.StringAttribute{
							Description: schemaDescriptions["tunnel_public_ip"],
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *vpnGatewayStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	gatewayId := model.GatewayId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)

	gatewayResponse, err := d.client.DefaultAPI.GetGatewayStatus(ctx, projectId, region, gatewayId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN gateway", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, gatewayResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN gateway", fmt.Sprintf("Processing response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN gateway read", map[string]any{
		"gateway_id": gatewayId,
	})
}

func mapFields(ctx context.Context, gatewayStatus *vpn.GatewayStatusResponse, model *Model, region string) error {
	if gatewayStatus == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var gatewayId string
	if model.GatewayId.ValueString() != "" {
		gatewayId = model.GatewayId.ValueString()
	} else if gatewayStatus.Id != nil {
		gatewayId = *gatewayStatus.Id
	} else {
		return fmt.Errorf("gateway id not present")
	}

	model.Id = tfutils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, gatewayId)
	model.GatewayId = types.StringValue(gatewayId)
	model.Region = types.StringValue(region)

	if gatewayStatus.DisplayName != nil {
		model.DisplayName = types.StringValue(*gatewayStatus.DisplayName)
	}

	tfTunnels, err := mapTunnels(ctx, gatewayStatus.Tunnels)
	if err != nil {
		return fmt.Errorf("map tunnels: %w", err)
	} else if tfTunnels != nil {
		model.Tunnels = *tfTunnels
	}

	return nil
}

func mapTunnels(ctx context.Context, vpnTunnels []vpn.VPNTunnels) (*basetypes.ListValue, error) {
	tunnels := []attr.Value{}

	for _, tunnelItem := range vpnTunnels {
		tunnel := Tunnel{}

		if tunnelItem.InternalNextHopIP != nil {
			tunnel.InternalNextHopIP = types.StringValue(string(*tunnelItem.InternalNextHopIP))
		}
		if tunnelItem.Name != nil {
			tunnel.Name = types.StringValue(string(*tunnelItem.Name))
		}
		if tunnelItem.PublicIP != nil {
			tunnel.PublicIP = types.StringValue(string(*tunnelItem.PublicIP))
		}

		tunnelValue, diags := types.ObjectValueFrom(ctx, tunnelsType, tunnel)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping tunnel: %w", core.DiagsToError(diags))
		}

		tunnels = append(tunnels, tunnelValue)
	}

	tfTunnels, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: tunnelsType}, tunnels)
	if diags.HasError() {
		return nil, fmt.Errorf("mapping tunnels: %w", core.DiagsToError(diags))
	}

	return &tfTunnels, nil
}
