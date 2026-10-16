package instance

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ufw "github.com/stackitcloud/stackit-sdk-go/services/ufw/v1api"
)

func TestMapFields(t *testing.T) {
	const (
		testRegion    = "eu01"
		testProjectId = "1234"
		testRuleId    = "4321"
	)

	tests := []struct {
		description string
		input       *ufw.RuleResponse
		state       *Model
		expected    *Model
		isValid     bool
	}{
		{
			description: "valid_mapping",
			input: &ufw.RuleResponse{
				InstanceId: "target-instance-123",
				Product:    "edge-cloud",
				SourceIP:   "192.168.0.0/24",
				Type:       "ACL",
			},
			state: &Model{
				ProjectId: types.StringValue(testProjectId),
				RuleId:    types.StringValue(testRuleId),
			},
			expected: &Model{
				ProjectId:  types.StringValue(testProjectId),
				RuleId:     types.StringValue(testRuleId),
				Region:     types.StringValue(testRegion),
				InstanceId: types.StringValue("target-instance-123"),
				Product:    types.StringValue("edge-cloud"),
				SourceIP:   types.StringValue("192.168.0.0/24"),
				Type:       types.StringValue("ACL"),
			},
			isValid: true,
		},
		{
			description: "nil_response",
			input:       nil,
			state: &Model{
				ProjectId: types.StringValue(testProjectId),
			},
			expected: &Model{},
			isValid:  false,
		},
		{
			description: "nil_model",
			input:       &ufw.RuleResponse{},
			state:       nil,
			expected:    nil,
			isValid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(tt.input, tt.state, testRegion)

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

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *ufw.CreateRulePayload
		isValid     bool
	}{
		{
			description: "valid_payload",
			input: &Model{
				ProjectId:  types.StringValue("1234"),
				Region:     types.StringValue("eu01"),
				InstanceId: types.StringValue("target-instance-123"),
				Product:    types.StringValue("edge-cloud"),
				SourceIP:   types.StringValue("192.168.0.0/24"),
				Type:       types.StringValue("ACL"),
			},
			expected: ufw.NewCreateRulePayload(
				"target-instance-123",
				"edge-cloud",
				"192.168.0.0/24",
				"ACL",
			),
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
			output, err := toCreatePayload(tt.input)

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
		input       *Model
		expected    *ufw.UpdateRulePayload
		isValid     bool
	}{
		{
			description: "valid_payload",
			input: &Model{
				SourceIP: types.StringValue("10.0.0.0/8"),
			},
			expected: ufw.NewUpdateRulePayload("10.0.0.0/8"),
			isValid:  true,
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
			output, err := toUpdatePayload(tt.input)

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
