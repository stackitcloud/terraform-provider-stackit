package containerregistry_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	registry "github.com/stackitcloud/stackit-sdk-go/services/registry/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceMin string

//go:embed testdata/resource-max.tf
var resourceMax string

//go:embed testdata/datasource-artifactories*
var datasourceArtifactories string

//go:embed testdata/datasource-replication-adapters.tf
var datasourceReplicationAdapters string

var (
	artifactoryNameMin = fmt.Sprintf("cr-min-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
	artifactoryNameMax = fmt.Sprintf("cr-max-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))

	testConfigVarsMin = config.Variables{
		"project_id": config.StringVariable(testutil.ProjectId),
		"name":       config.StringVariable(artifactoryNameMin),
	}
)

func TestAccContainerRegistryArtifactoryLifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContainerRegistryArtifactoryDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create minimal Artifactory
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMin,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_containerregistry_artifactory.this", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_containerregistry_artifactory.this", "name", artifactoryNameMin),
					resource.TestCheckResourceAttrSet("stackit_containerregistry_artifactory.this", "id"),
					resource.TestCheckResourceAttrSet("stackit_containerregistry_artifactory.this", "artifactory_id"),
					resource.TestCheckResourceAttrSet("stackit_containerregistry_artifactory.this", "url"),
				),
			},
			// Step 2: Read via Artifactory DataSource
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMin + `
data "stackit_containerregistry_artifactory" "this" {
  project_id     = stackit_containerregistry_artifactory.this.project_id
  artifactory_id = stackit_containerregistry_artifactory.this.artifactory_id
}`,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"stackit_containerregistry_artifactory.this", "artifactory_id",
						"data.stackit_containerregistry_artifactory.this", "artifactory_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_containerregistry_artifactory.this", "name",
						"data.stackit_containerregistry_artifactory.this", "name",
					),
				),
			},
			// Step 3: Read via Artifactories (plural) DataSource
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMin + `
data "stackit_containerregistry_artifactories" "this" {
  project_id = stackit_containerregistry_artifactory.this.project_id
}`,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_containerregistry_artifactories.this", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("data.stackit_containerregistry_artifactories.this", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_containerregistry_artifactories.this", "artifactories.#"),
				),
			},
			// Step 4: Import state
			{
				ConfigVariables:   testConfigVarsMin,
				ResourceName:      "stackit_containerregistry_artifactory.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccContainerRegistryArtifactoryMax(t *testing.T) {
	dockerUsername := os.Getenv("TF_ACC_CONTAINER_REGISTRY_DOCKER_USERNAME")
	dockerPassword := os.Getenv("TF_ACC_CONTAINER_REGISTRY_DOCKER_PASSWORD")
	if dockerUsername == "" || dockerPassword == "" {
		t.Skip("Skipping TestAccContainerRegistryArtifactoryMax: TF_ACC_CONTAINER_REGISTRY_DOCKER_USERNAME or TF_ACC_CONTAINER_REGISTRY_DOCKER_PASSWORD not set")
	}

	testConfigVarsMax := config.Variables{
		"project_id":              config.StringVariable(testutil.ProjectId),
		"name":                    config.StringVariable(artifactoryNameMax),
		"docker_username":         config.StringVariable(dockerUsername),
		"docker_password":         config.StringVariable(dockerPassword),
		"docker_password_version": config.IntegerVariable(1),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContainerRegistryArtifactoryDestroy,
		Steps: []resource.TestStep{
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMax,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_containerregistry_artifactory.this", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_containerregistry_artifactory.this", "name", artifactoryNameMax),
					resource.TestCheckResourceAttr("stackit_containerregistry_artifactory.this", "proxy.registry_type", "docker-hub"),
					resource.TestCheckResourceAttr("stackit_containerregistry_artifactory.this", "proxy.url", "https://registry-1.docker.io"),
					resource.TestCheckResourceAttr("stackit_containerregistry_artifactory.this", "proxy.description", "Docker Hub mirror"),
					resource.TestCheckResourceAttr("stackit_containerregistry_artifactory.this", "proxy.credentials.access_key", dockerUsername),
					resource.TestCheckResourceAttr("stackit_containerregistry_artifactory.this", "proxy.credentials.access_secret_wo_version", "1"),
					resource.TestCheckNoResourceAttr("stackit_containerregistry_artifactory.this", "proxy.credentials.access_secret"),
					resource.TestCheckNoResourceAttr("stackit_containerregistry_artifactory.this", "proxy.credentials.access_secret_wo"),
					resource.TestCheckResourceAttrSet("stackit_containerregistry_artifactory.this", "id"),
				),
			},
		},
	})
}

func TestAccContainerRegistryArtifactoriesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + datasourceArtifactories,
				ConfigVariables: config.Variables{
					"project_id": config.StringVariable(testutil.ProjectId),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_containerregistry_artifactories.this", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("data.stackit_containerregistry_artifactories.this", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_containerregistry_artifactories.this", "artifactories.#"),
				),
			},
		},
	})
}

func TestAccContainerRegistryReplicationAdaptersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + datasourceReplicationAdapters,
				ConfigVariables: config.Variables{
					"project_id": config.StringVariable(testutil.ProjectId),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_containerregistry_replication_adapters.this", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("data.stackit_containerregistry_replication_adapters.this", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_containerregistry_replication_adapters.this", "replication_adapters.#"),
				),
			},
		},
	})
}

func testAccCheckContainerRegistryArtifactoryDestroy(s *terraform.State) error {
	ctx := context.Background()

	client, err := registry.NewAPIClient(
		testutil.NewConfigBuilder().BuildClientOptions(testutil.ContainerRegistryCustomEndpoint, false)...,
	)
	if err != nil {
		return fmt.Errorf("creating API client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_containerregistry_artifactory" {
			continue
		}

		projectId := rs.Primary.Attributes["project_id"]
		region := rs.Primary.Attributes["region"]
		artifactoryId := rs.Primary.Attributes["artifactory_id"]

		artifactory, err := client.DefaultAPI.GetArtifactory(ctx, projectId, region, artifactoryId).Execute()
		if err == nil {
			if artifactory != nil && artifactory.State != registry.ARTIFACTORYSTATE_DELETED {
				return fmt.Errorf("artifactory %q still exists with state %q", artifactoryId, artifactory.State)
			}
			continue
		}

		var oapiErr *oapierror.GenericOpenAPIError
		if !errors.As(err, &oapiErr) || oapiErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("verifying artifactory destroyed: %w", err)
		}
	}

	return nil
}
