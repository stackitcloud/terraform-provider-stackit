package utils

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	sdkClients "github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	testVersion        = "1.2.3"
	testCustomEndpoint = "https://iaas-custom-endpoint.api.stackit.cloud"
)

func TestConfigureClient(t *testing.T) {
	/* mock authentication by setting service account token env variable */
	os.Clearenv()
	err := os.Setenv(sdkClients.ServiceAccountToken, "mock-val")
	if err != nil {
		t.Errorf("error setting env variable: %v", err)
	}

	type args struct {
		providerData *core.ProviderData
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		expected *iaas.APIClient
	}{
		{
			name: "default endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version: testVersion,
				},
			},
			expected: func() *iaas.APIClient {
				apiClient, err := iaas.NewAPIClient(
					utils.UserAgentConfigOption(testVersion),
				)
				if err != nil {
					t.Errorf("error configuring client: %v", err)
				}
				return apiClient
			}(),
			wantErr: false,
		},
		{
			name: "custom endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version:            testVersion,
					IaaSCustomEndpoint: testCustomEndpoint,
				},
			},
			expected: func() *iaas.APIClient {
				apiClient, err := iaas.NewAPIClient(
					utils.UserAgentConfigOption(testVersion),
					config.WithEndpoint(testCustomEndpoint),
				)
				if err != nil {
					t.Errorf("error configuring client: %v", err)
				}
				return apiClient
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			diags := diag.Diagnostics{}

			actual := ConfigureClient(ctx, tt.args.providerData, &diags)
			if diags.HasError() != tt.wantErr {
				t.Errorf("ConfigureClient() error = %v, want %v", diags.HasError(), tt.wantErr)
			}

			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("ConfigureClient() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestMapLabels(t *testing.T) {
	type args struct {
		responseLabels *map[string]interface{}
		currentLabels  types.Map
	}
	tests := []struct {
		name    string
		args    args
		want    basetypes.MapValue
		wantErr bool
	}{
		{
			name: "response labels is set",
			args: args{
				responseLabels: &map[string]interface{}{
					"foo1": "bar1",
					"foo2": "bar2",
				},
				currentLabels: types.MapUnknown(types.StringType),
			},
			wantErr: false,
			want: types.MapValueMust(types.StringType, map[string]attr.Value{
				"foo1": types.StringValue("bar1"),
				"foo2": types.StringValue("bar2"),
			}),
		},
		{
			name: "response labels is set but empty",
			args: args{
				responseLabels: &map[string]interface{}{},
				currentLabels:  types.MapUnknown(types.StringType),
			},
			wantErr: false,
			want:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
		},
		{
			name: "response labels is nil and model labels is nil",
			args: args{
				responseLabels: nil,
				currentLabels:  types.MapNull(types.StringType),
			},
			wantErr: false,
			want:    types.MapNull(types.StringType),
		},
		{
			name: "response labels is nil and model labels is set",
			args: args{
				responseLabels: nil,
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"foo1": types.StringValue("bar1"),
					"foo2": types.StringValue("bar2"),
				}),
			},
			wantErr: false,
			want:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
		},
		{
			name: "response labels is nil and model labels is set but empty",
			args: args{
				responseLabels: nil,
				currentLabels:  types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			wantErr: false,
			want:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := MapLabels(ctx, tt.args.responseLabels, tt.args.currentLabels)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapLabels() got = %v, want %v", got, tt.want)
			}
		})
	}
}
