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
