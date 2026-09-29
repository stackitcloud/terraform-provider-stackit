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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	lbiplists "github.com/stackitcloud/stackit-sdk-go/services/lbiplists/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// RetryTransport wraps an underlying RoundTripper to retry on HTTP 429 rate limit errors with jitter.
// Temporary per-service solution (copied from objectstorage/utils) until retrying is rolled out centrally.
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

	// Preserve request body for retries if present without mutating the original request
	getBody := req.GetBody
	if getBody == nil && req.Body != nil && req.Body != http.NoBody {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		err = req.Body.Close()
		if err != nil {
			return nil, err
		}

		getBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.MaxRetries; attempt++ {
		reqClone := req.Clone(req.Context())
		if getBody != nil {
			var bodyErr error
			reqClone.Body, bodyErr = getBody()
			if bodyErr != nil {
				return nil, bodyErr
			}
		}

		resp, err = base.RoundTrip(reqClone)

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

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *lbiplists.APIClient {
	// Add middleware to retry on HTTP 429 rate limits (the upload endpoint is limited to 1 req/min).
	// This solution is copied from objectstorage, which should not be done, but the resource is pretty much unusable without this
	retryRoundTripper := &RetryTransport{
		Base:        providerData.RoundTripper,
		MaxRetries:  5,
		BaseBackoff: 60 * time.Second,
		MaxJitter:   500 * time.Millisecond, // Always added to wait time
	}

	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(retryRoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.LoadBalancerCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.LoadBalancerCustomEndpoint))
	}
	apiClient, err := lbiplists.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}
