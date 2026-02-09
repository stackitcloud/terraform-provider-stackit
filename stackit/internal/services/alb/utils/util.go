package utils

import (
	"context"
	"fmt"

	albSdk "github.com/stackitcloud/stackit-sdk-go/services/alb"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *albSdk.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.ALBCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.ALBCustomEndpoint))
	}
	apiClient, err := albSdk.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

type sdkEnums interface {
	albSdk.ListenerProtocol | albSdk.LoadBalancerErrorTypes | albSdk.NetworkRole
}

func ToStringList[T sdkEnums](in []T) []string {
	out := make([]string, len(in))
	for i, o := range in {
		out[i] = string(o)
	}
	return out
}
