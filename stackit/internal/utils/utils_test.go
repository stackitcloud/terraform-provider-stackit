package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestReconcileStrLists(t *testing.T) {
	tests := []struct {
		description string
		list1       []string
		list2       []string
		expected    []string
	}{
		{
			"empty lists",
			[]string{},
			[]string{},
			[]string{},
		},
		{
			"list1 empty",
			[]string{},
			[]string{"a", "b", "c"},
			[]string{"a", "b", "c"},
		},
		{
			"list2 empty",
			[]string{"a", "b", "c"},
			[]string{},
			[]string{},
		},
		{
			"no common elements",
			[]string{"a", "b", "c"},
			[]string{"d", "e", "f"},
			[]string{"d", "e", "f"},
		},
		{
			"common elements",
			[]string{"d", "a", "c"},
			[]string{"b", "c", "d", "e"},
			[]string{"d", "c", "b", "e"},
		},
		{
			"common elements with empty string",
			[]string{"d", "", "c"},
			[]string{"", "c", "d"},
			[]string{"d", "", "c"},
		},
		{
			"common elements with duplicates",
			[]string{"a", "b", "c", "c"},
			[]string{"b", "c", "d", "e"},
			[]string{"b", "c", "c", "d", "e"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := ReconcileStringSlices(tt.list1, tt.list2)
			diff := cmp.Diff(output, tt.expected)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func TestListValuetoStrSlice(t *testing.T) {
	tests := []struct {
		description string
		input       basetypes.ListValue
		expected    []string
		isValid     bool
	}{
		{
			description: "empty list",
			input:       types.ListValueMust(types.StringType, []attr.Value{}),
			expected:    []string{},
			isValid:     true,
		},
		{
			description: "values ok",
			input: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("a"),
				types.StringValue("b"),
				types.StringValue("c"),
			}),
			expected: []string{"a", "b", "c"},
			isValid:  true,
		},
		{
			description: "different type",
			input: types.ListValueMust(types.Int64Type, []attr.Value{
				types.Int64Value(12),
			}),
			isValid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := ListValuetoStringSlice(tt.input)
			if err != nil {
				if !tt.isValid {
					return
				}
				t.Fatalf("Should not have failed: %v", err)
			}
			if !tt.isValid {
				t.Fatalf("Should have failed")
			}
			diff := cmp.Diff(output, tt.expected)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}
