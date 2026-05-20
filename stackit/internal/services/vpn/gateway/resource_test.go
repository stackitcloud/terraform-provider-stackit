package gateway

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
)

var (
	projectId = uuid.NewString()
	region    = "eu01"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *vpn.GatewayResponse
		expected    Model
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &vpn.GatewayResponse{
				Id:          new("gateway-id"),
				DisplayName: "test-gateway",
				PlanId:      "p500",
				RoutingType: vpn.ROUTINGTYPE_ROUTE_BASED,
				AvailabilityZones: vpn.GatewayAvailabilityZones{
					Tunnel1: "eu01-1",
					Tunnel2: "eu01-2",
				},
				State: utils.Ptr(vpn.GATEWAYSTATUS_READY),
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
				State: utils.Ptr(vpn.GATEWAYSTATUS_READY),
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
			description: "nil_response",
			input:       nil,
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "nil_gateway_id",
			input: &vpn.GatewayResponse{
				Id:          nil,
				DisplayName: "test-gateway",
			},
			expected: Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var model Model
			model.ProjectId = types.StringValue(projectId)

			err := mapFields(context.Background(), tt.input, &model, region)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.isValid {
				if diff := cmp.Diff(tt.expected, model); diff != "" {
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
			},
			isValid: true,
		},
		{
			description: "with_bgp",
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
