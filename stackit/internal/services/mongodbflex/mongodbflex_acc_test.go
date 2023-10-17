package mongodbflex_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Instance resource data
var instanceResource = map[string]string{
	"project_id":         testutil.ProjectId,
	"name":               fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum)),
	"acl":                "192.168.0.0/16",
	"flavor_cpu":         "2",
	"flavor_ram":         "4",
	"flavor_description": "Small, Compute optimized",
	"replicas":           "1",
	"storage_class":      "premium-perf2-mongodb",
	"storage_size":       "10",
	"version":            "5.0",
	"version_updated":    "6.0",
	"options_type":       "Single",
	"flavor_id":          "2.4",
}

func configResources(version string) string {
	return fmt.Sprintf(`
				%s

				resource "stackit_mongodbflex_instance" "instance" {
					project_id = "%s"
					name    = "%s"
					acl = ["%s"]
					flavor = {
						cpu = %s
						ram = %s
					}
					replicas = %s
					storage = {
						class = "%s"
						size = %s
					}
					version = "%s"
					options = {
						type = "%s"
					}
				}
				`,
		testutil.MongoDBFlexProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["acl"],
		instanceResource["flavor_cpu"],
		instanceResource["flavor_ram"],
		instanceResource["replicas"],
		instanceResource["storage_class"],
		instanceResource["storage_size"],
		version,
		instanceResource["options_type"],
	)
}

func TestAccMongoDBFlexFlexResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMongoDBFlexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: configResources(instanceResource["version"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "replicas", instanceResource["replicas"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.class", instanceResource["storage_class"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.size", instanceResource["storage_size"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.type", instanceResource["options_type"]),
				),
			},
			// data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_mongodbflex_instance" "instance" {
						project_id     = stackit_mongodbflex_instance.instance.project_id
						instance_id    = stackit_mongodbflex_instance.instance.instance_id
					}
					`,
					configResources(instanceResource["version"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttrPair(
						"data.stackit_mongodbflex_instance.instance", "project_id",
						"stackit_mongodbflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_mongodbflex_instance.instance", "instance_id",
						"stackit_mongodbflex_instance.instance", "instance_id",
					),

					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.id", instanceResource["flavor_id"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.description", instanceResource["flavor_description"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "replicas", instanceResource["replicas"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "options.type", instanceResource["options_type"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_mongodbflex_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_mongodbflex_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_mongodbflex_instance.instance")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: configResources(instanceResource["version_updated"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "replicas", instanceResource["replicas"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.class", instanceResource["storage_class"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.size", instanceResource["storage_size"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "version", instanceResource["version_updated"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.type", instanceResource["options_type"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckMongoDBFlexDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *mongodbflex.APIClient
	var err error
	if testutil.MongoDBFlexCustomEndpoint == "" {
		client, err = mongodbflex.NewAPIClient()
	} else {
		client, err = mongodbflex.NewAPIClient(
			config.WithEndpoint(testutil.MongoDBFlexCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_mongodbflex_instance" {
			continue
		}
		// instance terraform ID: = "[project_id],[instance_id]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.GetInstances(ctx, testutil.ProjectId).Tag("").Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	items := *instancesResp.Items
	for i := range items {
		if items[i].Id == nil {
			continue
		}
		if utils.Contains(instancesToDestroy, *items[i].Id) {
			err := client.DeleteInstanceExecute(ctx, testutil.ProjectId, *items[i].Id)
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: %w", *items[i].Id, err)
			}
			_, err = wait.DeleteInstanceWaitHandler(ctx, client, testutil.ProjectId, *items[i].Id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: waiting for deletion %w", *items[i].Id, err)
			}
		}
	}
	return nil
}
