package core

import (
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	alb "github.com/stackitcloud/stackit-sdk-go/services/alb/v2api"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"
	authorization "github.com/stackitcloud/stackit-sdk-go/services/authorization/v2api"
	cdn "github.com/stackitcloud/stackit-sdk-go/services/cdn/v1api"
	certificates "github.com/stackitcloud/stackit-sdk-go/services/certificates/v2api"
	dns "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
	dremio "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1betaapi"
	edge "github.com/stackitcloud/stackit-sdk-go/services/edge/v1beta1api"
	git "github.com/stackitcloud/stackit-sdk-go/services/git/v1betaapi"
	iaasV2Alpha "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
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

const (
	testCustomEndpoint = "https://test-custom-endpoint.api.stackit.cloud"
)

func Test_initClientCollection(t *testing.T) {
	type args struct {
		clientFactory ClientFactory
	}
	tests := []struct {
		name    string
		args    args
		want    *ClientCollection
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := initClientCollection(tt.args.clientFactory)
			if (err != nil) != tt.wantErr {
				t.Fatalf("initClientCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("initClientCollection() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitClientCollection_NoNilFields(t *testing.T) {
	// This test case makes sure that all fields in the client collection are initialized after initializing it
	// using a client factory. It uses reflection to make sure all fields are covered, also new ones which got added.
	// A regular unit test without reflection couldn't cover this.

	mockFactory := &MockClientFactory{}

	cc, err := initClientCollection(mockFactory)
	if err != nil {
		t.Fatalf("unexpected error during initialization: %v", err)
	}

	if cc == nil {
		t.Fatal("expected clientCollection pointer to be non-nil")
	}

	val := reflect.ValueOf(*cc)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldName := typ.Field(i).Name

		if fieldVal.IsZero() {
			t.Fatalf("field %q is nil or uninitialized", fieldName)
		}
	}
}

func TestDefaultClientFactory_defaultConfigOptions(t *testing.T) {
	var randomRoundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}

	type fields struct {
		RoundTripper http.RoundTripper
		UserAgent    string
	}

	type args struct {
		customEndpoint string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   []config.ConfigurationOption
	}{
		{
			name: "custom endpoint is set",
			fields: fields{
				RoundTripper: randomRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
			},
			args: args{
				customEndpoint: "https://my-custom-endpoint.api.stackit.cloud",
			},
			want: []config.ConfigurationOption{
				config.WithCustomAuth(randomRoundTripper),
				config.WithUserAgent("stackit-terraform-provider/1.2.3"),
				config.WithEndpoint("https://my-custom-endpoint.api.stackit.cloud"),
			},
		},
		{
			name: "custom endpoint is not set",
			fields: fields{
				RoundTripper: randomRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
			},
			args: args{
				customEndpoint: "",
			},
			want: []config.ConfigurationOption{
				config.WithCustomAuth(randomRoundTripper),
				config.WithUserAgent("stackit-terraform-provider/1.2.3"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &DefaultClientFactory{
				RoundTripper: tt.fields.RoundTripper,
				UserAgent:    tt.fields.UserAgent,
			}

			got := f.defaultConfigOptions(tt.args.customEndpoint)

			buildConfig := func(configOptions []config.ConfigurationOption) config.Configuration {
				cfg := config.Configuration{}

				for _, opt := range configOptions {
					err := opt(&cfg)
					if err != nil {
						t.Fatalf("error during configuration options: %v", err)
					}
				}

				return cfg
			}

			opts := cmpopts.IgnoreUnexported(http.Transport{}, config.Configuration{}, tls.Config{})
			if diff := cmp.Diff(buildConfig(tt.want), buildConfig(got), opts); diff != "" {
				t.Fatalf("defaultConfigOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type testHelper[T any] struct {
	factoryMethod        func(*DefaultClientFactory) (T, error)
	clientInitFunc       func(opts ...config.ConfigurationOption) T
	customEndpointSetter func(endpointConfig *CustomEndpointConfig)
}

func (h *testHelper[T]) run(t *testing.T) {
	t.Helper()

	var testRoundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}

	type fields struct {
		RoundTripper          http.RoundTripper
		UserAgent             string
		CustomEndpoints       CustomEndpointConfig
		ProviderDefaultRegion string
	}

	tests := []struct {
		name    string
		fields  fields
		want    T
		wantErr bool
	}{
		{
			name: "without custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: CustomEndpointConfig{
					ALBCustomEndpoint: "",
				},
				ProviderDefaultRegion: "eu01",
			},
			want: h.clientInitFunc(
				config.WithUserAgent("stackit-terraform-provider/1.2.3"),
				config.WithCustomAuth(testRoundTripper),
			),
		},
		{
			name: "with custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: func() CustomEndpointConfig {
					cfg := CustomEndpointConfig{}
					h.customEndpointSetter(&cfg)
					return cfg
				}(),
				ProviderDefaultRegion: "eu01",
			},
			want: h.clientInitFunc(
				config.WithUserAgent("stackit-terraform-provider/1.2.3"),
				config.WithEndpoint(testCustomEndpoint),
				config.WithCustomAuth(testRoundTripper),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &DefaultClientFactory{
				RoundTripper:          tt.fields.RoundTripper,
				UserAgent:             tt.fields.UserAgent,
				CustomEndpoints:       tt.fields.CustomEndpoints,
				ProviderDefaultRegion: tt.fields.ProviderDefaultRegion,
			}

			got, err := h.factoryMethod(f)
			if (err != nil) != tt.wantErr {
				t.Fatalf("mismatch error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("mismatch = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultClientFactory_newAlbV2Client(t *testing.T) {
	test := testHelper[alb.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newAlbV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) alb.DefaultAPI {
			client, err := alb.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ALBCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newGitV1BetaClient(t *testing.T) {
	test := testHelper[git.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newGitV1BetaClient,
		clientInitFunc: func(opts ...config.ConfigurationOption) git.DefaultAPI {
			client, err := git.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.GitCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newIntakeV1BetaClient(t *testing.T) {
	test := testHelper[intake.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newIntakeV1BetaClient,
		clientInitFunc: func(opts ...config.ConfigurationOption) intake.DefaultAPI {
			client, err := intake.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.IntakeCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newKmsV1Client(t *testing.T) {
	test := testHelper[kms.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newKmsV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) kms.DefaultAPI {
			client, err := kms.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.KMSCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newLoadbalancerV2Client(t *testing.T) {
	test := testHelper[loadbalancer.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newLoadbalancerV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) loadbalancer.DefaultAPI {
			client, err := loadbalancer.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.LoadBalancerCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newLogmeV2Client(t *testing.T) {
	test := testHelper[logme.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newLogmeV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) logme.DefaultAPI {
			client, err := logme.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.LogMeCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newMariadbV2Client(t *testing.T) {
	test := testHelper[mariadb.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newMariadbV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) mariadb.DefaultAPI {
			client, err := mariadb.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.MariaDBCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newModelServingV1Client(t *testing.T) {
	test := testHelper[modelserving.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newModelServingV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) modelserving.DefaultAPI {
			client, err := modelserving.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ModelServingCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newMongoDbFlexV2Client(t *testing.T) {
	test := testHelper[mongodbflex.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newMongoDbFlexV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) mongodbflex.DefaultAPI {
			client, err := mongodbflex.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.MongoDBFlexCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newObjectStorageV2Client(t *testing.T) {
	test := testHelper[objectstorage.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newObjectStorageV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) objectstorage.DefaultAPI {
			client, err := objectstorage.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ObjectStorageCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newOpensearchV2Client(t *testing.T) {
	test := testHelper[opensearch.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newOpensearchV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) opensearch.DefaultAPI {
			client, err := opensearch.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.OpenSearchCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newRabbitMqV2Client(t *testing.T) {
	test := testHelper[rabbitmq.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newRabbitMqV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) rabbitmq.DefaultAPI {
			client, err := rabbitmq.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.RabbitMQCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newRedisV2Client(t *testing.T) {
	test := testHelper[redis.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newRedisV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) redis.DefaultAPI {
			client, err := redis.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.RedisCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newScfV1Client(t *testing.T) {
	test := testHelper[scf.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newScfV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) scf.DefaultAPI {
			client, err := scf.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ScfCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newServerUpdateV2Client(t *testing.T) {
	test := testHelper[serverupdate.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newServerUpdateV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) serverupdate.DefaultAPI {
			client, err := serverupdate.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ServerUpdateCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newServiceAccountV2Client(t *testing.T) {
	test := testHelper[serviceaccount.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newServiceAccountV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) serviceaccount.DefaultAPI {
			client, err := serviceaccount.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ServiceAccountCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newSfsV1Client(t *testing.T) {
	test := testHelper[sfs.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newSfsV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) sfs.DefaultAPI {
			client, err := sfs.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.SfsCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newSkeV2Client(t *testing.T) {
	test := testHelper[ske.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newSkeV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) ske.DefaultAPI {
			client, err := ske.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.SKECustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newSqlServerFlexV3Client(t *testing.T) {
	test := testHelper[sqlserverflex.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newSqlServerFlexV3Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) sqlserverflex.DefaultAPI {
			client, err := sqlserverflex.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.SQLServerFlexCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newTelemetryLinkV1Client(t *testing.T) {
	test := testHelper[telemetrylink.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newTelemetryLinkV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) telemetrylink.DefaultAPI {
			client, err := telemetrylink.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.TelemetryLinkCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newTelemetryRouterV1Client(t *testing.T) {
	test := testHelper[telemetryrouter.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newTelemetryRouterV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) telemetryrouter.DefaultAPI {
			client, err := telemetryrouter.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.TelemetryRouterCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newAlbCertificatesV2Client(t *testing.T) {
	test := testHelper[certificates.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newAlbCertificatesV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) certificates.DefaultAPI {
			client, err := certificates.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ALBCertificatesCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newCdnV1Client(t *testing.T) {
	test := testHelper[cdn.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newCdnV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) cdn.DefaultAPI {
			client, err := cdn.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.CdnCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newIaaSV2AlphaClient(t *testing.T) {
	test := testHelper[iaasV2Alpha.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newIaaSV2AlphaClient,
		clientInitFunc: func(opts ...config.ConfigurationOption) iaasV2Alpha.DefaultAPI {
			client, err := iaasV2Alpha.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.IaaSCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newEdgeV1Client(t *testing.T) {
	test := testHelper[edge.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newEdgeV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) edge.DefaultAPI {
			client, err := edge.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.EdgeCloudCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newAlbWafV1Client(t *testing.T) {
	test := testHelper[albWaf.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newAlbWafV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) albWaf.DefaultAPI {
			client, err := albWaf.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.AlbWafCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newAuthorizationV2Client(t *testing.T) {
	test := testHelper[authorization.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newAuthorizationV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) authorization.DefaultAPI {
			client, err := authorization.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.AuthorizationCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newDnsV1Client(t *testing.T) {
	test := testHelper[dns.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newDnsV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) dns.DefaultAPI {
			client, err := dns.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.DnsCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newLogsV1Client(t *testing.T) {
	test := testHelper[logs.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newLogsV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) logs.DefaultAPI {
			client, err := logs.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.LogsCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newPostgresflexV3Client(t *testing.T) {
	test := testHelper[postgresflex.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newPostgresflexV3Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) postgresflex.DefaultAPI {
			client, err := postgresflex.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.PostgresFlexCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newServerBackupV2Client(t *testing.T) {
	test := testHelper[serverbackup.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newServerBackupV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) serverbackup.DefaultAPI {
			client, err := serverbackup.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ServerBackupCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newVpnV1Client(t *testing.T) {
	test := testHelper[vpn.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newVpnV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) vpn.DefaultAPI {
			client, err := vpn.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.VpnCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newServiceEnablementV2Client(t *testing.T) {
	test := testHelper[serviceenablement.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newServiceEnablementV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) serviceenablement.DefaultAPI {
			client, err := serviceenablement.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ServiceEnablementCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newIaaSV2Client(t *testing.T) {
	test := testHelper[iaas.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newIaaSV2Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) iaas.DefaultAPI {
			client, err := iaas.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.IaaSCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newResourceManagerClient(t *testing.T) {
	test := testHelper[resourcemanager.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newResourceManagerClient,
		clientInitFunc: func(opts ...config.ConfigurationOption) resourcemanager.DefaultAPI {
			client, err := resourcemanager.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ResourceManagerCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newModelExperimentsV1Client(t *testing.T) {
	test := testHelper[modelexperiments.DefaultAPI]{
		factoryMethod: (*DefaultClientFactory).newModelExperimentsV1Client,
		clientInitFunc: func(opts ...config.ConfigurationOption) modelexperiments.DefaultAPI {
			client, err := modelexperiments.NewAPIClient(opts...)
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			return client.DefaultAPI
		},
		customEndpointSetter: func(cfg *CustomEndpointConfig) {
			cfg.ModelExperimentsCustomEndpoint = testCustomEndpoint
		},
	}

	test.run(t)
}

func TestDefaultClientFactory_newDremioV1BetaClient(t *testing.T) {
	var testRoundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}

	type fields struct {
		RoundTripper          http.RoundTripper
		UserAgent             string
		CustomEndpoints       CustomEndpointConfig
		ProviderDefaultRegion string
	}

	tests := []struct {
		name    string
		fields  fields
		want    dremio.DefaultAPI
		wantErr bool
	}{
		{
			name: "without custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: CustomEndpointConfig{
					DremioCustomEndpoint: "",
				},
				ProviderDefaultRegion: "eu01",
			},
			want: func() dremio.DefaultAPI {
				apiClient, err := dremio.NewAPIClient(
					config.WithUserAgent("stackit-terraform-provider/1.2.3"),
					config.WithCustomAuth(testRoundTripper),
					config.WithRegion("eu01"),
				)
				if err != nil {
					t.Fatalf("error configuring client: %v", err)
				}
				return apiClient.DefaultAPI
			}(),
		},
		{
			name: "with custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: CustomEndpointConfig{
					DremioCustomEndpoint: testCustomEndpoint,
				},
				ProviderDefaultRegion: "eu01",
			},
			want: func() dremio.DefaultAPI {
				apiClient, err := dremio.NewAPIClient(
					config.WithUserAgent("stackit-terraform-provider/1.2.3"),
					config.WithEndpoint(testCustomEndpoint),
					config.WithRegion("eu01"),
					config.WithCustomAuth(testRoundTripper),
				)
				if err != nil {
					t.Fatalf("error configuring client: %v", err)
				}
				return apiClient.DefaultAPI
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &DefaultClientFactory{
				RoundTripper:          tt.fields.RoundTripper,
				UserAgent:             tt.fields.UserAgent,
				CustomEndpoints:       tt.fields.CustomEndpoints,
				ProviderDefaultRegion: tt.fields.ProviderDefaultRegion,
			}
			got, err := f.newDremioV1BetaClient()
			if (err != nil) != tt.wantErr {
				t.Fatalf("newDremioV1BetaClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("newDremioV1BetaClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultClientFactory_newSecretsManagerV1Client(t *testing.T) {
	var testRoundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}

	type fields struct {
		RoundTripper          http.RoundTripper
		UserAgent             string
		CustomEndpoints       CustomEndpointConfig
		ProviderDefaultRegion string
	}

	tests := []struct {
		name    string
		fields  fields
		want    secretsmanager.DefaultAPI
		wantErr bool
	}{
		{
			name: "without custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: CustomEndpointConfig{
					SecretsManagerCustomEndpoint: "",
				},
				ProviderDefaultRegion: "eu01",
			},
			want: func() secretsmanager.DefaultAPI {
				apiClient, err := secretsmanager.NewAPIClient(
					config.WithUserAgent("stackit-terraform-provider/1.2.3"),
					config.WithCustomAuth(testRoundTripper),
					config.WithRegion("eu01"),
				)
				if err != nil {
					t.Fatalf("error configuring client: %v", err)
				}
				return apiClient.DefaultAPI
			}(),
		},
		{
			name: "with custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: CustomEndpointConfig{
					SecretsManagerCustomEndpoint: testCustomEndpoint,
				},
				ProviderDefaultRegion: "eu01",
			},
			want: func() secretsmanager.DefaultAPI {
				apiClient, err := secretsmanager.NewAPIClient(
					config.WithUserAgent("stackit-terraform-provider/1.2.3"),
					config.WithEndpoint(testCustomEndpoint),
					config.WithCustomAuth(testRoundTripper),
				)
				if err != nil {
					t.Fatalf("error configuring client: %v", err)
				}
				return apiClient.DefaultAPI
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &DefaultClientFactory{
				RoundTripper:          tt.fields.RoundTripper,
				UserAgent:             tt.fields.UserAgent,
				CustomEndpoints:       tt.fields.CustomEndpoints,
				ProviderDefaultRegion: tt.fields.ProviderDefaultRegion,
			}
			got, err := f.newSecretsManagerV1Client()
			if (err != nil) != tt.wantErr {
				t.Fatalf("newSecretsManagerV1Client() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("newSecretsManagerV1Client() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultClientFactory_newSecretsManagerV1AlphaClient(t *testing.T) {
	var testRoundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}

	type fields struct {
		RoundTripper          http.RoundTripper
		UserAgent             string
		CustomEndpoints       CustomEndpointConfig
		ProviderDefaultRegion string
	}

	tests := []struct {
		name    string
		fields  fields
		want    secretsmanagerV1Alpha.DefaultAPI
		wantErr bool
	}{
		{
			name: "without custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: CustomEndpointConfig{
					SecretsManagerCustomEndpoint: "",
				},
				ProviderDefaultRegion: "eu01",
			},
			want: func() secretsmanagerV1Alpha.DefaultAPI {
				apiClient, err := secretsmanagerV1Alpha.NewAPIClient(
					config.WithUserAgent("stackit-terraform-provider/1.2.3"),
					config.WithCustomAuth(testRoundTripper),
					config.WithRegion("eu01"),
				)
				if err != nil {
					t.Fatalf("error configuring client: %v", err)
				}
				return apiClient.DefaultAPI
			}(),
		},
		{
			name: "with custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: CustomEndpointConfig{
					SecretsManagerCustomEndpoint: testCustomEndpoint,
				},
				ProviderDefaultRegion: "eu01",
			},
			want: func() secretsmanagerV1Alpha.DefaultAPI {
				apiClient, err := secretsmanagerV1Alpha.NewAPIClient(
					config.WithUserAgent("stackit-terraform-provider/1.2.3"),
					config.WithEndpoint(testCustomEndpoint),
					config.WithCustomAuth(testRoundTripper),
				)
				if err != nil {
					t.Fatalf("error configuring client: %v", err)
				}
				return apiClient.DefaultAPI
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &DefaultClientFactory{
				RoundTripper:          tt.fields.RoundTripper,
				UserAgent:             tt.fields.UserAgent,
				CustomEndpoints:       tt.fields.CustomEndpoints,
				ProviderDefaultRegion: tt.fields.ProviderDefaultRegion,
			}
			got, err := f.newSecretsManagerV1AlphaClient()
			if (err != nil) != tt.wantErr {
				t.Fatalf("newSecretsManagerV1AlphaClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("newSecretsManagerV1AlphaClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultClientFactory_newObservabilityV1Client(t *testing.T) {
	var testRoundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}

	type fields struct {
		RoundTripper          http.RoundTripper
		UserAgent             string
		CustomEndpoints       CustomEndpointConfig
		ProviderDefaultRegion string
	}

	tests := []struct {
		name    string
		fields  fields
		want    observability.DefaultAPI
		wantErr bool
	}{
		{
			name: "without custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: CustomEndpointConfig{
					ObservabilityCustomEndpoint: "",
				},
				ProviderDefaultRegion: "eu01",
			},
			want: func() observability.DefaultAPI {
				apiClient, err := observability.NewAPIClient(
					config.WithUserAgent("stackit-terraform-provider/1.2.3"),
					config.WithCustomAuth(testRoundTripper),
					config.WithRegion("eu01"),
				)
				if err != nil {
					t.Fatalf("error configuring client: %v", err)
				}
				return apiClient.DefaultAPI
			}(),
		},
		{
			name: "with custom endpoint",
			fields: fields{
				RoundTripper: testRoundTripper,
				UserAgent:    "stackit-terraform-provider/1.2.3",
				CustomEndpoints: CustomEndpointConfig{
					ObservabilityCustomEndpoint: testCustomEndpoint,
				},
				ProviderDefaultRegion: "eu01",
			},
			want: func() observability.DefaultAPI {
				apiClient, err := observability.NewAPIClient(
					config.WithUserAgent("stackit-terraform-provider/1.2.3"),
					config.WithEndpoint(testCustomEndpoint),
					config.WithCustomAuth(testRoundTripper),
				)
				if err != nil {
					t.Fatalf("error configuring client: %v", err)
				}
				return apiClient.DefaultAPI
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &DefaultClientFactory{
				RoundTripper:          tt.fields.RoundTripper,
				UserAgent:             tt.fields.UserAgent,
				CustomEndpoints:       tt.fields.CustomEndpoints,
				ProviderDefaultRegion: tt.fields.ProviderDefaultRegion,
			}
			got, err := f.newObservabilityV1Client()
			if (err != nil) != tt.wantErr {
				t.Fatalf("newObservabilityV1Client() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("newObservabilityV1Client() got = %v, want %v", got, tt.want)
			}
		})
	}
}
