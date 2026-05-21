package gateway

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
)

var (
	projectId = uuid.NewString()
	region    = "eu01"
)

func TestMapFields(t *testing.T) {
	type args struct {
		state Model
		input *vpn.GatewayResponse
	}
	tests := []struct {
		description string
		args        args
		expected    Model
		isValid     bool
	}{
		{
			description: "default_ok",
			args: args{
				state: Model{
					ProjectId: types.StringValue(projectId),
				},
				input: &vpn.GatewayResponse{
					Id:          new("gateway-id"),
					DisplayName: "test-gateway",
					PlanId:      "p500",
					RoutingType: vpn.ROUTINGTYPE_ROUTE_BASED,
					AvailabilityZones: vpn.GatewayAvailabilityZones{
						Tunnel1: "eu01-1",
						Tunnel2: "eu01-2",
					},
					State: new(vpn.GatewayStatus("READY")),
				},
			},
			expected: Model{
				Id:          types.StringValue(fmt.Sprintf("%s,%s,%s", projectId, region, "gateway-id")),
				ProjectId:   types.StringValue(projectId),
				Region:      types.StringValue(region),
				GatewayId:   types.StringValue("gateway-id"),
				DisplayName: types.StringValue("test-gateway"),
				PlanId:      types.StringValue("p500"),
				RoutingType: types.StringValue("ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
				Bgp:    nil,
				Labels: types.MapNull(types.StringType),
				State:  types.StringValue("READY"),
			},
			isValid: true,
		},
		{
			description: "with_bgp_and_labels",
			args: args{
				state: Model{
					ProjectId: types.StringValue(projectId),
				},
				input: &vpn.GatewayResponse{
					Id:          new("gateway-id"),
					DisplayName: "test-gateway",
					PlanId:      "p500",
					RoutingType: vpn.ROUTINGTYPE_BGP_ROUTE_BASED,
					AvailabilityZones: vpn.GatewayAvailabilityZones{
						Tunnel1: "eu01-1",
						Tunnel2: "eu01-2",
					},
					Bgp: &vpn.BGPGatewayConfig{
						LocalAsn:                 new(int64(65000)),
						OverrideAdvertisedRoutes: []string{"10.0.0.0/16", "192.168.0.0/24"},
					},
					Labels: &map[string]string{
						"env":  "prod",
						"team": "network",
					},
					State: new(vpn.GatewayStatus("READY")),
				},
			},
			expected: Model{
				Id:          types.StringValue(fmt.Sprintf("%s,%s,%s", projectId, region, "gateway-id")),
				ProjectId:   types.StringValue(projectId),
				Region:      types.StringValue(region),
				GatewayId:   types.StringValue("gateway-id"),
				DisplayName: types.StringValue("test-gateway"),
				PlanId:      types.StringValue("p500"),
				RoutingType: types.StringValue("BGP_ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
				Bgp: &BGPGatewayConfigModel{
					LocalAsn: types.Int64Value(65000),
					OverrideAdvertisedRoutes: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("10.0.0.0/16"),
						types.StringValue("192.168.0.0/24"),
					}),
				},
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"env":  types.StringValue("prod"),
					"team": types.StringValue("network"),
				}),
				State: types.StringValue("READY"),
			},
			isValid: true,
		},
		{
			description: "preserve_empty_routes_and_labels_from_state",
			args: args{
				state: Model{
					ProjectId: types.StringValue(projectId),
					Bgp: &BGPGatewayConfigModel{
						OverrideAdvertisedRoutes: types.ListValueMust(types.StringType, []attr.Value{}),
					},
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{}),
				},
				input: &vpn.GatewayResponse{
					Id:          new("gateway-id"),
					DisplayName: "test-gateway",
					PlanId:      "p500",
					RoutingType: vpn.ROUTINGTYPE_BGP_ROUTE_BASED,
					AvailabilityZones: vpn.GatewayAvailabilityZones{
						Tunnel1: "eu01-1",
						Tunnel2: "eu01-2",
					},
					Bgp: &vpn.BGPGatewayConfig{
						LocalAsn:                 new(int64(65000)),
						OverrideAdvertisedRoutes: nil,
					},
					Labels: nil,
					State:  new(vpn.GatewayStatus("READY")),
				},
			},
			expected: Model{
				Id:          types.StringValue(fmt.Sprintf("%s,%s,%s", projectId, region, "gateway-id")),
				ProjectId:   types.StringValue(projectId),
				Region:      types.StringValue(region),
				GatewayId:   types.StringValue("gateway-id"),
				DisplayName: types.StringValue("test-gateway"),
				PlanId:      types.StringValue("p500"),
				RoutingType: types.StringValue("BGP_ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
				Bgp: &BGPGatewayConfigModel{
					LocalAsn:                 types.Int64Value(65000),
					OverrideAdvertisedRoutes: types.ListValueMust(types.StringType, []attr.Value{}),
				},
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{}),
				State:  types.StringValue("READY"),
			},
			isValid: true,
		},
		{
			description: "nil_response",
			args: args{
				state: Model{},
				input: nil,
			},
			expected: Model{},
			isValid:  false,
		},
		{
			description: "nil_gateway_id",
			args: args{
				state: Model{
					ProjectId: types.StringValue(projectId),
				},
				input: &vpn.GatewayResponse{
					Id:          nil,
					DisplayName: "test-gateway",
				},
			},
			expected: Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.args.input, &tt.args.state, region)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.isValid {
				if diff := cmp.Diff(tt.expected, tt.args.state); diff != "" {
					t.Fatalf("Data mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *vpn.CreateGatewayPayload
		isValid     bool
	}{
		{
			description: "basic_gateway",
			input: &Model{
				DisplayName: types.StringValue("test-gateway"),
				PlanId:      types.StringValue("p500"),
				RoutingType: types.StringValue("ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
			},
			expected: &vpn.CreateGatewayPayload{
				DisplayName: "test-gateway",
				PlanId:      "p500",
				RoutingType: vpn.RoutingType("ROUTE_BASED"),
				AvailabilityZones: vpn.CreateGatewayPayloadAvailabilityZones{
					Tunnel1: "eu01-1",
					Tunnel2: "eu01-2",
				},
				Labels: &map[string]string{},
			},
			isValid: true,
		},
		{
			description: "with_bgp_routes_and_labels",
			input: &Model{
				DisplayName: types.StringValue("test-gateway"),
				PlanId:      types.StringValue("p500"),
				RoutingType: types.StringValue("BGP_ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
				Bgp: &BGPGatewayConfigModel{
					LocalAsn: types.Int64Value(65000),
				},
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"env":  types.StringValue("prod"),
					"team": types.StringValue("network"),
				}),
			},
			expected: &vpn.CreateGatewayPayload{
				DisplayName: "test-gateway",
				PlanId:      "p500",
				RoutingType: vpn.RoutingType("BGP_ROUTE_BASED"),
				AvailabilityZones: vpn.CreateGatewayPayloadAvailabilityZones{
					Tunnel1: "eu01-1",
					Tunnel2: "eu01-2",
				},
				Bgp: &vpn.BGPGatewayConfig{
					LocalAsn: new(int64(65000)),
				},
				Labels: &map[string]string{
					"env":  "prod",
					"team": "network",
				},
			},
			isValid: true,
		},
		{
			description: "nil_model",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toCreatePayload(context.Background(), tt.input)

			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.expected, payload)
				if diff != "" {
					t.Fatalf("Data does not match (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *vpn.UpdateGatewayPayload
		isValid     bool
	}{
		{
			description: "basic_update",
			input: &Model{
				DisplayName: types.StringValue("updated-gateway"),
				PlanId:      types.StringValue("p1000"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
			},
			expected: &vpn.UpdateGatewayPayload{
				DisplayName: "updated-gateway",
				PlanId:      "p1000",
				AvailabilityZones: vpn.UpdateGatewayPayloadAvailabilityZones{
					Tunnel1: "eu01-1",
					Tunnel2: "eu01-2",
				},
				Labels: &map[string]string{},
			},
			isValid: true,
		},
		{
			description: "with_bgp_routes_and_labels",
			input: &Model{
				DisplayName: types.StringValue("test-gateway"),
				PlanId:      types.StringValue("p500"),
				RoutingType: types.StringValue("BGP_ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
				Bgp: &BGPGatewayConfigModel{
					LocalAsn: types.Int64Value(65000),
				},
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"env":  types.StringValue("prod"),
					"team": types.StringValue("network"),
				}),
			},
			expected: &vpn.UpdateGatewayPayload{
				DisplayName: "test-gateway",
				PlanId:      "p500",
				RoutingType: vpn.RoutingType("BGP_ROUTE_BASED"),
				AvailabilityZones: vpn.UpdateGatewayPayloadAvailabilityZones{
					Tunnel1: "eu01-1",
					Tunnel2: "eu01-2",
				},
				Bgp: &vpn.BGPGatewayConfig{
					LocalAsn: new(int64(65000)),
				},
				Labels: &map[string]string{
					"env":  "prod",
					"team": "network",
				},
			},
			isValid: true,
		},
		{
			description: "nil_model",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toUpdatePayload(context.Background(), tt.input)

			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.expected, payload)
				if diff != "" {
					t.Fatalf("Data does not match (-want +got):\n%s", diff)
				}
			}
		})
	}
}
