package utils

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	sdkClients "github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	testUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	testVersion        = "1.2.3"
	testCustomEndpoint = "https://authorization-custom-endpoint.api.stackit.cloud"
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
		expected *authorization.APIClient
	}{
		{
			name: "default endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version: testVersion,
				},
			},
			expected: func() *authorization.APIClient {
				apiClient, err := authorization.NewAPIClient(
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
					AuthorizationCustomEndpoint: testCustomEndpoint,
				},
			},
			expected: func() *authorization.APIClient {
				apiClient, err := authorization.NewAPIClient(
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

func TestTypeConverter(t *testing.T) {
	tests := []struct {
		name        string
		input       authorization.MembersResponse
		expected    *authorization.ListMembersResponse
		expectError bool
	}{
		{
			name: "success - all fields populated",
			input: authorization.MembersResponse{
				Members: &[]authorization.Member{
					{
						Role:    testUtils.Ptr("editor"),
						Subject: testUtils.Ptr("foo.bar@stackit.cloud"),
					},
				},
				ResourceId:   testUtils.Ptr("project-123"),
				ResourceType: testUtils.Ptr("project"),
			},
			expected: &authorization.ListMembersResponse{
				Members: &[]authorization.Member{
					{
						Role:    testUtils.Ptr("editor"),
						Subject: testUtils.Ptr("foo.bar@stackit.cloud"),
					},
				},
				ResourceId:   testUtils.Ptr("project-123"),
				ResourceType: testUtils.Ptr("project"),
			},
			expectError: false,
		},
		{
			name:        "success - completely empty input",
			input:       authorization.MembersResponse{},
			expected:    &authorization.ListMembersResponse{},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := TypeConverter[authorization.ListMembersResponse](tc.input)

			if (err != nil) != tc.expectError {
				t.Fatalf("unexpected error: got error=%v, expectError=%v", err, tc.expectError)
			}

			if !tc.expectError && !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("\nUnexpected result:\nactual:   %+v\nexpected: %+v", actual, tc.expected)
			}
		})
	}
}
