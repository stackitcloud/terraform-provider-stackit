package core

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Separator used for concatenation of TF-internal resource ID
const Separator = ","

type ProviderData struct {
	RoundTripper                  http.RoundTripper
	ServiceAccountEmail           string
	Region                        string
	ArgusCustomEndpoint           string
	DnsCustomEndpoint             string
	LoadBalancerCustomEndpoint    string
	LogMeCustomEndpoint           string
	MariaDBCustomEndpoint         string
	MongoDBFlexCustomEndpoint     string
	ObjectStorageCustomEndpoint   string
	OpenSearchCustomEndpoint      string
	PostgresFlexCustomEndpoint    string
	PostgreSQLCustomEndpoint      string
	RabbitMQCustomEndpoint        string
	RedisCustomEndpoint           string
	ResourceManagerCustomEndpoint string
	SecretsManagerCustomEndpoint  string
	SKECustomEndpoint             string
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

// StringValueToPointer converts basetypes.StringValue to a pointer to string.
// It returns nil if the value is null or unknown.
func StringValueToPointer(s basetypes.StringValue) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	value := s.ValueString()
	return &value
}

// Int64ValueToPointer converts basetypes.Int64Value to a pointer to int64.
// It returns nil if the value is null or unknown.
func Int64ValueToPointer(s basetypes.Int64Value) *int64 {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	value := s.ValueInt64()
	return &value
}

// BoolValueToPointer converts basetypes.BoolValue to a pointer to bool.
// It returns nil if the value is null or unknown.
func BoolValueToPointer(s basetypes.BoolValue) *bool {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	value := s.ValueBool()
	return &value
}
