package mariadb_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/mariadb"
	"github.com/stackitcloud/stackit-sdk-go/services/mariadb/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Instance resource data
var instanceResource = map[string]string{
	"project_id": testutil.ProjectId,
	"name":       testutil.ResourceNameWithDateTime("mariadb"),
	"plan_id":    "683be856-3587-42de-b1b5-a792ff854f52",
	"plan_name":  "stackit-qa-mariadb-1.4.10-single",
	"version":    "10.6",
	"sgw_acl-1":  "192.168.0.0/16",
	"sgw_acl-2":  "192.168.0.0/24",
}

func resourceConfig(acls string) string {
	return fmt.Sprintf(`
				%s

				resource "stackit_mariadb_instance" "instance" {
					project_id = "%s"
					name       = "%s"
					plan_name  = "%s"
 				 	version    = "%s"
					parameters = {
						sgw_acl = "%s"
					}
				}

				resource "stackit_mariadb_credential" "credentials" {
					project_id = stackit_mariadb_instance.instance.project_id
					instance_id = stackit_mariadb_instance.instance.instance_id
				}
				`,
		testutil.MariaDBProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["plan_name"],
		instanceResource["version"],
		acls,
	)
}
func TestAccMariaDBResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMariaDBDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfig(instanceResource["sgw_acl-1"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-1"]),

					// Credentials data
					resource.TestCheckResourceAttrPair(
						"stackit_mariadb_credential.credentials", "project_id",
						"stackit_mariadb_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_mariadb_credential.credentials", "instance_id",
						"stackit_mariadb_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credentials", "credentials_id"),
					resource.TestCheckResourceAttrSet("stackit_mariadb_credential.credentials", "host"),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_mariadb_instance" "instance" {
						project_id  = stackit_mariadb_instance.instance.project_id
						instance_id = stackit_mariadb_instance.instance.instance_id
					}

					data "stackit_mariadb_credential" "credentials" {
						project_id     = stackit_mariadb_credential.credentials.project_id
						instance_id    = stackit_mariadb_credential.credentials.instance_id
					    credentials_id = stackit_mariadb_credential.credentials.credentials_id
					}`,
					resourceConfig(instanceResource["sgw_acl-1"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrPair("stackit_mariadb_instance.instance", "instance_id",
						"data.stackit_mariadb_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrPair("stackit_mariadb_credential.credentials", "credentials_id",
						"data.stackit_mariadb_credential.credentials", "credentials_id"),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_mariadb_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-1"]),

					// Credentials data
					resource.TestCheckResourceAttr("data.stackit_mariadb_credential.credentials", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credentials", "credentials_id"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credentials", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credentials", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_mariadb_credential.credentials", "uri"),
				),
			},
			// Import
			{
				ResourceName: "stackit_mariadb_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_mariadb_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_mariadb_instance.instance")
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
				ResourceName: "stackit_mariadb_credential.credentials",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_mariadb_credential.credentials"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_mariadb_credential.credentials")
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
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_mariadb_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_mariadb_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl-2"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckMariaDBDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *mariadb.APIClient
	var err error
	if testutil.MariaDBCustomEndpoint == "" {
		client, err = mariadb.NewAPIClient()
	} else {
		client, err = mariadb.NewAPIClient(
			config.WithEndpoint(testutil.MariaDBCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_mariadb_instance" {
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

func checkInstanceDeleteSuccess(i *mariadb.Instance) bool {
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
