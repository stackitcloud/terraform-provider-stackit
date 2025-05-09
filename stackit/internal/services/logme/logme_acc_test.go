package logme_test

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"

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
	//go:embed testdata/resource-max.tf
	resourceMaxConfig string
)

// Instance resource data
var testConfigVarsMax = map[string]any{
	"params_sgw_acl":            []string{"192.168.0.0/16", "192.168.0.0/24"},
	"params_fluentd_tcp":        "4",
	"params_max_disk_threshold": "80",
	"params_syslog":             []string{"syslog.example.com:514"},
	"params_ism_jitter":         "0.6",

	"project_id":                testutil.ProjectId,
	"name":                      testutil.ResourceNameWithDateTime("logme"),
	"plan_id":                   "201d743c-0f06-4af2-8f20-649baf4819ae",
	"plan_name":                 "stackit-logme2-1.2.50-replica",
	"logme_version":             "2",
	"params_enable_monitoring":  "false",
	"params_fluentd_tcp":              "",
	"params_fluentd_tls":              "",
	"params_fluentd_tls_ciphers":      "",
	"params_fluentd_tls_max_version":  "",
	"params_fluentd_tls_min_version":  "",
	"params_fluentd_tls_version":      "",
	"params_fluentd_udp":              "",
	"params_graphite":                 "",
	"params_ism_deletion_after":       "",
	"params_ism_jitter":               "",
	"params_ism_job_interval":         "",
	"params_java_heapspace":           "",
	"params_java_maxmetaspace":        "",
	"params_max_disk_threshold":       "",
	"params_metrics_frequency":        "",
	"params_metrics_prefix":           "",
	"params_monitoring_instance_id":   "",
	"params_opensearch_tls_cipher1":   "",
	"params_opensearch_tls_cipher2":   "",
	"params_opensearch_tls_protocols": "",
	"params_sgw_acl":                  "",
	"params_syslog1":                  "",
	"params_syslog2":                  "",
}

func TestAccLogMeResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLogMeDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config:          testutil.LogMeProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: testutil.ConvertToVariables(testConfigVarsMax),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "project_id", testConfigVarsMax["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_logme_instance.instance", "instance_id"),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_id", testConfigVarsMax["plan_id"]),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_name", testConfigVarsMax["plan_name"]),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "version", testConfigVarsMax["logme_version"]),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "name", testConfigVarsMax["name"]),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.sgw_acl", testConfigVarsMax["params_sgw_acl"]),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tcp", testConfigVarsMax["params_fluent_tcp"]),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.max_disk_threshold", testConfigVarsMax["params_max_disk_threshold"]),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.enable_monitoring", testConfigVarsMax["params_enable_monitoring"]),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.#", "1"),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.0", testConfigVarsMax["params_syslog"].([]string)[0]),
					testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.ism_jitter", testConfigVarsMax["params_ism_jitter"]),

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
			// // Data source
			// {
			// 	Config: fmt.Sprintf(`
			// 		%s

			// 		data "stackit_logme_instance" "instance" {
			// 			project_id  = stackit_logme_instance.instance.project_id
			// 			instance_id = stackit_logme_instance.instance.instance_id
			// 		}

			// 		data "stackit_logme_credential" "credential" {
			// 			project_id     = stackit_logme_credential.credential.project_id
			// 			instance_id    = stackit_logme_credential.credential.instance_id
			// 		    credential_id = stackit_logme_credential.credential.credential_id
			// 		}`,
			// 		resourceConfig(map[string]string{
			// 			"sgw_acl":            testConfigVarsMax["sgw_acl-1"],
			// 			"fluentd_tcp":        testConfigVarsMax["fluent_tcp"],
			// 			"max_disk_threshold": testConfigVarsMax["max_disk_threshold"],
			// 			"enable_monitoring":  testConfigVarsMax["enable_monitoring"],
			// 			"syslog":             fmt.Sprintf(`[%q]`, testConfigVarsMax["syslog-0"]),
			// 			"ism_jitter":         testConfigVarsMax["ism_jitter"],
			// 		}),
			// 	),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		// Instance data
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "project_id", testConfigVarsMax["project_id"]),

			// 		testutil.TestCheckResourceAttrPair("stackit_logme_instance.instance", "instance_id",
			// 			"data.stackit_logme_instance.instance", "instance_id"),
			// 		testutil.TestCheckResourceAttrPair("stackit_logme_credential.credential", "credential_id",
			// 			"data.stackit_logme_credential.credential", "credential_id"),
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "plan_id", testConfigVarsMax["plan_id"]),
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "name", testConfigVarsMax["name"]),
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.sgw_acl", testConfigVarsMax["sgw_acl-1"]),
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.fluentd_tcp", testConfigVarsMax["fluent_tcp"]),
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.max_disk_threshold", testConfigVarsMax["max_disk_threshold"]),
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.enable_monitoring", testConfigVarsMax["enable_monitoring"]),
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.syslog.#", "1"),
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.syslog.0", testConfigVarsMax["syslog-0"]),
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.ism_jitter", testConfigVarsMax["ism_jitter"]),

			// 		// Credential data
			// 		testutil.TestCheckResourceAttr("data.stackit_logme_credential.credential", "project_id", testConfigVarsMax["project_id"]),
			// 		testutil.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "credential_id"),
			// 		testutil.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "host"),
			// 		testutil.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "port"),
			// 		testutil.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "uri"),
			// 	),
			// },
			// // Import
			// {
			// 	ResourceName: "stackit_logme_instance.instance",
			// 	ImportStateIdFunc: func(s *terraform.State) (string, error) {
			// 		r, ok := s.RootModule().Resources["stackit_logme_instance.instance"]
			// 		if !ok {
			// 			return "", fmt.Errorf("couldn't find resource stackit_logme_instance.instance")
			// 		}
			// 		instanceId, ok := r.Primary.Attributes["instance_id"]
			// 		if !ok {
			// 			return "", fmt.Errorf("couldn't find attribute instance_id")
			// 		}
			// 		return fmt.Sprintf("%s,%s", testutil.ProjectId, instanceId), nil
			// 	},
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
			// {
			// 	ResourceName: "stackit_logme_credential.credential",
			// 	ImportStateIdFunc: func(s *terraform.State) (string, error) {
			// 		r, ok := s.RootModule().Resources["stackit_logme_credential.credential"]
			// 		if !ok {
			// 			return "", fmt.Errorf("couldn't find resource stackit_logme_credential.credential")
			// 		}
			// 		instanceId, ok := r.Primary.Attributes["instance_id"]
			// 		if !ok {
			// 			return "", fmt.Errorf("couldn't find attribute instance_id")
			// 		}
			// 		credentialId, ok := r.Primary.Attributes["credential_id"]
			// 		if !ok {
			// 			return "", fmt.Errorf("couldn't find attribute credential_id")
			// 		}
			// 		return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, credentialId), nil
			// 	},
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
			// // Update
			// {
			// 	Config: resourceConfig(map[string]string{
			// 		"sgw_acl":            testConfigVarsMax["sgw_acl-2"],
			// 		"fluentd_tcp":        testConfigVarsMax["fluent_tcp"],
			// 		"max_disk_threshold": testConfigVarsMax["max_disk_threshold"],
			// 		"enable_monitoring":  testConfigVarsMax["enable_monitoring"],
			// 		"syslog":             fmt.Sprintf(`[%q]`, testConfigVarsMax["syslog-0"]),
			// 		"ism_jitter":         testConfigVarsMax["ism_jitter"],
			// 	}),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		// Instance data
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "project_id", testConfigVarsMax["project_id"]),
			// 		testutil.TestCheckResourceAttrSet("stackit_logme_instance.instance", "instance_id"),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_id", testConfigVarsMax["plan_id"]),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_name", testConfigVarsMax["plan_name"]),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "version", testConfigVarsMax["logme_version"]),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "name", testConfigVarsMax["name"]),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.sgw_acl", testConfigVarsMax["sgw_acl-2"]),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.fluentd_tcp", testConfigVarsMax["fluent_tcp"]),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.max_disk_threshold", testConfigVarsMax["max_disk_threshold"]),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.enable_monitoring", testConfigVarsMax["enable_monitoring"]),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.#", "1"),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.syslog.0", testConfigVarsMax["syslog-0"]),
			// 		testutil.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.ism_jitter", testConfigVarsMax["ism_jitter"]),
			// 	),
			// },
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
	if *i.LastOperation.Type != wait.InstanceOperationTypeDelete {
		return false
	}

	if *i.LastOperation.Type == wait.InstanceOperationTypeDelete {
		if *i.LastOperation.State != wait.InstanceOperationStateSucceeded {
			return false
		} else if strings.Contains(*i.LastOperation.Description, "DeleteFailed") || strings.Contains(*i.LastOperation.Description, "failed") {
			return false
		}
	}
	return true
}
