package containerregistry_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	registry "github.com/stackitcloud/stackit-sdk-go/services/registry/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestContainerRegistryArtifactorySavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	artifactoryId := uuid.NewString()

	const (
		name   = "my-artifactory"
		region = "eu01"
	)

	s := testutil.NewMockServer(t)
	defer s.Server.Close()

	tfConfig := fmt.Sprintf(`
provider "stackit" {
  container_registry_custom_endpoint = "%s"
  service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_containerregistry_artifactory" "this" {
  project_id = "%s"
  region     = "%s"
  name       = "%s"
}
`, s.Server.URL, projectId, region, name)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create artifactory",
							ToJsonBody: registry.Artifactory{
								Id:   artifactoryId,
								Name: name,
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating artifactory.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/artifactories/%s", projectId, region, artifactoryId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}

								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete"},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusNotFound},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading artifactory.*"),
			},
		},
	})
}
