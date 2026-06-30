package listplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// suppressNullEmptyListModifier implements the plan modifier.
type suppressNullEmptyListModifier struct{}

func (m suppressNullEmptyListModifier) Description(_ context.Context) string {
	return "Suppresses plan diffs where the configuration is null but the state is an empty list."
}

func (m suppressNullEmptyListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m suppressNullEmptyListModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) { // nolint:gocritic // function signature required by Terraform
	// If the user explicitly configured a value (even an empty list), let Terraform handle it.
	if !req.ConfigValue.IsNull() {
		return
	}

	// If the prior state is a known, empty list, copy it to the plan to suppress the diff.
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
	}
}

func SuppressNullEmptyList() planmodifier.List {
	return suppressNullEmptyListModifier{}
}
