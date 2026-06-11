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

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/vpn/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = (*vpnConnectionDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vpnConnectionDataSource)(nil)
)

var datasourceSchemaDescriptions = map[string]string{
	"id":            "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`gateway_id`,`connection_id`\".",
	"project_id":    "STACKIT project ID.",
	"region":        "STACKIT region.",
	"gateway_id":    "The UUID of the parent VPN gateway.",
	"connection_id": "The server-generated UUID of the VPN connection.",
	"display_name":  "A user-friendly name for the connection.",
	"enabled":       "Whether this connection is enabled.",
	"remote_subnet": "List of remote IPv4 CIDRs accessible via this connection.",
	"local_subnet":  "List of local IPv4 CIDRs to route through this connection.",
	"static_routes": "List of static routes (IPv4 CIDRs) for route-based VPN.",
	"labels":        "Map of custom labels.",
}

var datasourceTunnelSchemaDescriptions = map[string]string{
	"remote_address":               "Remote peer IPv4 address for this tunnel.",
	"phase1":                       "IKE Phase 1 configuration.",
	"phase1_dh_groups":             "Diffie-Hellman groups.",
	"phase1_encryption_algorithms": "Encryption algorithms.",
	"phase1_integrity_algorithms":  "Integrity/hash algorithms.",
	"phase1_rekey_time":            "IKE re-keying time in seconds.",
	"phase2":                       "IKE Phase 2 configuration.",
	"phase2_dh_groups":             "Diffie-Hellman groups for PFS.",
	"phase2_encryption_algorithms": "Encryption algorithms.",
	"phase2_integrity_algorithms":  "Integrity/hash algorithms.",
	"phase2_rekey_time":            "Child SA re-keying time in seconds.",
	"phase2_start_action":          "Start action (none or start).",
	"phase2_dpd_action":            "DPD timeout action (clear or restart).",
	"peering":                      "Tunnel interface peering configuration.",
	"peering_local_address":        "Local tunnel interface IPv4 address.",
	"peering_remote_address":       "Remote tunnel interface IPv4 address.",
	"bgp":                          "BGP configuration for this tunnel.",
	"bgp_remote_asn":               "Remote AS number.",
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

func (d *vpnConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	tunnelSchema := schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"remote_address": schema.StringAttribute{
				Description: datasourceTunnelSchemaDescriptions["remote_address"],
				Computed:    true,
			},
			"phase1": schema.SingleNestedAttribute{
				Description: datasourceTunnelSchemaDescriptions["phase1"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"dh_groups": schema.ListAttribute{
						Description: datasourceTunnelSchemaDescriptions["phase1_dh_groups"],
						Computed:    true,
						ElementType: types.StringType,
					},
					"encryption_algorithms": schema.ListAttribute{
						Description: datasourceTunnelSchemaDescriptions["phase1_encryption_algorithms"],
						Computed:    true,
						ElementType: types.StringType,
					},
					"integrity_algorithms": schema.ListAttribute{
						Description: datasourceTunnelSchemaDescriptions["phase1_integrity_algorithms"],
						Computed:    true,
						ElementType: types.StringType,
					},
					"rekey_time": schema.Int32Attribute{
						Description: datasourceTunnelSchemaDescriptions["phase1_rekey_time"],
						Computed:    true,
					},
				},
			},
			"phase2": schema.SingleNestedAttribute{
				Description: datasourceTunnelSchemaDescriptions["phase2"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"dh_groups": schema.ListAttribute{
						Description: datasourceTunnelSchemaDescriptions["phase2_dh_groups"],
						Computed:    true,
						ElementType: types.StringType,
					},
					"encryption_algorithms": schema.ListAttribute{
						Description: datasourceTunnelSchemaDescriptions["phase2_encryption_algorithms"],
						Computed:    true,
						ElementType: types.StringType,
					},
					"integrity_algorithms": schema.ListAttribute{
						Description: datasourceTunnelSchemaDescriptions["phase2_integrity_algorithms"],
						Computed:    true,
						ElementType: types.StringType,
					},
					"rekey_time": schema.Int32Attribute{
						Description: datasourceTunnelSchemaDescriptions["phase2_rekey_time"],
						Computed:    true,
					},
					"start_action": schema.StringAttribute{
						Description: datasourceTunnelSchemaDescriptions["phase2_start_action"],
						Computed:    true,
					},
					"dpd_action": schema.StringAttribute{
						Description: datasourceTunnelSchemaDescriptions["phase2_dpd_action"],
						Computed:    true,
					},
				},
			},
			"peering": schema.SingleNestedAttribute{
				Description: datasourceTunnelSchemaDescriptions["peering"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"local_address": schema.StringAttribute{
						Description: datasourceTunnelSchemaDescriptions["peering_local_address"],
						Computed:    true,
					},
					"remote_address": schema.StringAttribute{
						Description: datasourceTunnelSchemaDescriptions["peering_remote_address"],
						Computed:    true,
					},
				},
			},
			"bgp": schema.SingleNestedAttribute{
				Description: datasourceTunnelSchemaDescriptions["bgp"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"remote_asn": schema.Int64Attribute{
						Description: datasourceTunnelSchemaDescriptions["bgp_remote_asn"],
						Computed:    true,
					},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("VPN Connection data source schema. %s", core.DatasourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: datasourceSchemaDescriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: datasourceSchemaDescriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: datasourceSchemaDescriptions["region"],
				Computed:    true,
			},
			"gateway_id": schema.StringAttribute{
				Description: datasourceSchemaDescriptions["gateway_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"connection_id": schema.StringAttribute{
				Description: datasourceSchemaDescriptions["connection_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: datasourceSchemaDescriptions["display_name"],
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: datasourceSchemaDescriptions["enabled"],
				Computed:    true,
			},
			"remote_subnet": schema.ListAttribute{
				Description: datasourceSchemaDescriptions["remote_subnet"],
				Computed:    true,
				ElementType: types.StringType,
			},
			"local_subnet": schema.ListAttribute{
				Description: datasourceSchemaDescriptions["local_subnet"],
				Computed:    true,
				ElementType: types.StringType,
			},
			"static_routes": schema.ListAttribute{
				Description: datasourceSchemaDescriptions["static_routes"],
				Computed:    true,
				ElementType: types.StringType,
			},
			"tunnel1": tunnelSchema,
			"tunnel2": tunnelSchema,
			"labels": schema.MapAttribute{
				Description: datasourceSchemaDescriptions["labels"],
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *vpnConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
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

	err = mapFields(ctx, connResp, &model, region)
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
