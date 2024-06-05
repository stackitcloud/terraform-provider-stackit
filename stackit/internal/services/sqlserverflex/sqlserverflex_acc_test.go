package sqlserverflex_test

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
	"github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex"
	"github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Instance resource data
var instanceResource = map[string]string{
	"project_id":              testutil.ProjectId,
	"name":                    fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum)),
	"acl":                     "192.168.0.0/16",
	"flavor_cpu":              "4",
	"flavor_ram":              "16",
	"flavor_description":      "SQLServer-Flex-4.16-Standard-EU01",
	"storage_class":           "premium-perf2-stackit",
	"storage_size":            "40",
	"version":                 "2022",
	"replicas":                "1",
	"options_edition":         "developer",
	"options_retention_days":  "64",
	"flavor_id":               "4.16-Single",
	"backup_schedule":         "00 6 * * *",
	"backup_schedule_updated": "00 12 * * *",
}

func configResources(backupSchedule string) string {
	return fmt.Sprintf(`
				%s

				resource "stackit_sqlserverflex_instance" "instance" {
					project_id = "%s"
					name    = "%s"
					acl = ["%s"]
					flavor = {
						cpu = %s
						ram = %s
					}
					storage = {
						class = "%s"
						size = %s
					}
					version = "%s"
					options = {
						edition = "%s"
						retention_days = %s
					}
					backup_schedule = "%s"
				}
				`,
		testutil.SQLServerFlexProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["acl"],
		instanceResource["flavor_cpu"],
		instanceResource["flavor_ram"],
		instanceResource["storage_class"],
		instanceResource["storage_size"],
		instanceResource["version"],
		instanceResource["options_edition"],
		instanceResource["options_retention_days"],
		backupSchedule,
	)
}

func TestAccSQLServerFlexResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccChecksqlserverflexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: configResources(instanceResource["backup_schedule"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.description", instanceResource["flavor_description"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "replicas", instanceResource["replicas"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.class", instanceResource["storage_class"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.size", instanceResource["storage_size"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "options.edition", instanceResource["options_edition"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "options.retention_days", instanceResource["options_retention_days"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "backup_schedule", instanceResource["backup_schedule"]),
				),
			},
			// data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_sqlserverflex_instance" "instance" {
						project_id     = stackit_sqlserverflex_instance.instance.project_id
						instance_id    = stackit_sqlserverflex_instance.instance.instance_id
					}
					`,
					configResources(instanceResource["backup_schedule"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_instance.instance", "project_id",
						"stackit_sqlserverflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_instance.instance", "instance_id",
						"stackit_sqlserverflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "flavor.id", instanceResource["flavor_id"]),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "flavor.description", instanceResource["flavor_description"]),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "replicas", instanceResource["replicas"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "options.edition", instanceResource["options_edition"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "options.retention_days", instanceResource["options_retention_days"]),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "backup_schedule", instanceResource["backup_schedule"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_sqlserverflex_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sqlserverflex_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sqlserverflex_instance.instance")
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
					if s[0].Attributes["backup_schedule"] != instanceResource["backup_schedule"] {
						return fmt.Errorf("expected backup_schedule %s, got %s", instanceResource["backup_schedule"], s[0].Attributes["backup_schedule"])
					}
					return nil
				},
			},
			// Update
			{
				Config: configResources(instanceResource["backup_schedule_updated"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.0", instanceResource["acl"]),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.cpu", instanceResource["flavor_cpu"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.ram", instanceResource["flavor_ram"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "replicas", instanceResource["replicas"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.class", instanceResource["storage_class"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.size", instanceResource["storage_size"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "options.edition", instanceResource["options_edition"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "options.retention_days", instanceResource["options_retention_days"]),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "backup_schedule", instanceResource["backup_schedule_updated"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccChecksqlserverflexDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *sqlserverflex.APIClient
	var err error
	if testutil.SQLServerFlexCustomEndpoint == "" {
		client, err = sqlserverflex.NewAPIClient()
	} else {
		client, err = sqlserverflex.NewAPIClient(
			config.WithEndpoint(testutil.SQLServerFlexCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_sqlserverflex_instance" {
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