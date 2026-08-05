package validate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnlyIfBoolValidator checks if this string attribute is set when a target bool is true.
type OnlyIfBoolValidator struct {
	Target path.Expression
	Value  bool
}

// Ensure the validator implements the String validator interface
var _ validator.String = OnlyIfBoolValidator{}

func (v OnlyIfBoolValidator) Description(_ context.Context) string {
	return "The attribute can only be set if the boolean is set to the provided Value."
}

func (v OnlyIfBoolValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v OnlyIfBoolValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) { // nolint:gocritic // function signature required by Terraform
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

func OnlyIfBool(target path.Expression, value bool) validator.String {
	return OnlyIfBoolValidator{
		Target: target,
		Value:  value,
	}
}
