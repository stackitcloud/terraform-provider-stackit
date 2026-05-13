package gateway

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *vpn.GatewayResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_ok",
			&vpn.GatewayResponse{
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
			Model{
				GatewayID:   types.StringValue("gateway-id"),
				DisplayName: types.StringValue("test-gateway"),
				PlanID:      types.StringValue("p500"),
				RoutingType: types.StringValue("ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
				Bgp:    nil,
				Labels: types.MapNull(types.StringType),
				State:  types.StringValue("READY"),
			},
			true,
		},
		{
			"with_bgp_and_labels",
			&vpn.GatewayResponse{
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
			Model{
				GatewayID:   types.StringValue("gateway-id"),
				DisplayName: types.StringValue("test-gateway"),
				PlanID:      types.StringValue("p500"),
				RoutingType: types.StringValue("BGP_ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
				Bgp: &BGPGatewayConfigModel{
					LocalAsn: types.Int64Value(65000),
				},
				State: types.StringValue("READY"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"nil_gateway_id",
			&vpn.GatewayResponse{
				Id:          nil,
				DisplayName: "test-gateway",
			},
			Model{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var model Model
			model.ProjectID = types.StringValue("test-project")
			model.Region = types.StringValue("eu01")

			err := mapFields(context.Background(), tt.input, &model, "eu01")

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !tt.isValid {
				return
			}

			if diff := cmp.Diff(model.GatewayID, tt.expected.GatewayID); diff != "" {
				t.Fatalf("GatewayID mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(model.DisplayName, tt.expected.DisplayName); diff != "" {
				t.Fatalf("DisplayName mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(model.PlanID, tt.expected.PlanID); diff != "" {
				t.Fatalf("PlanID mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(model.RoutingType, tt.expected.RoutingType); diff != "" {
				t.Fatalf("RoutingType mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(model.State, tt.expected.State); diff != "" {
				t.Fatalf("State mismatch (-got +want):\n%s", diff)
			}

			if diff := cmp.Diff(model.AvailabilityZones.Tunnel1, tt.expected.AvailabilityZones.Tunnel1); diff != "" {
				t.Fatalf("AZ Tunnel1 mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(model.AvailabilityZones.Tunnel2, tt.expected.AvailabilityZones.Tunnel2); diff != "" {
				t.Fatalf("AZ Tunnel2 mismatch (-got +want):\n%s", diff)
			}

			if tt.expected.Bgp != nil {
				if model.Bgp == nil {
					t.Fatalf("expected BGP config, got nil")
				}
				if diff := cmp.Diff(model.Bgp.LocalAsn, tt.expected.Bgp.LocalAsn); diff != "" {
					t.Fatalf("BGP LocalAsn mismatch (-got +want):\n%s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       Model
		isValid     bool
	}{
		{
			"basic_gateway",
			Model{
				DisplayName: types.StringValue("test-gateway"),
				PlanID:      types.StringValue("p500"),
				RoutingType: types.StringValue("ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
			},
			true,
		},
		{
			"with_bgp",
			Model{
				DisplayName: types.StringValue("test-gateway"),
				PlanID:      types.StringValue("p500"),
				RoutingType: types.StringValue("BGP_ROUTE_BASED"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
				Bgp: &BGPGatewayConfigModel{
					LocalAsn: types.Int64Value(65000),
				},
			},
			true,
		},
		{
			"nil_model",
			Model{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var model *Model
			if tt.isValid {
				model = &tt.input
			}

			payload, err := toCreatePayload(context.Background(), model)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !tt.isValid {
				return
			}

			if payload.DisplayName != tt.input.DisplayName.ValueString() {
				t.Errorf("DisplayName mismatch: got %v, want %v", payload.DisplayName, tt.input.DisplayName.ValueString())
			}
			if payload.PlanId != tt.input.PlanID.ValueString() {
				t.Errorf("PlanId mismatch: got %v, want %v", payload.PlanId, tt.input.PlanID.ValueString())
			}
			if string(payload.RoutingType) != tt.input.RoutingType.ValueString() {
				t.Errorf("RoutingType mismatch: got %v, want %v", payload.RoutingType, tt.input.RoutingType.ValueString())
			}

			if tt.input.Bgp != nil {
				if payload.Bgp == nil {
					t.Errorf("expected BGP config, got nil")
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       Model
		isValid     bool
	}{
		{
			"basic_update",
			Model{
				DisplayName: types.StringValue("updated-gateway"),
				PlanID:      types.StringValue("p1000"),
				AvailabilityZones: &AvailabilityZonesModel{
					Tunnel1: types.StringValue("eu01-1"),
					Tunnel2: types.StringValue("eu01-2"),
				},
			},
			true,
		},
		{
			"nil_model",
			Model{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var model *Model
			if tt.isValid {
				model = &tt.input
			}

			payload, err := toUpdatePayload(context.Background(), model)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !tt.isValid {
				return
			}

			if payload.DisplayName != tt.input.DisplayName.ValueString() {
				t.Errorf("DisplayName mismatch: got %v, want %v", payload.DisplayName, tt.input.DisplayName.ValueString())
			}
			if payload.PlanId != tt.input.PlanID.ValueString() {
				t.Errorf("PlanId mismatch: got %v, want %v", payload.PlanId, tt.input.PlanID.ValueString())
			}
		})
	}
}
