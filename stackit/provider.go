package stackit

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
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
	roleAssignements "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/authorization/roleassignments"
	cdnCustomDomain "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/customdomain"
	cdn "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/distribution"
	dnsRecordSet "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/recordset"
	dnsZone "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/zone"
	gitInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/git/instance"
	iaasAffinityGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/affinitygroup"
	iaasImage "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/image"
	iaasImageV2 "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/imagev2"
	iaasKeyPair "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/keypair"
	machineType "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/machinetype"
	iaasNetwork "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network"
	iaasNetworkArea "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkarea"
	iaasNetworkAreaRoute "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkarearoute"
	iaasNetworkInterface "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkinterface"
	iaasNetworkInterfaceAttach "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkinterfaceattach"
	iaasProject "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/project"
	iaasPublicIp "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/publicip"
	iaasPublicIpAssociate "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/publicipassociate"
	iaasPublicIpRanges "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/publicipranges"
	iaasSecurityGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/securitygroup"
	iaasSecurityGroupRule "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/securitygrouprule"
	iaasServer "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/server"
	iaasServiceAccountAttach "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/serviceaccountattach"
	iaasVolume "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/volume"
	iaasVolumeAttach "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/volumeattach"
	iaasalphaRoutingTableRoute "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/route"
	iaasalphaRoutingTableRoutes "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/routes"
	iaasalphaRoutingTable "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/table"
	iaasalphaRoutingTables "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/tables"
	kmsKey "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/key"
	kmsKeyRing "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/keyring"
	kmsWrappingKey "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/wrapping-key"
	loadBalancer "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/loadbalancer/loadbalancer"
	loadBalancerObservabilityCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/loadbalancer/observability-credential"
	logMeCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logme/credential"
	logMeInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logme/instance"
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
	skeCluster "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/cluster"
	skeKubeconfig "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/kubeconfig"
	sqlServerFlexInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/instance"
	sqlServerFlexUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/user"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &Provider{}
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
	ServiceAccountEmail   types.String `tfsdk:"service_account_email"` // Deprecated: ServiceAccountEmail is not required and will be removed after 12th June 2025
	ServiceAccountKey     types.String `tfsdk:"service_account_key"`
	ServiceAccountKeyPath types.String `tfsdk:"service_account_key_path"`
	PrivateKey            types.String `tfsdk:"private_key"`
	PrivateKeyPath        types.String `tfsdk:"private_key_path"`
	Token                 types.String `tfsdk:"service_account_token"`
	// Deprecated: Use DefaultRegion instead
	Region                          types.String `tfsdk:"region"`
	DefaultRegion                   types.String `tfsdk:"default_region"`
	CdnCustomEndpoint               types.String `tfsdk:"cdn_custom_endpoint"`
	DNSCustomEndpoint               types.String `tfsdk:"dns_custom_endpoint"`
	GitCustomEndpoint               types.String `tfsdk:"git_custom_endpoint"`
	IaaSCustomEndpoint              types.String `tfsdk:"iaas_custom_endpoint"`
	KMSCustomEndpoint               types.String `tfsdk:"kms_custom_endpoint"`
	PostgresFlexCustomEndpoint      types.String `tfsdk:"postgresflex_custom_endpoint"`
	MongoDBFlexCustomEndpoint       types.String `tfsdk:"mongodbflex_custom_endpoint"`
	ModelServingCustomEndpoint      types.String `tfsdk:"modelserving_custom_endpoint"`
	LoadBalancerCustomEndpoint      types.String `tfsdk:"loadbalancer_custom_endpoint"`
	LogMeCustomEndpoint             types.String `tfsdk:"logme_custom_endpoint"`
	RabbitMQCustomEndpoint          types.String `tfsdk:"rabbitmq_custom_endpoint"`
	MariaDBCustomEndpoint           types.String `tfsdk:"mariadb_custom_endpoint"`
	AuthorizationCustomEndpoint     types.String `tfsdk:"authorization_custom_endpoint"`
	ObjectStorageCustomEndpoint     types.String `tfsdk:"objectstorage_custom_endpoint"`
	ObservabilityCustomEndpoint     types.String `tfsdk:"observability_custom_endpoint"`
	OpenSearchCustomEndpoint        types.String `tfsdk:"opensearch_custom_endpoint"`
	RedisCustomEndpoint             types.String `tfsdk:"redis_custom_endpoint"`
	SecretsManagerCustomEndpoint    types.String `tfsdk:"secretsmanager_custom_endpoint"`
	SQLServerFlexCustomEndpoint     types.String `tfsdk:"sqlserverflex_custom_endpoint"`
	SKECustomEndpoint               types.String `tfsdk:"ske_custom_endpoint"`
	ServerBackupCustomEndpoint      types.String `tfsdk:"server_backup_custom_endpoint"`
	ServerUpdateCustomEndpoint      types.String `tfsdk:"server_update_custom_endpoint"`
	ServiceAccountCustomEndpoint    types.String `tfsdk:"service_account_custom_endpoint"`
	ResourceManagerCustomEndpoint   types.String `tfsdk:"resourcemanager_custom_endpoint"`
	ScfCustomEndpoint               types.String `tfsdk:"scf_custom_endpoint"`
	TokenCustomEndpoint             types.String `tfsdk:"token_custom_endpoint"`
	EnableBetaResources             types.Bool   `tfsdk:"enable_beta_resources"`
	ServiceEnablementCustomEndpoint types.String `tfsdk:"service_enablement_custom_endpoint"`
	Experiments                     types.List   `tfsdk:"experiments"`
}

// Schema defines the provider-level schema for configuration data.
func (p *Provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	descriptions := map[string]string{
		"credentials_path":                   "Path of JSON from where the credentials are read. Takes precedence over the env var `STACKIT_CREDENTIALS_PATH`. Default value is `~/.stackit/credentials.json`.",
		"service_account_token":              "Token used for authentication. If set, the token flow will be used to authenticate all operations.",
		"service_account_key_path":           "Path for the service account key used for authentication. If set, the key flow will be used to authenticate all operations.",
		"service_account_key":                "Service account key used for authentication. If set, the key flow will be used to authenticate all operations.",
		"private_key_path":                   "Path for the private RSA key used for authentication, relevant for the key flow. It takes precedence over the private key that is included in the service account key.",
		"private_key":                        "Private RSA key used for authentication, relevant for the key flow. It takes precedence over the private key that is included in the service account key.",
		"service_account_email":              "Service account email. It can also be set using the environment variable STACKIT_SERVICE_ACCOUNT_EMAIL. It is required if you want to use the resource manager project resource.",
		"region":                             "Region will be used as the default location for regional services. Not all services require a region, some are global",
		"default_region":                     "Region will be used as the default location for regional services. Not all services require a region, some are global",
		"cdn_custom_endpoint":                "Custom endpoint for the CDN service",
		"dns_custom_endpoint":                "Custom endpoint for the DNS service",
		"git_custom_endpoint":                "Custom endpoint for the Git service",
		"iaas_custom_endpoint":               "Custom endpoint for the IaaS service",
		"kms_custom_endpoint":                "Custom endpoint for the KMS service",
		"mongodbflex_custom_endpoint":        "Custom endpoint for the MongoDB Flex service",
		"modelserving_custom_endpoint":       "Custom endpoint for the AI Model Serving service",
		"loadbalancer_custom_endpoint":       "Custom endpoint for the Load Balancer service",
		"logme_custom_endpoint":              "Custom endpoint for the LogMe service",
		"rabbitmq_custom_endpoint":           "Custom endpoint for the RabbitMQ service",
		"mariadb_custom_endpoint":            "Custom endpoint for the MariaDB service",
		"authorization_custom_endpoint":      "Custom endpoint for the Membership service",
		"objectstorage_custom_endpoint":      "Custom endpoint for the Object Storage service",
		"observability_custom_endpoint":      "Custom endpoint for the Observability service",
		"opensearch_custom_endpoint":         "Custom endpoint for the OpenSearch service",
		"postgresflex_custom_endpoint":       "Custom endpoint for the PostgresFlex service",
		"redis_custom_endpoint":              "Custom endpoint for the Redis service",
		"server_backup_custom_endpoint":      "Custom endpoint for the Server Backup service",
		"server_update_custom_endpoint":      "Custom endpoint for the Server Update service",
		"service_account_custom_endpoint":    "Custom endpoint for the Service Account service",
		"resourcemanager_custom_endpoint":    "Custom endpoint for the Resource Manager service",
		"scf_custom_endpoint":                "Custom endpoint for the Cloud Foundry (SCF) service",
		"secretsmanager_custom_endpoint":     "Custom endpoint for the Secrets Manager service",
		"sqlserverflex_custom_endpoint":      "Custom endpoint for the SQL Server Flex service",
		"ske_custom_endpoint":                "Custom endpoint for the Kubernetes Engine (SKE) service",
		"service_enablement_custom_endpoint": "Custom endpoint for the Service Enablement API",
		"token_custom_endpoint":              "Custom endpoint for the token API, which is used to request access tokens when using the key flow",
		"enable_beta_resources":              "Enable beta resources. Default is false.",
		"experiments":                        fmt.Sprintf("Enables experiments. These are unstable features without official support. More information can be found in the README. Available Experiments: %v", strings.Join(features.AvailableExperiments, ", ")),
	}

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"credentials_path": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["credentials_path"],
			},
			"service_account_email": schema.StringAttribute{
				Optional:           true,
				Description:        descriptions["service_account_email"],
				DeprecationMessage: "The `service_account_email` field has been deprecated because it is not required. Will be removed after June 12th 2025.",
			},
			"service_account_token": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["service_account_token"],
				DeprecationMessage: "Authentication via Service Account Token is deprecated and will be removed on December 17, 2025. " +
					"Please use `service_account_key` or `service_account_key_path` instead. " +
					"For a smooth transition, refer to our migration guide: https://docs.stackit.cloud/stackit/en/deprecation-plan-for-service-account-access-tokens-and-migration-guide-373293307.html",
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
			"cdn_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["cdn_custom_endpoint"],
			},
			"dns_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["dns_custom_endpoint"],
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
			"token_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["token_custom_endpoint"],
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
	setStringField(providerConfig.ServiceAccountKey, func(v string) { sdkConfig.ServiceAccountKey = v })
	setStringField(providerConfig.ServiceAccountKeyPath, func(v string) { sdkConfig.ServiceAccountKeyPath = v })
	setStringField(providerConfig.PrivateKey, func(v string) { sdkConfig.PrivateKey = v })
	setStringField(providerConfig.PrivateKeyPath, func(v string) { sdkConfig.PrivateKeyPath = v })
	setStringField(providerConfig.Token, func(v string) { sdkConfig.Token = v })
	setStringField(providerConfig.TokenCustomEndpoint, func(v string) { sdkConfig.TokenCustomUrl = v })

	// Provider Data Configuration
	setStringField(providerConfig.DefaultRegion, func(v string) { providerData.DefaultRegion = v })
	setStringField(providerConfig.Region, func(v string) { providerData.Region = v }) // nolint:staticcheck // preliminary handling of deprecated attribute
	setStringField(providerConfig.CdnCustomEndpoint, func(v string) { providerData.CdnCustomEndpoint = v })
	setStringField(providerConfig.DNSCustomEndpoint, func(v string) { providerData.DnsCustomEndpoint = v })
	setStringField(providerConfig.GitCustomEndpoint, func(v string) { providerData.GitCustomEndpoint = v })
	setStringField(providerConfig.IaaSCustomEndpoint, func(v string) { providerData.IaaSCustomEndpoint = v })
	setStringField(providerConfig.PostgresFlexCustomEndpoint, func(v string) { providerData.PostgresFlexCustomEndpoint = v })
	setStringField(providerConfig.KMSCustomEndpoint, func(v string) { providerData.KMSCustomEndpoint = v })
	setStringField(providerConfig.ModelServingCustomEndpoint, func(v string) { providerData.ModelServingCustomEndpoint = v })
	setStringField(providerConfig.MongoDBFlexCustomEndpoint, func(v string) { providerData.MongoDBFlexCustomEndpoint = v })
	setStringField(providerConfig.LoadBalancerCustomEndpoint, func(v string) { providerData.LoadBalancerCustomEndpoint = v })
	setStringField(providerConfig.LogMeCustomEndpoint, func(v string) { providerData.LogMeCustomEndpoint = v })
	setStringField(providerConfig.RabbitMQCustomEndpoint, func(v string) { providerData.RabbitMQCustomEndpoint = v })
	setStringField(providerConfig.MariaDBCustomEndpoint, func(v string) { providerData.MariaDBCustomEndpoint = v })
	setStringField(providerConfig.AuthorizationCustomEndpoint, func(v string) { providerData.AuthorizationCustomEndpoint = v })
	setStringField(providerConfig.ObjectStorageCustomEndpoint, func(v string) { providerData.ObjectStorageCustomEndpoint = v })
	setStringField(providerConfig.ObservabilityCustomEndpoint, func(v string) { providerData.ObservabilityCustomEndpoint = v })
	setStringField(providerConfig.OpenSearchCustomEndpoint, func(v string) { providerData.OpenSearchCustomEndpoint = v })
	setStringField(providerConfig.RedisCustomEndpoint, func(v string) { providerData.RedisCustomEndpoint = v })
	setStringField(providerConfig.ResourceManagerCustomEndpoint, func(v string) { providerData.ResourceManagerCustomEndpoint = v })
	setStringField(providerConfig.ScfCustomEndpoint, func(v string) { providerData.ScfCustomEndpoint = v })
	setStringField(providerConfig.SecretsManagerCustomEndpoint, func(v string) { providerData.SecretsManagerCustomEndpoint = v })
	setStringField(providerConfig.SQLServerFlexCustomEndpoint, func(v string) { providerData.SQLServerFlexCustomEndpoint = v })
	setStringField(providerConfig.ServiceAccountCustomEndpoint, func(v string) { providerData.ServiceAccountCustomEndpoint = v })
	setStringField(providerConfig.SKECustomEndpoint, func(v string) { providerData.SKECustomEndpoint = v })
	setStringField(providerConfig.ServiceEnablementCustomEndpoint, func(v string) { providerData.ServiceEnablementCustomEndpoint = v })
	setBoolField(providerConfig.EnableBetaResources, func(v bool) { providerData.EnableBetaResources = v })

	if !(providerConfig.Experiments.IsUnknown() || providerConfig.Experiments.IsNull()) {
		var experimentValues []string
		diags := providerConfig.Experiments.ElementsAs(ctx, &experimentValues, false)
		if diags.HasError() {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring provider", fmt.Sprintf("Setting up experiments: %v", diags.Errors()))
		}
		providerData.Experiments = experimentValues
	}

	roundTripper, err := sdkauth.SetupAuth(sdkConfig)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring provider", fmt.Sprintf("Setting up authentication: %v", err))
		return
	}

	// Make round tripper and custom endpoints available during DataSource and Resource
	// type Configure methods.
	providerData.RoundTripper = roundTripper
	resp.DataSourceData = providerData
	resp.ResourceData = providerData

	providerData.Version = p.version
}

// DataSources defines the data sources implemented in the provider.
func (p *Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		alertGroup.NewAlertGroupDataSource,
		cdn.NewDistributionDataSource,
		cdnCustomDomain.NewCustomDomainDataSource,
		dnsZone.NewZoneDataSource,
		dnsRecordSet.NewRecordSetDataSource,
		gitInstance.NewGitDataSource,
		iaasAffinityGroup.NewAffinityGroupDatasource,
		iaasImage.NewImageDataSource,
		iaasImageV2.NewImageV2DataSource,
		iaasNetwork.NewNetworkDataSource,
		iaasNetworkArea.NewNetworkAreaDataSource,
		iaasNetworkAreaRoute.NewNetworkAreaRouteDataSource,
		iaasNetworkInterface.NewNetworkInterfaceDataSource,
		iaasVolume.NewVolumeDataSource,
		iaasProject.NewProjectDataSource,
		iaasPublicIp.NewPublicIpDataSource,
		iaasPublicIpRanges.NewPublicIpRangesDataSource,
		iaasKeyPair.NewKeyPairDataSource,
		iaasServer.NewServerDataSource,
		iaasSecurityGroup.NewSecurityGroupDataSource,
		iaasalphaRoutingTable.NewRoutingTableDataSource,
		iaasalphaRoutingTableRoute.NewRoutingTableRouteDataSource,
		iaasalphaRoutingTables.NewRoutingTablesDataSource,
		iaasalphaRoutingTableRoutes.NewRoutingTableRoutesDataSource,
		iaasSecurityGroupRule.NewSecurityGroupRuleDataSource,
		kmsKey.NewKeyDataSource,
		kmsKeyRing.NewKeyRingDataSource,
		kmsWrappingKey.NewWrappingKeyDataSource,
		loadBalancer.NewLoadBalancerDataSource,
		logMeInstance.NewInstanceDataSource,
		logMeCredential.NewCredentialDataSource,
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
	}
}

// Resources defines the resources implemented in the provider.
func (p *Provider) Resources(_ context.Context) []func() resource.Resource {
	resources := []func() resource.Resource{
		alertGroup.NewAlertGroupResource,
		cdn.NewDistributionResource,
		cdnCustomDomain.NewCustomDomainResource,
		dnsZone.NewZoneResource,
		dnsRecordSet.NewRecordSetResource,
		gitInstance.NewGitResource,
		iaasAffinityGroup.NewAffinityGroupResource,
		iaasImage.NewImageResource,
		iaasNetwork.NewNetworkResource,
		iaasNetworkArea.NewNetworkAreaResource,
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
		iaasalphaRoutingTable.NewRoutingTableResource,
		iaasalphaRoutingTableRoute.NewRoutingTableRouteResource,
		kmsKey.NewKeyResource,
		kmsKeyRing.NewKeyRingResource,
		kmsWrappingKey.NewWrappingKeyResource,
		loadBalancer.NewLoadBalancerResource,
		loadBalancerObservabilityCredential.NewObservabilityCredentialResource,
		logMeInstance.NewInstanceResource,
		logMeCredential.NewCredentialResource,
		logAlertGroup.NewLogAlertGroupResource,
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
	}
	resources = append(resources, roleAssignements.NewRoleAssignmentResources()...)

	return resources
}
