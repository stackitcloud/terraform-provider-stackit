package serviceaccount

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-service-account.tf
	resourceServiceAccount string

	//go:embed testdata/resource-service-account-federated-identity-provider-without-assertions.tf
	resourceServiceAccountFederatedIdentityProviderWithoutAssertions string

	//go:embed testdata/resource-service-account-federated-identity-provider-without-aud.tf
	resourceServiceAccountFederatedIdentityProviderWithoutAud string

	//go:embed testdata/resource-service-account-federated-identity-provider.tf
	resourceServiceAccountFederatedIdentityProvider string

	//go:embed testdata/datasource-service-account.tf
	datasourceServiceAccount string

	//go:embed testdata/datasource-service-accounts.tf
	datasourceServiceAccounts string

	//go:embed testdata/datasource-service-accounts-regex.tf
	datasourceServiceAccountsRegex string

	//go:embed testdata/datasource-service-accounts-suffix.tf
	datasourceServiceAccountsSuffix string

	//go:embed testdata/datasource-service-account-exact-not-found.tf
	datasourceServiceAccountExactNotFound string
)

var testConfigVars = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("satest01"),
}

var testConfigVarsUpdate = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("satest02"),
}

var testConfigVarsPluralRegex = config.Variables{
	"project_id":  config.StringVariable(testutil.ProjectId),
	"name":        config.StringVariable("satest02"),
	"email_regex": config.StringVariable(`^satest02-.*@(?:ske\.)?sa\.stackit\.cloud$`),
}

var testConfigVarsPluralSuffix = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"name":         config.StringVariable("satest02"),
	"email_suffix": config.StringVariable(`@sa.stackit.cloud`),
}

var testConfigVarsExactNotFound = config.Variables{
	"project_id":      config.StringVariable(testutil.ProjectId),
	"name":            config.StringVariable("satest02"),
	"not_found_email": config.StringVariable("does-not-exist-123@sa.stackit.cloud"),
}

var testConfigVarsFederatedIdentityProviderWithoutAssertions = config.Variables{
	"project_id":    config.StringVariable(testutil.ProjectId),
	"provider_name": config.StringVariable("provider-no-assertions"),
}

var testConfigVarsFederatedIdentityProviderWithoutAud = config.Variables{
	"project_id":    config.StringVariable(testutil.ProjectId),
	"provider_name": config.StringVariable("provider-no-aud"),
}

var testConfigVarsFederatedIdentityProviderCreate = config.Variables{
	"project_id":    config.StringVariable(testutil.ProjectId),
	"provider_name": config.StringVariable("provider1"),
	"sub":           config.StringVariable("user@mail.com"),
}

var testConfigVarsFederatedIdentityProviderUpdate = config.Variables{
	"project_id":    config.StringVariable(testutil.ProjectId),
	"provider_name": config.StringVariable("provider1-updated"),
	"sub":           config.StringVariable("other@mail.com"),
}

func TestServiceAccount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVars,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccount,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account.sa", "project_id", testutil.ConvertConfigVariable(testConfigVars["project_id"])),
					resource.TestCheckResourceAttr("stackit_service_account.sa", "name", testutil.ConvertConfigVariable(testConfigVars["name"])),
					resource.TestCheckResourceAttrSet("stackit_service_account.sa", "email"),
					resource.TestCheckResourceAttrSet("stackit_service_account.sa", "service_account_id"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "ttl_days"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "json"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "service_account_email"),
					resource.TestCheckResourceAttrPair("stackit_service_account.sa", "email", "stackit_service_account_key.key", "service_account_email"),
				),
			},
			// Update
			{
				ConfigVariables: testConfigVarsUpdate,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccount,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account.sa", "project_id", testutil.ConvertConfigVariable(testConfigVarsUpdate["project_id"])),
					resource.TestCheckResourceAttr("stackit_service_account.sa", "name", testutil.ConvertConfigVariable(testConfigVarsUpdate["name"])),
					resource.TestCheckResourceAttrSet("stackit_service_account.sa", "email"),
					resource.TestCheckResourceAttrSet("stackit_service_account.sa", "service_account_id"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "ttl_days"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "json"),
					resource.TestCheckResourceAttrSet("stackit_service_account_key.key", "service_account_email"),
					resource.TestCheckResourceAttrPair("stackit_service_account.sa", "email", "stackit_service_account_key.key", "service_account_email"),
				),
			},
			// Data source (Using exact email)
			{
				ConfigVariables: testConfigVarsUpdate,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccount + "\n" + datasourceServiceAccount,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_service_account.sa", "project_id", testutil.ConvertConfigVariable(testConfigVarsUpdate["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_service_account.sa", "service_account_id"),
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
					resource.TestCheckResourceAttrPair(
						"stackit_service_account.sa", "service_account_id",
						"data.stackit_service_account.sa", "service_account_id",
					),
				),
			},
			// Data source (Singular Exact Email - Not Found Expectation)
			{
				ConfigVariables: testConfigVarsExactNotFound,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccount + "\n" + datasourceServiceAccountExactNotFound,
				ExpectError:     regexp.MustCompile(`Service account not found`),
			},
			// Data source (Plural / List of Service Accounts - No filter)
			{
				ConfigVariables: testConfigVarsUpdate,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccount + "\n" + datasourceServiceAccounts,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_service_accounts.list", "project_id", testutil.ConvertConfigVariable(testConfigVarsUpdate["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_service_accounts.list", "items.0.email"),
					resource.TestCheckResourceAttrSet("data.stackit_service_accounts.list", "items.0.name"),
					resource.TestCheckResourceAttrSet("data.stackit_service_accounts.list", "items.0.service_account_id"),
				),
			},
			// Data source (Plural - Filtered by Regex)
			{
				ConfigVariables: testConfigVarsPluralRegex,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccount + "\n" + datasourceServiceAccountsRegex,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_service_accounts.list_regex", "project_id", testutil.ConvertConfigVariable(testConfigVarsPluralRegex["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_service_accounts.list_regex", "items.0.email"),
					resource.TestCheckResourceAttrSet("data.stackit_service_accounts.list_regex", "items.0.service_account_id"),
				),
			},
			// Data source (Plural - Filtered by Suffix)
			{
				ConfigVariables: testConfigVarsPluralSuffix,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccount + "\n" + datasourceServiceAccountsSuffix,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_service_accounts.list_suffix", "project_id", testutil.ConvertConfigVariable(testConfigVarsPluralSuffix["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_service_accounts.list_suffix", "items.0.email"),
					resource.TestCheckResourceAttrSet("data.stackit_service_accounts.list_suffix", "items.0.service_account_id"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsUpdate,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccount,
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
			// Federated identity provider - Attempt without assertions (should fail)
			{
				ConfigVariables: testConfigVarsFederatedIdentityProviderWithoutAssertions,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccountFederatedIdentityProviderWithoutAssertions,
				ExpectError:     regexp.MustCompile(`The argument "assertions" is required`),
			},
			// Federated identity provider - Attempt without aud assertion (should fail)
			{
				ConfigVariables: testConfigVarsFederatedIdentityProviderWithoutAud,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccountFederatedIdentityProviderWithoutAud,
				ExpectError:     regexp.MustCompile(`Missing Required Assertion`),
			},
			// Federated identity provider - Creation with assertions
			{
				ConfigVariables: testConfigVarsFederatedIdentityProviderCreate,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccountFederatedIdentityProvider,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "name", "provider1"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "issuer", "https://accounts.stackit.cloud"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.#", "3"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.0.item", "iss"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.0.operator", "equals"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.0.value", "https://accounts.stackit.cloud"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.1.item", "sub"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.1.operator", "equals"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.1.value", "user@mail.com"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.2.item", "aud"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.2.operator", "equals"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.2.value", "sts.accounts.stackit.cloud"),
					resource.TestCheckResourceAttrSet("stackit_service_account_federated_identity_provider.provider", "id"),
					resource.TestCheckResourceAttrSet("stackit_service_account_federated_identity_provider.provider", "service_account_email"),
				),
			},
			// Federated identity provider - Update with assertions
			{
				ConfigVariables: testConfigVarsFederatedIdentityProviderUpdate,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceServiceAccountFederatedIdentityProvider,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "name", "provider1-updated"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "issuer", "https://accounts.stackit.cloud"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.#", "3"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.0.item", "iss"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.0.operator", "equals"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.0.value", "https://accounts.stackit.cloud"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.1.item", "sub"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.1.operator", "equals"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.1.value", "other@mail.com"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.2.item", "aud"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.2.operator", "equals"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.2.value", "sts.accounts.stackit.cloud"),
					resource.TestCheckResourceAttrSet("stackit_service_account_federated_identity_provider.provider", "id"),
					resource.TestCheckResourceAttrSet("stackit_service_account_federated_identity_provider.provider", "service_account_email"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckServiceAccountDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := serviceaccount.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.ServiceAccountCustomEndpoint, false)...)
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

	instancesResp, err := client.DefaultAPI.ListServiceAccounts(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting service accounts: %w", err)
	}

	serviceAccounts := instancesResp.Items
	for i := range serviceAccounts {
		if slices.Contains(instancesToDestroy, serviceAccounts[i].Email) {
			err := client.DefaultAPI.DeleteServiceAccount(ctx, testutil.ProjectId, serviceAccounts[i].Email).Execute()
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: %w", serviceAccounts[i].Email, err)
			}
		}
	}
	return nil
}
