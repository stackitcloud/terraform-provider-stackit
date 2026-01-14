package utils

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkClients "github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/edge"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	testVersion        = "1.2.3"
	testCustomEndpoint = "https://edge-custom-endpoint.api.stackit.cloud"
)

func TestConfigureClient(t *testing.T) {
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
		expected *edge.APIClient
	}{
		{
			name: "default endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version: testVersion,
				},
			},
			expected: func() *edge.APIClient {
				apiClient, err := edge.NewAPIClient(
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
					Version:                 testVersion,
					EdgeCloudCustomEndpoint: testCustomEndpoint,
				},
			},
			expected: func() *edge.APIClient {
				apiClient, err := edge.NewAPIClient(
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

func TestCheckExpiration(t *testing.T) {
	// Reference time for testing
	now := time.Date(2025, 10, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		expiresAt       types.String
		recreateBefore  types.Int64
		currentTime     time.Time
		expectedExpired bool
		expectedErr     bool
	}{
		{
			name:            "Is not expired",
			expiresAt:       types.StringValue(now.Add(1 * time.Hour).Format(time.RFC3339)),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: false,
			expectedErr:     false,
		},
		{
			name:            "Is expired",
			expiresAt:       types.StringValue(now.Add(-1 * time.Hour).Format(time.RFC3339)),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: true,
			expectedErr:     false,
		},
		{
			name:            "Expires at the exact current time",
			expiresAt:       types.StringValue(now.Format(time.RFC3339)),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: true, // Should be considered expired if the times are equal.
			expectedErr:     false,
		},
		{
			name:            "ExpiresAt is null",
			expiresAt:       types.StringNull(),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: true,
			expectedErr:     false,
		},
		{
			name:            "ExpiresAt is unknown",
			expiresAt:       types.StringUnknown(),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: true, // Should be treated as expired to force re-creation
			expectedErr:     false,
		},
		{
			name:            "ExpiresAt has invalid format",
			expiresAt:       types.StringValue("invalid-time-format"),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: false,
			expectedErr:     true,
		},
		{
			name:            "Is considered expired due to recreateBefore",
			expiresAt:       types.StringValue(now.Add(30 * time.Minute).Format(time.RFC3339)),
			recreateBefore:  types.Int64Value(3600),
			currentTime:     now,
			expectedExpired: true,
			expectedErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasExpired, err := CheckExpiration(tt.expiresAt, tt.recreateBefore, tt.currentTime)

			if (err != nil) != tt.expectedErr {
				t.Errorf("CheckExpiration() error = %v, wantErr %v", err, tt.expectedErr)
				return
			}
			if hasExpired != tt.expectedExpired {
				t.Errorf("CheckExpiration() = %v, want %v", hasExpired, tt.expectedExpired)
			}
		})
	}
}
