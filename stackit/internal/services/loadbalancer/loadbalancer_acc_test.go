package loadbalancer_test

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
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testfiles/resource-min.tf
var resourceMinConfig string

//go:embed testfiles/resource-max.tf
var resourceMaxConfig string

var testConfigVarsMin = config.Variables{
	"project_id":                        config.StringVariable(testutil.ProjectId),
	"plan_id":                           config.StringVariable("p10"),
	"disable_security_group_assignment": config.BoolVariable(false),
	"network_name":                      config.StringVariable(fmt.Sprintf("tf-acc-n%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"server_name":                       config.StringVariable(fmt.Sprintf("tf-acc-s%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"loadbalancer_name":                 config.StringVariable(fmt.Sprintf("tf-acc-l%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"target_pool_name":                  config.StringVariable("example-target-pool"),
	"target_port":                       config.StringVariable("5432"),
	"target_display_name":               config.StringVariable("example-target"),
	"listener_port":                     config.StringVariable("5432"),
	"listener_protocol":                 config.StringVariable("PROTOCOL_TLS_PASSTHROUGH"),
	"network_role":                      config.StringVariable("ROLE_LISTENERS_AND_TARGETS"),

	"obs_display_name": config.StringVariable("obs-user"),
	"obs_username":     config.StringVariable("obs-username"),
	"obs_password":     config.StringVariable("obs-password1"),
}

var testConfigVarsMax = config.Variables{
	"project_id":                        config.StringVariable(testutil.ProjectId),
	"plan_id":                           config.StringVariable("p10"),
	"disable_security_group_assignment": config.BoolVariable(true),
	"network_name":                      config.StringVariable(fmt.Sprintf("tf-acc-n%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"network_role":                      config.StringVariable("ROLE_LISTENERS_AND_TARGETS"),
	"server_name":                       config.StringVariable(fmt.Sprintf("tf-acc-s%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"loadbalancer_name":                 config.StringVariable(fmt.Sprintf("tf-acc-l%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),

	"target_display_name": config.StringVariable("example-target"),

	"sni_target_pool_name":                config.StringVariable("example-target-pool"),
	"sni_target_port":                     config.StringVariable("5432"),
	"sni_listener_port":                   config.StringVariable("5432"),
	"sni_listener_protocol":               config.StringVariable("PROTOCOL_TLS_PASSTHROUGH"),
	"sni_idle_timeout":                    config.StringVariable("42s"),
	"sni_listener_display_name":           config.StringVariable("example-listener"),
	"sni_listener_server_name_indicators": config.StringVariable("acc-test.runs.onstackit.cloud"),
	"sni_healthy_threshold":               config.StringVariable("3"),
	"sni_health_interval":                 config.StringVariable("10s"),
	"sni_health_interval_jitter":          config.StringVariable("5s"),
	"sni_health_timeout":                  config.StringVariable("10s"),
	"sni_unhealthy_threshold":             config.StringVariable("3"),
	"sni_use_source_ip_address":           config.StringVariable("true"),

	"udp_target_pool_name":      config.StringVariable("udp-target-pool"),
	"udp_target_port":           config.StringVariable("53"),
	"udp_listener_port":         config.StringVariable("53"),
	"udp_listener_protocol":     config.StringVariable("PROTOCOL_UDP"),
	"udp_idle_timeout":          config.StringVariable("43s"),
	"udp_listener_display_name": config.StringVariable("udp-listener"),

	"private_network_only": config.StringVariable("false"),
	"acl":                  config.StringVariable("192.168.0.0/24"),

	"observability_logs_push_url":               config.StringVariable("https://logs.observability.dummy.stackit.cloud"),
	"observability_metrics_push_url":            config.StringVariable("https://metrics.observability.dummy.stackit.cloud"),
	"observability_credential_logs_name":        config.StringVariable(fmt.Sprintf("tf-acc-l%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"observability_credential_logs_username":    config.StringVariable("obs-cred-logs-username"),
	"observability_credential_logs_password":    config.StringVariable("obs-cred-logs-password"),
	"observability_credential_metrics_name":     config.StringVariable(fmt.Sprintf("tf-acc-m%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"observability_credential_metrics_username": config.StringVariable("obs-cred-metrics-username"),
	"observability_credential_metrics_password": config.StringVariable("obs-cred-metrics-password"),
}

func configVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(tempConfig, testConfigVarsMin)
	tempConfig["plan_id"] = config.StringVariable("p50")

	tempConfig["target_port"] = config.StringVariable("6543")
	tempConfig["target_pool_name"] = config.StringVariable("example-target-pool-updated")
	tempConfig["target_display_name"] = config.StringVariable("example-target-updated")

	tempConfig["listener_protocol"] = config.StringVariable("PROTOCOL_TCP")
	tempConfig["listener_port"] = config.StringVariable("6543")
	return tempConfig
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMax))
	maps.Copy(tempConfig, testConfigVarsMax)
	tempConfig["plan_id"] = config.StringVariable("p50")
	tempConfig["acl"] = config.StringVariable("10.2.1.0/24")
	tempConfig["target_display_name"] = config.StringVariable("example-target-updated")

	tempConfig["sni_target_pool_name"] = config.StringVariable("example-target-pool-updated")
	tempConfig["sni_target_port"] = config.StringVariable("6543")
	tempConfig["sni_listener_port"] = config.StringVariable("6543")
	tempConfig["sni_listener_protocol"] = config.StringVariable("PROTOCOL_TCP")
	tempConfig["sni_idle_timeout"] = config.StringVariable("21s")
	tempConfig["sni_listener_display_name"] = config.StringVariable("example-listener-updated")
	tempConfig["sni_listener_server_name_indicators"] = config.StringVariable("")
	tempConfig["sni_healthy_threshold"] = config.StringVariable("4")
	tempConfig["sni_health_interval"] = config.StringVariable("12s")
	tempConfig["sni_health_interval_jitter"] = config.StringVariable("7s")
	tempConfig["sni_health_timeout"] = config.StringVariable("15s")
	tempConfig["sni_unhealthy_threshold"] = config.StringVariable("4")
	tempConfig["sni_use_source_ip_address"] = config.StringVariable("false")

	tempConfig["udp_target_pool_name"] = config.StringVariable("udp-target-pool-updated")
	tempConfig["udp_target_port"] = config.StringVariable("67")
	tempConfig["udp_listener_port"] = config.StringVariable("67")
	tempConfig["udp_idle_timeout"] = config.StringVariable("44s")
	tempConfig["udp_listener_display_name"] = config.StringVariable("udp-listener-updated")

	tempConfig["observability_logs_push_url"] = config.StringVariable("https://logs.observability.dummy.stackit.cloud")
	tempConfig["observability_metrics_push_url"] = config.StringVariable("https://metrics.observability.dummy.stackit.cloud")
	return tempConfig
}

func TestAccLoadBalancerResourceMin(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLoadBalancerDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance resource
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMin["loadbalancer_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(testConfigVarsMin["target_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMin["target_display_name"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "listeners.0.display_name"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(testConfigVarsMin["listener_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(testConfigVarsMin["listener_protocol"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_name"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "networks.0.role", testutil.ConvertConfigVariable(testConfigVarsMin["network_role"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "disable_security_group_assignment", "false"),
					resource.TestCheckNoResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckNoResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.logs.push_url"),
					resource.TestCheckNoResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckNoResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.metrics.push_url"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "security_group_id"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "version"),

					// Loadbalancer observability credentials resource
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.obs_credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.obs_credential", "display_name", testutil.ConvertConfigVariable(testConfigVarsMin["obs_display_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.obs_credential", "username", testutil.ConvertConfigVariable(testConfigVarsMin["obs_username"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.obs_credential", "password", testutil.ConvertConfigVariable(testConfigVarsMin["obs_password"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_observability_credential.obs_credential", "credentials_ref"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
						%s

						data "stackit_loadbalancer" "loadbalancer" {
							project_id     = stackit_loadbalancer.loadbalancer.project_id
							name    = stackit_loadbalancer.loadbalancer.name
						}
						`,
					testutil.NewConfigBuilder().BuildProviderConfig()+resourceMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMin["loadbalancer_name"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMin["plan_id"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_loadbalancer.loadbalancer", "project_id",
						"stackit_loadbalancer.loadbalancer", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_loadbalancer.loadbalancer", "name",
						"stackit_loadbalancer.loadbalancer", "name",
					),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_name"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(testConfigVarsMin["target_port"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMin["target_display_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "listeners.0.display_name"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(testConfigVarsMin["listener_port"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(testConfigVarsMin["listener_protocol"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "networks.0.role", testutil.ConvertConfigVariable(testConfigVarsMin["network_role"])),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "disable_security_group_assignment", "false"),
					resource.TestCheckNoResourceAttr("data.stackit_loadbalancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckNoResourceAttr("data.stackit_loadbalancer.loadbalancer", "options.observability.logs.push_url"),
					resource.TestCheckNoResourceAttr("data.stackit_loadbalancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckNoResourceAttr("data.stackit_loadbalancer.loadbalancer", "options.observability.metrics.push_url"),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "security_group_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_loadbalancer.loadbalancer", "security_group_id",
						"data.stackit_loadbalancer.loadbalancer", "security_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_loadbalancer.loadbalancer", "version",
						"data.stackit_loadbalancer.loadbalancer", "version",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_loadbalancer.loadbalancer",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_loadbalancer.loadbalancer"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_loadbalancer.loadbalancer")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, region, name), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"options.private_network_only"},
			},
			// Update
			{
				ConfigVariables: configVarsMinUpdated(),
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceMinConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_loadbalancer.loadbalancer", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_loadbalancer_observability_credential.obs_credential", plancheck.ResourceActionNoop),

						plancheck.ExpectResourceAction("stackit_network.network", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_network_interface.network_interface", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_public_ip.public_ip", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_server.server", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance resource
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "name", testutil.ConvertConfigVariable(configVarsMinUpdated()["loadbalancer_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(configVarsMinUpdated()["target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(configVarsMinUpdated()["target_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(configVarsMinUpdated()["target_display_name"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "listeners.0.display_name"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(configVarsMinUpdated()["listener_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(configVarsMinUpdated()["listener_protocol"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", testutil.ConvertConfigVariable(configVarsMinUpdated()["target_pool_name"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "networks.0.role", testutil.ConvertConfigVariable(configVarsMinUpdated()["network_role"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "disable_security_group_assignment", "false"),
					resource.TestCheckNoResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckNoResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.logs.push_url"),
					resource.TestCheckNoResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckNoResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.metrics.push_url"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "security_group_id"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "version"),

					// Loadbalancer observability credentials resource
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.obs_credential", "project_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.obs_credential", "display_name", testutil.ConvertConfigVariable(configVarsMinUpdated()["obs_display_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.obs_credential", "username", testutil.ConvertConfigVariable(configVarsMinUpdated()["obs_username"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.obs_credential", "password", testutil.ConvertConfigVariable(configVarsMinUpdated()["obs_password"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_observability_credential.obs_credential", "credentials_ref"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccLoadBalancerResourceMax(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLoadBalancerDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance resource
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMax["loadbalancer_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMax["plan_id"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "networks.0.role", testutil.ConvertConfigVariable(testConfigVarsMax["network_role"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "disable_security_group_assignment", testutil.ConvertConfigVariable(testConfigVarsMax["disable_security_group_assignment"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "security_group_id"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "version"),

					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["sni_listener_display_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(testConfigVarsMax["sni_listener_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(testConfigVarsMax["sni_listener_protocol"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["sni_target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.server_name_indicators.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["sni_listener_server_name_indicators"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.tcp.idle_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["sni_idle_timeout"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["sni_target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["sni_target_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.healthy_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["sni_healthy_threshold"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval", testutil.ConvertConfigVariable(testConfigVarsMax["sni_health_interval"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval_jitter", testutil.ConvertConfigVariable(testConfigVarsMax["sni_health_interval_jitter"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.timeout", testutil.ConvertConfigVariable(testConfigVarsMax["sni_health_timeout"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.unhealthy_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["sni_unhealthy_threshold"])),

					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["udp_listener_display_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.port", testutil.ConvertConfigVariable(testConfigVarsMax["udp_listener_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.protocol", testutil.ConvertConfigVariable(testConfigVarsMax["udp_listener_protocol"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["udp_target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.udp.idle_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["udp_idle_timeout"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.1.name", testutil.ConvertConfigVariable(testConfigVarsMax["udp_target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.1.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["udp_target_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.1.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "target_pools.1.targets.0.ip"),

					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.session_persistence.use_source_ip_address", testutil.ConvertConfigVariable(testConfigVarsMax["sni_use_source_ip_address"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.private_network_only", testutil.ConvertConfigVariable(testConfigVarsMax["private_network_only"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),

					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckResourceAttrPair("stackit_loadbalancer_observability_credential.logs", "credentials_ref", "stackit_loadbalancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.logs.push_url", testutil.ConvertConfigVariable(testConfigVarsMax["observability_logs_push_url"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckResourceAttrPair("stackit_loadbalancer_observability_credential.metrics", "credentials_ref", "stackit_loadbalancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.metrics.push_url", testutil.ConvertConfigVariable(testConfigVarsMax["observability_metrics_push_url"])),

					// Loadbalancer observability credential resource
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.logs", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.logs", "display_name", testutil.ConvertConfigVariable(testConfigVarsMax["observability_credential_logs_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.logs", "username", testutil.ConvertConfigVariable(testConfigVarsMax["observability_credential_logs_username"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.logs", "password", testutil.ConvertConfigVariable(testConfigVarsMax["observability_credential_logs_password"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_observability_credential.logs", "credentials_ref"),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.metrics", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.metrics", "display_name", testutil.ConvertConfigVariable(testConfigVarsMax["observability_credential_metrics_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.metrics", "username", testutil.ConvertConfigVariable(testConfigVarsMax["observability_credential_metrics_username"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.metrics", "password", testutil.ConvertConfigVariable(testConfigVarsMax["observability_credential_metrics_password"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_observability_credential.metrics", "credentials_ref"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
						%s

						data "stackit_loadbalancer" "loadbalancer" {
							project_id     = stackit_loadbalancer.loadbalancer.project_id
							name    = stackit_loadbalancer.loadbalancer.name
						}
						`,
					testutil.NewConfigBuilder().BuildProviderConfig()+resourceMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMax["loadbalancer_name"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMax["plan_id"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_loadbalancer.loadbalancer", "project_id",
						"stackit_loadbalancer.loadbalancer", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_loadbalancer.loadbalancer", "name",
						"stackit_loadbalancer.loadbalancer", "name",
					),
					// Load balancer instance
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "networks.0.role", testutil.ConvertConfigVariable(testConfigVarsMax["network_role"])),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "disable_security_group_assignment", testutil.ConvertConfigVariable(testConfigVarsMax["disable_security_group_assignment"])),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "security_group_id"),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "version"),

					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["sni_target_pool_name"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["sni_target_port"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["sni_listener_display_name"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(testConfigVarsMax["sni_listener_port"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(testConfigVarsMax["sni_listener_protocol"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["sni_target_pool_name"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.server_name_indicators.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["sni_listener_server_name_indicators"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.tcp.idle_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["sni_idle_timeout"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.healthy_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["sni_healthy_threshold"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval", testutil.ConvertConfigVariable(testConfigVarsMax["sni_health_interval"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval_jitter", testutil.ConvertConfigVariable(testConfigVarsMax["sni_health_interval_jitter"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.timeout", testutil.ConvertConfigVariable(testConfigVarsMax["sni_health_timeout"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.unhealthy_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["sni_unhealthy_threshold"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.session_persistence.use_source_ip_address", testutil.ConvertConfigVariable(testConfigVarsMax["sni_use_source_ip_address"])),

					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["udp_listener_display_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.port", testutil.ConvertConfigVariable(testConfigVarsMax["udp_listener_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.protocol", testutil.ConvertConfigVariable(testConfigVarsMax["udp_listener_protocol"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["udp_target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.udp.idle_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["udp_idle_timeout"])),

					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "options.acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckResourceAttrPair("stackit_loadbalancer_observability_credential.logs", "credentials_ref", "data.stackit_loadbalancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "options.observability.logs.push_url", testutil.ConvertConfigVariable(testConfigVarsMax["observability_logs_push_url"])),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckResourceAttrPair("stackit_loadbalancer_observability_credential.metrics", "credentials_ref", "data.stackit_loadbalancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "options.observability.metrics.push_url", testutil.ConvertConfigVariable(testConfigVarsMax["observability_metrics_push_url"])),
					resource.TestCheckResourceAttrPair(
						"stackit_loadbalancer.loadbalancer", "security_group_id",
						"data.stackit_loadbalancer.loadbalancer", "security_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_loadbalancer.loadbalancer", "version",
						"data.stackit_loadbalancer.loadbalancer", "version",
					),
				)},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_loadbalancer.loadbalancer",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_loadbalancer.loadbalancer"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_loadbalancer.loadbalancer")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, region, name), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"options.private_network_only"},
			},
			// Update
			{
				ConfigVariables: configVarsMaxUpdated(),
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceMaxConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_loadbalancer.loadbalancer", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_loadbalancer_observability_credential.logs", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_loadbalancer_observability_credential.metrics", plancheck.ResourceActionNoop),

						plancheck.ExpectResourceAction("stackit_network.network", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_network_interface.network_interface", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_public_ip.public_ip", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_server.server", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance resource
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["loadbalancer_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "plan_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["plan_id"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "networks.0.role", testutil.ConvertConfigVariable(configVarsMaxUpdated()["network_role"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "disable_security_group_assignment", testutil.ConvertConfigVariable(configVarsMaxUpdated()["disable_security_group_assignment"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "security_group_id"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "version"),

					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.display_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_listener_display_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_listener_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_listener_protocol"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_target_pool_name"])),
					resource.TestCheckNoResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.server_name_indicators.0.name"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.tcp.idle_timeout", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_idle_timeout"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_target_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["target_display_name"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.healthy_threshold", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_healthy_threshold"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_health_interval"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval_jitter", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_health_interval_jitter"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.timeout", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_health_timeout"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.unhealthy_threshold", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_unhealthy_threshold"])),

					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.display_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["udp_listener_display_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.port", testutil.ConvertConfigVariable(configVarsMaxUpdated()["udp_listener_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.protocol", testutil.ConvertConfigVariable(configVarsMaxUpdated()["udp_listener_protocol"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.target_pool", testutil.ConvertConfigVariable(configVarsMaxUpdated()["udp_target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.1.udp.idle_timeout", testutil.ConvertConfigVariable(configVarsMaxUpdated()["udp_idle_timeout"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.1.name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["udp_target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.1.target_port", testutil.ConvertConfigVariable(configVarsMaxUpdated()["udp_target_port"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.1.targets.0.display_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["target_display_name"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "target_pools.1.targets.0.ip"),

					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.session_persistence.%", "1"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.session_persistence.use_source_ip_address", testutil.ConvertConfigVariable(configVarsMaxUpdated()["sni_use_source_ip_address"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.private_network_only", testutil.ConvertConfigVariable(configVarsMaxUpdated()["private_network_only"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.acl.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["acl"])),

					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckResourceAttrPair("stackit_loadbalancer_observability_credential.logs", "credentials_ref", "stackit_loadbalancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.logs.push_url", testutil.ConvertConfigVariable(configVarsMaxUpdated()["observability_logs_push_url"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckResourceAttrPair("stackit_loadbalancer_observability_credential.metrics", "credentials_ref", "stackit_loadbalancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.observability.metrics.push_url", testutil.ConvertConfigVariable(configVarsMaxUpdated()["observability_metrics_push_url"])),

					// Loadbalancer observability credential resource
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.logs", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.logs", "display_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["observability_credential_logs_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.logs", "username", testutil.ConvertConfigVariable(configVarsMaxUpdated()["observability_credential_logs_username"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.logs", "password", testutil.ConvertConfigVariable(configVarsMaxUpdated()["observability_credential_logs_password"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_observability_credential.logs", "credentials_ref"),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.metrics", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.metrics", "display_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["observability_credential_metrics_name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.metrics", "username", testutil.ConvertConfigVariable(configVarsMaxUpdated()["observability_credential_metrics_username"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_observability_credential.metrics", "password", testutil.ConvertConfigVariable(configVarsMaxUpdated()["observability_credential_metrics_password"])),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_observability_credential.metrics", "credentials_ref"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckLoadBalancerDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := loadbalancer.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.LoadBalancerCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	region := "eu01"
	if testutil.Region != "" {
		region = testutil.Region
	}
	loadbalancersToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_loadbalancer" {
			continue
		}
		// loadbalancer terraform ID: = "[project_id],[name]"
		loadbalancerName := strings.Split(rs.Primary.ID, core.Separator)[1]
		loadbalancersToDestroy = append(loadbalancersToDestroy, loadbalancerName)
	}

	loadbalancersResp, err := client.DefaultAPI.ListLoadBalancers(ctx, testutil.ProjectId, region).Execute()
	if err != nil {
		return fmt.Errorf("getting loadbalancersResp: %w", err)
	}

	if len(loadbalancersResp.LoadBalancers) == 0 {
		fmt.Print("No load balancers found for project \n")
		return nil
	}

	items := loadbalancersResp.LoadBalancers
	for i := range items {
		if items[i].Name == nil {
			continue
		}
		if utils.Contains(loadbalancersToDestroy, *items[i].Name) {
			_, err := client.DefaultAPI.DeleteLoadBalancer(ctx, testutil.ProjectId, region, *items[i].Name).Execute()
			if err != nil {
				return fmt.Errorf("destroying load balancer %s during CheckDestroy: %w", *items[i].Name, err)
			}
			_, err = wait.DeleteLoadBalancerWaitHandler(ctx, client.DefaultAPI, testutil.ProjectId, region, *items[i].Name).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying load balancer %s during CheckDestroy: waiting for deletion %w", *items[i].Name, err)
			}
		}
	}
	return nil
}
