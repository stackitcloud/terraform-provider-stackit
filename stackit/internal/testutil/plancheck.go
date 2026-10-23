package testutil

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// ExpectOnlyEphemeralOpen returns a plan check that allows no-op actions and
// OpenTofu "open" actions for the given ephemeral resource addresses.
func ExpectOnlyEphemeralOpen(configAddresses ...string) plancheck.PlanCheck {
	return expectOnlyEphemeralOpen{configAddresses: configAddresses}
}

type expectOnlyEphemeralOpen struct {
	configAddresses []string
}

func (c expectOnlyEphemeralOpen) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, resourceChange := range req.Plan.ResourceChanges {
		if resourceChange.Change == nil || resourceChange.Change.Actions.NoOp() {
			continue
		}

		actions := resourceChange.Change.Actions
		isSpecificResource := slices.Contains(c.configAddresses, resourceChange.Address)
		if isSpecificResource &&
			string(resourceChange.Mode) == "ephemeral" &&
			len(actions) == 1 && string(actions[0]) == "open" {
			continue
		}

		resp.Error = fmt.Errorf("unexpected planned action(s) %v for %s", actions, resourceChange.Address)
		return
	}

	for name, outputChange := range req.Plan.OutputChanges {
		if outputChange != nil && !outputChange.Actions.NoOp() {
			resp.Error = fmt.Errorf("unexpected planned action(s) %v for output %s", outputChange.Actions, name)
			return
		}
	}
}
