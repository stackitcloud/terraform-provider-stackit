package utils

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
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

func TestConvertPointerSliceToStringSlice(t *testing.T) {
	tests := []struct {
		description string
		input       []*string
		expected    []string
	}{
		{
			description: "nil slice",
			input:       nil,
			expected:    []string{},
		},
		{
			description: "empty slice",
			input:       []*string{},
			expected:    []string{},
		},
		{
			description: "slice with valid pointers",
			input:       []*string{utils.Ptr("apple"), utils.Ptr("banana"), utils.Ptr("cherry")},
			expected:    []string{"apple", "banana", "cherry"},
		},
		{
			description: "slice with some nil pointers",
			input:       []*string{utils.Ptr("apple"), nil, utils.Ptr("cherry"), nil},
			expected:    []string{"apple", "cherry"},
		},
		{
			description: "slice with all nil pointers",
			input:       []*string{nil, nil, nil},
			expected:    []string{},
		},
		{
			description: "slice with a pointer to an empty string",
			input:       []*string{utils.Ptr("apple"), utils.Ptr(""), utils.Ptr("cherry")},
			expected:    []string{"apple", "", "cherry"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := ConvertPointerSliceToStringSlice(tt.input)
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

func TestFormatPossibleValues(t *testing.T) {
	gotPrefix := "Possible values are:"

	type args struct {
		values []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "single string value",
			args: args{
				values: []string{"foo"},
			},
			want: fmt.Sprintf("%s `foo`.", gotPrefix),
		},
		{
			name: "multiple string value",
			args: args{
				values: []string{"foo", "bar", "trololol"},
			},
			want: fmt.Sprintf("%s `foo`, `bar`, `trololol`.", gotPrefix),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatPossibleValues(tt.args.values...); got != tt.want {
				t.Errorf("FormatPossibleValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUndefined(t *testing.T) {
	type args struct {
		val value
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "undefined value",
			args: args{
				val: types.StringNull(),
			},
			want: true,
		},
		{
			name: "unknown value",
			args: args{
				val: types.StringUnknown(),
			},
			want: true,
		},
		{
			name: "string value",
			args: args{
				val: types.StringValue(""),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUndefined(tt.args.val); got != tt.want {
				t.Errorf("IsUndefined() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildInternalTerraformId(t *testing.T) {
	type args struct {
		idParts []string
	}
	tests := []struct {
		name string
		args args
		want types.String
	}{
		{
			name: "no id parts",
			args: args{
				idParts: []string{},
			},
			want: types.StringValue(""),
		},
		{
			name: "multiple id parts",
			args: args{
				idParts: []string{"abc", "foo", "bar", "xyz"},
			},
			want: types.StringValue("abc,foo,bar,xyz"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildInternalTerraformId(tt.args.idParts...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildInternalTerraformId() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckListRemoval(t *testing.T) {
	type model struct {
		AllowedAddresses types.List `tfsdk:"allowed_addresses"`
	}
	tests := []struct {
		description          string
		configModelList      types.List
		planModelList        types.List
		path                 path.Path
		listType             attr.Type
		createEmptyList      bool
		expectedAdjustedResp bool
	}{
		{
			"config and plan are the same - no change",
			types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("value1"),
			}),
			types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("value1"),
			}),
			path.Root("allowed_addresses"),
			types.StringType,
			false,
			false,
		},
		{
			"list was removed from config",
			types.ListNull(types.StringType),
			types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("value1"),
			}),
			path.Root("allowed_addresses"),
			types.StringType,
			false,
			true,
		},
		{
			"list was added to config",
			types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("value1"),
			}),
			types.ListNull(types.StringType),
			path.Root("allowed_addresses"),
			types.StringType,
			false,
			false,
		},
		{
			"no list provided at all",
			types.ListNull(types.StringType),
			types.ListNull(types.StringType),
			path.Root("allowed_addresses"),
			types.StringType,
			false,
			false,
		},
		{
			"create empty list test - list was removed from config",
			types.ListNull(types.StringType),
			types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("value1"),
			}),
			path.Root("allowed_addresses"),
			types.StringType,
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// create resp
			plan := tfsdk.Plan{
				Schema: schema.Schema{
					Attributes: map[string]schema.Attribute{
						"allowed_addresses": schema.ListAttribute{
							ElementType: basetypes.StringType{},
						},
					},
				},
			}

			// set input planModelList to plan
			if diags := plan.Set(context.Background(), model{tt.planModelList}); diags.HasError() {
				t.Fatalf("cannot create test model: %v", diags)
			}
			resp := resource.ModifyPlanResponse{
				Plan: plan,
			}

			CheckListRemoval(context.Background(), tt.configModelList, tt.planModelList, tt.path, tt.listType, tt.createEmptyList, &resp)
			// check targetList
			var respList types.List
			resp.Plan.GetAttribute(context.Background(), tt.path, &respList)

			if tt.createEmptyList {
				emptyList, _ := types.ListValueFrom(context.Background(), tt.listType, []string{})
				diffEmptyList := cmp.Diff(emptyList, respList)
				if diffEmptyList != "" {
					t.Fatalf("an empty list should have been created but was not: %s", diffEmptyList)
				}
			}

			// compare planModelList and resp list
			diff := cmp.Diff(tt.planModelList, respList)
			if tt.expectedAdjustedResp {
				if diff == "" {
					t.Fatalf("plan should be adjusted but was not")
				}
			} else {
				if diff != "" {
					t.Fatalf("plan should not be adjusted but diff is: %s", diff)
				}
			}
		})
	}
}

func TestSetAndLogStateFields(t *testing.T) {
	testSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_id":  schema.StringAttribute{},
			"instance_id": schema.StringAttribute{},
		},
	}

	type args struct {
		diags  *diag.Diagnostics
		state  *tfsdk.State
		values map[string]interface{}
	}
	type want struct {
		hasError bool
		state    *tfsdk.State
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "empty map",
			args: args{
				diags:  &diag.Diagnostics{},
				state:  &tfsdk.State{},
				values: map[string]interface{}{},
			},
			want: want{
				hasError: false,
				state:    &tfsdk.State{},
			},
		},
		{
			name: "base",
			args: args{
				diags: &diag.Diagnostics{},
				state: func() *tfsdk.State {
					ctx := context.Background()
					state := tfsdk.State{
						Raw: tftypes.NewValue(testSchema.Type().TerraformType(ctx), map[string]tftypes.Value{
							"project_id":  tftypes.NewValue(tftypes.String, "9b15d120-86f8-45f5-81d8-a554f09c7582"),
							"instance_id": tftypes.NewValue(tftypes.String, nil),
						}),
						Schema: testSchema,
					}
					return &state
				}(),
				values: map[string]interface{}{
					"project_id":  "a414f971-3f7a-4e9a-8671-51a8acb7bcc8",
					"instance_id": "97073250-8cad-46c3-8424-6258ac0b3731",
				},
			},
			want: want{
				hasError: false,
				state: func() *tfsdk.State {
					ctx := context.Background()
					state := tfsdk.State{
						Raw: tftypes.NewValue(testSchema.Type().TerraformType(ctx), map[string]tftypes.Value{
							"project_id":  tftypes.NewValue(tftypes.String, nil),
							"instance_id": tftypes.NewValue(tftypes.String, nil),
						}),
						Schema: testSchema,
					}
					state.SetAttribute(ctx, path.Root("project_id"), "a414f971-3f7a-4e9a-8671-51a8acb7bcc8")
					state.SetAttribute(ctx, path.Root("instance_id"), "97073250-8cad-46c3-8424-6258ac0b3731")
					return &state
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			SetAndLogStateFields(ctx, tt.args.diags, tt.args.state, tt.args.values)

			if tt.args.diags.HasError() != tt.want.hasError {
				t.Errorf("TestSetAndLogStateFields() error count = %v, hasErr %v", tt.args.diags.ErrorsCount(), tt.want.hasError)
			}

			diff := cmp.Diff(tt.args.state, tt.want.state)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func TestSetModelFieldsToNull(t *testing.T) {
	ctx := context.Background()

	type TestModel struct {
		StringField  types.String  `tfsdk:"string_field"`
		BoolField    types.Bool    `tfsdk:"bool_field"`
		Int64Field   types.Int64   `tfsdk:"int64_field"`
		Float64Field types.Float64 `tfsdk:"float64_field"`
		ListField    types.List    `tfsdk:"list_field"`
		SetField     types.Set     `tfsdk:"set_field"`
		MapField     types.Map     `tfsdk:"map_field"`
		ObjectField  types.Object  `tfsdk:"object_field"`
	}

	tests := []struct {
		name        string
		input       *TestModel
		expected    *TestModel
		expectError bool
	}{
		{
			name: "all unknown fields should be set to null",
			input: &TestModel{
				StringField:  types.StringUnknown(),
				BoolField:    types.BoolUnknown(),
				Int64Field:   types.Int64Unknown(),
				Float64Field: types.Float64Unknown(),
				ListField:    types.ListUnknown(types.StringType),
				SetField:     types.SetUnknown(types.StringType),
				MapField:     types.MapUnknown(types.StringType),
				ObjectField:  types.ObjectUnknown(map[string]attr.Type{"field1": types.StringType}),
			},
			expected: &TestModel{
				StringField:  types.StringNull(),
				BoolField:    types.BoolNull(),
				Int64Field:   types.Int64Null(),
				Float64Field: types.Float64Null(),
				ListField:    types.ListNull(types.StringType),
				SetField:     types.SetNull(types.StringType),
				MapField:     types.MapNull(types.StringType),
				ObjectField:  types.ObjectNull(map[string]attr.Type{"field1": types.StringType}),
			},
			expectError: false,
		},
		{
			name: "all null fields should remain null",
			input: &TestModel{
				StringField:  types.StringNull(),
				BoolField:    types.BoolNull(),
				Int64Field:   types.Int64Null(),
				Float64Field: types.Float64Null(),
				ListField:    types.ListNull(types.StringType),
				SetField:     types.SetNull(types.StringType),
				MapField:     types.MapNull(types.StringType),
				ObjectField:  types.ObjectNull(map[string]attr.Type{"field1": types.StringType}),
			},
			expected: &TestModel{
				StringField:  types.StringNull(),
				BoolField:    types.BoolNull(),
				Int64Field:   types.Int64Null(),
				Float64Field: types.Float64Null(),
				ListField:    types.ListNull(types.StringType),
				SetField:     types.SetNull(types.StringType),
				MapField:     types.MapNull(types.StringType),
				ObjectField:  types.ObjectNull(map[string]attr.Type{"field1": types.StringType}),
			},
			expectError: false,
		},
		{
			name: "known fields should not be modified",
			input: &TestModel{
				StringField:  types.StringValue("test"),
				BoolField:    types.BoolValue(true),
				Int64Field:   types.Int64Value(42),
				Float64Field: types.Float64Value(3.14),
				ListField:    types.ListValueMust(types.StringType, []attr.Value{types.StringValue("item")}),
				SetField:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("item")}),
				MapField:     types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				ObjectField:  types.ObjectValueMust(map[string]attr.Type{"field1": types.StringType}, map[string]attr.Value{"field1": types.StringValue("value")}),
			},
			expected: &TestModel{
				StringField:  types.StringValue("test"),
				BoolField:    types.BoolValue(true),
				Int64Field:   types.Int64Value(42),
				Float64Field: types.Float64Value(3.14),
				ListField:    types.ListValueMust(types.StringType, []attr.Value{types.StringValue("item")}),
				SetField:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("item")}),
				MapField:     types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				ObjectField:  types.ObjectValueMust(map[string]attr.Type{"field1": types.StringType}, map[string]attr.Value{"field1": types.StringValue("value")}),
			},
			expectError: false,
		},
		{
			name: "mixed fields - some unknown, some known",
			input: &TestModel{
				StringField:  types.StringUnknown(),
				BoolField:    types.BoolValue(true),
				Int64Field:   types.Int64Unknown(),
				Float64Field: types.Float64Value(2.71),
				ListField:    types.ListNull(types.StringType),
				SetField:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("item")}),
				MapField:     types.MapUnknown(types.StringType),
				ObjectField:  types.ObjectValueMust(map[string]attr.Type{"field1": types.StringType}, map[string]attr.Value{"field1": types.StringValue("value")}),
			},
			expected: &TestModel{
				StringField:  types.StringNull(),
				BoolField:    types.BoolValue(true),
				Int64Field:   types.Int64Null(),
				Float64Field: types.Float64Value(2.71),
				ListField:    types.ListNull(types.StringType),
				SetField:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("item")}),
				MapField:     types.MapNull(types.StringType),
				ObjectField:  types.ObjectValueMust(map[string]attr.Type{"field1": types.StringType}, map[string]attr.Value{"field1": types.StringValue("value")}),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetModelFieldsToNull(ctx, tt.input)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Compare each field
			if diff := cmp.Diff(tt.input.StringField, tt.expected.StringField); diff != "" {
				t.Errorf("StringField mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(tt.input.BoolField, tt.expected.BoolField); diff != "" {
				t.Errorf("BoolField mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(tt.input.Int64Field, tt.expected.Int64Field); diff != "" {
				t.Errorf("Int64Field mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(tt.input.Float64Field, tt.expected.Float64Field); diff != "" {
				t.Errorf("Float64Field mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(tt.input.ListField, tt.expected.ListField); diff != "" {
				t.Errorf("ListField mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(tt.input.SetField, tt.expected.SetField); diff != "" {
				t.Errorf("SetField mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(tt.input.MapField, tt.expected.MapField); diff != "" {
				t.Errorf("MapField mismatch (-got +want):\n%s", diff)
			}
			if diff := cmp.Diff(tt.input.ObjectField, tt.expected.ObjectField); diff != "" {
				t.Errorf("ObjectField mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestSetModelFieldsToNull_Errors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     any
		wantError string
	}{
		{
			name:      "nil model",
			input:     nil,
			wantError: "model cannot be nil",
		},
		{
			name:      "non-pointer",
			input:     struct{}{},
			wantError: "model must be a pointer",
		},
		{
			name:      "pointer to non-struct",
			input:     func() *string { s := "test"; return &s }(),
			wantError: "model must point to a struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetModelFieldsToNull(ctx, tt.input)
			if err == nil {
				t.Fatal("expected error but got nil")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("expected error containing %q, got %q", tt.wantError, err.Error())
			}
		})
	}
}

func TestSetModelFieldsToNull_ComplexStructures(t *testing.T) {
	ctx := context.Background()

	// Test nested objects
	t.Run("object with unknown fields inside known object", func(t *testing.T) {
		type NestedModel struct {
			NestedObject types.Object `tfsdk:"nested_object"`
		}

		input := &NestedModel{
			NestedObject: types.ObjectValueMust(
				map[string]attr.Type{
					"field1": types.StringType,
					"field2": types.Int64Type,
				},
				map[string]attr.Value{
					"field1": types.StringUnknown(),
					"field2": types.Int64Value(42),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the object was modified
		attrs := input.NestedObject.Attributes()
		if !attrs["field1"].IsNull() {
			t.Error("field1 should be null after processing unknown field in nested object")
		}
		if attrs["field2"].IsNull() {
			t.Error("field2 should remain non-null")
		}
	})

	// Test list with unknown elements
	t.Run("list with unknown and null elements", func(t *testing.T) {
		type ListModel struct {
			MyList types.List `tfsdk:"my_list"`
		}

		input := &ListModel{
			MyList: types.ListValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("known"),
					types.StringUnknown(),
					types.StringNull(),
					types.StringValue("another_known"),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		elements := input.MyList.Elements()
		if len(elements) != 4 {
			t.Fatalf("expected 4 elements, got %d", len(elements))
		}

		// Check that unknown was converted to null
		if !elements[1].IsNull() {
			t.Error("element at index 1 (was unknown) should be null")
		}
		// Check that null remained null
		if !elements[2].IsNull() {
			t.Error("element at index 2 (was null) should remain null")
		}
		// Check known values remain unchanged
		if elements[0].IsNull() || elements[3].IsNull() {
			t.Error("known elements should not be null")
		}
	})

	// Test list of objects with unknown fields
	t.Run("list of objects with unknown fields", func(t *testing.T) {
		type ListOfObjectsModel struct {
			Objects types.List `tfsdk:"objects"`
		}

		objectType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name": types.StringType,
				"age":  types.Int64Type,
			},
		}

		input := &ListOfObjectsModel{
			Objects: types.ListValueMust(
				objectType,
				[]attr.Value{
					types.ObjectValueMust(
						objectType.AttrTypes,
						map[string]attr.Value{
							"name": types.StringValue("Alice"),
							"age":  types.Int64Unknown(),
						},
					),
					types.ObjectValueMust(
						objectType.AttrTypes,
						map[string]attr.Value{
							"name": types.StringUnknown(),
							"age":  types.Int64Value(30),
						},
					),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		elements := input.Objects.Elements()
		if len(elements) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(elements))
		}

		// Check first object - age should be null
		obj1 := elements[0].(types.Object)
		if !obj1.Attributes()["age"].IsNull() {
			t.Error("first object's age field should be null")
		}
		if obj1.Attributes()["name"].IsNull() {
			t.Error("first object's name field should not be null")
		}

		// Check second object - name should be null
		obj2 := elements[1].(types.Object)
		if !obj2.Attributes()["name"].IsNull() {
			t.Error("second object's name field should be null")
		}
		if obj2.Attributes()["age"].IsNull() {
			t.Error("second object's age field should not be null")
		}
	})

	// Test deeply nested objects
	t.Run("deeply nested objects", func(t *testing.T) {
		type DeepModel struct {
			Level1 types.Object `tfsdk:"level1"`
		}

		level3Type := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"deep_field": types.StringType,
			},
		}

		level2Type := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"level3": level3Type,
			},
		}

		level1Type := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"level2": level2Type,
			},
		}

		input := &DeepModel{
			Level1: types.ObjectValueMust(
				level1Type.AttrTypes,
				map[string]attr.Value{
					"level2": types.ObjectValueMust(
						level2Type.AttrTypes,
						map[string]attr.Value{
							"level3": types.ObjectValueMust(
								level3Type.AttrTypes,
								map[string]attr.Value{
									"deep_field": types.StringUnknown(),
								},
							),
						},
					),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Navigate to the deep field
		level2 := input.Level1.Attributes()["level2"].(types.Object)
		level3 := level2.Attributes()["level3"].(types.Object)
		deepField := level3.Attributes()["deep_field"]

		if !deepField.IsNull() {
			t.Error("deep_field should be null after processing")
		}
	})

	// Test list of lists (nested lists)
	t.Run("list of lists with unknown elements", func(t *testing.T) {
		type NestedListModel struct {
			OuterList types.List `tfsdk:"outer_list"`
		}

		innerListType := types.ListType{ElemType: types.StringType}

		input := &NestedListModel{
			OuterList: types.ListValueMust(
				innerListType,
				[]attr.Value{
					types.ListValueMust(
						types.StringType,
						[]attr.Value{
							types.StringValue("a"),
							types.StringUnknown(),
						},
					),
					types.ListValueMust(
						types.StringType,
						[]attr.Value{
							types.StringUnknown(),
							types.StringValue("b"),
						},
					),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		outerElements := input.OuterList.Elements()

		// Check first inner list
		innerList1 := outerElements[0].(types.List)
		innerElements1 := innerList1.Elements()
		if !innerElements1[1].IsNull() {
			t.Error("second element of first inner list should be null")
		}

		// Check second inner list
		innerList2 := outerElements[1].(types.List)
		innerElements2 := innerList2.Elements()
		if !innerElements2[0].IsNull() {
			t.Error("first element of second inner list should be null")
		}
	})

	// Test map with object values containing unknown fields
	t.Run("map with object values containing unknown fields", func(t *testing.T) {
		type MapModel struct {
			MyMap types.Map `tfsdk:"my_map"`
		}

		objectType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"field1": types.StringType,
				"field2": types.BoolType,
			},
		}

		input := &MapModel{
			MyMap: types.MapValueMust(
				objectType,
				map[string]attr.Value{
					"key1": types.ObjectValueMust(
						objectType.AttrTypes,
						map[string]attr.Value{
							"field1": types.StringValue("known"),
							"field2": types.BoolUnknown(),
						},
					),
					"key2": types.ObjectValueMust(
						objectType.AttrTypes,
						map[string]attr.Value{
							"field1": types.StringUnknown(),
							"field2": types.BoolValue(true),
						},
					),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		elements := input.MyMap.Elements()

		// Check key1 object
		obj1 := elements["key1"].(types.Object)
		if !obj1.Attributes()["field2"].IsNull() {
			t.Error("key1 object's field2 should be null")
		}
		if obj1.Attributes()["field1"].IsNull() {
			t.Error("key1 object's field1 should not be null")
		}

		// Check key2 object
		obj2 := elements["key2"].(types.Object)
		if !obj2.Attributes()["field1"].IsNull() {
			t.Error("key2 object's field1 should be null")
		}
		if obj2.Attributes()["field2"].IsNull() {
			t.Error("key2 object's field2 should not be null")
		}
	})

	// Test set with unknown elements
	t.Run("set with unknown elements", func(t *testing.T) {
		type SetModel struct {
			MySet types.Set `tfsdk:"my_set"`
		}

		input := &SetModel{
			MySet: types.SetValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("known"),
					types.StringUnknown(),
					types.StringNull(),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		elements := input.MySet.Elements()

		// Count null elements (should have at least 2: the original null and the converted unknown)
		nullCount := 0
		for _, elem := range elements {
			if elem.IsNull() {
				nullCount++
			}
		}

		if nullCount < 2 {
			t.Errorf("expected at least 2 null elements, got %d", nullCount)
		}
	})

	// Test set of objects with unknown fields
	t.Run("set of objects with unknown fields", func(t *testing.T) {
		type SetOfObjectsModel struct {
			Objects types.Set `tfsdk:"objects"`
		}

		objectType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":   types.StringType,
				"name": types.StringType,
			},
		}

		input := &SetOfObjectsModel{
			Objects: types.SetValueMust(
				objectType,
				[]attr.Value{
					types.ObjectValueMust(
						objectType.AttrTypes,
						map[string]attr.Value{
							"id":   types.StringValue("1"),
							"name": types.StringUnknown(),
						},
					),
					types.ObjectValueMust(
						objectType.AttrTypes,
						map[string]attr.Value{
							"id":   types.StringUnknown(),
							"name": types.StringValue("Test"),
						},
					),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		elements := input.Objects.Elements()
		if len(elements) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(elements))
		}

		// Check that unknown fields within objects were converted to null
		for _, elem := range elements {
			obj := elem.(types.Object)
			attrs := obj.Attributes()

			// At least one field in each object should be null (the unknown one)
			if !attrs["name"].IsNull() && !attrs["id"].IsNull() {
				t.Error("expected at least one field to be null in each object")
			}
		}
	})

	// Test map with list values containing objects
	t.Run("map with list values containing objects with unknown fields", func(t *testing.T) {
		type ComplexMapModel struct {
			MyMap types.Map `tfsdk:"my_map"`
		}

		objectType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"prop": types.StringType,
			},
		}
		listOfObjectsType := types.ListType{ElemType: objectType}

		input := &ComplexMapModel{
			MyMap: types.MapValueMust(
				listOfObjectsType,
				map[string]attr.Value{
					"key1": types.ListValueMust(
						objectType,
						[]attr.Value{
							types.ObjectValueMust(
								objectType.AttrTypes,
								map[string]attr.Value{
									"prop": types.StringUnknown(),
								},
							),
						},
					),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		elements := input.MyMap.Elements()
		list := elements["key1"].(types.List)
		listElements := list.Elements()
		obj := listElements[0].(types.Object)

		if !obj.Attributes()["prop"].IsNull() {
			t.Error("prop field should be null after processing")
		}
	})

	// Test top-level null object (should remain null)
	t.Run("top-level null object", func(t *testing.T) {
		type NullObjectModel struct {
			MyObject types.Object `tfsdk:"my_object"`
		}

		attrTypes := map[string]attr.Type{"field": types.StringType}
		input := &NullObjectModel{
			MyObject: types.ObjectNull(attrTypes),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !input.MyObject.IsNull() {
			t.Error("top-level null object should remain null")
		}
	})

	// Test top-level unknown list (should be converted to null)
	t.Run("top-level unknown list", func(t *testing.T) {
		type UnknownListModel struct {
			MyList types.List `tfsdk:"my_list"`
		}

		input := &UnknownListModel{
			MyList: types.ListUnknown(types.StringType),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !input.MyList.IsNull() {
			t.Error("top-level unknown list should be converted to null")
		}
		if input.MyList.IsUnknown() {
			t.Error("top-level list should no longer be unknown")
		}
	})

	// Test empty list (should remain unchanged)
	t.Run("empty list", func(t *testing.T) {
		type EmptyListModel struct {
			MyList types.List `tfsdk:"my_list"`
		}

		input := &EmptyListModel{
			MyList: types.ListValueMust(types.StringType, []attr.Value{}),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if input.MyList.IsNull() {
			t.Error("empty list should not become null")
		}
		if len(input.MyList.Elements()) != 0 {
			t.Error("list should remain empty")
		}
	})

	// Test object with all null fields
	t.Run("object with all null fields", func(t *testing.T) {
		type AllNullFieldsModel struct {
			MyObject types.Object `tfsdk:"my_object"`
		}

		attrTypes := map[string]attr.Type{
			"field1": types.StringType,
			"field2": types.Int64Type,
		}

		input := &AllNullFieldsModel{
			MyObject: types.ObjectValueMust(
				attrTypes,
				map[string]attr.Value{
					"field1": types.StringNull(),
					"field2": types.Int64Null(),
				},
			),
		}

		err := SetModelFieldsToNull(ctx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		attrs := input.MyObject.Attributes()
		if !attrs["field1"].IsNull() || !attrs["field2"].IsNull() {
			t.Error("all fields should remain null")
		}
	})
}

func TestShouldWait(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		setEnv   bool
		expected bool
	}{
		{
			name:     "env not set - should wait",
			setEnv:   false,
			expected: true,
		},
		{
			name:     "env set to empty string - should wait",
			envValue: "",
			setEnv:   true,
			expected: true,
		},
		{
			name:     "env set to 'true' - should wait",
			envValue: "true",
			setEnv:   true,
			expected: true,
		},
		{
			name:     "env set to 'TRUE' - should wait (case insensitive)",
			envValue: "TRUE",
			setEnv:   true,
			expected: true,
		},
		{
			name:     "env set to 'True' - should wait (case insensitive)",
			envValue: "True",
			setEnv:   true,
			expected: true,
		},
		{
			name:     "env set to 'false' - should not wait",
			envValue: "false",
			setEnv:   true,
			expected: false,
		},
		{
			name:     "env set to 'FALSE' - should not wait",
			envValue: "FALSE",
			setEnv:   true,
			expected: false,
		},
		{
			name:     "env set to '0' - should not wait",
			envValue: "0",
			setEnv:   true,
			expected: false,
		},
		{
			name:     "env set to 'no' - should not wait",
			envValue: "no",
			setEnv:   true,
			expected: false,
		},
		{
			name:     "env set to random value - should not wait",
			envValue: "random",
			setEnv:   true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env value
			originalValue, wasSet := os.LookupEnv("STACKIT_TF_WAIT_FOR_READY")
			defer func() {
				if wasSet {
					_ = os.Setenv("STACKIT_TF_WAIT_FOR_READY", originalValue)
				} else {
					_ = os.Unsetenv("STACKIT_TF_WAIT_FOR_READY")
				}
			}()

			// Set up test environment
			if tt.setEnv {
				_ = os.Setenv("STACKIT_TF_WAIT_FOR_READY", tt.envValue)
			} else {
				_ = os.Unsetenv("STACKIT_TF_WAIT_FOR_READY")
			}

			// Test
			result := ShouldWait()
			if result != tt.expected {
				t.Errorf("ShouldWait() = %v, want %v", result, tt.expected)
			}
		})
	}
}
