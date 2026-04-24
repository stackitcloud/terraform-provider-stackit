package utils

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	sdkClients "github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	testVersion        = "1.2.3"
	testCustomEndpoint = "https://serviceaccount-custom-endpoint.api.stackit.cloud"
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
		expected *serviceaccount.APIClient
	}{
		{
			name: "default endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version: testVersion,
				},
			},
			expected: func() *serviceaccount.APIClient {
				apiClient, err := serviceaccount.NewAPIClient(
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
					Version:                      testVersion,
					ServiceAccountCustomEndpoint: testCustomEndpoint,
				},
			},
			expected: func() *serviceaccount.APIClient {
				apiClient, err := serviceaccount.NewAPIClient(
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

func TestParseNameFromEmail(t *testing.T) {
	testCases := []struct {
		email       string
		expected    string
		shouldError bool
	}{
		// Standard SA domain (Positive: 7 to 10 random characters)
		{"foo-vshp191@sa.stackit.cloud", "foo", false},           // 7 chars
		{"bar-8565oq12@sa.stackit.cloud", "bar", false},          // 8 chars
		{"foo-bar-acfj2s123@sa.stackit.cloud", "foo-bar", false}, // 9 chars
		{"baz-abcdefghij@sa.stackit.cloud", "baz", false},        // 10 chars

		// Standard SA domain (Negative: 6 and 11 random characters)
		{"foo-vshp19@sa.stackit.cloud", "", true},      // 6 chars (Too short)
		{"bar-8565oq12345@sa.stackit.cloud", "", true}, // 11 chars (Too long)

		// SKE SA domain (Positive: 7 to 10 random characters)
		{"foo-qnmbwo1@ske.sa.stackit.cloud", "foo", false},           // 7 chars
		{"bar-qnmbwo12@ske.sa.stackit.cloud", "bar", false},          // 8 chars
		{"foo-bar-qnmbwo123@ske.sa.stackit.cloud", "foo-bar", false}, // 9 chars
		{"baz-abcdefghij@ske.sa.stackit.cloud", "baz", false},        // 10 chars

		// SKE SA domain (Negative: 6 and 11 random characters)
		{"foo-qnmbwo@ske.sa.stackit.cloud", "", true},      // 6 chars (Too short)
		{"bar-qnmbwo12345@ske.sa.stackit.cloud", "", true}, // 11 chars (Too long)

		// Invalid cases (Formatting & Unknown Domains)
		{"invalid-email@sa.stackit.cloud", "", true},
		{"missingcode-@sa.stackit.cloud", "", true},
		{"nohyphen8565oq1@sa.stackit.cloud", "", true},
		{"eu01-qnmbwo1@unknown.stackit.cloud", "", true},
		{"eu01-qnmbwo1@ske.stackit.com", "", true}, // Missing .sa. and ends in .com
		{"someotherformat@sa.stackit.cloud", "", true},
		{"invalid-format@ske.sa.stackit.cloud", "", true}, // SKE domain but missing the character suffix completely
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			name, err := ParseNameFromEmail(tc.email)
			if tc.shouldError {
				if err == nil {
					t.Errorf("expected an error for email: %s, but got none", tc.email)
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error for email: %s, but got: %v", tc.email, err)
				}
				if name != tc.expected {
					t.Errorf("expected name: %s, got: %s for email: %s", tc.expected, name, tc.email)
				}
			}
		})
	}
}
