package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *iaas.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.IaaSCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.IaaSCustomEndpoint))
	}

	apiClient, err := iaas.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

func MapLabels(ctx context.Context, responseLabels *map[string]interface{}, currentLabels types.Map) (basetypes.MapValue, error) { //nolint:gocritic // Linter wants to have a non-pointer type for the map, but this would mean a nil check has to be done before every usage of this func.
	labelsTF, diags := types.MapValueFrom(ctx, types.StringType, map[string]interface{}{})
	if diags.HasError() {
		return labelsTF, fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
	}

	if responseLabels != nil && len(*responseLabels) != 0 {
		var diags diag.Diagnostics
		labelsTF, diags = types.MapValueFrom(ctx, types.StringType, *responseLabels)
		if diags.HasError() {
			return labelsTF, fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else if currentLabels.IsNull() {
		labelsTF = types.MapNull(types.StringType)
	}

	return labelsTF, nil
}
