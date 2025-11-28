package authorization_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"testing"

	_ "embed"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testfiles/prerequisites.tf
var prerequisites string

//go:embed testfiles/double-definition.tf
var doubleDefinition string

//go:embed testfiles/project-owner.tf
var projectOwner string

//go:embed testfiles/invalid-role.tf
var invalidRole string

//go:embed testfiles/organization-role.tf
var organizationRole string

//go:embed testfiles/custom-role.tf
var customRole string

var testConfigVars = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"test_service_account": config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"organization_id":      config.StringVariable(testutil.OrganizationId),
}

var testConfigVarsCustomRole = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"test_service_account": config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"organization_id":      config.StringVariable(testutil.OrganizationId),
	"role_name":            config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"role_description":     config.StringVariable("Some description"),
	"role_permissions_0":   config.StringVariable("iam.role.list"),
}

var testConfigVarsCustomRoleUpdated = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"test_service_account": config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"organization_id":      config.StringVariable(testutil.OrganizationId),
	"role_name":            config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"role_description":     config.StringVariable("Updated description"),
	"role_permissions_0":   config.StringVariable("iam.role.edit"),
}

func TestAccProjectRoleAssignmentResource(t *testing.T) {
	t.Log(testutil.AuthorizationProviderConfig())
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigVariables: testConfigVars,
				Config:          testutil.AuthorizationProviderConfig() + prerequisites,
				Check: func(_ *terraform.State) error {
					client, err := authApiClient()
					if err != nil {
						return err
					}

					members, err := client.ListMembers(context.Background(), "project", testutil.ProjectId).Execute()
					if err != nil {
						return err
					}

					if !slices.ContainsFunc(*members.Members, func(m authorization.Member) bool {
						return *m.Role == "reader" && *m.Subject == testutil.TestProjectServiceAccountEmail
					}) {
						t.Log(members.Members)
						return errors.New("Membership not found")
					}
					return nil
				},
			},
			{
				// Assign a resource to an organization
				ConfigVariables: testConfigVars,
				Config:          testutil.AuthorizationProviderConfig() + prerequisites + organizationRole,
			},
			{
				// The Service Account inherits owner permissions for the project from the organization. Check if you can still assign owner permissions on the project explicitly
				ConfigVariables: testConfigVars,
				Config:          testutil.AuthorizationProviderConfig() + prerequisites + organizationRole + projectOwner,
			},
			{
				// Expect failure on creating an already existing role_assignment
				// Would be bad, since two resources could be created and deletion of one would lead to state drift for the second TF resource
				ConfigVariables: testConfigVars,
				Config:          testutil.AuthorizationProviderConfig() + prerequisites + doubleDefinition,
				ExpectError:     regexp.MustCompile(".+"),
			},
			{
				// Assign a non-existent role. Expect failure
				ConfigVariables: testConfigVars,
				Config:          testutil.AuthorizationProviderConfig() + prerequisites + invalidRole,
				ExpectError:     regexp.MustCompile(".+"),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigVariables: testConfigVarsCustomRole,
				Config:          testutil.AuthorizationProviderConfig() + customRole,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.custom-role", "resource_id", testutil.ConvertConfigVariable(testConfigVarsCustomRole["project_id"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.custom-role", "name", testutil.ConvertConfigVariable(testConfigVarsCustomRole["role_name"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.custom-role", "description", testutil.ConvertConfigVariable(testConfigVarsCustomRole["role_description"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.custom-role", "permissions.#", "1"),
					resource.TestCheckTypeSetElemAttr("stackit_authorization_project_custom_role.custom-role", "permissions.*", testutil.ConvertConfigVariable(testConfigVarsCustomRole["role_permissions_0"])),
					resource.TestCheckResourceAttrSet("stackit_authorization_project_custom_role.custom-role", "role_id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsCustomRole,
				Config: fmt.Sprintf(`
					%s

					data "stackit_authorization_project_custom_role" "custom-role" {
						resource_id  = stackit_authorization_project_custom_role.custom-role.resource_id
						role_id  = stackit_authorization_project_custom_role.custom-role.role_id
					}
					`,
					testutil.AuthorizationProviderConfig()+customRole,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_authorization_project_custom_role.custom-role", "resource_id", testutil.ConvertConfigVariable(testConfigVarsCustomRole["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.custom-role", "resource_id",
						"data.stackit_authorization_project_custom_role.custom-role", "resource_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.custom-role", "role_id",
						"data.stackit_authorization_project_custom_role.custom-role", "role_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.custom-role", "name",
						"data.stackit_authorization_project_custom_role.custom-role", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.custom-role", "description",
						"data.stackit_authorization_project_custom_role.custom-role", "description",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.custom-role", "permissions",
						"data.stackit_authorization_project_custom_role.custom-role", "permissions",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsCustomRole,
				ResourceName:    "stackit_authorization_project_custom_role.custom-role",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_authorization_project_custom_role.custom-role"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_authorization_project_custom_role.custom-role")
					}
					roleId, ok := r.Primary.Attributes["role_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute role_id")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, roleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsCustomRoleUpdated,
				Config:          testutil.AuthorizationProviderConfig() + customRole,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.custom-role", "resource_id", testutil.ConvertConfigVariable(testConfigVarsCustomRoleUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.custom-role", "name", testutil.ConvertConfigVariable(testConfigVarsCustomRoleUpdated["role_name"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.custom-role", "description", testutil.ConvertConfigVariable(testConfigVarsCustomRoleUpdated["role_description"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.custom-role", "permissions.#", "1"),
					resource.TestCheckTypeSetElemAttr("stackit_authorization_project_custom_role.custom-role", "permissions.*", testutil.ConvertConfigVariable(testConfigVarsCustomRoleUpdated["role_permissions_0"])),
					resource.TestCheckResourceAttrSet("stackit_authorization_project_custom_role.custom-role", "role_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func authApiClient() (*authorization.APIClient, error) {
	var client *authorization.APIClient
	var err error
	if testutil.AuthorizationCustomEndpoint == "" || testutil.TokenCustomEndpoint == "" {
		client, err = authorization.NewAPIClient()
	} else {
		client, err = authorization.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.AuthorizationCustomEndpoint),
			stackitSdkConfig.WithTokenEndpoint(testutil.TokenCustomEndpoint),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return client, nil
}
