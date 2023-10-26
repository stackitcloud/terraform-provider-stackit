package stackit

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	argusCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/argus/credential"
	argusInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/argus/instance"
	argusScrapeConfig "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/argus/scrapeconfig"
	dnsRecordSet "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/recordset"
	dnsZone "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/zone"
	logMeCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logme/credential"
	logMeInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logme/instance"
	mariaDBCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mariadb/credential"
	mariaDBInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mariadb/instance"
	mongoDBFlexInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mongodbflex/instance"
	mongoDBFlexUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mongodbflex/user"
	objectStorageBucket "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/bucket"
	objecStorageCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/credential"
	objecStorageCredentialsGroup "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/credentialsgroup"
	openSearchCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/opensearch/credential"
	openSearchInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/opensearch/instance"
	postgresFlexInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresflex/instance"
	postgresFlexUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresflex/user"
	postgresCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresql/credential"
	postgresInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresql/instance"
	rabbitMQCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/rabbitmq/credential"
	rabbitMQInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/rabbitmq/instance"
	redisCredential "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/redis/credential"
	redisInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/redis/instance"
	resourceManagerProject "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/project"
	secretsManagerInstance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/secretsmanager/instance"
	secretsManagerUser "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/secretsmanager/user"
	skeCluster "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/cluster"
	skeProject "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/project"

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
	CredentialsFilePath           types.String `tfsdk:"credentials_path"`
	ServiceAccountEmail           types.String `tfsdk:"service_account_email"`
	ServiceAccountKey             types.String `tfsdk:"service_account_key"`
	ServiceAccountKeyPath         types.String `tfsdk:"service_account_key_path"`
	PrivateKey                    types.String `tfsdk:"private_key"`
	PrivateKeyPath                types.String `tfsdk:"private_key_path"`
	Token                         types.String `tfsdk:"service_account_token"`
	Region                        types.String `tfsdk:"region"`
	DNSCustomEndpoint             types.String `tfsdk:"dns_custom_endpoint"`
	PostgreSQLCustomEndpoint      types.String `tfsdk:"postgresql_custom_endpoint"`
	PostgresFlexCustomEndpoint    types.String `tfsdk:"postgresflex_custom_endpoint"`
	MongoDBFlexCustomEndpoint     types.String `tfsdk:"mongodbflex_custom_endpoint"`
	LoadBalancerCustomEndpoint    types.String `tfsdk:"loadbalancer_custom_endpoint"`
	LogMeCustomEndpoint           types.String `tfsdk:"logme_custom_endpoint"`
	RabbitMQCustomEndpoint        types.String `tfsdk:"rabbitmq_custom_endpoint"`
	MariaDBCustomEndpoint         types.String `tfsdk:"mariadb_custom_endpoint"`
	ObjectStorageCustomEndpoint   types.String `tfsdk:"objectstorage_custom_endpoint"`
	OpenSearchCustomEndpoint      types.String `tfsdk:"opensearch_custom_endpoint"`
	RedisCustomEndpoint           types.String `tfsdk:"redis_custom_endpoint"`
	SecretsManagerCustomEndpoint  types.String `tfsdk:"secretsmanager_custom_endpoint"`
	ArgusCustomEndpoint           types.String `tfsdk:"argus_custom_endpoint"`
	SKECustomEndpoint             types.String `tfsdk:"ske_custom_endpoint"`
	ResourceManagerCustomEndpoint types.String `tfsdk:"resourcemanager_custom_endpoint"`
	TokenCustomEndpoint           types.String `tfsdk:"token_custom_endpoint"`
	JWKSCustomEndpoint            types.String `tfsdk:"jwks_custom_endpoint"`
}

// Schema defines the provider-level schema for configuration data.
func (p *Provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	descriptions := map[string]string{
		"credentials_path":                "Path of JSON from where the credentials are read. Takes precedence over the env var `STACKIT_CREDENTIALS_PATH`. Default value is `~/.stackit/credentials.json`.",
		"service_account_token":           "Token used for authentication. If set, the token flow will be used to authenticate all operations.",
		"service_account_key_path":        "Path for the service account key used for authentication. If set alongside the private key, the key flow will be used to authenticate all operations.",
		"service_account_key":             "Service account key used for authentication. If set alongside private key, the key flow will be used to authenticate all operations.",
		"private_key_path":                "Path for the private RSA key used for authentication. If set alongside the service account key, the key flow will be used to authenticate all operations.",
		"private_key":                     "Private RSA key used for authentication. If set alongside the service account key, the key flow will be used to authenticate all operations.",
		"service_account_email":           "Service account email. It can also be set using the environment variable STACKIT_SERVICE_ACCOUNT_EMAIL",
		"region":                          "Region will be used as the default location for regional services. Not all services require a region, some are global",
		"dns_custom_endpoint":             "Custom endpoint for the DNS service",
		"postgresql_custom_endpoint":      "Custom endpoint for the PostgreSQL service",
		"postgresflex_custom_endpoint":    "Custom endpoint for the PostgresFlex service",
		"mongodbflex_custom_endpoint":     "Custom endpoint for the MongoDB Flex service",
		"logme_custom_endpoint":           "Custom endpoint for the LogMe service",
		"rabbitmq_custom_endpoint":        "Custom endpoint for the RabbitMQ service",
		"mariadb_custom_endpoint":         "Custom endpoint for the MariaDB service",
		"objectstorage_custom_endpoint":   "Custom endpoint for the Object Storage service",
		"opensearch_custom_endpoint":      "Custom endpoint for the OpenSearch service",
		"argus_custom_endpoint":           "Custom endpoint for the Argus service",
		"ske_custom_endpoint":             "Custom endpoint for the Kubernetes Engine (SKE) service",
		"resourcemanager_custom_endpoint": "Custom endpoint for the Resource Manager service",
		"secretsmanager_custom_endpoint":  "Custom endpoint for the Secrets Manager service",
		"token_custom_endpoint":           "Custom endpoint for the token API, which is used to request access tokens when using the key flow",
		"jwks_custom_endpoint":            "Custom endpoint for the jwks API, which is used to get the json web key sets (jwks) to validate tokens when using the key flow",
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
				Optional:    true,
				Description: descriptions["region"],
			},
			"dns_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["dns_custom_endpoint"],
			},
			"postgresql_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["postgresql_custom_endpoint"],
			},
			"postgresflex_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["postgresflex_custom_endpoint"],
			},
			"mongodbflex_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["mongodbflex_custom_endpoint"],
			},
			"logme_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["logme_custom_endpoint"],
			},
			"rabbitmq_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["rabbitmq_custom_endpoint"],
			},
			"mariadb_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["mariadb_custom_endpoint"],
			},
			"objectstorage_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["objectstorage_custom_endpoint"],
			},
			"opensearch_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["opensearch_custom_endpoint"],
			},
			"redis_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["redis_custom_endpoint"],
			},
			"secretsmanager_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["secretsmanager_custom_endpoint"],
			},
			"argus_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["argus_custom_endpoint"],
			},
			"ske_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["ske_custom_endpoint"],
			},
			"resourcemanager_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["resourcemanager_custom_endpoint"],
			},
			"token_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["token_custom_endpoint"],
			},
			"jwks_custom_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["jwks_custom_endpoint"],
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
	if !(providerConfig.ServiceAccountEmail.IsUnknown() || providerConfig.ServiceAccountEmail.IsNull()) {
		providerData.ServiceAccountEmail = providerConfig.ServiceAccountEmail.ValueString()
		sdkConfig.ServiceAccountEmail = providerConfig.ServiceAccountEmail.ValueString()
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
	if !(providerConfig.Region.IsUnknown() || providerConfig.Region.IsNull()) {
		providerData.Region = providerConfig.Region.ValueString()
	}
	if !(providerConfig.DNSCustomEndpoint.IsUnknown() || providerConfig.DNSCustomEndpoint.IsNull()) {
		providerData.DnsCustomEndpoint = providerConfig.DNSCustomEndpoint.ValueString()
	}
	if !(providerConfig.PostgreSQLCustomEndpoint.IsUnknown() || providerConfig.PostgreSQLCustomEndpoint.IsNull()) {
		providerData.PostgreSQLCustomEndpoint = providerConfig.PostgreSQLCustomEndpoint.ValueString()
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
	if !(providerConfig.ObjectStorageCustomEndpoint.IsUnknown() || providerConfig.ObjectStorageCustomEndpoint.IsNull()) {
		providerData.ObjectStorageCustomEndpoint = providerConfig.ObjectStorageCustomEndpoint.ValueString()
	}
	if !(providerConfig.OpenSearchCustomEndpoint.IsUnknown() || providerConfig.OpenSearchCustomEndpoint.IsNull()) {
		providerData.OpenSearchCustomEndpoint = providerConfig.OpenSearchCustomEndpoint.ValueString()
	}
	if !(providerConfig.RedisCustomEndpoint.IsUnknown() || providerConfig.RedisCustomEndpoint.IsNull()) {
		providerData.RedisCustomEndpoint = providerConfig.RedisCustomEndpoint.ValueString()
	}
	if !(providerConfig.ArgusCustomEndpoint.IsUnknown() || providerConfig.ArgusCustomEndpoint.IsNull()) {
		providerData.ArgusCustomEndpoint = providerConfig.ArgusCustomEndpoint.ValueString()
	}
	if !(providerConfig.SKECustomEndpoint.IsUnknown() || providerConfig.SKECustomEndpoint.IsNull()) {
		providerData.SKECustomEndpoint = providerConfig.SKECustomEndpoint.ValueString()
	}
	if !(providerConfig.ResourceManagerCustomEndpoint.IsUnknown() || providerConfig.ResourceManagerCustomEndpoint.IsNull()) {
		providerData.ResourceManagerCustomEndpoint = providerConfig.ResourceManagerCustomEndpoint.ValueString()
	}
	if !(providerConfig.SecretsManagerCustomEndpoint.IsUnknown() || providerConfig.SecretsManagerCustomEndpoint.IsNull()) {
		providerData.SecretsManagerCustomEndpoint = providerConfig.SecretsManagerCustomEndpoint.ValueString()
	}
	if !(providerConfig.TokenCustomEndpoint.IsUnknown() || providerConfig.TokenCustomEndpoint.IsNull()) {
		sdkConfig.TokenCustomUrl = providerConfig.TokenCustomEndpoint.ValueString()
	}
	if !(providerConfig.JWKSCustomEndpoint.IsUnknown() || providerConfig.JWKSCustomEndpoint.IsNull()) {
		sdkConfig.JWKSCustomUrl = providerConfig.JWKSCustomEndpoint.ValueString()
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
		logMeInstance.NewInstanceDataSource,
		logMeCredential.NewCredentialDataSource,
		mariaDBInstance.NewInstanceDataSource,
		mariaDBCredential.NewCredentialDataSource,
		mongoDBFlexInstance.NewInstanceDataSource,
		mongoDBFlexUser.NewUserDataSource,
		objectStorageBucket.NewBucketDataSource,
		objecStorageCredentialsGroup.NewCredentialsGroupDataSource,
		objecStorageCredential.NewCredentialDataSource,
		openSearchInstance.NewInstanceDataSource,
		openSearchCredential.NewCredentialDataSource,
		postgresFlexInstance.NewInstanceDataSource,
		postgresFlexUser.NewUserDataSource,
		postgresInstance.NewInstanceDataSource,
		postgresCredential.NewCredentialDataSource,
		rabbitMQInstance.NewInstanceDataSource,
		rabbitMQCredential.NewCredentialDataSource,
		redisInstance.NewInstanceDataSource,
		redisCredential.NewCredentialDataSource,
		resourceManagerProject.NewProjectDataSource,
		secretsManagerInstance.NewInstanceDataSource,
		secretsManagerUser.NewUserDataSource,
		skeProject.NewProjectDataSource,
		skeCluster.NewClusterDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *Provider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		argusCredential.NewCredentialResource,
		argusInstance.NewInstanceResource,
		argusScrapeConfig.NewScrapeConfigResource,
		dnsZone.NewZoneResource,
		dnsRecordSet.NewRecordSetResource,
		logMeInstance.NewInstanceResource,
		logMeCredential.NewCredentialResource,
		mariaDBInstance.NewInstanceResource,
		mariaDBCredential.NewCredentialResource,
		mongoDBFlexInstance.NewInstanceResource,
		mongoDBFlexUser.NewUserResource,
		objectStorageBucket.NewBucketResource,
		objecStorageCredentialsGroup.NewCredentialsGroupResource,
		objecStorageCredential.NewCredentialResource,
		openSearchInstance.NewInstanceResource,
		openSearchCredential.NewCredentialResource,
		postgresFlexInstance.NewInstanceResource,
		postgresFlexUser.NewUserResource,
		postgresInstance.NewInstanceResource,
		postgresCredential.NewCredentialResource,
		rabbitMQInstance.NewInstanceResource,
		rabbitMQCredential.NewCredentialResource,
		redisInstance.NewInstanceResource,
		redisCredential.NewCredentialResource,
		resourceManagerProject.NewProjectResource,
		secretsManagerInstance.NewInstanceResource,
		secretsManagerUser.NewUserResource,
		skeProject.NewProjectResource,
		skeCluster.NewClusterResource,
	}
}
