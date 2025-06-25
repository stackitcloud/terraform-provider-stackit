package shared

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
)

type RouteReadModel struct {
	RouteId     types.String `tfsdk:"route_id"`
	Destination types.Object `tfsdk:"destination"`
	NextHop     types.Object `tfsdk:"next_hop"`
	Labels      types.Map    `tfsdk:"labels"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func RouteReadModelTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"route_id":    types.StringType,
		"destination": types.ObjectType{AttrTypes: RouteDestinationTypes},
		"next_hop":    types.ObjectType{AttrTypes: RouteNextHopTypes},
		"labels":      types.MapType{ElemType: types.StringType},
		"created_at":  types.StringType,
		"updated_at":  types.StringType,
	}
}

type RouteModel struct {
	RouteReadModel
	Id             types.String `tfsdk:"id"` // needed by TF
	OrganizationId types.String `tfsdk:"organization_id"`
	RoutingTableId types.String `tfsdk:"routing_table_id"`
	NetworkAreaId  types.String `tfsdk:"network_area_id"`
	Region         types.String `tfsdk:"region"`
}

func RouteModelTypes() map[string]attr.Type {
	modelTypes := RouteReadModelTypes()
	modelTypes["id"] = types.StringType
	modelTypes["organization_id"] = types.StringType
	modelTypes["routing_table_id"] = types.StringType
	modelTypes["network_area_id"] = types.StringType
	modelTypes["region"] = types.StringType
	return modelTypes
}

// RouteDestination is the struct corresponding to RouteReadModel.Destination
type RouteDestination struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

// RouteDestinationTypes Types corresponding to routeDestination
var RouteDestinationTypes = map[string]attr.Type{
	"type":  types.StringType,
	"value": types.StringType,
}

// RouteNextHop is the struct corresponding to RouteReadModel.NextHop
type RouteNextHop struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

// RouteNextHopTypes Types corresponding to routeNextHop
var RouteNextHopTypes = map[string]attr.Type{
	"type":  types.StringType,
	"value": types.StringType,
}

func MapRouteModel(ctx context.Context, route *iaasalpha.Route, model *RouteModel, region string) error {
	if route == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	err := MapRouteReadModel(ctx, route, &model.RouteReadModel)
	if err != nil {
		return err
	}

	idParts := []string{
		model.OrganizationId.ValueString(),
		region,
		model.NetworkAreaId.ValueString(),
		model.RoutingTableId.ValueString(),
		model.RouteId.ValueString(),
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.Region = types.StringValue(region)

	return nil
}

func MapRouteReadModel(ctx context.Context, route *iaasalpha.Route, model *RouteReadModel) error {
	if route == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var routeId string
	if model.RouteId.ValueString() != "" {
		routeId = model.RouteId.ValueString()
	} else if route.Id != nil {
		routeId = *route.Id
	} else {
		return fmt.Errorf("routing table route id not present")
	}

	labels, err := iaasUtils.MapLabels(ctx, route.Labels, model.Labels)
	if err != nil {
		return err
	}

	// created at and updated at
	createdAtTF, updatedAtTF := types.StringNull(), types.StringNull()
	if route.CreatedAt != nil {
		createdAtValue := *route.CreatedAt
		createdAtTF = types.StringValue(createdAtValue.Format(time.RFC3339))
	}
	if route.UpdatedAt != nil {
		updatedAtValue := *route.CreatedAt
		updatedAtTF = types.StringValue(updatedAtValue.Format(time.RFC3339))
	}

	// destination
	model.Destination, err = MapRouteDestination(route)
	if err != nil {
		return fmt.Errorf("error mapping route destination: %w", err)
	}

	// next hop
	model.NextHop, err = MapRouteNextHop(route)
	if err != nil {
		return fmt.Errorf("error mapping route next hop: %w", err)
	}

	model.RouteId = types.StringValue(routeId)
	model.CreatedAt = createdAtTF
	model.UpdatedAt = updatedAtTF
	model.Labels = labels
	return nil
}

func MapRouteNextHop(routeResp *iaasalpha.Route) (types.Object, error) {
	if routeResp.Nexthop == nil {
		return types.ObjectNull(RouteNextHopTypes), nil
	}

	nextHopMap := map[string]attr.Value{}
	switch i := routeResp.Nexthop.GetActualInstance().(type) {
	case *iaasalpha.NexthopIPv4:
		nextHopMap["type"] = types.StringPointerValue(i.Type)
		nextHopMap["value"] = types.StringPointerValue(i.Value)
	case *iaasalpha.NexthopIPv6:
		nextHopMap["type"] = types.StringPointerValue(i.Type)
		nextHopMap["value"] = types.StringPointerValue(i.Value)
	case *iaasalpha.NexthopBlackhole:
		nextHopMap["type"] = types.StringPointerValue(i.Type)
		nextHopMap["value"] = types.StringNull()
	case *iaasalpha.NexthopInternet:
		nextHopMap["type"] = types.StringPointerValue(i.Type)
		nextHopMap["value"] = types.StringNull()
	default:
		return types.ObjectNull(RouteNextHopTypes), fmt.Errorf("unexpected Nexthop type: %T", i)
	}

	nextHopTF, diags := types.ObjectValue(RouteNextHopTypes, nextHopMap)
	if diags.HasError() {
		return types.ObjectNull(RouteNextHopTypes), core.DiagsToError(diags)
	}

	return nextHopTF, nil
}

func MapRouteDestination(routeResp *iaasalpha.Route) (types.Object, error) {
	if routeResp.Destination == nil {
		return types.ObjectNull(RouteDestinationTypes), nil
	}

	destinationMap := map[string]attr.Value{}
	switch i := routeResp.Destination.GetActualInstance().(type) {
	case *iaasalpha.DestinationCIDRv4:
		destinationMap["type"] = types.StringPointerValue(i.Type)
		destinationMap["value"] = types.StringPointerValue(i.Value)
	case *iaasalpha.DestinationCIDRv6:
		destinationMap["type"] = types.StringPointerValue(i.Type)
		destinationMap["value"] = types.StringPointerValue(i.Value)
	default:
		return types.ObjectNull(RouteDestinationTypes), fmt.Errorf("unexpected Destionation type: %T", i)
	}

	destinationTF, diags := types.ObjectValue(RouteDestinationTypes, destinationMap)
	if diags.HasError() {
		return types.ObjectNull(RouteDestinationTypes), core.DiagsToError(diags)
	}

	return destinationTF, nil
}
