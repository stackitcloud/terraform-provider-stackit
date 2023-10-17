package secretsmanager_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/secretsmanager"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Instance resource data
var instanceResource = map[string]string{
	"project_id":    testutil.ProjectId,
	"name":          fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"acl-1":         "1.2.3.4/5",
	"acl-2":         "111.222.111.222/11",
	"acl-2-updated": "111.222.111.222/22",
}

func resourceConfig(acl2 string) string {
	return fmt.Sprintf(`
				%s

				resource "stackit_secretsmanager_instance" "instance" {
					project_id = "%s"
					name       = "%s"
					acls = ["%s", "%s"]
				}
				`,
		testutil.SecretsManagerProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["acl-1"],
		acl2,
	)
}
func TestAccSecretsManager(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSecretsManagerDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfig(instanceResource["acl-2"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.#", "2"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.0", instanceResource["acl-1"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.1", instanceResource["acl-2"]),
				),
			},
			{ // Data source
				Config: fmt.Sprintf(`
					%s

					data "stackit_secretsmanager_instance" "instance" {
						project_id  = stackit_secretsmanager_instance.instance.project_id
						instance_id = stackit_secretsmanager_instance.instance.instance_id
					}`,
					resourceConfig(instanceResource["acl-2"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_instance.instance", "instance_id",
						"data.stackit_secretsmanager_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "acls.0", instanceResource["acl-1"]),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "acls.1", instanceResource["acl-2"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_secretsmanager_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_secretsmanager_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_secretsmanager_instance.instance")
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
			// Update,
			{
				Config: resourceConfig(instanceResource["acl-2-updated"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.#", "2"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.0", instanceResource["acl-1"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.1", instanceResource["acl-2-updated"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckSecretsManagerDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *secretsmanager.APIClient
	var err error
	if testutil.SecretsManagerCustomEndpoint == "" {
		client, err = secretsmanager.NewAPIClient()
	} else {
		client, err = secretsmanager.NewAPIClient(
			config.WithEndpoint(testutil.SecretsManagerCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_secretsmanager_instance" {
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
		if instances[i].Id == nil {
			continue
		}
		if utils.Contains(instancesToDestroy, *instances[i].Id) {
			err := client.DeleteInstanceExecute(ctx, testutil.ProjectId, *instances[i].Id)
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: %w", *instances[i].Id, err)
			}
		}
	}
	return nil
}
