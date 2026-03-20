package mariadb

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/mariadb"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestMariaDBInstanceSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	planId := uuid.NewString()
	const (
		region   = "eu01"
		version  = "10.11"
		planName = "mariadb-plan"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	mariadb_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
	enable_beta_resources = true
}
resource "stackit_mariadb_instance" "example" {
  project_id = "%s"
  name       = "example-instance"
  version    = "%s"
  plan_name  = "%s"
  parameters = {
    sgw_acl = "193.148.160.0/19,45.129.40.0/21,45.135.244.0/22"
  }
}
`, region, s.Server.URL, projectId, version, planName)

	planList := testutil.MockResponse{
		Description: "plan instance",
		ToJsonBody: mariadb.ListOfferingsResponse{
			Offerings: &[]mariadb.Offering{
				{
					Plans: &[]mariadb.Plan{
						{
							Id:   utils.Ptr(planId),
							Name: utils.Ptr(planName),
						},
					},
					Version: utils.Ptr(version),
				},
			},
		},
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						planList,
						testutil.MockResponse{
							Description: "create instance",
							ToJsonBody: mariadb.CreateInstanceResponse{
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
							StatusCode:  http.StatusGone,
						},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading instance.*"),
			},
		},
	})
}

func TestMariaDBCredentialsSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	credentialId := uuid.NewString()
	const (
		region = "eu01"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	mariadb_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
	enable_beta_resources = true
}
resource "stackit_mariadb_credential" "example" {
  project_id  = "%s"
  instance_id = "%s"
}
`, region, s.Server.URL, projectId, instanceId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create credentials",
							ToJsonBody: mariadb.CredentialsResponse{
								Id: utils.Ptr(credentialId),
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
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
				ExpectError:  regexp.MustCompile("Error reading credential.*"),
			},
		},
	})
}
