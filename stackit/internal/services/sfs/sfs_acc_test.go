package sfs_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/export-policy-min.tf
	resourceExportPolicyMinConfig string

	//go:embed testdata/export-policy-max.tf
	resourceExportPolicyMaxConfig string

	//go:embed testdata/resource-pool-min.tf
	resourceResourcePoolMinConfig string

	//go:embed testdata/resource-pool-max.tf
	resourceResourcePoolMaxConfig string

	//go:embed testdata/share-min.tf
	resourceShareMinConfig string

	//go:embed testdata/share-max.tf
	resourceShareMaxConfig string

	//go:embed testdata/project-lock-min.tf
	resourceProjectLockConfig string
)

// EXPORT POLICY - MIN

var testConfigExportPolicyVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
}

var testConfigExportPolicyVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigExportPolicyVarsMin)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	return updatedConfig
}

// EXPORT POLICY - MAX

var testConfigExportPolicyVarsMax = config.Variables{
	"project_id":             config.StringVariable(testutil.ProjectId),
	"region":                 config.StringVariable(testutil.Region),
	"name":                   config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"first_rule_description": config.StringVariable("Some description"),
	"first_rule_ip_acl_1":    config.StringVariable("172.16.0.0/24"),
	"first_rule_ip_acl_2":    config.StringVariable("172.16.0.250/32"),
	"first_rule_set_uuid":    config.BoolVariable(true),
	"second_rule_ip_acl_1":   config.StringVariable("172.16.0.0/24"),
	"second_rule_ip_acl_2":   config.StringVariable("172.16.0.250/32"),
	"second_rule_read_only":  config.BoolVariable(true),
	"second_rule_super_user": config.BoolVariable(false),
}

var testConfigExportPolicyVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigExportPolicyVarsMax)

	updatedConfig["first_rule_description"] = config.StringVariable("Some other description")
	updatedConfig["first_rule_ip_acl_1"] = config.StringVariable("172.17.0.0/24")
	updatedConfig["first_rule_ip_acl_2"] = config.StringVariable("172.17.0.250/32")

	return updatedConfig
}

// Resource Pool - MIN

var testConfigResourcePoolVarsMin = config.Variables{
	"project_id":        config.StringVariable(testutil.ProjectId),
	"name":              config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"ip_acl_1":          config.StringVariable("192.168.42.1/32"),
	"ip_acl_2":          config.StringVariable("192.168.42.2/32"),
	"availability_zone": config.StringVariable("eu01-m"),
	"performance_class": config.StringVariable("Standard"),
	"size_gigabytes":    config.IntegerVariable(500),
}

var testConfigResourcePoolVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigResourcePoolVarsMin)
	updatedConfig["performance_class"] = config.StringVariable("Premium")
	updatedConfig["size_gigabytes"] = config.IntegerVariable(512)
	updatedConfig["ip_acl_1"] = config.StringVariable("172.17.0.0/24")
	updatedConfig["ip_acl_2"] = config.StringVariable("172.17.0.250/32")
	return updatedConfig
}

// Resource Pool - MAX

var testConfigResourcePoolVarsMax = config.Variables{
	"project_id":            config.StringVariable(testutil.ProjectId),
	"name":                  config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"ip_acl_1":              config.StringVariable("192.168.42.1/32"),
	"ip_acl_2":              config.StringVariable("192.168.42.2/32"),
	"region":                config.StringVariable(testutil.Region),
	"availability_zone":     config.StringVariable("eu01-m"),
	"performance_class":     config.StringVariable("Standard"),
	"size_gigabytes":        config.IntegerVariable(512),
	"snapshots_are_visible": config.BoolVariable(true),
}

var testConfigResourcePoolVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigResourcePoolVarsMax)
	updatedConfig["performance_class"] = config.StringVariable("Premium")
	updatedConfig["snapshots_are_visible"] = config.BoolVariable(false)
	updatedConfig["size_gigabytes"] = config.IntegerVariable(1024)
	updatedConfig["ip_acl_1"] = config.StringVariable("172.17.0.0/24")
	updatedConfig["ip_acl_2"] = config.StringVariable("172.17.0.250/32")
	return updatedConfig
}

// Share - MIN

var testConfigShareVarsMin = config.Variables{
	"project_id":                 config.StringVariable(testutil.ProjectId),
	"name":                       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"resource_pool_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"space_hard_limit_gigabytes": config.IntegerVariable(42),
}

var testConfigShareVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigShareVarsMin)
	updatedConfig["space_hard_limit_gigabytes"] = config.IntegerVariable(50)
	return updatedConfig
}

// Share - MAX

var testConfigShareVarsMax = config.Variables{
	"project_id":                 config.StringVariable(testutil.ProjectId),
	"name":                       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"region":                     config.StringVariable(testutil.Region),
	"resource_pool_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"space_hard_limit_gigabytes": config.IntegerVariable(42),
	"export_policy_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
}

var testConfigShareVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigShareVarsMax)
	updatedConfig["space_hard_limit_gigabytes"] = config.IntegerVariable(50)
	return updatedConfig
}

// Project lock - MIN

var testConfigProjectLockVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
}

func TestAccExportPolicyMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigExportPolicyVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceExportPolicyMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "policy_id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMin["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "0"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigExportPolicyVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_export_policy" "policy_data_test" {
					  project_id = stackit_sfs_export_policy.exportpolicy.project_id
					  policy_id  = stackit_sfs_export_policy.exportpolicy.policy_id
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceExportPolicyMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_export_policy.policy_data_test", "id",
						"stackit_sfs_export_policy.exportpolicy", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_export_policy.policy_data_test", "policy_id",
						"stackit_sfs_export_policy.exportpolicy", "policy_id",
					),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMin["name"])),

					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.#", "0"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigExportPolicyVarsMin,
				ResourceName:    "stackit_sfs_export_policy.exportpolicy",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sfs_export_policy.exportpolicy"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sfs_export_policy.exportpolicy")
					}
					policyId, ok := r.Primary.Attributes["policy_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute policy_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, policyId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigExportPolicyVarsMinUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceExportPolicyMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sfs_export_policy.exportpolicy", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "policy_id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMinUpdated()["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "0"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccExportPolicyMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigExportPolicyVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceExportPolicyMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "policy_id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.description", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["first_rule_description"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["first_rule_ip_acl_1"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["first_rule_ip_acl_2"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"), // default value
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["first_rule_set_uuid"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"), // default value

					resource.TestCheckNoResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.description"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.order", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.0", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["second_rule_ip_acl_1"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.1", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["second_rule_ip_acl_2"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.read_only", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["second_rule_read_only"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.set_uuid", "false"), // default value
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.super_user", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["second_rule_super_user"])),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigExportPolicyVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_export_policy" "policy_data_test" {
					  project_id = stackit_sfs_export_policy.exportpolicy.project_id
					  policy_id  = stackit_sfs_export_policy.exportpolicy.policy_id
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceExportPolicyMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_export_policy.policy_data_test", "id",
						"stackit_sfs_export_policy.exportpolicy", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_export_policy.policy_data_test", "policy_id",
						"stackit_sfs_export_policy.exportpolicy", "policy_id",
					),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["name"])),

					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.description", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["first_rule_description"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.ip_acl.0", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["first_rule_ip_acl_1"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.ip_acl.1", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["first_rule_ip_acl_2"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.read_only", "false"), // default value
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.set_uuid", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["first_rule_set_uuid"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.super_user", "true"), // default value

					resource.TestCheckNoResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.description"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.order", "2"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.ip_acl.0", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["second_rule_ip_acl_1"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.ip_acl.1", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["second_rule_ip_acl_2"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.read_only", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["second_rule_read_only"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.set_uuid", "false"), // default value
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.super_user", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["second_rule_super_user"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigExportPolicyVarsMax,
				ResourceName:    "stackit_sfs_export_policy.exportpolicy",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sfs_export_policy.exportpolicy"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sfs_export_policy.exportpolicy")
					}
					policyId, ok := r.Primary.Attributes["policy_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute policy_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, policyId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigExportPolicyVarsMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceExportPolicyMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sfs_export_policy.exportpolicy", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "policy_id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.description", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["first_rule_description"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["first_rule_ip_acl_1"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["first_rule_ip_acl_2"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"), // default value
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["first_rule_set_uuid"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"), // default value

					resource.TestCheckNoResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.description"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.order", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.0", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["second_rule_ip_acl_1"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.1", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["second_rule_ip_acl_2"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.read_only", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["second_rule_read_only"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.set_uuid", "false"), // default value
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.super_user", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["second_rule_super_user"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccResourcePoolResourceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigResourcePoolVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceResourcePoolMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["ip_acl_1"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["ip_acl_2"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", "false"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigResourcePoolVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_resource_pool" "resource_pool_ds" {
					  project_id       = stackit_sfs_resource_pool.resourcepool.project_id
					  resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceResourcePoolMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_resource_pool.resource_pool_ds", "id",
						"stackit_sfs_resource_pool.resourcepool", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_resource_pool.resource_pool_ds", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["availability_zone"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["performance_class"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["size_gigabytes"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "ip_acl.0", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["ip_acl_1"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "ip_acl.1", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["ip_acl_2"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "snapshots_are_visible", "false"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigResourcePoolVarsMin,
				ResourceName:    "stackit_sfs_resource_pool.resourcepool",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					res, found := s.RootModule().Resources["stackit_sfs_resource_pool.resourcepool"]
					if !found {
						return "", fmt.Errorf("could not find resource stackit_sfs_resource_pool.resourcepool")
					}
					resourcepoolId, ok := res.Primary.Attributes["resource_pool_id"]
					if !ok {
						return "", fmt.Errorf("resource pool id attribute not found")
					}
					return testutil.ProjectId + "," + testutil.Region + "," + resourcepoolId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigResourcePoolVarsMinUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceResourcePoolMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sfs_resource_pool.resourcepool", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["ip_acl_1"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["ip_acl_2"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", "false"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccResourcePoolResourceMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigResourcePoolVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceResourcePoolMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["ip_acl_1"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["ip_acl_2"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["snapshots_are_visible"])),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigResourcePoolVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_resource_pool" "resource_pool_ds" {
					  project_id       = stackit_sfs_resource_pool.resourcepool.project_id
					  resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceResourcePoolMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_resource_pool.resource_pool_ds", "id",
						"stackit_sfs_resource_pool.resourcepool", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_resource_pool.resource_pool_ds", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["availability_zone"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["performance_class"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["size_gigabytes"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "ip_acl.0", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["ip_acl_1"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "ip_acl.1", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["ip_acl_2"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "snapshots_are_visible", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["snapshots_are_visible"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigResourcePoolVarsMax,
				ResourceName:    "stackit_sfs_resource_pool.resourcepool",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					res, found := s.RootModule().Resources["stackit_sfs_resource_pool.resourcepool"]
					if !found {
						return "", fmt.Errorf("could not find resource stackit_sfs_resource_pool.resourcepool")
					}
					resourcepoolId, ok := res.Primary.Attributes["resource_pool_id"]
					if !ok {
						return "", fmt.Errorf("resource pool id attribute not found")
					}
					return testutil.ProjectId + "," + testutil.Region + "," + resourcepoolId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigResourcePoolVarsMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceResourcePoolMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sfs_resource_pool.resourcepool", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["ip_acl_1"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["ip_acl_2"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["snapshots_are_visible"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccShareResourceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigShareVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceShareMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMin["space_hard_limit_gigabytes"])),
					resource.TestCheckNoResourceAttr("stackit_sfs_share.share", "export_policy"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "mount_path"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigShareVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_share" "share_ds" {
					  project_id       = stackit_sfs_resource_pool.resourcepool.project_id
					  resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id
					  share_id         = stackit_sfs_share.share.share_id
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceShareMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_share.share_ds", "id",
						"stackit_sfs_share.share", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_share.share_ds", "share_id",
						"stackit_sfs_share.share", "share_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_share.share_ds", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "name", testutil.ConvertConfigVariable(testConfigShareVarsMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMin["space_hard_limit_gigabytes"])),
					resource.TestCheckNoResourceAttr("data.stackit_sfs_share.share_ds", "export_policy"),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "mount_path"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigShareVarsMin,
				ResourceName:    "stackit_sfs_share.share",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					res, found := s.RootModule().Resources["stackit_sfs_share.share"]
					if !found {
						return "", fmt.Errorf("could not find resource stackit_sfs_share.share")
					}
					resourcepoolId, ok := res.Primary.Attributes["resource_pool_id"]
					if !ok {
						return "", fmt.Errorf("resource pool id attribute not found")
					}
					shareId, ok := res.Primary.Attributes["share_id"]
					if !ok {
						return "", fmt.Errorf("share id attribute not found")
					}
					return testutil.ProjectId + "," + testutil.Region + "," + resourcepoolId + "," + shareId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigShareVarsMinUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceShareMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sfs_share.share", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMinUpdated()["space_hard_limit_gigabytes"])),
					resource.TestCheckNoResourceAttr("stackit_sfs_share.share", "export_policy"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "mount_path"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccShareResourceMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigShareVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceShareMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMax["space_hard_limit_gigabytes"])),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name",
					),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "mount_path"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigShareVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_share" "share_ds" {
					  project_id       = stackit_sfs_resource_pool.resourcepool.project_id
					  resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id
					  share_id         = stackit_sfs_share.share.share_id
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceShareMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_share.share_ds", "id",
						"stackit_sfs_share.share", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_share.share_ds", "share_id",
						"stackit_sfs_share.share", "share_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_share.share_ds", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "name", testutil.ConvertConfigVariable(testConfigShareVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMax["space_hard_limit_gigabytes"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_share.share_ds", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name",
					),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "mount_path"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigShareVarsMax,
				ResourceName:    "stackit_sfs_share.share",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					res, found := s.RootModule().Resources["stackit_sfs_share.share"]
					if !found {
						return "", fmt.Errorf("could not find resource stackit_sfs_share.share")
					}
					resourcepoolId, ok := res.Primary.Attributes["resource_pool_id"]
					if !ok {
						return "", fmt.Errorf("resource pool id attribute not found")
					}
					shareId, ok := res.Primary.Attributes["share_id"]
					if !ok {
						return "", fmt.Errorf("share id attribute not found")
					}
					return testutil.ProjectId + "," + testutil.Region + "," + resourcepoolId + "," + shareId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigShareVarsMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceShareMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sfs_share.share", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMaxUpdated()["space_hard_limit_gigabytes"])),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name",
					),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "mount_path"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccProjectLockMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectLockDestroyed,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigProjectLockVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceProjectLockConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_project_lock.project_lock", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_project_lock.project_lock", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_project_lock.project_lock", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_project_lock.project_lock", "lock_id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigProjectLockVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_project_lock" "project_lock" {
					  project_id = stackit_sfs_project_lock.project_lock.project_id
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceProjectLockConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_project_lock.project_lock", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_project_lock.project_lock", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_project_lock.project_lock", "id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_project_lock.project_lock", "lock_id",
						"stackit_sfs_project_lock.project_lock", "lock_id",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigProjectLockVarsMin,
				ResourceName:    "stackit_sfs_project_lock.project_lock",
				ImportStateIdFunc: func(_ *terraform.State) (string, error) {
					return testutil.ProjectId + "," + testutil.Region, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func createClient() (*sfs.APIClient, error) {
	client, err := sfs.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.SFSCustomEndpoint, false)...)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	return client, nil
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAccExportPolicyDestroy,
		testAccResourcePoolDestroyed,
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

func testAccExportPolicyDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := createClient()
	if err != nil {
		return err
	}

	policyToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_sfs_export_policy" {
			continue
		}
		// export policy transform id: "[projectId],[region],[policyId]"
		policyId := strings.Split(rs.Primary.ID, core.Separator)[1]
		policyToDestroy = append(policyToDestroy, policyId)
	}

	policiesResp, err := client.DefaultAPI.ListShareExportPolicies(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting policiesResp: %w", err)
	}

	// iterate over policiesResp
	policies := policiesResp.ShareExportPolicies
	for i := range policies {
		id := *policies[i].Id
		if utils.Contains(policyToDestroy, id) {
			_, err := client.DefaultAPI.DeleteShareExportPolicy(ctx, testutil.ProjectId, testutil.Region, id).Execute()
			if err != nil {
				return fmt.Errorf("deleting policy %s during CheckDestroy: %w", *policies[i].Id, err)
			}
		}
	}
	return nil
}

func testAccResourcePoolDestroyed(s *terraform.State) error {
	ctx := context.Background()
	client, err := createClient()
	if err != nil {
		return err
	}

	resourcePoolsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_sfs_resource_pool" {
			continue
		}
		// export policy transform id: "[projectId],[resource_pool_id]"
		resourcePoolId := strings.Split(rs.Primary.ID, core.Separator)[1]
		resourcePoolsToDestroy = append(resourcePoolsToDestroy, resourcePoolId)
	}

	resourcePoolsResp, err := client.DefaultAPI.ListResourcePools(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting resource pools: %w", err)
	}

	// iterate over policiesResp
	for _, pool := range resourcePoolsResp.GetResourcePools() {
		id := pool.Id

		if utils.Contains(resourcePoolsToDestroy, *id) {
			shares, err := client.DefaultAPI.ListShares(ctx, testutil.ProjectId, testutil.Region, *id).Execute()
			if err != nil {
				return fmt.Errorf("cannot list shares: %w", err)
			}
			if shares.Shares != nil {
				for _, share := range shares.Shares {
					_, err := client.DefaultAPI.DeleteShare(ctx, testutil.ProjectId, testutil.Region, *id, *share.Id).Execute()
					if err != nil {
						return fmt.Errorf("cannot delete share %q in pool %q: %w", *share.Id, *id, err)
					}
				}
			}

			_, err = client.DefaultAPI.DeleteResourcePool(ctx, testutil.ProjectId, testutil.Region, *id).
				Execute()
			if err != nil {
				return fmt.Errorf("deleting resourcepool %s during CheckDestroy: %w", *pool.Id, err)
			}
		}
	}
	return nil
}

func testAccProjectLockDestroyed(s *terraform.State) error {
	ctx := context.Background()
	client, err := createClient()
	if err != nil {
		return err
	}

	var errs []error

	type projectLock struct {
		ProjectId string
		Region    string
	}

	var projectLocksToDestroy []projectLock
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_sfs_project_lock" {
			continue
		}
		// transform id: "[projectId],[region]"
		projectId := rs.Primary.Attributes["project_id"]
		region := rs.Primary.Attributes["region"]
		projectLocksToDestroy = append(projectLocksToDestroy,
			projectLock{
				ProjectId: projectId,
				Region:    region,
			},
		)
	}

	for _, lock := range projectLocksToDestroy {
		_, err := client.DefaultAPI.GetLock(ctx, lock.Region, lock.ProjectId).Execute()
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			ok := errors.As(err, &oapiErr)
			if !(ok && oapiErr.StatusCode == http.StatusNotFound) {
				errs = append(errs, err)
			}
			continue
		}

		_, err = client.DefaultAPI.DisableLock(ctx, lock.Region, lock.ProjectId).Execute()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
