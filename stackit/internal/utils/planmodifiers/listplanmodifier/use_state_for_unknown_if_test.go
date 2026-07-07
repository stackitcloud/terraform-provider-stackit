package listplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUseStateForUnknownIf_PlanModifyList(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		stateValue        types.List
		planValue         types.List
		configValue       types.List
		ifFunc            UseStateForUnknownIfFunc
		expectedPlanValue types.List
		expectedError     bool
	}{
		{
			name:        "State is Null (Creation)",
			stateValue:  types.ListNull(types.StringType),
			planValue:   types.ListUnknown(types.StringType),
			configValue: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("value1")}),
			ifFunc: func(_ context.Context, _ planmodifier.ListRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the state is null
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.ListUnknown(types.StringType),
		},
		{
			name:        "Plan is already known - (User updated the value)",
			stateValue:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("1")}),
			planValue:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("2")}),
			configValue: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("2")}),
			ifFunc: func(_ context.Context, _ planmodifier.ListRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the plan is known
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("2")}),
		},
		{
			name:        "Config is Unknown (Interpolation)",
			stateValue:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("1")}),
			planValue:   types.ListUnknown(types.StringType),
			configValue: types.ListUnknown(types.StringType),
			ifFunc: func(_ context.Context, _ planmodifier.ListRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.ListUnknown(types.StringType),
		},
		{
			name:        "Condition returns False (Do not use state)",
			stateValue:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("1")}),
			planValue:   types.ListUnknown(types.StringType),
			configValue: types.ListNull(types.StringType), // Simulating computed only
			ifFunc: func(_ context.Context, _ planmodifier.ListRequest, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = false
			},
			expectedPlanValue: types.ListUnknown(types.StringType),
		},
		{
			name:        "Condition returns True (Use state)",
			stateValue:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("1")}),
			planValue:   types.ListUnknown(types.StringType),
			configValue: types.ListNull(types.StringType),
			ifFunc: func(_ context.Context, _ planmodifier.ListRequest, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("1")}),
		},
		{
			name:        "Func returns Error",
			stateValue:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("1")}),
			planValue:   types.ListUnknown(types.StringType),
			configValue: types.ListNull(types.StringType),
			ifFunc: func(_ context.Context, _ planmodifier.ListRequest, resp *UseStateForUnknownFuncResponse) {
				resp.Diagnostics.AddError("Test Error", "Something went wrong")
			},
			expectedPlanValue: types.ListUnknown(types.StringType),
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the modifier
			modifier := UseStateForUnknownIf(tt.ifFunc, "test description")

			// Construct request
			req := planmodifier.ListRequest{
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
				ConfigValue: tt.configValue,
			}

			// Construct response
			// Note: In the framework, resp.PlanValue is initialized to req.PlanValue
			// before the modifier is called. We must simulate this.
			resp := &planmodifier.ListResponse{
				PlanValue: tt.planValue,
			}

			// Run the modifier
			modifier.PlanModifyList(ctx, req, resp)

			// Check Errors
			if tt.expectedError {
				if !resp.Diagnostics.HasError() {
					t.Error("Expected error, got none")
				}
			} else {
				if resp.Diagnostics.HasError() {
					t.Errorf("Unexpected error: %s", resp.Diagnostics)
				}
			}

			// Check Plan Value
			if !resp.PlanValue.Equal(tt.expectedPlanValue) {
				t.Errorf("PlanValue mismatch.\nExpected: %s\nGot:      %s", tt.expectedPlanValue, resp.PlanValue)
			}
		})
	}
}
