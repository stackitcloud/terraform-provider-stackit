package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUseStateForUnknownIf_PlanModifyString(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		stateValue        types.String
		planValue         types.String
		configValue       types.String
		ifFunc            UseStateForUnknownIfFunc
		expectedPlanValue types.String
		expectedError     bool
	}{
		{
			name:        "State is Null (Creation)",
			stateValue:  types.StringNull(),
			planValue:   types.StringUnknown(),
			configValue: types.StringValue("some-config"),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the state is null
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.StringUnknown(),
		},
		{
			name:        "Plan is already known - (User updated the value)",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringValue("new-plan"),
			configValue: types.StringValue("new-plan"),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the plan is known
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.StringValue("new-plan"),
		},
		{
			name:        "Config is Unknown (Interpolation)",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringUnknown(),
			configValue: types.StringUnknown(),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.StringUnknown(),
		},
		{
			name:        "Condition returns False (Do not use state)",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(), // Simulating computed only
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = false
			},
			expectedPlanValue: types.StringUnknown(),
		},
		{
			name:        "Condition returns True (Use state)",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.StringValue("old-state"),
		},
		{
			name:        "Func returns Error",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				resp.Diagnostics.AddError("Test Error", "Something went wrong")
			},
			expectedPlanValue: types.StringUnknown(),
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the modifier
			modifier := UseStateForUnknownIf(tt.ifFunc, "test description")

			// Construct request
			req := planmodifier.StringRequest{
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
				ConfigValue: tt.configValue,
			}

			// Construct response
			// Note: In the framework, resp.PlanValue is initialized to req.PlanValue
			// before the modifier is called. We must simulate this.
			resp := &planmodifier.StringResponse{
				PlanValue: tt.planValue,
			}

			// Run the modifier
			modifier.PlanModifyString(ctx, req, resp)

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
