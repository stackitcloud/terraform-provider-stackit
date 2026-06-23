package utils

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestMapLabels(t *testing.T) {
	type args struct {
		currentLabels  types.Map
		responseLabels *map[string]string
	}
	tests := []struct {
		name           string
		input          args
		expectedOutput basetypes.MapValue
		isValid        bool
	}{
		{
			name: "No labels, no map",
			input: args{
				currentLabels:  types.MapNull(types.StringType),
				responseLabels: &map[string]string{},
			},
			expectedOutput: types.MapNull(types.StringType),
			isValid:        true,
		},
		{
			name: "No labels, empty map",
			input: args{
				currentLabels:  types.MapValueMust(types.StringType, map[string]attr.Value{}),
				responseLabels: &map[string]string{},
			},
			expectedOutput: types.MapValueMust(types.StringType, map[string]attr.Value{}),
			isValid:        true,
		},
		{
			name: "Add Labels",
			input: args{
				currentLabels: types.MapNull(types.StringType),
				responseLabels: &map[string]string{
					"foo": "bar",
				},
			},
			expectedOutput: types.MapValueMust(types.StringType, map[string]attr.Value{
				"foo": types.StringValue("bar"),
			}),
			isValid: true,
		},
		{
			name: "Remove Labels",
			input: args{
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"foo": types.StringValue("bar"),
				}),
				responseLabels: &map[string]string{},
			},
			expectedOutput: types.MapValueMust(types.StringType, map[string]attr.Value{}),
			isValid:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := MapLabels(context.Background(), tt.input.responseLabels, tt.input.currentLabels)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expectedOutput)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestLabelsToPayload(t *testing.T) {
	tests := []struct {
		name           string
		input          types.Map
		expectedOutput map[string]string
		isValid        bool
	}{
		{
			name:           "No labels, no map",
			input:          types.MapNull(types.StringType),
			expectedOutput: map[string]string{},
			isValid:        true,
		},
		{
			name:           "No labels, empty map",
			input:          types.MapValueMust(types.StringType, map[string]attr.Value{}),
			expectedOutput: map[string]string{},
			isValid:        true,
		},
		{
			name: "Valid Labels",
			input: types.MapValueMust(types.StringType, map[string]attr.Value{
				"foo": types.StringValue("bar"),
			}),
			expectedOutput: map[string]string{
				"foo": "bar",
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := LabelsToPayload(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expectedOutput)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
