package observability

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestObservabilityInstanceSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	planId := uuid.NewString()
	instanceId := uuid.NewString()
	const (
		region   = "eu01"
		name     = "observability-instance"
		planName = "Observability-Medium-EU01"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	observability_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}
resource "stackit_observability_instance" "instance" {
  project_id = "%s"
  name       = "%s"
  plan_name  = "%s"
}
`, region, s.Server.URL, projectId, name, planName)

	planList := testutil.MockResponse{
		Description: "plan list",
		ToJsonBody: observability.PlansResponse{
			Plans: utils.Ptr([]observability.Plan{
				{
					Name:   utils.Ptr(planName),
					PlanId: utils.Ptr(planId),
				},
			}),
		},
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						planList,
						planList,
						planList,
						testutil.MockResponse{
							Description: "create instance",
							ToJsonBody: observability.CreateInstanceResponse{
								InstanceId: utils.Ptr(instanceId),
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating instance.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/instances/%s", projectId, instanceId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{
							Description: "delete waiter",
							ToJsonBody: observability.GetInstanceResponse{
								Id:     utils.Ptr(instanceId),
								Status: observability.GETINSTANCERESPONSESTATUS_DELETE_SUCCEEDED.Ptr(),
							},
						},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading instance*"),
			},
		},
	})
}

func TestObservabilityScrapeConfigSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	const (
		region = "eu01"
		name   = "scrape-config"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	observability_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}
resource "stackit_observability_scrapeconfig" "instance" {
  project_id   = "%s"
  instance_id  = "%s"
  name         = "%s"
  metrics_path = "/my-metrics"
  targets = [
    {
      urls = ["url1", "urls2"]
      labels = {
        "url1" = "dev"
      }
    }
  ]
}
`, region, s.Server.URL, projectId, instanceId, name)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create scrape config",
							ToJsonBody:  observability.ScrapeConfigsResponse{},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating scrape config.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/instances/%s/scrapeconfigs/%s", projectId, instanceId, name)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{
							Description: "delete waiter",
							ToJsonBody: observability.ListScrapeConfigsResponse{
								Data: utils.Ptr([]observability.Job{}),
							},
						},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading scrape config*"),
			},
		},
	})
}
