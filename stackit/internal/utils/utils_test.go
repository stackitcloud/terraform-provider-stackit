package utils

import (
	"context"
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

func TestSimplifyBackupSchedule(t *testing.T) {
	tests := []struct {
		description string
		input       string
		expected    string
	}{
		{
			"simple schedule",
			"0 0 * * *",
			"0 0 * * *",
		},
		{
			"schedule with leading zeros",
			"00 00 * * *",
			"0 0 * * *",
		},
		{
			"schedule with leading zeros 2",
			"00 001 * * *",
			"0 1 * * *",
		},
		{
			"schedule with leading zeros 3",
			"00 0010 * * *",
			"0 10 * * *",
		},
		{
			"simple schedule with slash",
			"0 0/6 * * *",
			"0 0/6 * * *",
		},
		{
			"schedule with leading zeros and slash",
			"00 00/6 * * *",
			"0 0/6 * * *",
		},
		{
			"schedule with leading zeros and slash 2",
			"00 001/06 * * *",
			"0 1/6 * * *",
		},
		{
			"simple schedule with comma",
			"0 10,15 * * *",
			"0 10,15 * * *",
		},
		{
			"schedule with leading zeros and comma",
			"0 010,0015 * * *",
			"0 10,15 * * *",
		},
		{
			"simple schedule with comma and slash",
			"0 0-11/10 * * *",
			"0 0-11/10 * * *",
		},
		{
			"schedule with leading zeros, comma, and slash",
			"00 000-011/010 * * *",
			"0 0-11/10 * * *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := SimplifyBackupSchedule(tt.input)
			if output != tt.expected {
				t.Fatalf("Data does not match: %s", output)
			}
		})
	}
}

func TestSupportedValuesDocumentation(t *testing.T) {
	tests := []struct {
		description string
		values      []string
		expected    string
	}{
		{
			"empty values",
			[]string{},
			"",
		},
		{
			"single value",
			[]string{"value"},
			"Supported values are: `value`.",
		},
		{
			"multiple values",
			[]string{"value1", "value2", "value3"},
			"Supported values are: `value1`, `value2`, `value3`.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := SupportedValuesDocumentation(tt.values)
			if output != tt.expected {
				t.Fatalf("Data does not match: %s", output)
			}
		})
	}
}

func TestIsLegacyProjectRole(t *testing.T) {
	tests := []struct {
		description string
		role        string
		expected    bool
	}{
		{
			"non legacy role",
			"owner",
			false,
		},
		{
			"leagcy role",
			"project.owner",
			true,
		},
		{
			"leagcy role 2",
			"project.admin",
			true,
		},
		{
			"leagcy role 3",
			"project.member",
			true,
		},
		{
			"leagcy role 4",
			"project.auditor",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := IsLegacyProjectRole(tt.role)
			if output != tt.expected {
				t.Fatalf("Data does not match: %v", output)
			}
		})
	}
}

func TestToJSONMapUpdatePayload(t *testing.T) {
	tests := []struct {
		description   string
		currentLabels types.Map
		desiredLabels types.Map
		expected      map[string]interface{}
		isValid       bool
	}{
		{
			"nothing_to_update",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			map[string]interface{}{
				"key": "value",
			},
			true,
		},
		{
			"update_key_value",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("updated_value"),
			}),
			map[string]interface{}{
				"key": "updated_value",
			},
			true,
		},
		{
			"remove_key",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			map[string]interface{}{
				"key":  "value",
				"key2": nil,
			},
			true,
		},
		{
			"add_new_key",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			map[string]interface{}{
				"key":  "value",
				"key2": "value2",
			},
			true,
		},
		{
			"empty_desired_map",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			types.MapValueMust(types.StringType, map[string]attr.Value{}),
			map[string]interface{}{
				"key":  nil,
				"key2": nil,
			},
			true,
		},
		{
			"nil_desired_map",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			types.MapNull(types.StringType),
			map[string]interface{}{
				"key":  nil,
				"key2": nil,
			},
			true,
		},
		{
			"empty_current_map",
			types.MapValueMust(types.StringType, map[string]attr.Value{}),
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			map[string]interface{}{
				"key":  "value",
				"key2": "value2",
			},
			true,
		},
		{
			"nil_current_map",
			types.MapNull(types.StringType),
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			map[string]interface{}{
				"key":  "value",
				"key2": "value2",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := ToJSONMapPartialUpdatePayload(context.Background(), tt.currentLabels, tt.desiredLabels)
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
