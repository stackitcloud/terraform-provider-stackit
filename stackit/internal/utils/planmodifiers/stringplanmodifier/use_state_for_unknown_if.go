package stringplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UseStateForUnknownFuncResponse struct {
	UseStateForUnknown bool
	Diagnostics        diag.Diagnostics
}

// UseStateForUnknownIfFunc is a conditional function used in UseStateForUnknownIf
type UseStateForUnknownIfFunc func(context.Context, planmodifier.StringRequest, *UseStateForUnknownFuncResponse)

type useStateForUnknownIf struct {
	ifFunc      UseStateForUnknownIfFunc
	description string
}

// UseStateForUnknownIf returns a plan modifier similar to UseStateForUnknown with a conditional
func UseStateForUnknownIf(f UseStateForUnknownIfFunc, description string) planmodifier.String {
	return useStateForUnknownIf{
		ifFunc:      f,
		description: description,
	}
}

func (m useStateForUnknownIf) Description(context.Context) string {
	return m.description
}

func (m useStateForUnknownIf) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForUnknownIf) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) { // nolint:gocritic // function signature required by Terraform
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
	// (https://github.com/hashicorp/terraform-plugin-framework/blob/44348af3923c82a93c64ae7dca906d9850ba956b/resource/schema/stringplanmodifier/use_state_for_unknown.go#L38)

	funcResponse := &UseStateForUnknownFuncResponse{}
	m.ifFunc(ctx, req, funcResponse)

	resp.Diagnostics.Append(funcResponse.Diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	if funcResponse.UseStateForUnknown {
		resp.PlanValue = req.StateValue
	}
}

// StringUnchanged sets UseStateForUnkown to true if the attribute's planned value matches the current state
func StringUnchanged(attributePath path.Path) UseStateForUnknownIfFunc { // nolint:gocritic // function signature required by Terraform
	return func(ctx context.Context, request planmodifier.StringRequest, response *UseStateForUnknownFuncResponse) {
		var attributePlan types.String
		diags := request.Plan.GetAttribute(ctx, attributePath, &attributePlan)
		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}

		var attributeState types.String
		diags = request.State.GetAttribute(ctx, attributePath, &attributeState)
		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}

		if attributeState == attributePlan {
			response.UseStateForUnknown = true
			return
		}
	}
}

// Int64Unchanged sets UseStateForUnkown to true if the attribute's planned value matches the current state
func Int64Unchanged(attributePath path.Path) UseStateForUnknownIfFunc { // nolint:gocritic // function signature required by Terraform
	return func(ctx context.Context, request planmodifier.StringRequest, response *UseStateForUnknownFuncResponse) {
		var attributePlan types.Int64
		diags := request.Plan.GetAttribute(ctx, attributePath, &attributePlan)
		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}

		var attributeState types.Int64
		diags = request.State.GetAttribute(ctx, attributePath, &attributeState)
		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}

		if attributeState == attributePlan {
			response.UseStateForUnknown = true
			return
		}
	}
}
