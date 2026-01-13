package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/edge"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	DisplayNameMinimumChars = 4
	DisplayNameMaximumChars = 8
	DescriptionMaxLength    = 256
	TokenMinDuration        = 600
	TokenMaxDuration        = 15552000
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *edge.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.EdgeCloudCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.EdgeCloudCustomEndpoint))
	}
	apiClient, err := edge.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

func CheckExpiration(expiresAt types.String, recreateBefore types.Int64, currentTime time.Time) (bool, error) {
	if expiresAt.IsNull() {
		return true, nil
	}

	if expiresAt.IsUnknown() {
		return true, nil
	}

	expiresAtTime, err := time.Parse(time.RFC3339, expiresAt.ValueString())
	if err != nil {
		return false, fmt.Errorf("failed to convert expiresAt field to timestamp: %w", err)
	}

	if !recreateBefore.IsNull() {
		expiresAtTime = expiresAtTime.Add(-time.Duration(recreateBefore.ValueInt64()) * time.Second)
	}

	// The value is considered expired if the expiration time is not after the current time.
	// This correctly handles cases where the expiration is before or exactly at the current time.
	if !expiresAtTime.After(currentTime) {
		return true, nil
	}

	return false, nil
}
