package core

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Separator used for concatenation of TF-internal resource ID
const Separator = ","

type ProviderData struct {
	RoundTripper                    http.RoundTripper
	ServiceAccountEmail             string // Deprecated: ServiceAccountEmail is not required and will be removed after 12th June 2025.
	Region                          string
	ArgusCustomEndpoint             string
	AuthorizationCustomEndpoint     string
	DnsCustomEndpoint               string
	IaaSCustomEndpoint              string
	LoadBalancerCustomEndpoint      string
	LogMeCustomEndpoint             string
	MariaDBCustomEndpoint           string
	MongoDBFlexCustomEndpoint       string
	ObjectStorageCustomEndpoint     string
	ObservabilityCustomEndpoint     string
	OpenSearchCustomEndpoint        string
	PostgresFlexCustomEndpoint      string
	RabbitMQCustomEndpoint          string
	RedisCustomEndpoint             string
	ResourceManagerCustomEndpoint   string
	SecretsManagerCustomEndpoint    string
	SQLServerFlexCustomEndpoint     string
	ServerBackupCustomEndpoint      string
	ServerUpdateCustomEndpoint      string
	SKECustomEndpoint               string
	ServiceEnablementCustomEndpoint string
	ServiceAccountCustomEndpoint    string
	EnableBetaResources             bool
}

// DiagsToError Converts TF diagnostics' errors into an error with a human-readable description.
// If there are no errors, the output is nil
func DiagsToError(diags diag.Diagnostics) error {
	if !diags.HasError() {
		return nil
	}

	diagsError := diags.Errors()
	diagsStrings := make([]string, 0)
	for _, diagnostic := range diagsError {
		diagsStrings = append(diagsStrings, fmt.Sprintf(
			"(%s) %s",
			diagnostic.Summary(),
			diagnostic.Detail(),
		))
	}
	return fmt.Errorf("%s", strings.Join(diagsStrings, ";"))
}

// LogAndAddError Logs the error and adds it to the diags
func LogAndAddError(ctx context.Context, diags *diag.Diagnostics, summary, detail string) {
	tflog.Error(ctx, fmt.Sprintf("%s | %s", summary, detail))
	diags.AddError(summary, detail)
}

// LogAndAddWarning Logs the warning and adds it to the diags
func LogAndAddWarning(ctx context.Context, diags *diag.Diagnostics, summary, detail string) {
	tflog.Warn(ctx, fmt.Sprintf("%s | %s", summary, detail))
	diags.AddWarning(summary, detail)
}

func LogAndAddWarningBeta(ctx context.Context, diags *diag.Diagnostics, name, resourceType string) {
	warnTitle := fmt.Sprintf("The %s %q is in beta", resourceType, name)
	warnContent := fmt.Sprintf("The %s %q is in beta and may be subject to breaking changes in the future. Use with caution.", resourceType, name)
	tflog.Warn(ctx, fmt.Sprintf("%s | %s", warnTitle, warnContent))
	diags.AddWarning(warnTitle, warnContent)
}

func LogAndAddErrorBeta(ctx context.Context, diags *diag.Diagnostics, name, resourceType string) {
	errTitle := fmt.Sprintf("The %s %q is in beta and beta is not enabled", resourceType, name)
	errContent := fmt.Sprintf(`The %s %q is in beta and the beta functionality is currently not enabled. To enable it, set the environment variable STACKIT_TF_ENABLE_BETA_RESOURCES to "true" or set the "enable_beta_resources" provider field to true.`, resourceType, name)
	tflog.Error(ctx, fmt.Sprintf("%s | %s", errTitle, errContent))
	diags.AddError(errTitle, errContent)
}
