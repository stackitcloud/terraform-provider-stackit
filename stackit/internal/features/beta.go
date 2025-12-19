// Copyright (c) STACKIT

package features

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

// BetaResourcesEnabled returns whether this provider has beta functionality enabled.
//
// In order of precedence, beta functionality can be managed by:
//   - Environment Variable `STACKIT_TF_ENABLE_BETA_RESOURCES` - `true` is enabled, `false` is disabled.
//   - Provider configuration feature flag `enable_beta` - `true` is enabled, `false` is disabled.
func BetaResourcesEnabled(ctx context.Context, data *core.ProviderData, diags *diag.Diagnostics) bool {
	value, set := os.LookupEnv("STACKIT_TF_ENABLE_BETA_RESOURCES")
	if set {
		if strings.EqualFold(value, "true") {
			return true
		}
		if strings.EqualFold(value, "false") {
			return false
		}
		warnDetails := fmt.Sprintf(`The value of the environment variable that enables beta functionality must be either "true" or "false", got %q.
Defaulting to the provider feature flag.`, value)
		core.LogAndAddWarning(ctx, diags, "Invalid value for STACKIT_TF_ENABLE_BETA_RESOURCES environment variable.", warnDetails)
	}
	// ProviderData should always be set, but we check just in case
	if data == nil {
		return false
	}
	return data.EnableBetaResources
}

// CheckBetaResourcesEnabled is a helper function to log and add a warning or error if the beta functionality is not enabled.
//
// Should be called in the Configure method of a beta resource.
// Then, check for Errors in the diags using the diags.HasError() method.
func CheckBetaResourcesEnabled(ctx context.Context, data *core.ProviderData, diags *diag.Diagnostics, resourceName string, resourceType core.ResourceType) {
	if !BetaResourcesEnabled(ctx, data, diags) {
		core.LogAndAddErrorBeta(ctx, diags, resourceName, resourceType)
		return
	}
	core.LogAndAddWarningBeta(ctx, diags, resourceName, resourceType)
}

func AddBetaDescription(description string, resourceType core.ResourceType) string {
	// Callout block: https://developer.hashicorp.com/terraform/registry/providers/docs#callouts
	return fmt.Sprintf("%s\n\n~> %s %s",
		description,
		fmt.Sprintf("This %s is in beta and may be subject to breaking changes in the future. Use with caution.", resourceType),
		"See our [guide](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs/guides/opting_into_beta_resources) for how to opt-in to use beta resources.",
	)
}
