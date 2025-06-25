package routingtable

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/shared"
)

const (
	testRegion = "eu01"
)

var (
	organizationId = uuid.New()
	networkAreaId  = uuid.New()
	routingTableId = uuid.New()
	route1Id       = uuid.New()
	route2Id       = uuid.New()
)

func TestMapDataFields(t *testing.T) {
	id := fmt.Sprintf("%s,%s,%s,%s", organizationId.String(), testRegion, networkAreaId.String(), routingTableId.String())

	tests := []struct {
		description string
		state       shared.RoutingTableDataSourceModel
		input       *iaasalpha.RoutingTable
		expected    shared.RoutingTableDataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			shared.RoutingTableDataSourceModel{
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				RoutingTableReadModel: shared.RoutingTableReadModel{
					Routes: types.ListNull(types.StringType),
				},
			},
			&iaasalpha.RoutingTable{
				Id:   utils.Ptr(routingTableId.String()),
				Name: utils.Ptr("default_values"),
			},
			shared.RoutingTableDataSourceModel{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
				RoutingTableReadModel: shared.RoutingTableReadModel{
					RoutingTableId: types.StringValue(routingTableId.String()),
					Name:           types.StringValue("default_values"),
					Labels:         types.MapNull(types.StringType),
					Routes: types.ListValueMust(
						types.ObjectType{AttrTypes: shared.RouteReadModelTypes()}, []attr.Value{},
					),
				},
			},
			true,
		},
		{
			"values_ok",
			shared.RoutingTableDataSourceModel{
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				RoutingTableReadModel: shared.RoutingTableReadModel{
					Routes: types.ListValueMust(
						types.ObjectType{AttrTypes: shared.RouteReadModelTypes()}, []attr.Value{},
					),
				},
			},
			&iaasalpha.RoutingTable{
				Id:          utils.Ptr(routingTableId.String()),
				Name:        utils.Ptr("values_ok"),
				Description: utils.Ptr("Description"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Routes: &[]iaasalpha.Route{
					{
						Id: utils.Ptr(route1Id.String()),
						Labels: &map[string]interface{}{
							"route1-key": "route1-value",
						},
						Destination: utils.Ptr(iaasalpha.DestinationCIDRv4AsRouteDestination(
							iaasalpha.NewDestinationCIDRv4("cidrv4", "58.251.236.138/32"),
						)),
						Nexthop: utils.Ptr(
							iaasalpha.NexthopIPv4AsRouteNexthop(iaasalpha.NewNexthopIPv4("ipv4", "10.20.42.2")),
						),
						CreatedAt: nil,
						UpdatedAt: nil,
					},
					{
						Id: utils.Ptr(route2Id.String()),
						Labels: &map[string]interface{}{
							"route2-key": "route2-value",
						},
						Destination: utils.Ptr(iaasalpha.DestinationCIDRv6AsRouteDestination(
							iaasalpha.NewDestinationCIDRv6("cidrv6", "2001:0db8:3c4d:1a2b::/64"),
						)),
						Nexthop: utils.Ptr(iaasalpha.NexthopIPv6AsRouteNexthop(
							iaasalpha.NewNexthopIPv6("ipv6", "172b:f881:46fe:d89a:9332:90f7:3485:236d"),
						)),
						CreatedAt: nil,
						UpdatedAt: nil,
					},
				},
			},
			shared.RoutingTableDataSourceModel{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
				RoutingTableReadModel: shared.RoutingTableReadModel{
					RoutingTableId: types.StringValue(routingTableId.String()),
					Name:           types.StringValue("values_ok"),
					Description:    types.StringValue("Description"),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
						"key": types.StringValue("value"),
					}),
					Routes: types.ListValueMust(
						types.ObjectType{AttrTypes: shared.RouteReadModelTypes()}, []attr.Value{
							types.ObjectValueMust(shared.RouteReadModelTypes(), map[string]attr.Value{
								"route_id":   types.StringValue(route1Id.String()),
								"created_at": types.StringNull(),
								"updated_at": types.StringNull(),
								"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
									"route1-key": types.StringValue("route1-value"),
								}),
								"destination": types.ObjectValueMust(shared.RouteDestinationTypes, map[string]attr.Value{
									"type":  types.StringValue("cidrv4"),
									"value": types.StringValue("58.251.236.138/32"),
								}),
								"next_hop": types.ObjectValueMust(shared.RouteNextHopTypes, map[string]attr.Value{
									"type":  types.StringValue("ipv4"),
									"value": types.StringValue("10.20.42.2"),
								}),
							}),
							types.ObjectValueMust(shared.RouteReadModelTypes(), map[string]attr.Value{
								"route_id":   types.StringValue(route2Id.String()),
								"created_at": types.StringNull(),
								"updated_at": types.StringNull(),
								"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
									"route2-key": types.StringValue("route2-value"),
								}),
								"destination": types.ObjectValueMust(shared.RouteDestinationTypes, map[string]attr.Value{
									"type":  types.StringValue("cidrv6"),
									"value": types.StringValue("2001:0db8:3c4d:1a2b::/64"),
								}),
								"next_hop": types.ObjectValueMust(shared.RouteNextHopTypes, map[string]attr.Value{
									"type":  types.StringValue("ipv6"),
									"value": types.StringValue("172b:f881:46fe:d89a:9332:90f7:3485:236d"),
								}),
							}),
						},
					),
				},
			},
			true,
		},
		{
			"response_fields_nil_fail",
			shared.RoutingTableDataSourceModel{},
			&iaasalpha.RoutingTable{
				Id: nil,
			},
			shared.RoutingTableDataSourceModel{},
			false,
		},
		{
			"response_nil_fail",
			shared.RoutingTableDataSourceModel{},
			nil,
			shared.RoutingTableDataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			shared.RoutingTableDataSourceModel{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
			},
			&iaasalpha.RoutingTable{},
			shared.RoutingTableDataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := shared.MapDataSourceFields(context.Background(), tt.input, &tt.state, testRegion)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
