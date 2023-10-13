package objectstorage_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Bucket resource data
var bucketResource = map[string]string{
	"project_id":  testutil.ProjectId,
	"bucket_name": fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(20, acctest.CharSetAlpha)),
}

// Credentials group resource data
var credentialsGroupResource = map[string]string{
	"project_id": testutil.ProjectId,
	"name":       fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(20, acctest.CharSetAlpha)),
}

// Credential resource data
var credentialResource = map[string]string{
	"expiration_timestamp": "2027-01-02T03:04:05Z",
}

func resourceConfig() string {
	return fmt.Sprintf(`
				%s

				resource "stackit_objectstorage_bucket" "bucket" {
					project_id = "%s"
					bucket_name    = "%s"
				}

				resource "stackit_objectstorage_credentials_group" "credentials_group" {
					project_id = "%s"
					name    = "%s"
				}

				resource "stackit_objectstorage_credential" "credential" {
					project_id = stackit_objectstorage_credentials_group.credentials_group.project_id
					credentials_group_id = stackit_objectstorage_credentials_group.credentials_group.credentials_group_id
					expiration_timestamp    = "%s"
				}
				`,
		testutil.ObjectStorageProviderConfig(),
		bucketResource["project_id"],
		bucketResource["bucket_name"],
		credentialsGroupResource["project_id"],
		credentialsGroupResource["name"],
		credentialResource["expiration_timestamp"],
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
					// Bucket data
					resource.TestCheckResourceAttr("stackit_objectstorage_bucket.bucket", "project_id", bucketResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_objectstorage_bucket.bucket", "bucket_name", bucketResource["bucket_name"]),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_bucket.bucket", "url_path_style"),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_bucket.bucket", "url_virtual_hosted_style"),

					// Credentials group data
					resource.TestCheckResourceAttr("stackit_objectstorage_credentials_group.credentials_group", "project_id", credentialsGroupResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_objectstorage_credentials_group.credentials_group", "name", credentialsGroupResource["name"]),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_credentials_group.credentials_group", "credentials_group_id"),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_credentials_group.credentials_group", "urn"),

					// Credential data
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credential.credential", "project_id",
						"stackit_objectstorage_credentials_group.credentials_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credential.credential", "credentials_group_id",
						"stackit_objectstorage_credentials_group.credentials_group", "credentials_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_credential.credential", "credential_id"),
					resource.TestCheckResourceAttr("stackit_objectstorage_credential.credential", "expiration_timestamp", credentialResource["expiration_timestamp"]),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_credential.credential", "name"),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_credential.credential", "access_key"),
					resource.TestCheckResourceAttrSet("stackit_objectstorage_credential.credential", "secret_access_key"),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_objectstorage_bucket" "bucket" {
						project_id  = stackit_objectstorage_bucket.bucket.project_id
						bucket_name = stackit_objectstorage_bucket.bucket.bucket_name
					}
					
					data "stackit_objectstorage_credentials_group" "credentials_group" {
						project_id  = stackit_objectstorage_credentials_group.credentials_group.project_id
						credentials_group_id = stackit_objectstorage_credentials_group.credentials_group.credentials_group_id
					}
					
					data "stackit_objectstorage_credential" "credential" {
						project_id  = stackit_objectstorage_credential.credential.project_id
						credentials_group_id = stackit_objectstorage_credential.credential.credentials_group_id
						credential_id  = stackit_objectstorage_credential.credential.credential_id
					}`,
					resourceConfig(),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Bucket data
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

					// Credentials group data
					resource.TestCheckResourceAttr("data.stackit_objectstorage_credentials_group.credentials_group", "project_id", credentialsGroupResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credentials_group.credentials_group", "credentials_group_id",
						"data.stackit_objectstorage_credentials_group.credentials_group", "credentials_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credentials_group.credentials_group", "name",
						"data.stackit_objectstorage_credentials_group.credentials_group", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credentials_group.credentials_group", "urn",
						"data.stackit_objectstorage_credentials_group.credentials_group", "urn",
					),

					// Credential data
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credential.credential", "project_id",
						"data.stackit_objectstorage_credential.credential", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credential.credential", "credentials_group_id",
						"data.stackit_objectstorage_credential.credential", "credentials_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credential.credential", "credential_id",
						"data.stackit_objectstorage_credential.credential", "credential_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credential.credential", "name",
						"data.stackit_objectstorage_credential.credential", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_objectstorage_credential.credential", "expiration_timestamp",
						"data.stackit_objectstorage_credential.credential", "expiration_timestamp",
					),
				),
			},
			// Import
			{
				ResourceName: "stackit_objectstorage_credentials_group.credentials_group",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_objectstorage_credentials_group.credentials_group"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_objectstorage_credentials_group.credentials_group")
					}
					credentialsGroupId, ok := r.Primary.Attributes["credentials_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute credentials_group_id")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, credentialsGroupId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: "stackit_objectstorage_credential.credential",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_objectstorage_credential.credential"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_objectstorage_credential.credential")
					}
					credentialsGroupId, ok := r.Primary.Attributes["credentials_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute credentials_group_id")
					}
					credentialId, ok := r.Primary.Attributes["credential_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute credential_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, credentialsGroupId, credentialId), nil
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
		client, err = objectstorage.NewAPIClient(
			config.WithRegion("eu01"),
		)
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
		if utils.Contains(bucketsToDestroy, bucketName) {
			_, err := client.DeleteBucketExecute(ctx, testutil.ProjectId, bucketName)
			if err != nil {
				return fmt.Errorf("destroying bucket %s during CheckDestroy: %w", bucketName, err)
			}
			_, err = wait.DeleteBucketWaitHandler(ctx, client, testutil.ProjectId, bucketName).SetTimeout(1 * time.Minute).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: waiting for deletion %w", bucketName, err)
			}
		}
	}

	credentialsGroupsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_objectstorage_credentials_group" {
			continue
		}
		// credentials group terraform ID: "[project_id],[credentials_group_id]"
		credentialsGroupId := strings.Split(rs.Primary.ID, core.Separator)[1]
		credentialsGroupsToDestroy = append(credentialsGroupsToDestroy, credentialsGroupId)
	}

	credentialsGroupsResp, err := client.GetCredentialsGroups(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting bucketsResp: %w", err)
	}

	groups := *credentialsGroupsResp.CredentialsGroups
	for _, group := range groups {
		if group.CredentialsGroupId == nil {
			continue
		}
		groupId := *group.CredentialsGroupId
		if utils.Contains(credentialsGroupsToDestroy, groupId) {
			_, err := client.DeleteCredentialsGroupExecute(ctx, testutil.ProjectId, groupId)
			if err != nil {
				return fmt.Errorf("destroying credentials group %s during CheckDestroy: %w", groupId, err)
			}
		}
	}
	return nil
}
