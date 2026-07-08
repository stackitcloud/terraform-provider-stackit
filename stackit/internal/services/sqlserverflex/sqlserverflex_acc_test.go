package sqlserverflex_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"
	"github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-max.tf
	resourceMaxConfig string
	//go:embed testdata/resource-min.tf
	resourceMinConfig string
)
var testConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"replicas":   config.IntegerVariable(1),
	"flavor_id":  config.StringVariable("4.16-Single"),
	"username":   config.StringVariable(fmt.Sprintf("tf-acc-user-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlpha))),
	"role":       config.StringVariable("##STACKIT_LoginManager##"),
}

var testConfigVarsMax = config.Variables{
	"project_id":      config.StringVariable(testutil.ProjectId),
	"name":            config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"acl1":            config.StringVariable("192.168.0.0/16"),
	"storage_class":   config.StringVariable("premium-perf2-stackit"),
	"storage_size":    config.IntegerVariable(40),
	"server_version":  config.StringVariable("2022"),
	"replicas":        config.IntegerVariable(1),
	"access_scope":    config.StringVariable(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
	"retention_days":  config.IntegerVariable(32),
	"flavor_id":       config.StringVariable("4.16-Single"),
	"backup_schedule": config.StringVariable("0 6 * * *"),
	"username":        config.StringVariable(fmt.Sprintf("tf-acc-user-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlpha))),
	"role":            config.StringVariable("##STACKIT_LoginManager##"),
	"region":          config.StringVariable(testutil.Region),
}

func configVarsMinUpdated() config.Variables {
	temp := maps.Clone(testConfigVarsMin)
	temp["name"] = config.StringVariable(testutil.ConvertConfigVariable(temp["name"]) + "changed")
	return temp
}

func configVarsMaxUpdated() config.Variables {
	temp := maps.Clone(testConfigVarsMax)
	temp["backup_schedule"] = config.StringVariable("0 12 * * *")
	temp["acl1"] = config.StringVariable("192.168.2.0/16")
	temp["retention_days"] = config.IntegerVariable(40)
	return temp
}

func TestAccSQLServerFlexMinResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccChecksqlserverflexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sqlserverflex_instance.instance", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_sqlserverflex_user.user", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "flavor_id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMin["replicas"])),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.ram"),
					// User
					resource.TestCheckResourceAttrPair(
						"stackit_sqlserverflex_user.user", "project_id",
						"stackit_sqlserverflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_sqlserverflex_user.user", "instance_id",
						"stackit_sqlserverflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_user.user", "user_id"),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_user.user", "password"),
				),
			},
			// data source
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMinConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sqlserverflex_instance.instance", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_sqlserverflex_user.user", plancheck.ResourceActionNoop),
					},
				},
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_instance.instance", "project_id",
						"stackit_sqlserverflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_instance.instance", "instance_id",
						"stackit_sqlserverflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_user.user", "instance_id",
						"stackit_sqlserverflex_user.user", "instance_id",
					),

					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor_id"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "flavor.id", testutil.ConvertConfigVariable(testConfigVarsMin["flavor_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.cpu"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.ram"),

					// User data
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_user.user", "user_id"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "username", testutil.ConvertConfigVariable(testConfigVarsMin["username"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "roles.0", testutil.ConvertConfigVariable(testConfigVarsMax["role"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_sqlserverflex_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sqlserverflex_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sqlserverflex_instance.instance")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"backup_schedule",
					// Will be added during the import, because it's unknown which attribute defined the flavor
					"flavor.cpu",
					"flavor.description",
					"flavor.id",
					"flavor.ram",
				},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					if len(s) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(s))
					}
					return nil
				},
			},
			{
				ResourceName:    "stackit_sqlserverflex_user.user",
				ConfigVariables: testConfigVarsMin,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sqlserverflex_user.user"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sqlserverflex_user.user")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					userId, ok := r.Primary.Attributes["user_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute user_id")
					}

					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, userId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMinConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sqlserverflex_instance.instance", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_sqlserverflex_user.user", plancheck.ResourceActionNoop),
					},
				},
				ConfigVariables: configVarsMinUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "project_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "name", testutil.ConvertConfigVariable(configVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "flavor_id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.ram"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccSQLServerFlexMaxResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccChecksqlserverflexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMaxConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sqlserverflex_instance.instance", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_sqlserverflex_user.user", plancheck.ResourceActionCreate),
					},
				},
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "network.acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl1"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "network.access_scope", testutil.ConvertConfigVariable(testConfigVarsMax["access_scope"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "flavor_id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMax["replicas"])),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigVarsMax["storage_class"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigVarsMax["storage_size"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["server_version"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["retention_days"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigVarsMax["backup_schedule"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "region", testutil.Region),
					// User
					resource.TestCheckResourceAttrPair(
						"stackit_sqlserverflex_user.user", "project_id",
						"stackit_sqlserverflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_sqlserverflex_user.user", "instance_id",
						"stackit_sqlserverflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_user.user", "user_id"),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_user.user", "password"),
				),
			},
			// data source
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_instance.instance", "project_id",
						"stackit_sqlserverflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_instance.instance", "instance_id",
						"stackit_sqlserverflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_user.user", "instance_id",
						"stackit_sqlserverflex_user.user", "instance_id",
					),

					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl1"])),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor_id"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.cpu"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMax["replicas"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["retention_days"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigVarsMax["backup_schedule"])),

					// User data
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_user.user", "user_id"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "username", testutil.ConvertConfigVariable(testConfigVarsMax["username"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "roles.0", testutil.ConvertConfigVariable(testConfigVarsMax["role"])),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_user.user", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_user.user", "port"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_sqlserverflex_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sqlserverflex_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sqlserverflex_instance.instance")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"backup_schedule",
					// Will be added during the import, because it's unknown which attribute defined the flavor
					"flavor.cpu",
					"flavor.description",
					"flavor.id",
					"flavor.ram",
				},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					if len(s) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(s))
					}
					if s[0].Attributes["backup_schedule"] != testutil.ConvertConfigVariable(testConfigVarsMax["backup_schedule"]) {
						return fmt.Errorf("expected backup_schedule %s, got %s", testConfigVarsMax["backup_schedule"], s[0].Attributes["backup_schedule"])
					}
					return nil
				},
			},
			{
				ResourceName:    "stackit_sqlserverflex_user.user",
				ConfigVariables: testConfigVarsMax,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sqlserverflex_user.user"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sqlserverflex_user.user")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					userId, ok := r.Primary.Attributes["user_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute user_id")
					}

					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, userId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: configVarsMaxUpdated(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sqlserverflex_instance.instance", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_sqlserverflex_user.user", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["acl1"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "flavor_id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "replicas", testutil.ConvertConfigVariable(configVarsMaxUpdated()["replicas"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(configVarsMaxUpdated()["storage_class"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(configVarsMaxUpdated()["storage_size"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "version", testutil.ConvertConfigVariable(configVarsMaxUpdated()["server_version"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "retention_days", testutil.ConvertConfigVariable(configVarsMaxUpdated()["retention_days"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(configVarsMaxUpdated()["backup_schedule"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccChecksqlserverflexDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := sqlserverflex.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.SQLServerFlexCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_sqlserverflex_instance" {
			continue
		}
		// instance terraform ID: = "[project_id],[region],[instance_id]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[2]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.DefaultAPI.ListInstances(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	items := instancesResp.Instances
	for i := range items {
		if items[i].Id == "" {
			continue
		}
		if utils.Contains(instancesToDestroy, items[i].Id) {
			err := client.DefaultAPI.DeleteInstance(ctx, testutil.ProjectId, items[i].Id, testutil.Region).Execute()
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: %w", items[i].Id, err)
			}
			_, err = wait.DeleteInstanceWaitHandler(ctx, client.DefaultAPI, testutil.ProjectId, items[i].Id, testutil.Region).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: waiting for deletion %w", items[i].Id, err)
			}
		}
	}
	return nil
}
