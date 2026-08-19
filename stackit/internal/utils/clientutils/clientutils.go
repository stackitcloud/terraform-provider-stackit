package clientutils

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

func defaultConfigOptions(providerData *core.ProviderData, customEndpoint string) []config.ConfigurationOption {
	options := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
		config.WithMiddleware(responseLoggingMiddleware),
	}
	if customEndpoint != "" {
		options = append(options, config.WithEndpoint(customEndpoint))
	}
	return options
}

func responseLoggingMiddleware(next http.RoundTripper) http.RoundTripper {
	return responseLoggingRoundTripper{next: next}
}

type responseLoggingRoundTripper struct {
	next http.RoundTripper
}

func (rt responseLoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	resp, err := rt.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	traceId := resp.Header.Get("x-trace-id")
	tflog.Info(ctx, "response data", map[string]any{
		"x-trace-id": traceId,
	})
	return resp, err
}

type DefaultClientFactory struct {
}

func (f *DefaultClientFactory) NewServiceEnablementV2Client(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) serviceenablementV2.DefaultAPI {
	apiClientConfigOptions := defaultConfigOptions(providerData, providerData.ServiceEnablementCustomEndpoint)
	apiClient, err := serviceenablementV2.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient.DefaultAPI
}

func (f *DefaultClientFactory) NewIaaSV2Client(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) iaasV2.DefaultAPI {
	apiClientConfigOptions := defaultConfigOptions(providerData, providerData.IaaSCustomEndpoint)
	apiClient, err := iaasV2.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient.DefaultAPI
}
