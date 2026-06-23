package int32planmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUseStateForUnknownIf_PlanModifyInt32(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		stateValue        types.Int32
		planValue         types.Int32
		configValue       types.Int32
		ifFunc            UseStateForUnknownIfFunc
		expectedPlanValue types.Int32
		expectedError     bool
	}{
		{
			name:        "State is Null (Creation)",
			stateValue:  types.Int32Null(),
			planValue:   types.Int32Unknown(),
			configValue: types.Int32Value(10),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int32Request, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the state is null
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.Int32Unknown(),
		},
		{
			name:        "Plan is already known - (User updated the value)",
			stateValue:  types.Int32Value(5),
			planValue:   types.Int32Value(10),
			configValue: types.Int32Value(10),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int32Request, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the plan is known
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.Int32Value(10),
		},
		{
			name:        "Config is Unknown (Interpolation)",
			stateValue:  types.Int32Value(5),
			planValue:   types.Int32Unknown(),
			configValue: types.Int32Unknown(),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int32Request, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.Int32Unknown(),
		},
		{
			name:        "Condition returns False (Do not use state)",
			stateValue:  types.Int32Value(5),
			planValue:   types.Int32Unknown(),
			configValue: types.Int32Null(), // Simulating computed only
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int32Request, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = false
			},
			expectedPlanValue: types.Int32Unknown(),
		},
		{
			name:        "Condition returns True (Use state)",
			stateValue:  types.Int32Value(5),
			planValue:   types.Int32Unknown(),
			configValue: types.Int32Null(),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int32Request, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.Int32Value(5),
		},
		{
			name:        "Func returns Error",
			stateValue:  types.Int32Value(5),
			planValue:   types.Int32Unknown(),
			configValue: types.Int32Null(),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int32Request, resp *UseStateForUnknownFuncResponse) {
				resp.Diagnostics.AddError("Test Error", "Something went wrong")
			},
			expectedPlanValue: types.Int32Unknown(),
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the modifier
			modifier := UseStateForUnknownIf(tt.ifFunc, "", "test description")

			// Construct request
			req := planmodifier.Int32Request{
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
				ConfigValue: tt.configValue,
			}

			// Construct response
			// Note: In the framework, resp.PlanValue is initialized to req.PlanValue
			// before the modifier is called. We must simulate this.
			resp := &planmodifier.Int32Response{
				PlanValue: tt.planValue,
			}

			// Run the modifier
			modifier.PlanModifyInt32(ctx, req, resp)

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
