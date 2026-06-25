package connection

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/vpn/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = (*vpnConnectionDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vpnConnectionDataSource)(nil)
)

type DataSourceTunnelModel struct {
	RemoteAddress types.String          `tfsdk:"remote_address"`
	Phase1        *Phase1Model          `tfsdk:"phase1"`
	Phase2        *Phase2Model          `tfsdk:"phase2"`
	Peering       *PeeringConfigModel   `tfsdk:"peering"`
	Bgp           *BGPTunnelConfigModel `tfsdk:"bgp"`
}

type DataSourceModel struct {
	CommonModel
	Tunnel1 *DataSourceTunnelModel `tfsdk:"tunnel1"`
	Tunnel2 *DataSourceTunnelModel `tfsdk:"tunnel2"`
}

type vpnConnectionDataSource struct {
	client       *vpn.APIClient
	providerData core.ProviderData
}

func NewVPNConnectionDataSource() datasource.DataSource {
	return &vpnConnectionDataSource{}
}

func (d *vpnConnectionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	d.providerData = providerData
	tflog.Info(ctx, "VPN connection data source configured")
}

func (d *vpnConnectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_connection"
}

func DataSourceTunnelSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"remote_address": schema.StringAttribute{
				Description: "Remote peer IPv4 address for this tunnel.",
				Computed:    true,
			},
			"phase1": schema.SingleNestedAttribute{
				Description: "IKE Phase 1 configuration.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"dh_groups": schema.ListAttribute{
						Description: "Diffie-Hellman groups.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"encryption_algorithms": schema.ListAttribute{
						Description: "Encryption algorithms.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"integrity_algorithms": schema.ListAttribute{
						Description: "Integrity/hash algorithms.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"rekey_time": schema.Int32Attribute{
						Description: "IKE re-keying time in seconds.",
						Computed:    true,
					},
				},
			},
			"phase2": schema.SingleNestedAttribute{
				Description: "IKE Phase 2 configuration.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"dh_groups": schema.ListAttribute{
						Description: "Diffie-Hellman groups for PFS.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"encryption_algorithms": schema.ListAttribute{
						Description: "Encryption algorithms.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"integrity_algorithms": schema.ListAttribute{
						Description: "Integrity/hash algorithms.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"rekey_time": schema.Int32Attribute{
						Description: "Child SA re-keying time in seconds.",
						Computed:    true,
					},
					"start_action": schema.StringAttribute{
						Description: "Start action (none or start).",
						Computed:    true,
					},
					"dpd_action": schema.StringAttribute{
						Description: "DPD timeout action (clear or restart).",
						Computed:    true,
					},
				},
			},
			"peering": schema.SingleNestedAttribute{
				Description: "Tunnel interface peering configuration.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"local_address": schema.StringAttribute{
						Description: "Local tunnel interface IPv4 address.",
						Computed:    true,
					},
					"remote_address": schema.StringAttribute{
						Description: "Remote tunnel interface IPv4 address.",
						Computed:    true,
					},
				},
			},
			"bgp": schema.SingleNestedAttribute{
				Description: "BGP configuration for this tunnel.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"remote_asn": schema.Int64Attribute{
						Description: "Remote AS number.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *vpnConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("VPN Connection data source schema. %s", core.DatasourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`gateway_id`,`connection_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "STACKIT region.",
				Computed:    true,
			},
			"gateway_id": schema.StringAttribute{
				Description: "The UUID of the parent VPN gateway.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"connection_id": schema.StringAttribute{
				Description: "The server-generated UUID of the VPN connection.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "A user-friendly name for the connection.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether this connection is enabled.",
				Computed:    true,
			},
			"remote_subnet": schema.ListAttribute{
				Description: "List of remote IPv4 CIDRs accessible via this connection.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"local_subnet": schema.ListAttribute{
				Description: "List of local IPv4 CIDRs to route through this connection.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"static_routes": schema.ListAttribute{
				Description: "List of static routes (IPv4 CIDRs) for route-based VPN.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"tunnel1": DataSourceTunnelSchema(),
			"tunnel2": DataSourceTunnelSchema(),
			"labels": schema.MapAttribute{
				Description: "Map of custom labels.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *vpnConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	gatewayId := model.GatewayID.ValueString()
	connectionId := model.ConnectionID.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "connection_id", connectionId)

	connResp, err := d.client.DefaultAPI.GetGatewayConnection(ctx, projectId, region, gatewayId, connectionId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN connection", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapDataSourceFields(ctx, connResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN connection", fmt.Sprintf("Processing response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN connection read", map[string]any{
		"connection_id": connectionId,
	})
}

func mapDataSourceFields(ctx context.Context, conn connectionResponse, model *DataSourceModel, region string) error {
	err := mapCommonFields(ctx, conn, &model.CommonModel, region)
	if err != nil {
		return err
	}

	tunnel1 := conn.GetTunnel1()
	if model.Tunnel1 == nil {
		model.Tunnel1 = &DataSourceTunnelModel{}
	}
	err = mapTunnel(ctx, &tunnel1, model.Tunnel1)
	if err != nil {
		return fmt.Errorf("mapping tunnel1: %w", err)
	}

	tunnel2 := conn.GetTunnel2()
	if model.Tunnel2 == nil {
		model.Tunnel2 = &DataSourceTunnelModel{}
	}
	err = mapTunnel(ctx, &tunnel2, model.Tunnel2)
	if err != nil {
		return fmt.Errorf("mapping tunnel2: %w", err)
	}

	return nil
}
