package core

import (
	albSdk "github.com/stackitcloud/stackit-sdk-go/services/alb/v2api"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"
	authorization "github.com/stackitcloud/stackit-sdk-go/services/authorization/v2api"
	cdnSdk "github.com/stackitcloud/stackit-sdk-go/services/cdn/v1api"
	certificates "github.com/stackitcloud/stackit-sdk-go/services/certificates/v2api"
	dns "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
	dremioSdk "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1betaapi"
	edge "github.com/stackitcloud/stackit-sdk-go/services/edge/v1beta1api"
	git "github.com/stackitcloud/stackit-sdk-go/services/git/v1betaapi"
	iaasV2Alpha "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"
	iaasV2 "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	intake "github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi"
	kms "github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
	logmeSdk "github.com/stackitcloud/stackit-sdk-go/services/logme/v2api"
	logs "github.com/stackitcloud/stackit-sdk-go/services/logs/v1api"
	mariadb "github.com/stackitcloud/stackit-sdk-go/services/mariadb/v2api"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	modelserving "github.com/stackitcloud/stackit-sdk-go/services/modelserving/v1api"
	mongodbflex "github.com/stackitcloud/stackit-sdk-go/services/mongodbflex/v2api"
	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"
	observabilitySdk "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"
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
)

var _ ClientFactory = &MockClientFactory{}

type MockClientFactory struct {
	AlbCertificatesV2ClientMock     certificates.DefaultAPI
	AlbV2ClientMock                 albSdk.DefaultAPI
	AlbWafV1ClientMock              albWaf.DefaultAPI
	AuthorizationV2ClientMock       authorization.DefaultAPI
	CdnV1ClientMock                 cdnSdk.DefaultAPI
	DnsV1ClientMock                 dns.DefaultAPI
	DremioV2BetaClientMock          dremioSdk.DefaultAPI
	EdgeV1ClientMock                edge.DefaultAPI
	GitV1BetaClientMock             git.DefaultAPI
	IaaSV2ClientMock                iaasV2.DefaultAPI
	IaasV2AlphaClientMock           iaasV2Alpha.DefaultAPI
	IntakeV1BetaClientMock          intake.DefaultAPI
	KmsV1ClientMock                 kms.DefaultAPI
	LoadbalancerV2ClientMock        loadbalancer.DefaultAPI
	LogmeV2ClientMock               logmeSdk.DefaultAPI
	LogsV1ClientMock                logs.DefaultAPI
	MariadbV2ClientMock             mariadb.DefaultAPI
	ModelExperimentsV1ClientMock    modelexperiments.DefaultAPI
	ModelServerV1ClientMock         modelserving.DefaultAPI
	MongoDbFlexV2ClientMock         mongodbflex.DefaultAPI
	ObjectStorageV2ClientMock       objectstorage.DefaultAPI
	ObservabilityV1ClientMock       observabilitySdk.DefaultAPI
	OpensearchV2ClientMock          opensearch.DefaultAPI
	PostgresflexV3ClientMock        postgresflex.DefaultAPI
	RabbitMqV2ClientMock            rabbitmq.DefaultAPI
	RedisV2ClientMock               redis.DefaultAPI
	ResourceManagerClientMock       resourcemanager.DefaultAPI
	ScfV1ClientMock                 scf.DefaultAPI
	SecretsManagerV1AlphaClientMock secretsmanagerV1Alpha.DefaultAPI
	SecretsManagerV1ClientMock      secretsmanager.DefaultAPI
	ServerBackupV2ClientMock        serverbackup.DefaultAPI
	ServerUpdateV2ClientMock        serverupdate.DefaultAPI
	ServiceAccountV2ClientMock      serviceaccount.DefaultAPI
	ServiceEnablementV2ClientMock   serviceenablementV2.DefaultAPI
	SfsV1ClientMock                 sfs.DefaultAPI
	SkeV2ClientMock                 ske.DefaultAPI
	SqlServerFlexV3ClientMock       sqlserverflex.DefaultAPI
	TelemetryLinkV1ClientMock       telemetrylink.DefaultAPI
	TelemetryRouterV1ClientMock     telemetryrouter.DefaultAPI
	VpnV1ClientMock                 vpn.DefaultAPI
}

func (m *MockClientFactory) newAlbV2Client() (albSdk.DefaultAPI, error) {
	if m.AlbV2ClientMock != nil {
		return m.AlbV2ClientMock, nil
	}

	return albSdk.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newDremioV1BetaClient() (dremioSdk.DefaultAPI, error) {
	if m.DremioV2BetaClientMock != nil {
		return m.DremioV2BetaClientMock, nil
	}

	return dremioSdk.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newGitV1BetaClient() (git.DefaultAPI, error) {
	if m.GitV1BetaClientMock != nil {
		return m.GitV1BetaClientMock, nil
	}

	return git.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newIntakeV1BetaClient() (intake.DefaultAPI, error) {
	if m.IntakeV1BetaClientMock != nil {
		return m.IntakeV1BetaClientMock, nil
	}

	return intake.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newKmsV1Client() (kms.DefaultAPI, error) {
	if m.KmsV1ClientMock != nil {
		return m.KmsV1ClientMock, nil
	}

	return kms.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newLoadbalancerV2Client() (loadbalancer.DefaultAPI, error) {
	if m.LoadbalancerV2ClientMock != nil {
		return m.LoadbalancerV2ClientMock, nil
	}

	return loadbalancer.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newLogmeV2Client() (logmeSdk.DefaultAPI, error) {
	if m.LogmeV2ClientMock != nil {
		return m.LogmeV2ClientMock, nil
	}

	return logmeSdk.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newMariadbV2Client() (mariadb.DefaultAPI, error) {
	if m.MariadbV2ClientMock != nil {
		return m.MariadbV2ClientMock, nil
	}

	return mariadb.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newModelServingV1Client() (modelserving.DefaultAPI, error) {
	if m.ModelServerV1ClientMock != nil {
		return m.ModelServerV1ClientMock, nil
	}

	return modelserving.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newMongoDbFlexV2Client() (mongodbflex.DefaultAPI, error) {
	if m.MongoDbFlexV2ClientMock != nil {
		return m.MongoDbFlexV2ClientMock, nil
	}

	return mongodbflex.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newObjectStorageV2Client() (objectstorage.DefaultAPI, error) {
	if m.ObjectStorageV2ClientMock != nil {
		return m.ObjectStorageV2ClientMock, nil
	}

	return objectstorage.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newObservabilityV1Client() (observabilitySdk.DefaultAPI, error) {
	if m.ObservabilityV1ClientMock != nil {
		return m.ObservabilityV1ClientMock, nil
	}

	return observabilitySdk.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newOpensearchV2Client() (opensearch.DefaultAPI, error) {
	if m.OpensearchV2ClientMock != nil {
		return m.OpensearchV2ClientMock, nil
	}

	return opensearch.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newRabbitMqV2Client() (rabbitmq.DefaultAPI, error) {
	if m.RabbitMqV2ClientMock != nil {
		return m.RabbitMqV2ClientMock, nil
	}

	return rabbitmq.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newRedisV2Client() (redis.DefaultAPI, error) {
	if m.RedisV2ClientMock != nil {
		return m.RedisV2ClientMock, nil
	}

	return redis.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newScfV1Client() (scf.DefaultAPI, error) {
	if m.ScfV1ClientMock != nil {
		return m.ScfV1ClientMock, nil
	}

	return scf.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newSecretsManagerV1AlphaClient() (secretsmanagerV1Alpha.DefaultAPI, error) {
	if m.SecretsManagerV1AlphaClientMock != nil {
		return m.SecretsManagerV1AlphaClientMock, nil
	}

	return secretsmanagerV1Alpha.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newSecretsManagerV1Client() (secretsmanager.DefaultAPI, error) {
	if m.SecretsManagerV1ClientMock != nil {
		return m.SecretsManagerV1ClientMock, nil
	}

	return secretsmanager.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newServerUpdateV2Client() (serverupdate.DefaultAPI, error) {
	if m.ServerUpdateV2ClientMock != nil {
		return m.ServerUpdateV2ClientMock, nil
	}

	return serverupdate.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newServiceAccountV2Client() (serviceaccount.DefaultAPI, error) {
	if m.ServiceAccountV2ClientMock != nil {
		return m.ServiceAccountV2ClientMock, nil
	}

	return serviceaccount.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newSfsV1Client() (sfs.DefaultAPI, error) {
	if m.SfsV1ClientMock != nil {
		return m.SfsV1ClientMock, nil
	}

	return sfs.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newSkeV2Client() (ske.DefaultAPI, error) {
	if m.SkeV2ClientMock != nil {
		return m.SkeV2ClientMock, nil
	}

	return ske.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newSqlServerFlexV3Client() (sqlserverflex.DefaultAPI, error) {
	if m.SqlServerFlexV3ClientMock != nil {
		return m.SqlServerFlexV3ClientMock, nil
	}

	return sqlserverflex.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newTelemetryLinkV1Client() (telemetrylink.DefaultAPI, error) {
	if m.TelemetryLinkV1ClientMock != nil {
		return m.TelemetryLinkV1ClientMock, nil
	}

	return telemetrylink.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newTelemetryRouterV1Client() (telemetryrouter.DefaultAPI, error) {
	if m.TelemetryRouterV1ClientMock != nil {
		return m.TelemetryRouterV1ClientMock, nil
	}

	return telemetryrouter.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newAlbCertificatesV2Client() (certificates.DefaultAPI, error) {
	if m.AlbCertificatesV2ClientMock != nil {
		return m.AlbCertificatesV2ClientMock, nil
	}

	return certificates.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newCdnV1Client() (cdnSdk.DefaultAPI, error) {
	if m.CdnV1ClientMock != nil {
		return m.CdnV1ClientMock, nil
	}

	return cdnSdk.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newIaaSV2AlphaClient() (iaasV2Alpha.DefaultAPI, error) {
	if m.IaaSV2ClientMock != nil {
		return m.IaasV2AlphaClientMock, nil
	}

	return iaasV2Alpha.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newEdgeV1Client() (edge.DefaultAPI, error) {
	if m.EdgeV1ClientMock != nil {
		return m.EdgeV1ClientMock, nil
	}

	return edge.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newAlbWafV1Client() (albWaf.DefaultAPI, error) {
	if m.AlbWafV1ClientMock != nil {
		return m.AlbWafV1ClientMock, nil
	}

	return albWaf.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newAuthorizationV2Client() (authorization.DefaultAPI, error) {
	if m.AuthorizationV2ClientMock != nil {
		return m.AuthorizationV2ClientMock, nil
	}

	return authorization.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newDnsV1Client() (dns.DefaultAPI, error) {
	if m.DnsV1ClientMock != nil {
		return m.DnsV1ClientMock, nil
	}

	return dns.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newLogsV1Client() (logs.DefaultAPI, error) {
	if m.LogsV1ClientMock != nil {
		return m.LogsV1ClientMock, nil
	}

	return logs.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newPostgresflexV3Client() (postgresflex.DefaultAPI, error) {
	if m.PostgresflexV3ClientMock != nil {
		return m.PostgresflexV3ClientMock, nil
	}

	return postgresflex.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newServerBackupV2Client() (serverbackup.DefaultAPI, error) {
	if m.ServerBackupV2ClientMock != nil {
		return m.ServerBackupV2ClientMock, nil
	}

	return serverbackup.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newVpnV1Client() (vpn.DefaultAPI, error) {
	if m.VpnV1ClientMock != nil {
		return m.VpnV1ClientMock, nil
	}

	return vpn.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newServiceEnablementV2Client() (serviceenablementV2.DefaultAPI, error) {
	if m.ServiceEnablementV2ClientMock != nil {
		return m.ServiceEnablementV2ClientMock, nil
	}

	return serviceenablementV2.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newIaaSV2Client() (iaasV2.DefaultAPI, error) {
	if m.IaaSV2ClientMock != nil {
		return m.IaaSV2ClientMock, nil
	}

	return iaasV2.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newResourceManagerClient() (resourcemanager.DefaultAPI, error) {
	if m.ResourceManagerClientMock != nil {
		return m.ResourceManagerClientMock, nil
	}

	return resourcemanager.DefaultAPIServiceMock{}, nil
}

func (m *MockClientFactory) newModelExperimentsV1Client() (modelexperiments.DefaultAPI, error) {
	if m.ModelExperimentsV1ClientMock != nil {
		return m.ModelExperimentsV1ClientMock, nil
	}

	return modelexperiments.DefaultAPIServiceMock{}, nil
}
