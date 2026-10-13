package conversion

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	opensearch "github.com/stackitcloud/stackit-sdk-go/services/opensearch/v1api"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestFromTerraformStringMapToInterfaceMap(t *testing.T) {
	type args struct {
		ctx context.Context
		m   basetypes.MapValue
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr bool
	}{
		{
			name: "base",
			args: args{
				ctx: context.Background(),
				m: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key":  types.StringValue("value"),
					"key2": types.StringValue("value2"),
					"key3": types.StringValue("value3"),
				}),
			},
			want: map[string]any{
				"key":  "value",
				"key2": "value2",
				"key3": "value3",
			},
			wantErr: false,
		},
		{
			name: "empty",
			args: args{
				ctx: context.Background(),
				m:   types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			want:    map[string]any{},
			wantErr: false,
		},
		{
			name: "nil",
			args: args{
				ctx: context.Background(),
				m:   types.MapNull(types.StringType),
			},
			want:    map[string]any{},
			wantErr: false,
		},
		{
			name: "invalid type map (non-string)",
			args: args{
				ctx: context.Background(),
				m: types.MapValueMust(types.Int64Type, map[string]attr.Value{
					"key": types.Int64Value(1),
				}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToStringInterfaceMap(tt.args.ctx, tt.args.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromTerraformStringMapToInterfaceMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromTerraformStringMapToInterfaceMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToStringStringPointerMap(t *testing.T) {
	type args struct {
		m basetypes.MapValue
	}

	tests := []struct {
		name    string
		args    args
		want    map[string]*string
		wantErr bool
	}{
		{
			name: "base",
			args: args{
				m: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key":  types.StringValue("value"),
					"key2": types.StringValue("value2"),
					"key3": types.StringValue("value3"),
				}),
			},
			want: map[string]*string{
				"key":  new("value"),
				"key2": new("value2"),
				"key3": new("value3"),
			},
			wantErr: false,
		},
		{
			name: "empty",
			args: args{
				m: types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			want:    map[string]*string{},
			wantErr: false,
		},
		{
			name: "nil",
			args: args{
				m: types.MapNull(types.StringType),
			},
			want:    map[string]*string{},
			wantErr: false,
		},
		{
			name: "invalid type map (non-string)",
			args: args{
				m: types.MapValueMust(types.Int64Type, map[string]attr.Value{
					"key": types.Int64Value(1),
				}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToStringStringPointerMap(context.Background(), tt.args.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToStringStringPointerMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToStringStringPointerMap() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToOptStringPointerMap(t *testing.T) {
	tests := []struct {
		name    string
		tfMap   map[string]attr.Value
		want    *map[string]*string
		wantErr bool
	}{
		{
			name: "base",
			tfMap: map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			},
			want: &map[string]*string{
				"key":  new("value"),
				"key2": new("value2"),
			},
			wantErr: false,
		},
		{
			name:    "empty",
			tfMap:   map[string]attr.Value{},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "nil",
			tfMap:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name: "invalid type map (non-string)",
			tfMap: map[string]attr.Value{
				"key": types.Int64Value(1),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToOptStringPointerMap(tt.tfMap)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ToOptStringPointerMap() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Fatalf("ToOptStringPointerMap() mismatch: %s", diff)
			}
		})
	}
}

func TestToJSONMapUpdatePayload(t *testing.T) {
	tests := []struct {
		description   string
		currentLabels types.Map
		desiredLabels types.Map
		expected      map[string]any
		isValid       bool
	}{
		{
			description: "nothing_to_update",
			currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			expected: map[string]any{
				"key": "value",
			},
			isValid: true,
		},
		{
			description: "update_key_value",
			currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("updated_value"),
			}),
			expected: map[string]any{
				"key": "updated_value",
			},
			isValid: true,
		},
		{
			description: "remove_key",
			currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			expected: map[string]any{
				"key":  "value",
				"key2": nil,
			},
			isValid: true,
		},
		{
			description: "add_new_key",
			currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key": types.StringValue("value"),
			}),
			desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			expected: map[string]any{
				"key":  "value",
				"key2": "value2",
			},
			isValid: true,
		},
		{
			description: "empty_desired_map",
			currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{}),
			expected: map[string]any{
				"key":  nil,
				"key2": nil,
			},
			isValid: true,
		},
		{
			description: "nil_desired_map",
			currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			desiredLabels: types.MapNull(types.StringType),
			expected: map[string]any{
				"key":  nil,
				"key2": nil,
			},
			isValid: true,
		},
		{
			description:   "empty_current_map",
			currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{}),
			desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			expected: map[string]any{
				"key":  "value",
				"key2": "value2",
			},
			isValid: true,
		},
		{
			description:   "nil_current_map",
			currentLabels: types.MapNull(types.StringType),
			desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"key":  types.StringValue("value"),
				"key2": types.StringValue("value2"),
			}),
			expected: map[string]any{
				"key":  "value",
				"key2": "value2",
			},
			isValid: true,
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

func TestToLabelsMapPartialUpdatePayload(t *testing.T) {
	type args struct {
		currentLabels types.Map
		desiredLabels types.Map
	}
	tests := []struct {
		description string
		args        args
		expected    map[string]*string
		isValid     bool
	}{
		{
			description: "nothing_to_update",
			args: args{
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			expected: map[string]*string{
				"key": new("value"),
			},
			isValid: true,
		},
		{
			description: "update_key_value",
			args: args{
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("updated_value"),
				}),
			},
			expected: map[string]*string{
				"key": new("updated_value"),
			},
			isValid: true,
		},
		{
			description: "remove_key",
			args: args{
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key":  types.StringValue("value"),
					"key2": types.StringValue("value2"),
				}),
				desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			expected: map[string]*string{
				"key":  new("value"),
				"key2": nil,
			},
			isValid: true,
		},
		{
			description: "add_new_key",
			args: args{
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key":  types.StringValue("value"),
					"key2": types.StringValue("value2"),
				}),
			},
			expected: map[string]*string{
				"key":  new("value"),
				"key2": new("value2"),
			},
			isValid: true,
		},
		{
			description: "empty_desired_map",
			args: args{
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key":  types.StringValue("value"),
					"key2": types.StringValue("value2"),
				}),
				desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			expected: map[string]*string{
				"key":  nil,
				"key2": nil,
			},
			isValid: true,
		},
		{
			description: "nil_desired_map",
			args: args{
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key":  types.StringValue("value"),
					"key2": types.StringValue("value2"),
				}),
				desiredLabels: types.MapNull(types.StringType),
			},
			expected: map[string]*string{
				"key":  nil,
				"key2": nil,
			},
			isValid: true,
		},
		{
			description: "empty_current_map",
			args: args{
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{}),
				desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key":  types.StringValue("value"),
					"key2": types.StringValue("value2"),
				}),
			},
			expected: map[string]*string{
				"key":  new("value"),
				"key2": new("value2"),
			},
			isValid: true,
		},
		{
			description: "nil_current_map",
			args: args{
				currentLabels: types.MapNull(types.StringType),
				desiredLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key":  types.StringValue("value"),
					"key2": types.StringValue("value2"),
				}),
			},
			expected: map[string]*string{
				"key":  new("value"),
				"key2": new("value2"),
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := ToLabelsMapPartialUpdatePayload(context.Background(), tt.args.currentLabels, tt.args.desiredLabels)
			if (err != nil) == tt.isValid {
				t.Errorf("ToLabelsMapPartialUpdatePayload() error = %v, isValid %v", err, tt.isValid)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ToLabelsMapPartialUpdatePayload() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStringSetToSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      basetypes.SetValue
		want    []string
		wantErr bool
	}{
		{
			name: "unknown",
			in:   basetypes.NewSetUnknown(types.StringType),
			want: nil,
		},
		{
			name: "null",
			in:   basetypes.NewSetNull(types.StringType),
			want: nil,
		},
		{
			name:    "invalid type",
			in:      basetypes.NewSetValueMust(types.Int64Type, []attr.Value{types.Int64Value(123)}),
			wantErr: true,
		},
		{
			name: "some values, sorting",
			in: basetypes.NewSetValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("xyz"),
					types.StringValue("abc"),
				},
			),
			want: []string{
				"abc",
				"xyz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := StringSetToSlice(tt.in)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if d := cmp.Diff(got, tt.want); d != "" {
				t.Fatalf("no match, diff: %s", d)
			}
		})
	}
}

func TestStringListToSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      basetypes.ListValue
		want    []string
		wantErr bool
	}{
		{
			name: "unknown",
			in:   basetypes.NewListUnknown(types.StringType),
			want: nil,
		},
		{
			name: "null",
			in:   basetypes.NewListNull(types.StringType),
			want: nil,
		},
		{
			name: "empty list",
			in:   basetypes.NewListValueMust(types.StringType, []attr.Value{}),
			want: []string{},
		},
		{
			name:    "invalid type",
			in:      basetypes.NewListValueMust(types.Int64Type, []attr.Value{types.Int64Value(123)}),
			wantErr: true,
		},
		{
			name: "some values",
			in: basetypes.NewListValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("abc"),
					types.StringValue("xyz"),
				},
			),
			want: []string{
				"abc",
				"xyz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := StringListToSlice(tt.in)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if d := cmp.Diff(got, tt.want); d != "" {
				t.Fatalf("no match, diff: %s", d)
			}
		})
	}
}

func TestStringListToSet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		stringList []string
		want       types.Set
		wantErr    bool
	}{
		{
			name:       "nil list",
			stringList: nil,
			want:       basetypes.NewSetNull(types.StringType),
			wantErr:    false,
		},
		{
			name:       "empty list",
			stringList: []string{},
			want:       basetypes.NewSetValueMust(types.StringType, []attr.Value{}),
			wantErr:    false,
		},
		{
			name:       "valid list",
			stringList: []string{"value1", "value2"},
			want: basetypes.NewSetValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("value1"),
					types.StringValue("value2"),
				},
			),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			var diags diag.Diagnostics

			got := StringListToSet(ctx, tt.stringList, &diags)

			if diags.HasError() != tt.wantErr {
				t.Fatalf("expected error presence: %v, got diagnostics: %v", tt.wantErr, diags)
			}

			if !got.Equal(tt.want) {
				t.Fatalf("expected set: %v, got set: %v", tt.want, got)
			}
		})
	}
}

func TestTerraformStringSetToList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tfSet   basetypes.SetValue
		want    []string
		wantErr bool
	}{
		{
			name:    "unknown",
			tfSet:   basetypes.NewSetUnknown(types.StringType),
			want:    nil,
			wantErr: false,
		},
		{
			name:    "null",
			tfSet:   basetypes.NewSetNull(types.StringType),
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid type",
			tfSet:   basetypes.NewSetValueMust(types.Int64Type, []attr.Value{types.Int64Value(123)}),
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid string set",
			tfSet: basetypes.NewSetValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("value1"),
					types.StringValue("value2"),
				},
			),
			want: []string{
				"value1",
				"value2",
			},
			wantErr: false,
		},
		{
			name: "empty string set",
			tfSet: basetypes.NewSetValueMust(
				types.StringType,
				[]attr.Value{},
			),
			want:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			var diags diag.Diagnostics

			got := TerraformStringSetToList(ctx, tt.tfSet, &diags)

			if diags.HasError() != tt.wantErr {
				t.Fatalf("expected error presence: %v, got diagnostics: %v", tt.wantErr, diags)
			}

			if d := cmp.Diff(got, tt.want); d != "" {
				t.Fatalf("no match, diff: %s", d)
			}
		})
	}
}

func TestStringValueToPointer(t *testing.T) {
	type args struct {
		s basetypes.StringValue
	}
	tests := []struct {
		name string
		args args
		want *string
	}{
		{
			name: "default",
			args: args{
				s: basetypes.NewStringValue("abc"),
			},
			want: new("abc"),
		},
		{
			name: "unknown",
			args: args{
				s: basetypes.NewStringUnknown(),
			},
			want: nil,
		},
		{
			name: "null",
			args: args{
				s: basetypes.NewStringNull(),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringValueToPointer(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringValueToPointer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringValueToEnumPointer(t *testing.T) {
	type args struct {
		s basetypes.StringValue
	}
	type testCase[T interface{ ~string }] struct {
		name string
		args args
		want *T
	}
	tests := []testCase[opensearch.InstanceParametersJavaGarbageCollector]{
		{
			name: "default",
			args: args{
				s: basetypes.NewStringValue("UseG1GC"),
			},
			want: new(opensearch.INSTANCEPARAMETERSJAVAGARBAGECOLLECTOR_USE_G1_GC),
		},
		{
			name: "unknown",
			args: args{
				s: basetypes.NewStringUnknown(),
			},
			want: nil,
		},
		{
			name: "null",
			args: args{
				s: basetypes.NewStringNull(),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringValueToEnumPointer[opensearch.InstanceParametersJavaGarbageCollector](tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringValueToEnumPointer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringListToEnumSlice(t *testing.T) {
	type args struct {
		list basetypes.ListValue
	}
	type testCase[T interface{ ~string }] struct {
		name    string
		args    args
		want    []T
		wantErr bool
	}
	tests := []testCase[opensearch.InstanceParametersPluginsInner]{
		{
			name: "default",
			args: args{
				list: basetypes.NewListValueMust(types.StringType, []attr.Value{
					types.StringValue("repository-s3"),
					types.StringValue("repository-azure"),
				}),
			},
			want: []opensearch.InstanceParametersPluginsInner{
				opensearch.INSTANCEPARAMETERSPLUGINSINNER_REPOSITORY_S3,
				opensearch.INSTANCEPARAMETERSPLUGINSINNER_REPOSITORY_AZURE,
			},
			wantErr: false,
		},
		{
			name: "unknown",
			args: args{
				list: basetypes.NewListUnknown(types.StringType),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "null",
			args: args{
				list: basetypes.NewListNull(types.StringType),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "empty list",
			args: args{
				list: basetypes.NewListValueMust(types.StringType, []attr.Value{}),
			},
			want:    []opensearch.InstanceParametersPluginsInner{},
			wantErr: false,
		},
		{
			name: "invalid type",
			args: args{
				list: basetypes.NewListValueMust(types.Int64Type, []attr.Value{types.Int64Value(123)}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringListToEnumSlice[opensearch.InstanceParametersPluginsInner](tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("StringListToEnumSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringListToEnumSlice() got = %v, want %v", got, tt.want)
			}
		})
	}
}
