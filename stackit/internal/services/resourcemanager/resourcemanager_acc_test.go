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

				resource "stackit_resourcemanager_project" "parent_by_container" {
					parent_container_id = "%[2]s"
					name = "%[3]s"
					labels = {
						"billing_reference" = "%[4]s"
						%[5]s
					}
					owner_email = "%[7]s"
				}

				resource "stackit_resourcemanager_project" "parent_by_uuid" {
					parent_container_id = "%[6]s"
					name = "%[3]s-uuid"
                    owner_email = "%[7]s"
				}
				`,
		testutil.ResourceManagerProviderConfig(),
		projectResource["parent_container_id"],
		name,
		projectResource["billing_reference"],
		labelConfig,
		projectResource["parent_uuid"],
		testutil.TestProjectServiceAccountEmail,
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
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.parent_by_container", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.parent_by_container", "project_id"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "name", projectResource["name"]),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "parent_container_id", projectResource["parent_container_id"]),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "labels.billing_reference", projectResource["billing_reference"]),

					// Parent UUID project data
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.parent_by_uuid", "container_id"),
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.parent_by_uuid", "project_id"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_uuid", "name", fmt.Sprintf("%s-uuid", projectResource["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_uuid", "parent_container_id", projectResource["parent_uuid"]),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_resourcemanager_project" "project_by_container" {
						container_id = stackit_resourcemanager_project.parent_by_container.container_id
					}
					
					data "stackit_resourcemanager_project" "project_by_uuid" {
						project_id = stackit_resourcemanager_project.parent_by_container.project_id
					}

					data "stackit_resourcemanager_project" "project_by_both" {
						container_id = stackit_resourcemanager_project.parent_by_container.container_id
						project_id = stackit_resourcemanager_project.parent_by_container.project_id
					}
					`,
					resourceConfig(projectResource["name"], nil),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Container project data
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_container", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_container", "container_id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_container", "project_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_by_container", "name", projectResource["name"]),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_container", "parent_container_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_by_container", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_by_container", "labels.billing_reference", projectResource["billing_reference"]),

					// UUID project data
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_uuid", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_uuid", "container_id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_uuid", "project_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_by_uuid", "name", projectResource["name"]),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_uuid", "parent_container_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_by_uuid", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_by_uuid", "labels.billing_reference", projectResource["billing_reference"]),

					// Both project data
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_both", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_both", "container_id"),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_both", "project_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_by_both", "name", projectResource["name"]),
					resource.TestCheckResourceAttrSet("data.stackit_resourcemanager_project.project_by_both", "parent_container_id"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_by_both", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_resourcemanager_project.project_by_both", "labels.billing_reference", projectResource["billing_reference"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_resourcemanager_project.parent_by_container",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_resourcemanager_project.parent_by_container"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_resourcemanager_project.parent_by_container")
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
					resource.TestCheckResourceAttrSet("stackit_resourcemanager_project.parent_by_container", "container_id"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "name", fmt.Sprintf("%s-new", projectResource["name"])),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "parent_container_id", projectResource["parent_container_id"]),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "labels.%", "2"),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "labels.billing_reference", projectResource["billing_reference"]),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "labels.new_label", projectResource["new_label"]),
					resource.TestCheckResourceAttr("stackit_resourcemanager_project.parent_by_container", "owner_email", testutil.TestProjectServiceAccountEmail),
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

	projectsResp, err := client.ListProjects(ctx).ContainerParentId(projectResource["parent_container_id"]).Execute()
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
