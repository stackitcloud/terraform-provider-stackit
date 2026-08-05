package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *sqlserverflex.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.SQLServerFlexCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.SQLServerFlexCustomEndpoint))
	}
	apiClient, err := sqlserverflex.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

var RetryConfig = utils.RetryConfig{
	Attempts: 5,
	Backoff: func(attempt int) time.Duration {
		// Wait for every attempt 5 seconds longer. 5s, 10s, 15s and so on
		return time.Duration(attempt*5) * time.Second
	},
	RetryStatusCodes: []int{
		http.StatusLocked,
		http.StatusTooEarly,
	},
}
