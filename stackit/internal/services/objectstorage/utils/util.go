package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	enableProjectAttempts   = 4
	enableProjectRetryDelay = 2 * time.Second
)

// EnableProject enables object storage for the specified project. If the project is already enabled, nothing happens.
// Two resources created in the same apply call this concurrently and the API rejects the losing call with
// 409 project.create_conflict; retrying is safe, since enabling an already enabled project succeeds.
func EnableProject(ctx context.Context, projectId, region string, client objectstorage.DefaultAPI) error {
	retryConfig := utils.RetryConfig{
		Attempts:         enableProjectAttempts,
		Delay:            enableProjectRetryDelay,
		RetryStatusCodes: []int{http.StatusConflict},
	}
	if _, err := utils.RetryRequest(ctx, client.EnableService(ctx, projectId, region).Execute, retryConfig); err != nil {
		return fmt.Errorf("enable object storage project: %w", err)
	}
	return nil
}

// RetryTransport wraps an underlying RoundTripper to retry on HTTP 429 rate limit errors with jitter.
type RetryTransport struct {
	Base        http.RoundTripper
	MaxRetries  int
	BaseBackoff time.Duration
	MaxJitter   time.Duration
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	// Preserve request body for retries if present
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		err = req.Body.Close()
		if err != nil {
			return nil, err
		}
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.MaxRetries; attempt++ {
		// Re-hydrate the request body on each attempt
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err = base.RoundTrip(req)

		// If success or non-429 error, return immediately
		if err != nil || resp.StatusCode != http.StatusTooManyRequests {
			return resp, err
		}

		// Stop if max retries reached
		if attempt == t.MaxRetries {
			break
		}

		// Calculate base sleep duration (value of Retry-After Header, if header isn't present falls back to exponential backoff)
		wait := t.getWaitDuration(resp, attempt)

		// Always add random jitter regardless of Retry-After header presence. Else all resource / datasource
		// goroutines would try again in parallel after exactly the same interval.
		jitter := time.Duration(rand.Int64N(int64(t.MaxJitter))) //nolint:gosec // only used for jitter
		totalWait := wait + jitter

		// Drain and close response body before retrying to reuse TCP connections
		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			return nil, err
		}

		err = resp.Body.Close()
		if err != nil {
			return nil, err
		}

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(totalWait):
		}
	}

	return resp, err
}

func (t *RetryTransport) getWaitDuration(resp *http.Response, attempt int) time.Duration {
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		// Try parsing as integer seconds
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
		// Try parsing as HTTP-Date string
		if date, err := http.ParseTime(retryAfter); err == nil {
			if d := time.Until(date); d > 0 {
				return d
			}
		}
	}

	// Fallback to exponential backoff
	return t.BaseBackoff * (1 << attempt)
}

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *objectstorage.APIClient {
	// Add middleware to retry on HTTP 429 rate limits.
	// This solution is **not** intended to be copied to each and every service (!!).
	// This should be rolled out centrally instead, should be easily doable after
	// this refactoring: https://github.com/stackitcloud/terraform-provider-stackit/pull/1663
	retryRoundTripper := &RetryTransport{
		Base:        providerData.RoundTripper,
		MaxRetries:  3,
		BaseBackoff: 1 * time.Second,
		MaxJitter:   500 * time.Millisecond, // Always added to wait time
	}

	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(retryRoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}

	if providerData.ObjectStorageCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.ObjectStorageCustomEndpoint))
	}

	apiClient, err := objectstorage.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}
