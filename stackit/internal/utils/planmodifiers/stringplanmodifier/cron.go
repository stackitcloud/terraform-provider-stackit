package stringplanmodifier

import (
	"context"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type CronNormalizationModifier struct{}

func (m CronNormalizationModifier) Description(_ context.Context) string {
	return "Prevents drift when the API normalizes cron strings (e.g., removing leading zeros)."
}

func (m CronNormalizationModifier) MarkdownDescription(_ context.Context) string {
	return "Prevents drift when the API normalizes cron strings (e.g., removing leading zeros)."
}

func (m CronNormalizationModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) { // nolint:gocritic // function signature required by Terraform
	if req.ConfigValue.IsNull() || req.StateValue.IsNull() {
		return
	}

	requestValueNormalized := utils.SimplifyCronString(req.ConfigValue.ValueString())
	stateValueNormalized := utils.SimplifyCronString(req.StateValue.ValueString())

	if requestValueNormalized == stateValueNormalized {
		resp.PlanValue = req.StateValue
	}
}
