package routingtable_routes

import (
	"context"
	"fmt"
	"testing"

	"dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/iaasalpha"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/shared"
)

const (
	testRegion = "eu02"
)

var (
	testOrganizationId = uuid.NewString()
	testNetworkAreaId  = uuid.NewString()
	testRoutingTableId = uuid.NewString()
	testRouteId1       = uuid.NewString()
	testRouteId2       = uuid.NewString()
)

func Test_mapDataSourceRoutingTableRoutes(t *testing.T) {
	type args struct {
		routes *iaasalpha.RouteListResponse
		model  *RoutingTableRoutesDataSourceModel
		region string
	}
	tests := []struct {
		name          string
		args          args
		wantErr       bool
		expectedModel *RoutingTableRoutesDataSourceModel
	}{
		{
			name: "model is nil",
			args: args{
				model: nil,
				routes: &iaasalpha.RouteListResponse{
					Items: &[]iaasalpha.Route{},
				},
			},
			wantErr: true,
		},
		{
			name: "response is nil",
			args: args{
				model:  &RoutingTableRoutesDataSourceModel{},
				routes: nil,
			},
			wantErr: true,
		},
		{
			name: "response items is nil",
			args: args{
				model: nil,
				routes: &iaasalpha.RouteListResponse{
					Items: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "response items is empty",
			args: args{
				model: &RoutingTableRoutesDataSourceModel{
					OrganizationId: types.StringValue(testOrganizationId),
					NetworkAreaId:  types.StringValue(testNetworkAreaId),
					RoutingTableId: types.StringValue(testRoutingTableId),
					Region:         types.StringValue(testRegion),
				},
				routes: &iaasalpha.RouteListResponse{
					Items: &[]iaasalpha.Route{},
				},
				region: testRegion,
			},
			wantErr: false,
			expectedModel: &RoutingTableRoutesDataSourceModel{
				Id:             types.StringValue(fmt.Sprintf("%s,%s,%s,%s", testOrganizationId, testRegion, testNetworkAreaId, testRoutingTableId)),
				OrganizationId: types.StringValue(testOrganizationId),
				NetworkAreaId:  types.StringValue(testNetworkAreaId),
				RoutingTableId: types.StringValue(testRoutingTableId),
				Region:         types.StringValue(testRegion),
				Routes: types.ListValueMust(
					types.ObjectType{AttrTypes: shared.RouteReadModelTypes()}, []attr.Value{},
				),
			},
		},
		{
			name: "response items has items",
			args: args{
				model: &RoutingTableRoutesDataSourceModel{
					OrganizationId: types.StringValue(testOrganizationId),
					NetworkAreaId:  types.StringValue(testNetworkAreaId),
					RoutingTableId: types.StringValue(testRoutingTableId),
					Region:         types.StringValue(testRegion),
				},
				routes: &iaasalpha.RouteListResponse{
					Items: &[]iaasalpha.Route{
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
				region: testRegion,
			},
			wantErr: false,
			expectedModel: &RoutingTableRoutesDataSourceModel{
				Id:             types.StringValue(fmt.Sprintf("%s,%s,%s,%s", testOrganizationId, testRegion, testNetworkAreaId, testRoutingTableId)),
				OrganizationId: types.StringValue(testOrganizationId),
				NetworkAreaId:  types.StringValue(testNetworkAreaId),
				RoutingTableId: types.StringValue(testRoutingTableId),
				Region:         types.StringValue(testRegion),
				Routes: types.ListValueMust(
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := mapDataSourceRoutingTableRoutes(ctx, tt.args.routes, tt.args.model, tt.args.region); (err != nil) != tt.wantErr {
				t.Errorf("mapDataSourceRoutingTableRoutes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			diff := cmp.Diff(tt.args.model, tt.expectedModel)
			if diff != "" && !tt.wantErr {
				t.Fatalf("mapFieldsFromList(): %s", diff)
			}
		})
	}
}
