package authorization_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"sync"
	"testing"

	_ "embed"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-project-role-assignment.tf
	resourceProjectRoleAssignment string

	//go:embed testdata/resource-project-role-assignment-duplicate.tf
	resourceProjectRoleAssignmentDuplicate string

	//go:embed testdata/resource-folder-role-assignment.tf
	resourceFolderRoleAssignment string

	//go:embed testdata/resource-folder-role-assignment-duplicate.tf
	resourceFolderRoleAssignmentDuplicate string

	//go:embed testdata/resource-org-role-assignment.tf
	resourceOrgRoleAssignment string

	//go:embed testdata/resource-org-role-assignment-duplicate.tf
	resourceOrgRoleAssignmentDuplicate string

	//go:embed testdata/resource-project-custom-role.tf
	resourceProjectCustomRole string

	//go:embed testdata/resource-folder-custom-role.tf
	resourceFolderCustomRole string

	//go:embed testdata/resource-organization-custom-role.tf
	resourceOrganizationCustomRole string

	//go:embed testdata/resource-service-account-role-assignment.tf
	resourceServiceAccountRoleAssignment string

	//go:embed testdata/resource-service-account-role-assignment-duplicate.tf
	resourceServiceAccountRoleAssignmentDuplicate string
)

var (
	testProjectName          = fmt.Sprintf("proj-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
	testFolderName           = fmt.Sprintf("folder-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
	testCustomRoleFolderName = fmt.Sprintf("folder-custom-role-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
)

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

var testConfigVarsProjectCustomRole = config.Variables{
	"project_id":         config.StringVariable(testutil.ProjectId),
	"role_name":          config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"role_description":   config.StringVariable("Some description"),
	"role_permissions_0": config.StringVariable("iam.role.list"),
}

var testConfigVarsProjectCustomRoleUpdated = config.Variables{
	"project_id":         config.StringVariable(testutil.ProjectId),
	"role_name":          config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"role_description":   config.StringVariable("Updated description"),
	"role_permissions_0": config.StringVariable("iam.role.edit"),
}

var testConfigVarsFolderCustomRole = config.Variables{
	"folder_name":         config.StringVariable(testCustomRoleFolderName),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.OrganizationId),
	"role_name":           config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"role_description":    config.StringVariable("Some description"),
	"role_permissions_0":  config.StringVariable("iam.role.list"),
}

var testConfigVarsFolderCustomRoleUpdated = config.Variables{
	"folder_name":         config.StringVariable(testCustomRoleFolderName),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.OrganizationId),
	"role_name":           config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"role_description":    config.StringVariable("Updated description"),
	"role_permissions_0":  config.StringVariable("iam.role.edit"),
}

var testConfigVarsOrganizationCustomRole = config.Variables{
	"organization_id":    config.StringVariable(testutil.OrganizationId),
	"role_name":          config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"role_description":   config.StringVariable("Some description"),
	"role_permissions_0": config.StringVariable("iam.role.list"),
}

var testConfigVarsOrganizationCustomRoleUpdated = config.Variables{
	"organization_id":    config.StringVariable(testutil.OrganizationId),
	"role_name":          config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"role_description":   config.StringVariable("Updated description"),
	"role_permissions_0": config.StringVariable("iam.role.edit"),
}

var testConfigVarsServiceAccountRoleAssignment = config.Variables{
	"project_id":  config.StringVariable(testutil.ProjectId),
	"name":        config.StringVariable(fmt.Sprintf("sa-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"act_as_name": config.StringVariable(fmt.Sprintf("act-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"role":        config.StringVariable("user"),
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

func testConfigVarsServiceAccountRoleAssignmentUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsServiceAccountRoleAssignment))
	maps.Copy(tempConfig, testConfigVarsServiceAccountRoleAssignment)

	tempConfig["role"] = config.StringVariable("owner")
	return tempConfig
}

func TestAccProjectRoleAssignmentResource(t *testing.T) {
	t.Log("Testing project role assignment resource")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		// deleting project will also delete project role assignments
		CheckDestroy: testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsProjectRoleAssignment,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceProjectRoleAssignment,
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
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceProjectRoleAssignment,
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
			// Duplicate assignment should fail
			{
				ConfigVariables: testConfigVarsProjectRoleAssignmentUpdated(),
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceProjectRoleAssignmentDuplicate,
				ExpectError:     regexp.MustCompile(`Error while checking for duplicate role assignments`),
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
		CheckDestroy: testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsFolderRoleAssignment,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceFolderRoleAssignment,
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
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceFolderRoleAssignment,
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
			// Duplicate assignment should fail
			{
				ConfigVariables: testConfigVarsFolderRoleAssignmentUpdated(),
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceFolderRoleAssignmentDuplicate,
				ExpectError:     regexp.MustCompile(`Error while checking for duplicate role assignments`),
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
		CheckDestroy: testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsOrgRoleAssignment,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceOrgRoleAssignment,
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
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceOrgRoleAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_authorization_organization_role_assignment.ora", "resource_id"),
					resource.TestCheckResourceAttrSet("stackit_authorization_organization_role_assignment.ora", "id"),
					resource.TestCheckResourceAttr("stackit_authorization_organization_role_assignment.ora", "role", testutil.ConvertConfigVariable(testConfigVarsOrgRoleAssignmentUpdated()["role"])),
					resource.TestCheckResourceAttr("stackit_authorization_organization_role_assignment.ora", "subject", testutil.ConvertConfigVariable(testConfigVarsOrgRoleAssignmentUpdated()["subject"])),
				),
			},
			// Duplicate assignment should fail
			{
				ConfigVariables: testConfigVarsOrgRoleAssignmentUpdated(),
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceOrgRoleAssignmentDuplicate,
				ExpectError:     regexp.MustCompile(`Error while checking for duplicate role assignments`),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccServiceAccountRoleAssignmentResource(t *testing.T) {
	t.Log("Testing service-account (act-as) role assignment resource")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsServiceAccountRoleAssignment,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceServiceAccountRoleAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_authorization_service_account_role_assignment.sa", "resource_id"),
					resource.TestCheckResourceAttrSet("stackit_authorization_service_account_role_assignment.sa", "id"),
					resource.TestCheckResourceAttr("stackit_authorization_service_account_role_assignment.sa", "role", testutil.ConvertConfigVariable(testConfigVarsServiceAccountRoleAssignment["role"])),
					resource.TestCheckResourceAttrPair(
						"stackit_service_account.act_as", "email",
						"stackit_authorization_service_account_role_assignment.sa", "subject",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsServiceAccountRoleAssignment,
				ResourceName:    "stackit_authorization_service_account_role_assignment.sa",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_authorization_service_account_role_assignment.sa"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource")
					}
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["resource_id"], r.Primary.Attributes["role"], r.Primary.Attributes["subject"]), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsServiceAccountRoleAssignmentUpdated(),
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceServiceAccountRoleAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_authorization_service_account_role_assignment.sa", "resource_id"),
					resource.TestCheckResourceAttrSet("stackit_authorization_service_account_role_assignment.sa", "id"),
					resource.TestCheckResourceAttr("stackit_authorization_service_account_role_assignment.sa", "role", testutil.ConvertConfigVariable(testConfigVarsServiceAccountRoleAssignmentUpdated()["role"])),
					resource.TestCheckResourceAttrPair(
						"stackit_service_account.act_as", "email",
						"stackit_authorization_service_account_role_assignment.sa", "subject",
					),
				),
			},
			// Duplicate assignment should fail
			{
				ConfigVariables: testConfigVarsServiceAccountRoleAssignmentUpdated(),
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + "\n" + resourceServiceAccountRoleAssignmentDuplicate,
				ExpectError:     regexp.MustCompile(`Error while checking for duplicate role assignments`),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccProjectCustomRoleResource(t *testing.T) {
	t.Log("Testing project custom role resource")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigVariables: testConfigVarsProjectCustomRole,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + resourceProjectCustomRole,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.project_custom_role", "resource_id", testutil.ConvertConfigVariable(testConfigVarsProjectCustomRole["project_id"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.project_custom_role", "name", testutil.ConvertConfigVariable(testConfigVarsProjectCustomRole["role_name"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.project_custom_role", "description", testutil.ConvertConfigVariable(testConfigVarsProjectCustomRole["role_description"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.project_custom_role", "permissions.#", "1"),
					resource.TestCheckTypeSetElemAttr("stackit_authorization_project_custom_role.project_custom_role", "permissions.*", testutil.ConvertConfigVariable(testConfigVarsProjectCustomRole["role_permissions_0"])),
					resource.TestCheckResourceAttrSet("stackit_authorization_project_custom_role.project_custom_role", "role_id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsProjectCustomRole,
				Config: fmt.Sprintf(`
                %s

                data "stackit_authorization_project_custom_role" "project_custom_role" {
                   resource_id  = stackit_authorization_project_custom_role.project_custom_role.resource_id
                   role_id  = stackit_authorization_project_custom_role.project_custom_role.role_id
                }
                `,
					testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig()+resourceProjectCustomRole,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_authorization_project_custom_role.project_custom_role", "resource_id", testutil.ConvertConfigVariable(testConfigVarsProjectCustomRole["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.project_custom_role", "resource_id",
						"data.stackit_authorization_project_custom_role.project_custom_role", "resource_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.project_custom_role", "role_id",
						"data.stackit_authorization_project_custom_role.project_custom_role", "role_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.project_custom_role", "name",
						"data.stackit_authorization_project_custom_role.project_custom_role", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.project_custom_role", "description",
						"data.stackit_authorization_project_custom_role.project_custom_role", "description",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.project_custom_role", "permissions.#",
						"data.stackit_authorization_project_custom_role.project_custom_role", "permissions.#",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_project_custom_role.project_custom_role", "permissions.*",
						"data.stackit_authorization_project_custom_role.project_custom_role", "permissions.*",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsProjectCustomRole,
				ResourceName:    "stackit_authorization_project_custom_role.project_custom_role",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_authorization_project_custom_role.project_custom_role"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_authorization_project_custom_role.project_custom_role")
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
				ConfigVariables: testConfigVarsProjectCustomRoleUpdated,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + resourceProjectCustomRole,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.project_custom_role", "resource_id", testutil.ConvertConfigVariable(testConfigVarsProjectCustomRoleUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.project_custom_role", "name", testutil.ConvertConfigVariable(testConfigVarsProjectCustomRoleUpdated["role_name"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.project_custom_role", "description", testutil.ConvertConfigVariable(testConfigVarsProjectCustomRoleUpdated["role_description"])),
					resource.TestCheckResourceAttr("stackit_authorization_project_custom_role.project_custom_role", "permissions.#", "1"),
					resource.TestCheckTypeSetElemAttr("stackit_authorization_project_custom_role.project_custom_role", "permissions.*", testutil.ConvertConfigVariable(testConfigVarsProjectCustomRoleUpdated["role_permissions_0"])),
					resource.TestCheckResourceAttrSet("stackit_authorization_project_custom_role.project_custom_role", "role_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccFolderCustomRoleResource(t *testing.T) {
	t.Log("Testing folder custom role resource")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigVariables: testConfigVarsFolderCustomRole,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + resourceFolderCustomRole,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_authorization_folder_custom_role.folder_custom_role", "name", testutil.ConvertConfigVariable(testConfigVarsFolderCustomRole["role_name"])),
					resource.TestCheckResourceAttr("stackit_authorization_folder_custom_role.folder_custom_role", "description", testutil.ConvertConfigVariable(testConfigVarsFolderCustomRole["role_description"])),
					resource.TestCheckResourceAttr("stackit_authorization_folder_custom_role.folder_custom_role", "permissions.#", "1"),
					resource.TestCheckTypeSetElemAttr("stackit_authorization_folder_custom_role.folder_custom_role", "permissions.*", testutil.ConvertConfigVariable(testConfigVarsFolderCustomRole["role_permissions_0"])),
					resource.TestCheckResourceAttrSet("stackit_authorization_folder_custom_role.folder_custom_role", "role_id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsFolderCustomRole,
				Config: fmt.Sprintf(`
                %s

                data "stackit_authorization_folder_custom_role" "folder_custom_role" {
                   resource_id  = stackit_authorization_folder_custom_role.folder_custom_role.resource_id
                   role_id  = stackit_authorization_folder_custom_role.folder_custom_role.role_id
                }
                `,
					testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig()+resourceFolderCustomRole,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_folder_custom_role.folder_custom_role", "resource_id",
						"data.stackit_authorization_folder_custom_role.folder_custom_role", "resource_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_folder_custom_role.folder_custom_role", "role_id",
						"data.stackit_authorization_folder_custom_role.folder_custom_role", "role_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_folder_custom_role.folder_custom_role", "name",
						"data.stackit_authorization_folder_custom_role.folder_custom_role", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_folder_custom_role.folder_custom_role", "description",
						"data.stackit_authorization_folder_custom_role.folder_custom_role", "description",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_folder_custom_role.folder_custom_role", "permissions.#",
						"data.stackit_authorization_folder_custom_role.folder_custom_role", "permissions.#",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_folder_custom_role.folder_custom_role", "permissions.*",
						"data.stackit_authorization_folder_custom_role.folder_custom_role", "permissions.*",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsFolderCustomRole,
				ResourceName:    "stackit_authorization_folder_custom_role.folder_custom_role",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_authorization_folder_custom_role.folder_custom_role"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_authorization_folder_custom_role.folder_custom_role")
					}
					roleId, ok := r.Primary.Attributes["role_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute role_id")
					}
					folderId, ok := r.Primary.Attributes["resource_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute resource_id")
					}

					return fmt.Sprintf("%s,%s", folderId, roleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsFolderCustomRoleUpdated,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + resourceFolderCustomRole,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_authorization_folder_custom_role.folder_custom_role", "name", testutil.ConvertConfigVariable(testConfigVarsFolderCustomRoleUpdated["role_name"])),
					resource.TestCheckResourceAttr("stackit_authorization_folder_custom_role.folder_custom_role", "description", testutil.ConvertConfigVariable(testConfigVarsFolderCustomRoleUpdated["role_description"])),
					resource.TestCheckResourceAttr("stackit_authorization_folder_custom_role.folder_custom_role", "permissions.#", "1"),
					resource.TestCheckTypeSetElemAttr("stackit_authorization_folder_custom_role.folder_custom_role", "permissions.*", testutil.ConvertConfigVariable(testConfigVarsFolderCustomRoleUpdated["role_permissions_0"])),
					resource.TestCheckResourceAttrSet("stackit_authorization_folder_custom_role.folder_custom_role", "role_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccOrganizationCustomRoleResource(t *testing.T) {
	t.Log("Testing org custom role resource")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigVariables: testConfigVarsOrganizationCustomRole,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + resourceOrganizationCustomRole,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_authorization_organization_custom_role.organization_custom_role", "resource_id", testutil.ConvertConfigVariable(testConfigVarsOrganizationCustomRole["organization_id"])),
					resource.TestCheckResourceAttr("stackit_authorization_organization_custom_role.organization_custom_role", "name", testutil.ConvertConfigVariable(testConfigVarsOrganizationCustomRole["role_name"])),
					resource.TestCheckResourceAttr("stackit_authorization_organization_custom_role.organization_custom_role", "description", testutil.ConvertConfigVariable(testConfigVarsOrganizationCustomRole["role_description"])),
					resource.TestCheckResourceAttr("stackit_authorization_organization_custom_role.organization_custom_role", "permissions.#", "1"),
					resource.TestCheckTypeSetElemAttr("stackit_authorization_organization_custom_role.organization_custom_role", "permissions.*", testutil.ConvertConfigVariable(testConfigVarsOrganizationCustomRole["role_permissions_0"])),
					resource.TestCheckResourceAttrSet("stackit_authorization_organization_custom_role.organization_custom_role", "role_id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsOrganizationCustomRole,
				Config: fmt.Sprintf(`
                %s

                data "stackit_authorization_organization_custom_role" "organization_custom_role" {
                   resource_id  = stackit_authorization_organization_custom_role.organization_custom_role.resource_id
                   role_id  = stackit_authorization_organization_custom_role.organization_custom_role.role_id
                }
                `,
					testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig()+resourceOrganizationCustomRole,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_authorization_organization_custom_role.organization_custom_role", "resource_id", testutil.ConvertConfigVariable(testConfigVarsOrganizationCustomRole["organization_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_organization_custom_role.organization_custom_role", "resource_id",
						"data.stackit_authorization_organization_custom_role.organization_custom_role", "resource_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_organization_custom_role.organization_custom_role", "role_id",
						"data.stackit_authorization_organization_custom_role.organization_custom_role", "role_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_organization_custom_role.organization_custom_role", "name",
						"data.stackit_authorization_organization_custom_role.organization_custom_role", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_organization_custom_role.organization_custom_role", "description",
						"data.stackit_authorization_organization_custom_role.organization_custom_role", "description",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_organization_custom_role.organization_custom_role", "permissions.#",
						"data.stackit_authorization_organization_custom_role.organization_custom_role", "permissions.#",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_authorization_organization_custom_role.organization_custom_role", "permissions.*",
						"data.stackit_authorization_organization_custom_role.organization_custom_role", "permissions.*",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsOrganizationCustomRole,
				ResourceName:    "stackit_authorization_organization_custom_role.organization_custom_role",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_authorization_organization_custom_role.organization_custom_role"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_authorization_organization_custom_role.organization_custom_role")
					}
					roleId, ok := r.Primary.Attributes["role_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute role_id")
					}

					return fmt.Sprintf("%s,%s", testutil.OrganizationId, roleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsOrganizationCustomRoleUpdated,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig() + resourceOrganizationCustomRole,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_authorization_organization_custom_role.organization_custom_role", "resource_id", testutil.ConvertConfigVariable(testConfigVarsOrganizationCustomRoleUpdated["organization_id"])),
					resource.TestCheckResourceAttr("stackit_authorization_organization_custom_role.organization_custom_role", "name", testutil.ConvertConfigVariable(testConfigVarsOrganizationCustomRoleUpdated["role_name"])),
					resource.TestCheckResourceAttr("stackit_authorization_organization_custom_role.organization_custom_role", "description", testutil.ConvertConfigVariable(testConfigVarsOrganizationCustomRoleUpdated["role_description"])),
					resource.TestCheckResourceAttr("stackit_authorization_organization_custom_role.organization_custom_role", "permissions.#", "1"),
					resource.TestCheckTypeSetElemAttr("stackit_authorization_organization_custom_role.organization_custom_role", "permissions.*", testutil.ConvertConfigVariable(testConfigVarsOrganizationCustomRoleUpdated["role_permissions_0"])),
					resource.TestCheckResourceAttrSet("stackit_authorization_organization_custom_role.organization_custom_role", "role_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAccCheckServiceAccountRoleAssignmentDestroy,
		testAccCheckResourceManagerProjectsDestroy,
		testAccCheckResourceManagerFoldersDestroy,
		testAccCheckOrganizationRoleAssignmentDestroy,
	}

	var errs []error

	wg := sync.WaitGroup{}
	wg.Add(len(checkFunctions))

	for _, f := range checkFunctions {
		go func() {
			err := f(s)
			if err != nil {
				errs = append(errs, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

func testAccCheckResourceManagerProjectsDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := resourcemanager.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.ResourceManagerCustomEndpoint, false)...)
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
	client, err := resourcemanager.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.ResourceManagerCustomEndpoint, false)...)
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
	client, err := authorization.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.AuthorizationCustomEndpoint, false)...)
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
				Role:    new(terraformId[1]),
				Subject: new(terraformId[2]),
			},
		)
	}

	if testutil.OrganizationId == "" {
		return fmt.Errorf("no Org-ID is set")
	}
	containerParentId := testutil.OrganizationId

	payload := authorization.RemoveMembersPayload{
		ResourceType: new("organization"),
		Members:      &orgRoleAssignmentsToDestroy,
	}

	// Ignore error. If this request errors the org role assignment has been successfully deleted by terraform itself.
	_, _ = client.RemoveMembers(ctx, containerParentId).RemoveMembersPayload(payload).Execute()
	return nil
}

func testAccCheckServiceAccountRoleAssignmentDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := authorization.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.AuthorizationCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_authorization_service_account_role_assignment" {
			continue
		}

		terraformId := strings.Split(rs.Primary.ID, ",")
		if len(terraformId) != 3 {
			continue
		}

		resourceId := terraformId[0]
		payload := authorization.RemoveMembersPayload{
			ResourceType: new("service-account"),
			Members: &[]authorization.Member{
				{
					Role:    new(terraformId[1]),
					Subject: new(terraformId[2]),
				},
			},
		}

		_, err = client.RemoveMembers(ctx, resourceId).RemoveMembersPayload(payload).Execute()
		if err != nil && !strings.Contains(err.Error(), "400") {
			return fmt.Errorf("destroying assignment %s: %w", rs.Primary.ID, err)
		}
	}
	return nil
}
