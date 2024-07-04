package conversion

import (
	"context"
	"reflect"
	"testing"

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
			got, err := FromTerraformStringMapToInterfaceMap(tt.args.ctx, tt.args.m)
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
