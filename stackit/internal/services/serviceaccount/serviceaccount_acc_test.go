package serviceaccount

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Service Account resource data
var serviceAccountResource = map[string]string{
	"project_id": testutil.ProjectId,
	"name01":     "sa-test-01",
	"name02":     "sa-test-02",
}

func inputServiceAccountResourceConfig(name string) string {
	return fmt.Sprintf(`
				%s
			
				resource "stackit_service_account" "sa" {
					project_id = "%s"
					name = "%s"
				}

				resource "stackit_service_account_access_token" "token" {
					project_id = stackit_service_account.sa.project_id
  					service_account_email = stackit_service_account.sa.email
				}
				`,
		testutil.ServiceAccountProviderConfig(),
		serviceAccountResource["project_id"],
		name,
	)
}

func inputServiceAccountDataSourceConfig() string {
	return fmt.Sprintf(`
					%s

					data "stackit_service_account" "sa" {
						project_id  = stackit_service_account.sa.project_id
						email = stackit_service_account.sa.email
					}
					`,
		inputServiceAccountResourceConfig(serviceAccountResource["name01"]),
	)
}

func TestServiceAccount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				Config: inputServiceAccountResourceConfig(serviceAccountResource["name01"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account.sa", "project_id", serviceAccountResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_service_account.sa", "name", serviceAccountResource["name01"]),
					resource.TestCheckResourceAttrSet("stackit_service_account.sa", "email"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "token"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "valid_until"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "service_account_email"),
					resource.TestCheckResourceAttrPair("stackit_service_account.sa", "email", "stackit_service_account_access_token.token", "service_account_email"),
				),
			},
			// Update
			{
				Config: inputServiceAccountResourceConfig(serviceAccountResource["name02"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account.sa", "project_id", serviceAccountResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_service_account.sa", "name", serviceAccountResource["name02"]),
					resource.TestCheckResourceAttrSet("stackit_service_account.sa", "email"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "token"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "valid_until"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "service_account_email"),
					resource.TestCheckResourceAttrPair("stackit_service_account.sa", "email", "stackit_service_account_access_token.token", "service_account_email"),
				),
			},
			// Data source
			{
				Config: inputServiceAccountDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_service_account.sa", "project_id", serviceAccountResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_service_account.sa", "project_id",
						"data.stackit_service_account.sa", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_service_account.sa", "name",
						"data.stackit_service_account.sa", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_service_account.sa", "email",
						"data.stackit_service_account.sa", "email",
					),
				),
			},
			// Import
			{
				ResourceName: "stackit_service_account.sa",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_service_account.sa"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_service_account.sa")
					}
					email := strings.Split(r.Primary.ID, ",")[1]
					return fmt.Sprintf("%s,%s", testutil.ProjectId, email), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckServiceAccountDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *serviceaccount.APIClient
	var err error

	if testutil.ServiceAccountCustomEndpoint == "" {
		client, err = serviceaccount.NewAPIClient()
	} else {
		client, err = serviceaccount.NewAPIClient(
			config.WithEndpoint(testutil.ServiceAccountCustomEndpoint),
		)
	}

	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var instancesToDestroy []string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_service_account" {
			continue
		}
		serviceAccountEmail := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, serviceAccountEmail)
	}

	instancesResp, err := client.ListServiceAccounts(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting service accounts: %w", err)
	}

	serviceAccounts := *instancesResp.Items
	for i := range serviceAccounts {
		if serviceAccounts[i].Email == nil {
			continue
		}
		if utils.Contains(instancesToDestroy, *serviceAccounts[i].Email) {
			err := client.DeleteServiceAccount(ctx, testutil.ProjectId, *serviceAccounts[i].Email).Execute()
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: %w", *serviceAccounts[i].Email, err)
			}
		}
	}
	return nil
}
