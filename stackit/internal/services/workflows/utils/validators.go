package utils

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// URL returns a validator that checks the value is a syntactically valid
// absolute HTTP or HTTPS URL. Catches typos at plan time rather than letting
// them surface as opaque server-side errors after a wait.
func URL() validator.String {
	return urlValidator{description: "value must be a valid http:// or https:// URL"}
}

// URLHTTPSOnly is URL() but rejects http:// — for endpoints where plaintext is
// never acceptable (OIDC discovery, anything carrying credentials).
func URLHTTPSOnly() validator.String {
	return urlValidator{description: "value must be a valid https:// URL", httpsOnly: true}
}

type urlValidator struct {
	description string
	httpsOnly   bool
}

func (v urlValidator) Description(_ context.Context) string         { return v.description }
func (v urlValidator) MarkdownDescription(_ context.Context) string { return v.description }

func (v urlValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) { //nolint:gocritic // function signature required by Terraform
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	u, err := url.Parse(val)
	schemeOK := u != nil && (u.Scheme == "https" || (!v.httpsOnly && u.Scheme == "http"))
	if err != nil || u == nil || !u.IsAbs() || !schemeOK || u.Host == "" {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path,
			v.description,
			val,
		))
	}
}

// Airflow3Version rejects Workflows version strings that aren't Airflow 3+.
// The version format is `workflows-X.Y-airflow-A.B`. Airflow 2 instances need
// a `dagsRepository` field that the provider doesn't expose; this validator
// surfaces the constraint at plan time instead of failing with a server 400.
func Airflow3Version() validator.String { return airflow3VersionValidator{} }

type airflow3VersionValidator struct{}

var airflowMajorRE = regexp.MustCompile(`airflow-(\d+)\.`)

func (airflow3VersionValidator) Description(_ context.Context) string {
	return "version must use Airflow 3 or newer"
}

func (v airflow3VersionValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (airflow3VersionValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) { //nolint:gocritic // function signature required by Terraform
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	m := airflowMajorRE.FindStringSubmatch(val)
	if m == nil {
		return
	}
	major, err := strconv.Atoi(m[1])
	if err != nil || major >= 3 {
		return
	}
	resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
		req.Path,
		fmt.Sprintf("Unsupported Airflow version: %q is Airflow %d. This provider only supports Airflow 3+ — older versions require the deprecated `dagsRepository` field that is not exposed here. Use the `stackit_workflows_provider_options` data source to discover supported versions.", val, major),
		val,
	))
}
