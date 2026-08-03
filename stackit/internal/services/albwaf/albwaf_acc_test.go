package albwaf_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/custom-rule-group-min.tf
	customRuleGroupMinConfig string

	//go:embed testdata/custom-rule-group-max.tf
	customRuleGroupMaxConfig string

	//go:embed testdata/managed-rule-set.tf
	managedRuleSetConfig string
)

var testCustomRuleGroupMin = config.Variables{
	"project_id":     config.StringVariable(testutil.ProjectId),
	"name":           config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"action":         config.StringVariable("ACTION_DENY"),
	"operator_type":  config.StringVariable("OPERATOR_VALIDATE_UTF8_ENCODING"),
	"operator_value": config.StringVariable("foo"),
	"transformation": config.StringVariable("TRANSFORMATION_LOWERCASE"),
	"variable_type":  config.StringVariable("VARIABLE_RESPONSE_STATUS"),
}

var testCustomRuleGroupMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testCustomRuleGroupMin)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	return updatedConfig
}

var testCustomRuleGroupMax = config.Variables{
	"project_id":     config.StringVariable(testutil.ProjectId),
	"name":           config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"description":    config.StringVariable("foo bar"),
	"action":         config.StringVariable("ACTION_DENY"),
	"log":            config.BoolVariable(true),
	"log_msg":        config.StringVariable("foo-bar"),
	"operator_type":  config.StringVariable("OPERATOR_CONTAINS"),
	"operator_value": config.StringVariable("foo"),
	"transformation": config.StringVariable("TRANSFORMATION_LOWERCASE"),
	"variable_type":  config.StringVariable("VARIABLE_REQUEST_HEADERS"),
	"variable_value": config.StringVariable("bar"),
}

var testCustomRuleGroupMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testCustomRuleGroupMax)
	// Name should not be updated, test if the update works in place
	updatedConfig["log_msg"] = config.StringVariable("foo-bar:")
	// updatedConfig["log"] = config.BoolVariable(false)
	return updatedConfig
}

var testManagedRuleSet = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"type":       config.StringVariable("TYPE_OWASP_CRS"),
}

var testManagedRuleSetUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testManagedRuleSet)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	return updatedConfig
}

func TestAccCustomRuleGroupMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testCustomRuleGroupMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), customRuleGroupMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "id"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "name", testutil.ConvertConfigVariable(testCustomRuleGroupMin["name"])),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.#", "1"),
					// resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.id"),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.action", testutil.ConvertConfigVariable(testCustomRuleGroupMin["action"])),
					// resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.log", "false"),
					// resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.severity"),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.#", "1"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.operator.type", testutil.ConvertConfigVariable(testCustomRuleGroupMin["operator_type"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.transformations.#", "0"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.variable.type", testutil.ConvertConfigVariable(testCustomRuleGroupMin["variable_type"])),
				),
			},
			// Data source
			{
				ConfigVariables: testCustomRuleGroupMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_alb_waf_custom_rule_group" "custom_rule_group" {
					  project_id = stackit_alb_waf_custom_rule_group.custom_rule_group.project_id
					  name  = stackit_alb_waf_custom_rule_group.custom_rule_group.name
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), customRuleGroupMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_waf_custom_rule_group.custom_rule_group", "id",
						"stackit_alb_waf_custom_rule_group.custom_rule_group", "id",
					),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "name", testutil.ConvertConfigVariable(testCustomRuleGroupMin["name"])),

					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.id",
						"stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.id",
					),

					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.action", testutil.ConvertConfigVariable(testCustomRuleGroupMin["action"])),
					// resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.log", "false"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.severity",
						"stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.severity",
					),

					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.operator.type", testutil.ConvertConfigVariable(testCustomRuleGroupMin["operator_type"])),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.transformations.#", "0"),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.variable.type", testutil.ConvertConfigVariable(testCustomRuleGroupMin["variable_type"])),
				),
			},
			// Import
			{
				ConfigVariables: testCustomRuleGroupMin,
				ResourceName:    "stackit_alb_waf_custom_rule_group.custom_rule_group",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_alb_waf_custom_rule_group.custom_rule_group"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_alb_waf_custom_rule_group.custom_rule_group")
					}
					policyId, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, policyId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testCustomRuleGroupMinUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), customRuleGroupMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_alb_waf_custom_rule_group.custom_rule_group", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "id"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "name", testutil.ConvertConfigVariable(testCustomRuleGroupMinUpdated()["name"])),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.#", "1"),
					// resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "rule.0.id"),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.action", testutil.ConvertConfigVariable(testCustomRuleGroupMinUpdated()["action"])),
					// resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.log", "false"),
					// resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.severity"),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.#", "1"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.operator.type", testutil.ConvertConfigVariable(testCustomRuleGroupMinUpdated()["operator_type"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.transformations.#", "0"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.variable.type", testutil.ConvertConfigVariable(testCustomRuleGroupMinUpdated()["variable_type"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccCustomRuleGroupMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testCustomRuleGroupMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), customRuleGroupMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "id"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "name", testutil.ConvertConfigVariable(testCustomRuleGroupMax["name"])),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.#", "1"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.description", testutil.ConvertConfigVariable(testCustomRuleGroupMax["description"])),
					// resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.id"),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.action", testutil.ConvertConfigVariable(testCustomRuleGroupMax["action"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.log", testutil.ConvertConfigVariable(testCustomRuleGroupMax["log"])),
					// resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.log_msg", testutil.ConvertConfigVariable(testCustomRuleGroupMax["log_msg"])),
					// resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.severity"),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.#", "1"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.operator.type", testutil.ConvertConfigVariable(testCustomRuleGroupMax["operator_type"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.operator.value", testutil.ConvertConfigVariable(testCustomRuleGroupMax["operator_value"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.transformations.#", "1"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.transformations.0", testutil.ConvertConfigVariable(testCustomRuleGroupMax["transformation"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.variable.type", testutil.ConvertConfigVariable(testCustomRuleGroupMax["variable_type"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.variable.value", testutil.ConvertConfigVariable(testCustomRuleGroupMax["variable_value"])),
				),
			},
			// Data source
			{
				ConfigVariables: testCustomRuleGroupMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_alb_waf_custom_rule_group" "custom_rule_group" {
					  project_id = stackit_alb_waf_custom_rule_group.custom_rule_group.project_id
					  name  = stackit_alb_waf_custom_rule_group.custom_rule_group.name
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), customRuleGroupMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_waf_custom_rule_group.custom_rule_group", "id",
						"stackit_alb_waf_custom_rule_group.custom_rule_group", "id",
					),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "name", testutil.ConvertConfigVariable(testCustomRuleGroupMax["name"])),

					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.description", testutil.ConvertConfigVariable(testCustomRuleGroupMax["description"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.id",
						"stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.id",
					),

					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.action", testutil.ConvertConfigVariable(testCustomRuleGroupMax["action"])),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.log", testutil.ConvertConfigVariable(testCustomRuleGroupMax["log"])),
					// resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.log_msg", testutil.ConvertConfigVariable(testCustomRuleGroupMax["log_msg"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.severity",
						"stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.severity",
					),

					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.operator.type", testutil.ConvertConfigVariable(testCustomRuleGroupMax["operator_type"])),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.operator.value", testutil.ConvertConfigVariable(testCustomRuleGroupMax["operator_value"])),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.transformations.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.transformations.0", testutil.ConvertConfigVariable(testCustomRuleGroupMax["transformation"])),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.variable.type", testutil.ConvertConfigVariable(testCustomRuleGroupMax["variable_type"])),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.variable.value", testutil.ConvertConfigVariable(testCustomRuleGroupMax["variable_value"])),
				),
			},
			// Import
			{
				ConfigVariables: testCustomRuleGroupMax,
				ResourceName:    "stackit_alb_waf_custom_rule_group.custom_rule_group",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_alb_waf_custom_rule_group.custom_rule_group"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_alb_waf_custom_rule_group.custom_rule_group")
					}
					policyId, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, policyId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testCustomRuleGroupMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), customRuleGroupMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_alb_waf_custom_rule_group.custom_rule_group", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "id"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "name", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["name"])),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.#", "1"),
					// resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rule.0.description", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["description"])),
					// resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "rule.0.id"),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.action", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["action"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.log", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["log"])),
					// resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.log_msg", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["log_msg"])),
					// resource.TestCheckResourceAttrSet("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.behavior.severity"),

					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.#", "1"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.operator.type", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["operator_type"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.operator.value", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["operator_value"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.transformations.#", "1"),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.transformations.0", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["transformation"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.variable.type", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["variable_type"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_custom_rule_group.custom_rule_group", "rules.0.conditions.0.variable.value", testutil.ConvertConfigVariable(testCustomRuleGroupMaxUpdated()["variable_value"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccManagedRuleSet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testManagedRuleSet,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), managedRuleSetConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_alb_waf_managed_rule_set.managed_rule_set", "id"),
					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "name", testutil.ConvertConfigVariable(testManagedRuleSet["name"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "type", testutil.ConvertConfigVariable(testManagedRuleSet["type"])),
				),
			},
			// Data source
			{
				ConfigVariables: testManagedRuleSet,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_alb_waf_managed_rule_set" "managed_rule_set" {
					  project_id = stackit_alb_waf_managed_rule_set.managed_rule_set.project_id
					  name  = stackit_alb_waf_managed_rule_set.managed_rule_set.name
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), managedRuleSetConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_alb_waf_managed_rule_set.managed_rule_set", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_managed_rule_set.managed_rule_set", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_waf_managed_rule_set.managed_rule_set", "id",
						"stackit_alb_waf_managed_rule_set.managed_rule_set", "id",
					),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_managed_rule_set.managed_rule_set", "name", testutil.ConvertConfigVariable(testManagedRuleSet["name"])),
					resource.TestCheckResourceAttr("data.stackit_alb_waf_managed_rule_set.managed_rule_set", "type", testutil.ConvertConfigVariable(testManagedRuleSet["type"])),
				),
			},
			// Import
			{
				ConfigVariables: testManagedRuleSet,
				ResourceName:    "stackit_alb_waf_managed_rule_set.managed_rule_set",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_alb_waf_managed_rule_set.managed_rule_set"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_alb_waf_managed_rule_set.managed_rule_set")
					}
					policyId, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, policyId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testManagedRuleSetUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), managedRuleSetConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_alb_waf_managed_rule_set.managed_rule_set", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_alb_waf_managed_rule_set.managed_rule_set", "id"),
					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "name", testutil.ConvertConfigVariable(testManagedRuleSetUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "type", testutil.ConvertConfigVariable(testManagedRuleSetUpdated()["type"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func createClient() (*albWaf.APIClient, error) {
	client, err := albWaf.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.AlbWafCustomEndpoint, false)...)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	return client, nil
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAlbWafCustomRuleGroupDestroy,
		testAlbWafManagedRuleSetDestroy,
	}
	var errs []error

	for _, f := range checkFunctions {
		func() {
			err := f(s)
			if err != nil {
				errs = append(errs, err)
			}
		}()
	}
	return errors.Join(errs...)
}

func testAlbWafCustomRuleGroupDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := createClient()
	if err != nil {
		return err
	}

	customRuleGroupsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_alb_waf_custom_rule_group" {
			continue
		}
		// custom rule group transform id: "[projectId],[region],[name]"
		name := strings.Split(rs.Primary.ID, core.Separator)[2]
		customRuleGroupsToDestroy = append(customRuleGroupsToDestroy, name)
	}

	resp, err := client.DefaultAPI.ListCustomRuleGroup(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting resp: %w", err)
	}

	for _, item := range resp.Items {
		if utils.Contains(customRuleGroupsToDestroy, item.GetName()) {
			_, err := client.DefaultAPI.DeleteCustomRuleGroup(ctx, testutil.ProjectId, testutil.Region, item.GetName()).Execute()
			if err != nil {
				return fmt.Errorf("deleting policy %s during CheckDestroy: %w", item.GetName(), err)
			}
		}
	}
	return nil
}

func testAlbWafManagedRuleSetDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := createClient()
	if err != nil {
		return err
	}

	managedRuleSetsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_alb_waf_managed_rule_set" {
			continue
		}
		// managed rule set transform id: "[projectId],[region],[name]"
		name := strings.Split(rs.Primary.ID, core.Separator)[2]
		managedRuleSetsToDestroy = append(managedRuleSetsToDestroy, name)
	}

	resp, err := client.DefaultAPI.ListManagedRuleSets(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting resp: %w", err)
	}

	for _, item := range resp.Items {
		if utils.Contains(managedRuleSetsToDestroy, item.GetName()) {
			_, err := client.DefaultAPI.DeleteManagedRuleSet(ctx, testutil.ProjectId, testutil.Region, item.GetName()).Execute()
			if err != nil {
				return fmt.Errorf("deleting policy %s during CheckDestroy: %w", item.GetName(), err)
			}
		}
	}
	return nil
}
