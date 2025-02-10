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
	stackit_sdk_config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testfiles/prerequisites.tf
var prerequisites string

//go:embed testfiles/double-definition.tf
var double_definition string

//go:embed testfiles/project-owner.tf
var project_owner string

//go:embed testfiles/invalid-role.tf
var invalid_role string

var testConfigVars = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"test_service_account": config.StringVariable(testutil.TestProjectServiceAccountEmail),
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
				// Expect failure on creating an already existing role_assignment
				// Would be bad, since two resources could be created and deletion of one would lead to state drift for the second TF resource
				ConfigVariables: testConfigVars,
				Config:          testutil.AuthorizationProviderConfig() + prerequisites + double_definition,
				ExpectError:     regexp.MustCompile(".+"),
			},
			{
				// The Service Account inherits owner permissions for the project from the organization. Check if you can still assign owner permissions on the project explicitly
				ConfigVariables: testConfigVars,
				Config:          testutil.AuthorizationProviderConfig() + prerequisites + project_owner,
			},
			{
				// Assign a non-existent role. Expect failure
				ConfigVariables: testConfigVars,
				Config:          testutil.AuthorizationProviderConfig() + prerequisites + invalid_role,
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
			stackit_sdk_config.WithRegion("eu01"),
		)
	} else {
		client, err = authorization.NewAPIClient(
			stackit_sdk_config.WithEndpoint(testutil.AuthorizationCustomEndpoint),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return client, nil
}
