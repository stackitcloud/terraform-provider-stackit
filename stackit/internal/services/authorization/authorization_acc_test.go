package authorization_test

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"testing"

	_ "embed"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-project-role-assignment.tf
	resourceProjectRoleAssignment string

	//go:embed testdata/resource-folder-role-assignment.tf
	resourceFolderRoleAssignment string

	//go:embed testdata/resource-org-role-assignment.tf
	resourceOrgRoleAssignment string
)

var testProjectName = fmt.Sprintf("proj-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var testFolderName = fmt.Sprintf("folder-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))

var testConfigVarsProjectRoleAssignment = config.Variables{
	"name":                config.StringVariable(testProjectName),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.OrganizationId),
	"role":                config.StringVariable("reader"),
	"subject":             config.StringVariable(testutil.TestProjectServiceAccountEmail),
}

var testConfigVarsFolderRoleAssignment = config.Variables{
	"name":                config.StringVariable(testFolderName),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.OrganizationId),
	"role":                config.StringVariable("reader"),
	"subject":             config.StringVariable(testutil.TestProjectServiceAccountEmail),
}

var testConfigVarsOrgRoleAssignment = config.Variables{
	"parent_container_id": config.StringVariable(testutil.OrganizationId),
	"role":                config.StringVariable("iaas.admin"),
	"subject":             config.StringVariable(testutil.TestProjectServiceAccountEmail),
}

func testConfigVarsProjectRoleAssignmentUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsProjectRoleAssignment))
	maps.Copy(tempConfig, testConfigVarsProjectRoleAssignment)

	tempConfig["role"] = config.StringVariable("editor")
	return tempConfig
}

func testConfigVarsFolderRoleAssignmentUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsFolderRoleAssignment))
	maps.Copy(tempConfig, testConfigVarsFolderRoleAssignment)

	tempConfig["role"] = config.StringVariable("editor")
	return tempConfig
}

func testConfigVarsOrgRoleAssignmentUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsOrgRoleAssignment))
	maps.Copy(tempConfig, testConfigVarsOrgRoleAssignment)

	tempConfig["role"] = config.StringVariable("iaas.project.admin")
	return tempConfig
}

func TestAccProjectRoleAssignmentResource(t *testing.T) {
	t.Log("Testing project role assignment resource")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		// deleting project will also delete project role assignments
		CheckDestroy: testAccCheckResourceManagerProjectsDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsProjectRoleAssignment,
				Config:          testutil.AuthorizationProviderConfig() + "\n" + resourceProjectRoleAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "name", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignment["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "owner_email", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignment["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "parent_container_id", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignment["parent_container_id"])),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "project_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "update_time"),

					resource.TestCheckResourceAttrSet("stackit_authorization_project_role_assignment.pra", "resource_id"),
					resource.TestCheckResourceAttrSet("stackit_authorization_project_role_assignment.pra", "id"),
					resource.TestCheckResourceAttr("stackit_authorization_project_role_assignment.pra", "role", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignment["role"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_role_assignment.pra", "subject", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignment["subject"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsProjectRoleAssignment,
				ResourceName:    "stackit_authorization_project_role_assignment.pra",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_authorization_project_role_assignment.pra"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_authorization_project_role_assignment.pra")
					}
					resourceId, ok := r.Primary.Attributes["resource_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute resource_id")
					}
					role, ok := r.Primary.Attributes["role"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute role")
					}
					subject, ok := r.Primary.Attributes["subject"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute subject")
					}

					return fmt.Sprintf("%s,%s,%s", resourceId, role, subject), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsProjectRoleAssignmentUpdated(),
				Config:          testutil.AuthorizationProviderConfig() + "\n" + resourceProjectRoleAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "name", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignmentUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "owner_email", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignmentUpdated()["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "parent_container_id", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignmentUpdated()["parent_container_id"])),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "project_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "update_time"),

					resource.TestCheckResourceAttrSet("stackit_authorization_project_role_assignment.pra", "resource_id"),
					resource.TestCheckResourceAttrSet("stackit_authorization_project_role_assignment.pra", "id"),
					resource.TestCheckResourceAttr("stackit_authorization_project_role_assignment.pra", "role", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignmentUpdated()["role"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_role_assignment.pra", "subject", testutil.ConvertConfigVariable(testConfigVarsProjectRoleAssignmentUpdated()["subject"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccFolderRoleAssignmentResource(t *testing.T) {
	t.Log("Testing folder role assignment resource")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		// deleting folder will also delete project role assignments
		CheckDestroy: testAccCheckResourceManagerFoldersDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsFolderRoleAssignment,
				Config:          testutil.AuthorizationProviderConfig() + "\n" + resourceFolderRoleAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.folder", "name", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignment["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.folder", "owner_email", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignment["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.folder", "parent_container_id", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignment["parent_container_id"])),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "folder_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "update_time"),

					resource.TestCheckResourceAttrSet("stackit_authorization_folder_role_assignment.fra", "resource_id"),
					resource.TestCheckResourceAttrSet("stackit_authorization_folder_role_assignment.fra", "id"),
					resource.TestCheckResourceAttr("stackit_authorization_folder_role_assignment.fra", "role", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignment["role"])),
					resource.TestCheckResourceAttr("stackit_authorization_folder_role_assignment.fra", "subject", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignment["subject"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsProjectRoleAssignment,
				ResourceName:    "stackit_authorization_folder_role_assignment.fra",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_authorization_folder_role_assignment.fra"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_authorization_folder_role_assignment.fra")
					}
					resourceId, ok := r.Primary.Attributes["resource_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute resource_id")
					}
					role, ok := r.Primary.Attributes["role"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute role")
					}
					subject, ok := r.Primary.Attributes["subject"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute subject")
					}

					return fmt.Sprintf("%s,%s,%s", resourceId, role, subject), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsFolderRoleAssignmentUpdated(),
				Config:          testutil.AuthorizationProviderConfig() + "\n" + resourceFolderRoleAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.folder", "name", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignmentUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.folder", "owner_email", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignmentUpdated()["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.folder", "parent_container_id", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignmentUpdated()["parent_container_id"])),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "folder_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.folder", "update_time"),

					resource.TestCheckResourceAttrSet("stackit_authorization_folder_role_assignment.fra", "resource_id"),
					resource.TestCheckResourceAttrSet("stackit_authorization_folder_role_assignment.fra", "id"),
					resource.TestCheckResourceAttr("stackit_authorization_folder_role_assignment.fra", "role", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignmentUpdated()["role"])),
					resource.TestCheckResourceAttr("stackit_authorization_folder_role_assignment.fra", "subject", testutil.ConvertConfigVariable(testConfigVarsFolderRoleAssignmentUpdated()["subject"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccOrgRoleAssignmentResource(t *testing.T) {
	t.Log("Testing org role assignment resource")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		// only deleting the role assignment of org level
		CheckDestroy: testAccCheckOrganizationRoleAssignmentDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsOrgRoleAssignment,
				Config:          testutil.AuthorizationProviderConfig() + "\n" + resourceOrgRoleAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_authorization_organization_role_assignment.ora", "resource_id"),
					resource.TestCheckResourceAttrSet("stackit_authorization_organization_role_assignment.ora", "id"),
					resource.TestCheckResourceAttr("stackit_authorization_organization_role_assignment.ora", "role", testutil.ConvertConfigVariable(testConfigVarsOrgRoleAssignment["role"])),
					resource.TestCheckResourceAttr("stackit_authorization_organization_role_assignment.ora", "subject", testutil.ConvertConfigVariable(testConfigVarsOrgRoleAssignment["subject"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsProjectRoleAssignment,
				ResourceName:    "stackit_authorization_organization_role_assignment.ora",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_authorization_organization_role_assignment.ora"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_authorization_organization_role_assignment.ora")
					}
					resourceId, ok := r.Primary.Attributes["resource_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute resource_id")
					}
					role, ok := r.Primary.Attributes["role"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute role")
					}
					subject, ok := r.Primary.Attributes["subject"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute subject")
					}

					return fmt.Sprintf("%s,%s,%s", resourceId, role, subject), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsOrgRoleAssignmentUpdated(),
				Config:          testutil.AuthorizationProviderConfig() + "\n" + resourceOrgRoleAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_authorization_organization_role_assignment.ora", "resource_id"),
					resource.TestCheckResourceAttrSet("stackit_authorization_organization_role_assignment.ora", "id"),
					resource.TestCheckResourceAttr("stackit_authorization_organization_role_assignment.ora", "role", testutil.ConvertConfigVariable(testConfigVarsOrgRoleAssignmentUpdated()["role"])),
					resource.TestCheckResourceAttr("stackit_authorization_organization_role_assignment.ora", "subject", testutil.ConvertConfigVariable(testConfigVarsOrgRoleAssignmentUpdated()["subject"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckResourceManagerProjectsDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *resourcemanager.APIClient
	var err error
	if testutil.ResourceManagerCustomEndpoint == "" {
		client, err = resourcemanager.NewAPIClient()
	} else {
		client, err = resourcemanager.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.ResourceManagerCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var projectsToDestroy []string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_resourcemanager_project" {
			continue
		}
		// project terraform ID: "[container_id]"
		containerId := rs.Primary.ID
		projectsToDestroy = append(projectsToDestroy, containerId)
	}

	if testutil.OrganizationId == "" {
		return fmt.Errorf("no Org-ID is set")
	}
	containerParentId := testutil.OrganizationId

	projectsResp, err := client.ListProjects(ctx).ContainerParentId(containerParentId).Execute()
	if err != nil {
		return fmt.Errorf("getting projectsResp: %w", err)
	}

	items := *projectsResp.Items
	for i := range items {
		if *items[i].LifecycleState == resourcemanager.LIFECYCLESTATE_DELETING {
			continue
		}
		if !utils.Contains(projectsToDestroy, *items[i].ContainerId) {
			continue
		}

		err := client.DeleteProjectExecute(ctx, *items[i].ContainerId)
		if err != nil {
			return fmt.Errorf("destroying project %s during CheckDestroy: %w", *items[i].ContainerId, err)
		}
		_, err = wait.DeleteProjectWaitHandler(ctx, client, *items[i].ContainerId).WaitWithContext(ctx)
		if err != nil {
			return fmt.Errorf("destroying project %s during CheckDestroy: waiting for deletion %w", *items[i].ContainerId, err)
		}
	}
	return nil
}

func testAccCheckResourceManagerFoldersDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *resourcemanager.APIClient
	var err error
	if testutil.ResourceManagerCustomEndpoint == "" {
		client, err = resourcemanager.NewAPIClient()
	} else {
		client, err = resourcemanager.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.ResourceManagerCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	foldersToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_resourcemanager_folder" {
			continue
		}
		// project terraform ID: "[container_id]"
		containerId := rs.Primary.ID
		foldersToDestroy = append(foldersToDestroy, containerId)
	}

	if testutil.OrganizationId == "" {
		return fmt.Errorf("no Org-ID is set")
	}
	containerParentId := testutil.OrganizationId

	foldersResponse, err := client.ListFolders(ctx).ContainerParentId(containerParentId).Execute()
	if err != nil {
		return fmt.Errorf("getting foldersResponse: %w", err)
	}

	items := *foldersResponse.Items
	for i := range items {
		if !utils.Contains(foldersToDestroy, *items[i].ContainerId) {
			continue
		}

		err := client.DeleteFolder(ctx, *items[i].ContainerId).Execute()
		if err != nil {
			return fmt.Errorf("destroying folder %s during CheckDestroy: %w", *items[i].ContainerId, err)
		}
	}
	return nil
}

func testAccCheckOrganizationRoleAssignmentDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *authorization.APIClient
	var err error
	if testutil.AuthorizationCustomEndpoint == "" {
		client, err = authorization.NewAPIClient()
	} else {
		client, err = authorization.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.AuthorizationCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var orgRoleAssignmentsToDestroy []authorization.Member
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_authorization_organization_role_assignment" {
			continue
		}
		// project terraform ID: "resource_id,role,subject"
		terraformId := strings.Split(rs.Primary.ID, ",")

		orgRoleAssignmentsToDestroy = append(
			orgRoleAssignmentsToDestroy,
			authorization.Member{
				Role:    utils.Ptr(terraformId[1]),
				Subject: utils.Ptr(terraformId[2]),
			},
		)
	}

	if testutil.OrganizationId == "" {
		return fmt.Errorf("no Org-ID is set")
	}
	containerParentId := testutil.OrganizationId

	payload := authorization.RemoveMembersPayload{
		ResourceType: utils.Ptr("organization"),
		Members:      &orgRoleAssignmentsToDestroy,
	}

	// Ignore error. If this request errors the org role assignment has been successfully deleted by terraform itself.
	_, _ = client.RemoveMembers(ctx, containerParentId).RemoveMembersPayload(payload).Execute()
	return nil
}
