package utils

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	sdkClients "github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	testVersion        = "1.2.3"
	testCustomEndpoint = "https://ske-custom-endpoint.api.stackit.cloud"
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
		expected *ske.APIClient
	}{
		{
			name: "default endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version: testVersion,
				},
			},
			expected: func() *ske.APIClient {
				apiClient, err := ske.NewAPIClient(
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
					Version:           testVersion,
					SKECustomEndpoint: testCustomEndpoint,
				},
			},
			expected: func() *ske.APIClient {
				apiClient, err := ske.NewAPIClient(
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

func TestIsEmptyNetwork(t *testing.T) {
	tests := []struct {
		name  string
		input *ske.Network
		want  bool
	}{
		{
			name:  "nil network",
			input: nil,
			want:  true,
		},
		{
			name:  "empty",
			input: &ske.Network{},
			want:  true,
		},
		{
			name: "only AdditionalProperties are set",
			input: &ske.Network{
				AdditionalProperties: map[string]interface{}{
					"foo": "bar",
				},
			},
			want: true,
		},
		{
			name: "id set",
			input: &ske.Network{
				Id: new("network-id"),
			},
			want: false,
		},
		{
			name: "control plane set",
			input: &ske.Network{
				ControlPlane: &ske.V2ControlPlaneNetwork{},
			},
			want: false,
		},
		{
			name: "id and control plane set",
			input: &ske.Network{
				Id: new("network-id"),
				ControlPlane: &ske.V2ControlPlaneNetwork{
					AccessScope: ske.ACCESSSCOPE_SNA.Ptr(),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEmptyNetwork(tt.input); got != tt.want {
				t.Errorf("IsEmptyNetwork() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEmptyExtension(t *testing.T) {
	tests := []struct {
		name  string
		input *ske.Extension
		want  bool
	}{
		{
			name:  "nil extension",
			input: nil,
			want:  true,
		},
		{
			name:  "empty",
			input: &ske.Extension{},
			want:  true,
		},
		{
			name: "only AdditionalProperties are set",
			input: &ske.Extension{
				AdditionalProperties: map[string]interface{}{
					"foo": "bar",
				},
			},
			want: true,
		},
		{
			name: "acl set",
			input: &ske.Extension{
				Acl: ske.NewACL([]string{"1.1.1.0/24"}, true),
			},
			want: false,
		},
		{
			name: "observability set",
			input: &ske.Extension{
				Observability: ske.NewObservability(true, "instance-id"),
			},
			want: false,
		},
		{
			name: "dns set",
			input: &ske.Extension{
				Dns: ske.NewDNS(true),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEmptyExtension(tt.input); got != tt.want {
				t.Errorf("IsEmptyExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}
