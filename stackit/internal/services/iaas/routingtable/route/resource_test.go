package route

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/routingtable/shared"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

const (
	testRegion = "eu02"
)

var (
	organizationId = uuid.New()
	networkAreaId  = uuid.New()
	routingTableId = uuid.New()
	routeId        = uuid.New()
)

func Test_mapFieldsFromList(t *testing.T) {
	type args struct {
		routeResp *iaas.RouteListResponse
		model     *shared.RouteModel
		region    string
	}
	tests := []struct {
		name          string
		args          args
		wantErr       bool
		expectedModel *shared.RouteModel
	}{
		{
			name: "response is nil",
			args: args{
				model:     &shared.RouteModel{},
				routeResp: nil,
			},
			wantErr: true,
		},
		{
			name: "response items is nil",
			args: args{
				model: &shared.RouteModel{},
				routeResp: &iaas.RouteListResponse{
					Items: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "model is nil",
			args: args{
				model: nil,
				routeResp: &iaas.RouteListResponse{
					Items: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "response items is empty",
			args: args{
				model: &shared.RouteModel{},
				routeResp: &iaas.RouteListResponse{
					Items: &[]iaas.Route{},
				},
			},
			wantErr: true,
		},
		{
			name: "response items contains more than one route",
			args: args{
				model: &shared.RouteModel{},
				routeResp: &iaas.RouteListResponse{
					Items: &[]iaas.Route{
						{
							Id: utils.Ptr(uuid.NewString()),
						},
						{
							Id: utils.Ptr(uuid.NewString()),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "success",
			args: args{
				model: &shared.RouteModel{
					RouteReadModel: shared.RouteReadModel{
						RouteId: types.StringNull(),
					},
					RoutingTableId: types.StringValue(routingTableId.String()),
					OrganizationId: types.StringValue(organizationId.String()),
					NetworkAreaId:  types.StringValue(networkAreaId.String()),
				},
				routeResp: &iaas.RouteListResponse{
					Items: &[]iaas.Route{
						{
							Id: utils.Ptr(routeId.String()),
							Destination: utils.Ptr(iaas.DestinationCIDRv4AsRouteDestination(
								iaas.NewDestinationCIDRv4("cidrv4", "58.251.236.138/32"),
							)),
							Nexthop: utils.Ptr(iaas.NexthopIPv4AsRouteNexthop(
								iaas.NewNexthopIPv4("ipv4", "10.20.42.2"),
							)),
							Labels: &map[string]interface{}{
								"foo": "bar",
							},
							CreatedAt: nil,
							UpdatedAt: nil,
						},
					},
				},
				region: testRegion,
			},
			wantErr: false,
			expectedModel: &shared.RouteModel{
				RouteReadModel: shared.RouteReadModel{
					RouteId: types.StringValue(routeId.String()),
					NextHop: types.ObjectValueMust(shared.RouteNextHopTypes, map[string]attr.Value{
						"type":  types.StringValue("ipv4"),
						"value": types.StringValue("10.20.42.2"),
					}),
					Destination: types.ObjectValueMust(shared.RouteDestinationTypes, map[string]attr.Value{
						"type":  types.StringValue("cidrv4"),
						"value": types.StringValue("58.251.236.138/32"),
					}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
						"foo": types.StringValue("bar"),
					}),
					CreatedAt: types.StringNull(),
					UpdatedAt: types.StringNull(),
				},
				Id:             types.StringValue(fmt.Sprintf("%s,%s,%s,%s,%s", organizationId.String(), testRegion, networkAreaId.String(), routingTableId.String(), routeId.String())),
				RoutingTableId: types.StringValue(routingTableId.String()),
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := mapFieldsFromList(ctx, tt.args.routeResp, tt.args.model, tt.args.region); (err != nil) != tt.wantErr {
				t.Errorf("mapFieldsFromList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			diff := cmp.Diff(tt.args.model, tt.expectedModel)
			if diff != "" && !tt.wantErr {
				t.Fatalf("mapFieldsFromList(): %s", diff)
			}
		})
	}
}

func Test_toUpdatePayload(t *testing.T) {
	type args struct {
		model         *shared.RouteModel
		currentLabels types.Map
	}
	tests := []struct {
		name    string
		args    args
		want    *iaas.UpdateRouteOfRoutingTablePayload
		wantErr bool
	}{
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			wantErr: true,
		},
		{
			name: "max",
			args: args{
				model: &shared.RouteModel{
					RouteReadModel: shared.RouteReadModel{
						Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
							"foo1": types.StringValue("bar1"),
							"foo2": types.StringValue("bar2"),
						}),
					},
				},
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"foo1": types.StringValue("foobar"),
					"foo3": types.StringValue("bar3"),
				}),
			},
			want: &iaas.UpdateRouteOfRoutingTablePayload{
				Labels: &map[string]interface{}{
					"foo1": "bar1",
					"foo2": "bar2",
					"foo3": nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := toUpdatePayload(ctx, tt.args.model, tt.args.currentLabels)
			if (err != nil) != tt.wantErr {
				t.Errorf("toUpdatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			diff := cmp.Diff(got, tt.want)
			if diff != "" {
				t.Fatalf("toUpdatePayload(): %s", diff)
			}
		})
	}
}

func Test_toNextHopPayload(t *testing.T) {
	type args struct {
		model *shared.RouteReadModel
	}
	tests := []struct {
		name    string
		args    args
		want    *iaas.RouteNexthop
		wantErr bool
	}{
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			wantErr: true,
		},
		{
			name: "ipv4",
			args: args{
				model: &shared.RouteReadModel{
					NextHop: types.ObjectValueMust(shared.RouteNextHopTypes, map[string]attr.Value{
						"type":  types.StringValue("ipv4"),
						"value": types.StringValue("10.20.42.2"),
					}),
				},
			},
			wantErr: false,
			want: utils.Ptr(iaas.NexthopIPv4AsRouteNexthop(
				iaas.NewNexthopIPv4("ipv4", "10.20.42.2"),
			)),
		},
		{
			name: "ipv6",
			args: args{
				model: &shared.RouteReadModel{
					NextHop: types.ObjectValueMust(shared.RouteNextHopTypes, map[string]attr.Value{
						"type":  types.StringValue("ipv6"),
						"value": types.StringValue("172b:f881:46fe:d89a:9332:90f7:3485:236d"),
					}),
				},
			},
			wantErr: false,
			want: utils.Ptr(iaas.NexthopIPv6AsRouteNexthop(
				iaas.NewNexthopIPv6("ipv6", "172b:f881:46fe:d89a:9332:90f7:3485:236d"),
			)),
		},
		{
			name: "internet",
			args: args{
				model: &shared.RouteReadModel{
					NextHop: types.ObjectValueMust(shared.RouteNextHopTypes, map[string]attr.Value{
						"type":  types.StringValue("internet"),
						"value": types.StringNull(),
					}),
				},
			},
			wantErr: false,
			want: utils.Ptr(iaas.NexthopInternetAsRouteNexthop(
				iaas.NewNexthopInternet("internet"),
			)),
		},
		{
			name: "blackhole",
			args: args{
				model: &shared.RouteReadModel{
					NextHop: types.ObjectValueMust(shared.RouteNextHopTypes, map[string]attr.Value{
						"type":  types.StringValue("blackhole"),
						"value": types.StringNull(),
					}),
				},
			},
			wantErr: false,
			want: utils.Ptr(iaas.NexthopBlackholeAsRouteNexthop(
				iaas.NewNexthopBlackhole("blackhole"),
			)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := toNextHopPayload(ctx, tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toNextHopPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toNextHopPayload() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toDestinationPayload(t *testing.T) {
	type args struct {
		model *shared.RouteReadModel
	}
	tests := []struct {
		name    string
		args    args
		want    *iaas.RouteDestination
		wantErr bool
	}{
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			wantErr: true,
		},
		{
			name: "cidrv4",
			args: args{
				model: &shared.RouteReadModel{
					Destination: types.ObjectValueMust(shared.RouteDestinationTypes, map[string]attr.Value{
						"type":  types.StringValue("cidrv4"),
						"value": types.StringValue("58.251.236.138/32"),
					}),
				},
			},
			wantErr: false,
			want: utils.Ptr(iaas.DestinationCIDRv4AsRouteDestination(
				iaas.NewDestinationCIDRv4("cidrv4", "58.251.236.138/32"),
			)),
		},
		{
			name: "cidrv6",
			args: args{
				model: &shared.RouteReadModel{
					Destination: types.ObjectValueMust(shared.RouteDestinationTypes, map[string]attr.Value{
						"type":  types.StringValue("cidrv6"),
						"value": types.StringValue("2001:0db8:3c4d:1a2b::/64"),
					}),
				},
			},
			wantErr: false,
			want: utils.Ptr(iaas.DestinationCIDRv6AsRouteDestination(
				iaas.NewDestinationCIDRv6("cidrv6", "2001:0db8:3c4d:1a2b::/64"),
			)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := toDestinationPayload(ctx, tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toDestinationPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toDestinationPayload() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toCreatePayload(t *testing.T) {
	type args struct {
		model *shared.RouteReadModel
	}
	tests := []struct {
		name    string
		args    args
		want    *iaas.AddRoutesToRoutingTablePayload
		wantErr bool
	}{
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			wantErr: true,
		},
		{
			name: "max",
			args: args{
				model: &shared.RouteReadModel{
					NextHop: types.ObjectValueMust(shared.RouteNextHopTypes, map[string]attr.Value{
						"type":  types.StringValue("ipv4"),
						"value": types.StringValue("10.20.42.2"),
					}),
					Destination: types.ObjectValueMust(shared.RouteDestinationTypes, map[string]attr.Value{
						"type":  types.StringValue("cidrv4"),
						"value": types.StringValue("58.251.236.138/32"),
					}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
						"foo1": types.StringValue("bar1"),
						"foo2": types.StringValue("bar2"),
					}),
				},
			},
			want: &iaas.AddRoutesToRoutingTablePayload{
				Items: &[]iaas.Route{
					{
						Labels: &map[string]interface{}{
							"foo1": "bar1",
							"foo2": "bar2",
						},
						Nexthop: utils.Ptr(iaas.NexthopIPv4AsRouteNexthop(
							iaas.NewNexthopIPv4("ipv4", "10.20.42.2"),
						)),
						Destination: utils.Ptr(iaas.DestinationCIDRv4AsRouteDestination(
							iaas.NewDestinationCIDRv4("cidrv4", "58.251.236.138/32"),
						)),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := toCreatePayload(ctx, tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toCreatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			diff := cmp.Diff(got, tt.want)
			if diff != "" {
				t.Fatalf("toCreatePayload(): %s", diff)
			}
		})
	}
}
