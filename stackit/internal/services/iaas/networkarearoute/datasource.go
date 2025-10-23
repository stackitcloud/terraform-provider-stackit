package networkarearoute

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

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
	_ datasource.DataSource = &networkAreaRouteDataSource{}
)

// NewNetworkAreaRouteDataSource is a helper function to simplify the provider implementation.
func NewNetworkAreaRouteDataSource() datasource.DataSource {
	return &networkAreaRouteDataSource{}
}

// networkDataSource is the data source implementation.
type networkAreaRouteDataSource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *networkAreaRouteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_area_route"
}

func (d *networkAreaRouteDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the data source.
func (d *networkAreaRouteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Network area route data resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source ID. It is structured as \"`organization_id`,`region`,`network_area_id`,`network_area_route_id`\".",
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
				Description: "The network area ID to which the network area route is associated.",
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
			"network_area_route_id": schema.StringAttribute{
				Description: "The network area route ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"next_hop": schema.StringAttribute{
				Description: "The IP address of the routing system, that will route the prefix configured. Should be a valid IPv4 address.",
				Computed:    true,
			},
			"prefix": schema.StringAttribute{
				Description: "The network, that is reachable though the Next Hop. Should use CIDR notation.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *networkAreaRouteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	networkAreaRouteId := model.NetworkAreaRouteId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_route_id", networkAreaRouteId)

	networkAreaRouteResp, err := d.client.GetNetworkAreaRoute(ctx, organizationId, networkAreaId, region, networkAreaRouteId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading network area route",
			fmt.Sprintf("Network area route with ID %q or network area with ID %q does not exist in organization %q.", networkAreaRouteId, networkAreaId, organizationId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Organization with ID %q not found or forbidden access", organizationId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, networkAreaRouteResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area route read")
}
