package custom_rule_group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnlyAllowedIfBoolEqualsValidator prevents that this string attribute is set if a target bool does not equal the specified value.
type OnlyAllowedIfBoolEqualsValidator struct {
	Target path.Expression
	Value  bool
}

// Ensure the validator implements the String validator interface
var _ validator.String = OnlyAllowedIfBoolEqualsValidator{}

func (v OnlyAllowedIfBoolEqualsValidator) Description(_ context.Context) string {
	return "The attribute can only be set if the boolean is set to the provided value."
}

func (v OnlyAllowedIfBoolEqualsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v OnlyAllowedIfBoolEqualsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) { // nolint:gocritic // function signature required by Terraform
	expression := req.PathExpression.Merge(v.Target)

	matchedPaths, diags := req.Config.PathMatches(ctx, expression)
	resp.Diagnostics.Append(diags...)

	for _, target := range matchedPaths {
		var targetBool types.Bool
		diags := req.Config.GetAttribute(ctx, target, &targetBool)
		resp.Diagnostics.Append(diags...)

		if resp.Diagnostics.HasError() || targetBool.IsUnknown() {
			return
		}

		if targetBool.ValueBool() != v.Value && !req.ConfigValue.IsNull() {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Attribute can not be set",
				fmt.Sprintf("This attribute can only be configured when %q is set to %t.", target.String(), v.Value),
			)
		}
	}
}

func OnlyAllowedIfBoolEquals(target path.Expression, value bool) validator.String {
	return OnlyAllowedIfBoolEqualsValidator{
		Target: target,
		Value:  value,
	}
}
