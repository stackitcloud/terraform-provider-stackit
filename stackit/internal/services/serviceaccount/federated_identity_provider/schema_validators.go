package federated_identity_provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// assertionsValidator implements the validator.List interface.
type assertionsValidator struct{}

func (v assertionsValidator) Description(_ context.Context) string {
	return "Ensure assertions are correct."
}

func (v assertionsValidator) MarkdownDescription(_ context.Context) string {
	return "Ensure assertions are correct."
}

func (v assertionsValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) { //nolint:gocritic // function signature required by Terraform
	// Skip validation when the value is null or unknown, for example during plan with computed values.
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var assertions []AssertionModel
	diags := req.ConfigValue.ElementsAs(ctx, &assertions, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	foundAud := false
	for _, assertion := range assertions {
		if !assertion.Item.IsNull() && !assertion.Item.IsUnknown() && assertion.Item.ValueString() == "aud" {
			foundAud = true
			break
		}
	}

	// If no "aud" assertion is found, return an error pointing to the attribute path.
	if !foundAud {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing Required Assertion",
			"The 'assertions' list must contain at least one block where the 'item' field is exactly \"aud\".",
		)
	}
}

// requireAssertions returns the helper validator used in the schema.
func requireAssertions() validator.List {
	return assertionsValidator{}
}
