package serviceaccount

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-service-account.tf
	resourceServiceAccount string

	//go:embed testdata/datasource-service-account.tf
	datasourceServiceAccount string

	//go:embed testdata/datasource-service-account-regex.tf
	datasourceServiceAccountRegex string
)

var testConfigVars = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("satest01"),
}

var testConfigVarsUpdate = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("satest02"),
}

var testConfigVarsRegex = config.Variables{
	"project_id":  config.StringVariable(testutil.ProjectId),
	"name":        config.StringVariable("satest02"),
	"email_regex": config.StringVariable(`^satest02-\w{7,10}@(?:ske\.)?sa\.stackit\.cloud$`),
}

var testConfigVarsRegexNotFound = config.Variables{
	"project_id":  config.StringVariable(testutil.ProjectId),
	"name":        config.StringVariable("satest02"),
	"email_regex": config.StringVariable("not-found"),
}

func TestServiceAccount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVars,
				Config:          testutil.ServiceAccountProviderConfig() + "\n" + resourceServiceAccount,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account.sa", "project_id", testutil.ConvertConfigVariable(testConfigVars["project_id"])),
					resource.TestCheckResourceAttr("stackit_service_account.sa", "name", testutil.ConvertConfigVariable(testConfigVars["name"])),
					resource.TestCheckResourceAttrSet("stackit_service_account.sa", "email"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "token"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "valid_until"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "service_account_email"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "ttl_days"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "json"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "service_account_email"),
					resource.TestCheckResourceAttrPair("stackit_service_account.sa", "email", "stackit_service_account_access_token.token", "service_account_email"),
					resource.TestCheckResourceAttrPair("stackit_service_account.sa", "email", "stackit_service_account_key.key", "service_account_email"),
				),
			},
			// Update
			{
				ConfigVariables: testConfigVarsUpdate,
				Config:          testutil.ServiceAccountProviderConfig() + "\n" + resourceServiceAccount,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account.sa", "project_id", testutil.ConvertConfigVariable(testConfigVarsUpdate["project_id"])),
					resource.TestCheckResourceAttr("stackit_service_account.sa", "name", testutil.ConvertConfigVariable(testConfigVarsUpdate["name"])),
					resource.TestCheckResourceAttrSet("stackit_service_account.sa", "email"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "token"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "valid_until"),
					resource.TestCheckResourceAttrSet("stackit_service_account_access_token.token", "service_account_email"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "ttl_days"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "json"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "service_account_email"),
					resource.TestCheckResourceAttrPair("stackit_service_account.sa", "email", "stackit_service_account_access_token.token", "service_account_email"),
					resource.TestCheckResourceAttrPair("stackit_service_account.sa", "email", "stackit_service_account_key.key", "service_account_email"),
				),
			},
			// Data source (Using exact email)
			{
				ConfigVariables: testConfigVarsUpdate,
				Config:          testutil.ServiceAccountProviderConfig() + "\n" + resourceServiceAccount + "\n" + datasourceServiceAccount,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_service_account.sa", "project_id", testutil.ConvertConfigVariable(testConfigVarsUpdate["project_id"])),
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
			// Data source (Using email_regex)
			{
				ConfigVariables: testConfigVarsRegex,
				Config:          testutil.ServiceAccountProviderConfig() + "\n" + resourceServiceAccount + "\n" + datasourceServiceAccountRegex,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_service_account.sa_regex", "project_id", testutil.ConvertConfigVariable(testConfigVarsRegex["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_service_account.sa", "project_id",
						"data.stackit_service_account.sa_regex", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_service_account.sa", "name",
						"data.stackit_service_account.sa_regex", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_service_account.sa", "email",
						"data.stackit_service_account.sa_regex", "email",
					),
				),
			},
			// Data source (Using email_regex - Not Found Expectation)
			{
				ConfigVariables: testConfigVarsRegexNotFound,
				Config:          testutil.ServiceAccountProviderConfig() + "\n" + resourceServiceAccount + "\n" + datasourceServiceAccountRegex,
				ExpectError:     regexp.MustCompile(`Service Account not found`),
			},
			// Import
			{
				ConfigVariables: testConfigVarsUpdate,
				Config:          testutil.ServiceAccountProviderConfig() + "\n" + resourceServiceAccount,
				ResourceName:    "stackit_service_account.sa",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_service_account.sa"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_service_account.sa")
					}
					email, ok := r.Primary.Attributes["email"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute email")
					}
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
			stackitSdkConfig.WithEndpoint(testutil.ServiceAccountCustomEndpoint),
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
