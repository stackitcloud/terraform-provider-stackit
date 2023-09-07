package validate

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/google/uuid"
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
	return &Validator{
		description: "validate string is UUID",
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			if _, err := uuid.Parse(req.ConfigValue.ValueString()); err != nil {
				resp.Diagnostics.AddError("not a valid UUID", err.Error())
			}
		},
	}
}

func IP() *Validator {
	return &Validator{
		description: "validate string is IP address",
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			if net.ParseIP(req.ConfigValue.ValueString()) == nil {
				resp.Diagnostics.AddError("not a valid IP address", "")
			}
		},
	}
}

func NoSeparator() *Validator {
	return &Validator{
		description: "validate string does not contain internal separator",
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			if strings.Contains(req.ConfigValue.ValueString(), core.Separator) {
				resp.Diagnostics.AddError("Invalid character found.", fmt.Sprintf("The string should not contain a '%s'", core.Separator))
			}
		},
	}
}

func SemanticMinorVersion() *Validator {
	return &Validator{
		description: "validate string does not contain internal separator",
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			exp := `^\d+\.\d+?$`
			r := regexp.MustCompile(exp)
			version := req.ConfigValue.ValueString()
			if !r.MatchString(version) {
				resp.Diagnostics.AddError("Invalid version.", "The version should be a valid semantic version only containing major and minor version. The version should not contain a leading `v`. Got "+version)
			}
		},
	}
}
