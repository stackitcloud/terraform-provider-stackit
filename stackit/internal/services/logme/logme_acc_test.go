package logme_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	core_config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/logme"
	"github.com/stackitcloud/stackit-sdk-go/services/logme/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-min.tf
	resourceMinConfig string

	//go:embed testdata/resource-max.tf
	resourceMaxConfig string
)

var (
	minTestName = testutil.ResourceNameWithDateTime("logme-min")
	maxTestName = testutil.ResourceNameWithDateTime("logme-max")
)

// Instance resource data
var testConfigVarsMin = config.Variables{
	"project_id":    config.StringVariable(testutil.ProjectId),
	"name":          config.StringVariable(minTestName),
	"plan_id":       config.StringVariable("7a54492c-8a2e-4d3c-b6c2-a4f20cb65912"), // stackit-logme2-1.4.10-single
	"plan_name":     config.StringVariable("stackit-logme2-1.4.10-single"),
	"logme_version": config.StringVariable("2"),
}

var testConfigVarsMax = config.Variables{
	"project_id":    config.StringVariable(testutil.ProjectId),
	"name":          config.StringVariable(maxTestName),
	"plan_id":       config.StringVariable("7a54492c-8a2e-4d3c-b6c2-a4f20cb65912"), // stackit-logme2-1.4.10-single
	"logme_version": config.StringVariable("2"),

	"plan_name":                       config.StringVariable("stackit-logme2-1.4.10-single"),
	"params_enable_monitoring":        config.BoolVariable(false),
	"params_fluentd_tcp":              config.IntegerVariable(4),
	"params_fluentd_tls":              config.IntegerVariable(1),
	"params_fluentd_tls_ciphers":      config.StringVariable("ALL:!aNULL:!eNULL:!SSLv2"),
	"params_fluentd_tls_max_version":  config.StringVariable("TLS1_3"),
	"params_fluentd_tls_min_version":  config.StringVariable("TLS1_1"),
	"params_fluentd_tls_version":      config.StringVariable("TLS1_2"),
	"params_fluentd_udp":              config.IntegerVariable(1234),
	"params_graphite":                 config.StringVariable("graphite.example.com:12345"),
	"params_ism_deletion_after":       config.StringVariable("30d"),
	"params_ism_jitter":               config.FloatVariable(0.6),
	"params_ism_job_interval":         config.IntegerVariable(5),
	"params_java_heapspace":           config.IntegerVariable(256),
	"params_java_maxmetaspace":        config.IntegerVariable(512),
	"params_max_disk_threshold":       config.IntegerVariable(80),
	"params_metrics_frequency":        config.IntegerVariable(10),
	"params_metrics_prefix":           config.StringVariable("actest"),
	"params_monitoring_instance_id":   config.StringVariable(uuid.NewString()),
	"params_opensearch_tls_ciphers":   config.StringVariable("TLS_DHE_RSA_WITH_AES_256_CBC_SHA,TLS_DHE_DSS_WITH_AES_128_CBC_SHA256"),
	"params_opensearch_tls_cipher1":   config.StringVariable("TLS_DHE_RSA_WITH_AES_256_CBC_SHA"),
	"params_opensearch_tls_cipher2":   config.StringVariable("TLS_DHE_DSS_WITH_AES_128_CBC_SHA256"),
	"params_opensearch_tls_protocol1": config.StringVariable("TLSv1.2"),
	"params_opensearch_tls_protocol2": config.StringVariable("TLSv1.3"),
	"params_sgw_acl":                  config.StringVariable("192.168.0.0/16,192.168.0.0/24"),
	"params_syslog1":                  config.StringVariable("syslog1.example.com:514"),
	"params_syslog2":                  config.StringVariable("syslog2.example.com:514"),
}

func configVarsMinUpdated() config.Variables {
	updatedConfig := maps.Clone(testConfigVarsMax)
	updatedConfig["name"] = config.StringVariable(minTestName + "-updated")
	return updatedConfig
}

func configVarsMaxUpdated() config.Variables {
	updatedConfig := maps.Clone(testConfigVarsMax)
	updatedConfig["parameters_max_disk_threshold"] = config.IntegerVariable(85)
	updatedConfig["parameters_metrics_frequency"] = config.IntegerVariable(10)
	updatedConfig["parameters_graphite"] = config.StringVariable("graphite.stackit.cloud:2003")
	updatedConfig["parameters_sgw_acl"] = config.StringVariable("192.168.1.0/24")
	updatedConfig["parameters_syslog"] = config.StringVariable("test.log:514")
	return updatedConfig
}

func TestAccLogMeMinResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLogMeDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config:          testutil.LogMeProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMin["plan_id"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMin["logme_version"])),

					// Credential data
					resource.TestCheckResourceAttrPair(
						"stackit_logme_credential.credential", "project_id",
						"stackit_logme_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logme_credential.credential", "instance_id",
						"stackit_logme_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_logme_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_logme_credential.credential", "host"),
				),
			},
			// Data source
			{
				Config:          testutil.LogMeProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),

					resource.TestCheckResourceAttrPair(
						"stackit_logme_instance.instance", "instance_id",
						"data.stackit_logme_instance.instance", "instance_id",
					),

					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "image_url"),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMin["plan_id"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMin["logme_version"])),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_logme_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "uri"),
				),
			},
			// Import
			{
				Config:          testutil.LogMeProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_logme_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_logme_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_logme_instance.instance")
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
				ResourceName:    "stackit_logme_credential.credential",
				ConfigVariables: testConfigVarsMin,

				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_logme_credential.credential"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_logme_credential.credential")
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
				Config:          testutil.LogMeProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: configVarsMinUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "project_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["plan_id"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "name", testutil.ConvertConfigVariable(configVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_name", testutil.ConvertConfigVariable(configVarsMinUpdated()["plan_name"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "version", testutil.ConvertConfigVariable(configVarsMinUpdated()["logme_version"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccLogMeMaxResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLogMeDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config:          testutil.LogMeProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMax["plan_id"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMax["plan_name"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["logme_version"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(testConfigVarsMax["params_enable_monitoring"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tcp", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tcp"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls_ciphers", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls_ciphers"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls_max_version", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls_max_version"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls_min_version", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls_min_version"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls_version", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls_version"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_udp", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_udp"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(testConfigVarsMax["params_graphite"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.ism_deletion_after", testutil.ConvertConfigVariable(testConfigVarsMax["params_ism_deletion_after"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.ism_jitter", testutil.ConvertConfigVariable(testConfigVarsMax["params_ism_jitter"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.ism_job_interval", testutil.ConvertConfigVariable(testConfigVarsMax["params_ism_job_interval"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.java_heapspace", testutil.ConvertConfigVariable(testConfigVarsMax["params_java_heapspace"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.java_maxmetaspace", testutil.ConvertConfigVariable(testConfigVarsMax["params_java_maxmetaspace"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["params_max_disk_threshold"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(testConfigVarsMax["params_metrics_frequency"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(testConfigVarsMax["params_metrics_prefix"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_ciphers.#", "2"),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_ciphers.0", testutil.ConvertConfigVariable(testConfigVarsMax["params_opensearch_tls_cipher1"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_ciphers.1", testutil.ConvertConfigVariable(testConfigVarsMax["params_opensearch_tls_cipher2"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_protocols.#", "2"),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_protocols.0", testutil.ConvertConfigVariable(testConfigVarsMax["params_opensearch_tls_protocol1"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_protocols.1", testutil.ConvertConfigVariable(testConfigVarsMax["params_opensearch_tls_protocol2"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(testConfigVarsMax["params_sgw_acl"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.#", "2"),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.0", testutil.ConvertConfigVariable(testConfigVarsMax["params_syslog1"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.1", testutil.ConvertConfigVariable(testConfigVarsMax["params_syslog2"])),

					// // Credential data
					resource.TestCheckResourceAttrPair(
						"stackit_logme_credential.credential", "project_id",
						"stackit_logme_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logme_credential.credential", "instance_id",
						"stackit_logme_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_logme_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_logme_credential.credential", "host"),
				),
			},
			// Data source
			{
				Config:          testutil.LogMeProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),

					resource.TestCheckResourceAttrPair(
						"stackit_logme_instance.instance", "instance_id",
						"data.stackit_logme_instance.instance", "instance_id",
					),

					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_instance.instance", "image_url"),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMax["plan_id"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMax["plan_name"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["logme_version"])),

					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(testConfigVarsMax["params_enable_monitoring"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.fluentd_tcp", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tcp"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.fluentd_tls", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.fluentd_tls_ciphers", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls_ciphers"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.fluentd_tls_max_version", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls_max_version"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.fluentd_tls_min_version", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls_min_version"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.fluentd_tls_version", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_tls_version"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.fluentd_udp", testutil.ConvertConfigVariable(testConfigVarsMax["params_fluentd_udp"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.ism_deletion_after", testutil.ConvertConfigVariable(testConfigVarsMax["params_ism_deletion_after"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.ism_jitter", testutil.ConvertConfigVariable(testConfigVarsMax["params_ism_jitter"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.ism_job_interval", testutil.ConvertConfigVariable(testConfigVarsMax["params_ism_job_interval"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.java_heapspace", testutil.ConvertConfigVariable(testConfigVarsMax["params_java_heapspace"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.java_maxmetaspace", testutil.ConvertConfigVariable(testConfigVarsMax["params_java_maxmetaspace"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["params_max_disk_threshold"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(testConfigVarsMax["params_metrics_frequency"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(testConfigVarsMax["params_metrics_prefix"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.opensearch_tls_ciphers.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.opensearch_tls_ciphers.0", testutil.ConvertConfigVariable(testConfigVarsMax["params_opensearch_tls_cipher1"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.opensearch_tls_ciphers.1", testutil.ConvertConfigVariable(testConfigVarsMax["params_opensearch_tls_cipher2"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.opensearch_tls_protocols.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.opensearch_tls_protocols.0", testutil.ConvertConfigVariable(testConfigVarsMax["params_opensearch_tls_protocol1"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.opensearch_tls_protocols.1", testutil.ConvertConfigVariable(testConfigVarsMax["params_opensearch_tls_protocol2"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(testConfigVarsMax["params_sgw_acl"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.syslog.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.syslog.0", testutil.ConvertConfigVariable(testConfigVarsMax["params_syslog1"])),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.syslog.1", testutil.ConvertConfigVariable(testConfigVarsMax["params_syslog2"])),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_logme_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "uri"),
				),
			},
			// Import
			{
				Config:          testutil.LogMeProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_logme_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_logme_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_logme_instance.instance")
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
				ResourceName:    "stackit_logme_credential.credential",
				ConfigVariables: testConfigVarsMax,

				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_logme_credential.credential"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_logme_credential.credential")
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
				Config:          testutil.LogMeProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["plan_id"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["plan_name"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "version", testutil.ConvertConfigVariable(configVarsMaxUpdated()["logme_version"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_enable_monitoring"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tcp", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_fluentd_tcp"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_fluentd_tls"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls_ciphers", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_fluentd_tls_ciphers"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls_max_version", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_fluentd_tls_max_version"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls_min_version", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_fluentd_tls_min_version"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tls_version", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_fluentd_tls_version"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_udp", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_fluentd_udp"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_graphite"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.ism_deletion_after", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_ism_deletion_after"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.ism_jitter", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_ism_jitter"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.ism_job_interval", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_ism_job_interval"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.java_heapspace", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_java_heapspace"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.java_maxmetaspace", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_java_maxmetaspace"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_max_disk_threshold"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_metrics_frequency"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_metrics_prefix"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_ciphers.#", "2"),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_ciphers.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_opensearch_tls_cipher1"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_ciphers.1", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_opensearch_tls_cipher2"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_protocols.#", "2"),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_protocols.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_opensearch_tls_protocol1"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.opensearch_tls_protocols.1", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_opensearch_tls_protocol2"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_sgw_acl"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.#", "2"),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_syslog1"])),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.1", testutil.ConvertConfigVariable(configVarsMaxUpdated()["params_syslog2"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckLogMeDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *logme.APIClient
	var err error
	if testutil.LogMeCustomEndpoint == "" {
		client, err = logme.NewAPIClient(
			core_config.WithRegion("eu01"),
		)
	} else {
		client, err = logme.NewAPIClient(
			core_config.WithEndpoint(testutil.LogMeCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_logme_instance" {
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

func checkInstanceDeleteSuccess(i *logme.Instance) bool {
	if *i.LastOperation.Type != logme.INSTANCELASTOPERATIONTYPE_DELETE {
		return false
	}

	if *i.LastOperation.Type == logme.INSTANCELASTOPERATIONTYPE_DELETE {
		if *i.LastOperation.State != logme.INSTANCELASTOPERATIONSTATE_SUCCEEDED {
			return false
		} else if strings.Contains(*i.LastOperation.Description, "DeleteFailed") || strings.Contains(*i.LastOperation.Description, "failed") {
			return false
		}
	}
	return true
}
