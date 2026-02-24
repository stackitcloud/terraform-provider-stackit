package foo

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestFooSavesIDsOnError(t *testing.T) {
	/* Setup code:
	   - define known values for attributes used in id
	   - create mock server
	   - define minimal tf config with custom endpoint pointing to mock server
	*/
	var (
		projectId = uuid.NewString()
		barId     = uuid.NewString()
	)
	const region = "eu01"
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
  foo_custom_endpoint = "%s"
  service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_foo" "foo" {
  project_id = "%s"
}
`, s.Server.URL, projectId)

	/* Test steps:
	   1. Create resource with mocked backend
	   2. Verify with a refresh, that IDs are saved to state
	*/
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					/* Setup mock responses for create and waiter.
					   The create response succeeds and returns the barId, but the waiter fails with an error.
					   We can't check the state in this step, because the create returns early due to the waiter error.
					   TF won't execute any Checks of the TestStep if there is an error.
					*/
					s.Reset(
						testutil.MockResponse{
							Description: "create foo",
							ToJsonBody: &BarResponse{
								BarId: barId,
							},
						},
						testutil.MockResponse{Description: "failing waiter", StatusCode: http.StatusInternalServerError},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating foo.*"),
			},
			{
				PreConfig: func() {
					/* Setup mock responses for refresh and delete.
					   The refresh response fails with an error, but we want to verify that the URL contains the correct IDs.
					   After the test TF will automatically destroy the resource. So we set up mocks to simulate a successful dlete.
					*/
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/foo/%s", projectId, region, barId)
								if req.URL.Path != expected {
									t.Errorf("unexpected URL path: got %s, want %s", req.URL.Path, expected)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete"},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusGone},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading foo.*"),
			},
		},
	})
}
