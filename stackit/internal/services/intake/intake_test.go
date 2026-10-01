package intake_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	intake "github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestIntakesSavesIDsOnError(t *testing.T) {
	var (
		projectId = uuid.NewString()
		runnerId  = uuid.NewString()
		intakeId  = uuid.NewString()
	)
	const region = "eu01"
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
  intake_custom_endpoint = "%s"
  service_account_token = "mock-server-needs-no-auth"
  default_region = "%s"
}

resource "stackit_intakes" "example" {
  project_id        = "%s"
  runner_id         = "%s"
  display_name      = "example-intake"
  catalog_uri       = "https://catalog.iceberg.eu01.onstackit.cloud"
  catalog_warehouse = "default"
}
`, s.Server.URL, region, projectId, runnerId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create intake",
							ToJsonBody: &intake.IntakeResponse{
								Id:             intakeId,
								DisplayName:    "example-intake",
								IntakeRunnerId: runnerId,
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating intake.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1beta/projects/%s/regions/%s/intakes/%s", projectId, region, intakeId)
								if req.URL.Path != expected {
									t.Errorf("unexpected URL path: got %s, want %s", req.URL.Path, expected)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{
							Description: "delete",
							StatusCode:  http.StatusAccepted,
						},
						testutil.MockResponse{
							Description: "delete waiter",
							StatusCode:  http.StatusNotFound,
						},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading intake.*"),
			},
		},
	})
}
