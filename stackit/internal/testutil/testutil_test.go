package testutil

import (
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
		{
			name:     "quoted string",
			variable: config.StringVariable(`instance =~ ".*"`),
			want:     `instance =~ ".*"`,
		},
		{
			name:     "line breaks",
			variable: config.StringVariable(`line \n breaks`),
			want:     `line \n breaks`,
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
