// Copyright (c) STACKIT

package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/config"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *postgresflex.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.PostgresFlexCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.PostgresFlexCustomEndpoint))
	} else {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithRegion(providerData.GetRegion()))
	}
	apiClient, err := postgresflex.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}
