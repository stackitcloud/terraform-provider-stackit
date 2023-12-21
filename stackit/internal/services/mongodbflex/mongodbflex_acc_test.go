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
	"project_id":              testutil.ProjectId,
	"name":                    fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum)),
	"acl":                     "192.168.0.0/16",
	"flavor_cpu":              "2",
	"flavor_ram":              "4",
	"flavor_description":      "Small, Compute optimized",
	"replicas":                "1",
	"storage_class":           "premium-perf2-mongodb",
	"storage_size":            "10",
	"version":                 "5.0",
	"version_updated":         "6.0",
	"options_type":            "Single",
	"flavor_id":               "2.4",
	"backup_schedule":         "00 6 * * *",
	"backup_schedule_updated": "00 12 * * *",
	"backup_schedule_read":    "0 6 * * *",
}

// User resource data
var userResource = map[string]string{
	"username":   fmt.Sprintf("tf-acc-user-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlpha)),
	"role":       "read",
	"database":   "default",
	"project_id": instanceResource["project_id"],
}

func configResources(version, backupSchedule string) string {
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
					backup_schedule = "%s"
				}

				resource "stackit_mongodbflex_user" "user" {
					project_id = stackit_mongodbflex_instance.instance.project_id
					instance_id = stackit_mongodbflex_instance.instance.instance_id
					username = "%s"
					roles = ["%s"]
					database = "%s"
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
		backupSchedule,
		userResource["username"],
		userResource["role"],
		userResource["database"],
	)
}

func TestAccMongoDBFlexFlexResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMongoDBFlexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: configResources(instanceResource["version"], instanceResource["backup_schedule"]),
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
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "backup_schedule", instanceResource["backup_schedule"]),

					// User
					resource.TestCheckResourceAttrPair(
						"stackit_mongodbflex_user.user", "project_id",
						"stackit_mongodbflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_mongodbflex_user.user", "instance_id",
						"stackit_mongodbflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_user.user", "user_id"),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_user.user", "password"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_user.user", "username", userResource["username"]),
					resource.TestCheckResourceAttr("stackit_mongodbflex_user.user", "database", userResource["database"]),
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

					data "stackit_mongodbflex_user" "user" {
						project_id     = stackit_mongodbflex_instance.instance.project_id
						instance_id    = stackit_mongodbflex_instance.instance.instance_id
						user_id        = stackit_mongodbflex_user.user.user_id
					}
					`,
					configResources(instanceResource["version"], instanceResource["backup_schedule"]),
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
					resource.TestCheckResourceAttrPair(
						"data.stackit_mongodbflex_user.user", "instance_id",
						"stackit_mongodbflex_user.user", "instance_id",
					),

					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.id", instanceResource["flavor_id"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.description", instanceResource["flavor_description"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "replicas", instanceResource["replicas"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "options.type", instanceResource["options_type"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "backup_schedule", instanceResource["backup_schedule_read"]),

					// User data
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "project_id", userResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_mongodbflex_user.user", "user_id"),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "username", userResource["username"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "database", userResource["database"]),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "roles.0", userResource["role"]),
					resource.TestCheckResourceAttrSet("data.stackit_mongodbflex_user.user", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_mongodbflex_user.user", "port"),
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
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"backup_schedule"},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					if len(s) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(s))
					}
					if s[0].Attributes["backup_schedule"] != instanceResource["backup_schedule_read"] {
						return fmt.Errorf("expected backup_schedule %s, got %s", instanceResource["backup_schedule_read"], s[0].Attributes["backup_schedule"])
					}
					return nil
				},
			},
			{
				ResourceName: "stackit_mongodbflex_user.user",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_mongodbflex_user.user"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_mongodbflex_user.user")
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
				Config: configResources(instanceResource["version_updated"], instanceResource["backup_schedule_updated"]),
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
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "backup_schedule", instanceResource["backup_schedule_updated"]),
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

	instancesResp, err := client.ListInstances(ctx, testutil.ProjectId).Tag("").Execute()
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
