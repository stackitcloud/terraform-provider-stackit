package workflows_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// TestWorkflowsInstanceSavesIDsOnError confirms that when Create succeeds at
// the API but the subsequent wait fails, the instance_id is persisted to
// state so the user can recover via `terraform import` instead of orphaning
// the server-side instance.
//
// Per CONTRIBUTING.md §99-102 every async resource must carry this regression.
func TestWorkflowsInstanceSavesIDsOnError(t *testing.T) {
	projectID := uuid.NewString()
	instanceID := uuid.NewString()
	const region = "eu01"

	s := testutil.NewMockServer(t)
	defer s.Server.Close()

	tfConfig := fmt.Sprintf(`
provider "stackit" {
  default_region              = "%s"
  workflows_custom_endpoint   = "%s"
  service_account_token       = "mock-server-needs-no-auth"
  experiments                 = ["workflows"]
}
resource "stackit_workflows_instance" "example" {
  project_id   = "%s"
  display_name = "tf-savesid"
  version      = "workflows-3.0-airflow-3.1"
  identity_provider = {
    type               = "oauth2"
    name               = "azure"
    client_id          = "client"
    client_secret      = "secret"
    scope              = "openid"
    discovery_endpoint = "https://idp.example.com/.well-known/openid-configuration"
  }
}
`, region, s.Server.URL, projectID)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create instance returns id",
							ToJsonBody: workflows.Instance{
								Id:          instanceID,
								ProjectId:   projectID,
								RegionId:    region,
								DisplayName: "tf-savesid",
								Status:      workflows.INSTANCESTATUS_CREATING,
								IdentityProvider: workflows.OAuth2IdentityProviderAsIdentityProvider(&workflows.OAuth2IdentityProvider{
									Type:              workflows.OAUTH2IDENTITYPROVIDERTYPE_OAUTH2,
									Name:              "azure",
									ClientId:          "client",
									ClientSecret:      "secret",
									Scope:             "openid",
									DiscoveryEndpoint: "https://idp.example.com/.well-known/openid-configuration",
								}),
							},
						},
						testutil.MockResponse{
							Description: "wait poll fails",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating Workflows instance.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh hits the persisted instance_id",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1alpha/projects/%s/regions/%s/instances/%s", projectID, region, instanceID)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusNotFound},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading Workflows instance.*"),
			},
		},
	})
}
