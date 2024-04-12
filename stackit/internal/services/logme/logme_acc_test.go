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
	"github.com/stackitcloud/stackit-sdk-go/services/logme/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Instance resource data
var instanceResource = map[string]string{
	"project_id": testutil.ProjectId,
	"name":       testutil.ResourceNameWithDateTime("logme"),
	"plan_id":    "201d743c-0f06-4af2-8f20-649baf4819ae",
	"plan_name":  "stackit-logme2-1.2.50-replica",
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

				resource "stackit_logme_credential" "credential" {
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
				Config: fmt.Sprintf(`
					%s

					data "stackit_logme_instance" "instance" {
						project_id  = stackit_logme_instance.instance.project_id
						instance_id = stackit_logme_instance.instance.instance_id
					}

					data "stackit_logme_credential" "credential" {
						project_id     = stackit_logme_credential.credential.project_id
						instance_id    = stackit_logme_credential.credential.instance_id
					    credential_id = stackit_logme_credential.credential.credential_id
					}`,
					resourceConfig(instanceResource["sgw_acl-1"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "project_id", instanceResource["project_id"]),

					resource.TestCheckResourceAttrPair("stackit_logme_instance.instance", "instance_id",
						"data.stackit_logme_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrPair("stackit_logme_credential.credential", "credential_id",
						"data.stackit_logme_credential.credential", "credential_id"),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_logme_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-1"]),

					// Credential data
					resource.TestCheckResourceAttr("data.stackit_logme_credential.credential", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "credential_id"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_logme_credential.credential", "uri"),
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
				ResourceName: "stackit_logme_credential.credential",
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
		client, err = logme.NewAPIClient(
			config.WithRegion("eu01"),
		)
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
