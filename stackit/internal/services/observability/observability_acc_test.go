package observability_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/stackit-sdk-go/services/observability/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"

	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
)

//go:embed testdata/resource-min.tf
var resourceMinConfig string

//go:embed testdata/resource-max.tf
var resourceMaxConfig string

// To prevent conversion issues
var alert_rule_expression = "sum(kube_pod_status_phase{phase=\"Running\"}) > 0"
var logalertgroup_expression = "sum(rate({namespace=\"example\"} |= \"Simulated error message\" [1m])) > 0"
var alert_rule_expression_updated = "sum(kube_pod_status_phase{phase=\"Error\"}) > 0"
var logalertgroup_expression_updated = "sum(rate({namespace=\"example\"} |= \"Another error message\" [1m])) > 0"

var testConfigVarsMin = config.Variables{
	"project_id":                config.StringVariable(testutil.ProjectId),
	"alertgroup_name":           config.StringVariable(fmt.Sprintf("tf-acc-ag%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"alert_rule_name":           config.StringVariable("alert1"),
	"alert_rule_expression":     config.StringVariable(alert_rule_expression),
	"instance_name":             config.StringVariable(fmt.Sprintf("tf-acc-i%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"plan_name":                 config.StringVariable("Observability-Medium-EU01"),
	"logalertgroup_name":        config.StringVariable(fmt.Sprintf("tf-acc-lag%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"logalertgroup_alert":       config.StringVariable("alert1"),
	"logalertgroup_expression":  config.StringVariable(logalertgroup_expression),
	"scrapeconfig_name":         config.StringVariable(fmt.Sprintf("tf-acc-sc%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"scrapeconfig_metrics_path": config.StringVariable("/metrics"),
	"scrapeconfig_targets_url":  config.StringVariable("www.y97xyrrocx2gsxx.de"),
}

var testConfigVarsMax = config.Variables{
	"project_id":                 config.StringVariable(testutil.ProjectId),
	"alertgroup_name":            config.StringVariable(fmt.Sprintf("tf-acc-ag%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"alert_rule_name":            config.StringVariable("alert1"),
	"alert_rule_expression":      config.StringVariable(alert_rule_expression),
	"instance_name":              config.StringVariable(fmt.Sprintf("tf-acc-i%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"plan_name":                  config.StringVariable("Observability-Medium-EU01"),
	"logalertgroup_name":         config.StringVariable(fmt.Sprintf("tf-acc-lag%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"logalertgroup_alert":        config.StringVariable("alert1"),
	"logalertgroup_expression":   config.StringVariable(logalertgroup_expression),
	"scrapeconfig_name":          config.StringVariable(fmt.Sprintf("tf-acc-sc%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"scrapeconfig_metrics_path":  config.StringVariable("/metrics"),
	"scrapeconfig_targets_url_1": config.StringVariable("www.y97xyrrocx2gsxx.de"),
	"scrapeconfig_targets_url_2": config.StringVariable("f6zkn8gzeigwanh.de"),
	// alert group
	"alert_for_time":   config.StringVariable("60s"),
	"alert_label":      config.StringVariable("label1"),
	"alert_annotation": config.StringVariable("annotation1"),
	"alert_interval":   config.StringVariable("5h"),
	// max instance
	"logs_retention_days":                    config.StringVariable("30"),
	"traces_retention_days":                  config.StringVariable("30"),
	"metrics_retention_days":                 config.StringVariable("90"),
	"metrics_retention_days_5m_downsampling": config.StringVariable("90"),
	"metrics_retention_days_1h_downsampling": config.StringVariable("90"),
	"instance_acl_1":                         config.StringVariable("1.2.3.4/32"),
	"instance_acl_2":                         config.StringVariable("111.222.111.222/32"),
	"receiver_name":                          config.StringVariable("OpsGenieReceiverInfo"),
	"auth_identity":                          config.StringVariable("aa@bb.ccc"),
	"auth_password":                          config.StringVariable("password"),
	"auth_username":                          config.StringVariable("username"),
	"email_from":                             config.StringVariable("aa@bb.ccc"),
	"email_send_resolved":                    config.StringVariable("true"),
	"smart_host":                             config.StringVariable("smtp.gmail.com:587"),
	"email_to":                               config.StringVariable("bb@bb.ccc"),
	"opsgenie_api_key":                       config.StringVariable("example-api-key"),
	"opsgenie_api_tags":                      config.StringVariable("observability-alert"),
	"opsgenie_api_url":                       config.StringVariable("https://api.eu.opsgenie.com"),
	"opsgenie_priority":                      config.StringVariable("P3"),
	"opsgenie_send_resolved":                 config.StringVariable("false"),
	"webhook_configs_url":                    config.StringVariable("https://example.com"),
	"ms_teams":                               config.StringVariable("true"),
	"google_chat":                            config.StringVariable("false"),
	"webhook_configs_send_resolved":          config.StringVariable("false"),
	"group_by":                               config.StringVariable("alertname"),
	"group_interval":                         config.StringVariable("10m"),
	"group_wait":                             config.StringVariable("1m"),
	"repeat_interval":                        config.StringVariable("1h"),
	"resolve_timeout":                        config.StringVariable("5m"),
	"smtp_auth_identity":                     config.StringVariable("aa@bb.ccc"),
	"smtp_auth_password":                     config.StringVariable("password"),
	"smtp_auth_username":                     config.StringVariable("username"),
	"smtp_from":                              config.StringVariable("aa@bb.ccc"),
	"smtp_smart_host":                        config.StringVariable("smtp.gmail.com:587"),
	"match":                                  config.StringVariable("alert1"),
	"match_regex":                            config.StringVariable("alert1"),
	"matchers":                               config.StringVariable("instance =~ \".*\""),
	"continue":                               config.StringVariable("true"),
	// credential
	"credential_description": config.StringVariable("This is a description for the test credential."),
	// logalertgroup
	"logalertgroup_for_time":   config.StringVariable("60s"),
	"logalertgroup_label":      config.StringVariable("label1"),
	"logalertgroup_annotation": config.StringVariable("annotation1"),
	"logalertgroup_interval":   config.StringVariable("5h"),
	// scrapeconfig
	"scrapeconfig_label":             config.StringVariable("label1"),
	"scrapeconfig_interval":          config.StringVariable("4m"),
	"scrapeconfig_limit":             config.StringVariable("7"),
	"scrapeconfig_enable_url_params": config.StringVariable("false"),
	"scrapeconfig_scheme":            config.StringVariable("https"),
	"scrapeconfig_timeout":           config.StringVariable("2m"),
	"scrapeconfig_auth_username":     config.StringVariable("username"),
	"scrapeconfig_auth_password":     config.StringVariable("password"),
}

func configVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(tempConfig, testConfigVarsMin)
	tempConfig["alert_rule_name"] = config.StringVariable("alert1-updated")
	return tempConfig
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMax))
	maps.Copy(tempConfig, testConfigVarsMax)
	tempConfig["plan_name"] = config.StringVariable("Observability-Large-EU01")
	tempConfig["alert_interval"] = config.StringVariable("1h")
	tempConfig["alert_rule_expression"] = config.StringVariable(alert_rule_expression_updated)
	tempConfig["logalertgroup_interval"] = config.StringVariable("1h")
	tempConfig["logalertgroup_expression"] = config.StringVariable(logalertgroup_expression_updated)
	tempConfig["webhook_configs_url"] = config.StringVariable("https://chat.googleapis.com/api")
	tempConfig["ms_teams"] = config.StringVariable("false")
	tempConfig["google_chat"] = config.StringVariable("true")
	tempConfig["matchers"] = config.StringVariable("instance =~ \"my.*\"")
	return tempConfig
}

func TestAccResourceMin(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckObservabilityDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.ObservabilityProviderConfig() + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["instance_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "is_updatable"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_public_read_access"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_initial_admin_user"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_initial_admin_password"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "metrics_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "metrics_push_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "targets_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "alerting_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "logs_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "logs_push_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "jaeger_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "jaeger_ui_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "otlp_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "zipkin_spans_url"),

					// scrape config data
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "project_id",
						"stackit_observability_scrapeconfig.scrapeconfig", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_scrapeconfig.scrapeconfig", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "name", testutil.ConvertConfigVariable(testConfigVarsMin["scrapeconfig_name"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.#", "1"),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "metrics_path", testutil.ConvertConfigVariable(testConfigVarsMin["scrapeconfig_metrics_path"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.0", testutil.ConvertConfigVariable(testConfigVarsMin["scrapeconfig_targets_url"])),

					// credentials
					resource.TestCheckResourceAttr("stackit_observability_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_credential.credential", "instance_id",
					),
					resource.TestCheckNoResourceAttr("stackit_observability_credential.credential", "description"),
					resource.TestCheckResourceAttrSet("stackit_observability_credential.credential", "username"),
					resource.TestCheckResourceAttrSet("stackit_observability_credential.credential", "password"),

					// alertgroup
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_alertgroup.alertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMin["alertgroup_name"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMin["alert_rule_name"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.expression", alert_rule_expression),

					// logalertgroup
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_logalertgroup.logalertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMin["logalertgroup_name"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMin["logalertgroup_alert"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.expression", logalertgroup_expression),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
					%s

					data "stackit_observability_instance" "instance" {
					  	project_id  = stackit_observability_instance.instance.project_id
					  	instance_id = stackit_observability_instance.instance.instance_id
					}

					data "stackit_observability_scrapeconfig" "scrapeconfig" {
						project_id  = stackit_observability_scrapeconfig.scrapeconfig.project_id
					  	instance_id = stackit_observability_scrapeconfig.scrapeconfig.instance_id
					  	name        = stackit_observability_scrapeconfig.scrapeconfig.name
					}

					data "stackit_observability_alertgroup" "alertgroup" {
					  project_id  = stackit_observability_alertgroup.alertgroup.project_id
					  instance_id = stackit_observability_alertgroup.alertgroup.instance_id
					  name        = stackit_observability_alertgroup.alertgroup.name
					}

					data "stackit_observability_logalertgroup" "logalertgroup" {
					  project_id  = stackit_observability_logalertgroup.logalertgroup.project_id
					  instance_id = stackit_observability_logalertgroup.logalertgroup.instance_id
					  name        = stackit_observability_logalertgroup.logalertgroup.name
					}
					`,
					testutil.ObservabilityProviderConfig()+resourceMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["instance_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),

					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "project_id",
						"data.stackit_observability_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"data.stackit_observability_instance.instance", "instance_id",
					),

					// scrape config data
					resource.TestCheckResourceAttrPair(
						"stackit_observability_scrapeconfig.scrapeconfig", "project_id",
						"data.stackit_observability_scrapeconfig.scrapeconfig", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_scrapeconfig.scrapeconfig", "instance_id",
						"data.stackit_observability_scrapeconfig.scrapeconfig", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_scrapeconfig.scrapeconfig", "name",
						"data.stackit_observability_scrapeconfig.scrapeconfig", "name",
					),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "name", testutil.ConvertConfigVariable(testConfigVarsMin["scrapeconfig_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "metrics_path", testutil.ConvertConfigVariable(testConfigVarsMin["scrapeconfig_metrics_path"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.0", testutil.ConvertConfigVariable(testConfigVarsMin["scrapeconfig_targets_url"])),

					// alertgroup
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_alertgroup.alertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMin["alertgroup_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMin["alert_rule_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "rules.0.expression", alert_rule_expression),

					// logalertgroup
					resource.TestCheckResourceAttr("data.stackit_observability_logalertgroup.logalertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_logalertgroup.logalertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_observability_logalertgroup.logalertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMin["logalertgroup_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_logalertgroup.logalertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMin["logalertgroup_alert"])),
					resource.TestCheckResourceAttr("data.stackit_observability_logalertgroup.logalertgroup", "rules.0.expression", logalertgroup_expression),
				),
			},
			// Import 1
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_observability_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_observability_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_observability_instance.instance")
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
			// Import 2
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_observability_scrapeconfig.scrapeconfig",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_observability_scrapeconfig.scrapeconfig"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_observability_scrapeconfig.scrapeconfig")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Import 3
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_observability_alertgroup.alertgroup",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_observability_alertgroup.alertgroup"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_observability_alertgroup.alertgroup")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Import 4
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_observability_logalertgroup.logalertgroup",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_observability_logalertgroup.logalertgroup"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_observability_logalertgroup.logalertgroup")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: configVarsMinUpdated(),
				Config:          testutil.ObservabilityProviderConfig() + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["instance_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "is_updatable"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_public_read_access"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_initial_admin_user"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_initial_admin_password"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "metrics_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "metrics_push_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "targets_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "alerting_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "logs_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "logs_push_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "jaeger_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "jaeger_ui_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "otlp_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "zipkin_spans_url"),

					// scrape config data
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "project_id",
						"stackit_observability_scrapeconfig.scrapeconfig", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_scrapeconfig.scrapeconfig", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "name", testutil.ConvertConfigVariable(testConfigVarsMin["scrapeconfig_name"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.#", "1"),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "metrics_path", testutil.ConvertConfigVariable(testConfigVarsMin["scrapeconfig_metrics_path"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.0", testutil.ConvertConfigVariable(testConfigVarsMin["scrapeconfig_targets_url"])),

					// credentials
					resource.TestCheckResourceAttr("stackit_observability_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_credential.credential", "instance_id",
					),
					resource.TestCheckNoResourceAttr("stackit_observability_credential.credential", "description"),
					resource.TestCheckResourceAttrSet("stackit_observability_credential.credential", "username"),
					resource.TestCheckResourceAttrSet("stackit_observability_credential.credential", "password"),

					// alertgroup
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_alertgroup.alertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMin["alertgroup_name"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.alert", testutil.ConvertConfigVariable(configVarsMinUpdated()["alert_rule_name"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.expression", alert_rule_expression),

					// logalertgroup
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_logalertgroup.logalertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMin["logalertgroup_name"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMin["logalertgroup_alert"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.expression", logalertgroup_expression),
				),
			},
		},
	})
}

func TestAccResourceMax(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckObservabilityDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.ObservabilityProviderConfig() + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["instance_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMax["plan_name"])),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "is_updatable"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_public_read_access"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_initial_admin_user"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_initial_admin_password"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "metrics_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "metrics_push_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "targets_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "alerting_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "logs_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "logs_push_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "jaeger_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "jaeger_ui_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "otlp_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "zipkin_spans_url"),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "logs_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["logs_retention_days"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "traces_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["traces_retention_days"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "metrics_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_retention_days"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "metrics_retention_days_5m_downsampling", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_retention_days_5m_downsampling"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "metrics_retention_days_1h_downsampling", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_retention_days_1h_downsampling"])),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["instance_acl_1"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "acl.1", testutil.ConvertConfigVariable(testConfigVarsMax["instance_acl_2"])),

					// alert config
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["receiver_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.auth_identity", testutil.ConvertConfigVariable(testConfigVarsMax["auth_identity"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.auth_password", testutil.ConvertConfigVariable(testConfigVarsMax["auth_password"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.auth_username", testutil.ConvertConfigVariable(testConfigVarsMax["auth_username"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.from", testutil.ConvertConfigVariable(testConfigVarsMax["email_from"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.smart_host", testutil.ConvertConfigVariable(testConfigVarsMax["smart_host"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.to", testutil.ConvertConfigVariable(testConfigVarsMax["email_to"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.api_key", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_key"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.tags", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_tags"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.api_url", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_url"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.priority", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_priority"])),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.webhooks_configs.0.url", testutil.ConvertConfigVariable(testConfigVarsMax["webhook_configs_url"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.webhooks_configs.0.ms_teams", testutil.ConvertConfigVariable(testConfigVarsMax["ms_teams"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.webhooks_configs.0.google_chat", testutil.ConvertConfigVariable(testConfigVarsMax["google_chat"])),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.group_by.0", testutil.ConvertConfigVariable(testConfigVarsMax["group_by"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.group_interval", testutil.ConvertConfigVariable(testConfigVarsMax["group_interval"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.group_wait", testutil.ConvertConfigVariable(testConfigVarsMax["group_wait"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.receiver", testutil.ConvertConfigVariable(testConfigVarsMax["receiver_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.repeat_interval", testutil.ConvertConfigVariable(testConfigVarsMax["repeat_interval"])),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.group_by.0", testutil.ConvertConfigVariable(testConfigVarsMax["group_by"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.group_interval", testutil.ConvertConfigVariable(testConfigVarsMax["group_interval"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.group_wait", testutil.ConvertConfigVariable(testConfigVarsMax["group_wait"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.receiver", testutil.ConvertConfigVariable(testConfigVarsMax["receiver_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.repeat_interval", testutil.ConvertConfigVariable(testConfigVarsMax["repeat_interval"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.continue", testutil.ConvertConfigVariable(testConfigVarsMax["continue"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.match.match1", testutil.ConvertConfigVariable(testConfigVarsMax["match"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.match_regex.match_regex1", testutil.ConvertConfigVariable(testConfigVarsMax["match_regex"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.matchers.0", testutil.ConvertConfigVariable(testConfigVarsMax["matchers"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.matchers.#", "1"),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.opsgenie_api_key", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_key"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.opsgenie_api_url", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_url"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.resolve_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["resolve_timeout"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_auth_identity", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_auth_identity"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_auth_password", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_auth_password"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_auth_username", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_auth_username"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_from", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_from"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_smart_host", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_smart_host"])),

					// scrape config data
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "project_id",
						"stackit_observability_scrapeconfig.scrapeconfig", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_scrapeconfig.scrapeconfig", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "name", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_name"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.#", "2"),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "metrics_path", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_metrics_path"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.0", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_targets_url_1"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.1", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_targets_url_2"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.labels.label1", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_label"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "scrape_interval", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_interval"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "sample_limit", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_limit"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "saml2.enable_url_parameters", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_enable_url_params"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "scheme", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_scheme"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "scrape_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_timeout"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "basic_auth.username", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_auth_username"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "basic_auth.password", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_auth_password"])),

					// credentials
					resource.TestCheckResourceAttr("stackit_observability_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_credential.credential", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_credential.credential", "description", testutil.ConvertConfigVariable(testConfigVarsMax["credential_description"])),
					resource.TestCheckResourceAttrSet("stackit_observability_credential.credential", "username"),
					resource.TestCheckResourceAttrSet("stackit_observability_credential.credential", "password"),

					// alertgroup
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_alertgroup.alertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMax["alertgroup_name"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMax["alert_rule_name"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.expression", alert_rule_expression),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.for", testutil.ConvertConfigVariable(testConfigVarsMax["alert_for_time"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.labels.label1", testutil.ConvertConfigVariable(testConfigVarsMax["alert_label"])),

					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.annotations.annotation1", testutil.ConvertConfigVariable(testConfigVarsMax["alert_annotation"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "interval", testutil.ConvertConfigVariable(testConfigVarsMax["alert_interval"])),

					// logalertgroup
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_logalertgroup.logalertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_name"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_alert"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.expression", logalertgroup_expression),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.for", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_for_time"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.labels.label1", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_label"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.annotations.annotation1", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_annotation"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "interval", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_interval"])),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
					%s

					data "stackit_observability_instance" "instance" {
					  	project_id  = stackit_observability_instance.instance.project_id
					  	instance_id = stackit_observability_instance.instance.instance_id
					}

					data "stackit_observability_scrapeconfig" "scrapeconfig" {
						project_id  = stackit_observability_scrapeconfig.scrapeconfig.project_id
					  	instance_id = stackit_observability_scrapeconfig.scrapeconfig.instance_id
					  	name        = stackit_observability_scrapeconfig.scrapeconfig.name
					}

					data "stackit_observability_alertgroup" "alertgroup" {
					  project_id  = stackit_observability_alertgroup.alertgroup.project_id
					  instance_id = stackit_observability_alertgroup.alertgroup.instance_id
					  name        = stackit_observability_alertgroup.alertgroup.name
					}

					data "stackit_observability_logalertgroup" "logalertgroup" {
					  project_id  = stackit_observability_logalertgroup.logalertgroup.project_id
					  instance_id = stackit_observability_logalertgroup.logalertgroup.instance_id
					  name        = stackit_observability_logalertgroup.logalertgroup.name
					}
					`,
					testutil.ObservabilityProviderConfig()+resourceMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["instance_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMax["plan_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "is_updatable"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "grafana_public_read_access"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "grafana_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "grafana_initial_admin_user"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "grafana_initial_admin_password"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "metrics_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "metrics_push_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "targets_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "alerting_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "logs_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "logs_push_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "jaeger_traces_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "jaeger_ui_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "otlp_traces_url"),
					resource.TestCheckResourceAttrSet("data.stackit_observability_instance.instance", "zipkin_spans_url"),

					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "logs_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["logs_retention_days"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "traces_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["traces_retention_days"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "metrics_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_retention_days"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "metrics_retention_days_5m_downsampling", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_retention_days_5m_downsampling"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "metrics_retention_days_1h_downsampling", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_retention_days_1h_downsampling"])),

					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "acl.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["instance_acl_1"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "acl.1", testutil.ConvertConfigVariable(testConfigVarsMax["instance_acl_2"])),
					// alert configdata.
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["receiver_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.auth_identity", testutil.ConvertConfigVariable(testConfigVarsMax["auth_identity"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.auth_password", testutil.ConvertConfigVariable(testConfigVarsMax["auth_password"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.auth_username", testutil.ConvertConfigVariable(testConfigVarsMax["auth_username"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.from", testutil.ConvertConfigVariable(testConfigVarsMax["email_from"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.smart_host", testutil.ConvertConfigVariable(testConfigVarsMax["smart_host"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.to", testutil.ConvertConfigVariable(testConfigVarsMax["email_to"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.api_key", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_key"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.tags", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_tags"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.api_url", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_url"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.priority", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_priority"])),

					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.webhooks_configs.0.url", testutil.ConvertConfigVariable(testConfigVarsMax["webhook_configs_url"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.webhooks_configs.0.ms_teams", testutil.ConvertConfigVariable(testConfigVarsMax["ms_teams"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.receivers.0.webhooks_configs.0.google_chat", testutil.ConvertConfigVariable(testConfigVarsMax["google_chat"])),

					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.group_by.0", testutil.ConvertConfigVariable(testConfigVarsMax["group_by"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.group_interval", testutil.ConvertConfigVariable(testConfigVarsMax["group_interval"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.group_wait", testutil.ConvertConfigVariable(testConfigVarsMax["group_wait"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.receiver", testutil.ConvertConfigVariable(testConfigVarsMax["receiver_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.repeat_interval", testutil.ConvertConfigVariable(testConfigVarsMax["repeat_interval"])),

					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.group_by.0", testutil.ConvertConfigVariable(testConfigVarsMax["group_by"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.group_interval", testutil.ConvertConfigVariable(testConfigVarsMax["group_interval"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.group_wait", testutil.ConvertConfigVariable(testConfigVarsMax["group_wait"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.receiver", testutil.ConvertConfigVariable(testConfigVarsMax["receiver_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.repeat_interval", testutil.ConvertConfigVariable(testConfigVarsMax["repeat_interval"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.continue", testutil.ConvertConfigVariable(testConfigVarsMax["continue"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.match.match1", testutil.ConvertConfigVariable(testConfigVarsMax["match"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.match_regex.match_regex1", testutil.ConvertConfigVariable(testConfigVarsMax["match_regex"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.matchers.0", testutil.ConvertConfigVariable(testConfigVarsMax["matchers"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.route.routes.0.matchers.#", "1"),

					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.global.opsgenie_api_key", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_key"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.global.opsgenie_api_url", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_url"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.global.resolve_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["resolve_timeout"])),
					resource.TestCheckResourceAttr("data.stackit_observability_instance.instance", "alert_config.global.smtp_from", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_from"])),

					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "project_id",
						"data.stackit_observability_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"data.stackit_observability_instance.instance", "instance_id",
					),
					// scrape config data
					resource.TestCheckResourceAttrPair(
						"stackit_observability_scrapeconfig.scrapeconfig", "project_id",
						"data.stackit_observability_scrapeconfig.scrapeconfig", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_scrapeconfig.scrapeconfig", "instance_id",
						"data.stackit_observability_scrapeconfig.scrapeconfig", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_scrapeconfig.scrapeconfig", "name",
						"data.stackit_observability_scrapeconfig.scrapeconfig", "name",
					),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "name", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "metrics_path", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_metrics_path"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.0", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_targets_url_1"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.1", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_targets_url_2"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "targets.0.labels.label1", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_label"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "scrape_interval", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_interval"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "sample_limit", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_limit"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "saml2.enable_url_parameters", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_enable_url_params"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "scheme", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_scheme"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "scrape_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_timeout"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "basic_auth.username", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_auth_username"])),
					resource.TestCheckResourceAttr("data.stackit_observability_scrapeconfig.scrapeconfig", "basic_auth.password", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_auth_password"])),

					// alertgroup
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"data.stackit_observability_alertgroup.alertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMax["alertgroup_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMax["alert_rule_name"])),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "rules.0.expression", alert_rule_expression),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "rules.0.for", testutil.ConvertConfigVariable(testConfigVarsMax["alert_for_time"])),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "rules.0.labels.label1", testutil.ConvertConfigVariable(testConfigVarsMax["alert_label"])),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "rules.0.annotations.annotation1", testutil.ConvertConfigVariable(testConfigVarsMax["alert_annotation"])),
					resource.TestCheckResourceAttr("data.stackit_observability_alertgroup.alertgroup", "interval", testutil.ConvertConfigVariable(testConfigVarsMax["alert_interval"])),

					// logalertgroup
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"data.stackit_observability_logalertgroup.logalertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_name"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_alert"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.expression", logalertgroup_expression),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.for", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_for_time"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.labels.label1", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_label"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.annotations.annotation1", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_annotation"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "interval", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_interval"])),
				),
			},
			// Import 1
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_observability_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_observability_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_observability_instance.instance")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, instanceId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"alert_config.global.smtp_auth_identity", "alert_config.global.smtp_auth_password", "alert_config.global.smtp_auth_username", "alert_config.global.smtp_smart_host"},
			},
			// Import 2
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_observability_scrapeconfig.scrapeconfig",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_observability_scrapeconfig.scrapeconfig"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_observability_scrapeconfig.scrapeconfig")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, name), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"alert_config.global.smtp_auth_identity", "alert_config.global.smtp_auth_password", "alert_config.global.smtp_auth_username", "alert_config.global.smtp_smart_host"},
			},
			// Import 3
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_observability_alertgroup.alertgroup",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_observability_alertgroup.alertgroup"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_observability_alertgroup.alertgroup")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, name), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"alert_config.global.smtp_auth_identity", "alert_config.global.smtp_auth_password", "alert_config.global.smtp_auth_username", "alert_config.global.smtp_smart_host"},
			},
			// Import 4
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_observability_logalertgroup.logalertgroup",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_observability_logalertgroup.logalertgroup"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_observability_logalertgroup.logalertgroup")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, name), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"alert_config.global.smtp_auth_identity", "alert_config.global.smtp_auth_password", "alert_config.global.smtp_auth_username", "alert_config.global.smtp_smart_host"},
			},
			// Update
			{
				ConfigVariables: configVarsMaxUpdated(),
				Config:          testutil.ObservabilityProviderConfig() + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["instance_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "plan_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["plan_name"])),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "is_updatable"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_public_read_access"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_initial_admin_user"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "grafana_initial_admin_password"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "metrics_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "metrics_push_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "targets_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "alerting_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "logs_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "logs_push_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "jaeger_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "jaeger_ui_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "otlp_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_observability_instance.instance", "zipkin_spans_url"),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "logs_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["logs_retention_days"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "traces_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["traces_retention_days"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "metrics_retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_retention_days"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "metrics_retention_days_5m_downsampling", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_retention_days_5m_downsampling"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "metrics_retention_days_1h_downsampling", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_retention_days_1h_downsampling"])),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["instance_acl_1"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "acl.1", testutil.ConvertConfigVariable(testConfigVarsMax["instance_acl_2"])),

					// alert config
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["receiver_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.auth_identity", testutil.ConvertConfigVariable(testConfigVarsMax["auth_identity"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.auth_password", testutil.ConvertConfigVariable(testConfigVarsMax["auth_password"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.auth_username", testutil.ConvertConfigVariable(testConfigVarsMax["auth_username"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.from", testutil.ConvertConfigVariable(testConfigVarsMax["email_from"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.smart_host", testutil.ConvertConfigVariable(testConfigVarsMax["smart_host"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.email_configs.0.to", testutil.ConvertConfigVariable(testConfigVarsMax["email_to"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.api_key", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_key"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.tags", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_tags"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.api_url", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_url"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.opsgenie_configs.0.priority", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_priority"])),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.webhooks_configs.0.url", testutil.ConvertConfigVariable(configVarsMaxUpdated()["webhook_configs_url"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.webhooks_configs.0.ms_teams", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ms_teams"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.receivers.0.webhooks_configs.0.google_chat", testutil.ConvertConfigVariable(configVarsMaxUpdated()["google_chat"])),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.group_by.0", testutil.ConvertConfigVariable(testConfigVarsMax["group_by"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.group_interval", testutil.ConvertConfigVariable(testConfigVarsMax["group_interval"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.group_wait", testutil.ConvertConfigVariable(testConfigVarsMax["group_wait"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.receiver", testutil.ConvertConfigVariable(testConfigVarsMax["receiver_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.repeat_interval", testutil.ConvertConfigVariable(testConfigVarsMax["repeat_interval"])),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.group_by.0", testutil.ConvertConfigVariable(testConfigVarsMax["group_by"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.group_interval", testutil.ConvertConfigVariable(testConfigVarsMax["group_interval"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.group_wait", testutil.ConvertConfigVariable(testConfigVarsMax["group_wait"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.receiver", testutil.ConvertConfigVariable(testConfigVarsMax["receiver_name"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.repeat_interval", testutil.ConvertConfigVariable(testConfigVarsMax["repeat_interval"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.continue", testutil.ConvertConfigVariable(testConfigVarsMax["continue"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.match.match1", testutil.ConvertConfigVariable(testConfigVarsMax["match"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.match_regex.match_regex1", testutil.ConvertConfigVariable(testConfigVarsMax["match_regex"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.matchers.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["matchers"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.route.routes.0.matchers.#", "1"),

					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.opsgenie_api_key", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_key"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.opsgenie_api_url", testutil.ConvertConfigVariable(testConfigVarsMax["opsgenie_api_url"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.resolve_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["resolve_timeout"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_auth_identity", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_auth_identity"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_auth_password", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_auth_password"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_auth_username", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_auth_username"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_from", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_from"])),
					resource.TestCheckResourceAttr("stackit_observability_instance.instance", "alert_config.global.smtp_smart_host", testutil.ConvertConfigVariable(testConfigVarsMax["smtp_smart_host"])),

					// scrape config data
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "project_id",
						"stackit_observability_scrapeconfig.scrapeconfig", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_scrapeconfig.scrapeconfig", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "name", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_name"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.#", "2"),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "metrics_path", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_metrics_path"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.0", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_targets_url_1"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.urls.1", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_targets_url_2"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "targets.0.labels.label1", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_label"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "scrape_interval", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_interval"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "sample_limit", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_limit"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "saml2.enable_url_parameters", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_enable_url_params"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "scheme", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_scheme"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "scrape_timeout", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_timeout"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "basic_auth.username", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_auth_username"])),
					resource.TestCheckResourceAttr("stackit_observability_scrapeconfig.scrapeconfig", "basic_auth.password", testutil.ConvertConfigVariable(testConfigVarsMax["scrapeconfig_auth_password"])),

					// credentials
					resource.TestCheckResourceAttr("stackit_observability_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_credential.credential", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_credential.credential", "description", testutil.ConvertConfigVariable(testConfigVarsMax["credential_description"])),
					resource.TestCheckResourceAttrSet("stackit_observability_credential.credential", "username"),
					resource.TestCheckResourceAttrSet("stackit_observability_credential.credential", "password"),

					// alertgroup
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_alertgroup.alertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMax["alertgroup_name"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMax["alert_rule_name"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.expression", alert_rule_expression_updated),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.for", testutil.ConvertConfigVariable(testConfigVarsMax["alert_for_time"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.labels.label1", testutil.ConvertConfigVariable(testConfigVarsMax["alert_label"])),

					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "rules.0.annotations.annotation1", testutil.ConvertConfigVariable(testConfigVarsMax["alert_annotation"])),
					resource.TestCheckResourceAttr("stackit_observability_alertgroup.alertgroup", "interval", testutil.ConvertConfigVariable(configVarsMaxUpdated()["alert_interval"])),

					// logalertgroup
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_observability_logalertgroup.logalertgroup", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "name", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_name"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.alert", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_alert"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.expression", logalertgroup_expression_updated),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.for", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_for_time"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.labels.label1", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_label"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "rules.0.annotations.annotation1", testutil.ConvertConfigVariable(testConfigVarsMax["logalertgroup_annotation"])),
					resource.TestCheckResourceAttr("stackit_observability_logalertgroup.logalertgroup", "interval", testutil.ConvertConfigVariable(configVarsMaxUpdated()["logalertgroup_interval"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckObservabilityDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *observability.APIClient
	var err error
	if testutil.ObservabilityCustomEndpoint == "" {
		client, err = observability.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = observability.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.ObservabilityCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_observability_instance" {
			continue
		}
		// instance terraform ID: = "[project_id],[instance_id],[name]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.ListInstances(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	instances := *instancesResp.Instances
	for i := range instances {
		if utils.Contains(instancesToDestroy, *instances[i].Id) {
			if *instances[i].Status != observability.PROJECTINSTANCEFULLSTATUS_DELETE_SUCCEEDED {
				_, err := client.DeleteInstanceExecute(ctx, testutil.ProjectId, *instances[i].Id)
				if err != nil {
					return fmt.Errorf("destroying instance %s during CheckDestroy: %w", *instances[i].Id, err)
				}
				_, err = wait.DeleteInstanceWaitHandler(ctx, client, testutil.ProjectId, *instances[i].Id).WaitWithContext(ctx)
				if err != nil {
					return fmt.Errorf("destroying instance %s during CheckDestroy: waiting for deletion %w", *instances[i].Id, err)
				}
			}
		}
	}
	return nil
}
