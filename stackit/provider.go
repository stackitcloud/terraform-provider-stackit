package stackit

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	argusCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/argus/credential"
	argusInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/argus/instance"
	argusScrapeConfig "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/argus/scrapeconfig"
	roleassignments "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/authorization/roleassignments"
	dnsRecordSet "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/recordset"
	dnsZone "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/zone"
	iaasAffinityGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/affinitygroup"
	iaasImage "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/image"
	iaasKeyPair "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/keypair"
	iaasNetwork "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network"
	iaasNetworkArea "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkarea"
	iaasNetworkAreaRoute "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkarearoute"
	iaasNetworkInterface "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkinterface"
	iaasNetworkInterfaceAttach "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/networkinterfaceattach"
	iaasPublicIp "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/publicip"
	iaasPublicIpAssociate "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/publicipassociate"
	iaasPublicIpRanges "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/publicipranges"
	iaasSecurityGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/securitygroup"
	iaasSecurityGroupRule "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/securitygrouprule"
	iaasServer "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/server"
	iaasServiceAccountAttach "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/serviceaccountattach"
	iaasVolume "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/volume"
	iaasVolumeAttach "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/volumeattach"
	loadBalancerCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/loadbalancer/credential"
	loadBalancer "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/loadbalancer/loadbalancer"
	loadBalancerObservabilityCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/loadbalancer/observability-credential"
	logMeCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logme/credential"
	logMeInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logme/instance"
	mariaDBCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mariadb/credential"
	mariaDBInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mariadb/instance"
	mongoDBFlexInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mongodbflex/instance"
	mongoDBFlexUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mongodbflex/user"
	objectStorageBucket "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/bucket"
	objecStorageCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/credential"
	objecStorageCredentialsGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/credentialsgroup"
	observabilityCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/credential"
	observabilityInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/instance"
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
	resourceManagerProject "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/project"
	secretsManagerInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/secretsmanager/instance"
	secretsManagerUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/secretsmanager/user"
	serverBackupSchedule "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serverbackup/schedule"
	serverUpdateSchedule "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serverupdate/schedule"
	serviceAccount "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/account"
	serviceAccountToken "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/token"
	skeCluster "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/cluster"
	skeKubeconfig "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/kubeconfig"
	skeProject "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/project"
	sqlServerFlexInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/instance"
	sqlServerFlexUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/user"

	sdkauth "github.com/stackitcloud/stackit-sdk-go/core/auth"
	"github.com/stackitcloud/stackit-sdk-go/core/config"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
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
	ArgusCustomEndpoint             types.String `tfsdk:"argus_custom_endpoint"`
	DNSCustomEndpoint               types.String `tfsdk:"dns_custom_endpoint"`
	IaaSCustomEndpoint              types.String `tfsdk:"iaas_custom_endpoint"`
	PostgresFlexCustomEndpoint      types.String `tfsdk:"postgresflex_custom_endpoint"`
	MongoDBFlexCustomEndpoint       types.String `tfsdk:"mongodbflex_custom_endpoint"`
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
		"argus_custom_endpoint":              "Custom endpoint for the Argus service",
		"dns_custom_endpoint":                "Custom endpoint for the DNS service",
		"iaas_custom_endpoint":               "Custom endpoint for the IaaS service",
		"mongodbflex_custom_endpoint":        "Custom endpoint for the MongoDB Flex service",
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
		"secretsmanager_custom_endpoint":     "Custom endpoint for the Secrets Manager service",
		"sqlserverflex_custom_endpoint":      "Custom endpoint for the SQL Server Flex service",
		"ske_custom_endpoint":                "Custom endpoint for the Kubernetes Engine (SKE) service",
		"service_enablement_custom_endpoint": "Custom endpoint for the Service Enablement API",
		"token_custom_endpoint":              "Custom endpoint for the token API, which is used to request access tokens when using the key flow",
		"enable_beta_resources":              "Enable beta resources. Default is false.",
		"experiments":                        fmt.Sprintf("Enables experiments. These are unstable features without official support. More information can be found in the README. Available Experiments: %v", features.AvailableExperiments),
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
			"argus_custom_endpoint": schema.StringAttribute{
				Optional:           true,
				Description:        descriptions["argus_custom_endpoint"],
				DeprecationMessage: "Argus service has been deprecated and integration will be removed after February 26th 2025. Please use `observability_custom_endpoint` and `observability` resources instead, which offer the exact same functionality.",
			},
			"dns_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["dns_custom_endpoint"],
			},
			"iaas_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["iaas_custom_endpoint"],
			},
			"postgresflex_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["postgresflex_custom_endpoint"],
			},
			"mariadb_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["mariadb_custom_endpoint"],
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
	if !(providerConfig.CredentialsFilePath.IsUnknown() || providerConfig.CredentialsFilePath.IsNull()) {
		sdkConfig.CredentialsFilePath = providerConfig.CredentialsFilePath.ValueString()
	}
	if !(providerConfig.ServiceAccountKey.IsUnknown() || providerConfig.ServiceAccountKey.IsNull()) {
		sdkConfig.ServiceAccountKey = providerConfig.ServiceAccountKey.ValueString()
	}
	if !(providerConfig.ServiceAccountKeyPath.IsUnknown() || providerConfig.ServiceAccountKeyPath.IsNull()) {
		sdkConfig.ServiceAccountKeyPath = providerConfig.ServiceAccountKeyPath.ValueString()
	}
	if !(providerConfig.PrivateKey.IsUnknown() || providerConfig.PrivateKey.IsNull()) {
		sdkConfig.PrivateKey = providerConfig.PrivateKey.ValueString()
	}
	if !(providerConfig.PrivateKeyPath.IsUnknown() || providerConfig.PrivateKeyPath.IsNull()) {
		sdkConfig.PrivateKeyPath = providerConfig.PrivateKeyPath.ValueString()
	}
	if !(providerConfig.Token.IsUnknown() || providerConfig.Token.IsNull()) {
		sdkConfig.Token = providerConfig.Token.ValueString()
	}
	if !(providerConfig.DefaultRegion.IsUnknown() || providerConfig.DefaultRegion.IsNull()) {
		providerData.DefaultRegion = providerConfig.DefaultRegion.ValueString()
	} else if !(providerConfig.Region.IsUnknown() || providerConfig.Region.IsNull()) { // nolint:staticcheck // preliminary handling of deprecated attribute
		providerData.Region = providerConfig.Region.ValueString() // nolint:staticcheck // preliminary handling of deprecated attribute
	}
	if !(providerConfig.DNSCustomEndpoint.IsUnknown() || providerConfig.DNSCustomEndpoint.IsNull()) {
		providerData.DnsCustomEndpoint = providerConfig.DNSCustomEndpoint.ValueString()
	}
	if !(providerConfig.IaaSCustomEndpoint.IsUnknown() || providerConfig.IaaSCustomEndpoint.IsNull()) {
		providerData.IaaSCustomEndpoint = providerConfig.IaaSCustomEndpoint.ValueString()
	}
	if !(providerConfig.PostgresFlexCustomEndpoint.IsUnknown() || providerConfig.PostgresFlexCustomEndpoint.IsNull()) {
		providerData.PostgresFlexCustomEndpoint = providerConfig.PostgresFlexCustomEndpoint.ValueString()
	}
	if !(providerConfig.MongoDBFlexCustomEndpoint.IsUnknown() || providerConfig.MongoDBFlexCustomEndpoint.IsNull()) {
		providerData.MongoDBFlexCustomEndpoint = providerConfig.MongoDBFlexCustomEndpoint.ValueString()
	}
	if !(providerConfig.LoadBalancerCustomEndpoint.IsUnknown() || providerConfig.LoadBalancerCustomEndpoint.IsNull()) {
		providerData.LoadBalancerCustomEndpoint = providerConfig.LoadBalancerCustomEndpoint.ValueString()
	}
	if !(providerConfig.LogMeCustomEndpoint.IsUnknown() || providerConfig.LogMeCustomEndpoint.IsNull()) {
		providerData.LogMeCustomEndpoint = providerConfig.LogMeCustomEndpoint.ValueString()
	}
	if !(providerConfig.RabbitMQCustomEndpoint.IsUnknown() || providerConfig.RabbitMQCustomEndpoint.IsNull()) {
		providerData.RabbitMQCustomEndpoint = providerConfig.RabbitMQCustomEndpoint.ValueString()
	}
	if !(providerConfig.MariaDBCustomEndpoint.IsUnknown() || providerConfig.MariaDBCustomEndpoint.IsNull()) {
		providerData.MariaDBCustomEndpoint = providerConfig.MariaDBCustomEndpoint.ValueString()
	}
	if !(providerConfig.AuthorizationCustomEndpoint.IsUnknown() || providerConfig.AuthorizationCustomEndpoint.IsNull()) {
		providerData.AuthorizationCustomEndpoint = providerConfig.AuthorizationCustomEndpoint.ValueString()
	}
	if !(providerConfig.ObjectStorageCustomEndpoint.IsUnknown() || providerConfig.ObjectStorageCustomEndpoint.IsNull()) {
		providerData.ObjectStorageCustomEndpoint = providerConfig.ObjectStorageCustomEndpoint.ValueString()
	}
	if !(providerConfig.ObservabilityCustomEndpoint.IsUnknown() || providerConfig.ObservabilityCustomEndpoint.IsNull()) {
		providerData.ObservabilityCustomEndpoint = providerConfig.ObservabilityCustomEndpoint.ValueString()
	}
	if !(providerConfig.OpenSearchCustomEndpoint.IsUnknown() || providerConfig.OpenSearchCustomEndpoint.IsNull()) {
		providerData.OpenSearchCustomEndpoint = providerConfig.OpenSearchCustomEndpoint.ValueString()
	}
	if !(providerConfig.RedisCustomEndpoint.IsUnknown() || providerConfig.RedisCustomEndpoint.IsNull()) {
		providerData.RedisCustomEndpoint = providerConfig.RedisCustomEndpoint.ValueString()
	}
	if !(providerConfig.ResourceManagerCustomEndpoint.IsUnknown() || providerConfig.ResourceManagerCustomEndpoint.IsNull()) {
		providerData.ResourceManagerCustomEndpoint = providerConfig.ResourceManagerCustomEndpoint.ValueString()
	}
	if !(providerConfig.SecretsManagerCustomEndpoint.IsUnknown() || providerConfig.SecretsManagerCustomEndpoint.IsNull()) {
		providerData.SecretsManagerCustomEndpoint = providerConfig.SecretsManagerCustomEndpoint.ValueString()
	}
	if !(providerConfig.SQLServerFlexCustomEndpoint.IsUnknown() || providerConfig.SQLServerFlexCustomEndpoint.IsNull()) {
		providerData.SQLServerFlexCustomEndpoint = providerConfig.SQLServerFlexCustomEndpoint.ValueString()
	}
	if !(providerConfig.ServiceAccountCustomEndpoint.IsUnknown() || providerConfig.ServiceAccountCustomEndpoint.IsNull()) {
		providerData.ServiceAccountCustomEndpoint = providerConfig.ServiceAccountCustomEndpoint.ValueString()
	}
	if !(providerConfig.SKECustomEndpoint.IsUnknown() || providerConfig.SKECustomEndpoint.IsNull()) {
		providerData.SKECustomEndpoint = providerConfig.SKECustomEndpoint.ValueString()
	}
	if !(providerConfig.ServiceEnablementCustomEndpoint.IsUnknown() || providerConfig.ServiceEnablementCustomEndpoint.IsNull()) {
		providerData.ServiceEnablementCustomEndpoint = providerConfig.ServiceEnablementCustomEndpoint.ValueString()
	}
	if !(providerConfig.TokenCustomEndpoint.IsUnknown() || providerConfig.TokenCustomEndpoint.IsNull()) {
		sdkConfig.TokenCustomUrl = providerConfig.TokenCustomEndpoint.ValueString()
	}
	if !(providerConfig.EnableBetaResources.IsUnknown() || providerConfig.EnableBetaResources.IsNull()) {
		providerData.EnableBetaResources = providerConfig.EnableBetaResources.ValueBool()
	}
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
}

// DataSources defines the data sources implemented in the provider.
func (p *Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		argusInstance.NewInstanceDataSource,
		argusScrapeConfig.NewScrapeConfigDataSource,
		dnsZone.NewZoneDataSource,
		dnsRecordSet.NewRecordSetDataSource,
		iaasAffinityGroup.NewAffinityGroupDatasource,
		iaasImage.NewImageDataSource,
		iaasNetwork.NewNetworkDataSource,
		iaasNetworkArea.NewNetworkAreaDataSource,
		iaasNetworkAreaRoute.NewNetworkAreaRouteDataSource,
		iaasNetworkInterface.NewNetworkInterfaceDataSource,
		iaasVolume.NewVolumeDataSource,
		iaasPublicIp.NewPublicIpDataSource,
		iaasPublicIpRanges.NewPublicIpRangesDataSource,
		iaasKeyPair.NewKeyPairDataSource,
		iaasServer.NewServerDataSource,
		iaasSecurityGroup.NewSecurityGroupDataSource,
		iaasSecurityGroupRule.NewSecurityGroupRuleDataSource,
		loadBalancer.NewLoadBalancerDataSource,
		logMeInstance.NewInstanceDataSource,
		logMeCredential.NewCredentialDataSource,
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
		secretsManagerInstance.NewInstanceDataSource,
		secretsManagerUser.NewUserDataSource,
		sqlServerFlexInstance.NewInstanceDataSource,
		sqlServerFlexUser.NewUserDataSource,
		serverBackupSchedule.NewScheduleDataSource,
		serverBackupSchedule.NewSchedulesDataSource,
		serverUpdateSchedule.NewScheduleDataSource,
		serverUpdateSchedule.NewSchedulesDataSource,
		serviceAccount.NewServiceAccountDataSource,
		skeProject.NewProjectDataSource,
		skeCluster.NewClusterDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *Provider) Resources(_ context.Context) []func() resource.Resource {
	resources := []func() resource.Resource{
		argusCredential.NewCredentialResource,
		argusInstance.NewInstanceResource,
		argusScrapeConfig.NewScrapeConfigResource,
		dnsZone.NewZoneResource,
		dnsRecordSet.NewRecordSetResource,
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
		loadBalancer.NewLoadBalancerResource,
		loadBalancerCredential.NewCredentialResource,
		loadBalancerObservabilityCredential.NewObservabilityCredentialResource,
		logMeInstance.NewInstanceResource,
		logMeCredential.NewCredentialResource,
		mariaDBInstance.NewInstanceResource,
		mariaDBCredential.NewCredentialResource,
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
		secretsManagerInstance.NewInstanceResource,
		secretsManagerUser.NewUserResource,
		sqlServerFlexInstance.NewInstanceResource,
		sqlServerFlexUser.NewUserResource,
		serverBackupSchedule.NewScheduleResource,
		serverUpdateSchedule.NewScheduleResource,
		serviceAccount.NewServiceAccountResource,
		serviceAccountToken.NewServiceAccountTokenResource,
		skeProject.NewProjectResource,
		skeCluster.NewClusterResource,
		skeKubeconfig.NewKubeconfigResource,
	}
	resources = append(resources, roleassignments.NewRoleAssignmentResources()...)

	return resources
}
