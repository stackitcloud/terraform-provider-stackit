package postgresflexalpha

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type useStateForUnknownIfFlavorUnchangedModifier struct {
	Req resource.SchemaRequest
}

// UseStateForUnknownIfFlavorUnchanged returns a plan modifier similar to UseStateForUnknown
// if the RAM and CPU values are not changed in the plan. Otherwise, the plan modifier does nothing.
func UseStateForUnknownIfFlavorUnchanged(req resource.SchemaRequest) planmodifier.String {
	return useStateForUnknownIfFlavorUnchangedModifier{
		Req: req,
	}
}

func (m useStateForUnknownIfFlavorUnchangedModifier) Description(context.Context) string {
	return "UseStateForUnknownIfFlavorUnchanged returns a plan modifier similar to UseStateForUnknown if the RAM and CPU values are not changed in the plan. Otherwise, the plan modifier does nothing."
}

func (m useStateForUnknownIfFlavorUnchangedModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForUnknownIfFlavorUnchangedModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) { // nolint:gocritic // function signature required by Terraform
	// Do nothing if there is no state value.
	if req.StateValue.IsNull() {
		return
	}

	// Do nothing if there is a known planned value.
	if !req.PlanValue.IsUnknown() {
		return
	}

	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	// The above checks are taken from the UseStateForUnknown plan modifier implementation
	// (https://github.com/hashicorp/terraform-plugin-framework/blob/main/resource/schema/stringplanmodifier/use_state_for_unknown.go#L38)

	var stateModel Model
	diags := req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateFlavor = &flavorModel{}
	if !(stateModel.Flavor.IsNull() || stateModel.Flavor.IsUnknown()) {
		diags = stateModel.Flavor.As(ctx, stateFlavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var planModel Model
	diags = req.Plan.Get(ctx, &planModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planFlavor = &flavorModel{}
	if !(planModel.Flavor.IsNull() || planModel.Flavor.IsUnknown()) {
		diags = planModel.Flavor.As(ctx, planFlavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if planFlavor.CPU == stateFlavor.CPU && planFlavor.RAM == stateFlavor.RAM {
		resp.PlanValue = req.StateValue
	}
}
