// Copyright (c) STACKIT

package stackit_test

import (
	_ "embed"
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/provider-credentials.tf
var providerCredentialConfig string

//go:embed testdata/provider-invalid-attribute.tf
var providerInvalidAttribute string

//go:embed testdata/provider-all-attributes.tf
var providerValidAttributes string

var testConfigProviderCredentials = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-prov%s", acctest.RandStringFromCharSet(3, acctest.CharSetAlphaNum))),
}

// Helper function to obtain the home directory on different systems.
// Based on os.UserHomeDir().
func getHomeEnvVariableName() string {
	env := "HOME"
	switch runtime.GOOS {
	case "windows":
		env = "USERPROFILE"
	case "plan9":
		env = "home"
	}
	return env
}

// create temporary home and initialize the credentials file as well
func createTemporaryHome(createValidCredentialsFile bool, t *testing.T) string {
	// create a temporary file
	tempHome, err := os.MkdirTemp("", "tempHome")
	if err != nil {
		t.Fatalf("Failed to create temporary home directory: %v", err)
	}

	// create credentials file in temp directory
	stackitFolder := path.Join(tempHome, ".stackit")
	if err := os.Mkdir(stackitFolder, 0o750); err != nil {
		t.Fatalf("Failed to create stackit folder: %v", err)
	}

	filePath := path.Join(stackitFolder, "credentials.json")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Error while closing the file: %v", err)
		}
	}()

	// Define content, default = invalid token
	token := "foo_token"
	if createValidCredentialsFile {
		token = testutil.GetTestProjectServiceAccountToken("")
	}
	content := fmt.Sprintf(`
		{
    		"STACKIT_SERVICE_ACCOUNT_TOKEN": "%s"
		}`, token)

	if _, err = file.WriteString(content); err != nil {
		t.Fatalf("Error writing to file: %v", err)
	}

	return tempHome
}

// Function to overwrite the home folder
func setTemporaryHome(tempHomePath string) {
	env := getHomeEnvVariableName()
	if err := os.Setenv(env, tempHomePath); err != nil {
		fmt.Printf("Error setting temporary home directory %v", err)
	}
}

// cleanup the temporary home and reset the environment variable
func cleanupTemporaryHome(tempHomePath string, t *testing.T) {
	if err := os.RemoveAll(tempHomePath); err != nil {
		t.Fatalf("Error cleaning up temporary folder: %v", err)
	}
	originalHomeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to restore home directory back to normal: %v", err)
	}
	// revert back to original home folder
	env := getHomeEnvVariableName()
	if err := os.Setenv(env, originalHomeDir); err != nil {
		fmt.Printf("Error resetting temporary home directory %v", err)
	}
}

func getServiceAccountToken() (string, error) {
	token, set := os.LookupEnv("TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_TOKEN")
	if !set || token == "" {
		return "", fmt.Errorf("Token not set, please set TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_TOKEN to a valid token to perform tests")
	}
	return token, nil
}

func TestAccEnvVarTokenValid(t *testing.T) {
	// Check if acceptance tests should be run
	if v := os.Getenv(resource.EnvTfAcc); v == "" {
		t.Skipf(
			"Acceptance tests skipped unless env '%s' set",
			resource.EnvTfAcc)
		return
	}

	token, err := getServiceAccountToken()
	if err != nil {
		t.Fatalf("Can't get token: %v", err)
	}

	t.Setenv("STACKIT_CREDENTIALS_PATH", "")
	t.Setenv("STACKIT_SERVICE_ACCOUNT_TOKEN", token)
	tempHomeFolder := createTemporaryHome(false, t)
	defer cleanupTemporaryHome(tempHomeFolder, t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig:       func() { setTemporaryHome(tempHomeFolder) },
				ConfigVariables: testConfigProviderCredentials,
				Config:          providerCredentialConfig,
			},
		},
	})
}

func TestAccEnvVarTokenInvalid(t *testing.T) {
	t.Setenv("STACKIT_CREDENTIALS_PATH", "")
	t.Setenv("STACKIT_SERVICE_ACCOUNT_TOKEN", "foo")
	tempHomeFolder := createTemporaryHome(false, t)
	defer cleanupTemporaryHome(tempHomeFolder, t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig:       func() { setTemporaryHome(tempHomeFolder) },
				ConfigVariables: testConfigProviderCredentials,
				Config:          providerCredentialConfig,
				ExpectError:     regexp.MustCompile(`undefined response type, status code 401`),
			},
		},
	})
}

func TestAccCredentialsFileValid(t *testing.T) {
	t.Setenv("STACKIT_CREDENTIALS_PATH", "")
	t.Setenv("STACKIT_SERVICE_ACCOUNT_TOKEN", "")
	tempHomeFolder := createTemporaryHome(true, t)
	defer cleanupTemporaryHome(tempHomeFolder, t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig:       func() { setTemporaryHome(tempHomeFolder) },
				ConfigVariables: testConfigProviderCredentials,
				Config:          providerCredentialConfig,
			},
		},
	})
}

func TestAccCredentialsFileInvalid(t *testing.T) {
	t.Setenv("STACKIT_CREDENTIALS_PATH", "")
	t.Setenv("STACKIT_SERVICE_ACCOUNT_TOKEN", "")
	tempHomeFolder := createTemporaryHome(false, t)
	defer cleanupTemporaryHome(tempHomeFolder, t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig:       func() { setTemporaryHome(tempHomeFolder) },
				ConfigVariables: testConfigProviderCredentials,
				Config:          providerCredentialConfig,
				ExpectError:     regexp.MustCompile(`Jwt is not in(\r\n|\r|\n)the form of Header.Payload.Signature`),
			},
		},
	})
}

func TestAccProviderConfigureValidValues(t *testing.T) {
	// Check if acceptance tests should be run
	if v := os.Getenv(resource.EnvTfAcc); v == "" {
		t.Skipf(
			"Acceptance tests skipped unless env '%s' set",
			resource.EnvTfAcc)
		return
	}
	// use service account token for these tests
	token, err := getServiceAccountToken()
	if err != nil {
		t.Fatalf("Can't get token: %v", err)
	}

	t.Setenv("STACKIT_CREDENTIALS_PATH", "")
	t.Setenv("STACKIT_SERVICE_ACCOUNT_TOKEN", token)
	tempHomeFolder := createTemporaryHome(true, t)
	defer cleanupTemporaryHome(tempHomeFolder, t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // valid provider attributes
				ConfigVariables: testConfigProviderCredentials,
				Config:          providerValidAttributes,
			},
		},
	})
}

func TestAccProviderConfigureAnInvalidValue(t *testing.T) {
	// Check if acceptance tests should be run
	if v := os.Getenv(resource.EnvTfAcc); v == "" {
		t.Skipf(
			"Acceptance tests skipped unless env '%s' set",
			resource.EnvTfAcc)
		return
	}
	// use service account token for these tests
	token, err := getServiceAccountToken()
	if err != nil {
		t.Fatalf("Can't get token: %v", err)
	}

	t.Setenv("STACKIT_CREDENTIALS_PATH", "")
	t.Setenv("STACKIT_SERVICE_ACCOUNT_TOKEN", token)
	tempHomeFolder := createTemporaryHome(true, t)
	defer cleanupTemporaryHome(tempHomeFolder, t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // invalid test attribute should throw an error
				ConfigVariables: testConfigProviderCredentials,
				Config:          providerInvalidAttribute,
				ExpectError:     regexp.MustCompile(`An argument named "test" is not expected here\.`),
			},
		},
	})
}
