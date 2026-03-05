package int64planmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUseStateForUnknownIf_PlanModifyInt64(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		stateValue        types.Int64
		planValue         types.Int64
		configValue       types.Int64
		ifFunc            UseStateForUnknownIfFunc
		expectedPlanValue types.Int64
		expectedError     bool
	}{
		{
			name:        "State is Null (Creation)",
			stateValue:  types.Int64Null(),
			planValue:   types.Int64Unknown(),
			configValue: types.Int64Value(10),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int64Request, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the state is null
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.Int64Unknown(),
		},
		{
			name:        "Plan is already known - (User updated the value)",
			stateValue:  types.Int64Value(5),
			planValue:   types.Int64Value(10),
			configValue: types.Int64Value(10),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int64Request, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the plan is known
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.Int64Value(10),
		},
		{
			name:        "Config is Unknown (Interpolation)",
			stateValue:  types.Int64Value(5),
			planValue:   types.Int64Unknown(),
			configValue: types.Int64Unknown(),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int64Request, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.Int64Unknown(),
		},
		{
			name:        "Condition returns False (Do not use state)",
			stateValue:  types.Int64Value(5),
			planValue:   types.Int64Unknown(),
			configValue: types.Int64Null(), // Simulating computed only
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int64Request, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = false
			},
			expectedPlanValue: types.Int64Unknown(),
		},
		{
			name:        "Condition returns True (Use state)",
			stateValue:  types.Int64Value(5),
			planValue:   types.Int64Unknown(),
			configValue: types.Int64Null(),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int64Request, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.Int64Value(5),
		},
		{
			name:        "Func returns Error",
			stateValue:  types.Int64Value(5),
			planValue:   types.Int64Unknown(),
			configValue: types.Int64Null(),
			ifFunc: func(_ context.Context, _ string, _ planmodifier.Int64Request, resp *UseStateForUnknownFuncResponse) {
				resp.Diagnostics.AddError("Test Error", "Something went wrong")
			},
			expectedPlanValue: types.Int64Unknown(),
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the modifier
			modifier := UseStateForUnknownIf(tt.ifFunc, "", "test description")

			// Construct request
			req := planmodifier.Int64Request{
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
				ConfigValue: tt.configValue,
			}

			// Construct response
			// Note: In the framework, resp.PlanValue is initialized to req.PlanValue
			// before the modifier is called. We must simulate this.
			resp := &planmodifier.Int64Response{
				PlanValue: tt.planValue,
			}

			// Run the modifier
			modifier.PlanModifyInt64(ctx, req, resp)

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
