package utils

import (
	"context"
	"fmt"
	"reflect"
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
