package features

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

var AvailableExperiments []string = []string{"iam"}

// Check if an experiment is valid.
func ValidExperiment(experiment string, diags *diag.Diagnostics) bool {
	validExperiment := slices.ContainsFunc(AvailableExperiments, func(e string) bool {
		return strings.EqualFold(e, experiment)
	})
	if !validExperiment {
		diags.AddError("Invalid Experiment", fmt.Sprintf("The Experiment %s is invalid. This is most likely a bug in the STACKIT Provider. Please open an issue. Available Experiments: %v", experiment, AvailableExperiments))
	}

	return validExperiment
}

// Check if an experiment is enabled.
func CheckExperimentEnabled(ctx context.Context, data *core.ProviderData, experiment, resourceType string, diags *diag.Diagnostics) {
	if !ValidExperiment(experiment, diags) {
		return
	}
	experimentActive := slices.ContainsFunc(data.Experiments, func(e string) bool {
		return strings.EqualFold(e, experiment)
	})

	if experimentActive {
		warnTitle := fmt.Sprintf("%s is part of the %s experiment.", resourceType, experiment)
		warnContent := fmt.Sprintf("This resource is part of the %s experiment and is likely going to undergo significant changes or be removed in the future. Use it at your own discretion.", experiment)
		tflog.Warn(ctx, fmt.Sprintf("%s | %s", warnTitle, warnContent))
		diags.AddWarning(warnTitle, warnContent)
		return
	}
	errTitle := fmt.Sprintf("%s is part of the %s experiment, which is currently disabled by default", resourceType, experiment)
	errContent := fmt.Sprintf(`Enable the %s experiment by adding it into your provider block.`, experiment)
	tflog.Error(ctx, fmt.Sprintf("%s | %s", errTitle, errContent))
	diags.AddError(errTitle, errContent)
}

func AddExperimentDescription(description, experiment string) string {
	// Callout block: https://developer.hashicorp.com/terraform/registry/providers/docs#callouts
	return fmt.Sprintf("%s\n\n~> %s%s%s",
		description,
		"This resource is part of the ",
		experiment,
		" experiment and is likely going to undergo significant changes or be removed in the future. Use it at your own discretion.",
	)
}
