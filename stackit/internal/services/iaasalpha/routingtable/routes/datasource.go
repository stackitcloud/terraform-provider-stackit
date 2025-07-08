package routes

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	shared "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/shared"
	iaasalphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &routingTableRoutesDataSource{}
)

type RoutingTableRoutesDataSourceModel struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	OrganizationId types.String `tfsdk:"organization_id"`
	NetworkAreaId  types.String `tfsdk:"network_area_id"`
	RoutingTableId types.String `tfsdk:"routing_table_id"`
	Region         types.String `tfsdk:"region"`
	Routes         types.List   `tfsdk:"routes"`
}

// NewRoutingTableRoutesDataSource is a helper function to simplify the provider implementation.
func NewRoutingTableRoutesDataSource() datasource.DataSource {
	return &routingTableRoutesDataSource{}
}

// routingTableDataSource is the data source implementation.
type routingTableRoutesDataSource struct {
	client       *iaasalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *routingTableRoutesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_table_routes"
}

func (d *routingTableRoutesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &d.providerData, features.RoutingTablesExperiment, "stackit_routing_table_routes", core.Datasource, &resp.Diagnostics)
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
func (d *routingTableRoutesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Routing table routes datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.RoutingTablesExperiment, core.Datasource),
		Attributes:          shared.GetRoutesDataSourceAttributes(),
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *routingTableRoutesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model RoutingTableRoutesDataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	networkAreaId := model.NetworkAreaId.ValueString()
	routingTableId := model.RoutingTableId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)

	routesResp, err := d.client.ListRoutesOfRoutingTable(ctx, organizationId, networkAreaId, region, routingTableId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading routes of routing table",
			fmt.Sprintf("Routing table with ID %q in network area with ID %q does not exist in organization %q.", routingTableId, networkAreaId, organizationId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Organization with ID %q not found or forbidden access", organizationId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapDataSourceRoutingTableRoutes(ctx, routesResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading routing table routes", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table routes read")
}

func mapDataSourceRoutingTableRoutes(ctx context.Context, routes *iaasalpha.RouteListResponse, model *RoutingTableRoutesDataSourceModel, region string) error {
	if routes == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	if routes.Items == nil {
		return fmt.Errorf("items input is nil")
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	routingTableId := model.RoutingTableId.ValueString()

	idParts := []string{organizationId, region, networkAreaId, routingTableId}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	itemsList := []attr.Value{}
	for i, route := range *routes.Items {
		var routeModel shared.RouteReadModel
		err := shared.MapRouteReadModel(ctx, &route, &routeModel)
		if err != nil {
			return fmt.Errorf("mapping route: %w", err)
		}

		routeMap := map[string]attr.Value{
			"route_id":    routeModel.RouteId,
			"destination": routeModel.Destination,
			"next_hop":    routeModel.NextHop,
			"labels":      routeModel.Labels,
			"created_at":  routeModel.CreatedAt,
			"updated_at":  routeModel.UpdatedAt,
		}

		routeTF, diags := types.ObjectValue(shared.RouteReadModelTypes(), routeMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}
		itemsList = append(itemsList, routeTF)
	}

	routesListTF, diags := types.ListValue(types.ObjectType{AttrTypes: shared.RouteReadModelTypes()}, itemsList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	model.Region = types.StringValue(region)
	model.Routes = routesListTF

	return nil
}
