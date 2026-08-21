package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/runcommand/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	providerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, region string, diags *diag.Diagnostics) *v1api.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		config.WithRegion(region),
		providerUtils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.RunCommandCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.RunCommandCustomEndpoint))
	}
	apiClient, err := v1api.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}
