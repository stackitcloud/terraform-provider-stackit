package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	sdkClients "github.com/stackitcloud/stackit-sdk-go/core/clients"
)

func TestClientRetry(t *testing.T) {
	/* mock authentication by setting service account token env variable */
	os.Clearenv()
	err := os.Setenv(sdkClients.ServiceAccountToken, "mock-val")
	if err != nil {
		t.Errorf("error setting env variable: %v", err)
	}

	ctx := context.Background()

	testProjectId := uuid.New().String()
	const testRegion = "eu01"
	const testName = "test-ip-list"

	attempts := 0

	// Create mock server returning HTTP 429 on first & second call, HTTP 200 on final retry
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		if r.URL.Path != fmt.Sprintf("/v1alpha/projects/%s/regions/%s/ip-lists/%s", testProjectId, testRegion, testName) {
			t.Fatalf("invalid endpoint called: %s", r.URL.Path)
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
		_, err := w.Write([]byte(`{"name": "` + testName + `"}`))
		if err != nil {
			t.Fatalf("error writing response: %v", err)
		}
	}))
	defer server.Close()

	// short backoff for the test - the production values are set in ConfigureClient
	transport := &RetryTransport{MaxRetries: 5, BaseBackoff: 10 * time.Millisecond, MaxJitter: time.Millisecond}
	httpClient := &http.Client{Transport: transport}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1alpha/projects/%s/regions/%s/ip-lists/%s", server.URL, testProjectId, testRegion, testName), http.NoBody)
	if err != nil {
		t.Fatalf("error creating request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
