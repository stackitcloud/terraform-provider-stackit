package ske

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"
)

func TestGetKubeconfig(t *testing.T) {
	const (
		projectId   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
		clusterName = "cluster"
		region      = "eu01"
		kubeconfig  = "mock-kubeconfig"
	)
	expirationTime := time.Now().Add(time.Hour).Truncate(time.Second)

	tests := []struct {
		description    string
		expiration     *int64
		mockResponse   *ske.Kubeconfig
		mockStatusCode int
		expectError    bool
	}{
		{
			description: "success",
			expiration:  nil,
			mockResponse: &ske.Kubeconfig{
				Kubeconfig:           &[]string{kubeconfig}[0],
				ExpirationTimestamp:  &expirationTime,
				AdditionalProperties: make(map[string]any),
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			description: "success with expiration",
			expiration:  &[]int64{3600}[0],
			mockResponse: &ske.Kubeconfig{
				Kubeconfig:           &[]string{kubeconfig}[0],
				ExpirationTimestamp:  &expirationTime,
				AdditionalProperties: make(map[string]any),
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			description:    "api error",
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/v2/projects/%s/regions/%s/clusters/%s/kubeconfig", projectId, region, clusterName)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					_ = json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			cfg, err := ske.NewAPIClient(
				config.WithEndpoint(server.URL),
				config.WithoutAuthentication(),
			)
			if err != nil {
				t.Fatalf("Failed to create SKE client: %v", err)
			}

			resp, err := getKubeconfig(context.Background(), cfg, projectId, region, clusterName, tt.expiration)

			if (err != nil) != tt.expectError {
				t.Fatalf("getKubeconfig() error = %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError {
				if diff := cmp.Diff(resp, tt.mockResponse); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
