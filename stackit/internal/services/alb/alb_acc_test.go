package alb_test

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
	"github.com/stackitcloud/stackit-sdk-go/services/alb/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/alb"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testfiles/resource-min.tf
var resourceMinConfig string

//go:embed testfiles/resource-max.tf
var resourceMaxConfig string

var testConfigVarsMin = config.Variables{
	"project_id":          config.StringVariable(testutil.ProjectId),
	"region":              config.StringVariable(testutil.Region),
	"loadbalancer_name":   config.StringVariable(fmt.Sprintf("tf-acc-l%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"network_role":        config.StringVariable("ROLE_LISTENERS_AND_TARGETS"),
	"network_name":        config.StringVariable(fmt.Sprintf("tf-acc-n%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"plan_id":             config.StringVariable("p10"),
	"listener_port":       config.StringVariable("5432"),
	"host":                config.StringVariable("*"),
	"path_prefix":         config.StringVariable("/"),
	"protocol_http":       config.StringVariable("PROTOCOL_HTTP"),
	"target_pool_name":    config.StringVariable("my-target-pool"),
	"target_pool_port":    config.StringVariable("5432"),
	"target_display_name": config.StringVariable("my-target"),
}

var testConfigVarsMax = config.Variables{
	"project_id":                        config.StringVariable(testutil.ProjectId),
	"region":                            config.StringVariable(testutil.Region),
	"loadbalancer_name":                 config.StringVariable(fmt.Sprintf("tf-acc-l%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"labels_key_1":                      config.StringVariable("key1"),
	"labels_value_1":                    config.StringVariable("value1"),
	"labels_key_2":                      config.StringVariable("key2"),
	"labels_value_2":                    config.StringVariable("value2"),
	"plan_id":                           config.StringVariable("p10"),
	"protocol_http":                     config.StringVariable("PROTOCOL_HTTP"),
	"disable_security_group_assignment": config.BoolVariable(true),
	"listener_port_1":                   config.StringVariable("445"),
	"listener_port_4":                   config.StringVariable("80"),
	"tls_config_enabled":                config.BoolVariable(true),
	"tls_config_skip":                   config.BoolVariable(false),
	"tls_config_custom_ca":              config.StringVariable("-----BEGIN CERTIFICATE-----\nMIIDCzCCAfOgAwIBAgIUTyPsTWC9ly7o+wNFYm0uu1+P8IEwDQYJKoZIhvcNAQEL\nBQAwFTETMBEGA1UEAwwKTXlDdXN0b21DQTAeFw0yNTAyMTkxOTI0MjBaFw0yNjAy\nMTkxOTI0MjBaMBUxEzARBgNVBAMMCk15Q3VzdG9tQ0EwggEiMA0GCSqGSIb3DQEB\nAQUAA4IBDwAwggEKAoIBAQCQMEYKbiNxU37fEwBOxkvCshBR+0MwxwLW8Mi3/pvo\nn3huxjcm7EaKW9r7kIaoHXbTS1tnO6rHAHKBDxzuoYD7C2SMSiLxddquNRvpkLaP\n8qAXneQY2VP7LzsAgsC04PKG0YC1NgF5sJGsiWIRGIm+csYLnPMnwaAGx4IvY6mH\nAmM64b6QRCg36LK+P6N9KTvSQLvvmFdkA2sDToCmN/Amp6xNDFq+aQGLwdQQqHDP\nTaUqPmEyiFHKvFUaFMNQVk8B1Om8ASo69m8U3Eat4ZOVW1titE393QkOdA6ZypMC\nrJJpeNNLLJq3mIOWOd7GEyAvjUfmJwGhqEFS7lMG67hnAgMBAAGjUzBRMB0GA1Ud\nDgQWBBSk/IM5jaOAJL3/Knyq3cVva04YZDAfBgNVHSMEGDAWgBSk/IM5jaOAJL3/\nKnyq3cVva04YZDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQBe\nZ/mE8rNIbNbHQep/VppshaZUzgdy4nsmh0wvxMuHIQP0KHrxLCkhOn7A9fu4mY/P\nQ+8QqlnjTsM4cqiuFcd5V1Nk9VF/e5X3HXCDHh/jBFw+O5TGVAR/7DBw31lYv/Lt\nHakkjQCdawuvH3osO/UkElM/i2KC+iYBavTenm97AR7WGgW15/MIqxNaYE+nJth/\ndcVD0b5qSuYQaEmZ3CzMUi188R+go5ozCf2cOaa+3/LEYAaI3vKiSE8KTsshyoKm\nO6YZqrVxQCWCDTOsd28k7lHt8wJ+jzYcjCu60DUpg1ZpY+ZnmrE8vPPDb/zXhBn6\n/llXTWOUjmuTKnGsIDP5\n-----END CERTIFICATE-----"),
	"web_socket":                        config.BoolVariable(true),
	"query_parameters_name_1":           config.StringVariable("a"),
	"query_parameters_exact_match_1":    config.StringVariable("b"),
	"query_parameters_name_2":           config.StringVariable("c"),
	"query_parameters_exact_match_2":    config.StringVariable("d"),
	"headers_name_1":                    config.StringVariable("1"),
	"headers_exact_match_1":             config.StringVariable("2"),
	"headers_name_2":                    config.StringVariable("3"),
	"headers_exact_match_2":             config.StringVariable("4"),
	"headers_name_3":                    config.StringVariable("5"),
	"host_1":                            config.StringVariable("*"),
	"host_3":                            config.StringVariable("www.example.org"),
	"host_4":                            config.StringVariable("www.*"),
	"path_prefix_1":                     config.StringVariable("/specific-path-1"),
	"path_prefix_2":                     config.StringVariable("/specific-path-2"),
	"path_prefix_3":                     config.StringVariable("/specific-path-3"),
	"path_prefix_4":                     config.StringVariable("/specific-path-4"),
	"network_name_listener":             config.StringVariable(fmt.Sprintf("tf-acc-l%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"network_name_targets":              config.StringVariable(fmt.Sprintf("tf-acc-t%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"network_role_listeners":            config.StringVariable("ROLE_LISTENERS"),
	"network_role_targets":              config.StringVariable("ROLE_TARGETS"),
	"target_pool_name_1":                config.StringVariable("my-target-pool-1"),
	"target_pool_name_2":                config.StringVariable("my-target-pool-2"),
	"target_pool_name_3":                config.StringVariable("my-target-pool-3"),
	"target_pool_name_4":                config.StringVariable("my-target-pool-4"),
	"target_pool_port_1":                config.StringVariable("443"),
	"target_pool_port_2":                config.StringVariable("1337"),
	"target_pool_port_3":                config.StringVariable("9001"),
	"target_pool_port_4":                config.StringVariable("1234"),
	"target_display_name":               config.StringVariable("example-target"),
	"ahc_interval":                      config.StringVariable("1s"),
	"ahc_interval_jitter":               config.StringVariable("0.010s"),
	"ahc_timeout":                       config.StringVariable("1s"),
	"ahc_healthy_threshold":             config.StringVariable("3"),
	"ahc_unhealthy_threshold":           config.StringVariable("5"),
	"ahc_http_ok_status_200":            config.StringVariable("200"),
	"ahc_http_ok_status_201":            config.StringVariable("201"),
	"ahc_http_path":                     config.StringVariable("/healthy"),
	"ephemeral_address":                 config.BoolVariable(true),
	"private_network_only":              config.BoolVariable(false),
	"acl":                               config.StringVariable("192.168.0.0/24"),
	"observability_logs_push_url":       config.StringVariable("https://logs.observability.dummy.stackit.cloud"),
	"observability_metrics_push_url":    config.StringVariable("https://metrics.observability.dummy.stackit.cloud"),
	"observability_credential_name":     config.StringVariable(fmt.Sprintf("tf-acc-l%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"observability_credential_username": config.StringVariable("obs-cred-username"),
	"observability_credential_password": config.StringVariable("obs-cred-password"),
}

func configVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(tempConfig, testConfigVarsMin)
	tempConfig["target_pool_port"] = config.StringVariable("5431")
	return tempConfig
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMax))
	maps.Copy(tempConfig, testConfigVarsMax)
	tempConfig["ephemeral_address"] = config.BoolVariable(false)
	tempConfig["web_socket"] = config.BoolVariable(false)
	tempConfig["query_parameters_name_1"] = config.StringVariable("e")
	tempConfig["query_parameters_exact_match_1"] = config.StringVariable("f")
	tempConfig["query_parameters_name_2"] = config.StringVariable("g")
	tempConfig["query_parameters_exact_match_2"] = config.StringVariable("h")
	tempConfig["headers_name_1"] = config.StringVariable("6")
	tempConfig["headers_exact_match_1"] = config.StringVariable("7")
	tempConfig["headers_name_2"] = config.StringVariable("8")
	tempConfig["headers_exact_match_2"] = config.StringVariable("9")
	tempConfig["headers_name_3"] = config.StringVariable("0")
	tempConfig["host_1"] = config.StringVariable("www.example.*")
	tempConfig["target_pool_port_1"] = config.StringVariable("444")
	tempConfig["ahc_http_ok_status_200"] = config.StringVariable("202")
	tempConfig["ahc_http_ok_status_201"] = config.StringVariable("203")
	tempConfig["ahc_timeout"] = config.StringVariable("5s")
	tempConfig["ahc_healthy_threshold"] = config.StringVariable("5")
	tempConfig["acl"] = config.StringVariable("10.11.10.8/24")
	return tempConfig
}

func TestAccALBResourceMin(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckALBDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.ALBProviderConfig() + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance resource
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "region", testutil.ConvertConfigVariable(testConfigVarsMin["region"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMin["loadbalancer_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "networks.0.role", testutil.ConvertConfigVariable(testConfigVarsMin["network_role"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMin["plan_id"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(testConfigVarsMin["listener_port"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.host", testutil.ConvertConfigVariable(testConfigVarsMin["host"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMin["path_prefix"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(testConfigVarsMin["protocol_http"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_port"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMin["target_display_name"])),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_security_group.id"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_security_group.name"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.id"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.name"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "version"),
					resource.TestCheckNoResourceAttr("stackit_application_load_balancer.loadbalancer", "disable_security_group_assignment"),
					resource.TestCheckNoResourceAttr("stackit_application_load_balancer.loadbalancer", "options"),
					resource.TestCheckNoResourceAttr("stackit_application_load_balancer.loadbalancer", "labels"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
						%s

						data "stackit_application_load_balancer" "loadbalancer" {
							project_id     = stackit_application_load_balancer.loadbalancer.project_id
							name    = stackit_application_load_balancer.loadbalancer.name
						}
						`,
					testutil.ALBProviderConfig()+resourceMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "region", testutil.ConvertConfigVariable(testConfigVarsMin["region"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMin["loadbalancer_name"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "networks.0.role", testutil.ConvertConfigVariable(testConfigVarsMin["network_role"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMin["plan_id"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(testConfigVarsMin["listener_port"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.host", testutil.ConvertConfigVariable(testConfigVarsMin["host"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMin["path_prefix"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_name"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(testConfigVarsMin["protocol_http"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_name"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(testConfigVarsMin["target_pool_port"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMin["target_display_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_security_group.id"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_security_group.name"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.id"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.name"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "version"),
					resource.TestCheckNoResourceAttr("data.stackit_application_load_balancer.loadbalancer", "disable_security_group_assignment"),
					resource.TestCheckNoResourceAttr("data.stackit_application_load_balancer.loadbalancer", "options"),
					resource.TestCheckNoResourceAttr("data.stackit_application_load_balancer.loadbalancer", "labels"),
					resource.TestCheckNoResourceAttr("data.stackit_application_load_balancer.loadbalancer", "errors"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_application_load_balancer.loadbalancer", "project_id",
						"stackit_application_load_balancer.loadbalancer", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_application_load_balancer.loadbalancer", "region",
						"stackit_application_load_balancer.loadbalancer", "region",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_application_load_balancer.loadbalancer", "name",
						"stackit_application_load_balancer.loadbalancer", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_application_load_balancer.loadbalancer", "plan_id",
						"stackit_application_load_balancer.loadbalancer", "plan_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_application_load_balancer.loadbalancer", "external_address",
						"data.stackit_application_load_balancer.loadbalancer", "external_address",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_application_load_balancer.loadbalancer", "target_security_group.id",
						"data.stackit_application_load_balancer.loadbalancer", "target_security_group.id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_application_load_balancer.loadbalancer", "target_security_group.name",
						"data.stackit_application_load_balancer.loadbalancer", "target_security_group.name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.id",
						"data.stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.name",
						"data.stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_application_load_balancer.loadbalancer", "version",
						"data.stackit_application_load_balancer.loadbalancer", "version",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_application_load_balancer.loadbalancer", "status",
						"data.stackit_application_load_balancer.loadbalancer", "status",
					),
				)},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_application_load_balancer.loadbalancer",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_application_load_balancer.loadbalancer"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_application_load_balancer.loadbalancer")
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
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: configVarsMinUpdated(),
				Config:          testutil.ALBProviderConfig() + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMin["loadbalancer_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(configVarsMinUpdated()["target_pool_port"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccALBResourceMax(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckALBDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.ALBProviderConfig() + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance resource
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMax["loadbalancer_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMax["plan_id"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "labels.key1", testutil.ConvertConfigVariable(testConfigVarsMax["labels_value_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "labels.key2", testutil.ConvertConfigVariable(testConfigVarsMax["labels_value_2"])),
					resource.TestCheckTypeSetElemNestedAttrs("stackit_application_load_balancer.loadbalancer", "networks.*", map[string]string{"role": testutil.ConvertConfigVariable(testConfigVarsMax["network_role_listeners"])}),
					resource.TestCheckTypeSetElemNestedAttrs("stackit_application_load_balancer.loadbalancer", "networks.*", map[string]string{"role": testutil.ConvertConfigVariable(testConfigVarsMax["network_role_targets"])}),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "disable_target_security_group_assignment", testutil.ConvertConfigVariable(testConfigVarsMax["disable_security_group_assignment"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(testConfigVarsMax["listener_port_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.host", testutil.ConvertConfigVariable(testConfigVarsMax["host_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.web_socket", testutil.ConvertConfigVariable(testConfigVarsMax["web_socket"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["query_parameters_name_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.0.exact_match", testutil.ConvertConfigVariable(testConfigVarsMax["query_parameters_exact_match_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.1.name", testutil.ConvertConfigVariable(testConfigVarsMax["query_parameters_name_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.1.exact_match", testutil.ConvertConfigVariable(testConfigVarsMax["query_parameters_exact_match_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["headers_name_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.0.exact_match", testutil.ConvertConfigVariable(testConfigVarsMax["headers_exact_match_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.1.name", testutil.ConvertConfigVariable(testConfigVarsMax["headers_name_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.1.exact_match", testutil.ConvertConfigVariable(testConfigVarsMax["headers_exact_match_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.2.name", testutil.ConvertConfigVariable(testConfigVarsMax["headers_name_3"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMax["path_prefix_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.1.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMax["path_prefix_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.1.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.1.host", testutil.ConvertConfigVariable(testConfigVarsMax["host_3"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.1.rules.0.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMax["path_prefix_3"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.1.rules.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_3"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(testConfigVarsMax["protocol_http"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.1.port", testutil.ConvertConfigVariable(testConfigVarsMax["listener_port_4"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.1.http.hosts.0.host", testutil.ConvertConfigVariable(testConfigVarsMax["host_4"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.1.http.hosts.0.rules.0.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMax["path_prefix_4"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.1.http.hosts.0.rules.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_4"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.1.protocol", testutil.ConvertConfigVariable(testConfigVarsMax["protocol_http"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_port_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.interval", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_interval"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.interval_jitter", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_interval_jitter"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.timeout", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_timeout"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.healthy_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_healthy_threshold"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.unhealthy_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_unhealthy_threshold"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.http_health_checks.ok_status.0", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_http_ok_status_200"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.http_health_checks.ok_status.1", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_http_ok_status_201"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.http_health_checks.path", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_http_path"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.tls_config.enabled", testutil.ConvertConfigVariable(testConfigVarsMax["tls_config_enabled"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.tls_config.skip_certificate_validation", testutil.ConvertConfigVariable(testConfigVarsMax["tls_config_skip"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.tls_config.custom_ca", testutil.ConvertConfigVariable(testConfigVarsMax["tls_config_custom_ca"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.1.name", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.1.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_port_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.1.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.2.name", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_3"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.2.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_port_3"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.2.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.3.name", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_4"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.3.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_port_4"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.3.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "options.private_network_only", testutil.ConvertConfigVariable(testConfigVarsMax["private_network_only"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "options.ephemeral_address", testutil.ConvertConfigVariable(testConfigVarsMax["ephemeral_address"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "options.access_control.allowed_source_ranges.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "options.observability.logs.push_url", testutil.ConvertConfigVariable(testConfigVarsMax["observability_logs_push_url"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "options.observability.metrics.push_url", testutil.ConvertConfigVariable(testConfigVarsMax["observability_metrics_push_url"])),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "networks.1.network_id"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_pools.1.targets.0.ip"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_pools.2.targets.0.ip"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_pools.3.targets.0.ip"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_security_group.id"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "target_security_group.name"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.id"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.name"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttrSet("stackit_application_load_balancer.loadbalancer", "version"),
					resource.TestCheckNoResourceAttr("stackit_application_load_balancer.loadbalancer", "errors"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
						%s

						data "stackit_application_load_balancer" "loadbalancer" {
							project_id     = stackit_application_load_balancer.loadbalancer.project_id
							name    = stackit_application_load_balancer.loadbalancer.name
						}
						`,
					testutil.ALBProviderConfig()+resourceMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMax["loadbalancer_name"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMax["plan_id"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "labels.key1", testutil.ConvertConfigVariable(testConfigVarsMax["labels_value_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "labels.key2", testutil.ConvertConfigVariable(testConfigVarsMax["labels_value_2"])),
					resource.TestCheckTypeSetElemNestedAttrs("data.stackit_application_load_balancer.loadbalancer", "networks.*", map[string]string{"role": testutil.ConvertConfigVariable(testConfigVarsMax["network_role_listeners"])}),
					resource.TestCheckTypeSetElemNestedAttrs("data.stackit_application_load_balancer.loadbalancer", "networks.*", map[string]string{"role": testutil.ConvertConfigVariable(testConfigVarsMax["network_role_targets"])}),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "disable_target_security_group_assignment", testutil.ConvertConfigVariable(testConfigVarsMax["disable_security_group_assignment"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.port", testutil.ConvertConfigVariable(testConfigVarsMax["listener_port_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.host", testutil.ConvertConfigVariable(testConfigVarsMax["host_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.web_socket", testutil.ConvertConfigVariable(testConfigVarsMax["web_socket"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["query_parameters_name_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.0.exact_match", testutil.ConvertConfigVariable(testConfigVarsMax["query_parameters_exact_match_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.1.name", testutil.ConvertConfigVariable(testConfigVarsMax["query_parameters_name_2"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.1.exact_match", testutil.ConvertConfigVariable(testConfigVarsMax["query_parameters_exact_match_2"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["headers_name_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.0.exact_match", testutil.ConvertConfigVariable(testConfigVarsMax["headers_exact_match_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.1.name", testutil.ConvertConfigVariable(testConfigVarsMax["headers_name_2"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.1.exact_match", testutil.ConvertConfigVariable(testConfigVarsMax["headers_exact_match_2"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.2.name", testutil.ConvertConfigVariable(testConfigVarsMax["headers_name_3"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMax["path_prefix_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.1.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMax["path_prefix_2"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.1.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_2"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.1.host", testutil.ConvertConfigVariable(testConfigVarsMax["host_3"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.1.rules.0.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMax["path_prefix_3"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.1.rules.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_3"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.0.protocol", testutil.ConvertConfigVariable(testConfigVarsMax["protocol_http"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.1.port", testutil.ConvertConfigVariable(testConfigVarsMax["listener_port_4"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.1.http.hosts.0.host", testutil.ConvertConfigVariable(testConfigVarsMax["host_4"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.1.http.hosts.0.rules.0.path.prefix", testutil.ConvertConfigVariable(testConfigVarsMax["path_prefix_4"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.1.http.hosts.0.rules.0.target_pool", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_4"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "listeners.1.protocol", testutil.ConvertConfigVariable(testConfigVarsMax["protocol_http"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_port_1"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.interval", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_interval"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.interval_jitter", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_interval_jitter"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.timeout", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_timeout"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.healthy_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_healthy_threshold"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.unhealthy_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_unhealthy_threshold"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.http_health_checks.ok_status.0", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_http_ok_status_200"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.http_health_checks.ok_status.1", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_http_ok_status_201"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.http_health_checks.path", testutil.ConvertConfigVariable(testConfigVarsMax["ahc_http_path"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.tls_config.enabled", testutil.ConvertConfigVariable(testConfigVarsMax["tls_config_enabled"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.tls_config.skip_certificate_validation", testutil.ConvertConfigVariable(testConfigVarsMax["tls_config_skip"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.tls_config.custom_ca", testutil.ConvertConfigVariable(testConfigVarsMax["tls_config_custom_ca"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.1.name", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_2"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.1.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_port_2"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.1.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.2.name", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_3"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.2.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_port_3"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.2.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.3.name", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_name_4"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.3.target_port", testutil.ConvertConfigVariable(testConfigVarsMax["target_pool_port_4"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "target_pools.3.targets.0.display_name", testutil.ConvertConfigVariable(testConfigVarsMax["target_display_name"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "options.private_network_only", testutil.ConvertConfigVariable(testConfigVarsMax["private_network_only"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "options.ephemeral_address", testutil.ConvertConfigVariable(testConfigVarsMax["ephemeral_address"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "options.access_control.allowed_source_ranges.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "options.observability.logs.push_url", testutil.ConvertConfigVariable(testConfigVarsMax["observability_logs_push_url"])),
					resource.TestCheckResourceAttr("data.stackit_application_load_balancer.loadbalancer", "options.observability.metrics.push_url", testutil.ConvertConfigVariable(testConfigVarsMax["observability_metrics_push_url"])),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "networks.1.network_id"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "options.observability.logs.credentials_ref"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "options.observability.metrics.credentials_ref"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "target_pools.1.targets.0.ip"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "target_pools.2.targets.0.ip"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "target_pools.3.targets.0.ip"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "target_security_group.id"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "target_security_group.name"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.id"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "load_balancer_security_group.name"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "external_address"),
					resource.TestCheckResourceAttrSet("data.stackit_application_load_balancer.loadbalancer", "version"),
					resource.TestCheckNoResourceAttr("data.stackit_application_load_balancer.loadbalancer", "errors"),
				)},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_application_load_balancer.loadbalancer",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_application_load_balancer.loadbalancer"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_application_load_balancer.loadbalancer")
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
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: configVarsMaxUpdated(),
				Config:          testutil.ALBProviderConfig() + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "name", testutil.ConvertConfigVariable(testConfigVarsMax["loadbalancer_name"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "options.ephemeral_address", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ephemeral_address"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.host", testutil.ConvertConfigVariable(configVarsMaxUpdated()["host_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.web_socket", testutil.ConvertConfigVariable(configVarsMaxUpdated()["web_socket"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.0.name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["query_parameters_name_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.0.exact_match", testutil.ConvertConfigVariable(configVarsMaxUpdated()["query_parameters_exact_match_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.1.name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["query_parameters_name_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.query_parameters.1.exact_match", testutil.ConvertConfigVariable(configVarsMaxUpdated()["query_parameters_exact_match_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.0.name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["headers_name_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.0.exact_match", testutil.ConvertConfigVariable(configVarsMaxUpdated()["headers_exact_match_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.1.name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["headers_name_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.1.exact_match", testutil.ConvertConfigVariable(configVarsMaxUpdated()["headers_exact_match_2"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "listeners.0.http.hosts.0.rules.0.headers.2.name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["headers_name_3"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.target_port", testutil.ConvertConfigVariable(configVarsMaxUpdated()["target_pool_port_1"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.timeout", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ahc_timeout"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.healthy_threshold", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ahc_healthy_threshold"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.http_health_checks.ok_status.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ahc_http_ok_status_200"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "target_pools.0.active_health_check.http_health_checks.ok_status.1", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ahc_http_ok_status_201"])),
					resource.TestCheckResourceAttr("stackit_application_load_balancer.loadbalancer", "options.access_control.allowed_source_ranges.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["acl"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckALBDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *alb.APIClient
	var err error
	if testutil.ALBCustomEndpoint == "" {
		client, err = alb.NewAPIClient()
	} else {
		client, err = alb.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.ALBCustomEndpoint),
		)
	}
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

	loadbalancersResp, err := client.ListLoadBalancers(ctx, testutil.ProjectId, region).Execute()
	if err != nil {
		return fmt.Errorf("getting loadbalancersResp: %w", err)
	}

	if loadbalancersResp.LoadBalancers == nil || (loadbalancersResp.LoadBalancers != nil && len(*loadbalancersResp.LoadBalancers) == 0) {
		fmt.Print("No load balancers found for project \n")
		return nil
	}

	items := *loadbalancersResp.LoadBalancers
	for i := range items {
		if items[i].Name == nil {
			continue
		}
		if utils.Contains(loadbalancersToDestroy, *items[i].Name) {
			_, err := client.DeleteLoadBalancerExecute(ctx, testutil.ProjectId, region, *items[i].Name)
			if err != nil {
				return fmt.Errorf("destroying load balancer %s during CheckDestroy: %w", *items[i].Name, err)
			}
			_, err = wait.DeleteLoadbalancerWaitHandler(ctx, client, testutil.ProjectId, region, *items[i].Name).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying load balancer %s during CheckDestroy: waiting for deletion %w", *items[i].Name, err)
			}
		}
	}
	return nil
}
