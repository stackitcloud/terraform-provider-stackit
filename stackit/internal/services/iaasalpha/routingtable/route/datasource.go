package route

import (
	"context"
	"fmt"
	"net/http"

	shared "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/shared"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasalphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &routingTableRouteDataSource{}
)

// NewRoutingTableRouteDataSource is a helper function to simplify the provider implementation.
func NewRoutingTableRouteDataSource() datasource.DataSource {
	return &routingTableRouteDataSource{}
}

// routingTableRouteDataSource is the data source implementation.
type routingTableRouteDataSource struct {
	client       *iaasalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *routingTableRouteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_table_route"
}

func (d *routingTableRouteDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &d.providerData, features.RoutingTablesExperiment, "stackit_routing_table_route", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := iaasalphaUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the data source.
func (d *routingTableRouteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Routing table route datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.RoutingTablesExperiment, core.Datasource),
		Attributes:          shared.GetRouteDataSourceAttributes(),
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *routingTableRouteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model shared.RouteModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	routeId := model.RouteId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "route_id", routeId)

	routeResp, err := d.client.GetRouteOfRoutingTable(ctx, organizationId, networkAreaId, region, routingTableId, routeId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, err.Error(), err.Error())
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading routing table route",
			fmt.Sprintf("Routing table route with ID %q, routing table with ID %q or network area with ID %q does not exist in organization %q.", routeId, routingTableId, networkAreaId, organizationId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Organization with ID %q not found or forbidden access", organizationId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = shared.MapRouteModel(ctx, routeResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading routing table route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table route read")
}
