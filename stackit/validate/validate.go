package validate

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/core"
)

type Validator struct {
	description         string
	markdownDescription string
	validate            ValidationFn
}

type ValidationFn func(context.Context, validator.StringRequest, *validator.StringResponse)

var _ = validator.String(&Validator{})

func (v *Validator) Description(_ context.Context) string {
	return v.description
}

func (v *Validator) MarkdownDescription(_ context.Context) string {
	return v.markdownDescription
}

func (v *Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) { // nolint:gocritic // function signature required by Terraform
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	v.validate(ctx, req, resp)
}

func UUID() *Validator {
	description := "value must be an UUID"

	return &Validator{
		description: description,
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			if _, err := uuid.Parse(req.ConfigValue.ValueString()); err != nil {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					description,
					req.ConfigValue.ValueString(),
				))
			}
		},
	}
}

func IP() *Validator {
	description := "value must be an IP address"

	return &Validator{
		description: description,
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			if net.ParseIP(req.ConfigValue.ValueString()) == nil {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					description,
					req.ConfigValue.ValueString(),
				))
			}
		},
	}
}

func NoSeparator() *Validator {
	description := fmt.Sprintf("value must not contain identifier separator '%s'", core.Separator)

	return &Validator{
		description: description,
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			if strings.Contains(req.ConfigValue.ValueString(), core.Separator) {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					description,
					req.ConfigValue.ValueString(),
				))
			}
		},
	}
}

func MinorVersionNumber() *Validator {
	description := "value must be a minor version number, without a leading 'v': '[MAJOR].[MINOR]'"

	return &Validator{
		description: description,
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			exp := `^\d+\.\d+?$`
			r := regexp.MustCompile(exp)
			version := req.ConfigValue.ValueString()
			if !r.MatchString(version) {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					description,
					req.ConfigValue.ValueString(),
				))
			}
		},
	}
}
