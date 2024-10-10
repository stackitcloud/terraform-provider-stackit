package conversion

import (
	"context"
	"reflect"
	"testing"

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
		want    map[string]interface{}
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
			want: map[string]interface{}{
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
			want:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "nil",
			args: args{
				ctx: context.Background(),
				m:   types.MapNull(types.StringType),
			},
			want:    map[string]interface{}{},
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
