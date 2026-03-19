package federated_identity_provider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestAccServiceAccountFederatedIdentityProvider(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckFederatedIdentityProviderDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: testAccFederatedIdentityProviderConfig("provider1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "name", "provider1"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "issuer", "https://example.com"),
					resource.TestCheckResourceAttrSet("stackit_service_account_federated_identity_provider.provider", "id"),
					resource.TestCheckResourceAttrSet("stackit_service_account_federated_identity_provider.provider", "service_account_email"),
				),
			},
			// Update
			{
				Config: testAccFederatedIdentityProviderConfig("provider1-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "name", "provider1-updated"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "issuer", "https://example.com"),
					resource.TestCheckResourceAttrSet("stackit_service_account_federated_identity_provider.provider", "id"),
					resource.TestCheckResourceAttrSet("stackit_service_account_federated_identity_provider.provider", "service_account_email"),
				),
			},
			// Import
			{
				ResourceName: "stackit_service_account_federated_identity_provider.provider",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_service_account_federated_identity_provider.provider"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_service_account_federated_identity_provider.provider")
					}
					id, ok := r.Primary.Attributes["id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute id")
					}
					return id, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccServiceAccountFederatedIdentityProviderWithAssertions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckFederatedIdentityProviderDestroy,
		Steps: []resource.TestStep{
			// Creation with assertions
			{
				Config: testAccFederatedIdentityProviderConfigWithAssertions(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "name", "provider-with-assertions"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "issuer", "https://example.com"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.#", "2"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.0.item", "iss"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.0.operator", "EQUALS"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.0.value", "https://example.com"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.1.item", "sub"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.1.operator", "EQUALS"),
					resource.TestCheckResourceAttr("stackit_service_account_federated_identity_provider.provider", "assertions.1.value", "user@example.com"),
					resource.TestCheckResourceAttrSet("stackit_service_account_federated_identity_provider.provider", "id"),
				),
			},
		},
	})
}

func testAccFederatedIdentityProviderConfig(name string) string {
	return fmt.Sprintf(`
		%s

		resource "stackit_service_account" "sa" {
			project_id = "%s"
			name       = "test-sa"
		}

		resource "stackit_service_account_federated_identity_provider" "provider" {
			project_id            = stackit_service_account.sa.project_id
			service_account_email = stackit_service_account.sa.email
			name                  = "%s"
			issuer                = "https://example.com"
		}
	`, testutil.ServiceAccountProviderConfig(), testutil.ProjectId, name)
}

func testAccFederatedIdentityProviderConfigWithAssertions() string {
	return fmt.Sprintf(`
		%s

		resource "stackit_service_account" "sa" {
			project_id = "%s"
			name       = "test-sa-with-assertions"
		}

		resource "stackit_service_account_federated_identity_provider" "provider" {
			project_id            = stackit_service_account.sa.project_id
			service_account_email = stackit_service_account.sa.email
			name                  = "provider-with-assertions"
			issuer                = "https://example.com"

			assertions = [
				{
					item     = "iss"
					operator = "EQUALS"
					value    = "https://example.com"
				},
				{
					item     = "sub"
					operator = "EQUALS"
					value    = "user@example.com"
				}
			]
		}
	`, testutil.ServiceAccountProviderConfig(), testutil.ProjectId)
}

func testAccCheckFederatedIdentityProviderDestroy(s *terraform.State) error {
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

	var providersToDestroy []string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_service_account_federated_identity_provider" {
			continue
		}

		serviceAccountEmail, ok := rs.Primary.Attributes["service_account_email"]
		if !ok || serviceAccountEmail == "" {
			continue
		}

		providerName, ok := rs.Primary.Attributes["name"]
		if !ok || providerName == "" {
			continue
		}

		key := fmt.Sprintf("%s|%s", serviceAccountEmail, providerName)
		providersToDestroy = append(providersToDestroy, key)
	}

	// Check if any providers still exist
	listResp, err := client.ListServiceAccounts(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting service accounts: %w", err)
	}

	if listResp.Items == nil {
		return nil
	}

	for _, acc := range *listResp.Items {
		if acc.Email == nil {
			continue
		}

		providersResp, err := client.ListFederatedIdentityProviders(ctx, testutil.ProjectId, *acc.Email).Execute()
		if err != nil {
			// Ignore errors, provider might not exist
			continue
		}

		if providersResp.Resources == nil {
			continue
		}

		for _, provider := range *providersResp.Resources {
			if provider.Name == nil {
				continue
			}

			key := fmt.Sprintf("%s|%s", *acc.Email, *provider.Name)
			if utils.Contains(providersToDestroy, key) {
				err := client.DeleteServiceFederatedIdentityProvider(ctx, testutil.ProjectId, *acc.Email, *provider.Name).Execute()
				if err != nil {
					return fmt.Errorf("destroying federated identity provider %s during CheckDestroy: %w", *provider.Name, err)
				}
			}
		}
	}

	return nil
}
