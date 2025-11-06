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
	"github.com/hashicorp/terraform-plugin-testing/config"

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

	// E2ETestsEnabled checks if end-to-end tests should be run.
	// It is enabled when the TF_ACC environment variable is set to "1".
	E2ETestsEnabled = os.Getenv("TF_ACC") == "1"
	// OrganizationId is the id of organization used for tests
	OrganizationId = os.Getenv("TF_ACC_ORGANIZATION_ID")
	// ProjectId is the id of project used for tests
	ProjectId = os.Getenv("TF_ACC_PROJECT_ID")
	Region    = os.Getenv("TF_ACC_REGION")
	// ServerId is the id of a server used for some tests
	ServerId = getenv("TF_ACC_SERVER_ID", "")
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

	CdnCustomEndpoint             = os.Getenv("TF_ACC_CDN_CUSTOM_ENDPOINT")
	DnsCustomEndpoint             = os.Getenv("TF_ACC_DNS_CUSTOM_ENDPOINT")
	GitCustomEndpoint             = os.Getenv("TF_ACC_GIT_CUSTOM_ENDPOINT")
	IaaSCustomEndpoint            = os.Getenv("TF_ACC_IAAS_CUSTOM_ENDPOINT")
	LoadBalancerCustomEndpoint    = os.Getenv("TF_ACC_LOADBALANCER_CUSTOM_ENDPOINT")
	LogMeCustomEndpoint           = os.Getenv("TF_ACC_LOGME_CUSTOM_ENDPOINT")
	MariaDBCustomEndpoint         = os.Getenv("TF_ACC_MARIADB_CUSTOM_ENDPOINT")
	ModelServingCustomEndpoint    = os.Getenv("TF_ACC_MODELSERVING_CUSTOM_ENDPOINT")
	AuthorizationCustomEndpoint   = os.Getenv("TF_ACC_authorization_custom_endpoint")
	MongoDBFlexCustomEndpoint     = os.Getenv("TF_ACC_MONGODBFLEX_CUSTOM_ENDPOINT")
	OpenSearchCustomEndpoint      = os.Getenv("TF_ACC_OPENSEARCH_CUSTOM_ENDPOINT")
	ObservabilityCustomEndpoint   = os.Getenv("TF_ACC_OBSERVABILITY_CUSTOM_ENDPOINT")
	ObjectStorageCustomEndpoint   = os.Getenv("TF_ACC_OBJECTSTORAGE_CUSTOM_ENDPOINT")
	PostgresFlexCustomEndpoint    = os.Getenv("TF_ACC_POSTGRESFLEX_CUSTOM_ENDPOINT")
	RabbitMQCustomEndpoint        = os.Getenv("TF_ACC_RABBITMQ_CUSTOM_ENDPOINT")
	RedisCustomEndpoint           = os.Getenv("TF_ACC_REDIS_CUSTOM_ENDPOINT")
	ResourceManagerCustomEndpoint = os.Getenv("TF_ACC_RESOURCEMANAGER_CUSTOM_ENDPOINT")
	ScfCustomEndpoint             = os.Getenv("TF_ACC_SCF_CUSTOM_ENDPOINT")
	SecretsManagerCustomEndpoint  = os.Getenv("TF_ACC_SECRETSMANAGER_CUSTOM_ENDPOINT")
	SQLServerFlexCustomEndpoint   = os.Getenv("TF_ACC_SQLSERVERFLEX_CUSTOM_ENDPOINT")
	ServerBackupCustomEndpoint    = os.Getenv("TF_ACC_SERVER_BACKUP_CUSTOM_ENDPOINT")
	ServerUpdateCustomEndpoint    = os.Getenv("TF_ACC_SERVER_UPDATE_CUSTOM_ENDPOINT")
	ServiceAccountCustomEndpoint  = os.Getenv("TF_ACC_SERVICE_ACCOUNT_CUSTOM_ENDPOINT")
	SKECustomEndpoint             = os.Getenv("TF_ACC_SKE_CUSTOM_ENDPOINT")
)

// Provider config helper functions

func ObservabilityProviderConfig() string {
	if ObservabilityCustomEndpoint == "" {
		return `provider "stackit" {
			default_region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			observability_custom_endpoint = "%s"
		}`,
		ObservabilityCustomEndpoint,
	)
}
func CdnProviderConfig() string {
	if CdnCustomEndpoint == "" {
		return `
		provider "stackit" {
			enable_beta_resources = true
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			cdn_custom_endpoint = "%s"
			enable_beta_resources = true
		}`,
		CdnCustomEndpoint,
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
			default_region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			iaas_custom_endpoint = "%s"
		}`,
		IaaSCustomEndpoint,
	)
}

func IaaSProviderConfigWithBetaResourcesEnabled() string {
	if IaaSCustomEndpoint == "" {
		return `
		provider "stackit" {
			enable_beta_resources = true
			default_region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			enable_beta_resources = true
			iaas_custom_endpoint = "%s"
		}`,
		IaaSCustomEndpoint,
	)
}

func IaaSProviderConfigWithExperiments() string {
	if IaaSCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
  			experiments = [ "routing-tables", "network" ]
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			iaas_custom_endpoint = "%s"
			experiments = [ "routing-tables", "network" ]
		}`,
		IaaSCustomEndpoint,
	)
}

func LoadBalancerProviderConfig() string {
	if LoadBalancerCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
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
			default_region = "eu01"
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
			default_region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			mariadb_custom_endpoint = "%s"
		}`,
		MariaDBCustomEndpoint,
	)
}

func ModelServingProviderConfig() string {
	if ModelServingCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
		}
		`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			modelserving_custom_endpoint = "%s"
		}`,
		ModelServingCustomEndpoint,
	)
}

func MongoDBFlexProviderConfig() string {
	if MongoDBFlexCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
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
			default_region = "eu01"
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
			default_region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			opensearch_custom_endpoint = "%s"
		}`,
		OpenSearchCustomEndpoint,
	)
}

func PostgresFlexProviderConfig() string {
	if PostgresFlexCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
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
			default_region = "eu01"
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
			default_region = "eu01"
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
	token := GetTestProjectServiceAccountToken("")
	if ResourceManagerCustomEndpoint == "" || AuthorizationCustomEndpoint == "" {
		return fmt.Sprintf(`
		provider "stackit" {
			service_account_token = "%s"
		}`,
			token,
		)
	}
	return fmt.Sprintf(`
	provider "stackit" {
		resourcemanager_custom_endpoint = "%s"
		authorization_custom_endpoint = "%s"
		service_account_token = "%s"
	}`,
		ResourceManagerCustomEndpoint,
		AuthorizationCustomEndpoint,
		token,
	)
}

func SecretsManagerProviderConfig() string {
	if SecretsManagerCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
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
	if SQLServerFlexCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			sqlserverflex_custom_endpoint = "%s"
		}`,
		SQLServerFlexCustomEndpoint,
	)
}

func ServerBackupProviderConfig() string {
	if ServerBackupCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
			enable_beta_resources = true
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			server_backup_custom_endpoint = "%s"
			enable_beta_resources = true
		}`,
		ServerBackupCustomEndpoint,
	)
}

func ServerUpdateProviderConfig() string {
	if ServerUpdateCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
			enable_beta_resources = true
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			server_update_custom_endpoint = "%s"
			enable_beta_resources = true
		}`,
		ServerUpdateCustomEndpoint,
	)
}

func SKEProviderConfig() string {
	if SKECustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			ske_custom_endpoint = "%s"
		}`,
		SKECustomEndpoint,
	)
}

func AuthorizationProviderConfig() string {
	if AuthorizationCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
			experiments = ["iam"]
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			authorization_custom_endpoint = "%s"
			experiments = ["iam"]
		}`,
		AuthorizationCustomEndpoint,
	)
}

func ServiceAccountProviderConfig() string {
	if ServiceAccountCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
			enable_beta_resources = true
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			service_account_custom_endpoint = "%s"
			enable_beta_resources = true
		}`,
		ServiceAccountCustomEndpoint,
	)
}

func GitProviderConfig() string {
	if GitCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
			enable_beta_resources = true
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			git_custom_endpoint = "%s"
			enable_beta_resources = true
		}`,
		GitCustomEndpoint,
	)
}

func ScfProviderConfig() string {
	if ScfCustomEndpoint == "" {
		return `
		provider "stackit" {
			default_region = "eu01"
		}`
	}
	return fmt.Sprintf(`
		provider "stackit" {
			default_region = "eu01"
			scf_custom_endpoint = "%s"
		}`,
		ScfCustomEndpoint,
	)
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
	// In case the variable is a string, the quotes should be removed
	if tmpByteArray[0] == '"' && tmpByteArray[len(tmpByteArray)-1] == '"' {
		result := string(tmpByteArray[1 : len(tmpByteArray)-1])
		// Replace escaped quotes which where added MarshalJSON
		rawString := strings.ReplaceAll(result, `\"`, `"`)
		return rawString
	}
	return string(tmpByteArray)
}
