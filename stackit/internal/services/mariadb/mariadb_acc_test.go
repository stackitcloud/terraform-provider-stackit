package mariadb_test

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
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/mariadb"
	"github.com/stackitcloud/stackit-sdk-go/services/mariadb/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testfiles/resource-min.tf
var resourceMinConfig string

//go:embed testfiles/resource-max.tf
var resourceMaxConfig string

var testConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"plan_name":  config.StringVariable("stackit-mariadb-1.4.10-single"),
	"db_version": config.StringVariable("10.6"),
}

var testConfigVarsMax = config.Variables{
	"project_id":                       config.StringVariable(testutil.ProjectId),
	"name":                             config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"plan_name":                        config.StringVariable("stackit-mariadb-1.4.10-single"),
	"db_version":                       config.StringVariable("10.11"),
	"observability_instance_plan_name": config.StringVariable("Observability-Monitoring-Basic-EU01"),
	"parameters_enable_monitoring":     config.BoolVariable(true),
	"parameters_graphite":              config.StringVariable(fmt.Sprintf("%s.graphite.stackit.cloud:2003", acctest.RandStringFromCharSet(7, acctest.CharSetAlpha))),
	"parameters_max_disk_threshold":    config.IntegerVariable(75),
	"parameters_metrics_frequency":     config.IntegerVariable(15),
	"parameters_metrics_prefix":        config.StringVariable("acc-test"),
	"parameters_sgw_acl":               config.StringVariable("192.168.2.0/24"),
	"parameters_syslog":                config.StringVariable("acc.test.log:514"),
}

func configVarsMaxUpdated() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigVarsMax {
		updatedConfig[k] = v
	}
	updatedConfig["parameters_max_disk_threshold"] = config.IntegerVariable(85)
	updatedConfig["parameters_metrics_frequency"] = config.IntegerVariable(10)
	updatedConfig["parameters_graphite"] = config.StringVariable("graphite.stackit.cloud:2003")
	updatedConfig["parameters_sgw_acl"] = config.StringVariable("192.168.1.0/24")
	updatedConfig["parameters_syslog"] = config.StringVariable("test.log:514")
	return updatedConfig
}

// minimum configuration
func TestAccMariaDbResourceMin(t *testing.T) {
	t.Logf("Maria test instance name: %s", testutil.ConvertConfigVariable(testConfigVarsMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMariaDBDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.MariaDBProviderConfig(), resourceMinConfig),
				Check: resource.ComposeTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMin["db_version"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "plan_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "image_url"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "dashboard_url"),

					// Credential
					resource.TestCheckResourceAttrPair(
						"stackit_mariadb_instance.instance", "project_id",
						"stackit_mariadb_credential.credential", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_mariadb_instance.instance", "instance_id",
						"stackit_mariadb_credential.credential", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "name"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "username"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
					%s

					%s

					data "stackit_mariadb_instance" "instance" {
						project_id = stackit_mariadb_instance.instance.project_id
						instance_id = stackit_mariadb_instance.instance.instance_id
					}

					data "stackit_mariadb_credential" "credential" {
						project_id = stackit_mariadb_credential.credential.project_id
						instance_id = stackit_mariadb_credential.credential.instance_id
					    credential_id = stackit_mariadb_credential.credential.credential_id
					}`, testutil.MariaDBProviderConfig(), resourceMinConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttrPair(
						"data.stackit_mariadb_instance.instance", "instance_id",
						"stackit_mariadb_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMin["db_version"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "plan_id"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "image_url"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "dashboard_url"),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_mariadb_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "name"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "username"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_mariadb_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_mariadb_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_mariadb_instance.instance")
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
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_mariadb_credential.credential",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_mariadb_credential.credential"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_mariadb_credential.credential")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					credentialId, ok := r.Primary.Attributes["credential_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute credential_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, credentialId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// In this minimal setup, it's not possible to perform an update
			// Deletion is done by the framework implicitly
		},
	})
}

// maximum configuration
func TestAccMariaDbResourceMax(t *testing.T) {
	t.Logf("Maria test instance name: %s", testutil.ConvertConfigVariable(testConfigVarsMax["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMariaDBDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.MariaDBProviderConfig(), resourceMaxConfig),
				Check: resource.ComposeTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["db_version"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMax["plan_name"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_enable_monitoring"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_graphite"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_max_disk_threshold"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_metrics_frequency"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_metrics_prefix"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_sgw_acl"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.syslog.0", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_syslog"])),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "parameters.monitoring_instance_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "plan_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "image_url"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "dashboard_url"),

					// Credential
					resource.TestCheckResourceAttrPair(
						"stackit_mariadb_instance.instance", "project_id",
						"stackit_mariadb_credential.credential", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_mariadb_instance.instance", "instance_id",
						"stackit_mariadb_credential.credential", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "name"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "username"),

					// Observability
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.observability_instance", "instance_id",
						"stackit_mariadb_instance.instance", "parameters.monitoring_instance_id",
					),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
					%s

					%s

					data "stackit_mariadb_instance" "instance" {
						project_id = stackit_mariadb_instance.instance.project_id
						instance_id = stackit_mariadb_instance.instance.instance_id
					}

					data "stackit_mariadb_credential" "credential" {
						project_id = stackit_mariadb_credential.credential.project_id
						instance_id = stackit_mariadb_credential.credential.instance_id
					    credential_id = stackit_mariadb_credential.credential.credential_id
					}`, testutil.MariaDBProviderConfig(), resourceMaxConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttrPair(
						"data.stackit_mariadb_instance.instance", "instance_id",
						"stackit_mariadb_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_mariadb_instance.instance", "project_id",
						"stackit_mariadb_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["db_version"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMax["plan_name"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_enable_monitoring"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_graphite"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_max_disk_threshold"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_metrics_frequency"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_metrics_prefix"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_sgw_acl"])),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "parameters.syslog.0", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_syslog"])),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "plan_id"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "image_url"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_instance.instance", "dashboard_url"),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_mariadb_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "name"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credential", "username"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_mariadb_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_mariadb_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_mariadb_instance.instance")
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
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_mariadb_credential.credential",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_mariadb_credential.credential"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_mariadb_credential.credential")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					credentialId, ok := r.Primary.Attributes["credential_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute credential_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, credentialId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: configVarsMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.MariaDBProviderConfig(), resourceMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "version", testutil.ConvertConfigVariable(configVarsMaxUpdated()["db_version"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "plan_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["plan_name"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(configVarsMaxUpdated()["parameters_enable_monitoring"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(configVarsMaxUpdated()["parameters_graphite"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(configVarsMaxUpdated()["parameters_max_disk_threshold"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(configVarsMaxUpdated()["parameters_metrics_frequency"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(configVarsMaxUpdated()["parameters_metrics_prefix"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(configVarsMaxUpdated()["parameters_sgw_acl"])),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.syslog.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["parameters_syslog"])),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "plan_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "image_url"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "parameters.monitoring_instance_id"),

					// Credential
					resource.TestCheckResourceAttrPair(
						"stackit_mariadb_instance.instance", "project_id",
						"stackit_mariadb_credential.credential", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_mariadb_instance.instance", "instance_id",
						"stackit_mariadb_credential.credential", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "name"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credential", "username"),

					// Observability
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.observability_instance", "instance_id",
						"stackit_mariadb_instance.instance", "parameters.monitoring_instance_id",
					),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckMariaDBDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *mariadb.APIClient
	var err error
	if testutil.MariaDBCustomEndpoint == "" {
		client, err = mariadb.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = mariadb.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.MariaDBCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_mariadb_instance" {
			continue
		}
		// instance terraform ID: "[project_id],[instance_id]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.ListInstances(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	instances := *instancesResp.Instances
	for i := range instances {
		if instances[i].InstanceId == nil {
			continue
		}
		if utils.Contains(instancesToDestroy, *instances[i].InstanceId) {
			if !checkInstanceDeleteSuccess(&instances[i]) {
				err := client.DeleteInstanceExecute(ctx, testutil.ProjectId, *instances[i].InstanceId)
				if err != nil {
					return fmt.Errorf("destroying instance %s during CheckDestroy: %w", *instances[i].InstanceId, err)
				}
				_, err = wait.DeleteInstanceWaitHandler(ctx, client, testutil.ProjectId, *instances[i].InstanceId).WaitWithContext(ctx)
				if err != nil {
					return fmt.Errorf("destroying instance %s during CheckDestroy: waiting for deletion %w", *instances[i].InstanceId, err)
				}
			}
		}
	}
	return nil
}

func checkInstanceDeleteSuccess(i *mariadb.Instance) bool {
	if *i.LastOperation.Type != mariadb.INSTANCELASTOPERATIONTYPE_DELETE {
		return false
	}

	if *i.LastOperation.Type == mariadb.INSTANCELASTOPERATIONTYPE_DELETE {
		if *i.LastOperation.State != mariadb.INSTANCELASTOPERATIONSTATE_SUCCEEDED {
			return false
		} else if strings.Contains(*i.LastOperation.Description, "DeleteFailed") || strings.Contains(*i.LastOperation.Description, "failed") {
			return false
		}
	}
	return true
}
