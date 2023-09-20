package logme_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/logme"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/testutil"
)

// Instance resource data
var instanceResource = map[string]string{
	"project_id": testutil.ProjectId,
	"name":       testutil.ResourceNameWithDateTime("logme"),
	"plan_id":    "201d743c-0f06-4af2-8f20-649baf4819ae",
	"plan_name":  "stackit-qa-logme2-1.2.50-replica",
	"version":    "2",
	"sgw_acl-1":  "192.168.0.0/16",
	"sgw_acl-2":  "192.168.0.0/24",
}

func resourceConfig(acls string) string {
	return fmt.Sprintf(`
				%s

				resource "stackit_logme_instance" "instance" {
					project_id = "%s"
					name       = "%s"
					plan_name  = "%s"
 				 	version    = "%s"
					parameters = {
						sgw_acl = "%s"
					}
				}

				resource "stackit_logme_credentials" "credentials" {
					project_id = stackit_logme_instance.instance.project_id
					instance_id = stackit_logme_instance.instance.instance_id
				}
				`,
		testutil.LogMeProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["plan_name"],
		instanceResource["version"],
		acls,
	)
}
func TestAccLogMeResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLogMeDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfig(instanceResource["sgw_acl-1"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_logme_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-1"]),

					// Credentials data
					resource.TestCheckResourceAttrPair(
						"stackit_logme_credentials.credentials", "project_id",
						"stackit_logme_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logme_credentials.credentials", "instance_id",
						"stackit_logme_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_logme_credentials.credentials", "credentials_id"),
					resource.TestCheckResourceAttrSet("stackit_logme_credentials.credentials", "host"),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_logme_instance" "instance" {
						project_id  = stackit_logme_instance.instance.project_id
						instance_id = stackit_logme_instance.instance.instance_id
					}

					data "stackit_logme_credentials" "credentials" {
						project_id     = stackit_logme_credentials.credentials.project_id
						instance_id    = stackit_logme_credentials.credentials.instance_id
					    credentials_id = stackit_logme_credentials.credentials.credentials_id
					}`,
					resourceConfig(instanceResource["sgw_acl-1"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "project_id", instanceResource["project_id"]),

					resource.TestCheckResourceAttrPair("stackit_logme_instance.instance", "instance_id",
						"data.stackit_logme_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrPair("stackit_logme_credentials.credentials", "credentials_id",
						"data.stackit_logme_credentials.credentials", "credentials_id"),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-1"]),

					// Credentials data
					resource.TestCheckResourceAttr("data.stackit_logme_credentials.credentials", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credentials.credentials", "credentials_id"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credentials.credentials", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credentials.credentials", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credentials.credentials", "uri"),
				),
			},
			// Import
			{
				ResourceName: "stackit_logme_instance.instance",
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
				ResourceName: "stackit_logme_credentials.credentials",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_logme_credentials.credentials"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_logme_credentials.credentials")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					credentialsId, ok := r.Primary.Attributes["credentials_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute credentials_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, credentialsId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: resourceConfig(instanceResource["sgw_acl-2"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_logme_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_logme_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-2"]),
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
		client, err = logme.NewAPIClient()
	} else {
		client, err = logme.NewAPIClient(
			config.WithEndpoint(testutil.LogMeCustomEndpoint),
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
				_, err = logme.DeleteInstanceWaitHandler(ctx, client, testutil.ProjectId, *instances[i].InstanceId).WaitWithContext(ctx)
				if err != nil {
					return fmt.Errorf("destroying instance %s during CheckDestroy: waiting for deletion %w", *instances[i].InstanceId, err)
				}
			}
		}
	}
	return nil
}

func checkInstanceDeleteSuccess(i *logme.Instance) bool {
	if *i.LastOperation.Type != logme.InstanceTypeDelete {
		return false
	}

	if *i.LastOperation.Type == logme.InstanceTypeDelete {
		if *i.LastOperation.State != logme.InstanceStateSuccess {
			return false
		} else if strings.Contains(*i.LastOperation.Description, "DeleteFailed") || strings.Contains(*i.LastOperation.Description, "failed") {
			return false
		}
	}
	return true
}
