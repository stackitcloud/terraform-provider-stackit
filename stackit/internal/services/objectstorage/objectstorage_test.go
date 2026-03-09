package objectstorage

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestObjectStorageBucketSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	const (
		name   = "bucket-name"
		region = "eu01"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	objectstorage_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}
resource "stackit_objectstorage_bucket" "instance" {
  project_id = "%s"
  name       = "%s"
}
`, region, s.Server.URL, projectId, name)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "project enable",
							ToJsonBody: objectstorage.ProjectStatus{
								Project: utils.Ptr(projectId),
								Scope:   utils.Ptr(objectstorage.PROJECTSCOPE_PUBLIC),
							},
						},
						testutil.MockResponse{
							Description: "create bucket",
							ToJsonBody: objectstorage.Bucket{
								Name:   utils.Ptr(name),
								Region: utils.Ptr(region),
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating bucket.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v2/project/%s/regions/%s/bucket/%s", projectId, region, name)
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
				ExpectError:  regexp.MustCompile("Error reading bucket*"),
			},
		},
	})
}
