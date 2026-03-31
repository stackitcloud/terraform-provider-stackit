package stringplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCronNormalizationModifier(t *testing.T) {
	modifier := CronNormalizationModifier{}

	tests := []struct {
		name             string
		configValue      string
		stateValue       string
		expectSetToState bool // If true, we expect PlanValue to be forced to StateValue
	}{
		{
			name:             "exact match",
			configValue:      "0 0 * * *",
			stateValue:       "0 0 * * *",
			expectSetToState: true,
		},
		{
			name:             "normalized match (leading zeros)",
			configValue:      "00 00 * * *",
			stateValue:       "0 0 * * *",
			expectSetToState: true,
		},
		{
			name:             "normalized match (spacing)",
			configValue:      "0  0  * * *",
			stateValue:       "0 0 * * *",
			expectSetToState: true,
		},
		{
			name:             "actual difference",
			configValue:      "0 1 * * *",
			stateValue:       "0 0 * * *",
			expectSetToState: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			req := planmodifier.StringRequest{
				ConfigValue: types.StringValue(tt.configValue),
				StateValue:  types.StringValue(tt.stateValue),
			}
			resp := planmodifier.StringResponse{
				PlanValue: types.StringValue(tt.configValue), // Default behavior: Plan follows Config
			}

			modifier.PlanModifyString(ctx, req, &resp)

			if tt.expectSetToState {
				if !resp.PlanValue.Equal(types.StringValue(tt.stateValue)) {
					t.Errorf("Expected PlanValue to be overwritten by StateValue (%s), but got %s",
						tt.stateValue, resp.PlanValue.ValueString())
				}
			} else {
				if !resp.PlanValue.Equal(types.StringValue(tt.configValue)) {
					t.Errorf("Expected PlanValue to remain as ConfigValue (%s), but got %s",
						tt.configValue, resp.PlanValue.ValueString())
				}
			}
		})
	}
}
