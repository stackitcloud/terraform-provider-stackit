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
	"acl-0":         "1.2.3.4/5",
	"acl-1":         "111.222.111.222/11",
	"acl-1-updated": "111.222.111.222/22",
}

// User resource data
var userResource = map[string]string{
	"description":           testutil.ResourceNameWithDateTime("secretsmanager"),
	"write_enabled":         "false",
	"write_enabled_updated": "true",
}

func resourceConfig(acls *string, writeEnabled string) string {
	if acls == nil {
		return fmt.Sprintf(`
					%s
	
					resource "stackit_secretsmanager_instance" "instance" {
						project_id = "%s"
						name       = "%s"
					}

					resource "stackit_secretsmanager_user" "user" {
						project_id = stackit_secretsmanager_instance.instance.project_id
						instance_id = stackit_secretsmanager_instance.instance.instance_id
						description = "%s"
						write_enabled = %s
					}
					`,
			testutil.SecretsManagerProviderConfig(),
			instanceResource["project_id"],
			instanceResource["name"],
			userResource["description"],
			writeEnabled,
		)
	}

	return fmt.Sprintf(`
				%s

				resource "stackit_secretsmanager_instance" "instance" {
					project_id = "%s"
					name       = "%s"
					acls = %s
				}

				resource "stackit_secretsmanager_user" "user" {
					project_id = stackit_secretsmanager_instance.instance.project_id
					instance_id = stackit_secretsmanager_instance.instance.instance_id
					description = "%s"
					write_enabled = %s
				}
				`,
		testutil.SecretsManagerProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		*acls,
		userResource["description"],
		writeEnabled,
	)
}

func TestAccSecretsManager(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSecretsManagerDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfig(
					utils.Ptr(fmt.Sprintf(
						"[%q, %q, %q]",
						instanceResource["acl-0"],
						instanceResource["acl-1"],
						instanceResource["acl-1"],
					)),
					userResource["write_enabled"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.#", "2"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.0", instanceResource["acl-0"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.1", instanceResource["acl-1"]),

					// User
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "project_id",
						"stackit_secretsmanager_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "instance_id",
						"stackit_secretsmanager_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "user_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "description", userResource["description"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "write_enabled", userResource["write_enabled"]),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "username"),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "password"),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_secretsmanager_instance" "instance" {
						project_id  = stackit_secretsmanager_instance.instance.project_id
						instance_id = stackit_secretsmanager_instance.instance.instance_id
					}

					data "stackit_secretsmanager_user" "user" {
						project_id  = stackit_secretsmanager_user.user.project_id
						instance_id = stackit_secretsmanager_user.user.instance_id
						user_id = stackit_secretsmanager_user.user.user_id
					}`,
					resourceConfig(
						utils.Ptr(fmt.Sprintf(
							"[%q, %q]",
							instanceResource["acl-0"],
							instanceResource["acl-1"],
						)),
						userResource["write_enabled"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_instance.instance", "instance_id",
						"data.stackit_secretsmanager_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "acls.0", instanceResource["acl-0"]),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "acls.1", instanceResource["acl-1"]),

					// User
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "project_id",
						"data.stackit_secretsmanager_user.user", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "instance_id",
						"data.stackit_secretsmanager_user.user", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "user_id",
						"data.stackit_secretsmanager_user.user", "user_id",
					),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_user.user", "description", userResource["description"]),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_user.user", "write_enabled", userResource["write_enabled"]),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "username",
						"data.stackit_secretsmanager_user.user", "username",
					),
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
			{
				ResourceName: "stackit_secretsmanager_user.user",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_secretsmanager_user.user"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_secretsmanager_user.user")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					userId, ok := r.Primary.Attributes["user_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute user_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, userId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
				Check:                   resource.TestCheckNoResourceAttr("stackit_secretsmanager_user.user", "password"),
			},
			// Update
			{
				Config: resourceConfig(
					utils.Ptr(fmt.Sprintf(
						"[%q, %q]",
						instanceResource["acl-0"],
						instanceResource["acl-1-updated"],
					)),
					userResource["write_enabled_updated"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.#", "2"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.0", instanceResource["acl-0"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.1", instanceResource["acl-1-updated"]),

					// User
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "project_id",
						"stackit_secretsmanager_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "instance_id",
						"stackit_secretsmanager_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "user_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "description", userResource["description"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "write_enabled", userResource["write_enabled_updated"]),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "username"),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "password"),
				),
			},
			// Update, no ACLs
			{
				Config: resourceConfig(nil, userResource["write_enabled_updated"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.#", "0"),

					// User
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "project_id",
						"stackit_secretsmanager_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "instance_id",
						"stackit_secretsmanager_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "user_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "description", userResource["description"]),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "write_enabled", userResource["write_enabled_updated"]),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "username"),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "password"),
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
