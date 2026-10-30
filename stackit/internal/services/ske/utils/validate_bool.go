package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Bool validator factory
type BoolValidator struct {
	description         string
	markdownDescription string
	validate            BoolValidationFn
}

type BoolValidationFn func(context.Context, validator.BoolRequest, *validator.BoolResponse)

var _ = validator.Bool(&BoolValidator{})

func (v *BoolValidator) Description(_ context.Context) string {
	return v.description
}

func (v *BoolValidator) MarkdownDescription(_ context.Context) string {
	return v.markdownDescription
}

func (v *BoolValidator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) { // nolint:gocritic // function signature required by Terraform
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	v.validate(ctx, req, resp)
}

type fieldInterface interface {
	IsNull() bool
	IsUnknown() bool
}

// "enables" returns a bool validator that prevents another attribute from being set if
// the own value is not true. The attributed enables another attribute.
func enables[T fieldInterface](target path.Expression, callback func(*diag.Diagnostics, path.Path, path.Path)) *BoolValidator {
	description := "this attribute enables another attribute to be set by being true"

	return &BoolValidator{
		description: description,
		validate: func(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
			expression := req.PathExpression.Merge(target)

			matchedPaths, diags := req.Config.PathMatches(ctx, expression)
			resp.Diagnostics.Append(diags...)

			for _, targetPath := range matchedPaths {
				var target T

				diags := req.Config.GetAttribute(ctx, targetPath, &target)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				if target.IsNull() || target.IsUnknown() {
					return
				}

				if req.ConfigValue.ValueBoolPointer() != nil && *req.ConfigValue.ValueBoolPointer() != true {
					callback(&resp.Diagnostics, req.Path, targetPath)
				}
			}
		},
	}
}

// Returns a bool validator based on "enables", which throws a deprecation warning instead of an error
func EnablesElseDeprecationWarning[T fieldInterface](target path.Expression, deprecationDate string) *BoolValidator {
	return enables[T](target, func(diagnostics *diag.Diagnostics, ownPath path.Path, targetPath path.Path) {
		diagnostics.AddAttributeWarning(
			targetPath,
			"Deprecation: Attribute can not be set",
			fmt.Sprintf("This attribute %q can only be configured when %q is true. Will be enforced after %s.", targetPath.String(), ownPath.String(), deprecationDate),
		)
	})
}

// Returns a bool validator based on "enables", which throws an error
func EnablesElseError[T fieldInterface](target path.Expression) *BoolValidator {
	return enables[T](target, func(diagnostics *diag.Diagnostics, ownPath path.Path, targetPath path.Path) {
		diagnostics.AddAttributeError(
			targetPath,
			"Attribute can not be set",
			fmt.Sprintf("This attribute %q can only be configured when %q is true.", targetPath.String(), ownPath.String()),
		)
	})
}
