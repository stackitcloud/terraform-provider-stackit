package postgresflex_test

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
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Instance resource data
var instanceResource = map[string]string{
	"project_id":             testutil.ProjectId,
	"name":                   fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum)),
	"acl":                    "192.168.0.0/16",
	"backup_schedule":        "00 16 * * *",
	"backup_schedule_update": "00 12 * * *",
	"flavor_cpu":             "2",
	"flavor_ram":             "4",
	"flavor_description":     "Small, Compute optimized",
	"replicas":               "1",
	"storage_class":          "premium-perf12-stackit",
	"storage_size":           "5",
	"version":                "14",
	"flavor_id":              "2.4",
}

// User resource data
var userResource = map[string]string{
	"username":   fmt.Sprintf("tf-acc-user-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlpha)),
	"role":       "login",
	"project_id": instanceResource["project_id"],
}

func configResources() string {
	return fmt.Sprintf(`
				%s

				resource "stackit_postgresflex_instance" "instance" {
					project_id = "%s"
					name    = "%s"
					acl = ["%s"]
					backup_schedule = "%s"
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
				}

				resource "stackit_postgresflex_user" "user" {
					project_id = stackit_postgresflex_instance.instance.project_id
					instance_id = stackit_postgresflex_instance.instance.instance_id
					username = "%s"
					roles = ["%s"]
				}
				`,
		testutil.PostgresFlexProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["acl"],
		instanceResource["backup_schedule"],
		instanceResource["flavor_cpu"],
		instanceResource["flavor_ram"],
		instanceResource["replicas"],
		instanceResource["storage_class"],
		instanceResource["storage_size"],
		instanceResource["version"],
		userResource["username"],
		userResource["role"],
	)
}

func TestAccPostgresFlexFlexResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPostgresFlexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: configResources(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "backup_schedule", instanceResource["backup_schedule"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "replicas", instanceResource["replicas"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.class", instanceResource["storage_class"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.size", instanceResource["storage_size"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "version", instanceResource["version"]),

					// User
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_user.user", "project_id",
						"stackit_postgresflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_user.user", "instance_id",
						"stackit_postgresflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "user_id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "password"),
				),
			},
			// data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_postgresflex_instance" "instance" {
						project_id     = stackit_postgresflex_instance.instance.project_id
						instance_id    = stackit_postgresflex_instance.instance.instance_id
					}
					
					data "stackit_postgresflex_user" "user" {
						project_id     = stackit_postgresflex_instance.instance.project_id
						instance_id    = stackit_postgresflex_instance.instance.instance_id
						user_id        = stackit_postgresflex_user.user.user_id
					}
					`,
					configResources(),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttrPair(
						"data.stackit_postgresflex_instance.instance", "project_id",
						"stackit_postgresflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_postgresflex_instance.instance", "instance_id",
						"stackit_postgresflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_postgresflex_user.user", "instance_id",
						"stackit_postgresflex_user.user", "instance_id",
					),

					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "backup_schedule", instanceResource["backup_schedule"]),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "flavor.id", instanceResource["flavor_id"]),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "flavor.description", instanceResource["flavor_description"]),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "replicas", instanceResource["replicas"]),

					// User data
					resource.TestCheckResourceAttr("data.stackit_postgresflex_user.user", "project_id", userResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_user.user", "user_id"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_user.user", "username", userResource["username"]),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_user.user", "roles.0", userResource["role"]),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_user.user", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_user.user", "port"),
				),
			},
			// Import
			{
				ResourceName: "stackit_postgresflex_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_postgresflex_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_postgresflex_instance.instance")
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
			{
				ResourceName: "stackit_postgresflex_user.user",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_postgresflex_user.user"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_postgresflex_user.user")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					userId, ok := r.Primary.Attributes["user_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute user_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, userId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update
			{
				Config: fmt.Sprintf(`
				%s

				resource "stackit_postgresflex_instance" "instance" {
					project_id = "%s"
					name    = "%s"
					acl = ["%s"]
					backup_schedule = "%s"
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
				}
				`,
					testutil.PostgresFlexProviderConfig(),
					instanceResource["project_id"],
					instanceResource["name"],
					instanceResource["acl"],
					instanceResource["backup_schedule_update"],
					instanceResource["flavor_cpu"],
					instanceResource["flavor_ram"],
					instanceResource["replicas"],
					instanceResource["storage_class"],
					instanceResource["storage_size"],
					instanceResource["version"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "backup_schedule", instanceResource["backup_schedule_update"]),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "replicas", instanceResource["replicas"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.class", instanceResource["storage_class"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.size", instanceResource["storage_size"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "version", instanceResource["version"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckPostgresFlexDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *postgresflex.APIClient
	var err error
	if testutil.PostgresFlexCustomEndpoint == "" {
		client, err = postgresflex.NewAPIClient()
	} else {
		client, err = postgresflex.NewAPIClient(
			config.WithEndpoint(testutil.PostgresFlexCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_postgresql_instance" {
			continue
		}
		// instance terraform ID: = "[project_id],[instance_id]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.GetInstances(ctx, testutil.ProjectId).Execute()
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
