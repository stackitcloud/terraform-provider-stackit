package postgresflex

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFlavorType_Equal(t1 *testing.T) {
	type fields struct {
		ObjectType basetypes.ObjectType
	}
	type args struct {
		o attr.Type
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := FlavorType{
				ObjectType: tt.fields.ObjectType,
			}
			if got := t.Equal(tt.args.o); got != tt.want {
				t1.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorType_String(t1 *testing.T) {
	type fields struct {
		ObjectType basetypes.ObjectType
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := FlavorType{
				ObjectType: tt.fields.ObjectType,
			}
			if got := t.String(); got != tt.want {
				t1.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorType_ValueFromObject(t1 *testing.T) {
	type fields struct {
		ObjectType basetypes.ObjectType
	}
	type args struct {
		in0 context.Context
		in  basetypes.ObjectValue
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   basetypes.ObjectValuable
		want1  diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := FlavorType{
				ObjectType: tt.fields.ObjectType,
			}
			got, got1 := t.ValueFromObject(tt.args.in0, tt.args.in)
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("ValueFromObject() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t1.Errorf("ValueFromObject() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestFlavorType_ValueFromTerraform(t1 *testing.T) {
	type fields struct {
		ObjectType basetypes.ObjectType
	}
	type args struct {
		ctx context.Context
		in  tftypes.Value
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    attr.Value
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := FlavorType{
				ObjectType: tt.fields.ObjectType,
			}
			got, err := t.ValueFromTerraform(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t1.Errorf("ValueFromTerraform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("ValueFromTerraform() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorType_ValueType(t1 *testing.T) {
	type fields struct {
		ObjectType basetypes.ObjectType
	}
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   attr.Value
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := FlavorType{
				ObjectType: tt.fields.ObjectType,
			}
			if got := t.ValueType(tt.args.in0); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("ValueType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorValue_AttributeTypes(t *testing.T) {
	type fields struct {
		Cpu         basetypes.Int64Value
		Description basetypes.StringValue
		Id          basetypes.StringValue
		Ram         basetypes.Int64Value
		state       attr.ValueState
	}
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := FlavorValue{
				Cpu:         tt.fields.Cpu,
				Description: tt.fields.Description,
				Id:          tt.fields.Id,
				Ram:         tt.fields.Ram,
				state:       tt.fields.state,
			}
			if got := v.AttributeTypes(tt.args.in0); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorValue_Equal(t *testing.T) {
	type fields struct {
		Cpu         basetypes.Int64Value
		Description basetypes.StringValue
		Id          basetypes.StringValue
		Ram         basetypes.Int64Value
		state       attr.ValueState
	}
	type args struct {
		o attr.Value
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := FlavorValue{
				Cpu:         tt.fields.Cpu,
				Description: tt.fields.Description,
				Id:          tt.fields.Id,
				Ram:         tt.fields.Ram,
				state:       tt.fields.state,
			}
			if got := v.Equal(tt.args.o); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorValue_IsNull(t *testing.T) {
	type fields struct {
		Cpu         basetypes.Int64Value
		Description basetypes.StringValue
		Id          basetypes.StringValue
		Ram         basetypes.Int64Value
		state       attr.ValueState
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := FlavorValue{
				Cpu:         tt.fields.Cpu,
				Description: tt.fields.Description,
				Id:          tt.fields.Id,
				Ram:         tt.fields.Ram,
				state:       tt.fields.state,
			}
			if got := v.IsNull(); got != tt.want {
				t.Errorf("IsNull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorValue_IsUnknown(t *testing.T) {
	type fields struct {
		Cpu         basetypes.Int64Value
		Description basetypes.StringValue
		Id          basetypes.StringValue
		Ram         basetypes.Int64Value
		state       attr.ValueState
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := FlavorValue{
				Cpu:         tt.fields.Cpu,
				Description: tt.fields.Description,
				Id:          tt.fields.Id,
				Ram:         tt.fields.Ram,
				state:       tt.fields.state,
			}
			if got := v.IsUnknown(); got != tt.want {
				t.Errorf("IsUnknown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorValue_String(t *testing.T) {
	type fields struct {
		Cpu         basetypes.Int64Value
		Description basetypes.StringValue
		Id          basetypes.StringValue
		Ram         basetypes.Int64Value
		state       attr.ValueState
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := FlavorValue{
				Cpu:         tt.fields.Cpu,
				Description: tt.fields.Description,
				Id:          tt.fields.Id,
				Ram:         tt.fields.Ram,
				state:       tt.fields.state,
			}
			if got := v.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorValue_ToObjectValue(t *testing.T) {
	type fields struct {
		Cpu         basetypes.Int64Value
		Description basetypes.StringValue
		Id          basetypes.StringValue
		Ram         basetypes.Int64Value
		state       attr.ValueState
	}
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   basetypes.ObjectValue
		want1  diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := FlavorValue{
				Cpu:         tt.fields.Cpu,
				Description: tt.fields.Description,
				Id:          tt.fields.Id,
				Ram:         tt.fields.Ram,
				state:       tt.fields.state,
			}
			got, got1 := v.ToObjectValue(tt.args.in0)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToObjectValue() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ToObjectValue() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestFlavorValue_ToTerraformValue(t *testing.T) {
	type fields struct {
		Cpu         basetypes.Int64Value
		Description basetypes.StringValue
		Id          basetypes.StringValue
		Ram         basetypes.Int64Value
		state       attr.ValueState
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    tftypes.Value
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := FlavorValue{
				Cpu:         tt.fields.Cpu,
				Description: tt.fields.Description,
				Id:          tt.fields.Id,
				Ram:         tt.fields.Ram,
				state:       tt.fields.state,
			}
			got, err := v.ToTerraformValue(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToTerraformValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToTerraformValue() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlavorValue_Type(t *testing.T) {
	type fields struct {
		Cpu         basetypes.Int64Value
		Description basetypes.StringValue
		Id          basetypes.StringValue
		Ram         basetypes.Int64Value
		state       attr.ValueState
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := FlavorValue{
				Cpu:         tt.fields.Cpu,
				Description: tt.fields.Description,
				Id:          tt.fields.Id,
				Ram:         tt.fields.Ram,
				state:       tt.fields.state,
			}
			if got := v.Type(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFlavorValue(t *testing.T) {
	type args struct {
		attributeTypes map[string]attr.Type
		attributes     map[string]attr.Value
	}
	tests := []struct {
		name  string
		args  args
		want  FlavorValue
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := NewFlavorValue(tt.args.attributeTypes, tt.args.attributes)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFlavorValue() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("NewFlavorValue() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestNewFlavorValueMust(t *testing.T) {
	type args struct {
		attributeTypes map[string]attr.Type
		attributes     map[string]attr.Value
	}
	tests := []struct {
		name string
		args args
		want FlavorValue
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFlavorValueMust(tt.args.attributeTypes, tt.args.attributes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFlavorValueMust() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFlavorValueNull(t *testing.T) {
	tests := []struct {
		name string
		want FlavorValue
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFlavorValueNull(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFlavorValueNull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFlavorValueUnknown(t *testing.T) {
	tests := []struct {
		name string
		want FlavorValue
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFlavorValueUnknown(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFlavorValueUnknown() = %v, want %v", got, tt.want)
			}
		})
	}
}
