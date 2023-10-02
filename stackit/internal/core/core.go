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
	RoundTripper                  http.RoundTripper
	ServiceAccountEmail           string
	Region                        string
	DnsCustomEndpoint             string
	PostgreSQLCustomEndpoint      string
	PostgresFlexCustomEndpoint    string
	LogMeCustomEndpoint           string
	RabbitMQCustomEndpoint        string
	MariaDBCustomEndpoint         string
	ObjectStorageCustomEndpoint   string
	OpenSearchCustomEndpoint      string
	RedisCustomEndpoint           string
	ArgusCustomEndpoint           string
	SKECustomEndpoint             string
	ResourceManagerCustomEndpoint string
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
