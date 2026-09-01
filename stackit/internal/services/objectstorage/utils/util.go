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

const enableProjectAttempts = 4

// Overridden in tests to keep them fast.
var enableProjectRetryDelay = 2 * time.Second

// EnableProject enables object storage for the specified project. If the project is already enabled, nothing happens.
// Two resources created in the same apply call this concurrently and the API rejects the losing call with
// 409 project.create_conflict; retrying is safe, since enabling an already enabled project succeeds.
func EnableProject(ctx context.Context, projectId, region string, client objectstorage.DefaultAPI) error {
	config := utils.RetryConfig{
		Attempts:         enableProjectAttempts,
		Delay:            enableProjectRetryDelay,
		RetryStatusCodes: []int{http.StatusConflict},
	}
	if _, err := utils.RetryRequest(ctx, client.EnableService(ctx, projectId, region).Execute, config); err != nil {
		return fmt.Errorf("failed to create object storage project: %w", err)
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
