package logs

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	logs "github.com/stackitcloud/stackit-sdk-go/services/logs/v1api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestLogsInstanceSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	const (
		region = "eu01"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	logs_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}
resource "stackit_logs_instance" "logs" {
  project_id     = "%s"
  display_name   = "logs-instance-example"
  retention_days = 30
}
`, region, s.Server.URL, projectId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create instance",
							ToJsonBody: logs.LogsInstance{
								Id: instanceId,
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating Logs Instance.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/instances/%s", projectId, region, instanceId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{
							Description: "delete waiter",
							StatusCode:  http.StatusNotFound,
						},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading logs instance.*"),
			},
		},
	})
}
