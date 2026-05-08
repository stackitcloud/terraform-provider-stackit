package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	sdkConf "github.com/stackitcloud/stackit-sdk-go/core/config"

	"github.com/stackitcloud/terraform-provider-stackit/stackit"
)

const (
	// Default location of credentials JSON
	credentialsFilePath = ".stackit/credentials.json" //nolint:gosec // linter false positive
)

var (
	// TestAccProtoV6ProviderFactories is used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"stackit": providerserver.NewProtocol6WithError(stackit.New("test-version")()),
	}

	// TestEphemeralAccProtoV6ProviderFactories is used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	//
	// See the Terraform acceptance test documentation on ephemeral resources for more information:
	// https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests/ephemeral-resources
	TestEphemeralAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"stackit": providerserver.NewProtocol6WithError(stackit.New("test-version")()),
		"echo":    echoprovider.NewProviderServer(),
	}

	// E2ETestsEnabled checks if end-to-end tests should be run.
	// It is enabled when the TF_ACC environment variable is set to "1".
	E2ETestsEnabled = os.Getenv("TF_ACC") == "1"
	// OrganizationId is the id of organization used for tests
	OrganizationId = os.Getenv("TF_ACC_ORGANIZATION_ID")
	// ProjectId is the id of project used for tests
	ProjectId = os.Getenv("TF_ACC_PROJECT_ID")
	Region    = os.Getenv("TF_ACC_REGION")
	// TestProjectParentContainerID is the container id of the parent resource under which projects are created as part of the resource-manager acceptance tests
	TestProjectParentContainerID = os.Getenv("TF_ACC_TEST_PROJECT_PARENT_CONTAINER_ID")
	// TestProjectParentUUID is the uuid of the parent resource under which projects are created as part of the resource-manager acceptance tests
	TestProjectParentUUID = os.Getenv("TF_ACC_TEST_PROJECT_PARENT_UUID")
	// TestProjectServiceAccountEmail is the e-mail of a service account with admin permissions on the organization under which projects are created as part of the resource-manager acceptance tests
	TestProjectServiceAccountEmail = os.Getenv("TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_EMAIL")
	// TestProjectUserEmail is the e-mail of a user for the project created as part of the resource-manager acceptance tests
	// Default email: acc-test@sa.stackit.cloud
	TestProjectUserEmail = getenv("TF_ACC_TEST_PROJECT_USER_EMAIL", "acc-test@sa.stackit.cloud")
	// TestImageLocalFilePath is the local path to an image file used for image acceptance tests
	TestImageLocalFilePath = getenv("TF_ACC_TEST_IMAGE_LOCAL_FILE_PATH", "default")

	ALBCustomEndpoint              = customEndpointConfig{envVarName: "TF_ACC_ALB_CUSTOM_ENDPOINT", providerName: "alb_custom_endpoint"}
	ALBCertCustomEndpoint          = customEndpointConfig{envVarName: "TF_ACC_ALB_CERT_CUSTOM_ENDPOINT", providerName: "alb_certificates_custom_endpoint"}
	AlbWafCustomEndpoint           = customEndpointConfig{envVarName: "TF_ACC_ALB_WAF_CUSTOM_ENDPOINT", providerName: "alb_waf_custom_endpoint"}
	CdnCustomEndpoint              = customEndpointConfig{envVarName: "TF_ACC_CDN_CUSTOM_ENDPOINT", providerName: "cdn_custom_endpoint"}
	DnsCustomEndpoint              = customEndpointConfig{envVarName: "TF_ACC_DNS_CUSTOM_ENDPOINT", providerName: "dns_custom_endpoint"}
	DremioCustomEndpoint           = customEndpointConfig{envVarName: "TF_ACC_DREMIO_CUSTOM_ENDPOINT", providerName: "dremio_custom_endpoint"}
	EdgeCloudCustomEndpoint        = customEndpointConfig{envVarName: "TF_ACC_EDGECLOUD_CUSTOM_ENDPOINT", providerName: "edgecloud_custom_endpoint"}
	GitCustomEndpoint              = customEndpointConfig{envVarName: "TF_ACC_GIT_CUSTOM_ENDPOINT", providerName: "git_custom_endpoint"}
	IaaSCustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_IAAS_CUSTOM_ENDPOINT", providerName: "iaas_custom_endpoint"}
	KMSCustomEndpoint              = customEndpointConfig{envVarName: "TF_ACC_KMS_CUSTOM_ENDPOINT", providerName: "kms_custom_endpoint"}
	LoadBalancerCustomEndpoint     = customEndpointConfig{envVarName: "TF_ACC_LOADBALANCER_CUSTOM_ENDPOINT", providerName: "loadbalancer_custom_endpoint"}
	LogMeCustomEndpoint            = customEndpointConfig{envVarName: "TF_ACC_LOGME_CUSTOM_ENDPOINT", providerName: "logme_custom_endpoint"}
	LogsCustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_LOGS_CUSTOM_ENDPOINT", providerName: "logs_custom_endpoint"}
	MariaDBCustomEndpoint          = customEndpointConfig{envVarName: "TF_ACC_MARIADB_CUSTOM_ENDPOINT", providerName: "mariadb_custom_endpoint"}
	ModelServingCustomEndpoint     = customEndpointConfig{envVarName: "TF_ACC_MODELSERVING_CUSTOM_ENDPOINT", providerName: "modelserving_custom_endpoint"}
	ModelExperimentsCustomEndpoint = customEndpointConfig{envVarName: "TF_ACC_MODELEXPERIMENTS_CUSTOM_ENDPOINT", providerName: "modelexperiments_custom_endpoint"}
	AuthorizationCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_AUTHORIZATION_CUSTOM_ENDPOINT", providerName: "authorization_custom_endpoint"}
	MongoDBFlexCustomEndpoint      = customEndpointConfig{envVarName: "TF_ACC_MONGODBFLEX_CUSTOM_ENDPOINT", providerName: "mongodbflex_custom_endpoint"}
	OpenSearchCustomEndpoint       = customEndpointConfig{envVarName: "TF_ACC_OPENSEARCH_CUSTOM_ENDPOINT", providerName: "opensearch_custom_endpoint"}
	ObservabilityCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_OBSERVABILITY_CUSTOM_ENDPOINT", providerName: "observability_custom_endpoint"}
	ObjectStorageCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_OBJECTSTORAGE_CUSTOM_ENDPOINT", providerName: "objectstorage_custom_endpoint"}
	PostgresFlexCustomEndpoint     = customEndpointConfig{envVarName: "TF_ACC_POSTGRESFLEX_CUSTOM_ENDPOINT", providerName: "postgresflex_custom_endpoint"}
	RabbitMQCustomEndpoint         = customEndpointConfig{envVarName: "TF_ACC_RABBITMQ_CUSTOM_ENDPOINT", providerName: "rabbitmq_custom_endpoint"}
	RedisCustomEndpoint            = customEndpointConfig{envVarName: "TF_ACC_REDIS_CUSTOM_ENDPOINT", providerName: "redis_custom_endpoint"}
	ResourceManagerCustomEndpoint  = customEndpointConfig{envVarName: "TF_ACC_RESOURCEMANAGER_CUSTOM_ENDPOINT", providerName: "resourcemanager_custom_endpoint"}
	ScfCustomEndpoint              = customEndpointConfig{envVarName: "TF_ACC_SCF_CUSTOM_ENDPOINT", providerName: "scf_custom_endpoint"}
	SecretsManagerCustomEndpoint   = customEndpointConfig{envVarName: "TF_ACC_SECRETSMANAGER_CUSTOM_ENDPOINT", providerName: "secretsmanager_custom_endpoint"}
	SQLServerFlexCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_SQLSERVERFLEX_CUSTOM_ENDPOINT", providerName: "sqlserverflex_custom_endpoint"}
	ServerBackupCustomEndpoint     = customEndpointConfig{envVarName: "TF_ACC_SERVER_BACKUP_CUSTOM_ENDPOINT", providerName: "server_backup_custom_endpoint"}
	ServerUpdateCustomEndpoint     = customEndpointConfig{envVarName: "TF_ACC_SERVER_UPDATE_CUSTOM_ENDPOINT", providerName: "server_update_custom_endpoint"}
	SFSCustomEndpoint              = customEndpointConfig{envVarName: "TF_ACC_SFS_CUSTOM_ENDPOINT", providerName: "sfs_custom_endpoint"}
	ServiceAccountCustomEndpoint   = customEndpointConfig{envVarName: "TF_ACC_SERVICE_ACCOUNT_CUSTOM_ENDPOINT", providerName: "service_account_custom_endpoint"}
	TokenCustomEndpoint            = customEndpointConfig{envVarName: "TF_ACC_TOKEN_CUSTOM_ENDPOINT", providerName: "token_custom_endpoint"}
	VpnCustomEndpoint              = customEndpointConfig{envVarName: "TF_ACC_VPN_CUSTOM_ENDPOINT", providerName: "vpn_custom_endpoint"}
	SKECustomEndpoint              = customEndpointConfig{envVarName: "TF_ACC_SKE_CUSTOM_ENDPOINT", providerName: "ske_custom_endpoint"}
	IntakeCustomEndpoint           = customEndpointConfig{envVarName: "TF_ACC_INTAKE_CUSTOM_ENDPOINT", providerName: "intake_custom_endpoint"}
	TelemetryRouterCustomEndpoint  = customEndpointConfig{envVarName: "TF_ACC_TELEMETRYROUTER_CUSTOM_ENDPOINT", providerName: "telemetryrouter_custom_endpoint"}
	TelemetryLinkCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_TELEMETRYLINK_CUSTOM_ENDPOINT", providerName: "telemetrylink_custom_endpoint"}

	allCustomEndpoints = []customEndpointConfig{
		ALBCustomEndpoint,
		ALBCertCustomEndpoint,
		AlbWafCustomEndpoint,
		CdnCustomEndpoint,
		DnsCustomEndpoint,
		EdgeCloudCustomEndpoint,
		GitCustomEndpoint,
		IaaSCustomEndpoint,
		KMSCustomEndpoint,
		LoadBalancerCustomEndpoint,
		LogMeCustomEndpoint,
		LogsCustomEndpoint,
		MariaDBCustomEndpoint,
		ModelServingCustomEndpoint,
		ModelExperimentsCustomEndpoint,
		AuthorizationCustomEndpoint,
		MongoDBFlexCustomEndpoint,
		OpenSearchCustomEndpoint,
		ObservabilityCustomEndpoint,
		ObjectStorageCustomEndpoint,
		PostgresFlexCustomEndpoint,
		RabbitMQCustomEndpoint,
		RedisCustomEndpoint,
		ResourceManagerCustomEndpoint,
		ScfCustomEndpoint,
		SecretsManagerCustomEndpoint,
		SQLServerFlexCustomEndpoint,
		ServerBackupCustomEndpoint,
		ServerUpdateCustomEndpoint,
		SFSCustomEndpoint,
		ServiceAccountCustomEndpoint,
		TokenCustomEndpoint,
		VpnCustomEndpoint,
		SKECustomEndpoint,
		TelemetryRouterCustomEndpoint,
		TelemetryLinkCustomEndpoint,
	}
)

type Experiment string

const (
	ExperimentRoutingTables Experiment = "routing-tables"
	ExperimentNetwork       Experiment = "network"
	ExperimentIAM           Experiment = "iam"
	ExperimentDremio        Experiment = "dremio"
	ExperimentVPC           Experiment = "vpc"
)

type customEndpointConfig struct {
	envVarName   string
	providerName string
}

type ConfigBuilder struct {
	region              string
	enableBetaResources bool
	endpoints           map[string]string
	experiments         []string
	serviceAccountToken string
}

// NewConfigBuilder creates a new ConfigBuilder with region eu01 as default.
// All custom endpoints defined in TF_ACC_*_CUSTOM_ENDPOINT env vars are also set.
func NewConfigBuilder() *ConfigBuilder {
	b := &ConfigBuilder{
		region:    "eu01",
		endpoints: make(map[string]string),
	}
	for _, endpoint := range allCustomEndpoints {
		b.endpoints[endpoint.providerName] = os.Getenv(endpoint.envVarName)
	}
	return b
}

func (b *ConfigBuilder) Region(region string) *ConfigBuilder {
	b.region = region
	return b
}

func (b *ConfigBuilder) EnableBetaResources(enable bool) *ConfigBuilder {
	b.enableBetaResources = enable
	return b
}

func (b *ConfigBuilder) CustomEndpoint(endpoint customEndpointConfig, url string) *ConfigBuilder {
	b.endpoints[endpoint.providerName] = url
	return b
}

func (b *ConfigBuilder) Experiments(experiments ...Experiment) *ConfigBuilder {
	for _, e := range experiments {
		b.experiments = append(b.experiments, string(e))
	}
	return b
}

func (b *ConfigBuilder) ServiceAccountToken(token string) *ConfigBuilder {
	b.serviceAccountToken = token
	return b
}

func (b *ConfigBuilder) BuildProviderConfig() string {
	tmpl := `provider "stackit" {
    default_region = "{{ .Region }}"
    enable_beta_resources = {{ .EnableBetaResources }}
{{- if .Experiments }}
    experiments = {{ .Experiments | tfslice }}
{{- end }}
{{- if .ServiceAccountToken }}
    service_account_token = "{{ .ServiceAccountToken }}"
{{- end }}
{{- range $k, $v := .Endpoints }}
    {{ $k }} = "{{ $v }}"
{{- end }}
}`
	funcs := template.FuncMap{}
	funcs["tfslice"] = func(s []string) string {
		return "[\"" + strings.Join(s, "\", \"") + "\"]"
	}
	parsed := template.Must(template.New("providerConfig").Funcs(funcs).Parse(tmpl))
	var bs bytes.Buffer
	setEndpoints := make(map[string]string)
	for k, v := range b.endpoints {
		if v != "" {
			setEndpoints[k] = v
		}
	}
	// template needs public fields
	data := struct {
		Region              string
		EnableBetaResources bool
		Endpoints           map[string]string
		Experiments         []string
		ServiceAccountToken string
	}{
		b.region,
		b.enableBetaResources,
		setEndpoints,
		b.experiments,
		b.serviceAccountToken,
	}
	err := parsed.Execute(&bs, data)
	if err != nil {
		panic(err)
	}
	return bs.String()
}

func (b *ConfigBuilder) BuildClientOptions(service customEndpointConfig, setRegion bool) []sdkConf.ConfigurationOption {
	var opts []sdkConf.ConfigurationOption
	if b.serviceAccountToken != "" {
		opts = append(opts, sdkConf.WithToken(b.serviceAccountToken))
	}
	endpoint := b.endpoints[service.providerName]
	if endpoint != "" {
		opts = append(opts, sdkConf.WithEndpoint(endpoint))
	}
	tokenEndPoint := b.endpoints[TokenCustomEndpoint.providerName]
	if tokenEndPoint != "" {
		opts = append(opts, sdkConf.WithTokenEndpoint(tokenEndPoint))
	}
	if setRegion && b.region != "" {
		opts = append(opts, sdkConf.WithRegion(b.region))
	}
	return opts
}

func ResourceNameWithDateTime(name string) string {
	dateTime := time.Now().Format(time.RFC3339)
	// Remove timezone to have a smaller datetime
	dateTimeTrimmed, _, _ := strings.Cut(dateTime, "+")
	return fmt.Sprintf("tf-acc-%s-%s", name, dateTimeTrimmed)
}

func GetTestProjectServiceAccountToken(path string) string {
	var err error
	token, tokenSet := os.LookupEnv("TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_TOKEN")
	if !tokenSet || token == "" {
		token, err = readTestTokenFromCredentialsFile(path)
		if err != nil {
			return ""
		}
	}
	return token
}

func readTestTokenFromCredentialsFile(path string) (string, error) {
	if path == "" {
		customPath, customPathSet := os.LookupEnv("STACKIT_CREDENTIALS_PATH")
		if !customPathSet || customPath == "" {
			path = credentialsFilePath
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("getting home directory: %w", err)
			}
			path = filepath.Join(home, path)
		} else {
			path = customPath
		}
	}

	credentialsRaw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}

	var credentials struct {
		TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_TOKEN string `json:"TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_TOKEN"`
	}
	err = json.Unmarshal(credentialsRaw, &credentials)
	if err != nil {
		return "", fmt.Errorf("unmarshalling credentials: %w", err)
	}
	return credentials.TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_TOKEN, nil
}

func getenv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// CreateDefaultLocalFile is a helper for local_file_path. No real data is created
func CreateDefaultLocalFile() os.File {
	// Define the file name and size
	fileName := "test-512k.img"
	size := 512 * 1024 // 512 KB

	// Create the file
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}

	// Seek to the desired position (512 KB)
	_, err = file.Seek(int64(size), 0)
	if err != nil {
		panic(err)
	}

	return *file
}

// resolveVariablePath marshals a variable to JSON and traverses path using
// int indices (for arrays) and string keys (for objects). Panics on path mismatches.
// caller sets the panic message prefix for clear error tracing.
func resolveVariablePath(caller string, variable config.Variable, path ...any) json.RawMessage {
	tmpByteArray, err := variable.MarshalJSON()
	if err != nil {
		panic(fmt.Sprintf("%s: failed to marshal variable: %s", caller, err))
	}
	raw := json.RawMessage(tmpByteArray)

	for _, segment := range path {
		switch key := segment.(type) {
		case int:
			var list []json.RawMessage
			if err := json.Unmarshal(raw, &list); err != nil {
				panic(fmt.Sprintf("%s: cannot apply index %d, value is not a list: %s (%s)", caller, key, string(raw), err))
			}
			if key < 0 || key >= len(list) {
				panic(fmt.Sprintf("%s: index %d out of range (len %d): %s", caller, key, len(list), string(raw)))
			}
			raw = list[key]
		case string:
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(raw, &obj); err != nil {
				panic(fmt.Sprintf("%s: cannot apply key %q, value is not an object: %s (%s)", caller, key, string(raw), err))
			}
			value, ok := obj[key]
			if !ok {
				panic(fmt.Sprintf("%s: key %q not found in: %s", caller, key, string(raw)))
			}
			raw = value
		default:
			panic(fmt.Sprintf("%s: unsupported path segment type %T, must be int or string", caller, segment))
		}
	}

	return raw
}

// jsonScalarToString converts a single JSON scalar (string/number/bool/null)
// into the plain string form expected by resource.TestCheckResourceAttr.
func jsonScalarToString(raw json.RawMessage) string {
	input := string(raw)

	// If it's a JSON string (starts and ends with quotes)
	if strings.HasPrefix(input, `"`) && strings.HasSuffix(input, `"`) {
		// Unquote converts the "escaped" string back to a raw Go string
		// interpreting \n as a real newline, \" as a quote, etc.
		if unquoted, err := strconv.Unquote(input); err == nil {
			return unquoted
		}
	}

	return input
}

// ConvertConfigVariable converts a config.Variable to a string for resource.TestCheckResourceAttr.
// For composite types, pass int indices or string keys to select a nested leaf.
// E.g., ConvertConfigVariable(list, 0) or ConvertConfigVariable(obj, "key").
func ConvertConfigVariable(variable config.Variable, path ...any) string {
	return jsonScalarToString(resolveVariablePath("ConvertConfigVariable", variable, path...))
}

// buildAttrChecks recursively generates TestCheckFuncs for each leaf in raw JSON,
// mirroring Terraform's flattened state path format:
//   - Arrays: creates a count check (<path>.#) and recurses on elements (<path>.<i>).
//   - Objects: recurses on key-value pairs (<path>.<field>).
//   - Leaves: creates a TestCheckResourceAttr check for the final value.
//
// caller is used as the panic message prefix to identify which exported
// function failed.
func buildAttrChecks(caller, resourceName, path string, raw json.RawMessage) []resource.TestCheckFunc {
	trimmed := bytes.TrimSpace(raw)
	if string(trimmed) == "null" {
		// A nested config.ListVariable()/SetVariable() field with no elements
		// marshals to JSON null rather than []; treat it as an empty list.
		trimmed = json.RawMessage("[]")
	}

	if bytes.HasPrefix(trimmed, []byte("[")) {
		var items []json.RawMessage
		if err := json.Unmarshal(trimmed, &items); err != nil {
			panic(fmt.Sprintf("%s: %q: cannot parse list: %s (%s)", caller, path, err, string(trimmed)))
		}
		checks := []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, path+".#", strconv.Itoa(len(items))),
		}
		for i, item := range items {
			checks = append(checks, buildAttrChecks(caller, resourceName, fmt.Sprintf("%s.%d", path, i), item)...)
		}
		return checks
	}

	if bytes.HasPrefix(trimmed, []byte("{")) {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &obj); err != nil {
			panic(fmt.Sprintf("%s: %q: cannot parse object: %s (%s)", caller, path, err, string(trimmed)))
		}
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		checks := make([]resource.TestCheckFunc, 0, len(obj))
		for _, k := range keys {
			checks = append(checks, buildAttrChecks(caller, resourceName, path+"."+k, obj[k])...)
		}
		return checks
	}

	return []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, path, jsonScalarToString(trimmed)),
	}
}

// CheckListAttr asserts a list's count ("<attrPrefix>.#") and all elements in one call.
// Accepts an optional path for nested variables and recursively flattens
// elements (scalars or objects) to match Terraform state format
func CheckListAttr(resourceName, attrPrefix string, variable config.Variable, path ...any) resource.TestCheckFunc {
	raw := resolveVariablePath("CheckListAttr", variable, path...)

	trimmed := bytes.TrimSpace(raw)
	if string(trimmed) == "null" {
		// An empty config.ListVariable() marshals to JSON null, not [].
		trimmed = json.RawMessage("[]")
	}
	if !bytes.HasPrefix(trimmed, []byte("[")) {
		panic(fmt.Sprintf("CheckListAttr: resolved value is not a list: %s", string(raw)))
	}

	return resource.ComposeAggregateTestCheckFunc(buildAttrChecks("CheckListAttr", resourceName, attrPrefix, trimmed)...)
}

// CheckObjectAttr asserts all fields of an object variable in one call.
// Accepts an optional path for nested variables and recursively flattens
// fields (scalars, lists or nested objects) to match Terraform state format.
func CheckObjectAttr(resourceName, attrPrefix string, variable config.Variable, path ...any) resource.TestCheckFunc {
	raw := resolveVariablePath("CheckObjectAttr", variable, path...)

	trimmed := bytes.TrimSpace(raw)
	if string(trimmed) == "null" {
		// An empty config.ObjectVariable(nil) marshals to JSON null, not {}.
		trimmed = json.RawMessage("{}")
	}
	if !bytes.HasPrefix(trimmed, []byte("{")) {
		panic(fmt.Sprintf("CheckObjectAttr: resolved value is not an object: %s", string(raw)))
	}

	return resource.ComposeAggregateTestCheckFunc(buildAttrChecks("CheckObjectAttr", resourceName, attrPrefix, trimmed)...)
}

// CheckAttrHasPrefix returns a CheckResourceAttrWithFunc that validates
// whether an attribute value starts with the given prefix.
func CheckAttrHasPrefix(prefix string) resource.CheckResourceAttrWithFunc {
	return func(value string) error {
		if !strings.HasPrefix(value, prefix) {
			return fmt.Errorf("should start with %s", prefix)
		}
		return nil
	}
}
