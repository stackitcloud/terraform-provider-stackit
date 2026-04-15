package sfs

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestSfsResourcePoolSavesIDsOnError(t *testing.T) {
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
	sfs_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
	enable_beta_resources = true
}
resource "stackit_sfs_resource_pool" "resourcepool" {
  project_id        = "%s"
  name              = "sfs-instance"
  availability_zone = "eu01-m"
  performance_class = "Standard"
  size_gigabytes    = 512
  ip_acl            = ["192.168.2.0/24"]
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
							ToJsonBody: sfs.CreateResourcePoolResponse{
								ResourcePool: &sfs.CreateResourcePoolResponseResourcePool{
									Id: new(instanceId),
								},
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating resource pool.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/resourcePools/%s", projectId, region, instanceId)
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
				ExpectError:  regexp.MustCompile("Error reading resource pool*"),
			},
		},
	})
}

func TestSfsShareSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	resourcePoolId := uuid.NewString()
	const (
		region = "eu01"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	sfs_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
	enable_beta_resources = true
}
resource "stackit_sfs_share" "example" {
  project_id                 = "%s"
  resource_pool_id           = "%s"
  name                       = "my-nfs-share"
  export_policy              = "high-performance-class"
  space_hard_limit_gigabytes = 32
}
`, region, s.Server.URL, projectId, resourcePoolId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create instance",
							ToJsonBody: sfs.CreateShareResponse{
								Share: &sfs.CreateShareResponseShare{
									Id: new(instanceId),
								},
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating share.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/resourcePools/%s/shares/%s", projectId, region, resourcePoolId, instanceId)
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
				ExpectError:  regexp.MustCompile("Error reading share*"),
			},
		},
	})
}
