package instance_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/services/ufw/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestUfwInstanceResource(t *testing.T) {
	projectId := uuid.NewString()
	ruleId := uuid.NewString()
	region := "eu01"

	s := testutil.NewMockServer(t)
	defer s.Server.Close()

	tfConfig := fmt.Sprintf(`
       provider "stackit" {
          ufw_custom_endpoint   = "%s"
          service_account_token = "mock-server-needs-no-auth"
       }

       resource "stackit_ufw_instance" "example" {
          project_id  = "%s"
          region      = "%s"
          instance_id = "target-instance-123"
          product     = "edge-cloud"
          source_ip   = "192.168.0.0/24"
          type        = "ACL"
       }
    `, s.Server.URL, projectId, region)

	tfConfigUpdated := fmt.Sprintf(`
       provider "stackit" {
          ufw_custom_endpoint   = "%s"
          service_account_token = "mock-server-needs-no-auth"
       }

       resource "stackit_ufw_instance" "example" {
          project_id  = "%s"
          region      = "%s"
          instance_id = "target-instance-123"
          product     = "edge-cloud"
          source_ip   = "10.0.0.0/8"
          type        = "ACL"
       }
    `, s.Server.URL, projectId, region)

	validRuleResponse := v1api.RuleResponse{
		Destination: "0.0.0.0/0",
		InstanceId:  "target-instance-123",
		Product:     "edge-cloud",
		SourceIP:    "192.168.0.0/24",
		Status:      v1api.RULERESPONSESTATUS_ACTIVE,
		Type:        "ACL",
	}

	validRuleResponseUpdated := validRuleResponse
	validRuleResponseUpdated.SourceIP = "10.0.0.0/8"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "Create UFW Rule",
							ToJsonBody: v1api.CreateRuleResponse{
								RefId: &ruleId,
							},
						},
						testutil.MockResponse{
							Description: "Get UFW Rule (Create Waiter)",
							ToJsonBody:  validRuleResponse,
						},
						testutil.MockResponse{
							Description: "Read UFW Rule",
							ToJsonBody:  validRuleResponse,
						},
					)
				},
				Config: tfConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "project_id", projectId),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "source_ip", "192.168.0.0/24"),
					resource.TestCheckResourceAttrSet("stackit_ufw_instance.example", "rule_id"),
				),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "Read UFW Rule (Plan)",
							ToJsonBody:  validRuleResponse,
						},
						testutil.MockResponse{
							Description: "Delete UFW Rule (Replace)",
							StatusCode:  http.StatusAccepted,
						},
						testutil.MockResponse{
							Description: "Get UFW Rule (Delete Waiter)",
							StatusCode:  http.StatusNotFound,
						},
						testutil.MockResponse{
							Description: "Create UFW Rule (Replace)",
							ToJsonBody: v1api.CreateRuleResponse{
								RefId: &ruleId,
							},
						},
						testutil.MockResponse{
							Description: "Get UFW Rule (Create Waiter)",
							ToJsonBody:  validRuleResponseUpdated,
						},
						testutil.MockResponse{
							Description: "Read UFW Rule (Post-Replace)",
							ToJsonBody:  validRuleResponseUpdated,
						},
						testutil.MockResponse{
							Description: "Delete UFW Rule (Cleanup)",
							StatusCode:  http.StatusAccepted,
						},
						testutil.MockResponse{
							Description: "Get UFW Rule (Delete Waiter)",
							StatusCode:  http.StatusNotFound,
						},
					)
				},
				Config: tfConfigUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "source_ip", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestUfwInstanceSavesIDsOnError(t *testing.T) {
	var (
		projectId = uuid.NewString()
		ruleId    = uuid.NewString()
	)
	const region = "eu01"
	s := testutil.NewMockServer(t)
	defer s.Server.Close()

	tfConfig := fmt.Sprintf(`
       provider "stackit" {
          ufw_custom_endpoint   = "%s"
          service_account_token = "mock-server-needs-no-auth"
       }

       resource "stackit_ufw_instance" "example" {
          project_id  = "%s"
          region      = "%s"
          instance_id = "target-instance-123"
          product     = "edge-cloud"
          source_ip   = "192.168.0.0/24"
          type        = "ACL"
       }
    `, s.Server.URL, projectId, region)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "Create UFW Rule",
							ToJsonBody: &v1api.CreateRuleResponse{
								RefId: &ruleId,
							},
						},
						testutil.MockResponse{Description: "Failing waiter", StatusCode: http.StatusInternalServerError},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating .*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "Refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/rules/%s", projectId, region, ruleId)
								if req.URL.Path != expected {
									t.Errorf("unexpected URL path: got %s, want %s", req.URL.Path, expected)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "Delete UFW Rule", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "Delete Waiter", StatusCode: http.StatusNotFound},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading .*"),
			},
		},
	})
}
