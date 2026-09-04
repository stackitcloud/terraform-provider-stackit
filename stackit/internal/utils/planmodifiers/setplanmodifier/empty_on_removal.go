package setplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// emptyOnRemovalModifier implements the plan modifier.
type emptyOnRemovalModifier struct{}

func (m emptyOnRemovalModifier) Description(_ context.Context) string {
	return "Plans an empty set when the attribute is removed from the configuration, so that previously applied values are cleared remotely."
}

func (m emptyOnRemovalModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifySet implements the plan modification logic.
//
// For Optional+Computed set attributes, Terraform's default behavior on update is to keep the
// prior state value when the attribute is removed (null) from the configuration. Depending on
// whether other changes exist in the plan, the value is either marked as unknown
// "(known after apply)" or silently carried over from the prior state. In both cases the
// attribute is never planned for removal, which prevents users from clearing previously
// configured values via Terraform.
//
// This modifier instead plans an empty set when the attribute is removed from the
// configuration, producing a diff that clears the remote values on apply.
func (m emptyOnRemovalModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) { // nolint:gocritic // function signature required by Terraform
	// Do nothing on resource creation; there is no prior value to clear.
	if req.State.Raw.IsNull() {
		return
	}

	// Only act when the attribute was removed from the configuration.
	if !req.ConfigValue.IsNull() {
		return
	}

	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	// Do nothing when there is no prior value to clear; the empty plan would be a no-op diff.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	empty := types.SetValueMust(req.StateValue.ElementType(ctx), []attr.Value{})

	// Do nothing when the plan already proposes an empty set.
	if !req.PlanValue.IsUnknown() && req.PlanValue.Equal(empty) {
		return
	}

	resp.PlanValue = empty
}

// EmptyOnRemoval returns a plan modifier that plans an empty set when the attribute is
// removed from the configuration, so that previously applied values are cleared remotely.
func EmptyOnRemoval() planmodifier.Set {
	return emptyOnRemovalModifier{}
}
