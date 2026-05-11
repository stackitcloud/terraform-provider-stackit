package gateway

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1beta1api"
)

func TestDataSourceMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *vpn.GatewayResponse
		expected    Model
		isValid     bool
	}{
		{
			"basic_gateway",
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
				State: types.StringValue("READY"),
			},
			true,
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
		})
	}
}
