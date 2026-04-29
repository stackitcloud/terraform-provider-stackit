package conversion

import (
	"context"
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

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

func TestToJSONMapUpdatePayload(t *testing.T) {
	tests := []struct {
		description   string
		currentLabels types.Map
		desiredLabels types.Map
		expected      map[string]any
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
			map[string]any{
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
			map[string]any{
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
			map[string]any{
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
			map[string]any{
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
			map[string]any{
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
			map[string]any{
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
			map[string]any{
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
			map[string]any{
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

func TestParseProviderData(t *testing.T) {
	type args struct {
		providerData any
	}
	type want struct {
		ok           bool
		providerData core.ProviderData
	}
	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "provider has not been configured",
			args: args{
				providerData: nil,
			},
			want: want{
				ok: false,
			},
			wantErr: false,
		},
		{
			name: "invalid provider data",
			args: args{
				providerData: struct{}{},
			},
			want: want{
				ok: false,
			},
			wantErr: true,
		},
		{
			name: "valid provider data 1",
			args: args{
				providerData: core.ProviderData{},
			},
			want: want{
				ok:           true,
				providerData: core.ProviderData{},
			},
			wantErr: false,
		},
		{
			name: "valid provider data 2",
			args: args{
				providerData: core.ProviderData{
					DefaultRegion:          "eu02",
					RabbitMQCustomEndpoint: "https://rabbitmq-custom-endpoint.api.stackit.cloud",
					Version:                "1.2.3",
				},
			},
			want: want{
				ok: true,
				providerData: core.ProviderData{
					DefaultRegion:          "eu02",
					RabbitMQCustomEndpoint: "https://rabbitmq-custom-endpoint.api.stackit.cloud",
					Version:                "1.2.3",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			diags := diag.Diagnostics{}

			actual, ok := ParseProviderData(ctx, tt.args.providerData, &diags)
			if diags.HasError() != tt.wantErr {
				t.Errorf("ConfigureClient() error = %v, want %v", diags.HasError(), tt.wantErr)
			}
			if ok != tt.want.ok {
				t.Errorf("ParseProviderData() got = %v, want %v", ok, tt.want.ok)
			}
			if !reflect.DeepEqual(actual, tt.want.providerData) {
				t.Errorf("ParseProviderData() got = %v, want %v", actual, tt.want)
			}
		})
	}
}

func TestParseEphemeralProviderData(t *testing.T) {
	var randomRoundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}
	type args struct {
		providerData any
	}
	type want struct {
		ok           bool
		providerData core.EphemeralProviderData
	}
	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "provider has not been configured",
			args: args{
				providerData: nil,
			},
			want: want{
				ok: false,
			},
			wantErr: false,
		},
		{
			name: "invalid provider data",
			args: args{
				providerData: struct{}{},
			},
			want: want{
				ok: false,
			},
			wantErr: true,
		},
		{
			name: "valid provider data 1",
			args: args{
				providerData: core.EphemeralProviderData{},
			},
			want: want{
				ok:           true,
				providerData: core.EphemeralProviderData{},
			},
			wantErr: false,
		},
		{
			name: "valid provider data 2",
			args: args{
				providerData: core.EphemeralProviderData{
					ProviderData: core.ProviderData{
						RoundTripper: randomRoundTripper,
					},
				},
			},
			want: want{
				ok: true,
				providerData: core.EphemeralProviderData{
					ProviderData: core.ProviderData{
						RoundTripper: randomRoundTripper,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			diags := diag.Diagnostics{}

			actual, ok := ParseEphemeralProviderData(ctx, tt.args.providerData, &diags)
			if diags.HasError() != tt.wantErr {
				t.Errorf("ConfigureClient() error = %v, want %v", diags.HasError(), tt.wantErr)
			}
			if ok != tt.want.ok {
				t.Errorf("ParseProviderData() got = %v, want %v", ok, tt.want.ok)
			}
			if !reflect.DeepEqual(actual, tt.want.providerData) {
				t.Errorf("ParseProviderData() got = %v, want %v", actual, tt.want)
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
