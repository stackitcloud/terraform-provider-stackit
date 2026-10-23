package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"

	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"

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
