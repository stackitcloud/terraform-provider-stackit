package validate

import (
	"context"
	"fmt"
	_ "time/tzdata"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type Int64Validator struct {
	description         string
	markdownDescription string
	validate            Int64ValidationFn
}

type Int64ValidationFn func(context.Context, validator.Int64Request, *validator.Int64Response)

var _ = validator.Int64(&Int64Validator{})

func (v *Int64Validator) Description(_ context.Context) string {
	return v.description
}

func (v *Int64Validator) MarkdownDescription(_ context.Context) string {
	return v.markdownDescription
}

func (v *Int64Validator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) { // nolint:gocritic // function signature required by Terraform
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	v.validate(ctx, req, resp)
}

func OnlyUpdateToLargerValue() *Int64Validator {
	description := "value can only be updated to a larger value than the current one (%d)"

	return &Int64Validator{
		description: description,
		validate: func(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
			attributePath := req.Path
			var currentAttributeValue attr.Value
			diags := req.Config.GetAttribute(ctx, attributePath, currentAttributeValue)
			resp.Diagnostics.Append(diags...)
			if diags.HasError() {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					"get current value of the attribute failed, path might be wrong",
					string(req.ConfigValue.ValueInt64()),
				))
			}

			// If the current path value is null or unknown, there is no validation to be done
			if currentAttributeValue.IsNull() || currentAttributeValue.IsUnknown() {
				return
			}

			terraformValue, err := currentAttributeValue.ToTerraformValue(ctx)
			if err != nil {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					fmt.Sprintf("converting current attribute value to terraform value: %w", err),
					string(req.ConfigValue.ValueInt64()),
				))
			}
			var currentIntValue int64
			if err := terraformValue.Copy().As(&currentIntValue); err != nil {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					fmt.Sprintf("converting current attribute value to int64: %w", err),
					string(req.ConfigValue.ValueInt64()),
				))
			}

			if currentIntValue <= req.ConfigValue.ValueInt64() {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					fmt.Sprintf(description, currentIntValue),
					string(req.ConfigValue.ValueInt64()),
				)
			}
		},
	}
}
