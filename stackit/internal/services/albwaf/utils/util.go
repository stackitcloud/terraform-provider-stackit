package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *albWaf.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.AlbWafCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.AlbWafCustomEndpoint))
	}

	apiClient, err := albWaf.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

func WarnIfNameChanges(stateName, planName types.String, resourceLabel string, diags *diag.Diagnostics) {
	if utils.IsUndefined(stateName) {
		return
	}
	if planName.Equal(stateName) {
		return
	}
	diags.AddWarning(
		fmt.Sprintf("%s name change requires resource replacement", resourceLabel),
		fmt.Sprintf(
			"Changing the \"name\" attribute from %q to %q will destroy and recreate this resource. "+
				"If another resource references this %s by name, the replacement will fail. Remove or update that dependency before applying this change.",
			stateName.ValueString(), planName.ValueString(), resourceLabel,
		),
	)
}
