package sfs_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var exportPolicyResource = map[string]string{
	"name":            fmt.Sprintf("acc-sfs-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"project_id":      testutil.ProjectId,
	"region":          "eu01",
	"ip_acl_1":        "172.16.0.0/24",
	"ip_acl_2":        "172.16.0.250/32",
	"ip_acl_1_update": "172.17.0.0/24",
	"ip_acl_2_update": "172.17.0.250/32",
}

func resourceConfigExportPolicy() string {
	return fmt.Sprintf(`
		%s

		resource "stackit_sfs_export_policy" "exportpolicy" {
		project_id        = "%s"
		name              = "%s"
		rules = [
			{
			ip_acl = [%q, %q]
			order = 1
			}
		]
		}
	`,
		testutil.SFSProviderConfig(),
		exportPolicyResource["project_id"],
		exportPolicyResource["name"],
		exportPolicyResource["ip_acl_1"],
		exportPolicyResource["ip_acl_2"],
	)
}

func resourceConfigUpdateExportPolicy() string {
	return fmt.Sprintf(`
		%s

		resource "stackit_sfs_export_policy" "exportpolicy" {
		project_id        = "%s"
		name              = "%s"
		rules = [
			{
			ip_acl = [%q, %q]
			order = 1
			},
			{
			ip_acl = [%q, %q]
			order = 2
			}
		]
		}
	`,
		testutil.SFSProviderConfig(),
		exportPolicyResource["project_id"],
		exportPolicyResource["name"],
		exportPolicyResource["ip_acl_1"],
		exportPolicyResource["ip_acl_2"],
		exportPolicyResource["ip_acl_1_update"],
		exportPolicyResource["ip_acl_2_update"],
	)
}

var (
	testCreateResourcePool = map[string]string{
		"providerConfig":        testutil.SFSProviderConfig(),
		"name":                  fmt.Sprintf("acc-sfs-resource-pool-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
		"project_id":            testutil.ProjectId,
		"availability_zone":     "eu01-m",
		"performance_class":     "Standard",
		"acl":                   `["192.168.42.1/32", "192.168.42.2/32"]`,
		"size_gigabytes":        "500",
		"snapshots_are_visible": "true",
	}

	testUpdateResourcePool = map[string]string{
		"providerConfig":        testutil.SFSProviderConfig(),
		"name":                  fmt.Sprintf("acc-sfs-resource-pool-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
		"project_id":            testutil.ProjectId,
		"availability_zone":     "eu01-m",
		"performance_class":     "Premium",
		"acl":                   `["192.168.52.1/32", "192.168.52.2/32"]`,
		"size_gigabytes":        "500",
		"snapshots_are_visible": "false",
	}
)

func resourcePoolConfig(configParams map[string]string) string {
	tmpl := template.Must(template.New("config").
		Parse(`
		{{.providerConfig}}

		resource "stackit_sfs_resource_pool" "resourcepool" {
			project_id        =  "{{.project_id}}"
			name              = "{{.name}}"
			availability_zone = "{{.availability_zone}}"
			performance_class = "{{.performance_class}}"
			size_gigabytes = {{.size_gigabytes}}
			ip_acl = {{.acl}}
			snapshots_are_visible = {{.snapshots_are_visible}}
		}
		
		data "stackit_sfs_resource_pool" "resource_pool_ds" {
			project_id      = stackit_sfs_resource_pool.resourcepool.project_id
			resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id
		}
	`))
	var buffer strings.Builder
	if err := tmpl.ExecuteTemplate(&buffer, "config", configParams); err != nil {
		panic(fmt.Sprintf("cannot render template: %v", err))
	}
	return buffer.String()
}

var (
	testCreateShare = map[string]string{
		"providerConfig":             testutil.SFSProviderConfig(),
		"resource_pool_name":         fmt.Sprintf("acc-sfs-resource-pool-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
		"name":                       fmt.Sprintf("acc-sfs-share-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
		"project_id":                 testutil.ProjectId,
		"region":                     "eu01",
		"space_hard_limit_gigabytes": "42",
	}

	testUpdateShare = map[string]string{
		"providerConfig":             testutil.SFSProviderConfig(),
		"resource_pool_name":         fmt.Sprintf("acc-sfs-resource-pool-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
		"name":                       fmt.Sprintf("acc-sfs-share-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
		"project_id":                 testutil.ProjectId,
		"region":                     "eu02",
		"space_hard_limit_gigabytes": "42",
	}
)

func nsfShareConfig(configParams map[string]string) string {
	tmpl := template.Must(template.New("config").
		Parse(`
		{{.providerConfig}}


		resource "stackit_sfs_resource_pool" "resourcepool" {
			project_id        =  "{{.project_id}}"
			name              = "{{.resource_pool_name}}"
			availability_zone = "eu01-m"
			performance_class = "Standard"
			size_gigabytes = 512
			ip_acl = ["192.168.42.1/32"]
			region = "eu01"
		}

		resource "stackit_sfs_export_policy" "exportpolicy" {
			project_id        = "{{.project_id}}"
			name              = "{{.name}}"
			rules = [
				{
					ip_acl = ["192.168.2.0/24"]
					order = 1
				}
			]
		}

		resource "stackit_sfs_share" "share" {
			project_id                 =  "{{.project_id}}"
			resource_pool_id            = stackit_sfs_resource_pool.resourcepool.resource_pool_id
			name                       = "{{.name}}"
			export_policy              = stackit_sfs_export_policy.exportpolicy.name
			space_hard_limit_gigabytes = {{.space_hard_limit_gigabytes}}
		}

		data "stackit_sfs_share" "share_ds" {
			project_id        =  "{{.project_id}}"
			resource_pool_id = stackit_sfs_resource_pool.resourcepool.resource_pool_id
			share_id     = stackit_sfs_share.share.share_id
		}

	`))
	var buffer strings.Builder
	if err := tmpl.ExecuteTemplate(&buffer, "config", configParams); err != nil {
		panic(fmt.Sprintf("cannot render template: %v", err))
	}
	return buffer.String()
}

func TestAccExportPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccExportPolicyDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: resourceConfigExportPolicy(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", exportPolicyResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", exportPolicyResource["name"]),
					// check rule
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", exportPolicyResource["ip_acl_1"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", exportPolicyResource["ip_acl_2"]),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"),
				),
			},
			// data source
			{
				Config: fmt.Sprintf(`
									%s

									data "stackit_sfs_export_policy" "policy_data_test" {
										project_id  = stackit_sfs_export_policy.exportpolicy.project_id
										policy_id  = stackit_sfs_export_policy.exportpolicy.policy_id
									}

									`,
					resourceConfigExportPolicy(),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", exportPolicyResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", exportPolicyResource["name"]),
					// check rule
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", exportPolicyResource["ip_acl_1"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", exportPolicyResource["ip_acl_2"]),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"),

					// data source
					resource.TestCheckResourceAttr("data.stackit_sfs_export_policy.policy_data_test", "project_id", exportPolicyResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_export_policy.policy_data_test", "policy_id"),
				),
			},
			// Import
			{
				ResourceName: "stackit_sfs_export_policy.exportpolicy",
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
				Config: resourceConfigUpdateExportPolicy(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "project_id", exportPolicyResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_sfs_export_policy.exportpolicy", "id"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "name", exportPolicyResource["name"]),
					// check rules
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.order", "1"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.0", exportPolicyResource["ip_acl_1"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.ip_acl.1", exportPolicyResource["ip_acl_2"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.0.super_user", "true"),

					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.order", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.0", exportPolicyResource["ip_acl_1_update"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.ip_acl.1", exportPolicyResource["ip_acl_2_update"]),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.read_only", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.set_uuid", "false"),
					resource.TestCheckResourceAttr("stackit_sfs_export_policy.exportpolicy", "rules.1.super_user", "true"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccResourcePoolResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccResourcePoolDestroyed,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: resourcePoolConfig(testCreateResourcePool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testCreateResourcePool["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testCreateResourcePool["name"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testCreateResourcePool["availability_zone"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testCreateResourcePool["performance_class"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testCreateResourcePool["size_gigabytes"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", "192.168.42.1/32"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", "192.168.42.2/32"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "snapshots_are_visible", testCreateResourcePool["snapshots_are_visible"]),

					// datasource
					resource.TestCheckResourceAttr("data.stackit_sfs_resource_pool.resource_pool_ds", "project_id", testCreateResourcePool["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_resource_pool.resource_pool_ds", "resource_pool_id"),
				),
			},

			{ // import
				ResourceName: "stackit_sfs_resource_pool.resourcepool",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					res, found := s.RootModule().Resources["stackit_sfs_resource_pool.resourcepool"]
					if !found {
						return "", fmt.Errorf("could not find resource stackit_sfs_resource_pool.resourcepool")
					}
					resourcepoolId, ok := res.Primary.Attributes["resource_pool_id"]
					if !ok {
						return "", fmt.Errorf("resource pool id attribute not found")
					}
					return testCreateResourcePool["project_id"] + "," + testutil.Region + "," + resourcepoolId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update
			{
				Config: resourcePoolConfig(testUpdateResourcePool),
				Check: resource.ComposeAggregateTestCheckFunc(
					// resource
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "project_id", testUpdateResourcePool["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_sfs_resource_pool.resourcepool", "resource_pool_id"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "name", testUpdateResourcePool["name"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "availability_zone", testUpdateResourcePool["availability_zone"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "performance_class", testUpdateResourcePool["performance_class"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "size_gigabytes", testUpdateResourcePool["size_gigabytes"]),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.0", "192.168.52.1/32"),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "ip_acl.1", "192.168.52.2/32"),
				),
			},
		},
	})
}

func TestAccShareResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccResourcePoolDestroyed,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: nsfShareConfig(testCreateShare),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testCreateShare["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testCreateShare["name"]),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testCreateShare["space_hard_limit_gigabytes"]),
					resource.TestCheckResourceAttrPair("stackit_sfs_share.share", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name"),

					// datasource
					resource.TestCheckResourceAttr("data.stackit_sfs_share.share_ds", "project_id", testCreateResourcePool["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("data.stackit_sfs_share.share_ds", "share_id"),
				),
			},

			{ // import
				ResourceName: "stackit_sfs_share.share",
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
					return testCreateResourcePool["project_id"] + "," + testutil.Region + "," + resourcepoolId + "," + shareId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update
			{
				Config: nsfShareConfig(testUpdateShare),
				Check: resource.ComposeAggregateTestCheckFunc(
					// resource
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "project_id", testUpdateShare["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "resource_pool_id"),
					resource.TestCheckResourceAttrSet("stackit_sfs_share.share", "share_id"),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "name", testUpdateShare["name"]),
					resource.TestCheckResourceAttr("stackit_sfs_share.share", "space_hard_limit_gigabytes", testUpdateShare["space_hard_limit_gigabytes"]),
					resource.TestCheckResourceAttrPair("stackit_sfs_share.share", "export_policy",
						"stackit_sfs_export_policy.exportpolicy", "name"),
				),
			},
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
			config.WithEndpoint(testutil.SFSCustomEndpoint),
			config.WithTokenEndpoint(testutil.TokenCustomEndpoint),
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

	policiesResp, err := client.ListShareExportPoliciesExecute(ctx, testutil.ProjectId, exportPolicyResource["region"])
	if err != nil {
		return fmt.Errorf("getting policiesResp: %w", err)
	}

	// iterate over policiesResp
	policies := *policiesResp.ShareExportPolicies
	for i := range policies {
		id := *policies[i].Id
		if utils.Contains(policyToDestroy, id) {
			_, err := client.DeleteShareExportPolicy(ctx, testutil.ProjectId, exportPolicyResource["region"], id).Execute()
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

	region := testutil.Region
	resourcePoolsResp, err := client.ListResourcePoolsExecute(ctx, testutil.ProjectId, region)
	if err != nil {
		return fmt.Errorf("getting resource pools: %w", err)
	}

	// iterate over policiesResp
	for _, pool := range resourcePoolsResp.GetResourcePools() {
		id := pool.Id

		if utils.Contains(resourcePoolsToDestroy, *id) {
			shares, err := client.ListSharesExecute(ctx, testutil.ProjectId, region, *id)
			if err != nil {
				return fmt.Errorf("cannot list shares: %w", err)
			}
			if shares.Shares != nil {
				for _, share := range *shares.Shares {
					_, err := client.DeleteShareExecute(ctx, testutil.ProjectId, region, *id, *share.Id)
					if err != nil {
						return fmt.Errorf("cannot delete share %q in pool %q: %w", *share.Id, *id, err)
					}
				}
			}

			_, err = client.DeleteResourcePool(ctx, testutil.ProjectId, region, *id).
				Execute()
			if err != nil {
				return fmt.Errorf("deleting resourcepool %s during CheckDestroy: %w", *pool.Id, err)
			}
		}
	}
	return nil
}
