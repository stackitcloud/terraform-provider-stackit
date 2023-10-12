package rabbitmq_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Instance resource data
var instanceResource = map[string]string{
	"project_id":      testutil.ProjectId,
	"name":            testutil.ResourceNameWithDateTime("rabbitmq"),
	"plan_id":         "7e1f8394-5dd5-40b1-8608-16b4344eb51b",
	"plan_name":       "stackit-qa-rabbitmq-2.4.10-single",
	"version":         "3.10",
	"sgw_acl_invalid": "1.2.3.4/4",
	"sgw_acl_valid":   "1.2.3.4/31",
}

func resourceConfig(acls *string) string {
	aclsLine := ""
	if acls != nil {
		aclsLine = fmt.Sprintf(`sgw_acl = %q`, *acls)
	}
	return fmt.Sprintf(`
				%s

				resource "stackit_rabbitmq_instance" "instance" {
					project_id = "%s"
					name       = "%s"
					plan_name  = "%s"
 				 	version    = "%s"
					parameters = {
						%s
						metrics_frequency = "%s"
					}
				}

				%s
				`,
		testutil.RabbitMQProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["plan_name"],
		instanceResource["version"],
		aclsLine,
		instanceResource["metrics_frequency"],
		resourceConfigCredentials(),
	)
}

func resourceConfigWithUpdate() string {
	return fmt.Sprintf(`
				%s

				resource "stackit_rabbitmq_instance" "instance" {
					project_id = "%s"
					name    = "%s"
					plan_name  = "%s"
 				 	version    = "%s"
					parameters = {
						sgw_acl = "%s"
					}
				}

				%s
				`,
		testutil.RabbitMQProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["plan_name"],
		instanceResource["version"],
		instanceResource["sgw_acl_valid"],
		resourceConfigCredentials(),
	)
}

func resourceConfigCredentials() string {
	return `
		resource "stackit_rabbitmq_credential" "credentials" {
			project_id = stackit_rabbitmq_instance.instance.project_id
			instance_id = stackit_rabbitmq_instance.instance.instance_id
		}
    `
}

func TestAccRabbitMQResource(t *testing.T) {
	acls := instanceResource["sgw_acl_invalid"]
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRabbitMQDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:      resourceConfig(&acls),
				ExpectError: regexp.MustCompile(`.*sgw_acl is invalid.*`),
			},
			// Creation
			{
				Config: resourceConfig(nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "parameters.sgw_acl"),

					// Credentials data
					resource.TestCheckResourceAttrPair(
						"stackit_rabbitmq_credential.credentials", "project_id",
						"stackit_rabbitmq_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_rabbitmq_credential.credentials", "instance_id",
						"stackit_rabbitmq_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credentials", "credential_id"),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_credential.credentials", "host"),
				),
			},
			// data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_rabbitmq_instance" "instance" {
						project_id  = stackit_rabbitmq_instance.instance.project_id
						instance_id = stackit_rabbitmq_instance.instance.instance_id
					}

					data "stackit_rabbitmq_credential" "credentials" {
						project_id     = stackit_rabbitmq_credential.credentials.project_id
						instance_id    = stackit_rabbitmq_credential.credentials.instance_id
					    credential_id = stackit_rabbitmq_credential.credentials.credential_id
					}`,
					resourceConfig(nil),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrPair("stackit_rabbitmq_instance.instance", "instance_id",
						"data.stackit_rabbitmq_credential.credentials", "instance_id"),
					resource.TestCheckResourceAttrPair("data.stackit_rabbitmq_instance.instance", "instance_id",
						"data.stackit_rabbitmq_credential.credentials", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_instance.instance", "parameters.sgw_acl"),

					// Credentials data
					resource.TestCheckResourceAttr("data.stackit_rabbitmq_credential.credentials", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credentials", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credentials", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credentials", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_rabbitmq_credential.credentials", "uri"),
				),
			},
			// Import
			{
				ResourceName: "stackit_rabbitmq_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_rabbitmq_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_rabbitmq_instance.instance")
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
				ResourceName: "stackit_rabbitmq_credential.credentials",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_rabbitmq_credential.credentials"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_rabbitmq_credential.credentials")
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
				Config: resourceConfigWithUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_rabbitmq_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_rabbitmq_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl_valid"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckRabbitMQDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *rabbitmq.APIClient
	var err error
	if testutil.RabbitMQCustomEndpoint == "" {
		client, err = rabbitmq.NewAPIClient()
	} else {
		client, err = rabbitmq.NewAPIClient(
			config.WithEndpoint(testutil.RabbitMQCustomEndpoint),
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

	instancesResp, err := client.GetInstances(ctx, testutil.ProjectId).Execute()
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
	if *i.LastOperation.Type != wait.InstanceTypeDelete {
		return false
	}

	if *i.LastOperation.Type == wait.InstanceTypeDelete {
		if *i.LastOperation.State != wait.InstanceStateSuccess {
			return false
		} else if strings.Contains(*i.LastOperation.Description, "DeleteFailed") || strings.Contains(*i.LastOperation.Description, "failed") {
			return false
		}
	}
	return true
}
