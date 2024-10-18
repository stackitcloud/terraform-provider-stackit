package securitygrouprule

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
)

// UseNullForUnknownBasedOnProtocolModifier returns a plan modifier that sets a null
// value into the planned value, based on the value of the protocol.name attribute.
//
// To prevent Terraform errors, the framework automatically sets unconfigured
// and Computed attributes to an unknown value "(known after apply)" on update.
// To prevent always showing "(known after apply)" on update for an attribute, e.g. port_range, which never changes in case the protocol is a specific one,
// we set the value to null.
// Examples: port_range is only computed if protocol is not icmp and icmp_parameters is only computed if protocol is icmp
func UseNullForUnknownBasedOnProtocolModifier() planmodifier.Object {
	return useNullForUnknownBasedOnProtocolModifier{}
}

// useNullForUnknownBasedOnProtocolModifier implements the plan modifier.
type useNullForUnknownBasedOnProtocolModifier struct{}

func (m useNullForUnknownBasedOnProtocolModifier) Description(_ context.Context) string {
	return "If protocol.name attribute is set and the value corresponds to an icmp protocol, the value of this attribute in state will be set to null."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m useNullForUnknownBasedOnProtocolModifier) MarkdownDescription(_ context.Context) string {
	return "Once set, the value of this attribute in state will be set to null if protocol.name attribute is set and the value corresponds to an icmp protocol."
}

// PlanModifyBool implements the plan modification logic.
func (m useNullForUnknownBasedOnProtocolModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) { // nolint:gocritic // function signature required by Terraform
	// Check if the resource is being created.
	if req.State.Raw.IsNull() {
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

	// If there is an unknown configuration value, check if the value of protocol.name attribute corresponds to an icmp protocol. If it does, set the attribute value to null
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If protocol is not configured, return without error.
	if model.Protocol.IsNull() || model.Protocol.IsUnknown() {
		return
	}

	protocol := &protocolModel{}
	diags := model.Protocol.As(ctx, protocol, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	protocolName := conversion.StringValueToPointer(protocol.Name)

	if protocolName == nil {
		return
	}

	if slices.Contains(icmpProtocols, *protocolName) {
		if model.PortRange.IsUnknown() {
			resp.PlanValue = types.ObjectNull(portRangeTypes)
			return
		}
	} else {
		if model.IcmpParameters.IsUnknown() {
			resp.PlanValue = types.ObjectNull(icmpParametersTypes)
			return
		}
	}

	// use state for unknown if the value was not set to null
	resp.PlanValue = req.StateValue
}
