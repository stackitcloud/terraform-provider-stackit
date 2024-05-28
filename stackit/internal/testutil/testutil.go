package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

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

	// ProjectId is the id of project used for tests
	ProjectId = os.Getenv("TF_ACC_PROJECT_ID")
	// TestProjectParentContainerID is the container id of the parent resource under which projects are created as part of the resource-manager acceptance tests
	TestProjectParentContainerID = os.Getenv("TF_ACC_TEST_PROJECT_PARENT_CONTAINER_ID")
	// TestProjectParentContainerID is the uuid of the parent resource under which projects are created as part of the resource-manager acceptance tests
	TestProjectParentUUID = os.Getenv("TF_ACC_TEST_PROJECT_PARENT_UUID")
	// TestProjectServiceAccountEmail is the e-mail of a service account with admin permissions on the organization under which projects are created as part of the resource-manager acceptance tests
	TestProjectServiceAccountEmail = os.Getenv("TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_EMAIL")

	ArgusCustomEndpoint           = os.Getenv("TF_ACC_ARGUS_CUSTOM_ENDPOINT")
	DnsCustomEndpoint             = os.Getenv("TF_ACC_DNS_CUSTOM_ENDPOINT")
	IaaSCustomEndpoint            = os.Getenv("TF_ACC_IAAS_CUSTOM_ENDPOINT")
	LoadBalancerCustomEndpoint    = os.Getenv("TF_ACC_LOADBALANCER_CUSTOM_ENDPOINT")
	LogMeCustomEndpoint           = os.Getenv("TF_ACC_LOGME_CUSTOM_ENDPOINT")
	MariaDBCustomEndpoint         = os.Getenv("TF_ACC_MARIADB_CUSTOM_ENDPOINT")
	MongoDBFlexCustomEndpoint     = os.Getenv("TF_ACC_MONGODBFLEX_CUSTOM_ENDPOINT")
	OpenSearchCustomEndpoint      = os.Getenv("TF_ACC_OPENSEARCH_CUSTOM_ENDPOINT")
	ObjectStorageCustomEndpoint   = os.Getenv("TF_ACC_OBJECTSTORAGE_CUSTOM_ENDPOINT")
	PostgreSQLCustomEndpoint      = os.Getenv("TF_ACC_POSTGRESQL_CUSTOM_ENDPOINT")
	PostgresFlexCustomEndpoint    = os.Getenv("TF_ACC_POSTGRESFLEX_CUSTOM_ENDPOINT")
	RabbitMQCustomEndpoint        = os.Getenv("TF_ACC_RABBITMQ_CUSTOM_ENDPOINT")
	RedisCustomEndpoint           = os.Getenv("TF_ACC_REDIS_CUSTOM_ENDPOINT")
	ResourceManagerCustomEndpoint = os.Getenv("TF_ACC_RESOURCEMANAGER_CUSTOM_ENDPOINT")
	SecretsManagerCustomEndpoint  = os.Getenv("TF_ACC_SECRETSMANAGER_CUSTOM_ENDPOINT")
	SQLServerFlexCustomEndpoint   = os.Getenv("TF_ACC_SQLSERVERFLEX_CUSTOM_ENDPOINT")
	SKECustomEndpoint             = os.Getenv("TF_ACC_SKE_CUSTOM_ENDPOINT")

	// OpenStack user domain name
	OSUserDomainName = os.Getenv("TF_ACC_OS_USER_DOMAIN_NAME")
	// OpenStack user name
	OSUserName = os.Getenv("TF_ACC_OS_USER_NAME")
	// OpenStack password
	OSPassword = os.Getenv("TF_ACC_OS_PASSWORD")
)

// Provider config helper functions

func ArgusProviderConfig() string {
	if ArgusCustomEndpoint == "" {
		return `provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			argus_custom_endpoint = "%s"
		}`,
		ArgusCustomEndpoint,
	)
}

func DnsProviderConfig() string {
	if DnsCustomEndpoint == "" {
		return `provider "stackit" {}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			dns_custom_endpoint = "%s"
		}`,
		DnsCustomEndpoint,
	)
}

func IaaSProviderConfig() string {
	if IaaSCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			iaas_custom_endpoint = "%s"
		}`,
		IaaSCustomEndpoint,
	)
}

func LoadBalancerProviderConfig() string {
	if LoadBalancerCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			loadbalancer_custom_endpoint = "%s"
		}`,
		LoadBalancerCustomEndpoint,
	)
}

func LogMeProviderConfig() string {
	if LogMeCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			logme_custom_endpoint = "%s"
		}`,
		LogMeCustomEndpoint,
	)
}

func MariaDBProviderConfig() string {
	if MariaDBCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			mariadb_custom_endpoint = "%s"
		}`,
		MariaDBCustomEndpoint,
	)
}

func MongoDBFlexProviderConfig() string {
	if MongoDBFlexCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			mongodbflex_custom_endpoint = "%s"
		}`,
		MongoDBFlexCustomEndpoint,
	)
}

func ObjectStorageProviderConfig() string {
	if ObjectStorageCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			objectstorage_custom_endpoint = "%s"
		}`,
		ObjectStorageCustomEndpoint,
	)
}

func OpenSearchProviderConfig() string {
	if OpenSearchCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			opensearch_custom_endpoint = "%s"
		}`,
		OpenSearchCustomEndpoint,
	)
}

func PostgreSQLProviderConfig() string {
	if PostgreSQLCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			postgresql_custom_endpoint = "%s"
		}`,
		PostgreSQLCustomEndpoint,
	)
}

func PostgresFlexProviderConfig() string {
	if PostgresFlexCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			postgresflex_custom_endpoint = "%s"
		}`,
		PostgresFlexCustomEndpoint,
	)
}

func RabbitMQProviderConfig() string {
	if RabbitMQCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			rabbitmq_custom_endpoint = "%s"
		}`,
		RabbitMQCustomEndpoint,
	)
}

func RedisProviderConfig() string {
	if RedisCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			redis_custom_endpoint = "%s"
		}`,
		RedisCustomEndpoint,
	)
}

func ResourceManagerProviderConfig() string {
	token := getTestProjectServiceAccountToken("")
	if ResourceManagerCustomEndpoint == "" {
		return fmt.Sprintf(`
		provider "stackit" {
			service_account_email = "%s"
			service_account_token = "%s"
		}`,
			TestProjectServiceAccountEmail,
			token,
		)
	}
	return fmt.Sprintf(`
	provider "stackit" {
		resourcemanager_custom_endpoint = "%s"
		service_account_email = "%s"
		service_account_token = "%s"
	}`,
		ResourceManagerCustomEndpoint,
		TestProjectServiceAccountEmail,
		token,
	)
}

func SecretsManagerProviderConfig() string {
	if SecretsManagerCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			secretsmanager_custom_endpoint = "%s"
		}`,
		SecretsManagerCustomEndpoint,
	)
}

func SQLServerFlexProviderConfig() string {
	if MongoDBFlexCustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			sqlserverlex_custom_endpoint = "%s"
		}`,
		SQLServerFlexCustomEndpoint,
	)
}

func SKEProviderConfig() string {
	if SKECustomEndpoint == "" {
		return `
		provider "stackit" {
			region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			ske_custom_endpoint = "%s"
		}`,
		SKECustomEndpoint,
	)
}

func ResourceNameWithDateTime(name string) string {
	dateTime := time.Now().Format(time.RFC3339)
	// Remove timezone to have a smaller datetime
	dateTimeTrimmed, _, _ := strings.Cut(dateTime, "+")
	return fmt.Sprintf("tf-acc-%s-%s", name, dateTimeTrimmed)
}

func getTestProjectServiceAccountToken(path string) string {
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
