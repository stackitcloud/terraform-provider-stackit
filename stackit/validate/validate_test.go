package validate

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUUID(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"cae27bba-c43d-498a-861e-d11d241c4ff8",
			true,
		},
		{
			"too short",
			"a-b-c-d",
			false,
		},
		{
			"Empty",
			"",
			false,
		},
		{
			"not UUID",
			"www-541-%",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			UUID().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestIP(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok IP4",
			"111.222.111.222",
			true,
		},
		{
			"ok IP6",
			"2001:0db8:85a3:08d3::0370:7344",
			true,
		},
		{
			"too short",
			"0.1.2",
			false,
		},
		{
			"Empty",
			"",
			false,
		},
		{
			"Not an IP",
			"for-sure-not-an-IP",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			IP().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestNoSeparator(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"ABCD",
			true,
		},
		{
			"ok-2",
			"#$%&/()=.;-",
			true,
		},
		{
			"Empty",
			"",
			true,
		},
		{
			"not ok",
			"ab,",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			NoSeparator().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestMinorVersionNumber(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"1.20",
			true,
		},
		{
			"ok-2",
			"1.3",
			true,
		},
		{
			"ok-3",
			"10.1",
			true,
		},
		{
			"Empty",
			"",
			false,
		},
		{
			"not ok",
			"afssfdfs",
			false,
		},
		{
			"not ok-major-version",
			"1",
			false,
		},
		{
			"not ok-patch-version",
			"1.20.1",
			false,
		},
		{
			"not ok-version",
			"v1.20.1",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			MinorVersionNumber().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}
