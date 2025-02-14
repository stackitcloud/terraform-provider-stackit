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
	"project_id":              testutil.ProjectId,
	"name":                    fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum)),
	"acl":                     "192.168.0.0/16",
	"backup_schedule":         "00 16 * * *",
	"backup_schedule_updated": "00 12 * * *",
	"flavor_cpu":              "2",
	"flavor_ram":              "4",
	"flavor_description":      "Small, Compute optimized",
	"replicas":                "1",
	"storage_class":           "premium-perf12-stackit",
	"storage_size":            "5",
	"version":                 "14",
	"flavor_id":               "2.4",
}

// User resource data
var userResource = map[string]string{
	"username":   fmt.Sprintf("tfaccuser%s", acctest.RandStringFromCharSet(4, acctest.CharSetAlpha)),
	"role":       "createdb",
	"project_id": instanceResource["project_id"],
}

// Database resource data
var databaseResource = map[string]string{
	"name": fmt.Sprintf("tfaccdb%s", acctest.RandStringFromCharSet(4, acctest.CharSetAlphaNum)),
}

func configResources(backupSchedule string) string {
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

				resource "stackit_postgresflex_database" "database" {
					project_id = stackit_postgresflex_instance.instance.project_id
					instance_id = stackit_postgresflex_instance.instance.instance_id
					name = "%s"
					owner = stackit_postgresflex_user.user.username
				}
				`,
		testutil.PostgresFlexProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["acl"],
		backupSchedule,
		instanceResource["flavor_cpu"],
		instanceResource["flavor_ram"],
		instanceResource["replicas"],
		instanceResource["storage_class"],
		instanceResource["storage_size"],
		instanceResource["version"],
		userResource["username"],
		userResource["role"],
		databaseResource["name"],
	)
}

func TestAccPostgresFlexFlexResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPostgresFlexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: configResources(instanceResource["backup_schedule"]),
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

					// Database
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_database.database", "project_id",
						"stackit_postgresflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_database.database", "instance_id",
						"stackit_postgresflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_postgresflex_database.database", "name", databaseResource["name"]),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_database.database", "owner",
						"stackit_postgresflex_user.user", "username",
					),
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

					data "stackit_postgresflex_database" "database" {
						project_id     = stackit_postgresflex_instance.instance.project_id
						instance_id    = stackit_postgresflex_instance.instance.instance_id
						database_id    = stackit_postgresflex_database.database.database_id
					}
					`,
					configResources(instanceResource["backup_schedule"]),
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

					// Database data
					resource.TestCheckResourceAttr("data.stackit_postgresflex_database.database", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_database.database", "name", databaseResource["name"]),
					resource.TestCheckResourceAttrPair(
						"data.stackit_postgresflex_database.database", "instance_id",
						"stackit_postgresflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_postgresflex_database.database", "owner",
						"data.stackit_postgresflex_user.user", "username",
					),
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
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
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
				ImportStateVerifyIgnore: []string{"password", "uri"},
			},
			{
				ResourceName: "stackit_postgresflex_database.database",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_postgresflex_database.database"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_postgresflex_database.database")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					databaseId, ok := r.Primary.Attributes["database_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute database_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, databaseId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: configResources(instanceResource["backup_schedule_updated"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "backup_schedule", instanceResource["backup_schedule_updated"]),
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
		client, err = postgresflex.NewAPIClient(
			config.WithRegion("eu01"),
		)
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
		if rs.Type != "stackit_postgresflex_instance" {
			continue
		}
		// instance terraform ID: = "[project_id],[instance_id]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.ListInstances(ctx, testutil.ProjectId).Execute()
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
				return fmt.Errorf("deleting instance %s during CheckDestroy: %w", *items[i].Id, err)
			}
			_, err = wait.DeleteInstanceWaitHandler(ctx, client, testutil.ProjectId, *items[i].Id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("deleting instance %s during CheckDestroy: waiting for deletion %w", *items[i].Id, err)
			}
			err = client.ForceDeleteInstanceExecute(ctx, testutil.ProjectId, *items[i].Id)
			if err != nil {
				return fmt.Errorf("force deleting instance %s during CheckDestroy: %w", *items[i].Id, err)
			}
		}
	}
	return nil
}
