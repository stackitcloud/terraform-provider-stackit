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

var testConfigVars = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"test_service_account": config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"organization_id":      config.StringVariable(testutil.OrganizationId),
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

					members, err := client.ListMembers(context.TODO(), "project", testutil.ProjectId).Execute()

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
}

func authApiClient() (*authorization.APIClient, error) {
	var client *authorization.APIClient
	var err error
	if testutil.AuthorizationCustomEndpoint == "" {
		client, err = authorization.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = authorization.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.AuthorizationCustomEndpoint),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return client, nil
}
