package secretsmanager_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	core_config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/secretsmanager"
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
	"instance_name":    config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"user_description": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"write_enabled":    config.BoolVariable(true),
}

var testConfigVarsMax = config.Variables{
	"project_id":       config.StringVariable(testutil.ProjectId),
	"instance_name":    config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"user_description": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"acl1":             config.StringVariable("10.100.0.0/24"),
	"acl2":             config.StringVariable("10.100.1.0/24"),
	"write_enabled":    config.BoolVariable(true),
}

func configVarsInvalid(vars config.Variables) config.Variables {
	tempConfig := maps.Clone(vars)
	delete(tempConfig, "instance_name")
	return tempConfig
}

func configVarsMinUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMin)
	tempConfig["write_enabled"] = config.BoolVariable(false)
	return tempConfig
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMax)
	tempConfig["write_enabled"] = config.BoolVariable(false)
	tempConfig["acl2"] = config.StringVariable("10.100.2.0/24")
	return tempConfig
}

func TestAccSecretsManagerMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSecretsManagerDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.SecretsManagerProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: configVarsInvalid(testConfigVarsMin),
				ExpectError:     regexp.MustCompile(`input variable "instance_name" is not set,`),
			},
			// Creation
			{
				Config:          resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["instance_name"])),
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
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "description", testutil.ConvertConfigVariable(testConfigVarsMin["user_description"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "write_enabled", testutil.ConvertConfigVariable(testConfigVarsMin["write_enabled"])),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "username"),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "password"),
				),
			},
			// Data source
			{
				Config:          resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_instance.instance", "instance_id",
						"data.stackit_secretsmanager_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMin["instance_name"])),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "acls.#", "0"),

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
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_user.user", "description", testutil.ConvertConfigVariable(testConfigVarsMin["user_description"])),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_user.user", "write_enabled", testutil.ConvertConfigVariable(testConfigVarsMin["write_enabled"])),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "username",
						"data.stackit_secretsmanager_user.user", "username",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_secretsmanager_instance.instance",
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
				Config:          resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_secretsmanager_user.user",
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
				Config:          resourceMinConfig,
				ConfigVariables: configVarsMinUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "project_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "name", testutil.ConvertConfigVariable(configVarsMinUpdated()["instance_name"])),
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
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "description", testutil.ConvertConfigVariable(configVarsMinUpdated()["user_description"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "write_enabled", testutil.ConvertConfigVariable(configVarsMinUpdated()["write_enabled"])),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "username"),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "password"),
				),
			},

			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccSecretsManagerMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSecretsManagerDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.SecretsManagerProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: configVarsInvalid(testConfigVarsMax),
				ExpectError:     regexp.MustCompile(`input variable "instance_name" is not set,`),
			},
			// Creation
			{
				Config:          resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["instance_name"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.#", "2"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl1"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.1", testutil.ConvertConfigVariable(testConfigVarsMax["acl2"])),

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
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "description", testutil.ConvertConfigVariable(testConfigVarsMax["user_description"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "write_enabled", testutil.ConvertConfigVariable(testConfigVarsMax["write_enabled"])),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "username"),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_user.user", "password"),
				),
			},
			// Data source
			{
				Config:          resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_instance.instance", "instance_id",
						"data.stackit_secretsmanager_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMax["instance_name"])),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "acls.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "acls.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl1"])),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance", "acls.1", testutil.ConvertConfigVariable(testConfigVarsMax["acl2"])),

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
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_user.user", "description", testutil.ConvertConfigVariable(testConfigVarsMax["user_description"])),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_user.user", "write_enabled", testutil.ConvertConfigVariable(testConfigVarsMax["write_enabled"])),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_user.user", "username",
						"data.stackit_secretsmanager_user.user", "username",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_secretsmanager_instance.instance",
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
				Config:          resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_secretsmanager_user.user",
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
				Config:          resourceMaxConfig,
				ConfigVariables: configVarsMaxUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["instance_name"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.#", "2"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["acl1"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance", "acls.1", testutil.ConvertConfigVariable(configVarsMaxUpdated()["acl2"])),

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
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "description", testutil.ConvertConfigVariable(configVarsMaxUpdated()["user_description"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_user.user", "write_enabled", testutil.ConvertConfigVariable(configVarsMaxUpdated()["write_enabled"])),
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
		client, err = secretsmanager.NewAPIClient(
			core_config.WithRegion("eu01"),
		)
	} else {
		client, err = secretsmanager.NewAPIClient(
			core_config.WithEndpoint(testutil.SecretsManagerCustomEndpoint),
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

	instancesResp, err := client.ListInstances(ctx, testutil.ProjectId).Execute()
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
