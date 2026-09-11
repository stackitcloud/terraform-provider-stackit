package utils

import (
	"context"
	"fmt"
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
)

// RateLimitRetryConfig retries on HTTP 429 with exponential backoff.
//
// The Object Storage Control Plane is rate-limited to 80 req/min (~1.33 req/s).
// Large states trigger this during the parallel refresh phase (terraform plan/apply)
// and during bulk creates/deletes in a single apply.
//
// Practical example — 300 buckets in a single state:
//   - Minimum time to process all requests at the rate limit: 300/80*60 = 225s (~3.75 min).
//   - The first ~80 requests succeed immediately; the remaining ~220 receive 429 and retry.
//   - With Terraform's default parallelism of 10, the retry waves clear roughly every 7.5s
//     (10 goroutines / 1.33 req/s), so most goroutines need only 2–3 attempts.
//   - Total retry budget of ~435s comfortably exceeds the 225s floor.
//
// Design rationale:
//   - Starting delay of 5s: at 1.33 req/s refill, 500ms returns less than 1 new token —
//     all goroutines would immediately fail again, burning attempts without progress.
//     5s refills ~6.7 tokens, enough for the majority of competing goroutines to succeed.
//   - Cap of 60s: covers a full fixed-window rate-limit reset so goroutines do not exhaust
//     their budget before the 1-minute window clears.
//   - 10 attempts: backoff schedule 5s+10s+20s+40s+(5×60s) = 435s total budget.
var RateLimitRetryConfig = utils.RetryConfig{
	Attempts: 10,
	Backoff: func(attempt int) time.Duration {
		// Exponential backoff: 5s, 10s, 20s, 40s, 60s (capped)
		return min(5*time.Second*(1<<uint(attempt-1)), 60*time.Second)
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
