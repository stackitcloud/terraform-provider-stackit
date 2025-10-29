package resourcemanager_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	sdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-project.tf
var resourceProject string

//go:embed testdata/resource-folder.tf
var resourceFolder string

var defaultLabels = config.ObjectVariable(
	map[string]config.Variable{
		"env": config.StringVariable("prod"),
	},
)

var projectNameParentContainerId = fmt.Sprintf("tfe2e-project-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var projectNameParentContainerIdUpdated = fmt.Sprintf("%s-updated", projectNameParentContainerId)

var projectNameParentUUID = fmt.Sprintf("tfe2e-project-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var projectNameParentUUIDUpdated = fmt.Sprintf("%s-updated", projectNameParentUUID)

var folderNameParentContainerId = fmt.Sprintf("tfe2e-folder-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var folderNameParentContainerIdUpdated = fmt.Sprintf("%s-updated", folderNameParentContainerId)

var folderNameParentUUID = fmt.Sprintf("tfe2e-folder-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var folderNameParentUUIDUpdated = fmt.Sprintf("%s-updated", folderNameParentUUID)

var testConfigResourceProjectParentContainerId = config.Variables{
	"name":                config.StringVariable(projectNameParentContainerId),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"labels": config.ObjectVariable(
		map[string]config.Variable{
			"env": config.StringVariable("prod"),
		},
	),
}

var testConfigResourceProjectParentUUID = config.Variables{
	"name":                config.StringVariable(projectNameParentUUID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentUUID),
	"labels":              defaultLabels,
}

var testConfigResourceFolderParentContainerId = config.Variables{
	"name":                config.StringVariable(folderNameParentContainerId),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"labels":              defaultLabels,
}

var testConfigResourceFolderParentUUID = config.Variables{
	"name":                config.StringVariable(folderNameParentUUID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentUUID),
	"labels":              defaultLabels,
}

func testConfigProjectNameParentContainerIdUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigResourceProjectParentContainerId))
	maps.Copy(tempConfig, testConfigResourceProjectParentContainerId)
	tempConfig["name"] = config.StringVariable(projectNameParentContainerIdUpdated)
	return tempConfig
}

func testConfigProjectNameParentUUIDUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigResourceProjectParentUUID))
	maps.Copy(tempConfig, testConfigResourceProjectParentUUID)
	tempConfig["name"] = config.StringVariable(projectNameParentUUIDUpdated)
	return tempConfig
}

func testConfigFolderNameParentContainerIdUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigResourceFolderParentContainerId))
	maps.Copy(tempConfig, testConfigResourceFolderParentContainerId)
	tempConfig["name"] = config.StringVariable(folderNameParentContainerIdUpdated)
	return tempConfig
}

func testConfigFolderNameParentUUIDUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigResourceFolderParentUUID))
	maps.Copy(tempConfig, testConfigResourceFolderParentUUID)
	tempConfig["name"] = config.StringVariable(folderNameParentUUIDUpdated)
	return tempConfig
}

func TestAccResourceManagerProjectContainerId(t *testing.T) {
	t.Logf("TestAccResourceManagerProjectContainerId name: %s", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["name"]))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigResourceProjectParentContainerId,
				Config:          testutil.ResourceManagerProviderConfig() + resourceProject,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "name", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["parent_container_id"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "owner_email", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "project_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "owner_email"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "update_time"),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigResourceProjectParentContainerId,
				Config: fmt.Sprintf(`
                    %s
                    %s

                    data "stackit_resourcemanager_project" "example" {
                        project_id = stackit_resourcemanager_project.example.project_id
                    }
                `, testutil.ResourceManagerProviderConfig(), resourceProject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.example", "name", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["name"])),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["parent_container_id"])),
					resource.TestCheckResourceAttrPair("data.stackit_resourcemanager_project.example", "container_id", "stackit_resourcemanager_project.example", "container_id"),
					resource.TestCheckResourceAttrPair("data.stackit_resourcemanager_project.example", "project_id", "stackit_resourcemanager_project.example", "project_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.example", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.example", "creation_time"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.example", "update_time"),
				),
			},
			// Import
			{
				ConfigVariables:   testConfigResourceProjectParentContainerId,
				ResourceName:      "stackit_resourcemanager_project.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return getImportIdFromID(s, "stackit_resourcemanager_project.example", "container_id")
				},
				ImportStateVerifyIgnore: []string{"owner_email"},
			},
			// Update
			{
				ConfigVariables: testConfigProjectNameParentContainerIdUpdated(),
				Config:          testutil.ResourceManagerProviderConfig() + resourceProject,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "name", testutil.ConvertConfigVariable(testConfigProjectNameParentContainerIdUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["parent_container_id"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "owner_email", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "project_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "owner_email"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "update_time"),
				),
			},
		},
	})
}

func TestAccResourceManagerProjectParentUUID(t *testing.T) {
	t.Logf("TestAccResourceManagerProjectParentUUID name: %s", testutil.ConvertConfigVariable(testConfigResourceProjectParentUUID["name"]))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigResourceProjectParentUUID,
				Config:          testutil.ResourceManagerProviderConfig() + resourceProject,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "name", testutil.ConvertConfigVariable(testConfigResourceProjectParentUUID["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceProjectParentUUID["parent_container_id"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "owner_email", testutil.ConvertConfigVariable(testConfigResourceProjectParentUUID["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "project_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "owner_email"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "update_time"),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigResourceProjectParentUUID,
				Config: fmt.Sprintf(`
                    %s
                    %s

                    data "stackit_resourcemanager_project" "example" {
                        project_id = stackit_resourcemanager_project.example.project_id
                    }
                `, testutil.ResourceManagerProviderConfig(), resourceProject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.example", "name", testutil.ConvertConfigVariable(testConfigResourceProjectParentUUID["name"])),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.example", "parent_container_id"),
					resource.TestCheckResourceAttrPair("data.stackit_resourcemanager_project.example", "container_id", "stackit_resourcemanager_project.example", "container_id"),
					resource.TestCheckResourceAttrPair("data.stackit_resourcemanager_project.example", "project_id", "stackit_resourcemanager_project.example", "project_id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.example", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.example", "creation_time"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.example", "update_time"),
				),
			},
			// Import
			{
				ConfigVariables:   testConfigResourceProjectParentUUID,
				ResourceName:      "stackit_resourcemanager_project.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return getImportIdFromID(s, "stackit_resourcemanager_project.example", "container_id")
				},
				ImportStateVerifyIgnore: []string{"owner_email", "parent_container_id"},
			},
			// Update
			{
				ConfigVariables: testConfigProjectNameParentUUIDUpdated(),
				Config:          testutil.ResourceManagerProviderConfig() + resourceProject,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "name", testutil.ConvertConfigVariable(testConfigProjectNameParentUUIDUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceProjectParentUUID["parent_container_id"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "owner_email", testutil.ConvertConfigVariable(testConfigResourceProjectParentUUID["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "project_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "owner_email"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.example", "update_time"),
				),
			},
		},
	})
}

func TestAccResourceManagerFolderContainerId(t *testing.T) {
	t.Logf("TestAccResourceManagerFolderContainerId name: %s", testutil.ConvertConfigVariable(testConfigResourceFolderParentContainerId["name"]))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigResourceFolderParentContainerId,
				Config:          testutil.ResourceManagerProviderConfig() + resourceFolder,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "name", testutil.ConvertConfigVariable(testConfigResourceFolderParentContainerId["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceFolderParentContainerId["parent_container_id"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "owner_email", testutil.ConvertConfigVariable(testConfigResourceFolderParentContainerId["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "folder_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "update_time"),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigResourceFolderParentContainerId,
				Config: fmt.Sprintf(`
					%s
					%s
	
					data "stackit_resourcemanager_folder" "example" {
						container_id = stackit_resourcemanager_folder.example.container_id
					}
				`, testutil.ResourceManagerProviderConfig(), resourceFolder),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_folder.example", "name", testutil.ConvertConfigVariable(testConfigResourceFolderParentContainerId["name"])),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_folder.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_folder.example", "labels.env", "prod"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_folder.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceFolderParentContainerId["parent_container_id"])),
					resource.TestCheckResourceAttrPair("data.stackit_resourcemanager_folder.example", "container_id", "stackit_resourcemanager_folder.example", "container_id"),
					resource.TestCheckResourceAttrPair("data.stackit_resourcemanager_folder.example", "project_id", "stackit_resourcemanager_folder.example", "project_id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_folder.example", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_folder.example", "creation_time"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_folder.example", "update_time"),
				),
			},
			// Import
			{
				ConfigVariables:   testConfigResourceFolderParentContainerId,
				ResourceName:      "stackit_resourcemanager_folder.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return getImportIdFromID(s, "stackit_resourcemanager_folder.example", "container_id")
				},
				ImportStateVerifyIgnore: []string{"owner_email"},
			},
			// Update
			{
				ConfigVariables: testConfigFolderNameParentContainerIdUpdated(),
				Config:          testutil.ResourceManagerProviderConfig() + resourceFolder,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "name", testutil.ConvertConfigVariable(testConfigFolderNameParentContainerIdUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigFolderNameParentContainerIdUpdated()["parent_container_id"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "owner_email", testutil.ConvertConfigVariable(testConfigFolderNameParentContainerIdUpdated()["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "folder_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "owner_email"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "update_time"),
				),
			},
		},
	})
}

func TestAccResourceManagerFolderParentUUID(t *testing.T) {
	t.Logf("TestAccResourceManagerFolderParentUUID name: %s", testutil.ConvertConfigVariable(testConfigResourceFolderParentContainerId["name"]))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigResourceFolderParentUUID,
				Config:          testutil.ResourceManagerProviderConfig() + resourceFolder,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "name", testutil.ConvertConfigVariable(testConfigResourceFolderParentUUID["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceFolderParentUUID["parent_container_id"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "owner_email", testutil.ConvertConfigVariable(testConfigResourceFolderParentUUID["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "folder_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "update_time"),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigResourceFolderParentUUID,
				Config: fmt.Sprintf(`
					%s
					%s
	
					data "stackit_resourcemanager_folder" "example" {
						container_id = stackit_resourcemanager_folder.example.container_id
					}
				`, testutil.ResourceManagerProviderConfig(), resourceFolder),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_folder.example", "name", testutil.ConvertConfigVariable(testConfigResourceFolderParentUUID["name"])),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_folder.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_folder.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_folder.example", "parent_container_id"),
					resource.TestCheckResourceAttrPair("data.stackit_resourcemanager_folder.example", "container_id", "stackit_resourcemanager_folder.example", "container_id"),
					resource.TestCheckResourceAttrPair("data.stackit_resourcemanager_folder.example", "project_id", "stackit_resourcemanager_folder.example", "project_id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_folder.example", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_folder.example", "creation_time"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_folder.example", "update_time"),
				),
			},
			// Import
			{
				ConfigVariables:   testConfigResourceFolderParentUUID,
				ResourceName:      "stackit_resourcemanager_folder.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return getImportIdFromID(s, "stackit_resourcemanager_folder.example", "container_id")
				},
				ImportStateVerifyIgnore: []string{"owner_email", "parent_container_id"},
			},
			// Update
			{
				ConfigVariables: testConfigFolderNameParentUUIDUpdated(),
				Config:          testutil.ResourceManagerProviderConfig() + resourceFolder,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "name", testutil.ConvertConfigVariable(testConfigFolderNameParentUUIDUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "parent_container_id", testutil.ConvertConfigVariable(testConfigFolderNameParentUUIDUpdated()["parent_container_id"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "owner_email", testutil.ConvertConfigVariable(testConfigFolderNameParentUUIDUpdated()["owner_email"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_folder.example", "labels.env", "prod"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "folder_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "owner_email"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_folder.example", "update_time"),
				),
			},
		},
	})
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAccCheckResourceManagerProjectsDestroy,
		testAccCheckResourceManagerFoldersDestroy,
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
	var client *resourcemanager.APIClient
	var err error
	if testutil.ResourceManagerCustomEndpoint == "" {
		client, err = resourcemanager.NewAPIClient()
	} else {
		client, err = resourcemanager.NewAPIClient(
			sdkConfig.WithEndpoint(testutil.ResourceManagerCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	projectsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_resourcemanager_project" {
			continue
		}
		// project terraform ID: "[container_id]"
		containerId := rs.Primary.ID
		projectsToDestroy = append(projectsToDestroy, containerId)
	}

	var containerParentId string
	switch {
	case testutil.TestProjectParentContainerID != "":
		containerParentId = testutil.TestProjectParentContainerID
	case testutil.TestProjectParentUUID != "":
		containerParentId = testutil.TestProjectParentUUID
	default:
		return fmt.Errorf("either TestProjectParentContainerID or TestProjectParentUUID must be set")
	}

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
			sdkConfig.WithEndpoint(testutil.ResourceManagerCustomEndpoint),
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

	var containerParentId string
	switch {
	case testutil.TestProjectParentContainerID != "":
		containerParentId = testutil.TestProjectParentContainerID
	case testutil.TestProjectParentUUID != "":
		containerParentId = testutil.TestProjectParentUUID
	default:
		return fmt.Errorf("either TestProjectParentContainerID or TestProjectParentUUID must be set")
	}

	projectsResp, err := client.ListFolders(ctx).ContainerParentId(containerParentId).Execute()
	if err != nil {
		return fmt.Errorf("getting projectsResp: %w", err)
	}

	items := *projectsResp.Items
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

func getImportIdFromID(s *terraform.State, resourceName, keyName string) (string, error) {
	r, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return "", fmt.Errorf("couldn't find resource %s", resourceName)
	}
	id, ok := r.Primary.Attributes[keyName]
	if !ok {
		return "", fmt.Errorf("couldn't find attribute %s", keyName)
	}
	return id, nil
}
