package rabbitmq_test

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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceMinConfig string

//go:embed testdata/resource-max.tf
var resourceMaxConfig string

var testConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"db_version": config.StringVariable("3.13"),
	"plan_name":  config.StringVariable("stackit-rabbitmq-2.4.10-single"),
}

var testConfigVarsMax = config.Variables{
	"project_id":                    config.StringVariable(testutil.ProjectId),
	"name":                          config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"db_version":                    config.StringVariable("3.13"),
	"plan_name":                     config.StringVariable("stackit-rabbitmq-2.4.10-single"),
	"parameters_sgw_acl":            config.StringVariable("192.168.0.0/16"),
	"parameters_consumer_timeout":   config.IntegerVariable(1800000),
	"parameters_enable_monitoring":  config.BoolVariable(true),
	"parameters_graphite":           config.StringVariable("graphite.example.com:2003"),
	"parameters_max_disk_threshold": config.IntegerVariable(80),
	"parameters_metrics_frequency":  config.IntegerVariable(60),
	"parameters_metrics_prefix":     config.StringVariable("rabbitmq"),
	"parameters_plugins":            config.ListVariable(config.StringVariable("rabbitmq_federation")),
	"parameters_roles":              config.ListVariable(config.StringVariable("administrator")),
	"parameters_syslog":             config.ListVariable(config.StringVariable("syslog.example.com:514")),
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMax)
	tempConfig["parameters_max_disk_threshold"] = config.IntegerVariable(85)
	tempConfig["parameters_metrics_frequency"] = config.IntegerVariable(30)
	tempConfig["parameters_graphite"] = config.StringVariable("graphite.updated.com:2003")
	tempConfig["parameters_sgw_acl"] = config.StringVariable("192.168.1.0/24")
	return tempConfig
}

// minimum configuration
func TestAccRabbitMQResourceMin(t *testing.T) {
	t.Logf("RabbitMQ test instance name: %s", testutil.ConvertConfigVariable(testConfigVarsMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRabbitMQDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.RabbitMQProviderConfig(), resourceMinConfig),
				Check: resource.ComposeTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMin["db_version"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "plan_id"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "image_url"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "dashboard_url"),

					// Credential
					resource.TestCheckResourceAttrPair(
						"stackit_rabbitmq_instance.instance", "project_id",
						"stackit_rabbitmq_credential.credential", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_rabbitmq_instance.instance", "instance_id",
						"stackit_rabbitmq_credential.credential", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "username"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
					%s

					%s

					data "stackit_rabbitmq_instance" "instance" {
			project_id = stackit_rabbitmq_instance.instance.project_id
			instance_id = stackit_rabbitmq_instance.instance.instance_id
		}

					data "stackit_rabbitmq_credential" "credential" {
						project_id = stackit_rabbitmq_credential.credential.project_id
						instance_id = stackit_rabbitmq_credential.credential.instance_id
						credential_id = stackit_rabbitmq_credential.credential.credential_id
					}`, testutil.RabbitMQProviderConfig(), resourceMinConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttrPair(
						"data.stackit_rabbitmq_instance.instance", "instance_id",
						"stackit_rabbitmq_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMin["db_version"])),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "plan_id"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "image_url"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "dashboard_url"),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "username"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_rabbitmq_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_rabbitmq_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_rabbitmq_instance.instance")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					return fmt.Sprintf("%s,%s", projectId, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// maximum configuration
func TestAccRabbitMQResourceMax(t *testing.T) {
	t.Logf("RabbitMQ test instance name: %s", testutil.ConvertConfigVariable(testConfigVarsMax["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRabbitMQDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.RabbitMQProviderConfig(), resourceMaxConfig),
				Check: resource.ComposeTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["db_version"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMax["plan_name"])),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "plan_id"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "image_url"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "dashboard_url"),

					// Instance parameters
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_sgw_acl"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.consumer_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_consumer_timeout"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_enable_monitoring"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_graphite"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_max_disk_threshold"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_metrics_frequency"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_metrics_prefix"])),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.plugins.#", "1"),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.plugins.0", "rabbitmq_federation"),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.roles.#", "1"),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.roles.0", "administrator"),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.syslog.#", "1"),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.syslog.0", "syslog.example.com:514"),

					// Credential
					resource.TestCheckResourceAttrPair(
						"stackit_rabbitmq_instance.instance", "project_id",
						"stackit_rabbitmq_credential.credential", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_rabbitmq_instance.instance", "instance_id",
						"stackit_rabbitmq_credential.credential", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credential", "username"),
				),
			},
			// Update step removed - RabbitMQ requires monitoring instance ID for updates
			// Data source
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
					%s

					%s

					data "stackit_rabbitmq_instance" "instance" {
						project_id = stackit_rabbitmq_instance.instance.project_id
						instance_id = stackit_rabbitmq_instance.instance.instance_id
					}

					data "stackit_rabbitmq_credential" "credential" {
						project_id = stackit_rabbitmq_credential.credential.project_id
						instance_id = stackit_rabbitmq_credential.credential.instance_id
					    credential_id = stackit_rabbitmq_credential.credential.credential_id
					}`, testutil.RabbitMQProviderConfig(), resourceMaxConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttrPair(
						"data.stackit_rabbitmq_instance.instance", "instance_id",
						"stackit_rabbitmq_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["db_version"])),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMax["plan_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "plan_id"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "image_url"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "dashboard_url"),

					// Instance parameters data
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_max_disk_threshold"])),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_metrics_frequency"])),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_graphite"])),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(testConfigVarsMax["parameters_sgw_acl"])),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credential", "username"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_rabbitmq_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_rabbitmq_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_rabbitmq_instance.instance")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					return fmt.Sprintf("%s,%s", projectId, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckRabbitMQDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *rabbitmq.APIClient
	var err error
	if testutil.RabbitMQCustomEndpoint == "" {
		client, err = rabbitmq.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = rabbitmq.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.RabbitMQCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_rabbitmq_instance" {
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

func checkInstanceDeleteSuccess(i *rabbitmq.Instance) bool {
	if *i.LastOperation.Type != rabbitmq.INSTANCELASTOPERATIONTYPE_DELETE {
		return false
	}

	if *i.LastOperation.Type == rabbitmq.INSTANCELASTOPERATIONTYPE_DELETE {
		if *i.LastOperation.State != rabbitmq.INSTANCELASTOPERATIONSTATE_SUCCEEDED {
			return false
		} else if strings.Contains(*i.LastOperation.Description, "DeleteFailed") || strings.Contains(*i.LastOperation.Description, "failed") {
			return false
		}
	}
	return true
}
