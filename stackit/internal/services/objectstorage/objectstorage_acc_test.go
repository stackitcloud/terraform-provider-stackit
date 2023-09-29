package objectstorage_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Bucket resource data
var bucketResource = map[string]string{
	"project_id":  testutil.ProjectId,
	"bucket_name": fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(20, acctest.CharSetAlpha)),
}

func resourceConfig() string {
	return fmt.Sprintf(`
				%s

				resource "stackit_objectstorage_bucket" "bucket" {
					project_id = "%s"
					bucket_name    = "%s"
				}
				`,
		testutil.ObjectStorageProviderConfig(),
		bucketResource["project_id"],
		bucketResource["bucket_name"],
	)
}

func TestAccObjectStorageResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_objectstorage_bucket.bucket", "project_id", bucketResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_objectstorage_bucket.bucket", "bucket_name", bucketResource["bucket_name"]),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_bucket.bucket", "url_path_style"),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_bucket.bucket", "url_virtual_hosted_style"),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_objectstorage_bucket" "bucket" {
						project_id  = stackit_objectstorage_bucket.bucket.project_id
						bucket_name = stackit_objectstorage_bucket.bucket.bucket_name
					}`,
					resourceConfig(),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_objectstorage_bucket.bucket", "project_id", bucketResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_bucket.bucket", "bucket_name",
						"data.stackit_objectstorage_bucket.bucket", "bucket_name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_bucket.bucket", "url_path_style",
						"data.stackit_objectstorage_bucket.bucket", "url_path_style",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_bucket.bucket", "url_virtual_hosted_style",
						"data.stackit_objectstorage_bucket.bucket", "url_virtual_hosted_style",
					),
				),
			},
			// Import
			{
				ResourceName: "stackit_objectstorage_bucket.bucket",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_objectstorage_bucket.bucket"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_objectstorage_bucket.bucket")
					}
					bucketName, ok := r.Primary.Attributes["bucket_name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute bucket_name")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, bucketName), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Deletion is done by the framework implicitly
		},
	})
}
