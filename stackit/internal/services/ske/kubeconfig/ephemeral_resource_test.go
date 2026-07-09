package ske

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
		description  string
		expiration   *int64
		mockResponse *ske.Kubeconfig
		mockError    error
		expectError  bool
	}{
		{
			description: "success without expiration",
			expiration:  nil,
			mockResponse: &ske.Kubeconfig{
				Kubeconfig:           &[]string{kubeconfig}[0],
				ExpirationTimestamp:  &expirationTime,
				AdditionalProperties: make(map[string]any),
			},
			expectError: false,
		},
		{
			description: "success with expiration",
			expiration:  &[]int64{3600}[0],
			mockResponse: &ske.Kubeconfig{
				Kubeconfig:           &[]string{kubeconfig}[0],
				ExpirationTimestamp:  &expirationTime,
				AdditionalProperties: make(map[string]any),
			},
			expectError: false,
		},
		{
			description: "api error",
			mockError:   fmt.Errorf("api error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			mockResp := tt.mockResponse
			mockErr := tt.mockError
			createKubeconfigFn := func(_ ske.ApiCreateKubeconfigRequest) (*ske.Kubeconfig, error) {
				return mockResp, mockErr
			}
			client := &ske.DefaultAPIServiceMock{
				CreateKubeconfigExecuteMock: &createKubeconfigFn,
			}

			resp, err := getKubeconfig(context.Background(), client, projectId, region, clusterName, tt.expiration)

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
