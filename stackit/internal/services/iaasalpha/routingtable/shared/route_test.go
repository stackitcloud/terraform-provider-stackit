package shared

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
)

const (
	testRegion = "eu02"
)

var (
	testRouteId        = uuid.New()
	testOrganizationId = uuid.New()
	testNetworkAreaId  = uuid.New()
	testRoutingTableId = uuid.New()
)

func Test_MapRouteNextHop(t *testing.T) {
	type args struct {
		routeResp *iaasalpha.Route
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		expected types.Object
	}{
		{
			name: "nexthop is nil",
			args: args{
				routeResp: &iaasalpha.Route{
					Nexthop: nil,
				},
			},
			wantErr:  false,
			expected: types.ObjectNull(RouteNextHopTypes),
		},
		{
			name: "nexthop is empty",
			args: args{
				routeResp: &iaasalpha.Route{
					Nexthop: &iaasalpha.RouteNexthop{},
				},
			},
			wantErr: true,
		},
		{
			name: "nexthop ipv4",
			args: args{
				routeResp: &iaasalpha.Route{
					Nexthop: utils.Ptr(iaasalpha.NexthopIPv4AsRouteNexthop(
						iaasalpha.NewNexthopIPv4("ipv4", "10.20.42.2"),
					)),
				},
			},
			wantErr: false,
			expected: types.ObjectValueMust(RouteNextHopTypes, map[string]attr.Value{
				"type":  types.StringValue("ipv4"),
				"value": types.StringValue("10.20.42.2"),
			}),
		},
		{
			name: "nexthop ipv6",
			args: args{
				routeResp: &iaasalpha.Route{
					Nexthop: utils.Ptr(iaasalpha.NexthopIPv6AsRouteNexthop(
						iaasalpha.NewNexthopIPv6("ipv6", "172b:f881:46fe:d89a:9332:90f7:3485:236d"),
					)),
				},
			},
			wantErr: false,
			expected: types.ObjectValueMust(RouteNextHopTypes, map[string]attr.Value{
				"type":  types.StringValue("ipv6"),
				"value": types.StringValue("172b:f881:46fe:d89a:9332:90f7:3485:236d"),
			}),
		},
		{
			name: "nexthop internet",
			args: args{
				routeResp: &iaasalpha.Route{
					Nexthop: utils.Ptr(iaasalpha.NexthopInternetAsRouteNexthop(
						iaasalpha.NewNexthopInternet("internet"),
					)),
				},
			},
			wantErr: false,
			expected: types.ObjectValueMust(RouteNextHopTypes, map[string]attr.Value{
				"type":  types.StringValue("internet"),
				"value": types.StringNull(),
			}),
		},
		{
			name: "nexthop blackhole",
			args: args{
				routeResp: &iaasalpha.Route{
					Nexthop: utils.Ptr(iaasalpha.NexthopBlackholeAsRouteNexthop(
						iaasalpha.NewNexthopBlackhole("blackhole"),
					)),
				},
			},
			wantErr: false,
			expected: types.ObjectValueMust(RouteNextHopTypes, map[string]attr.Value{
				"type":  types.StringValue("blackhole"),
				"value": types.StringNull(),
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := MapRouteNextHop(tt.args.routeResp)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapNextHop() error = %v, wantErr %v", err, tt.wantErr)
			}

			diff := cmp.Diff(actual, tt.expected)
			if !tt.wantErr && diff != "" {
				t.Errorf("mapNextHop() result does not match: %s", diff)
			}
		})
	}
}

func Test_MapRouteDestination(t *testing.T) {
	type args struct {
		routeResp *iaasalpha.Route
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		expected types.Object
	}{

		{
			name: "destination is nil",
			args: args{
				routeResp: &iaasalpha.Route{
					Destination: nil,
				},
			},
			wantErr:  false,
			expected: types.ObjectNull(RouteDestinationTypes),
		},
		{
			name: "destination is empty",
			args: args{
				routeResp: &iaasalpha.Route{
					Destination: &iaasalpha.RouteDestination{},
				},
			},
			wantErr: true,
		},
		{
			name: "destination cidrv4",
			args: args{
				routeResp: &iaasalpha.Route{
					Destination: utils.Ptr(iaasalpha.DestinationCIDRv4AsRouteDestination(
						iaasalpha.NewDestinationCIDRv4("cidrv4", "58.251.236.138/32"),
					)),
				},
			},
			wantErr: false,
			expected: types.ObjectValueMust(RouteDestinationTypes, map[string]attr.Value{
				"type":  types.StringValue("cidrv4"),
				"value": types.StringValue("58.251.236.138/32"),
			}),
		},
		{
			name: "destination cidrv6",
			args: args{
				routeResp: &iaasalpha.Route{
					Destination: utils.Ptr(iaasalpha.DestinationCIDRv6AsRouteDestination(
						iaasalpha.NewDestinationCIDRv6("cidrv6", "2001:0db8:3c4d:1a2b::/64"),
					)),
				},
			},
			wantErr: false,
			expected: types.ObjectValueMust(RouteDestinationTypes, map[string]attr.Value{
				"type":  types.StringValue("cidrv6"),
				"value": types.StringValue("2001:0db8:3c4d:1a2b::/64"),
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := MapRouteDestination(tt.args.routeResp)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapDestination() error = %v, wantErr %v", err, tt.wantErr)
			}

			diff := cmp.Diff(actual, tt.expected)
			if !tt.wantErr && diff != "" {
				t.Errorf("mapDestination() result does not match: %s", diff)
			}
		})
	}
}

func TestMapRouteModel(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now().Add(5 * time.Minute)

	type args struct {
		route  *iaasalpha.Route
		model  *RouteModel
		region string
	}
	tests := []struct {
		name          string
		args          args
		wantErr       bool
		expectedModel *RouteModel
	}{
		{
			name: "route is nil",
			args: args{
				model:  &RouteModel{},
				route:  nil,
				region: testRegion,
			},
			wantErr: true,
		},
		{
			name: "model is nil",
			args: args{
				model:  nil,
				route:  &iaasalpha.Route{},
				region: testRegion,
			},
			wantErr: true,
		},
		{
			name: "max",
			args: args{
				model: &RouteModel{
					// state
					OrganizationId: types.StringValue(testOrganizationId.String()),
					NetworkAreaId:  types.StringValue(testNetworkAreaId.String()),
					RoutingTableId: types.StringValue(testRoutingTableId.String()),
				},
				route: &iaasalpha.Route{
					Id: utils.Ptr(testRouteId.String()),
					Destination: utils.Ptr(iaasalpha.DestinationCIDRv4AsRouteDestination(
						iaasalpha.NewDestinationCIDRv4("cidrv4", "58.251.236.138/32"),
					)),
					Labels: &map[string]interface{}{
						"foo1": "bar1",
						"foo2": "bar2",
					},
					Nexthop: utils.Ptr(
						iaasalpha.NexthopIPv4AsRouteNexthop(iaasalpha.NewNexthopIPv4("ipv4", "10.20.42.2")),
					),
					CreatedAt: &createdAt,
					UpdatedAt: &updatedAt,
				},
				region: testRegion,
			},
			wantErr: false,
			expectedModel: &RouteModel{
				Id: types.StringValue(fmt.Sprintf("%s,%s,%s,%s,%s",
					testOrganizationId.String(), testRegion, testNetworkAreaId.String(), testRoutingTableId.String(), testRouteId.String()),
				),
				OrganizationId: types.StringValue(testOrganizationId.String()),
				NetworkAreaId:  types.StringValue(testNetworkAreaId.String()),
				RoutingTableId: types.StringValue(testRoutingTableId.String()),
				RouteReadModel: RouteReadModel{
					RouteId: types.StringValue(testRouteId.String()),
					Destination: types.ObjectValueMust(RouteDestinationTypes, map[string]attr.Value{
						"type":  types.StringValue("cidrv4"),
						"value": types.StringValue("58.251.236.138/32"),
					}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
						"foo1": types.StringValue("bar1"),
						"foo2": types.StringValue("bar2"),
					}),
					NextHop: types.ObjectValueMust(RouteNextHopTypes, map[string]attr.Value{
						"type":  types.StringValue("ipv4"),
						"value": types.StringValue("10.20.42.2"),
					}),
					CreatedAt: types.StringValue(createdAt.Format(time.RFC3339)),
					UpdatedAt: types.StringValue(updatedAt.Format(time.RFC3339)),
				},
				Region: types.StringValue(testRegion),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := MapRouteModel(ctx, tt.args.route, tt.args.model, tt.args.region); (err != nil) != tt.wantErr {
				t.Errorf("MapRouteModel() error = %v, wantErr %v", err, tt.wantErr)
			}

			diff := cmp.Diff(tt.args.model, tt.expectedModel)
			if !tt.wantErr && diff != "" {
				t.Errorf("MapRouteModel() model does not match: %s", diff)
			}
		})
	}
}
