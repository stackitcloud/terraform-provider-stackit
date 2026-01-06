package stackit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	sdkauth "github.com/stackitcloud/stackit-sdk-go/core/auth"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/access_token"
	customRole "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/authorization/customrole"
	roleAssignements "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/authorization/roleassignments"
	cdnCustomDomain "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/customdomain"
	cdn "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/distribution"
	dnsRecordSet "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/recordset"
	dnsZone "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/zone"
	edgeCloudInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/edgecloud/instance"
	edgeCloudInstances "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/edgecloud/instances"
	edgeCloudKubeconfig "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/edgecloud/kubeconfig"
	edgeCloudPlans "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/edgecloud/plans"
	edgeCloudToken "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/edgecloud/token"
	gitInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/git/instance"
	iaasAffinityGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/affinitygroup"
	iaasImage "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/image"
	iaasImageV2 "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/imagev2"
	iaasKeyPair "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/keypair"
	machineType "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/machinetype"
	iaasNetwork "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network"
	iaasNetworkArea "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkarea"
	iaasNetworkAreaRegion "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkarearegion"
	iaasNetworkAreaRoute "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkarearoute"
	iaasNetworkInterface "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkinterface"
	iaasNetworkInterfaceAttach "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkinterfaceattach"
	iaasProject "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/project"
	iaasPublicIp "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/publicip"
	iaasPublicIpAssociate "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/publicipassociate"
	iaasPublicIpRanges "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/publicipranges"
	iaasRoutingTableRoute "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/routingtable/route"
	iaasRoutingTableRoutes "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/routingtable/routes"
	iaasRoutingTable "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/routingtable/table"
	iaasRoutingTables "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/routingtable/tables"
	iaasSecurityGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/securitygroup"
	iaasSecurityGroupRule "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/securitygrouprule"
	iaasServer "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/server"
	iaasServiceAccountAttach "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/serviceaccountattach"
	iaasVolume "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/volume"
	iaasVolumeAttach "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/volumeattach"
	kmsKey "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/key"
	kmsKeyRing "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/keyring"
	kmsWrappingKey "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/wrapping-key"
	loadBalancer "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/loadbalancer/loadbalancer"
	loadBalancerObservabilityCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/loadbalancer/observability-credential"
	logMeCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logme/credential"
	logMeInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logme/instance"
	logsAccessToken "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logs/accesstoken"
	logsInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logs/instance"
	mariaDBCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mariadb/credential"
	mariaDBInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mariadb/instance"
	modelServingToken "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelserving/token"
	mongoDBFlexInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mongodbflex/instance"
	mongoDBFlexUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mongodbflex/user"
	objectStorageBucket "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/bucket"
	objecStorageCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/credential"
	objecStorageCredentialsGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/credentialsgroup"
	alertGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/alertgroup"
	observabilityCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/credential"
	observabilityInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/instance"
	logAlertGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/log-alertgroup"
	observabilityScrapeConfig "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/scrapeconfig"
	openSearchCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/opensearch/credential"
	openSearchInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/opensearch/instance"
	postgresFlexDatabase "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresflex/database"
	postgresFlexInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresflex/instance"
	postgresFlexUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresflex/user"
	rabbitMQCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/rabbitmq/credential"
	rabbitMQInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/rabbitmq/instance"
	redisCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/redis/credential"
	redisInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/redis/instance"
	resourceManagerFolder "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/folder"
	resourceManagerProject "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/project"
	scfOrganization "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/scf/organization"
	scfOrganizationmanager "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/scf/organizationmanager"
	scfPlatform "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/scf/platform"
	secretsManagerInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/secretsmanager/instance"
	secretsManagerUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/secretsmanager/user"
	serverBackupSchedule "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serverbackup/schedule"
	serverUpdateSchedule "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serverupdate/schedule"
	serviceAccount "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/account"
	serviceAccountKey "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/key"
	serviceAccountToken "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/token"
	exportpolicy "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/export-policy"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/resourcepool"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/share"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/snapshots"
	skeCluster "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/cluster"
	skeKubeconfig "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/kubeconfig"
	skeKubernetesVersion "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/provideroptions/kubernetesversions"
	skeMachineImages "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/provideroptions/machineimages"
	sqlServerFlexInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/instance"
	sqlServerFlexUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/user"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider                       = &Provider{}
	_ provider.ProviderWithEphemeralResources = &Provider{}
)

// Provider is the provider implementation.
type Provider struct {
	version string
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Provider{
			version: version,
		}
	}
}

func (p *Provider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "stackit"
	resp.Version = p.version
}

type providerModel struct {
	CredentialsFilePath   types.String `tfsdk:"credentials_path"`
	ServiceAccountEmail   types.String `tfsdk:"service_account_email"`
	ServiceAccountKey     types.String `tfsdk:"service_account_key"`
	ServiceAccountKeyPath types.String `tfsdk:"service_account_key_path"`
	PrivateKey            types.String `tfsdk:"private_key"`
	PrivateKeyPath        types.String `tfsdk:"private_key_path"`
	Token                 types.String `tfsdk:"service_account_token"`
	WifFederatedTokenPath types.String `tfsdk:"service_account_federated_token_path"`
	UseOIDC               types.Bool   `tfsdk:"use_oidc"`

	// Deprecated: Use DefaultRegion instead
	Region        types.String `tfsdk:"region"`
	DefaultRegion types.String `tfsdk:"default_region"`

	// Custom endpoints
	AuthorizationCustomEndpoint     types.String `tfsdk:"authorization_custom_endpoint"`
	CdnCustomEndpoint               types.String `tfsdk:"cdn_custom_endpoint"`
	DnsCustomEndpoint               types.String `tfsdk:"dns_custom_endpoint"`
	EdgeCloudCustomEndpoint         types.String `tfsdk:"edgecloud_custom_endpoint"`
	GitCustomEndpoint               types.String `tfsdk:"git_custom_endpoint"`
	IaaSCustomEndpoint              types.String `tfsdk:"iaas_custom_endpoint"`
	KmsCustomEndpoint               types.String `tfsdk:"kms_custom_endpoint"`
	LoadBalancerCustomEndpoint      types.String `tfsdk:"loadbalancer_custom_endpoint"`
	LogMeCustomEndpoint             types.String `tfsdk:"logme_custom_endpoint"`
	LogsCustomEndpoint              types.String `tfsdk:"logs_custom_endpoint"`
	MariaDBCustomEndpoint           types.String `tfsdk:"mariadb_custom_endpoint"`
	ModelServingCustomEndpoint      types.String `tfsdk:"modelserving_custom_endpoint"`
	MongoDBFlexCustomEndpoint       types.String `tfsdk:"mongodbflex_custom_endpoint"`
	ObjectStorageCustomEndpoint     types.String `tfsdk:"objectstorage_custom_endpoint"`
	ObservabilityCustomEndpoint     types.String `tfsdk:"observability_custom_endpoint"`
	OpenSearchCustomEndpoint        types.String `tfsdk:"opensearch_custom_endpoint"`
	PostgresFlexCustomEndpoint      types.String `tfsdk:"postgresflex_custom_endpoint"`
	RabbitMQCustomEndpoint          types.String `tfsdk:"rabbitmq_custom_endpoint"`
	RedisCustomEndpoint             types.String `tfsdk:"redis_custom_endpoint"`
	ResourceManagerCustomEndpoint   types.String `tfsdk:"resourcemanager_custom_endpoint"`
	ScfCustomEndpoint               types.String `tfsdk:"scf_custom_endpoint"`
	SecretsManagerCustomEndpoint    types.String `tfsdk:"secretsmanager_custom_endpoint"`
	ServerBackupCustomEndpoint      types.String `tfsdk:"server_backup_custom_endpoint"`
	ServerUpdateCustomEndpoint      types.String `tfsdk:"server_update_custom_endpoint"`
	ServiceAccountCustomEndpoint    types.String `tfsdk:"service_account_custom_endpoint"`
	ServiceEnablementCustomEndpoint types.String `tfsdk:"service_enablement_custom_endpoint"`
	SfsCustomEndpoint               types.String `tfsdk:"sfs_custom_endpoint"`
	SkeCustomEndpoint               types.String `tfsdk:"ske_custom_endpoint"`
	SqlServerFlexCustomEndpoint     types.String `tfsdk:"sqlserverflex_custom_endpoint"`
	TokenCustomEndpoint             types.String `tfsdk:"token_custom_endpoint"`
	OIDCTokenRequestURL             types.String `tfsdk:"oidc_request_url"`
	OIDCTokenRequestToken           types.String `tfsdk:"oidc_request_token"`

	EnableBetaResources types.Bool `tfsdk:"enable_beta_resources"`
	Experiments         types.List `tfsdk:"experiments"`
}

// Schema defines the provider-level schema for configuration data.
func (p *Provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	descriptions := map[string]string{
		"credentials_path":                     "Path of JSON from where the credentials are read. Takes precedence over the env var `STACKIT_CREDENTIALS_PATH`. Default value is `~/.stackit/credentials.json`.",
		"service_account_token":                "Token used for authentication. If set, the token flow will be used to authenticate all operations.",
		"service_account_key_path":             "Path for the service account key used for authentication. If set, the key flow will be used to authenticate all operations.",
		"service_account_key":                  "Service account key used for authentication. If set, the key flow will be used to authenticate all operations.",
		"private_key_path":                     "Path for the private RSA key used for authentication, relevant for the key flow. It takes precedence over the private key that is included in the service account key.",
		"private_key":                          "Private RSA key used for authentication, relevant for the key flow. It takes precedence over the private key that is included in the service account key.",
		"service_account_email":                "Service account email. It can also be set using the environment variable STACKIT_SERVICE_ACCOUNT_EMAIL. It is required if you want to use the resource manager project resource.",
		"service_account_federated_token_path": "Path for workload identity assertion. It can also be set using the environment variable STACKIT_FEDERATED_TOKEN_FILE.",
		"use_oidc":                             "Should OIDC be used for Authentication? This can also be sourced from the `STACKIT_USE_OIDC` Environment Variable. Defaults to `false`.",
		"oidc_request_url":                     "The URL for the OIDC provider from which to request an ID token. For use when authenticating as a Service Account using OpenID Connect.",
		"oidc_request_token":                   "The bearer token for the request to the OIDC provider. For use when authenticating as a Service Account using OpenID Connect.",
		"region":                               "Region will be used as the default location for regional services. Not all services require a region, some are global",
		"default_region":                       "Region will be used as the default location for regional services. Not all services require a region, some are global",
		"cdn_custom_endpoint":                  "Custom endpoint for the CDN service",
		"dns_custom_endpoint":                  "Custom endpoint for the DNS service",
		"git_custom_endpoint":                  "Custom endpoint for the Git service",
		"iaas_custom_endpoint":                 "Custom endpoint for the IaaS service",
		"kms_custom_endpoint":                  "Custom endpoint for the KMS service",
		"mongodbflex_custom_endpoint":          "Custom endpoint for the MongoDB Flex service",
		"modelserving_custom_endpoint":         "Custom endpoint for the AI Model Serving service",
		"loadbalancer_custom_endpoint":         "Custom endpoint for the Load Balancer service",
		"logme_custom_endpoint":                "Custom endpoint for the LogMe service",
		"rabbitmq_custom_endpoint":             "Custom endpoint for the RabbitMQ service",
		"mariadb_custom_endpoint":              "Custom endpoint for the MariaDB service",
		"authorization_custom_endpoint":        "Custom endpoint for the Membership service",
		"objectstorage_custom_endpoint":        "Custom endpoint for the Object Storage service",
		"observability_custom_endpoint":        "Custom endpoint for the Observability service",
		"opensearch_custom_endpoint":           "Custom endpoint for the OpenSearch service",
		"postgresflex_custom_endpoint":         "Custom endpoint for the PostgresFlex service",
		"redis_custom_endpoint":                "Custom endpoint for the Redis service",
		"server_backup_custom_endpoint":        "Custom endpoint for the Server Backup service",
		"server_update_custom_endpoint":        "Custom endpoint for the Server Update service",
		"service_account_custom_endpoint":      "Custom endpoint for the Service Account service",
		"resourcemanager_custom_endpoint":      "Custom endpoint for the Resource Manager service",
		"scf_custom_endpoint":                  "Custom endpoint for the Cloud Foundry (SCF) service",
		"secretsmanager_custom_endpoint":       "Custom endpoint for the Secrets Manager service",
		"sqlserverflex_custom_endpoint":        "Custom endpoint for the SQL Server Flex service",
		"ske_custom_endpoint":                  "Custom endpoint for the Kubernetes Engine (SKE) service",
		"service_enablement_custom_endpoint":   "Custom endpoint for the Service Enablement API",
		"sfs_custom_endpoint":                  "Custom endpoint for the Stackit Filestorage API",
		"token_custom_endpoint":                "Custom endpoint for the token API, which is used to request access tokens when using the key flow",
		"enable_beta_resources":                "Enable beta resources. Default is false.",
		"experiments":                          fmt.Sprintf("Enables experiments. These are unstable features without official support. More information can be found in the README. Available Experiments: %v", strings.Join(features.AvailableExperiments, ", ")),
	}
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"credentials_path": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["credentials_path"],
			},
			"service_account_email": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["service_account_email"],
			},
			"service_account_token": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["service_account_token"],
				DeprecationMessage: "Authentication via Service Account Token is deprecated and will be removed on December 17, 2025. " +
					"Please use `service_account_key` or `service_account_key_path` instead. " +
					"For a smooth transition, refer to our migration guide: https://docs.stackit.cloud/platform/access-and-identity/service-accounts/migrate-flows/",
			},
			"service_account_key_path": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["service_account_key_path"],
			},
			"service_account_key": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["service_account_key"],
			},
			"private_key": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["private_key"],
			},
			"private_key_path": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["private_key_path"],
			},
			"service_account_federated_token_path": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["service_account_federated_token_path"],
			},
			"use_oidc": schema.BoolAttribute{
				Optional:    true,
				Description: descriptions["use_oidc"],
			},
			"oidc_request_token": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["oidc_request_token"],
			},
			"oidc_request_url": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["oidc_request_url"],
			},
			"region": schema.StringAttribute{
				Optional:           true,
				Description:        descriptions["region"],
				DeprecationMessage: "This attribute is deprecated. Use 'default_region' instead",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("default_region")),
				},
			},
			"default_region": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["default_region"],
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("region")),
				},
			},
			"enable_beta_resources": schema.BoolAttribute{
				Optional:    true,
				Description: descriptions["enable_beta_resources"],
			},
			"experiments": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: descriptions["experiments"],
			},
			// Custom endpoints
			"cdn_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["cdn_custom_endpoint"],
			},
			"dns_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["dns_custom_endpoint"],
			},
			"edgecloud_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["edgecloud_custom_endpoint"],
			},
			"git_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["git_custom_endpoint"],
			},
			"iaas_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["iaas_custom_endpoint"],
			},
			"kms_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["kms_custom_endpoint"],
			},
			"postgresflex_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["postgresflex_custom_endpoint"],
			},
			"mariadb_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["mariadb_custom_endpoint"],
			},
			"modelserving_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["modelserving_custom_endpoint"],
			},
			"authorization_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["authorization_custom_endpoint"],
			},
			"mongodbflex_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["mongodbflex_custom_endpoint"],
			},
			"loadbalancer_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["loadbalancer_custom_endpoint"],
			},
			"logme_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["logme_custom_endpoint"],
			},
			"logs_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["logs_custom_endpoint"],
			},
			"rabbitmq_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["rabbitmq_custom_endpoint"],
			},
			"objectstorage_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["objectstorage_custom_endpoint"],
			},
			"observability_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["observability_custom_endpoint"],
			},
			"opensearch_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["opensearch_custom_endpoint"],
			},
			"redis_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["redis_custom_endpoint"],
			},
			"scf_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["scf_custom_endpoint"],
			},
			"resourcemanager_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["resourcemanager_custom_endpoint"],
			},
			"secretsmanager_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["secretsmanager_custom_endpoint"],
			},
			"sqlserverflex_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["sqlserverflex_custom_endpoint"],
			},
			"ske_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["ske_custom_endpoint"],
			},
			"server_backup_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["server_backup_custom_endpoint"],
			},
			"server_update_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["server_update_custom_endpoint"],
			},
			"service_account_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["service_account_custom_endpoint"],
			},
			"service_enablement_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["service_enablement_custom_endpoint"],
			},
			"sfs_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["sfs_custom_endpoint"],
			},
			"token_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["token_custom_endpoint"],
			},
		},
	}
}

// Configure prepares a stackit API client for data sources and resources.
func (p *Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data and configuration
	var providerConfig providerModel
	diags := req.Config.Get(ctx, &providerConfig)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Configure SDK client
	sdkConfig := &config.Configuration{}
	var providerData core.ProviderData

	// Helper function to set a string field if it's known and not null
	setStringField := func(v basetypes.StringValue, setter func(string)) {
		if !v.IsUnknown() && !v.IsNull() {
			setter(v.ValueString())
		}
	}

	// Helper function to set a boolean field if it's known and not null
	setBoolField := func(v basetypes.BoolValuable, setter func(bool)) {
		if !v.IsUnknown() && !v.IsNull() {
			val, err := v.ToBoolValue(ctx)
			if err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring provider", fmt.Sprintf("Setting up bool value: %v", diags.Errors()))
			}
			setter(val.ValueBool())
		}
	}

	// Configure SDK client
	setStringField(providerConfig.CredentialsFilePath, func(v string) { sdkConfig.CredentialsFilePath = v })
	setStringField(providerConfig.ServiceAccountEmail, func(v string) { sdkConfig.ServiceAccountEmail = v })
	setStringField(providerConfig.ServiceAccountKey, func(v string) { sdkConfig.ServiceAccountKey = v })
	setStringField(providerConfig.ServiceAccountKeyPath, func(v string) { sdkConfig.ServiceAccountKeyPath = v })
	setStringField(providerConfig.PrivateKey, func(v string) { sdkConfig.PrivateKey = v })
	setStringField(providerConfig.PrivateKeyPath, func(v string) { sdkConfig.PrivateKeyPath = v })
	setStringField(providerConfig.WifFederatedTokenPath, func(v string) { sdkConfig.ServiceAccountFederatedTokenPath = v })
	setBoolField(providerConfig.UseOIDC, func(v bool) { sdkConfig.WorkloadIdentityFederation = v })
	setStringField(providerConfig.Token, func(v string) { sdkConfig.Token = v })
	setStringField(providerConfig.TokenCustomEndpoint, func(v string) { sdkConfig.TokenCustomUrl = v })

	setStringField(providerConfig.DefaultRegion, func(v string) { providerData.DefaultRegion = v })
	setStringField(providerConfig.Region, func(v string) { providerData.Region = v }) // nolint:staticcheck // preliminary handling of deprecated attribute
	setBoolField(providerConfig.EnableBetaResources, func(v bool) { providerData.EnableBetaResources = v })

	setStringField(providerConfig.AuthorizationCustomEndpoint, func(v string) { providerData.AuthorizationCustomEndpoint = v })
	setStringField(providerConfig.CdnCustomEndpoint, func(v string) { providerData.CdnCustomEndpoint = v })
	setStringField(providerConfig.DnsCustomEndpoint, func(v string) { providerData.DnsCustomEndpoint = v })
	setStringField(providerConfig.EdgeCloudCustomEndpoint, func(v string) { providerData.EdgeCloudCustomEndpoint = v })
	setStringField(providerConfig.GitCustomEndpoint, func(v string) { providerData.GitCustomEndpoint = v })
	setStringField(providerConfig.IaaSCustomEndpoint, func(v string) { providerData.IaaSCustomEndpoint = v })
	setStringField(providerConfig.KmsCustomEndpoint, func(v string) { providerData.KMSCustomEndpoint = v })
	setStringField(providerConfig.LoadBalancerCustomEndpoint, func(v string) { providerData.LoadBalancerCustomEndpoint = v })
	setStringField(providerConfig.LogMeCustomEndpoint, func(v string) { providerData.LogMeCustomEndpoint = v })
	setStringField(providerConfig.LogsCustomEndpoint, func(v string) { providerData.LogsCustomEndpoint = v })
	setStringField(providerConfig.MariaDBCustomEndpoint, func(v string) { providerData.MariaDBCustomEndpoint = v })
	setStringField(providerConfig.ModelServingCustomEndpoint, func(v string) { providerData.ModelServingCustomEndpoint = v })
	setStringField(providerConfig.MongoDBFlexCustomEndpoint, func(v string) { providerData.MongoDBFlexCustomEndpoint = v })
	setStringField(providerConfig.ObjectStorageCustomEndpoint, func(v string) { providerData.ObjectStorageCustomEndpoint = v })
	setStringField(providerConfig.ObservabilityCustomEndpoint, func(v string) { providerData.ObservabilityCustomEndpoint = v })
	setStringField(providerConfig.OpenSearchCustomEndpoint, func(v string) { providerData.OpenSearchCustomEndpoint = v })
	setStringField(providerConfig.PostgresFlexCustomEndpoint, func(v string) { providerData.PostgresFlexCustomEndpoint = v })
	setStringField(providerConfig.RabbitMQCustomEndpoint, func(v string) { providerData.RabbitMQCustomEndpoint = v })
	setStringField(providerConfig.RedisCustomEndpoint, func(v string) { providerData.RedisCustomEndpoint = v })
	setStringField(providerConfig.ResourceManagerCustomEndpoint, func(v string) { providerData.ResourceManagerCustomEndpoint = v })
	setStringField(providerConfig.ScfCustomEndpoint, func(v string) { providerData.ScfCustomEndpoint = v })
	setStringField(providerConfig.SecretsManagerCustomEndpoint, func(v string) { providerData.SecretsManagerCustomEndpoint = v })
	setStringField(providerConfig.ServerBackupCustomEndpoint, func(v string) { providerData.ServerBackupCustomEndpoint = v })
	setStringField(providerConfig.ServerUpdateCustomEndpoint, func(v string) { providerData.ServerUpdateCustomEndpoint = v })
	setStringField(providerConfig.ServiceAccountCustomEndpoint, func(v string) { providerData.ServiceAccountCustomEndpoint = v })
	setStringField(providerConfig.ServiceEnablementCustomEndpoint, func(v string) { providerData.ServiceEnablementCustomEndpoint = v })
	setStringField(providerConfig.SfsCustomEndpoint, func(v string) { providerData.SfsCustomEndpoint = v })
	setStringField(providerConfig.SkeCustomEndpoint, func(v string) { providerData.SKECustomEndpoint = v })
	setStringField(providerConfig.SqlServerFlexCustomEndpoint, func(v string) { providerData.SQLServerFlexCustomEndpoint = v })

	if !(providerConfig.Experiments.IsUnknown() || providerConfig.Experiments.IsNull()) {
		var experimentValues []string
		diags := providerConfig.Experiments.ElementsAs(ctx, &experimentValues, false)
		if diags.HasError() {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring provider", fmt.Sprintf("Setting up experiments: %v", diags.Errors()))
		}
		providerData.Experiments = experimentValues
	}

	if sdkConfig.WorkloadIdentityFederation {
		// https://docs.github.com/en/actions/reference/security/oidc#methods-for-requesting-the-oidc-token
		oidcReqURL := getEnvStringOrDefault(providerConfig.OIDCTokenRequestURL, "ACTIONS_ID_TOKEN_REQUEST_URL", "")
		oidcReqToken := getEnvStringOrDefault(providerConfig.OIDCTokenRequestToken, "ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")
		if oidcReqURL != "" && oidcReqToken != "" {
			id_token, err := githubAssertion(ctx, oidcReqURL, oidcReqToken)
			if err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring provider", fmt.Sprintf("Requesting id token from Github %v", err))
				return
			}
			sdkConfig.ServiceAccountFederatedToken = id_token
		}
	}

	roundTripper, err := sdkauth.SetupAuth(sdkConfig)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring provider", fmt.Sprintf("Setting up authentication: %v", err))
		return
	}

	// Make round tripper and custom endpoints available during DataSource and Resource
	// type Configure methods.
	providerData.RoundTripper = roundTripper

	providerData.Version = p.version

	resp.DataSourceData = providerData
	resp.ResourceData = providerData

	// Copy service account, private key credentials and custom-token endpoint to support ephemeral access token generation
	var ephemeralProviderData core.EphemeralProviderData
	ephemeralProviderData.ProviderData = providerData
	setStringField(providerConfig.ServiceAccountEmail, func(v string) { ephemeralProviderData.ServiceAccountEmail = v })
	setStringField(providerConfig.ServiceAccountKey, func(v string) { ephemeralProviderData.ServiceAccountKey = v })
	setStringField(providerConfig.ServiceAccountKeyPath, func(v string) { ephemeralProviderData.ServiceAccountKeyPath = v })
	setStringField(providerConfig.PrivateKey, func(v string) { ephemeralProviderData.PrivateKey = v })
	setStringField(providerConfig.PrivateKeyPath, func(v string) { ephemeralProviderData.PrivateKeyPath = v })
	setStringField(providerConfig.TokenCustomEndpoint, func(v string) { ephemeralProviderData.TokenCustomEndpoint = v })
	setStringField(providerConfig.WifFederatedTokenPath, func(v string) { ephemeralProviderData.ServiceAccountFederatedTokenPath = v })
	resp.EphemeralResourceData = ephemeralProviderData
}

// DataSources defines the data sources implemented in the provider.
func (p *Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	dataSources := []func() datasource.DataSource{
		alertGroup.NewAlertGroupDataSource,
		cdn.NewDistributionDataSource,
		cdnCustomDomain.NewCustomDomainDataSource,
		dnsZone.NewZoneDataSource,
		dnsRecordSet.NewRecordSetDataSource,
		edgeCloudInstances.NewInstancesDataSource,
		edgeCloudPlans.NewPlansDataSource,
		gitInstance.NewGitDataSource,
		iaasAffinityGroup.NewAffinityGroupDatasource,
		iaasImage.NewImageDataSource,
		iaasImageV2.NewImageV2DataSource,
		iaasNetwork.NewNetworkDataSource,
		iaasNetworkArea.NewNetworkAreaDataSource,
		iaasNetworkAreaRegion.NewNetworkAreaRegionDataSource,
		iaasNetworkAreaRoute.NewNetworkAreaRouteDataSource,
		iaasNetworkInterface.NewNetworkInterfaceDataSource,
		iaasVolume.NewVolumeDataSource,
		iaasProject.NewProjectDataSource,
		iaasPublicIp.NewPublicIpDataSource,
		iaasPublicIpRanges.NewPublicIpRangesDataSource,
		iaasKeyPair.NewKeyPairDataSource,
		iaasServer.NewServerDataSource,
		iaasSecurityGroup.NewSecurityGroupDataSource,
		iaasRoutingTable.NewRoutingTableDataSource,
		iaasRoutingTableRoute.NewRoutingTableRouteDataSource,
		iaasRoutingTables.NewRoutingTablesDataSource,
		iaasRoutingTableRoutes.NewRoutingTableRoutesDataSource,
		iaasSecurityGroupRule.NewSecurityGroupRuleDataSource,
		kmsKey.NewKeyDataSource,
		kmsKeyRing.NewKeyRingDataSource,
		kmsWrappingKey.NewWrappingKeyDataSource,
		loadBalancer.NewLoadBalancerDataSource,
		logMeInstance.NewInstanceDataSource,
		logMeCredential.NewCredentialDataSource,
		logsInstance.NewLogsInstanceDataSource,
		logsAccessToken.NewLogsAccessTokenDataSource,
		logAlertGroup.NewLogAlertGroupDataSource,
		machineType.NewMachineTypeDataSource,
		mariaDBInstance.NewInstanceDataSource,
		mariaDBCredential.NewCredentialDataSource,
		mongoDBFlexInstance.NewInstanceDataSource,
		mongoDBFlexUser.NewUserDataSource,
		objectStorageBucket.NewBucketDataSource,
		objecStorageCredentialsGroup.NewCredentialsGroupDataSource,
		objecStorageCredential.NewCredentialDataSource,
		observabilityInstance.NewInstanceDataSource,
		observabilityScrapeConfig.NewScrapeConfigDataSource,
		openSearchInstance.NewInstanceDataSource,
		openSearchCredential.NewCredentialDataSource,
		postgresFlexDatabase.NewDatabaseDataSource,
		postgresFlexInstance.NewInstanceDataSource,
		postgresFlexUser.NewUserDataSource,
		rabbitMQInstance.NewInstanceDataSource,
		rabbitMQCredential.NewCredentialDataSource,
		redisInstance.NewInstanceDataSource,
		redisCredential.NewCredentialDataSource,
		resourceManagerProject.NewProjectDataSource,
		scfOrganization.NewScfOrganizationDataSource,
		scfOrganizationmanager.NewScfOrganizationManagerDataSource,
		scfPlatform.NewScfPlatformDataSource,
		resourceManagerFolder.NewFolderDataSource,
		secretsManagerInstance.NewInstanceDataSource,
		secretsManagerUser.NewUserDataSource,
		sqlServerFlexInstance.NewInstanceDataSource,
		sqlServerFlexUser.NewUserDataSource,
		serverBackupSchedule.NewScheduleDataSource,
		serverBackupSchedule.NewSchedulesDataSource,
		serverUpdateSchedule.NewScheduleDataSource,
		serverUpdateSchedule.NewSchedulesDataSource,
		serviceAccount.NewServiceAccountDataSource,
		skeCluster.NewClusterDataSource,
		skeKubernetesVersion.NewKubernetesVersionsDataSource,
		skeMachineImages.NewKubernetesMachineImageVersionDataSource,
		resourcepool.NewResourcePoolDataSource,
		share.NewShareDataSource,
		exportpolicy.NewExportPolicyDataSource,
		snapshots.NewResourcePoolSnapshotDataSource,
	}
	dataSources = append(dataSources, customRole.NewCustomRoleDataSources()...)

	return dataSources
}

// Resources defines the resources implemented in the provider.
func (p *Provider) Resources(_ context.Context) []func() resource.Resource {
	resources := []func() resource.Resource{
		alertGroup.NewAlertGroupResource,
		cdn.NewDistributionResource,
		cdnCustomDomain.NewCustomDomainResource,
		dnsZone.NewZoneResource,
		dnsRecordSet.NewRecordSetResource,
		edgeCloudInstance.NewInstanceResource,
		edgeCloudKubeconfig.NewKubeconfigResource,
		edgeCloudToken.NewTokenResource,
		gitInstance.NewGitResource,
		iaasAffinityGroup.NewAffinityGroupResource,
		iaasImage.NewImageResource,
		iaasNetwork.NewNetworkResource,
		iaasNetworkArea.NewNetworkAreaResource,
		iaasNetworkAreaRegion.NewNetworkAreaRegionResource,
		iaasNetworkAreaRoute.NewNetworkAreaRouteResource,
		iaasNetworkInterface.NewNetworkInterfaceResource,
		iaasVolume.NewVolumeResource,
		iaasPublicIp.NewPublicIpResource,
		iaasKeyPair.NewKeyPairResource,
		iaasVolumeAttach.NewVolumeAttachResource,
		iaasNetworkInterfaceAttach.NewNetworkInterfaceAttachResource,
		iaasServiceAccountAttach.NewServiceAccountAttachResource,
		iaasPublicIpAssociate.NewPublicIpAssociateResource,
		iaasServer.NewServerResource,
		iaasSecurityGroup.NewSecurityGroupResource,
		iaasSecurityGroupRule.NewSecurityGroupRuleResource,
		iaasRoutingTable.NewRoutingTableResource,
		iaasRoutingTableRoute.NewRoutingTableRouteResource,
		kmsKey.NewKeyResource,
		kmsKeyRing.NewKeyRingResource,
		kmsWrappingKey.NewWrappingKeyResource,
		loadBalancer.NewLoadBalancerResource,
		loadBalancerObservabilityCredential.NewObservabilityCredentialResource,
		logMeInstance.NewInstanceResource,
		logMeCredential.NewCredentialResource,
		logAlertGroup.NewLogAlertGroupResource,
		logsInstance.NewLogsInstanceResource,
		logsAccessToken.NewLogsAccessTokenResource,
		mariaDBInstance.NewInstanceResource,
		mariaDBCredential.NewCredentialResource,
		modelServingToken.NewTokenResource,
		mongoDBFlexInstance.NewInstanceResource,
		mongoDBFlexUser.NewUserResource,
		objectStorageBucket.NewBucketResource,
		objecStorageCredentialsGroup.NewCredentialsGroupResource,
		objecStorageCredential.NewCredentialResource,
		observabilityCredential.NewCredentialResource,
		observabilityInstance.NewInstanceResource,
		observabilityScrapeConfig.NewScrapeConfigResource,
		openSearchInstance.NewInstanceResource,
		openSearchCredential.NewCredentialResource,
		postgresFlexDatabase.NewDatabaseResource,
		postgresFlexInstance.NewInstanceResource,
		postgresFlexUser.NewUserResource,
		rabbitMQInstance.NewInstanceResource,
		rabbitMQCredential.NewCredentialResource,
		redisInstance.NewInstanceResource,
		redisCredential.NewCredentialResource,
		resourceManagerProject.NewProjectResource,
		scfOrganization.NewScfOrganizationResource,
		scfOrganizationmanager.NewScfOrganizationManagerResource,
		resourceManagerFolder.NewFolderResource,
		secretsManagerInstance.NewInstanceResource,
		secretsManagerUser.NewUserResource,
		sqlServerFlexInstance.NewInstanceResource,
		sqlServerFlexUser.NewUserResource,
		serverBackupSchedule.NewScheduleResource,
		serverUpdateSchedule.NewScheduleResource,
		serviceAccount.NewServiceAccountResource,
		serviceAccountToken.NewServiceAccountTokenResource,
		serviceAccountKey.NewServiceAccountKeyResource,
		skeCluster.NewClusterResource,
		skeKubeconfig.NewKubeconfigResource,
		resourcepool.NewResourcePoolResource,
		share.NewShareResource,
		exportpolicy.NewExportPolicyResource,
	}
	resources = append(resources, roleAssignements.NewRoleAssignmentResources()...)
	resources = append(resources, customRole.NewCustomRoleResources()...)

	return resources
}

// EphemeralResources defines the ephemeral resources implemented in the provider.
func (p *Provider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		access_token.NewAccessTokenEphemeralResource,
	}
}

func githubAssertion(ctx context.Context, oidc_request_url, oidc_request_token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oidc_request_url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("githubAssertion: failed to build request: %+v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oidc_request_token))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("githubAssertion: cannot request token: %v", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("githubAssertion: cannot parse response: %v", err)
	}

	if c := resp.StatusCode; c < 200 || c > 299 {
		return "", fmt.Errorf("githubAssertion: received HTTP status %d with response: %s", resp.StatusCode, body)
	}

	var tokenRes struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &tokenRes); err != nil {
		return "", fmt.Errorf("githubAssertion: cannot unmarshal response: %v", err)
	}

	return tokenRes.Value, nil
}

// getEnvStringOrDefault takes a Framework StringValue and a corresponding Environment Variable name and returns
// either the string value set in the StringValue if not Null / Unknown _or_ the os.GetEnv() value of the Environment
// Variable provided. If both of these are empty, an empty string defaultValue is returned.
func getEnvStringOrDefault(val types.String, envVar string, defaultValue string) string {
	if val.IsNull() || val.IsUnknown() {
		if v := os.Getenv(envVar); v != "" {
			return os.Getenv(envVar)
		}
		return defaultValue
	}

	return val.ValueString()
}
