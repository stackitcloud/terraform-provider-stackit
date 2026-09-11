package core

import (
	"fmt"
	"net/http"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
	alb "github.com/stackitcloud/stackit-sdk-go/services/alb/v2api"
	albwaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"
	authorization "github.com/stackitcloud/stackit-sdk-go/services/authorization/v2api"
	cdn "github.com/stackitcloud/stackit-sdk-go/services/cdn/v1api"
	certificates "github.com/stackitcloud/stackit-sdk-go/services/certificates/v2api"
	dns "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
	dremio "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1betaapi"
	edge "github.com/stackitcloud/stackit-sdk-go/services/edge/v1beta1api"
	git "github.com/stackitcloud/stackit-sdk-go/services/git/v1betaapi"
	iaasV2Alpha "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"
	iaasV2 "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
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
	serviceenablementV2 "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"
	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"
	telemetrylink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1api"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1api"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
	"golang.org/x/sync/errgroup"
)

type ClientFactory interface {
	// methods are having the API versions in them here so we can still mix & match API versions just as we need

	newAlbCertificatesV2Client() (certificates.DefaultAPI, error)
	newAlbV2Client() (alb.DefaultAPI, error)
	newAlbWafV1Client() (albwaf.DefaultAPI, error)
	newAuthorizationV2Client() (authorization.DefaultAPI, error)
	newCdnV1Client() (cdn.DefaultAPI, error)
	newDnsV1Client() (dns.DefaultAPI, error)
	newDremioV1BetaClient() (dremio.DefaultAPI, error)
	newEdgeV1Client() (edge.DefaultAPI, error)
	newGitV1BetaClient() (git.DefaultAPI, error)
	newIaaSV2AlphaClient() (iaasV2Alpha.DefaultAPI, error)
	newIaaSV2Client() (iaasV2.DefaultAPI, error)
	newIntakeV1BetaClient() (intake.DefaultAPI, error)
	newKmsV1Client() (kms.DefaultAPI, error)
	newLoadbalancerV2Client() (loadbalancer.DefaultAPI, error)
	newLogmeV2Client() (logme.DefaultAPI, error)
	newLogsV1Client() (logs.DefaultAPI, error)
	newMariadbV2Client() (mariadb.DefaultAPI, error)
	newModelExperimentsV1Client() (modelexperiments.DefaultAPI, error)
	newModelServingV1Client() (modelserving.DefaultAPI, error)
	newMongoDbFlexV2Client() (mongodbflex.DefaultAPI, error)
	newObjectStorageV2Client() (objectstorage.DefaultAPI, error)
	newObservabilityV1Client() (observability.DefaultAPI, error)
	newOpensearchV2Client() (opensearch.DefaultAPI, error)
	newPostgresflexV3Client() (postgresflex.DefaultAPI, error)
	newRabbitMqV2Client() (rabbitmq.DefaultAPI, error)
	newRedisV2Client() (redis.DefaultAPI, error)
	newResourceManagerClient() (resourcemanager.DefaultAPI, error)
	newScfV1Client() (scf.DefaultAPI, error)
	newSecretsManagerV1AlphaClient() (secretsmanagerV1Alpha.DefaultAPI, error)
	newSecretsManagerV1Client() (secretsmanager.DefaultAPI, error)
	newServerBackupV2Client() (serverbackup.DefaultAPI, error)
	newServerUpdateV2Client() (serverupdate.DefaultAPI, error)
	newServiceAccountV2Client() (serviceaccount.DefaultAPI, error)
	newServiceEnablementV2Client() (serviceenablementV2.DefaultAPI, error)
	newSfsV1Client() (sfs.DefaultAPI, error)
	newSkeV2Client() (ske.DefaultAPI, error)
	newSqlServerFlexV3Client() (sqlserverflex.DefaultAPI, error)
	newTelemetryLinkV1Client() (telemetrylink.DefaultAPI, error)
	newTelemetryRouterV1Client() (telemetryrouter.DefaultAPI, error)
	newVpnV1Client() (vpn.DefaultAPI, error)
}

func initClientCollection(clientFactory ClientFactory) (*clientCollection, error) {
	var g errgroup.Group
	cc := &clientCollection{}

	// initialize clients in parallel
	g.Go(func() (err error) { cc.IaaSv2Client, err = clientFactory.newIaaSV2Client(); return err })
	g.Go(func() (err error) { cc.EdgeV1Client, err = clientFactory.newEdgeV1Client(); return err })
	g.Go(func() (err error) { cc.DnsV1Client, err = clientFactory.newDnsV1Client(); return err })
	g.Go(func() (err error) { cc.ServerBackupV2Client, err = clientFactory.newServerBackupV2Client(); return err })
	g.Go(func() (err error) { cc.AlbWafV1CLient, err = clientFactory.newAlbWafV1Client(); return err })
	g.Go(func() (err error) { cc.LogsV1Client, err = clientFactory.newLogsV1Client(); return err })
	g.Go(func() (err error) { cc.VpnV1Client, err = clientFactory.newVpnV1Client(); return err })
	g.Go(func() (err error) { cc.IaaSv2AlphaClient, err = clientFactory.newIaaSV2AlphaClient(); return err })
	g.Go(func() (err error) { cc.CdnV1Client, err = clientFactory.newCdnV1Client(); return err })
	g.Go(func() (err error) { cc.PostgresflexV3Client, err = clientFactory.newPostgresflexV3Client(); return err })
	g.Go(func() (err error) { cc.AlbV2Client, err = clientFactory.newAlbV2Client(); return err })
	g.Go(func() (err error) { cc.SkeV2Client, err = clientFactory.newSkeV2Client(); return err })
	g.Go(func() (err error) { cc.ModelservingV1Client, err = clientFactory.newModelServingV1Client(); return err })
	g.Go(func() (err error) { cc.LogmeV2Client, err = clientFactory.newLogmeV2Client(); return err })
	g.Go(func() (err error) { cc.OpensearchV2Client, err = clientFactory.newOpensearchV2Client(); return err })
	g.Go(func() (err error) { cc.GitV1BetaClient, err = clientFactory.newGitV1BetaClient(); return err })
	g.Go(func() (err error) { cc.RedisV2Client, err = clientFactory.newRedisV2Client(); return err })
	g.Go(func() (err error) { cc.ServerUpdateV2Client, err = clientFactory.newServerUpdateV2Client(); return err })
	g.Go(func() (err error) { cc.KmsV1Client, err = clientFactory.newKmsV1Client(); return err })
	g.Go(func() (err error) { cc.SfsV1Client, err = clientFactory.newSfsV1Client(); return err })
	g.Go(func() (err error) { cc.RabbitMqV2Client, err = clientFactory.newRabbitMqV2Client(); return err })
	g.Go(func() (err error) { cc.MongoDbFlexV2Client, err = clientFactory.newMongoDbFlexV2Client(); return err })
	g.Go(func() (err error) { cc.MariadbV2Client, err = clientFactory.newMariadbV2Client(); return err })
	g.Go(func() (err error) { cc.ScfV1Client, err = clientFactory.newScfV1Client(); return err })
	g.Go(func() (err error) { cc.LoadbalancerV2Client, err = clientFactory.newLoadbalancerV2Client(); return err })
	g.Go(func() (err error) { cc.IntakeV1BetaClient, err = clientFactory.newIntakeV1BetaClient(); return err })
	g.Go(func() (err error) { cc.DremioV1BetaClient, err = clientFactory.newDremioV1BetaClient(); return err })
	g.Go(func() (err error) {
		cc.ResourceManagerClient, err = clientFactory.newResourceManagerClient()
		return err
	})
	g.Go(func() (err error) {
		cc.ModelExperimentsV1Client, err = clientFactory.newModelExperimentsV1Client()
		return err
	})
	g.Go(func() (err error) {
		cc.ServiceEnablementV2Client, err = clientFactory.newServiceEnablementV2Client()
		return err
	})
	g.Go(func() (err error) {
		cc.AlbCertificatesV2Client, err = clientFactory.newAlbCertificatesV2Client()
		return err
	})
	g.Go(func() (err error) {
		cc.AuthorizationV2Client, err = clientFactory.newAuthorizationV2Client()
		return err
	})
	g.Go(func() (err error) {
		cc.SqlServerFlexV3Client, err = clientFactory.newSqlServerFlexV3Client()
		return err
	})
	g.Go(func() (err error) {
		cc.TelemetryRouterV1Client, err = clientFactory.newTelemetryRouterV1Client()
		return err
	})
	g.Go(func() (err error) {
		cc.TelemetryLinkV1Client, err = clientFactory.newTelemetryLinkV1Client()
		return err
	})
	g.Go(func() (err error) {
		cc.ServiceAccountV2Client, err = clientFactory.newServiceAccountV2Client()
		return err
	})
	g.Go(func() (err error) {
		cc.ServiceAccountV2Client, err = clientFactory.newServiceAccountV2Client()
		return err
	})
	g.Go(func() (err error) {
		cc.ObjectStorageV2Client, err = clientFactory.newObjectStorageV2Client()
		return err
	})
	g.Go(func() (err error) {
		cc.SecretsmanagerV1Client, err = clientFactory.newSecretsManagerV1Client()
		return err
	})
	g.Go(func() (err error) {
		cc.SecretsmanagerV1AlphaClient, err = clientFactory.newSecretsManagerV1AlphaClient()
		return err
	})
	g.Go(func() (err error) {
		cc.ObservabilityV1Client, err = clientFactory.newObservabilityV1Client()
		return err
	})

	// wait for initialization of all clients, handle errors
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return cc, nil
}

var _ ClientFactory = &DefaultClientFactory{}

type DefaultClientFactory struct {
	RoundTripper    http.RoundTripper
	UserAgent       string
	CustomEndpoints CustomEndpointConfig

	// Deprecated: This should be only used for legacy implementations, not for new ones!
	ProviderDefaultRegion string
}

func (f *DefaultClientFactory) defaultConfigOptions(customEndpoint string) []config.ConfigurationOption {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(f.RoundTripper),
		config.WithUserAgent(f.UserAgent),
	}

	if customEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(customEndpoint))
	}

	return apiClientConfigOptions
}

func (f *DefaultClientFactory) newAlbV2Client() (alb.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ALBCustomEndpoint)

	apiClient, err := alb.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newGitV1BetaClient() (git.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.GitCustomEndpoint)

	apiClient, err := git.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newIntakeV1BetaClient() (intake.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.IntakeCustomEndpoint)

	apiClient, err := intake.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newKmsV1Client() (kms.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.KMSCustomEndpoint)

	apiClient, err := kms.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newLoadbalancerV2Client() (loadbalancer.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.LoadBalancerCustomEndpoint)

	apiClient, err := loadbalancer.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newLogmeV2Client() (logme.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.LogMeCustomEndpoint)

	apiClient, err := logme.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newMariadbV2Client() (mariadb.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.MariaDBCustomEndpoint)

	apiClient, err := mariadb.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newModelServingV1Client() (modelserving.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ModelServingCustomEndpoint)

	apiClient, err := modelserving.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newMongoDbFlexV2Client() (mongodbflex.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.MongoDBFlexCustomEndpoint)

	apiClient, err := mongodbflex.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newObjectStorageV2Client() (objectstorage.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ObjectStorageCustomEndpoint)

	apiClient, err := objectstorage.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newOpensearchV2Client() (opensearch.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.OpenSearchCustomEndpoint)

	apiClient, err := opensearch.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newRabbitMqV2Client() (rabbitmq.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.RabbitMQCustomEndpoint)

	apiClient, err := rabbitmq.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newRedisV2Client() (redis.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.RedisCustomEndpoint)

	apiClient, err := redis.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newScfV1Client() (scf.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ScfCustomEndpoint)

	apiClient, err := scf.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newServerUpdateV2Client() (serverupdate.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ServerUpdateCustomEndpoint)

	apiClient, err := serverupdate.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newServiceAccountV2Client() (serviceaccount.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ServiceAccountCustomEndpoint)

	apiClient, err := serviceaccount.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newSfsV1Client() (sfs.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.SfsCustomEndpoint)

	apiClient, err := sfs.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newSkeV2Client() (ske.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.SKECustomEndpoint)

	apiClient, err := ske.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newSqlServerFlexV3Client() (sqlserverflex.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.SQLServerFlexCustomEndpoint)

	apiClient, err := sqlserverflex.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newTelemetryLinkV1Client() (telemetrylink.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.TelemetryLinkCustomEndpoint)

	apiClient, err := telemetrylink.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newTelemetryRouterV1Client() (telemetryrouter.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.TelemetryRouterCustomEndpoint)

	apiClient, err := telemetryrouter.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newAlbCertificatesV2Client() (certificates.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ALBCertificatesCustomEndpoint)

	apiClient, err := certificates.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newCdnV1Client() (cdn.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.CdnCustomEndpoint)

	apiClient, err := cdn.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newIaaSV2AlphaClient() (iaasV2Alpha.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.IaaSCustomEndpoint)

	apiClient, err := iaasV2Alpha.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newEdgeV1Client() (edge.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.EdgeCloudCustomEndpoint)

	apiClient, err := edge.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newAlbWafV1Client() (albwaf.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.AlbWafCustomEndpoint)

	apiClient, err := albwaf.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newAuthorizationV2Client() (authorization.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.AuthorizationCustomEndpoint)

	apiClient, err := authorization.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newDnsV1Client() (dns.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.DnsCustomEndpoint)

	apiClient, err := dns.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newLogsV1Client() (logs.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.LogsCustomEndpoint)

	apiClient, err := logs.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newPostgresflexV3Client() (postgresflex.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.PostgresFlexCustomEndpoint)

	apiClient, err := postgresflex.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newServerBackupV2Client() (serverbackup.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ServerBackupCustomEndpoint)

	apiClient, err := serverbackup.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newVpnV1Client() (vpn.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.VpnCustomEndpoint)

	apiClient, err := vpn.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newServiceEnablementV2Client() (serviceenablementV2.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ServiceEnablementCustomEndpoint)

	apiClient, err := serviceenablementV2.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newIaaSV2Client() (iaasV2.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.IaaSCustomEndpoint)

	apiClient, err := iaasV2.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newResourceManagerClient() (resourcemanager.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ResourceManagerCustomEndpoint)

	apiClient, err := resourcemanager.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newModelExperimentsV1Client() (modelexperiments.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ModelExperimentsCustomEndpoint)

	apiClient, err := modelexperiments.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newDremioV1BetaClient() (dremio.DefaultAPI, error) {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(f.RoundTripper),
		config.WithUserAgent(f.UserAgent),
		config.WithRegion(f.ProviderDefaultRegion),
	}
	if f.CustomEndpoints.DremioCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(f.CustomEndpoints.DremioCustomEndpoint))
	}
	apiClient, err := dremio.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newSecretsManagerV1Client() (secretsmanager.DefaultAPI, error) {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(f.RoundTripper),
		config.WithUserAgent(f.UserAgent),
	}

	if f.CustomEndpoints.SecretsManagerCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(f.CustomEndpoints.SecretsManagerCustomEndpoint))
	} else {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithRegion(f.ProviderDefaultRegion))
	}

	apiClient, err := secretsmanager.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newSecretsManagerV1AlphaClient() (secretsmanagerV1Alpha.DefaultAPI, error) {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(f.RoundTripper),
		config.WithUserAgent(f.UserAgent),
	}

	if f.CustomEndpoints.SecretsManagerCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(f.CustomEndpoints.SecretsManagerCustomEndpoint))
	} else {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithRegion(f.ProviderDefaultRegion))
	}

	apiClient, err := secretsmanagerV1Alpha.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newObservabilityV1Client() (observability.DefaultAPI, error) {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(f.RoundTripper),
		config.WithUserAgent(f.UserAgent),
	}

	if f.CustomEndpoints.ObservabilityCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(f.CustomEndpoints.ObservabilityCustomEndpoint))
	} else {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithRegion(f.ProviderDefaultRegion))
	}

	apiClient, err := observability.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("configuring client: %w. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}
