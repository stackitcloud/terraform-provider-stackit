package opensearch

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestOpensearchInstanceSavesIDsOnError(t *testing.T) {
	var (
		projectId  = uuid.NewString()
		instanceId = uuid.NewString()
	)
	const (
		name     = "opensearch-instance-test"
		version  = "version"
		planName = "plan-name"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
  opensearch_custom_endpoint = "%s"
  service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_opensearch_instance" "instance" {
  project_id = "%s"
  name = "%s"
  version = "%s"
  plan_name = "%s"
}
`, s.Server.URL, projectId, name, version, planName)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "offerings",
							ToJsonBody: &opensearch.ListOfferingsResponse{
								Offerings: &[]opensearch.Offering{
									{
										Name:    new("offering-name"),
										Version: utils.Ptr(version),
										Plans: &[]opensearch.Plan{
											{
												Id:   new("plan-id"),
												Name: utils.Ptr(planName),
											},
										},
									},
								},
							},
						},
						testutil.MockResponse{
							Description: "create instance",
							ToJsonBody: &opensearch.CreateInstanceResponse{
								InstanceId: new(instanceId),
							},
						},
						testutil.MockResponse{Description: "failing waiter", StatusCode: http.StatusInternalServerError},
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
									t.Errorf(fmt.Sprintf("unexpected URL path: got %s, want %s", req.URL.Path, expected), http.StatusBadRequest)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete"},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusGone},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading instance.*"),
			},
		},
	})
}

func TestOpensearchCredentialSavesIDsOnError(t *testing.T) {
	var (
		projectId    = uuid.NewString()
		instanceId   = uuid.NewString()
		credentialId = uuid.NewString()
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
  opensearch_custom_endpoint = "%s"
  service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_opensearch_credential" "credential" {
  project_id = "%s"
  instance_id = "%s"
}
`, s.Server.URL, projectId, instanceId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create credential",
							ToJsonBody: &opensearch.CredentialsResponse{
								Id: new(credentialId),
							},
						},
						testutil.MockResponse{Description: "create waiter", StatusCode: http.StatusInternalServerError},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating credential.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/instances/%s/credentials/%s", projectId, instanceId, credentialId)
								if req.URL.Path != expected {
									t.Errorf(fmt.Sprintf("unexpected URL path: got %s, want %s", req.URL.Path, expected), http.StatusBadRequest)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete"},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusGone},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading credential.*"),
			},
		},
	})
}
