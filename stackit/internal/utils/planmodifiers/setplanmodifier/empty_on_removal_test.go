package setplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// setStateRaw builds a tfsdk.State whose Raw value is either null (resource creation) or a
// non-null object carrying the given attribute value (existing resource).
func setStateRaw(null bool, setValue tftypes.Value) tfsdk.State {
	objectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"attr": tftypes.Set{ElementType: tftypes.String},
		},
	}

	if null {
		return tfsdk.State{Raw: tftypes.NewValue(objectType, nil)}
	}

	return tfsdk.State{Raw: tftypes.NewValue(objectType, map[string]tftypes.Value{
		"attr": setValue,
	})}
}

func TestEmptyOnRemovalModifier(t *testing.T) {
	t.Parallel()

	elementType := types.StringType

	emptySet := types.SetValueMust(elementType, []attr.Value{})
	populatedSet := types.SetValueMust(elementType, []attr.Value{
		types.StringValue("rule1"),
	})
	nullSet := types.SetNull(elementType)
	unknownSet := types.SetUnknown(elementType)

	// Non-null raw state values for existing resources.
	populatedRaw := tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
		tftypes.NewValue(tftypes.String, "rule1"),
	})
	emptyRaw := tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{})

	tests := []struct {
		description string
		state       tfsdk.State
		configValue types.Set // the value provided by the user in the Terraform configuration
		stateValue  types.Set // the value stored in the TF state
		planValue   types.Set // the value Terraform's default plan proposes
		expected    types.Set // expected result
	}{
		{
			description: "plan empty set: config removed, plan unknown, state has values",
			state:       setStateRaw(false, populatedRaw),
			configValue: nullSet,
			stateValue:  populatedSet,
			planValue:   unknownSet,
			expected:    emptySet,
		},
		{
			description: "plan empty set: config removed, plan unknown, state empty",
			state:       setStateRaw(false, emptyRaw),
			configValue: nullSet,
			stateValue:  emptySet,
			planValue:   unknownSet,
			expected:    emptySet,
		},
		{
			// Regression test for the case where removing the attribute is the only change:
			// Terraform's default plan silently carries over the prior state value (known,
			// not unknown), which must still be turned into an empty set.
			description: "plan empty set: config removed, plan carries over prior state value",
			state:       setStateRaw(false, populatedRaw),
			configValue: nullSet,
			stateValue:  populatedSet,
			planValue:   populatedSet,
			expected:    emptySet,
		},
		{
			description: "do nothing: config removed, plan already an empty set",
			state:       setStateRaw(false, populatedRaw),
			configValue: nullSet,
			stateValue:  populatedSet,
			planValue:   emptySet,
			expected:    emptySet,
		},
		{
			description: "do nothing: config has values",
			state:       setStateRaw(false, emptyRaw),
			configValue: populatedSet,
			stateValue:  emptySet,
			planValue:   populatedSet,
			expected:    populatedSet,
		},
		{
			description: "do nothing: config empty set explicitly",
			state:       setStateRaw(false, populatedRaw),
			configValue: emptySet,
			stateValue:  populatedSet,
			planValue:   emptySet,
			expected:    emptySet,
		},
		{
			description: "do nothing: config null, state null (nothing to clear)",
			state:       setStateRaw(false, emptyRaw),
			configValue: nullSet,
			stateValue:  nullSet,
			planValue:   unknownSet,
			expected:    unknownSet,
		},
		{
			description: "do nothing: config null, state unknown",
			state:       setStateRaw(false, emptyRaw),
			configValue: nullSet,
			stateValue:  unknownSet,
			planValue:   unknownSet,
			expected:    unknownSet,
		},
		{
			description: "do nothing on create: no prior state",
			state:       setStateRaw(true, tftypes.Value{}),
			configValue: nullSet,
			stateValue:  nullSet,
			planValue:   unknownSet,
			expected:    unknownSet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			// set up the request representing Terraform Core's state/config
			req := planmodifier.SetRequest{
				State:       tt.state,
				ConfigValue: tt.configValue,
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
			}

			// set up the response representing Terraform Core's proposed plan
			resp := planmodifier.SetResponse{
				PlanValue: tt.planValue,
			}

			// execute the modifier
			EmptyOnRemoval().PlanModifySet(ctx, req, &resp)

			if !resp.PlanValue.Equal(tt.expected) {
				t.Errorf("Test %q failed.\nExpected plan: %s\nGot plan: %s", tt.description, tt.expected, resp.PlanValue)
			}
		})
	}
}
