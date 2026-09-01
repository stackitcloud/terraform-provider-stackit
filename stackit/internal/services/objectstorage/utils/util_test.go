package utils

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	sdkClients "github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	testVersion        = "1.2.3"
	testCustomEndpoint = "https://objectstorage-custom-endpoint.api.stackit.cloud"
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
		expected *objectstorage.APIClient
	}{
		{
			name: "default endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version: testVersion,
				},
			},
			expected: func() *objectstorage.APIClient {
				apiClient, err := objectstorage.NewAPIClient(
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
					ObjectStorageCustomEndpoint: testCustomEndpoint,
				},
			},
			expected: func() *objectstorage.APIClient {
				apiClient, err := objectstorage.NewAPIClient(
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

func TestEnableProject(t *testing.T) {
	// EnableProject retries, and the mock returns a plain error rather than an
	// *oapierror.GenericOpenAPIError - RetryRequest only filters by status code
	// when it can type-assert the error, so the failing case uses up every
	// attempt. Without shrinking the delay this test would sleep for seconds.
	oldDelay := enableProjectRetryDelay
	enableProjectRetryDelay = time.Millisecond
	defer func() { enableProjectRetryDelay = oldDelay }()

	tests := []struct {
		description string
		enableFails bool
		isValid     bool
	}{
		{
			"default_values",
			false,
			true,
		},
		{
			"error_response",
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &objectstorage.DefaultAPIServiceMock{
				EnableServiceExecuteMock: new(func(_ objectstorage.ApiEnableServiceRequest) (*objectstorage.ProjectStatus, error) {
					if tt.enableFails {
						return nil, fmt.Errorf("create project failed")
					}

					return &objectstorage.ProjectStatus{}, nil
				}),
			}

			err := EnableProject(context.Background(), "pid", "eu01", client)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
		})
	}
}

// Two object storage resources created in the same apply enable the project concurrently.
// The API answers the losing call with 409 project.create_conflict; EnableProject must retry
// instead of failing the apply.
func TestEnableProjectRetriesOnConflict(t *testing.T) {
	tests := []struct {
		description  string
		conflicts    int
		isValid      bool
		wantAttempts int
	}{
		{"succeeds immediately", 0, true, 1},
		{"one conflict, then success", 1, true, 2},
		{"conflicts until the attempts are used up", enableProjectAttempts, false, enableProjectAttempts},
	}

	old := enableProjectRetryDelay
	enableProjectRetryDelay = time.Millisecond
	defer func() { enableProjectRetryDelay = old }()

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			attempts := 0
			client := &objectstorage.DefaultAPIServiceMock{
				EnableServiceExecuteMock: new(func(_ objectstorage.ApiEnableServiceRequest) (*objectstorage.ProjectStatus, error) {
					attempts++
					if attempts <= tt.conflicts {
						return nil, &oapierror.GenericOpenAPIError{StatusCode: http.StatusConflict}
					}
					return &objectstorage.ProjectStatus{}, nil
				}),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := EnableProject(ctx, "pid", "eu01", client)
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if !tt.isValid && err == nil {
				t.Fatal("Should have failed")
			}
			if attempts != tt.wantAttempts {
				t.Fatalf("Expected %d attempts, got %d", tt.wantAttempts, attempts)
			}
		})
	}
}

// A non-conflict error must not be retried.
func TestEnableProjectDoesNotRetryOtherErrors(t *testing.T) {
	attempts := 0
	client := &objectstorage.DefaultAPIServiceMock{
		EnableServiceExecuteMock: new(func(_ objectstorage.ApiEnableServiceRequest) (*objectstorage.ProjectStatus, error) {
			attempts++
			return nil, &oapierror.GenericOpenAPIError{StatusCode: http.StatusForbidden}
		}),
	}

	if err := EnableProject(context.Background(), "pid", "eu01", client); err == nil {
		t.Fatal("Should have failed")
	}
	if attempts != 1 {
		t.Fatalf("Expected a single attempt, got %d", attempts)
	}
}
