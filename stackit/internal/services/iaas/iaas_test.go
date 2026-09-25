package iaas

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestImageResource_ConfigValidators_ExactlyOneOf(t *testing.T) {
	projectId := uuid.NewString()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// STEP 1: Both image_file.local and image_file.download provided
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + fmt.Sprintf(`
resource "stackit_image" "test" {
  project_id  = "%s"
  name        = "test-image"
  disk_format = "qcow2"
  image_file = {
    local = {
      file_path = "/tmp/test.qcow2"
    }
    download = {
      url = "https://example.com/test.qcow2"
    }
  }
}
`, projectId),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
			// STEP 2: Invalid Step: Neither local_file_path nor image_file provided
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + fmt.Sprintf(`
resource "stackit_image" "test" {
  project_id  = "%s"
  name        = "test-image"
  disk_format = "qcow2"
}
`, projectId),
				ExpectError: regexp.MustCompile("Missing Attribute Configuration"),
			},
			// STEP 3: Invalid Step: Deprecated local_file_path AND new image_file.download provided
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + fmt.Sprintf(`
resource "stackit_image" "test" {
  project_id      = "%s"
  name            = "test-image"
  disk_format     = "qcow2"
  local_file_path = "/tmp/test.qcow2"
  image_file = {
    download = {
      url = "https://example.com/test.qcow2"
    }
  }
}
`, projectId),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}

func TestImageResource_DownloadAndCreateSuccess(t *testing.T) {
	t.Parallel()

	projectId := uuid.NewString()
	imageId := uuid.NewString()
	const region = "eu01"
	mockImageBytes := []byte("dummy-qcow2-binary-header-data")

	s := testutil.NewMockServer(t)
	defer s.Server.Close()

	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region        = "%s"
	iaas_custom_endpoint  = "%s"
	service_account_token = "mock-token"
}

resource "stackit_image" "test" {
  project_id  = "%s"
  name        = "downloaded-image"
  disk_format = "qcow2"
  image_file  = {
    download = {
      url = "%s/test-image.qcow2"
    }
  }
}
`, region, s.Server.URL, projectId, s.Server.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						// 1. Download file
						testutil.MockResponse{
							Description: "serve image file download",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								if req.URL.Path != "/test-image.qcow2" {
									t.Errorf("expected request path /test-image.qcow2, got %s", req.URL.Path)
								}
								w.Header().Set("Content-Type", "application/octet-stream")
								w.WriteHeader(http.StatusOK)
								_, _ = w.Write(mockImageBytes)
							},
						},
						// 2. Create Image POST
						testutil.MockResponse{
							Description: "create image API call",
							ToJsonBody: iaas.ImageCreateResponse{
								Id:        imageId,
								UploadUrl: s.Server.URL + "/upload",
							},
						},
						// 3. GetImage initial GET
						testutil.MockResponse{
							Description: "get image initial details",
							ToJsonBody: iaas.Image{
								Id:         &imageId,
								Name:       "downloaded-image",
								DiskFormat: "qcow2",
								Status:     iaas.PtrString(wait.CreateSuccess),
							},
						},
						// 4. Binary upload PUT
						testutil.MockResponse{
							Description: "upload binary image file (PUT)",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								if req.Method != http.MethodPut {
									t.Errorf("expected PUT method for upload, got %s", req.Method)
								}
								w.WriteHeader(http.StatusOK)
							},
						},
						// 5. Waiter poll GET
						testutil.MockResponse{
							Description: "waiter poll success",
							ToJsonBody: iaas.Image{
								Id:         &imageId,
								Name:       "downloaded-image",
								DiskFormat: "qcow2",
								Status:     iaas.PtrString(wait.ImageAvailableStatus),
							},
						},
						// 6. Read state check GET
						testutil.MockResponse{
							Description: "read state before destroy",
							ToJsonBody: iaas.Image{
								Id:         &imageId,
								Name:       "downloaded-image",
								DiskFormat: "qcow2",
								Status:     iaas.PtrString(wait.ImageAvailableStatus),
							},
						},
						// 7. Resource destroy DELETE
						testutil.MockResponse{
							Description: "delete image API call",
							StatusCode:  http.StatusNoContent,
						},
						// 8. Delete waiter GET (404)
						testutil.MockResponse{
							Description: "delete waiter check",
							StatusCode:  http.StatusNotFound,
						},
					)
				},
				Config: tfConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_image.test", "project_id", projectId),
					resource.TestCheckResourceAttr("stackit_image.test", "image_id", imageId),
					resource.TestCheckResourceAttr("stackit_image.test", "name", "downloaded-image"),
					resource.TestCheckResourceAttr("stackit_image.test", "disk_format", "qcow2"),
					resource.TestCheckResourceAttr("stackit_image.test", "image_file.download.url", fmt.Sprintf("%s/test-image.qcow2", s.Server.URL)),
				),
			},
		},
	})
}

func TestImageResource_DownloadFailure404(t *testing.T) {
	t.Parallel()

	projectId := uuid.NewString()
	const region = "eu01"

	s := testutil.NewMockServer(t)
	defer s.Server.Close()

	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region        = "%s"
	iaas_custom_endpoint  = "%s"
	service_account_token = "mock-token"
}

resource "stackit_image" "test" {
  project_id  = "%s"
  name        = "downloaded-image"
  disk_format = "qcow2"
  image_file  = {
    download = {
      url = "%s/test-image.qcow2"
    }
  }
}
`, region, s.Server.URL, projectId, s.Server.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "image download URL not found",
							StatusCode:  http.StatusNotFound,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile(".*failed to download image.*|.*404.*|.*Error downloading image.*"),
			},
		},
	})
}

func TestImageResource_APICreationFailure(t *testing.T) {
	t.Parallel()

	projectId := uuid.NewString()
	const region = "eu01"
	mockImageBytes := []byte("dummy-qcow2-binary-header-data")

	s := testutil.NewMockServer(t)
	defer s.Server.Close()

	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region        = "%s"
	iaas_custom_endpoint  = "%s"
	service_account_token = "mock-token"
}

resource "stackit_image" "test" {
  project_id  = "%s"
  name        = "downloaded-image"
  disk_format = "qcow2"
  image_file  = {
    download = {
      url = "%s/test-image.qcow2"
    }
  }
}
`, region, s.Server.URL, projectId, s.Server.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						// 1. Download succeeds
						testutil.MockResponse{
							Description: "serve image file download",
							Handler: func(w http.ResponseWriter, _ *http.Request) {
								w.Header().Set("Content-Type", "application/octet-stream")
								w.WriteHeader(http.StatusOK)
								_, _ = w.Write(mockImageBytes)
							},
						},
						// 2. API Creation fails
						testutil.MockResponse{
							Description: "create image API error",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating image.*"),
			},
		},
	})
}
