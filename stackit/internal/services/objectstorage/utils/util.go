package utils

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
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

	// RateLimitErrMsg is the user-facing error shown when the Object Storage Control Plane
	// rate limit (HTTP 429) is exceeded and all retry attempts are exhausted.
	RateLimitErrMsg = "API rate limit exceeded. The Object Storage Control Plane limits concurrent " +
		"requests; the provider retried but the limit was not resolved in time. " +
		"Consider re-running or reducing the number of parallel resources with -parallelism."
)

// RateLimitRetryConfig retries on HTTP 429 with exponential backoff and jitter.
//
// Tuned for the Object Storage Control Plane rate limit (≥60 req/min):
//   - 10s base start: at 60 req/min (1 req/s) this refills ~10 tokens, enough for all
//     goroutines competing at Terraform's default parallelism of 10 to succeed on first retry.
//   - ±25% jitter: spreads concurrent retries so they don't all hit the API at the same
//     instant after the rate-limit window resets (thundering herd). The 60 allowed requests
//     are typically consumed within a few seconds; without jitter all goroutines wake at the
//     exact same moment ~50s later and collide again.
//   - 60s cap: covers a full rate-limit window reset.
//   - 15 attempts: total budget ≈790s, covering up to ~790 concurrent goroutines and
//     well above the ~1000s run time for 1000 buckets at 60 req/min.
var RateLimitRetryConfig = utils.RetryConfig{
	Attempts: 15,
	Backoff: func(attempt int) time.Duration {
		base := min(10*time.Second*(1<<uint(attempt-1)), 60*time.Second)
		// Add ±25% jitter: spread = base/4, centered on base -Y [0.75×base, 1.25×base]
		jitter := time.Duration(rand.Int63n(int64(base/2))) - base/4 //nolint:gosec // non-crypto jitter
		return base + jitter
	},
	RetryStatusCodes: []int{http.StatusTooManyRequests},
}

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

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *objectstorage.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
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
