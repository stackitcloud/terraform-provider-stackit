package routingtables

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	organizationId       = uuid.New()
	networkAreaId        = uuid.New()
	routingTableId       = uuid.New()
	secondRoutingTableId = uuid.New()
	testRouteId1         = uuid.NewString()
	testRouteId2         = uuid.NewString()
)

func TestMapDataFields(t *testing.T) {
	terraformId := fmt.Sprintf("%s,%s,%s", organizationId.String(), testRegion, networkAreaId.String())

	tests := []struct {
		description string
		state       DataSourceModelTables
		input       *iaasalpha.RoutingTableListResponse
		expected    DataSourceModelTables
		isValid     bool
	}{
		{
			"default_values",
			DataSourceModelTables{
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
			},
			&iaasalpha.RoutingTableListResponse{
				Items: &[]iaasalpha.RoutingTable{
					{
						Id:           utils.Ptr(routingTableId.String()),
						Name:         utils.Ptr("test"),
						Description:  utils.Ptr("description"),
						Main:         utils.Ptr(true),
						CreatedAt:    &time.Time{},
						UpdatedAt:    &time.Time{},
						SystemRoutes: utils.Ptr(false),
					},
				},
			},
			DataSourceModelTables{
				Id:             types.StringValue(terraformId),
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
				Items: types.ListValueMust(types.ObjectType{AttrTypes: shared.RoutingTableReadModelTypes()}, []attr.Value{
					types.ObjectValueMust(shared.RoutingTableReadModelTypes(), map[string]attr.Value{
						"routing_table_id":   types.StringValue(routingTableId.String()),
						"name":               types.StringValue("test"),
						"description":        types.StringValue("description"),
						"main_routing_table": types.BoolValue(true),
						"system_routes":      types.BoolValue(false),
						"created_at":         types.StringNull(),
						"updated_at":         types.StringNull(),
						"labels":             types.MapNull(types.StringType),
						"routes": types.ListValueMust(
							types.ObjectType{AttrTypes: shared.RouteReadModelTypes()}, []attr.Value{},
						),
					}),
				}),
			},
			true,
		},
		{
			"two routing tables",
			DataSourceModelTables{
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
			},
			&iaasalpha.RoutingTableListResponse{
				Items: &[]iaasalpha.RoutingTable{
					{
						Id:           utils.Ptr(routingTableId.String()),
						Name:         utils.Ptr("test"),
						Description:  utils.Ptr("description"),
						Main:         utils.Ptr(true),
						CreatedAt:    &time.Time{},
						UpdatedAt:    &time.Time{},
						SystemRoutes: utils.Ptr(false),
						Routes: &[]iaasalpha.Route{
							{
								Id: utils.Ptr(testRouteId1),
								Destination: utils.Ptr(iaasalpha.DestinationCIDRv4AsRouteDestination(
									iaasalpha.NewDestinationCIDRv4("cidrv4", "58.251.236.138/32"),
								)),
								Nexthop: utils.Ptr(iaasalpha.NexthopIPv4AsRouteNexthop(
									iaasalpha.NewNexthopIPv4("ipv4", "10.20.42.2"),
								)),
								Labels: &map[string]interface{}{
									"foo": "bar",
								},
								CreatedAt: nil,
								UpdatedAt: nil,
							},
							{
								Id: utils.Ptr(testRouteId2),
								Destination: utils.Ptr(iaasalpha.DestinationCIDRv6AsRouteDestination(
									iaasalpha.NewDestinationCIDRv6("cidrv6", "2001:0db8:3c4d:1a2b::/64"),
								)),
								Nexthop: utils.Ptr(iaasalpha.NexthopIPv6AsRouteNexthop(
									iaasalpha.NewNexthopIPv6("ipv6", "172b:f881:46fe:d89a:9332:90f7:3485:236d"),
								)),
								Labels: &map[string]interface{}{
									"key": "value",
								},
								CreatedAt: nil,
								UpdatedAt: nil,
							},
						},
					},
					{
						Id:           utils.Ptr(secondRoutingTableId.String()),
						Name:         utils.Ptr("test2"),
						Description:  utils.Ptr("description2"),
						Main:         utils.Ptr(false),
						CreatedAt:    &time.Time{},
						UpdatedAt:    &time.Time{},
						SystemRoutes: utils.Ptr(false),
					},
				},
			},
			DataSourceModelTables{
				Id:             types.StringValue(terraformId),
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
				Items: types.ListValueMust(types.ObjectType{AttrTypes: shared.RoutingTableReadModelTypes()}, []attr.Value{
					types.ObjectValueMust(shared.RoutingTableReadModelTypes(), map[string]attr.Value{
						"routing_table_id":   types.StringValue(routingTableId.String()),
						"name":               types.StringValue("test"),
						"description":        types.StringValue("description"),
						"main_routing_table": types.BoolValue(true),
						"system_routes":      types.BoolValue(false),
						"created_at":         types.StringNull(),
						"updated_at":         types.StringNull(),
						"labels":             types.MapNull(types.StringType),
						"routes": types.ListValueMust(
							types.ObjectType{AttrTypes: shared.RouteReadModelTypes()}, []attr.Value{
								types.ObjectValueMust(shared.RouteReadModelTypes(), map[string]attr.Value{
									"route_id":   types.StringValue(testRouteId1),
									"created_at": types.StringNull(),
									"updated_at": types.StringNull(),
									"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
										"foo": types.StringValue("bar"),
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
									"route_id":   types.StringValue(testRouteId2),
									"created_at": types.StringNull(),
									"updated_at": types.StringNull(),
									"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
										"key": types.StringValue("value"),
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
					}),
					types.ObjectValueMust(shared.RoutingTableReadModelTypes(), map[string]attr.Value{
						"routing_table_id":   types.StringValue(secondRoutingTableId.String()),
						"name":               types.StringValue("test2"),
						"description":        types.StringValue("description2"),
						"main_routing_table": types.BoolValue(false),
						"system_routes":      types.BoolValue(false),
						"created_at":         types.StringNull(),
						"updated_at":         types.StringNull(),
						"labels":             types.MapNull(types.StringType),
						"routes": types.ListValueMust(
							types.ObjectType{AttrTypes: shared.RouteReadModelTypes()}, []attr.Value{},
						),
					}),
				}),
			},
			true,
		},
		{
			"response_fields_items_nil_fail",
			DataSourceModelTables{},
			&iaasalpha.RoutingTableListResponse{
				Items: nil,
			},
			DataSourceModelTables{},
			false,
		},
		{
			"response_nil_fail",
			DataSourceModelTables{},
			nil,
			DataSourceModelTables{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceRoutingTables(context.Background(), tt.input, &tt.state, testRegion)
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
