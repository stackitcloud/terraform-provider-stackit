package utils

import (
	"context"
	"fmt"

	modelexperiment "dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/modelexperiments/v1api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	INSTANCESTATE_CREATING    = "creating"
	INSTANCESTATE_ACTIVE      = "active"
	INSTANCESTATE_DELETING    = "deleting"
	INSTANCESTATE_PENDING     = "pending"
	INSTANCESTATE_UPDATING    = "updating"
	INSTANCESTATE_IMPAIRED    = "impaired"
	INSTANCESTATE_RECONCILING = "reconciling"

	TOKENSTATE_ACTIVE   = "active"
	TOKENSTATE_CREATING = "creating"
	TOKENSTATE_DELETING = "deleting"
	TOKENSTATE_INACTIVE = "inactive"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *modelexperiment.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.ModelExperimentsCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.ModelExperimentsCustomEndpoint))
	}
	apiClient, err := modelexperiment.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

func ConfigureServiceEnablementClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *serviceenablement.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.ServiceEnablementCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.ServiceEnablementCustomEndpoint))
	} else {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithRegion(providerData.GetRegion()))
	}
	apiClient, err := serviceenablement.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}
