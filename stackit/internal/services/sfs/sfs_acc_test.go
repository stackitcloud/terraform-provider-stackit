package sfs_test

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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	coreConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
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
)

// EXPORT POLICY - MIN

var testConfigExportPolicyVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
}

var testConfigExportPolicyVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigExportPolicyVarsMin)
	updatedConfig["name"] = config.StringVariable("tf-acc-updated")
	return updatedConfig
}

// EXPORT POLICY - MAX

const (
	exportPolicyMaxIpAcl1       = "172.16.0.0/24"
	exportPolicyMaxIpAcl2       = "172.16.0.250/32"
	exportPolicyMaxIpAcl1Update = "172.17.0.0/24"
	exportPolicyMaxIpAcl2Update = "172.17.0.250/32"
)

var testConfigExportPolicyVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"rules": config.ListVariable(
		config.ObjectVariable(map[string]config.Variable{
			"ip_acl": config.ListVariable(config.StringVariable(exportPolicyMaxIpAcl1), config.StringVariable(exportPolicyMaxIpAcl2)),
			"order":  config.IntegerVariable(1),
		}),
	),
}

var testConfigExportPolicyVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigExportPolicyVarsMax)
	updatedConfig["rules"] = config.ListVariable(
		config.ObjectVariable(map[string]config.Variable{
			"ip_acl": config.ListVariable(config.StringVariable(exportPolicyMaxIpAcl1), config.StringVariable(exportPolicyMaxIpAcl2)),
			"order":  config.IntegerVariable(1),
		}),
		config.ObjectVariable(map[string]config.Variable{
			"ip_acl": config.ListVariable(config.StringVariable(exportPolicyMaxIpAcl1Update), config.StringVariable(exportPolicyMaxIpAcl2Update)),
			"order":  config.IntegerVariable(2),
		}),
	)
	return updatedConfig
}

// Resource Pool - MIN

const (
	resourcePoolMinIpAcl1       = "192.168.42.1/32"
	resourcePoolMinIpAcl2       = "192.168.42.2/32"
	resourcePoolMinIpAcl1Update = "172.17.0.0/24"
	resourcePoolMinIpAcl2Update = "172.17.0.250/32"
)

var testConfigResourcePoolVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"acl": config.ListVariable(
		config.StringVariable(resourcePoolMinIpAcl1),
		config.StringVariable(resourcePoolMinIpAcl2),
	),
	"availability_zone": config.StringVariable("eu01-m"),
	"performance_class": config.StringVariable("Standard"),
	"size_gigabytes":    config.IntegerVariable(500),
}

var testConfigResourcePoolVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigResourcePoolVarsMin)
	updatedConfig["performance_class"] = config.StringVariable("Premium")
	updatedConfig["acl"] = config.ListVariable(
		config.StringVariable(resourcePoolMinIpAcl1Update),
		config.StringVariable(resourcePoolMinIpAcl2Update),
	)
	return updatedConfig
}

// Resource Pool - MAX

const (
	resourcePoolMaxIpAcl1       = "192.168.42.1/32"
	resourcePoolMaxIpAcl2       = "192.168.42.2/32"
	resourcePoolMaxIpAcl1Update = "172.17.0.0/24"
	resourcePoolMaxIpAcl2Update = "172.17.0.250/32"
)

var testConfigResourcePoolVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"acl": config.ListVariable(
		config.StringVariable(resourcePoolMaxIpAcl1),
		config.StringVariable(resourcePoolMaxIpAcl2),
	),
	"availability_zone":     config.StringVariable("eu01-m"),
	"performance_class":     config.StringVariable("Standard"),
	"size_gigabytes":        config.IntegerVariable(500),
	"snapshots_are_visible": config.BoolVariable(true),
}

var testConfigResourcePoolVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigResourcePoolVarsMax)
	updatedConfig["performance_class"] = config.StringVariable("Premium")
	updatedConfig["snapshots_are_visible"] = config.BoolVariable(false)
	updatedConfig["acl"] = config.ListVariable(
		config.StringVariable(resourcePoolMaxIpAcl1Update),
		config.StringVariable(resourcePoolMaxIpAcl2Update),
	)
	return updatedConfig
}

// Share - MIN

var testConfigShareVarsMin = config.Variables{
	"project_id":                 config.StringVariable(testutil.ProjectId),
	"name":                       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"region":                     config.StringVariable("eu01"),
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
	"region":                     config.StringVariable("eu01"),
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

func TestAccExportPolicyMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigExportPolicyVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceExportPolicyMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
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
					testutil.SFSProviderConfig(), resourceExportPolicyMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMin["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "0"),
					// data source
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_export_policy.policy_data_test", "policy_id"),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceExportPolicyMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceExportPolicyMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", exportPolicyMaxIpAcl1),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", exportPolicyMaxIpAcl2),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"),
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
					testutil.SFSProviderConfig(), resourceExportPolicyMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", exportPolicyMaxIpAcl1),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", exportPolicyMaxIpAcl2),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"),

					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_export_policy.policy_data_test", "policy_id"),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceExportPolicyMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMaxUpdated()["name"])),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", exportPolicyMaxIpAcl1),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", exportPolicyMaxIpAcl2),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.order", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.0", exportPolicyMaxIpAcl1Update),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.1", exportPolicyMaxIpAcl2Update),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.super_user", "true"),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceResourcePoolMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", resourcePoolMinIpAcl1),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", resourcePoolMinIpAcl2),
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
					testutil.SFSProviderConfig(), resourceResourcePoolMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMin["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", resourcePoolMinIpAcl1),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", resourcePoolMinIpAcl2),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", "false"),

					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_resource_pool.resource_pool_ds", "resource_pool_id"),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceResourcePoolMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMinUpdated()["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", resourcePoolMinIpAcl1Update),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", resourcePoolMinIpAcl2Update),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceResourcePoolMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", resourcePoolMaxIpAcl1),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", resourcePoolMaxIpAcl2),
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
					testutil.SFSProviderConfig(), resourceResourcePoolMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", resourcePoolMaxIpAcl1),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", resourcePoolMaxIpAcl2),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMax["snapshots_are_visible"])),

					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_resource_pool.resource_pool_ds", "resource_pool_id"),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceResourcePoolMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["performance_class"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testutil.ConvertConfigVariable(testConfigResourcePoolVarsMaxUpdated()["size_gigabytes"])),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", resourcePoolMaxIpAcl1Update),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", resourcePoolMaxIpAcl2Update),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceShareMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMin["space_hard_limit_gigabytes"])),
					resource.TestCheckNoResourceAttr("stackit_sfs_share.share", "export_policy"),
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
					testutil.SFSProviderConfig(), resourceShareMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMin["space_hard_limit_gigabytes"])),
					resource.TestCheckNoResourceAttr("stackit_sfs_share.share", "export_policy"),

					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "share_id"),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceShareMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMinUpdated()["space_hard_limit_gigabytes"])),
					resource.TestCheckNoResourceAttr("stackit_sfs_share.share", "export_policy"),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceShareMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMax["space_hard_limit_gigabytes"])),
					resource.TestCheckResourceAttrPair("stackit_sfs_share.share", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name"),
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
					testutil.SFSProviderConfig(), resourceShareMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMax["space_hard_limit_gigabytes"])),
					resource.TestCheckResourceAttrPair("stackit_sfs_share.share", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name"),

					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "share_id"),
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
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceShareMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testutil.ConvertConfigVariable(testConfigShareVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testutil.ConvertConfigVariable(testConfigShareVarsMaxUpdated()["space_hard_limit_gigabytes"])),
					resource.TestCheckResourceAttrPair("stackit_sfs_share.share", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func createClient() (*sfs.APIClient, error) {
	var client *sfs.APIClient
	var err error
	if testutil.SFSCustomEndpoint == "" {
		client, err = sfs.NewAPIClient()
	} else {
		client, err = sfs.NewAPIClient(
			coreConfig.WithEndpoint(testutil.SFSCustomEndpoint),
			coreConfig.WithTokenEndpoint(testutil.TokenCustomEndpoint),
		)
	}
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

	policiesResp, err := client.ListShareExportPoliciesExecute(ctx, testutil.ProjectId, testutil.Region)
	if err != nil {
		return fmt.Errorf("getting policiesResp: %w", err)
	}

	// iterate over policiesResp
	policies := *policiesResp.ShareExportPolicies
	for i := range policies {
		id := *policies[i].Id
		if utils.Contains(policyToDestroy, id) {
			_, err := client.DeleteShareExportPolicy(ctx, testutil.ProjectId, testutil.Region, id).Execute()
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

	resourcePoolsResp, err := client.ListResourcePoolsExecute(ctx, testutil.ProjectId, testutil.Region)
	if err != nil {
		return fmt.Errorf("getting resource pools: %w", err)
	}

	// iterate over policiesResp
	for _, pool := range resourcePoolsResp.GetResourcePools() {
		id := pool.Id

		if utils.Contains(resourcePoolsToDestroy, *id) {
			shares, err := client.ListSharesExecute(ctx, testutil.ProjectId, testutil.Region, *id)
			if err != nil {
				return fmt.Errorf("cannot list shares: %w", err)
			}
			if shares.Shares != nil {
				for _, share := range *shares.Shares {
					_, err := client.DeleteShareExecute(ctx, testutil.ProjectId, testutil.Region, *id, *share.Id)
					if err != nil {
						return fmt.Errorf("cannot delete share %q in pool %q: %w", *share.Id, *id, err)
					}
				}
			}

			_, err = client.DeleteResourcePool(ctx, testutil.ProjectId, testutil.Region, *id).
				Execute()
			if err != nil {
				return fmt.Errorf("deleting resourcepool %s during CheckDestroy: %w", *pool.Id, err)
			}
		}
	}
	return nil
}
