package secretsmanager_test

import (
	_ "embed"
	"fmt"
	"maps"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testdestroy"
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
	"project_id":           config.StringVariable(testutil.ProjectId),
	"instance_name":        config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"user_description":     config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"acl1":                 config.StringVariable("10.100.0.0/24"),
	"acl2":                 config.StringVariable("10.100.1.0/24"),
	"write_enabled":        config.BoolVariable(true),
	"service_account_mail": config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"use_kms_key":          config.BoolVariable(true),
}

func configVarsInvalid(vars config.Variables) config.Variables {
	tempConfig := maps.Clone(vars)
	delete(tempConfig, "instance_name")
	return tempConfig
}

func configVarsMinUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMin)
	tempConfig["instance_name"] = config.StringVariable(testutil.ConvertConfigVariable(tempConfig["instance_name"]) + "-updated")
	tempConfig["write_enabled"] = config.BoolVariable(false)
	return tempConfig
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMax)
	tempConfig["instance_name"] = config.StringVariable(testutil.ConvertConfigVariable(tempConfig["instance_name"]) + "-updated")
	tempConfig["write_enabled"] = config.BoolVariable(false)
	tempConfig["use_kms_key"] = config.BoolVariable(false)
	tempConfig["acl2"] = config.StringVariable("10.100.2.0/24")
	return tempConfig
}

func TestAccSecretsManagerMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testdestroy.AccTestCheckDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMinConfig,
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_secretsmanager_user.user", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_secretsmanager_instance.instance", plancheck.ResourceActionUpdate),
					},
				},
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
		CheckDestroy:             testdestroy.AccTestCheckDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceMaxConfig,
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

					// Instance with kms key
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance_with_key", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "name", testutil.ConvertConfigVariable(testConfigVarsMax["instance_name"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "acls.#", "2"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "acls.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl1"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "acls.1", testutil.ConvertConfigVariable(testConfigVarsMax["acl2"])),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_instance.instance_with_key", "kms_key.key_id",
						"stackit_kms_key.key", "key_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_secretsmanager_instance.instance_with_key", "kms_key.key_ring_id",
						"stackit_kms_keyring.keyring", "keyring_id",
					),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "kms_key.key_version", "1"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "kms_key.service_account_email", testutil.ConvertConfigVariable(testConfigVarsMax["service_account_mail"])),
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

					// Instance with kms key
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance_with_key", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_secretsmanager_instance.instance_with_key", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance_with_key", "name", testutil.ConvertConfigVariable(testConfigVarsMax["instance_name"])),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance_with_key", "acls.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance_with_key", "acls.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl1"])),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance_with_key", "acls.1", testutil.ConvertConfigVariable(testConfigVarsMax["acl2"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_secretsmanager_instance.instance_with_key", "kms_key.key_id",
						"stackit_kms_key.key", "key_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_secretsmanager_instance.instance_with_key", "kms_key.key_ring_id",
						"stackit_kms_keyring.keyring", "keyring_id",
					),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance_with_key", "kms_key.key_version", "1"),
					resource.TestCheckResourceAttr("data.stackit_secretsmanager_instance.instance_with_key", "kms_key.service_account_email", testutil.ConvertConfigVariable(testConfigVarsMax["service_account_mail"])),
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
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_secretsmanager_instance.instance_with_key",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_secretsmanager_instance.instance_with_key"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_secretsmanager_instance.instance_with_key")
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_secretsmanager_user.user", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_secretsmanager_instance.instance", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_secretsmanager_instance.instance_with_key", plancheck.ResourceActionUpdate),
					},
				},
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

					// Instance with kms key
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_secretsmanager_instance.instance_with_key", "instance_id"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["instance_name"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "acls.#", "2"),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "acls.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["acl1"])),
					resource.TestCheckResourceAttr("stackit_secretsmanager_instance.instance_with_key", "acls.1", testutil.ConvertConfigVariable(configVarsMaxUpdated()["acl2"])),
					resource.TestCheckNoResourceAttr("stackit_secretsmanager_instance.instance_with_key", "kms_key"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}
