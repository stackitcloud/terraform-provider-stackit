package scf

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/services/scf"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestScfOrganizationSavesIDsOnError(t *testing.T) {
	var (
		projectId = uuid.NewString()
		guid      = uuid.NewString()
	)
	const name = "scf-org-error-test"
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
  default_region = "eu01"
  scf_custom_endpoint = "%s"
  service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_scf_organization" "org" {
  project_id = "%s"
  name       = "%s"
}
`, s.Server.URL, projectId, name)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create",
							ToJsonBody: &scf.OrganizationCreateResponse{
								Guid: new(guid),
							},
						},
						testutil.MockResponse{Description: "create waiter", StatusCode: http.StatusNotFound},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating scf organization.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/organizations/%s", projectId, region, guid)
								if req.URL.Path != expected {
									t.Errorf("Expected request to %s but got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete"},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusNotFound},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading scf organization.*"),
			},
		},
	})
}
