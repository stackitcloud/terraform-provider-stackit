package resourcemanager_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Project resource data
var projectResource = map[string]string{
	"name":                fmt.Sprintf("acc-pj-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"parent_container_id": testutil.TestProjectParentContainerID,
	"parent_uuid":         testutil.TestProjectParentUUID,
	"billing_reference":   "TEST-REF",
	"new_label":           "a-label",
}

func resourceConfig(name string, label *string) string {
	labelConfig := ""
	if label != nil {
		labelConfig = fmt.Sprintf("new_label = %q", *label)
	}
	return fmt.Sprintf(`
				%[1]s

				resource "stackit_resourcemanager_project" "project_container_id" {
					parent_container_id = "%[2]s"
					name = "%[3]s"
					labels = {
						"billing_reference" = "%[4]s"
						%[5]s
					}
					owner_email = "%[6]s"
				}

				resource "stackit_resourcemanager_project" "project_uuid" {
					parent_container_id = "%[7]s"
					name = "%[3]s-uuid"
					owner_email = "%[6]s"
				}
				`,
		testutil.ResourceManagerProviderConfig(),
		projectResource["parent_container_id"],
		name,
		projectResource["billing_reference"],
		labelConfig,
		testutil.TestProjectServiceAccountEmail,
		projectResource["parent_uuid"],
	)
}

func TestAccResourceManagerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceManagerDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: resourceConfig(projectResource["name"], nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Parent container id project data
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project_container_id", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project_container_id", "project_id"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project_container_id", "name", projectResource["name"]),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project_container_id", "parent_container_id", projectResource["parent_container_id"]),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project_container_id", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project_container_id", "labels.billing_reference", projectResource["billing_reference"]),

					// Parent UUID project data
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project_uuid", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project_uuid", "project_id"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project_uuid", "name", fmt.Sprintf("%s-uuid", projectResource["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project_uuid", "parent_container_id", projectResource["parent_uuid"]),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_resourcemanager_project" "project_container" {
						container_id = stackit_resourcemanager_project.project_container_id.container_id
					}
					
					data "stackit_resourcemanager_project" "project_uuid" {
						project_id = stackit_resourcemanager_project.project_container_id.project_id
					}

					data "stackit_resourcemanager_project" "project_both" {
						container_id = stackit_resourcemanager_project.project_container_id.container_id
						project_id = stackit_resourcemanager_project.project_container_id.project_id
					}
					`,
					resourceConfig(projectResource["name"], nil),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Container project data
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_container", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_container", "container_id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_container", "project_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_container", "name", projectResource["name"]),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_container", "parent_container_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_container", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_container", "labels.billing_reference", projectResource["billing_reference"]),

					// UUID project data
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_uuid", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_uuid", "container_id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_uuid", "project_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_uuid", "name", projectResource["name"]),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_uuid", "parent_container_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_uuid", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_uuid", "labels.billing_reference", projectResource["billing_reference"]),

					// Both project data
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_both", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_both", "container_id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_both", "project_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_both", "name", projectResource["name"]),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_both", "parent_container_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_both", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_both", "labels.billing_reference", projectResource["billing_reference"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_resourcemanager_project.project",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_resourcemanager_project.project"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_resourcemanager_project.project")
					}
					containerId, ok := r.Primary.Attributes["container_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute container_id")
					}

					return containerId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// The owner_email attributes don't exist in the
				// API, therefore there is no value for it during import.
				ImportStateVerifyIgnore: []string{"owner_email"},
			},
			// Update
			{
				Config: resourceConfig(fmt.Sprintf("%s-new", projectResource["name"]), utils.Ptr("a-label")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Project data
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.project", "container_id"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "name", fmt.Sprintf("%s-new", projectResource["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "parent_container_id", projectResource["parent_container_id"]),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "labels.%", "2"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "labels.billing_reference", projectResource["billing_reference"]),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.project", "labels.new_label", projectResource["new_label"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckResourceManagerDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *resourcemanager.APIClient
	var err error
	if testutil.ResourceManagerCustomEndpoint == "" {
		client, err = resourcemanager.NewAPIClient()
	} else {
		client, err = resourcemanager.NewAPIClient(
			config.WithEndpoint(testutil.ResourceManagerCustomEndpoint),
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

	projectsResp, err := client.GetProjects(ctx).ContainerParentId(projectResource["parent_container_id"]).Execute()
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
