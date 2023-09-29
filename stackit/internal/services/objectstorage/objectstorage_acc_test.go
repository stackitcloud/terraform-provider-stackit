package objectstorage_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
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
		CheckDestroy:             testAccCheckObjectStorageDestroy,
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

func testAccCheckObjectStorageDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *objectstorage.APIClient
	var err error
	if testutil.ObjectStorageCustomEndpoint == "" {
		client, err = objectstorage.NewAPIClient()
	} else {
		client, err = objectstorage.NewAPIClient(
			config.WithEndpoint(testutil.ObjectStorageCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	bucketsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_objectstorage_bucket" {
			continue
		}
		// bucket terraform ID: "[project_id],[bucket_name]"
		bucketName := strings.Split(rs.Primary.ID, core.Separator)[1]
		bucketsToDestroy = append(bucketsToDestroy, bucketName)
	}

	bucketsResp, err := client.GetBuckets(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting bucketsResp: %w", err)
	}

	buckets := *bucketsResp.Buckets
	for _, bucket := range buckets {
		if bucket.Name == nil {
			continue
		}
		bucketName := *bucket.Name
		if utils.Contains(bucketsToDestroy, *bucket.Name) {
			_, err := client.DeleteBucketExecute(ctx, testutil.ProjectId, bucketName)
			if err != nil {
				return fmt.Errorf("destroying bucket %s during CheckDestroy: %w", bucketName, err)
			}
			_, err = objectstorage.DeleteBucketWaitHandler(ctx, client, testutil.ProjectId, bucketName).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: waiting for deletion %w", bucketName, err)
			}
		}
	}
	return nil
}
