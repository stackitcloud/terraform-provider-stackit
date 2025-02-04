package projectroleassignment_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackit_sdk_config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)


func TestProjectRoleAssignmentResource(t *testing.T)	{
	t.Log(testutil.AuthorizationProviderConfig())
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			resource.TestStep{
				ConfigVariables: config.Variables{
					"project_id": config.StringVariable(testutil.ProjectId),
					"test_service_account": config.StringVariable(testutil.TestProjectServiceAccountEmail),
				},
				Config: testutil.AuthorizationProviderConfig() +
					`
					variable "project_id" {}
					variable "test_service_account" {}

					resource "stackit_resourcemanager_project_role_assignment" "serviceaccount" {
						resource_id = var.project_id
						role = "owner"
						subject = var.test_service_account
					}
					`,
				Check: func(s *terraform.State) error {
					client, err := authApiClient()
					if err != nil	{
						return err
					}

					members, err := client.ListMembers(context.TODO(), "project", testutil.ProjectId).Execute()

					if err != nil 	{
						return err
					}
					
					if !slices.ContainsFunc(*members.Members, func(m authorization.Member) bool { return *m.Role == "owner" && *m.Subject == testutil.TestProjectServiceAccountEmail })	{
						t.Log(members.Members)
						 return errors.New("Membership not found")
					}
					return nil
				},
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