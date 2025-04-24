package mongodbflex_test

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testfiles/resource-min.tf
var resourceMinConfig string

//go:embed testfiles/resource-max.tf
var resourceMaxConfig string

var testConfigVarsMin = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"name":                 config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"acl":                  config.StringVariable("192.168.0.0/16"),
	"flavor_cpu":           config.StringVariable("2"),
	"flavor_ram":           config.StringVariable("4"),
	"flavor_description":   config.StringVariable("Small, Compute optimized"),
	"replicas":             config.StringVariable("3"),
	"storage_class":        config.StringVariable("premium-perf2-mongodb"),
	"storage_size":         config.StringVariable("10"),
	"version_db":           config.StringVariable("6.0"),
	"options_type":         config.StringVariable("Replica"),
	"flavor_id":            config.StringVariable("2.4"),
	"backup_schedule":      config.StringVariable("00 6 * * *"),
	"backup_schedule_read": config.StringVariable("0 6 * * *"),
	"role":                 config.StringVariable("read"),
	"database":             config.StringVariable("default"),
}

var testConfigVarsMax = config.Variables{
	"project_id":                        config.StringVariable(testutil.ProjectId),
	"name":                              config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"acl":                               config.StringVariable("192.168.0.0/16"),
	"flavor_cpu":                        config.StringVariable("2"),
	"flavor_ram":                        config.StringVariable("4"),
	"flavor_description":                config.StringVariable("Small, Compute optimized"),
	"replicas":                          config.StringVariable("3"),
	"storage_class":                     config.StringVariable("premium-perf2-mongodb"),
	"storage_size":                      config.StringVariable("10"),
	"version_db":                        config.StringVariable("6.0"),
	"options_type":                      config.StringVariable("Replica"),
	"flavor_id":                         config.StringVariable("2.4"),
	"backup_schedule":                   config.StringVariable("00 6 * * *"),
	"backup_schedule_read":              config.StringVariable("0 6 * * *"),
	"username":                          config.StringVariable(fmt.Sprintf("tf-acc-user-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlpha))),
	"role":                              config.StringVariable("read"),
	"database":                          config.StringVariable("default"),
	"snapshot_retention_days":           config.StringVariable("4"),
	"daily_snapshot_retention_days":     config.StringVariable("1"),
	"weekly_snapshot_retention_weeks":   config.StringVariable("7"),
	"monthly_snapshot_retention_months": config.StringVariable("12"),
	"point_in_time_window_hours":        config.StringVariable("5"),
}

func configVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(tempConfig, testConfigVarsMin)
	tempConfig["version_db"] = config.StringVariable("7.0")
	tempConfig["backup_schedule"] = config.StringVariable("00 12 * * *")
	return tempConfig
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMax))
	maps.Copy(tempConfig, testConfigVarsMax)
	tempConfig["version_db"] = config.StringVariable("8.0")
	tempConfig["backup_schedule"] = config.StringVariable("00 14 * * *")
	return tempConfig
}

// minimum configuration
func TestAccMongoDBFlexFlexResourceMin(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMongoDBFlexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.MongoDBFlexProviderConfig() + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMin["acl"])),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.cpu", testutil.ConvertConfigVariable(testConfigVarsMin["flavor_cpu"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.ram", testutil.ConvertConfigVariable(testConfigVarsMin["flavor_ram"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMin["replicas"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigVarsMin["storage_class"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigVarsMin["storage_size"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMin["version_db"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.type", testutil.ConvertConfigVariable(testConfigVarsMin["options_type"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigVarsMin["backup_schedule"])),

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
					resource.TestCheckResourceAttr("stackit_mongodbflex_user.user", "database", testutil.ConvertConfigVariable(testConfigVarsMin["database"])),
				),
			},
			// data source
			{
				ConfigVariables: testConfigVarsMin,
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
					testutil.MongoDBFlexProviderConfig()+resourceMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
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
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMin["acl"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.id", testutil.ConvertConfigVariable(testConfigVarsMin["flavor_id"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.description", testutil.ConvertConfigVariable(testConfigVarsMin["flavor_description"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.cpu", testutil.ConvertConfigVariable(testConfigVarsMin["flavor_cpu"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.ram", testutil.ConvertConfigVariable(testConfigVarsMin["flavor_ram"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMin["replicas"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "options.type", testutil.ConvertConfigVariable(testConfigVarsMin["options_type"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigVarsMin["backup_schedule_read"])),

					// User data
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_mongodbflex_user.user", "user_id"),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "database", testutil.ConvertConfigVariable(testConfigVarsMin["database"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "roles.0", testutil.ConvertConfigVariable(testConfigVarsMin["role"])),
					resource.TestCheckResourceAttrSet("data.stackit_mongodbflex_user.user", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_mongodbflex_user.user", "port"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
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
					if s[0].Attributes["backup_schedule"] != testutil.ConvertConfigVariable(testConfigVarsMin["backup_schedule_read"]) {
						return fmt.Errorf("expected backup_schedule %s, got %s", testutil.ConvertConfigVariable(testConfigVarsMin["backup_schedule_read"]), s[0].Attributes["backup_schedule"])
					}
					return nil
				},
			},
			{
				ConfigVariables: testConfigVarsMin,
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
				ImportStateVerifyIgnore: []string{"password", "uri"},
			},
			// Update
			{
				ConfigVariables: configVarsMinUpdated(),
				Config:          testutil.MongoDBFlexProviderConfig() + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMin["acl"])),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.cpu", testutil.ConvertConfigVariable(testConfigVarsMin["flavor_cpu"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.ram", testutil.ConvertConfigVariable(testConfigVarsMin["flavor_ram"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMin["replicas"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigVarsMin["storage_class"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigVarsMin["storage_size"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "version", testutil.ConvertConfigVariable(configVarsMinUpdated()["version_db"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.type", testutil.ConvertConfigVariable(testConfigVarsMin["options_type"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(configVarsMinUpdated()["backup_schedule"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

// maximum configuration
func TestAccMongoDBFlexFlexResourceMax(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMongoDBFlexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.MongoDBFlexProviderConfig() + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.cpu", testutil.ConvertConfigVariable(testConfigVarsMax["flavor_cpu"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.ram", testutil.ConvertConfigVariable(testConfigVarsMax["flavor_ram"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMax["replicas"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigVarsMax["storage_class"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigVarsMax["storage_size"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["version_db"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.type", testutil.ConvertConfigVariable(testConfigVarsMax["options_type"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigVarsMax["backup_schedule"])),

					// optional stuff
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.snapshot_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["snapshot_retention_days"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.daily_snapshot_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["daily_snapshot_retention_days"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.weekly_snapshot_retention_weeks", testutil.ConvertConfigVariable(testConfigVarsMax["weekly_snapshot_retention_weeks"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.monthly_snapshot_retention_months", testutil.ConvertConfigVariable(testConfigVarsMax["monthly_snapshot_retention_months"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.point_in_time_window_hours", testutil.ConvertConfigVariable(testConfigVarsMax["point_in_time_window_hours"])),

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
					resource.TestCheckResourceAttr("stackit_mongodbflex_user.user", "username", testutil.ConvertConfigVariable(testConfigVarsMax["username"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_user.user", "database", testutil.ConvertConfigVariable(testConfigVarsMax["database"])),
				),
			},
			// data source
			{
				ConfigVariables: testConfigVarsMax,
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
					testutil.MongoDBFlexProviderConfig()+resourceMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
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
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.id", testutil.ConvertConfigVariable(testConfigVarsMax["flavor_id"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.description", testutil.ConvertConfigVariable(testConfigVarsMax["flavor_description"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.cpu", testutil.ConvertConfigVariable(testConfigVarsMax["flavor_cpu"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "flavor.ram", testutil.ConvertConfigVariable(testConfigVarsMax["flavor_ram"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMax["replicas"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "options.type", testutil.ConvertConfigVariable(testConfigVarsMax["options_type"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigVarsMax["backup_schedule_read"])),

					// optional stuff
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.snapshot_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["snapshot_retention_days"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.daily_snapshot_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["daily_snapshot_retention_days"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.weekly_snapshot_retention_weeks", testutil.ConvertConfigVariable(testConfigVarsMax["weekly_snapshot_retention_weeks"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.monthly_snapshot_retention_months", testutil.ConvertConfigVariable(testConfigVarsMax["monthly_snapshot_retention_months"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.point_in_time_window_hours", testutil.ConvertConfigVariable(testConfigVarsMax["point_in_time_window_hours"])),

					// User data
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_mongodbflex_user.user", "user_id"),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "username", testutil.ConvertConfigVariable(testConfigVarsMax["username"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "database", testutil.ConvertConfigVariable(testConfigVarsMax["database"])),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_mongodbflex_user.user", "roles.0", testutil.ConvertConfigVariable(testConfigVarsMax["role"])),
					resource.TestCheckResourceAttrSet("data.stackit_mongodbflex_user.user", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_mongodbflex_user.user", "port"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
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
					if s[0].Attributes["backup_schedule"] != testutil.ConvertConfigVariable(testConfigVarsMax["backup_schedule_read"]) {
						return fmt.Errorf("expected backup_schedule %s, got %s", testutil.ConvertConfigVariable(testConfigVarsMax["backup_schedule_read"]), s[0].Attributes["backup_schedule"])
					}
					return nil
				},
			},
			{
				ConfigVariables: testConfigVarsMax,
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
				ImportStateVerifyIgnore: []string{"password", "uri"},
			},
			// Update
			{
				ConfigVariables: configVarsMaxUpdated(),
				Config:          testutil.MongoDBFlexProviderConfig() + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_mongodbflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.cpu", testutil.ConvertConfigVariable(testConfigVarsMax["flavor_cpu"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "flavor.ram", testutil.ConvertConfigVariable(testConfigVarsMax["flavor_ram"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMax["replicas"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigVarsMax["storage_class"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigVarsMax["storage_size"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "version", testutil.ConvertConfigVariable(configVarsMaxUpdated()["version_db"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.type", testutil.ConvertConfigVariable(testConfigVarsMax["options_type"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(configVarsMaxUpdated()["backup_schedule"])),

					// optional stuff
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.snapshot_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["snapshot_retention_days"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.daily_snapshot_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["daily_snapshot_retention_days"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.weekly_snapshot_retention_weeks", testutil.ConvertConfigVariable(testConfigVarsMax["weekly_snapshot_retention_weeks"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.monthly_snapshot_retention_months", testutil.ConvertConfigVariable(testConfigVarsMax["monthly_snapshot_retention_months"])),
					resource.TestCheckResourceAttr("stackit_mongodbflex_instance.instance", "options.point_in_time_window_hours", testutil.ConvertConfigVariable(testConfigVarsMax["point_in_time_window_hours"])),
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
		client, err = mongodbflex.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = mongodbflex.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.MongoDBFlexCustomEndpoint),
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
