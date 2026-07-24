package postgresflex

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	retentionDaysDefault = "32" // default value during deprecation period
)

var (
	// Instance

	//go:embed testdata/resource-instance-min.tf
	resourceInstanceMinConfig string

	//go:embed testdata/resource-instance-max.tf
	resourceInstanceMaxConfig string

	// Database

	//go:embed testdata/resource-database-min.tf
	resourceDatabaseMinConfig string

	// User

	//go:embed testdata/resource-user-min.tf
	resourceUserMinConfig string
)

// Instance - MIN
var testConfigInstanceVarsMin = config.Variables{
	"project_id":       config.StringVariable(testutil.ProjectId),
	"name":             config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"acl":              config.StringVariable("192.168.0.0/24"),
	"backup_schedule":  config.StringVariable("0 16 * * *"),
	"flavor_id":        config.StringVariable("4.8-replica"),
	"storage_class":    config.StringVariable("premium-perf2-stackit"),
	"storage_size":     config.IntegerVariable(5),
	"instance_version": config.StringVariable("16"),
	// Only used for the checks
	"replicas": config.IntegerVariable(3),
}

var testConfigInstanceVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigInstanceVarsMin)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf(
		"%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"]),
	))
	updatedConfig["acl"] = config.StringVariable("192.160.2.0/24")
	updatedConfig["backup_schedule"] = config.StringVariable("0 10 * * *")
	updatedConfig["flavor_id"] = config.StringVariable("4.8")
	updatedConfig["storage_size"] = config.IntegerVariable(10)
	updatedConfig["instance_version"] = config.StringVariable("17")
	return updatedConfig
}()

// Instance - MAX
var testConfigInstanceVarsMax = config.Variables{
	"project_id":       config.StringVariable(testutil.ProjectId),
	"name":             config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"acl":              config.StringVariable("192.168.0.0/24"),
	"access_scope":     config.StringVariable(string(postgresflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
	"backup_schedule":  config.StringVariable("0 16 * * *"),
	"flavor_id":        config.StringVariable("4.8-replica"),
	"flavor_cpu":       config.IntegerVariable(4),
	"flavor_ram":       config.IntegerVariable(8),
	"replicas":         config.IntegerVariable(3),
	"storage_class":    config.StringVariable("premium-perf2-stackit"),
	"storage_size":     config.IntegerVariable(5),
	"instance_version": config.StringVariable("16"),
	"retention_days":   config.IntegerVariable(40),
	"region":           config.StringVariable(testutil.Region),
}

var testConfigInstanceVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigInstanceVarsMax)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf(
		"%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"]),
	))
	updatedConfig["acl"] = config.StringVariable("192.160.2.0/24")
	updatedConfig["backup_schedule"] = config.StringVariable("0 10 * * *")
	updatedConfig["flavor_id"] = config.StringVariable("4.8")
	updatedConfig["flavor_cpu"] = config.IntegerVariable(8)
	updatedConfig["flavor_ram"] = config.IntegerVariable(16)
	updatedConfig["replicas"] = config.IntegerVariable(1)
	updatedConfig["storage_size"] = config.IntegerVariable(11)
	updatedConfig["instance_version"] = config.StringVariable("17")
	updatedConfig["retention_days"] = config.IntegerVariable(32)
	return updatedConfig
}()

// Database - MIN
var testConfigDatabaseVarsMin = config.Variables{
	"project_id":       config.StringVariable(testutil.ProjectId),
	"instance_name":    config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"username":         config.StringVariable(fmt.Sprintf("tf_db_acc_%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"acl":              config.StringVariable("192.168.0.0/24"),
	"backup_schedule":  config.StringVariable("0 16 * * *"),
	"flavor_id":        config.StringVariable("4.8-replica"),
	"storage_class":    config.StringVariable("premium-perf2-stackit"),
	"storage_size":     config.IntegerVariable(5),
	"instance_version": config.StringVariable("16"),
	"roles":            config.StringVariable("login"),

	"database_name": config.StringVariable("acc_test"),
}

var testConfigDatabaseVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigDatabaseVarsMin)
	updatedConfig["username"] = config.StringVariable(fmt.Sprintf(
		"%s_updated", testutil.ConvertConfigVariable(updatedConfig["username"]),
	))
	updatedConfig["roles"] = config.StringVariable("createdb")
	updatedConfig["database_name"] = config.StringVariable("acc_test_updated")
	return updatedConfig
}()

// User - MIN
var testConfigUserVarsMin = config.Variables{
	"project_id":       config.StringVariable(testutil.ProjectId),
	"name":             config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"acl":              config.StringVariable("192.168.0.0/24"),
	"backup_schedule":  config.StringVariable("0 16 * * *"),
	"flavor_id":        config.StringVariable("4.8-replica"),
	"storage_class":    config.StringVariable("premium-perf2-stackit"),
	"storage_size":     config.IntegerVariable(5),
	"instance_version": config.StringVariable("16"),

	"username": config.StringVariable("acc_test"),
	"roles":    config.StringVariable("login"),
}

var testConfigUserVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigUserVarsMin)
	updatedConfig["username"] = config.StringVariable("acc_test_updated")
	updatedConfig["roles"] = config.StringVariable("createdb")
	return updatedConfig
}()

func TestAccPostgresFlexInstanceMin(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigInstanceVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceInstanceMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["acl"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["acl"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "network.access_scope"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["backup_schedule"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "flavor_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["flavor_id"])),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "connection_info.write.port"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "replicas"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "retention_days", retentionDaysDefault),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["storage_class"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["storage_size"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["instance_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "region", testutil.Region),
				),
			},
			// data source
			{
				ConfigVariables: testConfigInstanceVarsMin,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionNoop),
					},
				},
				Config: fmt.Sprintf(`
					%s

					%s
					data "stackit_postgresflex_instance" "instance" {
						project_id     = stackit_postgresflex_instance.instance.project_id
						instance_id    = stackit_postgresflex_instance.instance.instance_id
					}
					`,
					testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceInstanceMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["acl"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["acl"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "network.access_scope"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["backup_schedule"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "flavor_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["flavor_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "flavor.cpu"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "connection_info.write.port"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["replicas"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "retention_days", retentionDaysDefault),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["storage_class"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["storage_size"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsMin["instance_version"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "region", testutil.Region),
				),
			},
			// Import
			{
				ConfigVariables: testConfigInstanceVarsMin,
				ResourceName:    "stackit_postgresflex_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_postgresflex_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_postgresflex_instance.instance")
					}

					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					return fmt.Sprintf("%s,%s,%s", projectId, region, instanceId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"flavor", "replicas"},
			},
			// Update
			{
				ConfigVariables: testConfigInstanceVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceInstanceMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMinUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsMinUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMinUpdated["acl"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMinUpdated["acl"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "network.access_scope"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsMinUpdated["backup_schedule"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "flavor_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMinUpdated["flavor_id"])),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "connection_info.write.port"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "replicas"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "retention_days", retentionDaysDefault),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsMinUpdated["storage_class"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsMinUpdated["storage_size"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsMinUpdated["instance_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "region", testutil.Region),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccPostgresFlexInstanceMax(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigInstanceVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceInstanceMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.with_flavor_id", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.with_flavor", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance with flavor id
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor_id", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["acl"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["acl"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor_id", "network.access_scope"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["backup_schedule"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "flavor_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["flavor_id"])),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "flavor.ram"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor_id", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor_id", "connection_info.write.port"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "replicas"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "retention_days", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["retention_days"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["storage_class"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["storage_size"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["instance_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "region", testutil.Region),

					// Instance with flavor
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["acl"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["acl"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "network.access_scope"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["backup_schedule"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "flavor_id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "flavor.id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "flavor.cpu", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["flavor_cpu"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "flavor.ram", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["flavor_ram"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "connection_info.write.port"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "replicas", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["replicas"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "retention_days", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["retention_days"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["storage_class"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["storage_size"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["instance_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "region", testutil.Region),
				),
			},
			// data source
			{
				ConfigVariables: testConfigInstanceVarsMax,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.with_flavor_id", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.with_flavor", plancheck.ResourceActionNoop),
					},
				},
				Config: fmt.Sprintf(`
					%s

					%s
					data "stackit_postgresflex_instance" "with_flavor_id" {
						project_id     = stackit_postgresflex_instance.with_flavor_id.project_id
						instance_id    = stackit_postgresflex_instance.with_flavor_id.instance_id
					}

					data "stackit_postgresflex_instance" "with_flavor" {
						project_id     = stackit_postgresflex_instance.with_flavor.project_id
						instance_id    = stackit_postgresflex_instance.with_flavor.instance_id
					}
					`,
					testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceInstanceMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance with flavor id
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor_id", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["acl"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["acl"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor_id", "network.access_scope"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["backup_schedule"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "flavor_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["flavor_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor_id", "flavor.id"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor_id", "flavor.description"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor_id", "flavor.cpu"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor_id", "flavor.ram"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor_id", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor_id", "connection_info.write.port"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor_id", "replicas"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "retention_days", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["retention_days"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["storage_class"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["storage_size"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["instance_version"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor_id", "region", testutil.Region),

					// Instance with flavor
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["acl"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["acl"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "network.access_scope"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["backup_schedule"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "flavor_id"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "flavor.id"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "flavor.description"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "flavor.cpu"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "flavor.ram"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "connection_info.write.port"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.with_flavor", "replicas"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "retention_days", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["retention_days"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["storage_class"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["storage_size"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsMax["instance_version"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.with_flavor", "region", testutil.Region),
				),
			},
			// Import with flavor id
			{
				ConfigVariables: testConfigInstanceVarsMax,
				ResourceName:    "stackit_postgresflex_instance.with_flavor_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_postgresflex_instance.with_flavor_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_postgresflex_instance.with_flavor_id")
					}

					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					return fmt.Sprintf("%s,%s,%s", projectId, region, instanceId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"flavor", "replicas"},
			},
			// Import with flavor
			{
				ConfigVariables: testConfigInstanceVarsMax,
				ResourceName:    "stackit_postgresflex_instance.with_flavor",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_postgresflex_instance.with_flavor"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_postgresflex_instance.with_flavor")
					}

					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					return fmt.Sprintf("%s,%s,%s", projectId, region, instanceId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"flavor", "replicas"},
			},
			// Update
			{
				ConfigVariables: testConfigInstanceVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceInstanceMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.with_flavor_id", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.with_flavor", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance with flavor id
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor_id", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["acl"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["acl"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor_id", "network.access_scope"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["backup_schedule"])),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "flavor.ram"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor_id", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor_id", "connection_info.write.port"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor_id", "replicas"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "retention_days", retentionDaysDefault),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["storage_class"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["storage_size"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["instance_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor_id", "region", testutil.Region),

					// Instance with flavor
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["acl"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["acl"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "network.access_scope"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["backup_schedule"])),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor", "flavor.ram"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.with_flavor", "connection_info.write.port"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.with_flavor", "replicas"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "retention_days", retentionDaysDefault),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["storage_class"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["storage_size"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsMaxUpdated["instance_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.with_flavor", "region", testutil.Region),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccPostgresFlexDatabaseMin(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigDatabaseVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceDatabaseMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_postgresflex_user.user", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_postgresflex_database.database", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Database
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "project_id",
						"stackit_postgresflex_database.database", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "instance_id",
						"stackit_postgresflex_database.database", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_user.user", "username",
						"stackit_postgresflex_database.database", "owner",
					),
					resource.TestCheckResourceAttr("stackit_postgresflex_database.database", "name", testutil.ConvertConfigVariable(testConfigDatabaseVarsMin["database_name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_database.database", "owner", testutil.ConvertConfigVariable(testConfigDatabaseVarsMin["username"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_database.database", "database_id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_database.database", "id"),
				),
			},
			// data source
			{
				ConfigVariables: testConfigDatabaseVarsMin,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_postgresflex_user.user", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_postgresflex_database.database", plancheck.ResourceActionNoop),
					},
				},
				Config: fmt.Sprintf(`
					%s

					%s
					data "stackit_postgresflex_database" "database" {
						project_id  = stackit_postgresflex_database.database.project_id
						instance_id = stackit_postgresflex_database.database.instance_id
						database_id = stackit_postgresflex_database.database.database_id
					}
					`,
					testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceDatabaseMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Database
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "project_id",
						"data.stackit_postgresflex_database.database", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "instance_id",
						"data.stackit_postgresflex_database.database", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_user.user", "username",
						"data.stackit_postgresflex_database.database", "owner",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_database.database", "database_id",
						"data.stackit_postgresflex_database.database", "database_id",
					),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_database.database", "owner", testutil.ConvertConfigVariable(testConfigDatabaseVarsMin["username"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_database.database", "name", testutil.ConvertConfigVariable(testConfigDatabaseVarsMin["database_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_database.database", "id"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigDatabaseVarsMin,
				ResourceName:    "stackit_postgresflex_database.database",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_postgresflex_database.database"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_postgresflex_database.database")
					}

					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					databaseId, ok := r.Primary.Attributes["database_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute database_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", projectId, region, instanceId, databaseId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigDatabaseVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceDatabaseMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_postgresflex_user.user", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_postgresflex_database.database", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Database
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "project_id",
						"stackit_postgresflex_database.database", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "instance_id",
						"stackit_postgresflex_database.database", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_user.user", "username",
						"stackit_postgresflex_database.database", "owner",
					),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_database.database", "database_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_database.database", "name", testutil.ConvertConfigVariable(testConfigDatabaseVarsMinUpdated["database_name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_database.database", "owner", testutil.ConvertConfigVariable(testConfigDatabaseVarsMinUpdated["username"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_database.database", "id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccPostgresFlexUserMin(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigUserVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceUserMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_postgresflex_user.user", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// User
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "project_id",
						"stackit_postgresflex_user.user", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "instance_id",
						"stackit_postgresflex_user.user", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_postgresflex_user.user", "username", testutil.ConvertConfigVariable(testConfigUserVarsMin["username"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_user.user", "roles.0", testutil.ConvertConfigVariable(testConfigUserVarsMin["roles"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "password"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "port"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "uri"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "user_id"),
				),
			},
			// data source
			{
				ConfigVariables: testConfigUserVarsMin,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_postgresflex_user.user", plancheck.ResourceActionNoop),
					},
				},
				Config: fmt.Sprintf(`
					%s

					%s
					data "stackit_postgresflex_user" "user" {
						project_id  = stackit_postgresflex_user.user.project_id
						instance_id = stackit_postgresflex_user.user.instance_id
						user_id     = stackit_postgresflex_user.user.user_id
					}
					`,
					testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceUserMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// User
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "project_id",
						"data.stackit_postgresflex_user.user", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "instance_id",
						"data.stackit_postgresflex_user.user", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_user.user", "user_id",
						"data.stackit_postgresflex_user.user", "user_id",
					),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_user.user", "username", testutil.ConvertConfigVariable(testConfigUserVarsMin["username"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_user.user", "roles.0", testutil.ConvertConfigVariable(testConfigUserVarsMin["roles"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_user.user", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_user.user", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_user.user", "port"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigUserVarsMin,
				ResourceName:    "stackit_postgresflex_user.user",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_postgresflex_user.user"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_postgresflex_user.user")
					}

					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					userId, ok := r.Primary.Attributes["user_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute user_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", projectId, region, instanceId, userId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "uri"},
			},
			// Update
			{
				ConfigVariables: testConfigUserVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceUserMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_postgresflex_user.user", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// User
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "project_id",
						"stackit_postgresflex_user.user", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_postgresflex_instance.instance", "instance_id",
						"stackit_postgresflex_user.user", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_postgresflex_user.user", "username", testutil.ConvertConfigVariable(testConfigUserVarsMinUpdated["username"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_user.user", "roles.0", testutil.ConvertConfigVariable(testConfigUserVarsMinUpdated["roles"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "password"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "port"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "uri"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_user.user", "user_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testCheckDestroy(s *terraform.State) error {
	checkDestroyFuncs := []resource.TestCheckFunc{
		testDatabaseDestroy,
		testUserDestroy,
		testInstanceDestroy,
	}

	var errs []error
	for _, checkDestroyFunc := range checkDestroyFuncs {
		err := checkDestroyFunc(s)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func testUserDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := postgresflex.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.PostgresFlexCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type postgresUser struct {
		projectId  string
		region     string
		instanceId string
		userId     int64
	}

	var errs []error
	usersToDestroy := []postgresUser{}
	for _, r := range s.RootModule().Resources {
		if r.Type != "stackit_postgresflex_user" {
			continue
		}
		projectId, ok := r.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", r.Primary))
			continue
		}
		region, ok := r.Primary.Attributes["region"]
		if !ok {
			errs = append(errs, fmt.Errorf("no region found in %s", r.Primary))
			continue
		}
		instanceId, ok := r.Primary.Attributes["instance_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no instance_id found in %s", r.Primary))
			continue
		}
		userIdStr, ok := r.Primary.Attributes["user_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no user_id found in %s", r.Primary))
			continue
		}
		userId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("parsing user id: %w", err))
			continue
		}

		usersToDestroy = append(usersToDestroy, postgresUser{
			projectId:  projectId,
			region:     region,
			instanceId: instanceId,
			userId:     userId,
		})
	}

	for _, user := range usersToDestroy {
		_, err = client.DefaultAPI.GetUser(ctx, user.projectId, user.region, user.instanceId, user.userId).Execute()
		if err == nil {
			retryConfig := utils.RetryConfig{
				Attempts: 15,
				Backoff: func(attempt int) time.Duration {
					// Wait for every attempt 5 seconds longer. 5s, 10s, 15s and so on
					return time.Duration(attempt*5) * time.Second
				},
				RetryStatusCodes: []int{http.StatusLocked},
			}
			err = utils.RetryRequestWithoutResponse(
				ctx,
				client.DefaultAPI.DeleteUser(ctx, user.projectId, user.region, user.instanceId, user.userId).Execute,
				retryConfig,
			)
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting user with ID %d in project %q, region %q, instance %q: %w", user.userId, user.projectId, user.region, user.instanceId, err))
				continue
			}
			_, err = wait.DeleteUserWaitHandler(ctx, client.DefaultAPI, user.projectId, user.region, user.instanceId, user.userId).WaitWithContext(ctx)
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting user with ID %d in project %q, region %q, instance %q during CheckDestroy: waiting for deletion %w", user.userId, user.projectId, user.region, user.instanceId, err))
			}
		}
	}
	return errors.Join(errs...)
}

func testDatabaseDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := postgresflex.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.PostgresFlexCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type postgresDatabase struct {
		projectId  string
		region     string
		instanceId string
		databaseId int64
	}

	var errs []error
	databasesToDestroy := []postgresDatabase{}
	for _, r := range s.RootModule().Resources {
		if r.Type != "stackit_postgresflex_database" {
			continue
		}
		projectId, ok := r.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", r.Primary))
			continue
		}
		region, ok := r.Primary.Attributes["region"]
		if !ok {
			errs = append(errs, fmt.Errorf("no region found in %s", r.Primary))
			continue
		}
		instanceId, ok := r.Primary.Attributes["instance_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no instance_id found in %s", r.Primary))
			continue
		}
		databaseIdStr, ok := r.Primary.Attributes["database_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no database_id found in %s", r.Primary))
			continue
		}
		databaseId, err := strconv.ParseInt(databaseIdStr, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("parsing database id: %w", err))
			continue
		}
		databasesToDestroy = append(databasesToDestroy, postgresDatabase{
			projectId:  projectId,
			region:     region,
			instanceId: instanceId,
			databaseId: databaseId,
		})
	}

	for _, db := range databasesToDestroy {
		_, err = client.DefaultAPI.GetDatabase(ctx, db.projectId, db.region, db.instanceId, db.databaseId).Execute()
		if err == nil {
			retryConfig := utils.RetryConfig{
				Attempts: 15,
				Backoff: func(attempt int) time.Duration {
					// Wait for every attempt 5 seconds longer. 5s, 10s, 15s and so on
					return time.Duration(attempt*5) * time.Second
				},
				RetryStatusCodes: []int{http.StatusLocked},
			}
			err = utils.RetryRequestWithoutResponse(
				ctx,
				client.DefaultAPI.DeleteDatabase(ctx, db.projectId, db.region, db.instanceId, db.databaseId).Execute,
				retryConfig,
			)
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting database with ID %d in project %q, region %q, instance id %q: %w", db.databaseId, db.projectId, db.region, db.instanceId, err))
				continue
			}
		}
	}
	return errors.Join(errs...)
}

func testInstanceDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := postgresflex.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.PostgresFlexCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type postgresInstance struct {
		projectId  string
		region     string
		instanceId string
	}

	var errs []error
	instancesToDestroy := []postgresInstance{}
	for _, r := range s.RootModule().Resources {
		if r.Type != "stackit_postgresflex_instance" {
			continue
		}
		projectId, ok := r.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", r.Primary))
			continue
		}
		region, ok := r.Primary.Attributes["region"]
		if !ok {
			errs = append(errs, fmt.Errorf("no region found in %s", r.Primary))
			continue
		}
		instanceId, ok := r.Primary.Attributes["instance_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no instance_id found in %s", r.Primary))
			continue
		}
		instancesToDestroy = append(instancesToDestroy, postgresInstance{
			projectId:  projectId,
			region:     region,
			instanceId: instanceId,
		})
	}

	for _, inst := range instancesToDestroy {
		_, err = client.DefaultAPI.GetInstance(ctx, inst.projectId, inst.region, inst.instanceId).Execute()
		if err == nil {
			retryConfig := utils.RetryConfig{
				Attempts: 15,
				Backoff: func(attempt int) time.Duration {
					// Wait for every attempt 5 seconds longer. 5s, 10s, 15s and so on
					return time.Duration(attempt*5) * time.Second
				},
				RetryStatusCodes: []int{http.StatusLocked},
			}
			err = utils.RetryRequestWithoutResponse(
				ctx,
				client.DefaultAPI.DeleteInstance(ctx, inst.projectId, inst.region, inst.instanceId).Execute,
				retryConfig,
			)
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting instance with ID %q in project %q, region %q: %w", inst.instanceId, inst.projectId, inst.region, err))
				continue
			}
			_, err = wait.DeleteInstanceWaitHandler(ctx, client.DefaultAPI, inst.projectId, inst.region, inst.instanceId).WaitWithContext(ctx)
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting instance with ID %q in project %q, region %q during CheckDestroy: waiting for deletion %w", inst.instanceId, inst.projectId, inst.region, err))
			}
		}
	}
	return errors.Join(errs...)
}
