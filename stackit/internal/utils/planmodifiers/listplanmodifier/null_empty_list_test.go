package listplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSuppressNullEmptyListModifier(t *testing.T) {
	elementType := types.StringType

	// Helper variables for clean test case definitions
	emptyList := types.ListValueMust(elementType, []attr.Value{})
	populatedList := types.ListValueMust(elementType, []attr.Value{
		types.StringValue("10.0.0.0/24"),
	})
	nullList := types.ListNull(elementType)

	tests := []struct {
		description string
		configValue types.List // the value provided by the user in the Terraform configuration
		stateValue  types.List // the value stored in the TF state
		planValue   types.List // the value Terraform's default plan proposes
		expected    types.List // expected result
	}{
		{
			description: "suppress diff: config null, state empty",
			configValue: nullList,
			stateValue:  emptyList,
			planValue:   nullList,  // Terraform's default plan initially proposes null
			expected:    emptyList, // plan modifier implementation should step in and change it to empty []
		},
		{
			description: "do nothing: config has values",
			configValue: populatedList,
			stateValue:  emptyList,
			planValue:   populatedList,
			expected:    populatedList,
		},
		{
			description: "do nothing: config null, state has values",
			configValue: nullList,
			stateValue:  populatedList,
			planValue:   nullList,
			expected:    nullList, // handled by Terraform's default plan (user implies removal)
		},
		{
			description: "do nothing: config empty, state empty",
			configValue: emptyList,
			stateValue:  emptyList,
			planValue:   emptyList,
			expected:    emptyList,
		},
		{
			description: "do nothing: config null, state null",
			configValue: nullList,
			stateValue:  nullList,
			planValue:   nullList,
			expected:    nullList,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			ctx := context.Background()

			// set up the request representing Terraform Core's state/config
			req := planmodifier.ListRequest{
				ConfigValue: tt.configValue,
				StateValue:  tt.stateValue,
			}

			// set up the response representing Terraform Core's proposed plan
			resp := planmodifier.ListResponse{
				PlanValue: tt.planValue,
			}

			// execute the modifier
			SuppressNullEmptyList().PlanModifyList(ctx, req, &resp)

			if !resp.PlanValue.Equal(tt.expected) {
				t.Errorf("Test %q failed.\nExpected plan: %s\nGot plan: %s", tt.description, tt.expected, resp.PlanValue)
			}
		})
	}
}
