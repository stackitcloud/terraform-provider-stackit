package testutil

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
)

func TestConvertConfigVariable(t *testing.T) {
	tests := []struct {
		name     string
		variable config.Variable
		want     string
	}{
		{
			name:     "string",
			variable: config.StringVariable("test"),
			want:     "test",
		},
		{
			name:     "bool: true",
			variable: config.BoolVariable(true),
			want:     "true",
		},
		{
			name:     "bool: false",
			variable: config.BoolVariable(false),
			want:     "false",
		},
		{
			name:     "integer",
			variable: config.IntegerVariable(10),
			want:     "10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertConfigVariable(tt.variable); got != tt.want {
				t.Errorf("ConvertConfigVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToVariable(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want config.Variable
	}{
		{
			"string",
			"bar",
			config.StringVariable("bar"),
		},
		{
			"int",
			42,
			config.IntegerVariable(42),
		},
		{
			"float",
			3.141592654,
			config.FloatVariable(3.141592654),
		},
		{
			"bool",
			true,
			config.BoolVariable(true),
		},
		{
			"map",
			map[string]any{
				"foo": "bar",
			},
			config.MapVariable(map[string]config.Variable{
				"foo": config.StringVariable("bar"),
			},
			),
		},
		{
			"any slice",
			[]any{"foo", 42, 3.141, true},
			config.ListVariable(
				config.StringVariable("foo"),
				config.IntegerVariable(42),
				config.FloatVariable(3.141),
				config.BoolVariable(true),
			),
		},
		{
			"stringslice",
			[]string{"foo", "bar", "baz"},
			config.ListVariable(config.StringVariable("foo"), config.StringVariable("bar"), config.StringVariable("baz")),
		},
		{
			"nested",
			map[string]any{
				"simple": "bar",
				"map": map[string]any{
					"list": []any{"foo", 42, 42.0, true},
				},
			},
			config.MapVariable(map[string]config.Variable{
				"simple": config.StringVariable("bar"),
				"map": config.MapVariable(map[string]config.Variable{
					"list": config.ListVariable(
						config.StringVariable("foo"),
						config.IntegerVariable(42),
						config.FloatVariable(42.0),
						config.BoolVariable(true),
					),
				}),
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertToVariable(tt.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertToVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}
