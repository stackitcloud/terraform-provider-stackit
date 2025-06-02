package opensearch_test

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	sdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-min.tf
	resourceMinConfig string

	//go:embed testdata/resource-max.tf
	resourceMaxConfig string
)

var testConfigVarsMin = config.Variables{
	"project_id":       config.StringVariable(testutil.ProjectId),
	"name":             config.StringVariable(testutil.ResourceNameWithDateTime("opensearch")),
	"instance_version": config.StringVariable("2"),
	"plan_name":        config.StringVariable("stackit-opensearch-1.4.10-single"),
	"plan_id":          config.StringVariable("24615c29-99e8-4cc2-bcc3-ad7f45a5d46f"),
}

var testConfigVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigVarsMin {
		updatedConfig[k] = v
	}
	updatedConfig["plan_name"] = config.StringVariable("stackit-opensearch-2.4.10-single")
	updatedConfig["plan_id"] = config.StringVariable("f97a4935-0a77-4939-bfd1-33ba2e1f2b36")
	return updatedConfig
}()

var testConfigVarsMax = config.Variables{
	"project_id":             config.StringVariable(testutil.ProjectId),
	"name":                   config.StringVariable(testutil.ResourceNameWithDateTime("opensearch")),
	"instance_version":       config.StringVariable("2"),
	"plan_name":              config.StringVariable("stackit-opensearch-1.4.10-single"),
	"plan_id":                config.StringVariable("24615c29-99e8-4cc2-bcc3-ad7f45a5d46f"),
	"enable_monitoring":      config.BoolVariable(true),
	"graphite":               config.StringVariable("graphite.example.com:2003"),
	"java_garbage_collector": config.StringVariable("UseSerialGC"),
	"java_heapspace":         config.IntegerVariable(256),
	"java_maxmetaspace":      config.IntegerVariable(512),
	"max_disk_threshold":     config.IntegerVariable(75),
	"metrics_frequency":      config.IntegerVariable(15),
	"metrics_prefix":         config.StringVariable("acctest"),
	"monitoring_instance_id": config.StringVariable(uuid.NewString()),
	"plugin":                 config.StringVariable("analysis-phonetic"),
	"sgw_acl":                config.StringVariable("192.168.0.0/16,192.168.0.0/24"),
	"syslog":                 config.StringVariable("syslog.example.com:514"),
	"tls_ciphers":            config.StringVariable("TLS_DHE_RSA_WITH_AES_256_CBC_SHA"),
	// "tls_protocols": config.StringVariable("TLSv1.2"),
	"observability_instance_plan_name": config.StringVariable("Observability-Monitoring-Basic-EU01"),
}

var testConfigVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigVarsMin {
		updatedConfig[k] = v
	}
	updatedConfig["plan_name"] = config.StringVariable("stackit-opensearch-2.4.10-single")
	updatedConfig["plan_id"] = config.StringVariable("f97a4935-0a77-4939-bfd1-33ba2e1f2b36")
	updatedConfig["enable_monitoring"] = config.BoolVariable(true)
	updatedConfig["graphite"] = config.StringVariable("graphite.updated.com:2004")
	updatedConfig["java_garbage_collector"] = config.StringVariable("UseParallelGC")
	updatedConfig["java_heapspace"] = config.IntegerVariable(260)
	updatedConfig["java_maxmetaspace"] = config.IntegerVariable(520)
	updatedConfig["max_disk_threshold"] = config.IntegerVariable(70)
	updatedConfig["metrics_frequency"] = config.IntegerVariable(5)
	updatedConfig["metrics_prefix"] = config.StringVariable("updated")
	updatedConfig["plugin"] = config.StringVariable("analysis-phonetic-update")
	updatedConfig["sgw_acl"] = config.StringVariable("192.168.0.0/24")
	updatedConfig["syslog"] = config.StringVariable("syslog.update.com:515")
	updatedConfig["tls_ciphers"] = config.StringVariable("TLS_AES_256_GCM_SHA384")
	// updatedConfig["tls_protocols"] = config.StringVariable("TLSv1.3")
	return updatedConfig
}()

// Instance resource data
var instanceResource = map[string]string{
	"project_id":         testutil.ProjectId,
	"name":               testutil.ResourceNameWithDateTime("opensearch"),
	"plan_id":            "9e4eac4b-b03d-4d7b-b01b-6d1224aa2d68",
	"plan_name":          "stackit-opensearch-1.2.10-replica",
	"version":            "2",
	"sgw_acl-1":          "192.168.0.0/16",
	"sgw_acl-2":          "192.168.0.0/24",
	"max_disk_threshold": "80",
	"enable_monitoring":  "false",
	"syslog-0":           "syslog.example.com:514",
}

func parametersConfig(params map[string]string) string {
	nonStringParams := []string{
		"enable_monitoring",
		"max_disk_threshold",
		"metrics_frequency",
		"java_heapspace",
		"java_maxmetaspace",
		"plugins",
		"syslog",
		"tls_ciphers",
	}
	parameters := "parameters = {"
	for k, v := range params {
		if utils.Contains(nonStringParams, k) {
			parameters += fmt.Sprintf("%s = %s\n", k, v)
		} else {
			parameters += fmt.Sprintf("%s = %q\n", k, v)
		}
	}
	parameters += "\n}"
	return parameters
}

func resourceConfig(params map[string]string) string {
	return fmt.Sprintf(`
				%s

				resource "stackit_opensearch_instance" "instance" {
					project_id = "%s"
					name    = "%s"
					plan_name  = "%s"
 				 	version    = "%s"
					%s
				}

				resource "stackit_opensearch_credential" "credential" {
					project_id = stackit_opensearch_instance.instance.project_id
					instance_id = stackit_opensearch_instance.instance.instance_id
				}
				`,
		testutil.OpenSearchProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["plan_name"],
		instanceResource["version"],
		parametersConfig(params),
	)
}

func TestAccOpenSearchResourceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckOpenSearchDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.OpenSearchProviderConfig(), resourceMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMin["plan_id"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMin["plan_name"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMin["instance_version"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "parameters.max_disk_threshold"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "parameters.sgw_acl"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "image_url"),

					// Credential data
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "project_id",
						"stackit_opensearch_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "instance_id",
						"stackit_opensearch_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "scheme"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "username"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_opensearch_instance" "instance" {
						project_id  = stackit_opensearch_instance.instance.project_id
						instance_id = stackit_opensearch_instance.instance.instance_id
					}

					data "stackit_opensearch_credential" "credential" {
						project_id     = stackit_opensearch_credential.credential.project_id
						instance_id    = stackit_opensearch_credential.credential.instance_id
					    credential_id = stackit_opensearch_credential.credential.credential_id
					}`,
					testutil.OpenSearchProviderConfig(), resourceMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair("stackit_opensearch_instance.instance", "instance_id",
						"data.stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrPair("stackit_opensearch_credential.credential", "credential_id",
						"data.stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMin["plan_id"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMin["instance_version"])),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "parameters.max_disk_threshold"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "parameters.sgw_acl"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "image_url"),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_opensearch_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "scheme"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "username"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_opensearch_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_opensearch_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_opensearch_instance.instance")
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
				ResourceName:    "stackit_opensearch_credential.credential",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_opensearch_credential.credential"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_opensearch_credential.credential")
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
				ConfigVariables: testConfigVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.OpenSearchProviderConfig(), resourceMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMinUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMinUpdated["plan_id"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMinUpdated["plan_name"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMinUpdated["instance_version"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMinUpdated["name"])),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "parameters.max_disk_threshold"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "parameters.sgw_acl"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "image_url"),

					// Credential data
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "project_id",
						"stackit_opensearch_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "instance_id",
						"stackit_opensearch_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "scheme"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "username"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccOpenSearchResourceMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckOpenSearchDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.OpenSearchProviderConfig(), resourceMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMax["plan_id"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMax["plan_name"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["instance_version"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),

					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(testConfigVarsMax["enable_monitoring"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(testConfigVarsMax["graphite"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.java_garbage_collector", testutil.ConvertConfigVariable(testConfigVarsMax["java_garbage_collector"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.java_heapspace", testutil.ConvertConfigVariable(testConfigVarsMax["java_heapspace"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.java_maxmetaspace", testutil.ConvertConfigVariable(testConfigVarsMax["java_maxmetaspace"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["max_disk_threshold"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_frequency"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_prefix"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_opensearch_instance.instance", "parameters.monitoring_instance_id",
					),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.plugin.#", "1"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.plugin.1", testutil.ConvertConfigVariable(testConfigVarsMax["plugin"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(testConfigVarsMax["sgw_acl"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.syslog", testutil.ConvertConfigVariable(testConfigVarsMax["syslog"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.tls_ciphers.#", "1"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.tls_ciphers.1", testutil.ConvertConfigVariable(testConfigVarsMax["tls_ciphers"])),
					// resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.tls_protocols", testutil.ConvertConfigVariable(testConfigVarsMax["tls_protocols"])),

					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "image_url"),

					// Credential data
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "project_id",
						"stackit_opensearch_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "instance_id",
						"stackit_opensearch_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "scheme"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "username"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_opensearch_instance" "instance" {
						project_id  = stackit_opensearch_instance.instance.project_id
						instance_id = stackit_opensearch_instance.instance.instance_id
					}

					data "stackit_opensearch_credential" "credential" {
						project_id     = stackit_opensearch_credential.credential.project_id
						instance_id    = stackit_opensearch_credential.credential.instance_id
					    credential_id = stackit_opensearch_credential.credential.credential_id
					}`,
					testutil.OpenSearchProviderConfig(), resourceMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair("stackit_opensearch_instance.instance", "instance_id",
						"data.stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrPair("stackit_opensearch_credential.credential", "credential_id",
						"data.stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMax["plan_id"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMax["instance_version"])),

					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(testConfigVarsMax["enable_monitoring"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(testConfigVarsMax["graphite"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.java_garbage_collector", testutil.ConvertConfigVariable(testConfigVarsMax["java_garbage_collector"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.java_heapspace", testutil.ConvertConfigVariable(testConfigVarsMax["java_heapspace"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.java_maxmetaspace", testutil.ConvertConfigVariable(testConfigVarsMax["java_maxmetaspace"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(testConfigVarsMax["max_disk_threshold"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_frequency"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(testConfigVarsMax["metrics_prefix"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"data.stackit_opensearch_instance.instance", "parameters.monitoring_instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.plugin.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.plugin.1", testutil.ConvertConfigVariable(testConfigVarsMax["plugin"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(testConfigVarsMax["sgw_acl"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.syslog", testutil.ConvertConfigVariable(testConfigVarsMax["syslog"])),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.tls_ciphers.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.tls_ciphers.1", testutil.ConvertConfigVariable(testConfigVarsMax["tls_ciphers"])),
					// resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.tls_protocols", testutil.ConvertConfigVariable(testConfigVarsMax["tls_protocols"])),

					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "image_url"),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_opensearch_credential.credential", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "scheme"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "username"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_opensearch_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_opensearch_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_opensearch_instance.instance")
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
				ResourceName:    "stackit_opensearch_credential.credential",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_opensearch_credential.credential"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_opensearch_credential.credential")
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
				ConfigVariables: testConfigVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.OpenSearchProviderConfig(), resourceMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_id", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["plan_id"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_name", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["plan_name"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["instance_version"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["name"])),

					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.enable_monitoring", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["enable_monitoring"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.graphite", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["graphite"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.java_garbage_collector", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["java_garbage_collector"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.java_heapspace", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["java_heapspace"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.java_maxmetaspace", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["java_maxmetaspace"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.max_disk_threshold", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["max_disk_threshold"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.metrics_frequency", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["metrics_frequency"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.metrics_prefix", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["metrics_prefix"])),
					resource.TestCheckResourceAttrPair(
						"stackit_observability_instance.instance", "instance_id",
						"stackit_opensearch_instance.instance", "parameters.monitoring_instance_id",
					),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.plugin.#", "1"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.plugin.1", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["plugin"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.sgw_acl", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["sgw_acl"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.syslog", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["syslog"])),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.tls_ciphers.#", "1"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.tls_ciphers.1", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["tls_ciphers"])),
					// resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.tls_protocols", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated["tls_protocols"])),

					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_organization_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "cf_space_guid"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "image_url"),

					// Credential data
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "project_id",
						"stackit_opensearch_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "instance_id",
						"stackit_opensearch_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "hosts.#"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "password"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "scheme"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "username"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccOpenSearchResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckOpenSearchDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfig(
					map[string]string{
						"sgw_acl":            instanceResource["sgw_acl-1"],
						"max_disk_threshold": instanceResource["max_disk_threshold"],
						"enable_monitoring":  instanceResource["enable_monitoring"],
						"syslog":             fmt.Sprintf(`[%q]`, instanceResource["syslog-0"]),
					}),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-1"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.max_disk_threshold", instanceResource["max_disk_threshold"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.enable_monitoring", instanceResource["enable_monitoring"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.syslog.#", "1"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.syslog.0", instanceResource["syslog-0"]),

					// Credential data
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "project_id",
						"stackit_opensearch_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credential.credential", "instance_id",
						"stackit_opensearch_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credential.credential", "host"),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_opensearch_instance" "instance" {
						project_id  = stackit_opensearch_instance.instance.project_id
						instance_id = stackit_opensearch_instance.instance.instance_id
					}

					data "stackit_opensearch_credential" "credential" {
						project_id     = stackit_opensearch_credential.credential.project_id
						instance_id    = stackit_opensearch_credential.credential.instance_id
					    credential_id = stackit_opensearch_credential.credential.credential_id
					}`,
					resourceConfig(
						map[string]string{
							"sgw_acl":            instanceResource["sgw_acl-1"],
							"max_disk_threshold": instanceResource["max_disk_threshold"],
							"enable_monitoring":  instanceResource["enable_monitoring"],
							"syslog":             fmt.Sprintf(`[%q]`, instanceResource["syslog-0"]),
						}),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrPair("stackit_opensearch_instance.instance", "instance_id",
						"data.stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrPair("stackit_opensearch_credential.credential", "credential_id",
						"data.stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-1"]),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.max_disk_threshold", instanceResource["max_disk_threshold"]),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.enable_monitoring", instanceResource["enable_monitoring"]),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.syslog.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "parameters.syslog.0", instanceResource["syslog-0"]),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_opensearch_credential.credential", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "uri"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credential.credential", "scheme"),
				),
			},
			// Import
			{
				ResourceName: "stackit_opensearch_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_opensearch_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_opensearch_instance.instance")
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
				ResourceName: "stackit_opensearch_credential.credential",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_opensearch_credential.credential"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_opensearch_credential.credential")
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
				Config: resourceConfig(
					map[string]string{
						"sgw_acl":            instanceResource["sgw_acl-2"],
						"max_disk_threshold": instanceResource["max_disk_threshold"],
						"enable_monitoring":  instanceResource["enable_monitoring"],
						"syslog":             fmt.Sprintf(`[%q]`, instanceResource["syslog-0"]),
					}),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-2"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.max_disk_threshold", instanceResource["max_disk_threshold"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.enable_monitoring", instanceResource["enable_monitoring"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.syslog.#", "1"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.syslog.0", instanceResource["syslog-0"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckOpenSearchDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *opensearch.APIClient
	var err error
	if testutil.OpenSearchCustomEndpoint == "" {
		client, err = opensearch.NewAPIClient(
			sdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = opensearch.NewAPIClient(
			sdkConfig.WithEndpoint(testutil.OpenSearchCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_opensearch_instance" {
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

func checkInstanceDeleteSuccess(i *opensearch.Instance) bool {
	if *i.LastOperation.Type != opensearch.INSTANCELASTOPERATIONTYPE_DELETE {
		return false
	}

	if *i.LastOperation.Type == opensearch.INSTANCELASTOPERATIONTYPE_DELETE {
		if *i.LastOperation.State != opensearch.INSTANCELASTOPERATIONSTATE_SUCCEEDED {
			return false
		} else if strings.Contains(*i.LastOperation.Description, "DeleteFailed") || strings.Contains(*i.LastOperation.Description, "failed") {
			return false
		}
	}
	return true
}
