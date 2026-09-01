package bgpfilter

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
)

var (
	projectId = uuid.NewString()
	gatewayId = uuid.NewString()
	region    = "eu01"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *vpn.BGPFilter
		expected    Model
		isValid     bool
	}{
		{
			description: "default_ok",
			state: Model{
				ProjectId: types.StringValue(projectId),
				GatewayId: types.StringValue(gatewayId),
			},
			input: &vpn.BGPFilter{
				Id:          new("filter-id"),
				DisplayName: "test-filter",
			},
			expected: Model{
				Id:          types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, region, gatewayId, "filter-id")),
				ProjectId:   types.StringValue(projectId),
				Region:      types.StringValue(region),
				GatewayId:   types.StringValue(gatewayId),
				FilterId:    types.StringValue("filter-id"),
				DisplayName: types.StringValue("test-filter"),
			},
			isValid: true,
		},
		{
			description: "nil_response",
			state:       Model{},
			input:       nil,
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "nil_filter_id",
			state: Model{
				ProjectId: types.StringValue(projectId),
				GatewayId: types.StringValue(gatewayId),
			},
			input: &vpn.BGPFilter{
				Id:          nil,
				DisplayName: "test-filter",
			},
			expected: Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := tt.state
			err := mapFields(tt.input, &state, region)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.isValid {
				if diff := cmp.Diff(tt.expected, state); diff != "" {
					t.Fatalf("Data mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	model := &Model{
		DisplayName: types.StringValue("test-filter"),
	}
	expected := &vpn.CreateGatewayBGPFilterPayload{
		DisplayName: "test-filter",
	}

	payload := toCreatePayload(model)

	if diff := cmp.Diff(expected, payload); diff != "" {
		t.Fatalf("Data does not match (-want +got):\n%s", diff)
	}
}

func TestToUpdatePayload(t *testing.T) {
	model := &Model{
		DisplayName: types.StringValue("updated-filter"),
	}
	expected := &vpn.UpdateGatewayBGPFilterPayload{
		DisplayName: "updated-filter",
	}

	payload := toUpdatePayload(model)

	if diff := cmp.Diff(expected, payload); diff != "" {
		t.Fatalf("Data does not match (-want +got):\n%s", diff)
	}
}
