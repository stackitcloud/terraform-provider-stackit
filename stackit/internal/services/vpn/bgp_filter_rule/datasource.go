package bgpfilterrule

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
	_ datasource.DataSource              = &bgpFilterRuleDataSource{}
	_ datasource.DataSourceWithConfigure = &bgpFilterRuleDataSource{}
)

type bgpFilterRuleDataSource struct {
	client       *vpn.APIClient
	providerData core.ProviderData
}

func NewVPNBGPFilterRuleDataSource() datasource.DataSource {
	return &bgpFilterRuleDataSource{}
}

func (d *bgpFilterRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *bgpFilterRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_bgp_filter_rule"
}

func (d *bgpFilterRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("VPN BGP filter rule data source schema. %s", core.DatasourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`gateway_id`,`filter_id`,`rule_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID associated with the BGP filter rule.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "STACKIT region name the resource is located in.",
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
			"filter_id": schema.StringAttribute{
				Description: "The UUID of the parent `stackit_vpn_bgp_filter`.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"rule_id": schema.StringAttribute{
				Description: "The server-generated UUID of the rule.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"action": schema.StringAttribute{
				Description: "The action to take if the route matches all criteria.",
				Computed:    true,
			},
			"sequence": schema.Int32Attribute{
				Description: "The evaluation order of the rule.",
				Computed:    true,
			},
			"match": schema.SingleNestedAttribute{
				Description: "Matching criteria.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"as_path_contains_any": schema.ListAttribute{
						Description: "Matches if the AS-PATH contains any one of the listed ASNs.",
						Computed:    true,
						ElementType: types.Int64Type,
					},
					"communities": schema.ListAttribute{
						Description: "Matches if the route carries any one of these BGP standard communities.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"first_asn": schema.Int64Attribute{
						Description: "Matches if the first ASN in the AS-PATH equals this ASN.",
						Computed:    true,
					},
					"max_prefix_length": schema.Int32Attribute{
						Description: "Maximum subnet mask length for matched prefixes.",
						Computed:    true,
					},
					"min_prefix_length": schema.Int32Attribute{
						Description: "Minimum subnet mask length for matched prefixes.",
						Computed:    true,
					},
					"peer": schema.StringAttribute{
						Description: "Matches the exact IPv4 address of the BGP neighbor that advertised the route.",
						Computed:    true,
					},
					"prefixes": schema.ListAttribute{
						Description: "List of IPv4 networks to match.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
			"set": schema.SingleNestedAttribute{
				Description: "BGP attributes applied when `action` is `PERMIT`.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"local_preference": schema.Int32Attribute{
						Description: "BGP LOCAL_PREF set on the route.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *bgpFilterRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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
	filterId := model.FilterId.ValueString()
	ruleId := model.RuleId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "filter_id", filterId)
	ctx = tflog.SetField(ctx, "rule_id", ruleId)

	ruleResp, err := d.client.DefaultAPI.GetGatewayBGPFilterRule(ctx, projectId, region, gatewayId, filterId, ruleId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN BGP filter rule", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, ruleResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN BGP filter rule", fmt.Sprintf("Processing response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN BGP filter rule read", map[string]any{
		"rule_id": ruleId,
	})
}
