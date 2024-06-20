package features

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func BetaFeatureFlagEnabled(data *core.ProviderData) bool {
	// ProviderData should always be set, but we check just in case
	if data == nil {
		return false
	}
	return data.EnableBetaResources
}

// BetaResourcesEnabled returns whether this provider has BETA functionality enabled.
//
// In order of precedence, beta functionality can be managed by:
//   - Environment Variable `STACKIT_TF_ENABLE_BETA_RESOURCES` - `true` is enabled, `false` is disabled.
//   - Provider configuration feature flag `enable_beta` - `true` is enabled, `false` is disabled.
func BetaResourcesEnabled(ctx context.Context, data *core.ProviderData, diags *diag.Diagnostics) bool {
	value, ok := os.LookupEnv("STACKIT_TF_ENABLE_BETA_RESOURCES")
	if ok {
		if strings.EqualFold(value, "true") {
			return true
		}
		if strings.EqualFold(value, "false") {
			return false
		}
		warnDetails := fmt.Sprintf("The value of the environment variable that enables BETA functionality must be either 'true' or 'false', got %q. \nDefaulting to the provider feature flag.", value)
		core.LogAndAddWarning(ctx, diags, "Invalid value for STACKIT_TF_ENABLE_BETA_RESOURCES environment variable.", warnDetails)
	}
	// ProviderData should always be set, but we check just in case
	if data == nil {
		return false
	}
	return BetaFeatureFlagEnabled(data)
}

// CheckBetaResourcesEnabled is a helper function to log and add a warning or error if the BETA functionality is not enabled.
//
// Should be called in the Configure method of a BETA resource and you need to check for Errors in the diags after calling.
func CheckBetaResourcesEnabled(ctx context.Context, data *core.ProviderData, diags *diag.Diagnostics, resourceName string) {
	if !BetaResourcesEnabled(ctx, data, diags) {
		core.LogAndAddErrorBeta(ctx, diags, resourceName)
		return
	}
	core.LogAndAddWarningBeta(ctx, diags, resourceName)
}
