package features

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

// BetaFeatureFlagEnabled returns whether this provider is running with the BETA feature flag (enable_beta) enabled.
func BetaFeatureFlagEnabled(data core.ProviderData) bool {
	return data.EnableBeta
}

// BetaResourcesEnabled returns whether this provider has BETA functionality enabled.
//
// In order of precedence, beta functionality can be managed by:
//   - Environment Variable `STACKIT_TERRAFORM_BETA_RESOURCES` - `true` is enabled, unset or any other value is disabled.
//   - Provider configuration feature flag `enable_beta` - `true` is enabled, `false` is disabled.
func BetaResourcesEnabled(data core.ProviderData) bool {
	value, ok := os.LookupEnv("STACKIT_TERRAFORM_BETA_RESOURCES")
	if !ok {
		return BetaFeatureFlagEnabled(data)
	}
	return strings.ToLower(value) == "true"
}

// CheckBetaResourcesEnabled is a helper function to log and add a warning or error if the BETA functionality is not enabled.
//
// Should be called in the Configure method of a BETA resource and you need to check for Errors in the diags after calling.
func CheckBetaResourcesEnabled(ctx context.Context, data core.ProviderData, diags *diag.Diagnostics) {
	if !BetaResourcesEnabled(data) {
		core.LogAndAddErrorBeta(ctx, diags, "DNS record set")
		return
	}
	core.LogAndAddWarningBeta(ctx, diags, "DNS record set")
}
