package objectplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUseStateForUnknownIf_PlanModifyObject(t *testing.T) {
	attrTypesObject := map[string]attr.Type{
		"key": types.StringType,
	}

	ctx := context.Background()

	tests := []struct {
		name              string
		stateValue        types.Object
		planValue         types.Object
		configValue       types.Object
		ifFunc            UseStateForUnknownIfFunc
		expectedPlanValue types.Object
		expectedError     bool
	}{
		{
			name:        "State is Null (Creation)",
			stateValue:  types.ObjectNull(attrTypesObject),
			planValue:   types.ObjectUnknown(attrTypesObject),
			configValue: types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value1")}),
			ifFunc: func(_ context.Context, _ planmodifier.ObjectRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the state is null
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.ObjectUnknown(attrTypesObject),
		},
		{
			name:        "Plan is already known - (User updated the value)",
			stateValue:  types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value1")}),
			planValue:   types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value2")}),
			configValue: types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value2")}),
			ifFunc: func(_ context.Context, _ planmodifier.ObjectRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the plan is known
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value2")}),
		},
		{
			name:        "Config is Unknown (Interpolation)",
			stateValue:  types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value1")}),
			planValue:   types.ObjectUnknown(attrTypesObject),
			configValue: types.ObjectUnknown(attrTypesObject),
			ifFunc: func(_ context.Context, _ planmodifier.ObjectRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.ObjectUnknown(attrTypesObject),
		},
		{
			name:        "Condition returns False (Do not use state)",
			stateValue:  types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value1")}),
			planValue:   types.ObjectUnknown(attrTypesObject),
			configValue: types.ObjectNull(attrTypesObject), // Simulating computed only
			ifFunc: func(_ context.Context, _ planmodifier.ObjectRequest, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = false
			},
			expectedPlanValue: types.ObjectUnknown(attrTypesObject),
		},
		{
			name:        "Condition returns True (Use state)",
			stateValue:  types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value1")}),
			planValue:   types.ObjectUnknown(attrTypesObject),
			configValue: types.ObjectNull(attrTypesObject),
			ifFunc: func(_ context.Context, _ planmodifier.ObjectRequest, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value1")}),
		},
		{
			name:        "Func returns Error",
			stateValue:  types.ObjectValueMust(attrTypesObject, map[string]attr.Value{"key": types.StringValue("value1")}),
			planValue:   types.ObjectUnknown(attrTypesObject),
			configValue: types.ObjectNull(attrTypesObject),
			ifFunc: func(_ context.Context, _ planmodifier.ObjectRequest, resp *UseStateForUnknownFuncResponse) {
				resp.Diagnostics.AddError("Test Error", "Something went wrong")
			},
			expectedPlanValue: types.ObjectUnknown(attrTypesObject),
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the modifier
			modifier := UseStateForUnknownIf(tt.ifFunc, "test description")

			// Construct request
			req := planmodifier.ObjectRequest{
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
				ConfigValue: tt.configValue,
			}

			// Construct response
			// Note: In the framework, resp.PlanValue is initialized to req.PlanValue
			// before the modifier is called. We must simulate this.
			resp := &planmodifier.ObjectResponse{
				PlanValue: tt.planValue,
			}

			// Run the modifier
			modifier.PlanModifyObject(ctx, req, resp)

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
