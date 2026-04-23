package utils

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *serviceaccount.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.ServiceAccountCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.ServiceAccountCustomEndpoint))
	}
	apiClient, err := serviceaccount.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

// ParseNameFromEmail extracts the name component from a service account email address.
// The expected email format is `name-<random7to10characters>@sa.stackit.cloud`
// or `name-<random7to10characters>@ske.sa.stackit.cloud`.
func ParseNameFromEmail(email string) (string, error) {
	namePattern := `^([a-z][a-z0-9]*(?:-[a-z0-9]+)*)-\w{7,10}@(?:ske\.)?sa\.stackit\.cloud$`
	re := regexp.MustCompile(namePattern)
	match := re.FindStringSubmatch(email)

	// If a match is found, return the name component
	if len(match) > 1 {
		return match[1], nil
	}

	// If no match is found, return an error
	return "", fmt.Errorf("unable to parse name from email")
}
