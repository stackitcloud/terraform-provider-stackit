package clientutils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	iaasV2 "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	serviceenablementV2 "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

type ClientFactory interface {
	// methods are having the API versions in them here so we can still mix & match API versions just as we need

	// Service enablement
	NewServiceEnablementV2Client(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) serviceenablementV2.DefaultAPI

	NewIaaSV2Client(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) iaasV2.DefaultAPI
}

type DefaultClientFactory struct {
}

func (f *DefaultClientFactory) NewServiceEnablementV2Client(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) serviceenablementV2.DefaultAPI {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.ServiceEnablementCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.ServiceEnablementCustomEndpoint))
	}
	apiClient, err := serviceenablementV2.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient.DefaultAPI
}

func (f *DefaultClientFactory) NewIaaSV2Client(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) iaasV2.DefaultAPI {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.IaaSCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.IaaSCustomEndpoint))
	}
	apiClient, err := iaasV2.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient.DefaultAPI
}
