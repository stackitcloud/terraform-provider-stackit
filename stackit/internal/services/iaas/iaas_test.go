package iaas

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

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
