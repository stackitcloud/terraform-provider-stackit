package gateway

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
	_ datasource.DataSource              = &vpnGatewayDataSource{}
	_ datasource.DataSourceWithConfigure = &vpnGatewayDataSource{}
)

type vpnGatewayDataSource struct {
	client       *vpn.APIClient
	providerData core.ProviderData
}

func NewVPNGatewayDataSource() datasource.DataSource {
	return &vpnGatewayDataSource{}
}

func (d *vpnGatewayDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *vpnGatewayDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_gateway"
}

func (d *vpnGatewayDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("VPN Gateway data source schema. %s", core.DatasourceRegionFallbackDocstring),
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
			"plan_id": schema.StringAttribute{
				Description: schemaDescriptions["plan_id"],
				Computed:    true,
			},
			"routing_type": schema.StringAttribute{
				Description: schemaDescriptions["routing_type"],
				Computed:    true,
			},
			"availability_zones": schema.SingleNestedAttribute{
				Description: schemaDescriptions["availability_zones"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"tunnel1": schema.StringAttribute{
						Description: schemaDescriptions["availability_zones_tunnel_1"],
						Computed:    true,
					},
					"tunnel2": schema.StringAttribute{
						Description: schemaDescriptions["availability_zones_tunnel_2"],
						Computed:    true,
					},
				},
			},
			"bgp": schema.SingleNestedAttribute{
				Description: schemaDescriptions["bgp"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"local_asn": schema.Int64Attribute{
						Description: schemaDescriptions["bgp_local_asn"],
						Computed:    true,
					},
					"override_advertised_routes": schema.ListAttribute{
						Description: schemaDescriptions["bgp_override_advertised_routes"],
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
			"labels": schema.MapAttribute{
				Description: schemaDescriptions["labels"],
				Computed:    true,
				ElementType: types.StringType,
			},
			"state": schema.StringAttribute{
				Description: schemaDescriptions["state"],
				Computed:    true,
			},
		},
	}
}

func (d *vpnGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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

	gatewayResponse, err := d.client.DefaultAPI.GetGateway(ctx, projectId, vpn.Region(region), gatewayId).Execute()
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
