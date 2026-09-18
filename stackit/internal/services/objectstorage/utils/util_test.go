package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
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

	var roundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
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
					Version:      testVersion,
					RoundTripper: roundTripper,
				},
			},
			expected: func() *objectstorage.APIClient {
				apiClient, err := objectstorage.NewAPIClient(
					utils.UserAgentConfigOption(testVersion),
					config.WithCustomAuth(&RetryTransport{
						Base:        roundTripper,
						MaxRetries:  3,
						BaseBackoff: 10 * time.Second,
						MaxJitter:   500 * time.Millisecond,
					}),
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
					RoundTripper:                roundTripper,
					ObjectStorageCustomEndpoint: testCustomEndpoint,
				},
			},
			expected: func() *objectstorage.APIClient {
				apiClient, err := objectstorage.NewAPIClient(
					utils.UserAgentConfigOption(testVersion),
					config.WithEndpoint(testCustomEndpoint),
					config.WithCustomAuth(&RetryTransport{
						Base:        roundTripper,
						MaxRetries:  3,
						BaseBackoff: 10 * time.Second,
						MaxJitter:   500 * time.Millisecond,
					}),
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

func TestClientRetry(t *testing.T) {
	ctx := context.Background()
	diags := diag.Diagnostics{}

	testProjectId := uuid.New().String()
	const testRegion = "eu01"
	const testBucketName = "karl-otto"

	attempts := 0

	// Create mock server returning HTTP 429 on first & second call, HTTP 200 on final retry
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		if r.URL.Path != fmt.Sprintf("/v2/project/%s/regions/%s/bucket/%s", testProjectId, testRegion, testBucketName) {
			t.Fatalf("invalid endpoint called")
		}

		// first request: HTTP 429 *with* Retry-After header
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, err := w.Write([]byte(`{"error": "rate_limit_exceeded"}`))
			if err != nil {
				t.Fatalf("error writing response: %v", err)
			}
			return
		}

		// second request: HTTP 429 *without* Retry-After header (we expect base backoff to be used now)
		if attempts == 2 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, err := w.Write([]byte(`{"error": "rate_limit_exceeded"}`))
			if err != nil {
				t.Fatalf("error writing response: %v", err)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
  			"bucket": {
				"name": "bucket-1",
				"objectLockEnabled": false,
				"region": "eu01",
				"urlPathStyle": "https://object.storage.eu01.onstackit.cloud/bucket-1",
				"urlVirtualHostedStyle": "https://bucket-1.object.storage.eu01.onstackit.cloud"
			},
			"project": "` + testProjectId + `"}`))
		if err != nil {
			t.Fatalf("error writing response: %v", err)
		}
	}))
	defer server.Close()

	client := ConfigureClient(ctx, &core.ProviderData{
		ObjectStorageCustomEndpoint: server.URL,
	}, &diags)
	if diags.HasError() {
		t.Fatalf("error configuring client: %v", diags)
	}

	_, err := client.DefaultAPI.GetBucket(ctx, testProjectId, testRegion, testBucketName).Execute()
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestEnableProject(t *testing.T) {
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
			synctest.Test(t, func(t *testing.T) {
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
		})
	}
}

// A 409 from a concurrent enable call must be retried instead of failing the apply.
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

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
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
