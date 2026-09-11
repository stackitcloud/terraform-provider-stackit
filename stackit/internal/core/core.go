package core

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/runtime"
	alb "github.com/stackitcloud/stackit-sdk-go/services/alb/v2api"
	albwaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"
	authorization "github.com/stackitcloud/stackit-sdk-go/services/authorization/v2api"
	cdn "github.com/stackitcloud/stackit-sdk-go/services/cdn/v1api"
	certSdk "github.com/stackitcloud/stackit-sdk-go/services/certificates/v2api"
	dns "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
	dremio "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1betaapi"
	edge "github.com/stackitcloud/stackit-sdk-go/services/edge/v1beta1api"
	git "github.com/stackitcloud/stackit-sdk-go/services/git/v1betaapi"
	iaasv2alpha "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"
	iaasv2 "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	intake "github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi"
	kms "github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
	logme "github.com/stackitcloud/stackit-sdk-go/services/logme/v2api"
	logs "github.com/stackitcloud/stackit-sdk-go/services/logs/v1api"
	mariadb "github.com/stackitcloud/stackit-sdk-go/services/mariadb/v2api"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	modelserving "github.com/stackitcloud/stackit-sdk-go/services/modelserving/v1api"
	mongodbflex "github.com/stackitcloud/stackit-sdk-go/services/mongodbflex/v2api"
	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"
	observability "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"
	opensearch "github.com/stackitcloud/stackit-sdk-go/services/opensearch/v2api"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api"
	rabbitmq "github.com/stackitcloud/stackit-sdk-go/services/rabbitmq/v2api"
	redis "github.com/stackitcloud/stackit-sdk-go/services/redis/v2api"
	resourcemanager "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v0api"
	scf "github.com/stackitcloud/stackit-sdk-go/services/scf/v1api"
	secretsmanagerV1Alpha "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1alphaapi"
	secretsmanager "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1api"
	serverbackup "github.com/stackitcloud/stackit-sdk-go/services/serverbackup/v2api"
	serverupdate "github.com/stackitcloud/stackit-sdk-go/services/serverupdate/v2api"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"
	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"
	telemetrylink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1api"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1api"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
)

type ResourceType string

const (
	Resource          ResourceType = "resource"
	Datasource        ResourceType = "datasource"
	EphemeralResource ResourceType = "ephemeral-resource"

	// Separator used for concatenation of TF-internal resource ID
	Separator = ","

	ResourceRegionFallbackDocstring   = "Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on resource level."
	DatasourceRegionFallbackDocstring = "Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on datasource level."
)

var DefaultTimeoutMargin = 3 * time.Minute
var DefaultOperationTimeout = 30 * time.Minute

type EphemeralProviderData struct {
	ProviderData
}

type CustomEndpointConfig struct {
	ALBCertificatesCustomEndpoint   string
	ALBCustomEndpoint               string
	AlbWafCustomEndpoint            string
	AuthorizationCustomEndpoint     string
	CdnCustomEndpoint               string
	DnsCustomEndpoint               string
	DremioCustomEndpoint            string
	EdgeCloudCustomEndpoint         string
	GitCustomEndpoint               string
	IaaSCustomEndpoint              string
	IntakeCustomEndpoint            string
	KMSCustomEndpoint               string
	LoadBalancerCustomEndpoint      string
	LogMeCustomEndpoint             string
	LogsCustomEndpoint              string
	MariaDBCustomEndpoint           string
	MongoDBFlexCustomEndpoint       string
	ModelServingCustomEndpoint      string
	ModelExperimentsCustomEndpoint  string
	ObjectStorageCustomEndpoint     string
	ObservabilityCustomEndpoint     string
	OpenSearchCustomEndpoint        string
	PostgresFlexCustomEndpoint      string
	RabbitMQCustomEndpoint          string
	RedisCustomEndpoint             string
	ResourceManagerCustomEndpoint   string
	ScfCustomEndpoint               string
	SecretsManagerCustomEndpoint    string
	SQLServerFlexCustomEndpoint     string
	ServerBackupCustomEndpoint      string
	ServerUpdateCustomEndpoint      string
	SKECustomEndpoint               string
	ServiceEnablementCustomEndpoint string
	SfsCustomEndpoint               string
	ServiceAccountCustomEndpoint    string
	TelemetryLinkCustomEndpoint     string
	TelemetryRouterCustomEndpoint   string
	VpnCustomEndpoint               string
}

func NewProviderDataInternal(providerData ProviderData, clientFactory ClientFactory) (providerDataInternal, error) {
	clients, err := initClientCollection(clientFactory)
	if err != nil {
		return providerDataInternal{}, err
	}

	return providerDataInternal{
		clients:      *clients,
		providerData: providerData,
	}, nil
}

type providerDataInternal struct {
	// providerData is the public provider data
	providerData ProviderData
	clients      clientCollection
}

type clientCollection struct {
	IaaSv2Client                iaasv2.DefaultAPI
	IaaSv2AlphaClient           iaasv2alpha.DefaultAPI
	ResourceManagerClient       resourcemanager.DefaultAPI
	ModelExperimentsV1Client    modelexperiments.DefaultAPI
	EdgeV1Client                edge.DefaultAPI
	DnsV1Client                 dns.DefaultAPI
	CdnV1Client                 cdn.DefaultAPI
	ServerBackupV2Client        serverbackup.DefaultAPI
	AlbCertificatesV2Client     certSdk.DefaultAPI
	ServiceEnablementV2Client   serviceenablement.DefaultAPI
	AlbWafV1CLient              albwaf.DefaultAPI
	LogsV1Client                logs.DefaultAPI
	VpnV1Client                 vpn.DefaultAPI
	AuthorizationV2Client       authorization.DefaultAPI
	PostgresflexV3Client        postgresflex.DefaultAPI
	AlbV2Client                 alb.DefaultAPI
	SkeV2Client                 ske.DefaultAPI
	SqlServerFlexV3Client       sqlserverflex.DefaultAPI
	ModelservingV1Client        modelserving.DefaultAPI
	LogmeV2Client               logme.DefaultAPI
	OpensearchV2Client          opensearch.DefaultAPI
	GitV1BetaClient             git.DefaultAPI
	RedisV2Client               redis.DefaultAPI
	TelemetryRouterV1Client     telemetryrouter.DefaultAPI
	TelemetryLinkV1Client       telemetrylink.DefaultAPI
	ServerUpdateV2Client        serverupdate.DefaultAPI
	KmsV1Client                 kms.DefaultAPI
	SfsV1Client                 sfs.DefaultAPI
	ServiceAccountV2Client      serviceaccount.DefaultAPI
	RabbitMqV2Client            rabbitmq.DefaultAPI
	MongoDbFlexV2Client         mongodbflex.DefaultAPI
	ObjectStorageV2Client       objectstorage.DefaultAPI
	MariadbV2Client             mariadb.DefaultAPI
	ScfV1Client                 scf.DefaultAPI
	LoadbalancerV2Client        loadbalancer.DefaultAPI
	IntakeV1BetaClient          intake.DefaultAPI
	DremioV1BetaClient          dremio.DefaultAPI
	SecretsmanagerV1Client      secretsmanager.DefaultAPI
	SecretsmanagerV1AlphaClient secretsmanagerV1Alpha.DefaultAPI
	ObservabilityV1Client       observability.DefaultAPI
}

func parseInternalProviderData(ctx context.Context, providerData any, diags *diag.Diagnostics) (providerDataInternal, bool) {
	// Prevent panic if the provider has not been configured.
	if providerData == nil {
		return providerDataInternal{}, false
	}

	stackitProviderDataInternal, ok := providerData.(providerDataInternal)
	if !ok {
		LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Expected configure type core.providerDataInternal, got %T", providerData))
		return providerDataInternal{}, false
	}
	return stackitProviderDataInternal, true
}

func ParseProviderData(ctx context.Context, providerData any, diags *diag.Diagnostics) (ProviderData, clientCollection, bool) {
	// Prevent panic if the provider has not been configured.
	if providerData == nil {
		return ProviderData{}, clientCollection{}, false
	}

	stackitProviderDataInternal, ok := providerData.(providerDataInternal)
	if !ok {
		LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Expected configure type core.providerDataInternal, got %T", providerData))
		return ProviderData{}, clientCollection{}, false
	}
	return stackitProviderDataInternal.providerData, stackitProviderDataInternal.clients, true
}

func ParseProviderDataLeg(ctx context.Context, providerData any, diags *diag.Diagnostics) (ProviderData, bool) {
	providerDataInternal, ok := parseInternalProviderData(ctx, providerData, diags)
	return providerDataInternal.providerData, ok
}

func ParseClients(ctx context.Context, providerData any, diags *diag.Diagnostics) (clientCollection, bool) {
	providerDataInternal, ok := parseInternalProviderData(ctx, providerData, diags)
	return providerDataInternal.clients, ok
}

type ProviderData struct {
	ServiceAccountEmail string
	DefaultRegion       string
	EnableBetaResources bool
	Experiments         []string
}

// GetRegion returns the effective region for the provider, falling back to the deprecated _region_ attribute
func (pd *ProviderData) GetRegion() string {
	if pd.DefaultRegion != "" {
		return pd.DefaultRegion
	}
	return "eu01"
}

func (pd *ProviderData) GetRegionWithOverride(overrideRegion types.String) string {
	if overrideRegion.IsUnknown() || overrideRegion.IsNull() {
		return pd.GetRegion()
	}
	return overrideRegion.ValueString()
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
	if traceId := runtime.GetTraceId(ctx); traceId != "" {
		detail = fmt.Sprintf("%s\nTrace ID: %q", detail, traceId)
	}

	tflog.Error(ctx, fmt.Sprintf("%s | %s", summary, detail))
	diags.AddError(summary, detail)
}

// LogAndAddWarning Logs the warning and adds it to the diags
func LogAndAddWarning(ctx context.Context, diags *diag.Diagnostics, summary, detail string) {
	if traceId := runtime.GetTraceId(ctx); traceId != "" {
		detail = fmt.Sprintf("%s\nTrace ID: %q", detail, traceId)
	}

	tflog.Warn(ctx, fmt.Sprintf("%s | %s", summary, detail))
	diags.AddWarning(summary, detail)
}

func LogAndAddWarningBeta(ctx context.Context, diags *diag.Diagnostics, name string, resourceType ResourceType) {
	warnTitle := fmt.Sprintf("The %s %q is in beta", resourceType, name)
	warnContent := fmt.Sprintf("The %s %q is in beta and may be subject to breaking changes in the future. Use with caution.", resourceType, name)
	tflog.Warn(ctx, fmt.Sprintf("%s | %s", warnTitle, warnContent))
	diags.AddWarning(warnTitle, warnContent)
}

func LogAndAddErrorBeta(ctx context.Context, diags *diag.Diagnostics, name string, resourceType ResourceType) {
	errTitle := fmt.Sprintf("The %s %q is in beta and beta is not enabled", resourceType, name)
	errContent := fmt.Sprintf(`The %s %q is in beta and the beta functionality is currently not enabled. To enable it, set the environment variable STACKIT_TF_ENABLE_BETA_RESOURCES to "true" or set the "enable_beta_resources" provider field to true.`, resourceType, name)
	tflog.Error(ctx, fmt.Sprintf("%s | %s", errTitle, errContent))
	diags.AddError(errTitle, errContent)
}

// InitProviderContext extends the context to capture the http response
func InitProviderContext(ctx context.Context) context.Context {
	// Capture http response to get trace-id
	var httpResp *http.Response
	return runtime.WithCaptureHTTPResponse(ctx, &httpResp)
}

// LogResponse logs the trace-id of the last request
func LogResponse(ctx context.Context) context.Context {
	// Logs the trace-id of the request
	traceId := runtime.GetTraceId(ctx)
	ctx = tflog.SetField(ctx, "x-trace-id", traceId)

	tflog.Info(ctx, "response data", map[string]any{
		"x-trace-id": traceId,
	})
	return ctx
}
