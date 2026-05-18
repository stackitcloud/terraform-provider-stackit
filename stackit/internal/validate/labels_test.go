package validate

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLabelValidators(t *testing.T) {
	tests := []struct {
		description string
		input       map[string]attr.Value
		isValid     bool
	}{
		{
			"ok",
			map[string]attr.Value{
				"foo": types.StringValue("bar"),
			},
			true,
		},
		{
			"all valid characters",
			map[string]attr.Value{
				"abcdefghijklmnopqrstuvwxyz-_.0123456789": types.StringValue("abcdefghijklmnopqrstuvwxyz-_.0123456789"),
			},
			true,
		},
		{
			"invalid character in key",
			map[string]attr.Value{
				"foo!1": types.StringValue("bar"),
			},
			false,
		},
		{
			"invalid start in key",
			map[string]attr.Value{
				"_foo": types.StringValue("bar"),
			},
			false,
		},
		{
			"invalid end in key",
			map[string]attr.Value{
				"foo_": types.StringValue("bar"),
			},
			false,
		},
		{
			"invalid character in value",
			map[string]attr.Value{
				"foo": types.StringValue("bar!1"),
			},
			false,
		},
		{
			"invalid start in value",
			map[string]attr.Value{
				"foo": types.StringValue("_bar"),
			},
			false,
		},
		{
			"invalid end in value",
			map[string]attr.Value{
				"foo": types.StringValue("bar_"),
			},
			false,
		},
		{
			"Max key length",
			map[string]attr.Value{
				"123456789012345678901234567890123456789012345678901234567890123": types.StringValue("bar"),
			},
			true,
		},
		{
			"Min key length",
			map[string]attr.Value{
				"1": types.StringValue("bar"),
			},
			true,
		},
		{
			"Key to long",
			map[string]attr.Value{
				"1234567890123456789012345678901234567890123456789012345678901234": types.StringValue("bar"),
			},
			false,
		},
		{
			"Key to short",
			map[string]attr.Value{
				"": types.StringValue("bar"),
			},
			false,
		},
		{
			"Max value length",
			map[string]attr.Value{
				"foo": types.StringValue("123456789012345678901234567890123456789012345678901234567890123"),
			},
			true,
		},
		{
			"Empty value",
			map[string]attr.Value{
				"foo": types.StringValue(""),
			},
			true,
		},
		{
			"Value to long",
			map[string]attr.Value{
				"foo": types.StringValue("1234567890123456789012345678901234567890123456789012345678901234"),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.MapResponse{}

			value, _ := types.MapValue(types.StringType, tt.input)

			for _, LabelValidator := range LabelValidators() {
				LabelValidator.ValidateMap(context.Background(), validator.MapRequest{
					ConfigValue: value,
				}, &r)
			}

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}
