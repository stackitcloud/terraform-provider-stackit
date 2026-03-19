package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
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

	ALBCustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_ALB_CUSTOM_ENDPOINT", providerName: "alb_custom_endpoint"}
	CdnCustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_CDN_CUSTOM_ENDPOINT", providerName: "cdn_custom_endpoint"}
	DnsCustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_DNS_CUSTOM_ENDPOINT", providerName: "dns_custom_endpoint"}
	EdgeCloudCustomEndpoint       = customEndpointConfig{envVarName: "TF_ACC_EDGECLOUD_CUSTOM_ENDPOINT", providerName: "edgecloud_custom_endpoint"}
	GitCustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_GIT_CUSTOM_ENDPOINT", providerName: "git_custom_endpoint"}
	IaaSCustomEndpoint            = customEndpointConfig{envVarName: "TF_ACC_IAAS_CUSTOM_ENDPOINT", providerName: "iaas_custom_endpoint"}
	KMSCustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_KMS_CUSTOM_ENDPOINT", providerName: "kms_custom_endpoint"}
	LoadBalancerCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_LOADBALANCER_CUSTOM_ENDPOINT", providerName: "loadbalancer_custom_endpoint"}
	LogMeCustomEndpoint           = customEndpointConfig{envVarName: "TF_ACC_LOGME_CUSTOM_ENDPOINT", providerName: "logme_custom_endpoint"}
	LogsCustomEndpoint            = customEndpointConfig{envVarName: "TF_ACC_LOGS_CUSTOM_ENDPOINT", providerName: "logs_custom_endpoint"}
	MariaDBCustomEndpoint         = customEndpointConfig{envVarName: "TF_ACC_MARIADB_CUSTOM_ENDPOINT", providerName: "mariadb_custom_endpoint"}
	ModelServingCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_MODELSERVING_CUSTOM_ENDPOINT", providerName: "modelserving_custom_endpoint"}
	AuthorizationCustomEndpoint   = customEndpointConfig{envVarName: "TF_ACC_AUTHORIZATION_CUSTOM_ENDPOINT", providerName: "authorization_custom_endpoint"}
	MongoDBFlexCustomEndpoint     = customEndpointConfig{envVarName: "TF_ACC_MONGODBFLEX_CUSTOM_ENDPOINT", providerName: "mongodbflex_custom_endpoint"}
	OpenSearchCustomEndpoint      = customEndpointConfig{envVarName: "TF_ACC_OPENSEARCH_CUSTOM_ENDPOINT", providerName: "opensearch_custom_endpoint"}
	ObservabilityCustomEndpoint   = customEndpointConfig{envVarName: "TF_ACC_OBSERVABILITY_CUSTOM_ENDPOINT", providerName: "observability_custom_endpoint"}
	ObjectStorageCustomEndpoint   = customEndpointConfig{envVarName: "TF_ACC_OBJECTSTORAGE_CUSTOM_ENDPOINT", providerName: "objectstorage_custom_endpoint"}
	PostgresFlexCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_POSTGRESFLEX_CUSTOM_ENDPOINT", providerName: "postgresflex_custom_endpoint"}
	RabbitMQCustomEndpoint        = customEndpointConfig{envVarName: "TF_ACC_RABBITMQ_CUSTOM_ENDPOINT", providerName: "rabbitmq_custom_endpoint"}
	RedisCustomEndpoint           = customEndpointConfig{envVarName: "TF_ACC_REDIS_CUSTOM_ENDPOINT", providerName: "redis_custom_endpoint"}
	ResourceManagerCustomEndpoint = customEndpointConfig{envVarName: "TF_ACC_RESOURCEMANAGER_CUSTOM_ENDPOINT", providerName: "resourcemanager_custom_endpoint"}
	ScfCustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_SCF_CUSTOM_ENDPOINT", providerName: "scf_custom_endpoint"}
	SecretsManagerCustomEndpoint  = customEndpointConfig{envVarName: "TF_ACC_SECRETSMANAGER_CUSTOM_ENDPOINT", providerName: "secretsmanager_custom_endpoint"}
	SQLServerFlexCustomEndpoint   = customEndpointConfig{envVarName: "TF_ACC_SQLSERVERFLEX_CUSTOM_ENDPOINT", providerName: "sqlserverflex_custom_endpoint"}
	ServerBackupCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_SERVER_BACKUP_CUSTOM_ENDPOINT", providerName: "server_backup_custom_endpoint"}
	ServerUpdateCustomEndpoint    = customEndpointConfig{envVarName: "TF_ACC_SERVER_UPDATE_CUSTOM_ENDPOINT", providerName: "server_update_custom_endpoint"}
	SFSCustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_SFS_CUSTOM_ENDPOINT", providerName: "sfs_custom_endpoint"}
	ServiceAccountCustomEndpoint  = customEndpointConfig{envVarName: "TF_ACC_SERVICE_ACCOUNT_CUSTOM_ENDPOINT", providerName: "service_account_custom_endpoint"}
	TokenCustomEndpoint           = customEndpointConfig{envVarName: "TF_ACC_TOKEN_CUSTOM_ENDPOINT", providerName: "token_custom_endpoint"}
	SKECustomEndpoint             = customEndpointConfig{envVarName: "TF_ACC_SKE_CUSTOM_ENDPOINT", providerName: "ske_custom_endpoint"}

	allCustomEndpoints = []customEndpointConfig{
		ALBCustomEndpoint,
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
		SKECustomEndpoint,
	}
)

type Experiment string

const (
	ExperimentRoutingTables Experiment = "routing-tables"
	ExperimentNetwork       Experiment = "network"
	ExperimentIAM           Experiment = "iam"
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

// NewConfigBuilder creates a new ConfigBuilder with enabled beta resources and region eu01 as default.
// All custom endpoints defined in TF_ACC_*_CUSTOM_ENDPOINT env vars are also set.
func NewConfigBuilder() *ConfigBuilder {
	b := &ConfigBuilder{
		region:              "eu01",
		enableBetaResources: true,
		endpoints:           make(map[string]string),
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

func (b *ConfigBuilder) BuildClientOptions(service customEndpointConfig) []sdkConf.ConfigurationOption {
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
	return opts
}

// Provider config helper functions

//go:fix inline
func ALBProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func ObservabilityProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func CdnProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func DnsProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func EdgeCloudProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func IaaSProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func IaaSProviderConfigWithBetaResourcesEnabled() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func IaaSProviderConfigWithExperiments() string {
	return NewConfigBuilder().
		Experiments(ExperimentNetwork, ExperimentRoutingTables).
		BuildProviderConfig()
}

//go:fix inline
func KMSProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func LoadBalancerProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func LogMeProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func LogsProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func MariaDBProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func ModelServingProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func MongoDBFlexProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func ObjectStorageProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func OpenSearchProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func PostgresFlexProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func RabbitMQProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func RedisProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func ResourceManagerProviderConfig() string {
	token := GetTestProjectServiceAccountToken("")
	return NewConfigBuilder().
		ServiceAccountToken(token).
		BuildProviderConfig()
}

//go:fix inline
func SecretsManagerProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func SQLServerFlexProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func ServerBackupProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func ServerUpdateProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func SFSProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func SKEProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func AuthorizationProviderConfig() string {
	return NewConfigBuilder().
		Experiments(ExperimentIAM).
		BuildProviderConfig()
}

//go:fix inline
func ServiceAccountProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func GitProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
}

//go:fix inline
func ScfProviderConfig() string {
	return NewConfigBuilder().BuildProviderConfig()
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

func ConvertConfigVariable(variable config.Variable) string {
	tmpByteArray, _ := variable.MarshalJSON()
	input := string(tmpByteArray)

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
