package sfs_test

import (
	"context"
	_ "embed"
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
	//go:embed testdata/export-policy-max.tf
	resourceExportPolicyMaxConfig string

	//go:embed testdata/export-policy-min.tf
	resourceExportPolicyMinConfig string
)

// EXPORT POLICY - MAX

var (
	ip_acl_1        = "172.16.0.0/24"
	ip_acl_2        = "172.16.0.250/32"
	ip_acl_1_update = "172.17.0.0/24"
	ip_acl_2_update = "172.17.0.250/32"
)

var testConfigExportPolicyVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"rules": config.ListVariable(
		config.ObjectVariable(map[string]config.Variable{
			"ip_acl": config.ListVariable(config.StringVariable(ip_acl_1), config.StringVariable(ip_acl_2)),
			"order":  config.IntegerVariable(1),
		}),
	),
}

var testConfigExportPolicyVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigExportPolicyVarsMax)
	updatedConfig["rules"] = config.ListVariable(
		config.ObjectVariable(map[string]config.Variable{
			"ip_acl": config.ListVariable(config.StringVariable(ip_acl_1), config.StringVariable(ip_acl_2)),
			"order":  config.IntegerVariable(1),
		}),
		config.ObjectVariable(map[string]config.Variable{
			"ip_acl": config.ListVariable(config.StringVariable(ip_acl_1_update), config.StringVariable(ip_acl_2_update)),
			"order":  config.IntegerVariable(2),
		}),
	)
	return updatedConfig
}

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

func TestAccExportPolicyMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccExportPolicyDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigExportPolicyVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceExportPolicyMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMax["name"])),
					// check rule
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", ip_acl_1),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", ip_acl_2),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"),
				),
			},
			// data source
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
					// check rule
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", ip_acl_1),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", ip_acl_2),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"),

					// data source
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
					// check rules
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", ip_acl_1),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", ip_acl_2),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.order", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.0", ip_acl_1_update),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.1", ip_acl_2_update),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.super_user", "true"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccExportPolicyMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccExportPolicyDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigExportPolicyVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.SFSProviderConfig(), resourceExportPolicyMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", testutil.ConvertConfigVariable(testConfigExportPolicyVarsMin["name"])),
				),
			},
			// data source
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
