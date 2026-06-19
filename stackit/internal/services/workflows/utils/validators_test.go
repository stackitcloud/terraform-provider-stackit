package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestURL(t *testing.T) {
	tests := []struct {
		description string
		input       string
		wantErr     bool
	}{
		{description: "http", input: "http://example.com"},
		{description: "https with path and query", input: "https://example.com/path?q=1"},
		{description: "https with userinfo", input: "https://user:pw@host.example.com"},
		{description: "https with ipv6 host", input: "https://[::1]:8080/foo"},

		{description: "empty", input: "", wantErr: true},
		{description: "missing scheme", input: "example.com/path", wantErr: true},
		{description: "relative path", input: "/path", wantErr: true},
		{description: "non http scheme", input: "ftp://example.com", wantErr: true},
		{description: "scheme only", input: "https://", wantErr: true},
		{description: "garbage", input: "::not a url", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			URL().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)
			if tt.wantErr && !r.Diagnostics.HasError() {
				t.Fatalf("URL(%q): expected error, got none", tt.input)
			}
			if !tt.wantErr && r.Diagnostics.HasError() {
				t.Fatalf("URL(%q): expected pass, got errors: %v", tt.input, r.Diagnostics.Errors())
			}
		})
	}
}

func TestAirflow3Version(t *testing.T) {
	tests := []struct {
		desc    string
		input   string
		wantErr bool
	}{
		{"airflow 3.1 ok", "workflows-3.0-airflow-3.1", false},
		{"airflow 3.0 ok", "workflows-3.0-airflow-3.0", false},
		{"future airflow 4 ok", "workflows-4.0-airflow-4.0", false},
		{"non-matching string is allowed (server validates exact version)", "workflows-x", false},
		{"empty is allowed (Required will catch it separately)", "", false},

		{"airflow 2.11 rejected", "workflows-2.3-airflow-2.11", true},
		{"airflow 2.10 rejected", "workflows-2.2-airflow-2.10", true},
		{"airflow 1.x rejected", "workflows-1.0-airflow-1.10", true},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			r := validator.StringResponse{}
			Airflow3Version().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)
			if tt.wantErr && !r.Diagnostics.HasError() {
				t.Fatalf("Airflow3Version(%q): expected error, got none", tt.input)
			}
			if !tt.wantErr && r.Diagnostics.HasError() {
				t.Fatalf("Airflow3Version(%q): expected pass, got errors: %v", tt.input, r.Diagnostics.Errors())
			}
		})
	}
}

func TestURLHTTPSOnly(t *testing.T) {
	tests := []struct {
		description string
		input       string
		wantErr     bool
	}{
		{description: "https ok", input: "https://example.com/.well-known/openid-configuration"},
		{description: "https with ipv6 host", input: "https://[::1]:8443"},

		{description: "http rejected", input: "http://example.com", wantErr: true},
		{description: "empty", input: "", wantErr: true},
		{description: "missing scheme", input: "example.com/x", wantErr: true},
		{description: "scheme only", input: "https://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			URLHTTPSOnly().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)
			if tt.wantErr && !r.Diagnostics.HasError() {
				t.Fatalf("URLHTTPSOnly(%q): expected error, got none", tt.input)
			}
			if !tt.wantErr && r.Diagnostics.HasError() {
				t.Fatalf("URLHTTPSOnly(%q): expected pass, got errors: %v", tt.input, r.Diagnostics.Errors())
			}
		})
	}
}
