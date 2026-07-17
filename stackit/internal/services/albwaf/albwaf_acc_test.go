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
	albwaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/managed-rule-set.tf
	managedRuleSetConfig string
)

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

					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "usage.count", "0"),
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

					resource.TestCheckResourceAttr("data.stackit_alb_waf_managed_rule_set.managed_rule_set", "usage.count", "0"),
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

					resource.TestCheckResourceAttr("stackit_alb_waf_managed_rule_set.managed_rule_set", "usage.count", "0"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func createClient() (*albwaf.APIClient, error) {
	client, err := albwaf.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.AlbWafCustomEndpoint, false)...)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	return client, nil
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
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
