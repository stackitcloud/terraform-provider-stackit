package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/planmodifiers/stringplanmodifier"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *ske.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.SKECustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.SKECustomEndpoint))
	}
	apiClient, err := ske.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

func IsEmptyNetwork(network *ske.Network) bool {
	if !network.HasId() && !network.HasControlPlane() {
		return true
	}
	return false
}

func IsEmptyExtension(extension *ske.Extension) bool {
	if !extension.HasDns() && !extension.HasAcl() && !extension.HasObservability() {
		return true
	}
	return false
}

// Deprecated: HasOsVersionMinChanged
func HasOsVersionMinChanged(ctx context.Context, request planmodifier.StringRequest, response *stringplanmodifier.UseStateForUnknownFuncResponse) { // nolint:gocritic // function signature required by Terraform
	dependencyPath := request.Path.ParentPath().AtName("os_version_min")

	var minVersionPlan types.String
	diags := request.Plan.GetAttribute(ctx, dependencyPath, &minVersionPlan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var minVersionState types.String
	diags = request.State.GetAttribute(ctx, dependencyPath, &minVersionState)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if minVersionState == minVersionPlan {
		response.UseStateForUnknown = true
		return
	}
}
