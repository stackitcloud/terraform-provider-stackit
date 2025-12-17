package networkarearoute

import (
	"context"
	"reflect"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapFields(t *testing.T) {
	type args struct {
		state  ModelV1
		input  *iaas.Route
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    ModelV1
		isValid     bool
	}{
		{
			description: "id_ok",
			args: args{
				state: ModelV1{
					OrganizationId:     types.StringValue("oid"),
					NetworkAreaId:      types.StringValue("naid"),
					NetworkAreaRouteId: types.StringValue("narid"),
				},
				input:  &iaas.Route{},
				region: "eu01",
			},
			expected: ModelV1{
				Id:                 types.StringValue("oid,naid,eu01,narid"),
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				NetworkAreaRouteId: types.StringValue("narid"),
				Destination: &DestinationModelV1{
					Type:  types.StringNull(),
					Value: types.StringNull(),
				},
				NextHop: &NexthopModelV1{
					Type:  types.StringNull(),
					Value: types.StringNull(),
				},
				Labels: types.MapNull(types.StringType),
				Region: types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "values_ok",
			args: args{
				state: ModelV1{
					OrganizationId:     types.StringValue("oid"),
					NetworkAreaId:      types.StringValue("naid"),
					NetworkAreaRouteId: types.StringValue("narid"),
					Region:             types.StringValue("eu01"),
				},
				input: &iaas.Route{
					Destination: &iaas.RouteDestination{
						DestinationCIDRv4: &iaas.DestinationCIDRv4{
							Type:  utils.Ptr("cidrv4"),
							Value: utils.Ptr("prefix"),
						},
						DestinationCIDRv6: nil,
					},
					Nexthop: &iaas.RouteNexthop{
						NexthopIPv4: &iaas.NexthopIPv4{
							Type:  utils.Ptr("ipv4"),
							Value: utils.Ptr("hop"),
						},
					},
					Labels: &map[string]interface{}{
						"key": "value",
					},
				},
				region: "eu02",
			},
			expected: ModelV1{
				Id:                 types.StringValue("oid,naid,eu02,narid"),
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				NetworkAreaRouteId: types.StringValue("narid"),
				Destination: &DestinationModelV1{
					Type:  types.StringValue("cidrv4"),
					Value: types.StringValue("prefix"),
				},
				NextHop: &NexthopModelV1{
					Type:  types.StringValue("ipv4"),
					Value: types.StringValue("hop"),
				},
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Region: types.StringValue("eu02"),
			},
			isValid: true,
		},
		{
			description: "response_fields_nil_fail",
			args: args{
				input: &iaas.Route{
					Destination: nil,
					Nexthop:     nil,
				},
			},
		},
		{
			description: "response_nil_fail",
		},
		{
			description: "no_resource_id",
			args: args{
				state: ModelV1{
					OrganizationId: types.StringValue("oid"),
					NetworkAreaId:  types.StringValue("naid"),
				},
				input: &iaas.Route{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.args.input, &tt.args.state, tt.args.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.args.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *ModelV1
		expected    *iaas.CreateNetworkAreaRoutePayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &ModelV1{
				Destination: &DestinationModelV1{
					Type:  types.StringValue("cidrv4"),
					Value: types.StringValue("prefix"),
				},
				NextHop: &NexthopModelV1{
					Type:  types.StringValue("ipv4"),
					Value: types.StringValue("hop"),
				},
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			expected: &iaas.CreateNetworkAreaRoutePayload{
				Items: &[]iaas.Route{
					{
						Destination: &iaas.RouteDestination{
							DestinationCIDRv4: &iaas.DestinationCIDRv4{
								Type:  utils.Ptr("cidrv4"),
								Value: utils.Ptr("prefix"),
							},
							DestinationCIDRv6: nil,
						},
						Nexthop: &iaas.RouteNexthop{
							NexthopIPv4: &iaas.NexthopIPv4{
								Type:  utils.Ptr("ipv4"),
								Value: utils.Ptr("hop"),
							},
						},
						Labels: &map[string]interface{}{
							"key": "value",
						},
					},
				},
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *ModelV1
		expected    *iaas.UpdateNetworkAreaRoutePayload
		isValid     bool
	}{
		{
			"default_ok",
			&ModelV1{
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				}),
			},
			&iaas.UpdateNetworkAreaRoutePayload{
				Labels: &map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, types.MapNull(types.StringType))
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToNextHopPayload(t *testing.T) {
	type args struct {
		model *ModelV1
	}
	tests := []struct {
		name    string
		args    args
		want    *iaas.RouteNexthop
		wantErr bool
	}{
		{
			name: "ipv4",
			args: args{
				model: &ModelV1{
					NextHop: &NexthopModelV1{
						Type:  types.StringValue("ipv4"),
						Value: types.StringValue("10.20.30.40"),
					},
				},
			},
			want: &iaas.RouteNexthop{
				NexthopIPv4: &iaas.NexthopIPv4{
					Type:  utils.Ptr("ipv4"),
					Value: utils.Ptr("10.20.30.40"),
				},
			},
			wantErr: false,
		},
		{
			name: "ipv6",
			args: args{
				model: &ModelV1{
					NextHop: &NexthopModelV1{
						Type:  types.StringValue("ipv6"),
						Value: types.StringValue("2001:db8:85a3:0:0:8a2e:370:7334"),
					},
				},
			},
			want: &iaas.RouteNexthop{
				NexthopIPv6: &iaas.NexthopIPv6{
					Type:  utils.Ptr("ipv6"),
					Value: utils.Ptr("2001:db8:85a3:0:0:8a2e:370:7334"),
				},
			},
			wantErr: false,
		},
		{
			name: "internet",
			args: args{
				model: &ModelV1{
					NextHop: &NexthopModelV1{
						Type: types.StringValue("internet"),
					},
				},
			},
			want: &iaas.RouteNexthop{
				NexthopInternet: &iaas.NexthopInternet{
					Type: utils.Ptr("internet"),
				},
			},
			wantErr: false,
		},
		{
			name: "blackhole",
			args: args{
				model: &ModelV1{
					NextHop: &NexthopModelV1{
						Type: types.StringValue("blackhole"),
					},
				},
			},
			want: &iaas.RouteNexthop{
				NexthopBlackhole: &iaas.NexthopBlackhole{
					Type: utils.Ptr("blackhole"),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			args: args{
				model: &ModelV1{
					NextHop: &NexthopModelV1{
						Type: types.StringValue("foobar"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			wantErr: true,
		},
		{
			name: "nexthop in model is nil",
			args: args{
				model: &ModelV1{
					NextHop: nil,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toNextHopPayload(tt.args.model)
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

func TestToDestinationPayload(t *testing.T) {
	type args struct {
		model *ModelV1
	}
	tests := []struct {
		name    string
		args    args
		want    *iaas.RouteDestination
		wantErr bool
	}{
		{
			name: "cidrv4",
			args: args{
				model: &ModelV1{
					Destination: &DestinationModelV1{
						Type:  types.StringValue("cidrv4"),
						Value: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			want: &iaas.RouteDestination{
				DestinationCIDRv4: &iaas.DestinationCIDRv4{
					Type:  utils.Ptr("cidrv4"),
					Value: utils.Ptr("192.168.1.0/24"),
				},
			},
			wantErr: false,
		},
		{
			name: "cidrv6",
			args: args{
				model: &ModelV1{
					Destination: &DestinationModelV1{
						Type:  types.StringValue("cidrv6"),
						Value: types.StringValue("2001:db8:1234::/48"),
					},
				},
			},
			want: &iaas.RouteDestination{
				DestinationCIDRv6: &iaas.DestinationCIDRv6{
					Type:  utils.Ptr("cidrv6"),
					Value: utils.Ptr("2001:db8:1234::/48"),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			args: args{
				model: &ModelV1{
					Destination: &DestinationModelV1{
						Type: types.StringValue("foobar"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			wantErr: true,
		},
		{
			name: "destination in model is nil",
			args: args{
				model: &ModelV1{
					Destination: nil,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toDestinationPayload(tt.args.model)
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

func TestMapRouteNextHop(t *testing.T) {
	type args struct {
		routeResp *iaas.Route
	}
	tests := []struct {
		name    string
		args    args
		want    *NexthopModelV1
		wantErr bool
	}{
		{
			name: "ipv4",
			args: args{
				routeResp: &iaas.Route{
					Nexthop: &iaas.RouteNexthop{
						NexthopIPv4: &iaas.NexthopIPv4{
							Type:  utils.Ptr("ipv4"),
							Value: utils.Ptr("192.168.1.0/24"),
						},
					},
				},
			},
			want: &NexthopModelV1{
				Type:  types.StringValue("ipv4"),
				Value: types.StringValue("192.168.1.0/24"),
			},
		},
		{
			name: "ipv6",
			args: args{
				routeResp: &iaas.Route{
					Nexthop: &iaas.RouteNexthop{
						NexthopIPv4: &iaas.NexthopIPv4{
							Type:  utils.Ptr("ipv6"),
							Value: utils.Ptr("2001:db8:85a3:0:0:8a2e:370:7334"),
						},
					},
				},
			},
			want: &NexthopModelV1{
				Type:  types.StringValue("ipv6"),
				Value: types.StringValue("2001:db8:85a3:0:0:8a2e:370:7334"),
			},
		},
		{
			name: "blackhole",
			args: args{
				routeResp: &iaas.Route{
					Nexthop: &iaas.RouteNexthop{
						NexthopBlackhole: &iaas.NexthopBlackhole{
							Type: utils.Ptr("blackhole"),
						},
					},
				},
			},
			want: &NexthopModelV1{
				Type: types.StringValue("blackhole"),
			},
		},
		{
			name: "internet",
			args: args{
				routeResp: &iaas.Route{
					Nexthop: &iaas.RouteNexthop{
						NexthopInternet: &iaas.NexthopInternet{
							Type: utils.Ptr("internet"),
						},
					},
				},
			},
			want: &NexthopModelV1{
				Type: types.StringValue("internet"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapRouteNextHop(tt.args.routeResp)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapRouteNextHop() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapRouteNextHop() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapRouteDestination(t *testing.T) {
	type args struct {
		routeResp *iaas.Route
	}
	tests := []struct {
		name    string
		args    args
		want    *DestinationModelV1
		wantErr bool
	}{
		{
			name: "cidrv4",
			args: args{
				routeResp: &iaas.Route{
					Destination: &iaas.RouteDestination{
						DestinationCIDRv4: &iaas.DestinationCIDRv4{
							Type:  utils.Ptr("cidrv4"),
							Value: utils.Ptr("192.168.1.0/24"),
						},
					},
				},
			},
			want: &DestinationModelV1{
				Type:  types.StringValue("cidrv4"),
				Value: types.StringValue("192.168.1.0/24"),
			},
		},
		{
			name: "cidrv6",
			args: args{
				routeResp: &iaas.Route{
					Destination: &iaas.RouteDestination{
						DestinationCIDRv4: &iaas.DestinationCIDRv4{
							Type:  utils.Ptr("cidrv6"),
							Value: utils.Ptr("2001:db8:1234::/48"),
						},
					},
				},
			},
			want: &DestinationModelV1{
				Type:  types.StringValue("cidrv6"),
				Value: types.StringValue("2001:db8:1234::/48"),
			},
		},
		{
			name: "destination in API response is nil",
			args: args{
				routeResp: &iaas.Route{
					Destination: nil,
				},
			},
			want: &DestinationModelV1{
				Type:  types.StringNull(),
				Value: types.StringNull(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapRouteDestination(tt.args.routeResp)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapRouteDestination() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapRouteDestination() got = %v, want %v", got, tt.want)
			}
		})
	}
}
