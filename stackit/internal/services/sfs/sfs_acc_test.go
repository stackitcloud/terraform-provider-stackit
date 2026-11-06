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
	snapshot_policy "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/snapshot-policy"
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

	//go:embed testdata/snapshot-policies-datasource.tf
	snapshotPoliciesDataSourceConfig string
)

// EXPORT POLICY - MIN
func TestAccExportPolicyMin(t *testing.T) {
	var configVars = config.Variables{
		"project_id": config.StringVariable(testutil.ProjectId),
		"name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	}

	var configVarsUpdated = func() config.Variables {
		updatedConfig := config.Variables{}
		maps.Copy(updatedConfig, configVars)
		updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
		return updatedConfig
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: configVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceExportPolicyMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "policy_id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(configVars["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "0"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "labels.%", "0"),
				),
			},
			// Data source
			{
				ConfigVariables: configVars,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_export_policy" "policy_data_test" {
					  project_id = stackit_sfs_export_policy.exportpolicy.project_id
					  policy_id  = stackit_sfs_export_policy.exportpolicy.policy_id
					}
					`,
					testutil.NewConfigBuilder().BuildProviderConfig(), resourceExportPolicyMinConfig,
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
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "name", testutil.ConvertConfigVariable(configVars["name"])),

					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "labels.%", "0"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.#", "0"),
				),
			},
			// Import
			{
				ConfigVariables: configVars,
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
				ConfigVariables: configVarsUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceExportPolicyMinConfig),
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
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(configVarsUpdated()["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "labels.%", "0"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "0"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

// EXPORT POLICY - MAX
func TestAccExportPolicyMax(t *testing.T) {
	var configVars = config.Variables{
		"project_id":             config.StringVariable(testutil.ProjectId),
		"region":                 config.StringVariable(testutil.Region),
		"name":                   config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"first_rule_description": config.StringVariable("Some description"),
		"first_rule_ip_acl":      config.ListVariable(config.StringVariable("172.16.0.0/24"), config.StringVariable("172.16.0.250/32")),
		"first_rule_set_uuid":    config.BoolVariable(true),
		"second_rule_ip_acl":     config.ListVariable(config.StringVariable("172.16.0.0/24"), config.StringVariable("172.16.0.250/32")),
		"second_rule_read_only":  config.BoolVariable(true),
		"second_rule_super_user": config.BoolVariable(false),
		"label":                  config.StringVariable("foo"),
	}

	var configVarsUpdated = func() config.Variables {
		updatedConfig := config.Variables{}
		maps.Copy(updatedConfig, configVars)

		updatedConfig["first_rule_description"] = config.StringVariable("Some other description")
		updatedConfig["first_rule_ip_acl"] = config.ListVariable(config.StringVariable("172.17.0.0/24"), config.StringVariable("172.17.0.250/32"))
		updatedConfig["label"] = config.StringVariable("bar")

		return updatedConfig
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: configVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceExportPolicyMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "policy_id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(configVars["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.description", testutil.ConvertConfigVariable(configVars["first_rule_description"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					testutil.CheckListAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl", configVars["first_rule_ip_acl"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"), // default value
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", testutil.ConvertConfigVariable(configVars["first_rule_set_uuid"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"), // default value

					resource.TestCheckNoResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.description"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.order", "2"),
					testutil.CheckListAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl", configVars["second_rule_ip_acl"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.read_only", testutil.ConvertConfigVariable(configVars["second_rule_read_only"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.set_uuid", "false"), // default value
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.super_user", testutil.ConvertConfigVariable(configVars["second_rule_super_user"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "labels.label", testutil.ConvertConfigVariable(configVars["label"])),
				),
			},
			// Data source
			{
				ConfigVariables: configVars,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_export_policy" "policy_data_test" {
					  project_id = stackit_sfs_export_policy.exportpolicy.project_id
					  policy_id  = stackit_sfs_export_policy.exportpolicy.policy_id
					}
					`,
					testutil.NewConfigBuilder().BuildProviderConfig(), resourceExportPolicyMaxConfig,
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
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "name", testutil.ConvertConfigVariable(configVars["name"])),

					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.description", testutil.ConvertConfigVariable(configVars["first_rule_description"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.order", "1"),
					testutil.CheckListAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.ip_acl", configVars["first_rule_ip_acl"]),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.read_only", "false"), // default value
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.set_uuid", testutil.ConvertConfigVariable(configVars["first_rule_set_uuid"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.0.super_user", "true"), // default value

					resource.TestCheckNoResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.description"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.order", "2"),
					testutil.CheckListAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.ip_acl", configVars["second_rule_ip_acl"]),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.read_only", testutil.ConvertConfigVariable(configVars["second_rule_read_only"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.set_uuid", "false"), // default value
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "rules.1.super_user", testutil.ConvertConfigVariable(configVars["second_rule_super_user"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "labels.label", testutil.ConvertConfigVariable(configVars["label"])),
				),
			},
			// Import
			{
				ConfigVariables: configVars,
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
				ConfigVariables: configVarsUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceExportPolicyMaxConfig),
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
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(configVarsUpdated()["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.description", testutil.ConvertConfigVariable(configVarsUpdated()["first_rule_description"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					testutil.CheckListAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl", configVarsUpdated()["first_rule_ip_acl"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"), // default value
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", testutil.ConvertConfigVariable(configVarsUpdated()["first_rule_set_uuid"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"), // default value

					resource.TestCheckNoResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.description"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.order", "2"),
					testutil.CheckListAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl", configVarsUpdated()["second_rule_ip_acl"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.read_only", testutil.ConvertConfigVariable(configVarsUpdated()["second_rule_read_only"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.set_uuid", "false"), // default value
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.super_user", testutil.ConvertConfigVariable(configVarsUpdated()["second_rule_super_user"])),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "labels.label", testutil.ConvertConfigVariable(configVarsUpdated()["label"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

// RESOURCE POOL - MIN
func TestAccResourcePoolResourceMin(t *testing.T) {
	var configVars = config.Variables{
		"project_id":        config.StringVariable(testutil.ProjectId),
		"name":              config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"ip_acl":            config.ListVariable(config.StringVariable("192.168.42.1/32"), config.StringVariable("192.168.42.2/32")),
		"availability_zone": config.StringVariable("eu01-m"),
		"performance_class": config.StringVariable("Standard"),
		"size_gigabytes":    config.IntegerVariable(500),
	}

	var configVarsUpdated = func() config.Variables {
		updatedConfig := config.Variables{}
		maps.Copy(updatedConfig, configVars)
		updatedConfig["performance_class"] = config.StringVariable("Premium")
		updatedConfig["size_gigabytes"] = config.IntegerVariable(512)
		updatedConfig["ip_acl"] = config.ListVariable(config.StringVariable("172.17.0.0/24"), config.StringVariable("172.17.0.250/32"))
		return updatedConfig
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: configVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceResourcePoolMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(configVars["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(configVars["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(configVars["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(configVars["size_gigabytes"])),
					testutil.CheckListAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl", configVars["ip_acl"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", "false"),
					resource.TestCheckNoResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshot_policy"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "labels.%", "0"),
				),
			},
			// Data source
			{
				ConfigVariables: configVars,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_resource_pool" "resource_pool_ds" {
					  project_id       = stackit_sfs_resource_pool.resourcepool.project_id
					  resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id
					}
					`,
					testutil.NewConfigBuilder().BuildProviderConfig(), resourceResourcePoolMinConfig,
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
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "name", testutil.ConvertConfigVariable(configVars["name"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "availability_zone", testutil.ConvertConfigVariable(configVars["availability_zone"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "performance_class", testutil.ConvertConfigVariable(configVars["performance_class"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "size_gigabytes", testutil.ConvertConfigVariable(configVars["size_gigabytes"])),
					testutil.CheckListAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "ip_acl", configVars["ip_acl"]),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "snapshots_are_visible", "false"),
					resource.TestCheckNoResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "snapshot_policy"),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "labels.%", "0"),
				),
			},
			// Import
			{
				ConfigVariables: configVars,
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
				ConfigVariables: configVarsUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceResourcePoolMinConfig),
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
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(configVarsUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(configVarsUpdated()["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(configVarsUpdated()["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(configVarsUpdated()["size_gigabytes"])),
					testutil.CheckListAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl", configVarsUpdated()["ip_acl"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", "false"),
					resource.TestCheckNoResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshot_policy"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "labels.%", "0"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

// RESOURCE POOL - MAX
func TestAccResourcePoolResourceMax(t *testing.T) {
	var configVars = config.Variables{
		"project_id":            config.StringVariable(testutil.ProjectId),
		"name":                  config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"ip_acl":                config.ListVariable(config.StringVariable("192.168.42.1/32"), config.StringVariable("192.168.42.2/32")),
		"region":                config.StringVariable(testutil.Region),
		"availability_zone":     config.StringVariable("eu01-m"),
		"performance_class":     config.StringVariable("Standard"),
		"size_gigabytes":        config.IntegerVariable(512),
		"snapshots_are_visible": config.BoolVariable(true),
		"snapshot_policy_id":    config.StringVariable("78a4035a-2452-11f1-97cd-d039eac4b54e"),
		"label":                 config.StringVariable("foo"),
	}

	var configVarsUpdated = func() config.Variables {
		updatedConfig := config.Variables{}
		maps.Copy(updatedConfig, configVars)
		updatedConfig["performance_class"] = config.StringVariable("Premium")
		updatedConfig["snapshots_are_visible"] = config.BoolVariable(false)
		updatedConfig["size_gigabytes"] = config.IntegerVariable(1024)
		updatedConfig["ip_acl"] = config.ListVariable(config.StringVariable("172.17.0.0/24"), config.StringVariable("172.17.0.250/32"))
		updatedConfig["label"] = config.StringVariable("bar")
		return updatedConfig
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: configVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceResourcePoolMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(configVars["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(configVars["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(configVars["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(configVars["size_gigabytes"])),
					testutil.CheckListAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl", configVars["ip_acl"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", testutil.ConvertConfigVariable(configVars["snapshots_are_visible"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshot_policy.id", testutil.ConvertConfigVariable(configVars["snapshot_policy_id"])),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "snapshot_policy.name"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "labels.label", testutil.ConvertConfigVariable(configVars["label"])),
				),
			},
			// Data source
			{
				ConfigVariables: configVars,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_resource_pool" "resource_pool_ds" {
					  project_id       = stackit_sfs_resource_pool.resourcepool.project_id
					  resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id

					  timeouts = {
					    read = "20m"
					  }
					}
					`,
					testutil.NewConfigBuilder().BuildProviderConfig(), resourceResourcePoolMaxConfig,
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
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "name", testutil.ConvertConfigVariable(configVars["name"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "availability_zone", testutil.ConvertConfigVariable(configVars["availability_zone"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "performance_class", testutil.ConvertConfigVariable(configVars["performance_class"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "size_gigabytes", testutil.ConvertConfigVariable(configVars["size_gigabytes"])),
					testutil.CheckListAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "ip_acl", configVars["ip_acl"]),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "snapshots_are_visible", testutil.ConvertConfigVariable(configVars["snapshots_are_visible"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "snapshot_policy.id", testutil.ConvertConfigVariable(configVars["snapshot_policy_id"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_resource_pool.resource_pool_ds", "snapshot_policy.name",
						"stackit_sfs_resource_pool.resourcepool", "snapshot_policy.name",
					),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "labels.label", testutil.ConvertConfigVariable(configVars["label"])),
				),
			},
			// Import
			{
				ConfigVariables: configVars,
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
				ConfigVariables: configVarsUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceResourcePoolMaxConfig),
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
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(configVarsUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(configVarsUpdated()["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(configVarsUpdated()["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(configVarsUpdated()["size_gigabytes"])),
					testutil.CheckListAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl", configVarsUpdated()["ip_acl"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", testutil.ConvertConfigVariable(configVarsUpdated()["snapshots_are_visible"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshot_policy.id", testutil.ConvertConfigVariable(configVars["snapshot_policy_id"])),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "snapshot_policy.name"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "labels.label", testutil.ConvertConfigVariable(configVarsUpdated()["label"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

// SHARE - MIN
func TestAccShareResourceMin(t *testing.T) {
	var configVars = config.Variables{
		"project_id":                 config.StringVariable(testutil.ProjectId),
		"name":                       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"resource_pool_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"space_hard_limit_gigabytes": config.IntegerVariable(42),
	}

	var configVarsUpdated = func() config.Variables {
		updatedConfig := config.Variables{}
		maps.Copy(updatedConfig, configVars)
		updatedConfig["space_hard_limit_gigabytes"] = config.IntegerVariable(50)
		return updatedConfig
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: configVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceShareMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(configVars["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(configVars["space_hard_limit_gigabytes"])),
					resource.TestCheckNoResourceAttr("stackit_sfs_share.share", "export_policy"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "mount_path"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "labels.%", "0"),
				),
			},
			// Data source
			{
				ConfigVariables: configVars,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_share" "share_ds" {
					  project_id       = stackit_sfs_resource_pool.resourcepool.project_id
					  resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id
					  share_id         = stackit_sfs_share.share.share_id
					}
					`,
					testutil.NewConfigBuilder().BuildProviderConfig(), resourceShareMinConfig,
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
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "name", testutil.ConvertConfigVariable(configVars["name"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(configVars["space_hard_limit_gigabytes"])),
					resource.TestCheckNoResourceAttr("data.stackit_sfs_share.share_ds", "export_policy"),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "mount_path"),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "labels.%", "0"),
				),
			},
			// Import
			{
				ConfigVariables: configVars,
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
				ConfigVariables: configVarsUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceShareMinConfig),
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
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(configVarsUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(configVarsUpdated()["space_hard_limit_gigabytes"])),
					resource.TestCheckNoResourceAttr("stackit_sfs_share.share", "export_policy"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "mount_path"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "labels.%", "0"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

// SHARE - MAX
func TestAccShareResourceMax(t *testing.T) {
	var configVars = config.Variables{
		"project_id":                 config.StringVariable(testutil.ProjectId),
		"name":                       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"region":                     config.StringVariable(testutil.Region),
		"resource_pool_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"space_hard_limit_gigabytes": config.IntegerVariable(42),
		"export_policy_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"label":                      config.StringVariable("foo"),
	}

	var configVarsUpdated = func() config.Variables {
		updatedConfig := config.Variables{}
		maps.Copy(updatedConfig, configVars)
		updatedConfig["space_hard_limit_gigabytes"] = config.IntegerVariable(50)
		updatedConfig["label"] = config.StringVariable("bar")
		return updatedConfig
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: configVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceShareMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "resource_pool_id",
						"stackit_sfs_resource_pool.resourcepool", "resource_pool_id",
					),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(configVars["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(configVars["space_hard_limit_gigabytes"])),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name",
					),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "mount_path"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "labels.label", testutil.ConvertConfigVariable(configVars["label"])),
				),
			},
			// Data source
			{
				ConfigVariables: configVars,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_share" "share_ds" {
					  project_id       = stackit_sfs_resource_pool.resourcepool.project_id
					  resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id
					  share_id         = stackit_sfs_share.share.share_id
					}
					`,
					testutil.NewConfigBuilder().BuildProviderConfig(), resourceShareMaxConfig,
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
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "name", testutil.ConvertConfigVariable(configVars["name"])),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(configVars["space_hard_limit_gigabytes"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sfs_share.share_ds", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name",
					),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "mount_path"),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "labels.label", testutil.ConvertConfigVariable(configVars["label"])),
				),
			},
			// Import
			{
				ConfigVariables: configVars,
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
				ConfigVariables: configVarsUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceShareMaxConfig),
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
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(configVarsUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(configVarsUpdated()["space_hard_limit_gigabytes"])),
					resource.TestCheckResourceAttrPair(
						"stackit_sfs_share.share", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name",
					),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "mount_path"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "labels.label", testutil.ConvertConfigVariable(configVarsUpdated()["label"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

// PROJECT LOCK - MIN
func TestAccProjectLockMin(t *testing.T) {
	var configVarsMin = config.Variables{
		"project_id": config.StringVariable(testutil.ProjectId),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectLockDestroyed,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: configVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceProjectLockConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_project_lock.project_lock", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sfs_project_lock.project_lock", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_sfs_project_lock.project_lock", "id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_project_lock.project_lock", "lock_id"),
				),
			},
			// Data source
			{
				ConfigVariables: configVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_sfs_project_lock" "project_lock" {
					  project_id = stackit_sfs_project_lock.project_lock.project_id
					}
					`,
					testutil.NewConfigBuilder().BuildProviderConfig(), resourceProjectLockConfig,
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
				ConfigVariables: configVarsMin,
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

// SNAPSHOT POLICIES
func TestAccSnapshotPolicies(t *testing.T) {
	projectId := config.StringVariable(testutil.ProjectId)
	cfg := fmt.Sprintf(`
%s

%s`, testutil.NewConfigBuilder().BuildProviderConfig(), snapshotPoliciesDataSourceConfig)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// all / default
			{
				Config: cfg,
				ConfigVariables: config.Variables{
					"project_id": projectId,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_snapshot_policies.snapshot_policies", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_snapshot_policies.snapshot_policies", "immutable", snapshot_policy.ImmutableFilterAll),
				),
			},
			// immutable
			{
				Config: cfg,
				ConfigVariables: config.Variables{
					"project_id": projectId,
					"immutable":  config.StringVariable(snapshot_policy.ImmutableFilterImmutableOnly),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_snapshot_policies.snapshot_policies", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_snapshot_policies.snapshot_policies", "immutable", snapshot_policy.ImmutableFilterImmutableOnly),
				),
			},
			// mutable
			{
				Config: cfg,
				ConfigVariables: config.Variables{
					"project_id": projectId,
					"immutable":  config.StringVariable(snapshot_policy.ImmutableFilterMutableOnly),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_sfs_snapshot_policies.snapshot_policies", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_sfs_snapshot_policies.snapshot_policies", "immutable", snapshot_policy.ImmutableFilterMutableOnly),
				),
			},
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
		policyId := strings.Split(rs.Primary.ID, core.Separator)[2]
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
		// export policy transform id: "[project_id],[region],[resource_pool_id]"
		resourcePoolId := strings.Split(rs.Primary.ID, core.Separator)[2]
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
