package utils

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	sdkClients "github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
)

const (
	testVersion        = "1.2.3"
	testCustomEndpoint = "https://sqlserverflex-custom-endpoint.api.stackit.cloud"
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
		expected *sqlserverflexalpha.APIClient
	}{
		{
			name: "default endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version: testVersion,
				},
			},
			expected: func() *sqlserverflexalpha.APIClient {
				apiClient, err := sqlserverflexalpha.NewAPIClient(
					config.WithRegion("eu01"),
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
					Version:                     testVersion,
					SQLServerFlexCustomEndpoint: testCustomEndpoint,
				},
			},
			expected: func() *sqlserverflexalpha.APIClient {
				apiClient, err := sqlserverflexalpha.NewAPIClient(
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
		t.Run(
			tt.name, func(t *testing.T) {
				ctx := context.Background()
				diags := diag.Diagnostics{}

				actual := ConfigureClient(ctx, tt.args.providerData, &diags)
				if diags.HasError() != tt.wantErr {
					t.Errorf("ConfigureClient() error = %v, want %v", diags.HasError(), tt.wantErr)
				}

				if !reflect.DeepEqual(actual, tt.expected) {
					t.Errorf("ConfigureClient() = %v, want %v", actual, tt.expected)
				}
			},
		)
	}
}
