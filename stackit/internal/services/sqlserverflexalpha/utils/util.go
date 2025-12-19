package utils

import (
	"context"
	"fmt"

	sqlserverflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
)

func ConfigureClient(
	ctx context.Context,
	providerData *core.ProviderData,
	diags *diag.Diagnostics,
) *sqlserverflex.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.SQLServerFlexCustomEndpoint != "" {
		apiClientConfigOptions = append(
			apiClientConfigOptions,
			config.WithEndpoint(providerData.SQLServerFlexCustomEndpoint),
		)
	} else {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithRegion(providerData.GetRegion()))
	}
	apiClient, err := sqlserverflex.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(
			ctx,
			diags,
			"Error configuring API client",
			fmt.Sprintf(
				"Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration",
				err,
			),
		)
		return nil
	}

	return apiClient
}
