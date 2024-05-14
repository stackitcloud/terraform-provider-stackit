package validate

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

const (
	MajorMinorVersionRegex = `^\d+\.\d+?$`
	FullVersionRegex       = `^\d+\.\d+.\d+?$`
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

func RecordSet() *Validator {
	const typePath = "type"
	return &Validator{
		description: "value must be a valid record set",
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			recordType := basetypes.StringValue{}
			req.Config.GetAttribute(ctx, path.Root(typePath), &recordType)
			switch recordType.ValueString() {
			case "A":
				ip := net.ParseIP(req.ConfigValue.ValueString())
				if ip == nil || ip.To4() == nil {
					resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
						req.Path,
						"value must be an IPv4 address",
						req.ConfigValue.ValueString(),
					))
				}
			case "AAAA":
				ip := net.ParseIP(req.ConfigValue.ValueString())
				if ip == nil || ip.To4() != nil || ip.To16() == nil {
					resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
						req.Path,
						"value must be an IPv6 address",
						req.ConfigValue.ValueString(),
					))
				}
			case "CNAME":
			case "NS":
			case "MX":
			case "TXT":
			case "ALIAS":
			case "DNAME":
			case "CAA":
			default:
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
			exp := MajorMinorVersionRegex
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

func VersionNumber() *Validator {
	description := "value must be a version number, without a leading 'v': '[MAJOR].[MINOR]' or '[MAJOR].[MINOR].[PATCH]'"

	return &Validator{
		description: description,
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			minorVersionExp := MajorMinorVersionRegex
			minorVersionRegex := regexp.MustCompile(minorVersionExp)

			versionExp := FullVersionRegex
			versionRegex := regexp.MustCompile(versionExp)

			version := req.ConfigValue.ValueString()
			if !minorVersionRegex.MatchString(version) && !versionRegex.MatchString(version) {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					description,
					req.ConfigValue.ValueString(),
				))
			}
		},
	}
}

func RFC3339SecondsOnly() *Validator {
	description := "value must be in RFC339 format (seconds only)"

	return &Validator{
		description: description,
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			t, err := time.Parse(time.RFC3339, req.ConfigValue.ValueString())
			if err != nil {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					description,
					req.ConfigValue.ValueString(),
				))
				return
			}

			// Check if it failed because it has nanoseconds
			if t.Nanosecond() != 0 {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					"value can't have fractional seconds",
					req.ConfigValue.ValueString(),
				))
			}
		},
	}
}

func CIDR() *Validator {
	description := "value must be in CIDR notation"

	return &Validator{
		description: description,
		validate: func(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
			_, _, err := net.ParseCIDR(req.ConfigValue.ValueString())
			if err != nil {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path,
					"parsing value in CIDR notation: invalid CIDR address",
					req.ConfigValue.ValueString(),
				))
			}
		},
	}
}
