package ske_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	coreConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/stackit-sdk-go/services/ske/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	minTestName = "acc-min" + acctest.RandStringFromCharSet(3, acctest.CharSetAlpha)
	maxTestName = "acc-max" + acctest.RandStringFromCharSet(3, acctest.CharSetAlpha)
)

var (
	//go:embed testdata/resource-min.tf
	resourceMin string

	//go:embed testdata/resource-max.tf
	resourceMax string
)

var skeProviderOptions = NewSkeProviderOptions("flatcar")

var testConfigVarsMin = config.Variables{
	"project_id":                  config.StringVariable(testutil.ProjectId),
	"name":                        config.StringVariable(minTestName),
	"nodepool_availability_zone1": config.StringVariable(fmt.Sprintf("%s-1", testutil.Region)),
	"nodepool_machine_type":       config.StringVariable("g1.2"),
	"nodepool_minimum":            config.StringVariable("1"),
	"nodepool_maximum":            config.StringVariable("2"),
	"nodepool_name":               config.StringVariable("np-acc-test"),
	"kubernetes_version_min":      config.StringVariable(skeProviderOptions.GetCreateK8sVersion()),
	"maintenance_enable_machine_image_version_updates": config.StringVariable("true"),
	"maintenance_enable_kubernetes_version_updates":    config.StringVariable("true"),
	"maintenance_start": config.StringVariable("02:00:00+01:00"),
	"maintenance_end":   config.StringVariable("04:00:00+01:00"),
	"region":            config.StringVariable(testutil.Region),
}

var testConfigVarsMax = config.Variables{
	"project_id":                                       config.StringVariable(testutil.ProjectId),
	"organization_id":                                  config.StringVariable(testutil.OrganizationId),
	"name":                                             config.StringVariable(maxTestName),
	"nodepool_availability_zone1":                      config.StringVariable(fmt.Sprintf("%s-1", testutil.Region)),
	"nodepool_machine_type":                            config.StringVariable("g1.2"),
	"nodepool_minimum":                                 config.StringVariable("1"),
	"nodepool_maximum":                                 config.StringVariable("2"),
	"nodepool_name":                                    config.StringVariable("np-acc-test"),
	"nodepool_allow_system_components":                 config.StringVariable("true"),
	"nodepool_cri":                                     config.StringVariable("containerd"),
	"nodepool_label_value":                             config.StringVariable("value"),
	"nodepool_max_surge":                               config.StringVariable("1"),
	"nodepool_max_unavailable":                         config.StringVariable("1"),
	"nodepool_os_name":                                 config.StringVariable(skeProviderOptions.nodePoolOsName),
	"nodepool_os_version_min":                          config.StringVariable(skeProviderOptions.GetCreateMachineVersion()),
	"nodepool_taints_effect":                           config.StringVariable("PreferNoSchedule"),
	"nodepool_taints_key":                              config.StringVariable("tkey"),
	"nodepool_taints_value":                            config.StringVariable("tvalue"),
	"nodepool_volume_size":                             config.StringVariable("20"),
	"nodepool_volume_type":                             config.StringVariable("storage_premium_perf0"),
	"ext_acl_enabled":                                  config.StringVariable("true"),
	"ext_acl_allowed_cidr1":                            config.StringVariable("10.0.100.0/24"),
	"ext_argus_enabled":                                config.StringVariable("false"),
	"ext_dns_enabled":                                  config.StringVariable("true"),
	"nodepool_hibernations1_start":                     config.StringVariable("0 18 * * *"),
	"nodepool_hibernations1_end":                       config.StringVariable("59 23 * * *"),
	"nodepool_hibernations1_timezone":                  config.StringVariable("Europe/Berlin"),
	"kubernetes_version_min":                           config.StringVariable(skeProviderOptions.GetCreateK8sVersion()),
	"maintenance_enable_machine_image_version_updates": config.StringVariable("true"),
	"maintenance_enable_kubernetes_version_updates":    config.StringVariable("true"),
	"maintenance_start":                                config.StringVariable("02:00:00+01:00"),
	"maintenance_end":                                  config.StringVariable("04:00:00+01:00"),
	"region":                                           config.StringVariable(testutil.Region),
	"expiration":                                       config.StringVariable("3600"),
	"refresh":                                          config.StringVariable("true"),
	"dns_zone_name":                                    config.StringVariable("acc-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)),
	"dns_name":                                         config.StringVariable("acc-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlpha) + ".runs.onstackit.cloud"),
}

func configVarsMinUpdated() config.Variables {
	updatedConfig := maps.Clone(testConfigVarsMin)
	updatedConfig["kubernetes_version_min"] = config.StringVariable(skeProviderOptions.GetUpdateK8sVersion())

	return updatedConfig
}

func configVarsMaxUpdated() config.Variables {
	updatedConfig := maps.Clone(testConfigVarsMax)
	updatedConfig["kubernetes_version_min"] = config.StringVariable(skeProviderOptions.GetUpdateK8sVersion())
	updatedConfig["nodepool_os_version_min"] = config.StringVariable(skeProviderOptions.GetUpdateMachineVersion())
	updatedConfig["maintenance_end"] = config.StringVariable("03:03:03+00:00")

	return updatedConfig
}

func TestAccSKEMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSKEDestroy,
		Steps: []resource.TestStep{

			// 1) Creation
			{
				Config:          testutil.SKEProviderConfig() + "\n" + resourceMin,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// cluster data
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.0", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_availability_zone1"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.machine_type", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_machine_type"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.maximum", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_maximum"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.minimum", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_minimum"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_name"])),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "node_pools.0.os_version_used"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_kubernetes_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_kubernetes_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_machine_image_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_machine_image_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.start", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_start"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.end", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_end"])),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "region"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "kubernetes_version_used"),

					// Kubeconfig
					resource.TestCheckResourceAttrPair(
						"stackit_ske_kubeconfig.kubeconfig", "project_id",
						"stackit_ske_cluster.cluster", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_ske_kubeconfig.kubeconfig", "cluster_name",
						"stackit_ske_cluster.cluster", "name",
					),
				),
			},
			// 2) Data source
			{
				Config:          resourceMin,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(

					// cluster data
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "id", fmt.Sprintf("%s,%s,%s",
						testutil.ConvertConfigVariable(testConfigVarsMin["project_id"]),
						testutil.Region,
						testutil.ConvertConfigVariable(testConfigVarsMin["name"]),
					)),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.availability_zones.0", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_availability_zone1"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.machine_type", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_machine_type"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.maximum", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_maximum"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.minimum", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_minimum"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_name"])),

					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster", "kubernetes_version_used"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_kubernetes_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_kubernetes_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_machine_image_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_machine_image_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.start", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_start"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.end", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_end"])),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "region"),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster", "kubernetes_version_used"),
				),
			},
			// 3) Import cluster
			{
				ResourceName:    "stackit_ske_cluster.cluster",
				ConfigVariables: testConfigVarsMin,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_ske_cluster.cluster"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_ske_cluster.cluster")
					}
					_, ok = r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// The fields are not provided in the SKE API when disabled, although set actively.
				ImportStateVerifyIgnore: []string{"kubernetes_version_min", "node_pools.0.os_version_min", "extensions.argus.%", "extensions.argus.argus_instance_id", "extensions.argus.enabled"},
			},
			// 4) Update kubernetes version, OS version and maintenance end, downgrade of kubernetes version
			{
				Config:          resourceMin,
				ConfigVariables: configVarsMinUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// cluster data
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.availability_zones.0", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_availability_zone1"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.machine_type", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_machine_type"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.maximum", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_maximum"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.minimum", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_minimum"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMin["nodepool_name"])),

					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster", "kubernetes_version_used"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_kubernetes_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_kubernetes_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_machine_image_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_machine_image_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.start", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_start"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.end", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_end"])),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster", "kubernetes_version_used"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "region"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccSKEMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSKEDestroy,
		Steps: []resource.TestStep{

			// 1) Creation
			{
				Config:          testutil.SKEProviderConfig() + "\n" + resourceMax,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// cluster data
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.0", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_availability_zone1"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.machine_type", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_machine_type"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.maximum", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_maximum"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.minimum", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_minimum"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_name"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.allow_system_components", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_allow_system_components"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.cri", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_cri"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.labels.label_key", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_label_value"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.max_surge", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_max_surge"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.max_unavailable", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_max_unavailable"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_name", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_os_name"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_version_min", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_os_version_min"])),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "node_pools.0.os_version_used"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.effect", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_taints_effect"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.key", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_taints_key"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.value", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_taints_value"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.volume_size", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_volume_size"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.volume_type", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_volume_type"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.enabled", testutil.ConvertConfigVariable(testConfigVarsMax["ext_acl_enabled"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.0", testutil.ConvertConfigVariable(testConfigVarsMax["ext_acl_allowed_cidr1"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.argus.enabled", testutil.ConvertConfigVariable(testConfigVarsMax["ext_argus_enabled"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.dns.enabled", testutil.ConvertConfigVariable(testConfigVarsMax["ext_dns_enabled"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.dns.zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.dns.zones.0", testutil.ConvertConfigVariable(testConfigVarsMax["dns_name"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.start", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_hibernations1_start"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.end", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_hibernations1_end"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.timezone", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_hibernations1_timezone"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "kubernetes_version_min", testutil.ConvertConfigVariable(testConfigVarsMax["kubernetes_version_min"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_kubernetes_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_kubernetes_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_machine_image_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_machine_image_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.start", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_start"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.end", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_end"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "region", testutil.ConvertConfigVariable(testConfigVarsMax["region"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "egress_address_ranges.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "egress_address_ranges.0"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "pod_address_ranges.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "pod_address_ranges.0"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "kubernetes_version_used"),

					// Kubeconfig
					resource.TestCheckResourceAttrPair(
						"stackit_ske_kubeconfig.kubeconfig", "project_id",
						"stackit_ske_cluster.cluster", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_ske_kubeconfig.kubeconfig", "cluster_name",
						"stackit_ske_cluster.cluster", "name",
					),
					resource.TestCheckResourceAttr("stackit_ske_kubeconfig.kubeconfig", "expiration", testutil.ConvertConfigVariable(testConfigVarsMax["expiration"])),
					resource.TestCheckResourceAttrSet("stackit_ske_kubeconfig.kubeconfig", "expires_at"),
				),
			},
			// 2) Data source
			{
				Config:          resourceMax,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(

					// cluster data
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "id", fmt.Sprintf("%s,%s,%s",
						testutil.ConvertConfigVariable(testConfigVarsMax["project_id"]),
						testutil.Region,
						testutil.ConvertConfigVariable(testConfigVarsMax["name"]),
					)),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.availability_zones.0", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_availability_zone1"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.machine_type", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_machine_type"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.maximum", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_maximum"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.minimum", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_minimum"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.name", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_name"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.allow_system_components", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_allow_system_components"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.cri", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_cri"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.labels.label_key", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_label_value"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.max_surge", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_max_surge"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.max_unavailable", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_max_unavailable"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.os_name", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_os_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster", "node_pools.0.os_version_used"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.taints.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.taints.0.effect", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_taints_effect"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.taints.0.key", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_taints_key"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.taints.0.value", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_taints_value"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.volume_size", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_volume_size"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.volume_type", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_volume_type"])),

					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "extensions.acl.enabled", testutil.ConvertConfigVariable(testConfigVarsMax["ext_acl_enabled"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.0", testutil.ConvertConfigVariable(testConfigVarsMax["ext_acl_allowed_cidr1"])),
					// no check for argus, as it was disabled in the setup
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "extensions.dns.enabled", testutil.ConvertConfigVariable(testConfigVarsMax["ext_dns_enabled"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "extensions.dns.zones.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "extensions.dns.zones.0", testutil.ConvertConfigVariable(testConfigVarsMax["dns_name"])),

					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "hibernations.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "hibernations.0.start", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_hibernations1_start"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "hibernations.0.end", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_hibernations1_end"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "hibernations.0.timezone", testutil.ConvertConfigVariable(testConfigVarsMax["nodepool_hibernations1_timezone"])),

					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster", "kubernetes_version_used"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "maintenance.enable_kubernetes_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_kubernetes_version_updates"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "maintenance.enable_machine_image_version_updates", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_enable_machine_image_version_updates"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "maintenance.start", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_start"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "maintenance.end", testutil.ConvertConfigVariable(testConfigVarsMax["maintenance_end"])),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "region", testutil.ConvertConfigVariable(testConfigVarsMax["region"])),

					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "egress_address_ranges.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster", "egress_address_ranges.0"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "pod_address_ranges.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster", "pod_address_ranges.0"),

					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster", "kubernetes_version_used"),
				),
			},
			// 3) Import cluster
			{
				ResourceName:    "stackit_ske_cluster.cluster",
				ConfigVariables: testConfigVarsMax,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_ske_cluster.cluster"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_ske_cluster.cluster")
					}
					_, ok = r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// The fields are not provided in the SKE API when disabled, although set actively.
				ImportStateVerifyIgnore: []string{"kubernetes_version_min", "node_pools.0.os_version_min", "extensions.argus.%", "extensions.argus.argus_instance_id", "extensions.argus.enabled"},
			},
			// 4) Update kubernetes version, OS version and maintenance end, downgrade of kubernetes version
			{
				Config:          resourceMax,
				ConfigVariables: configVarsMaxUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// cluster data
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["name"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_availability_zone1"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.machine_type", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_machine_type"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.maximum", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_maximum"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.minimum", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_minimum"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_name"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.allow_system_components", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_allow_system_components"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.cri", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_cri"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.labels.label_key", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_label_value"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.max_surge", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_max_surge"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.max_unavailable", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_max_unavailable"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_os_name"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_version_min", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_os_version_min"])),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "node_pools.0.os_version_used"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.effect", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_taints_effect"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.key", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_taints_key"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.value", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_taints_value"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.volume_size", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_volume_size"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.volume_type", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_volume_type"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.enabled", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ext_acl_enabled"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ext_acl_allowed_cidr1"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.argus.enabled", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ext_argus_enabled"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.dns.enabled", testutil.ConvertConfigVariable(configVarsMaxUpdated()["ext_dns_enabled"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.dns.zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.dns.zones.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["dns_name"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.start", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_hibernations1_start"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.end", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_hibernations1_end"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.timezone", testutil.ConvertConfigVariable(configVarsMaxUpdated()["nodepool_hibernations1_timezone"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "kubernetes_version_min", testutil.ConvertConfigVariable(configVarsMaxUpdated()["kubernetes_version_min"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_kubernetes_version_updates", testutil.ConvertConfigVariable(configVarsMaxUpdated()["maintenance_enable_kubernetes_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_machine_image_version_updates", testutil.ConvertConfigVariable(configVarsMaxUpdated()["maintenance_enable_machine_image_version_updates"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.start", testutil.ConvertConfigVariable(configVarsMaxUpdated()["maintenance_start"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.end", testutil.ConvertConfigVariable(configVarsMaxUpdated()["maintenance_end"])),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "region", testutil.ConvertConfigVariable(configVarsMaxUpdated()["region"])),

					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "egress_address_ranges.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "egress_address_ranges.0"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "pod_address_ranges.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "pod_address_ranges.0"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "kubernetes_version_used"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckSKEDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *ske.APIClient
	var err error
	if testutil.SKECustomEndpoint == "" {
		client, err = ske.NewAPIClient(
			coreConfig.WithRegion(testutil.Region),
		)
	} else {
		client, err = ske.NewAPIClient(
			coreConfig.WithEndpoint(testutil.SKECustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	clustersToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_ske_cluster" {
			continue
		}
		// cluster terraform ID: = "[project_id],[region],[cluster_name]"
		clusterName := strings.Split(rs.Primary.ID, core.Separator)[2]
		clustersToDestroy = append(clustersToDestroy, clusterName)
	}

	clustersResp, err := client.ListClusters(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting clustersResp: %w", err)
	}

	items := *clustersResp.Items
	for i := range items {
		if items[i].Name == nil {
			continue
		}
		if utils.Contains(clustersToDestroy, *items[i].Name) {
			_, err := client.DeleteClusterExecute(ctx, testutil.ProjectId, *items[i].Name)
			if err != nil {
				return fmt.Errorf("destroying cluster %s during CheckDestroy: %w", *items[i].Name, err)
			}
			_, err = wait.DeleteClusterWaitHandler(ctx, client, testutil.ProjectId, *items[i].Name).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying cluster %s during CheckDestroy: waiting for deletion %w", *items[i].Name, err)
			}
		}
	}
	return nil
}

type SkeProviderOptions struct {
	options        *ske.ProviderOptions
	nodePoolOsName string
}

// NewSkeProviderOptions fetches the latest available options from SKE.
func NewSkeProviderOptions(nodePoolOs string) *SkeProviderOptions {
	// skip if TF_ACC is not set
	if os.Getenv("TF_ACC") == "" {
		return &SkeProviderOptions{
			options:        nil,
			nodePoolOsName: nodePoolOs,
		}
	}

	ctx := context.Background()

	var client *ske.APIClient
	var err error

	if testutil.SKECustomEndpoint == "" {
		client, err = ske.NewAPIClient(coreConfig.WithRegion("eu01"))
	} else {
		client, err = ske.NewAPIClient(coreConfig.WithEndpoint(testutil.SKECustomEndpoint))
	}

	if err != nil {
		panic("failed to create SKE client: " + err.Error())
	}

	options, err := client.ListProviderOptions(ctx).Execute()
	if err != nil {
		panic("failed to fetch SKE provider options: " + err.Error())
	}

	return &SkeProviderOptions{
		options:        options,
		nodePoolOsName: nodePoolOs,
	}
}

// GetCreateK8sVersion returns the first supported Kubernetes version (used for create).
func (s *SkeProviderOptions) GetCreateK8sVersion() string {
	if s.options == nil || s.options.KubernetesVersions == nil {
		return ""
	}

	for _, v := range *s.options.KubernetesVersions {
		if v.State != nil && *v.State == "supported" && v.Version != nil {
			return *v.Version
		}
	}

	return ""
}

// GetUpdateK8sVersion returns the next supported Kubernetes version after the create version (used for update).
func (s *SkeProviderOptions) GetUpdateK8sVersion() string {
	if s.options == nil || s.options.KubernetesVersions == nil {
		return ""
	}

	supportedCount := 0

	for _, v := range *s.options.KubernetesVersions {
		if v.State != nil && *v.State == "supported" && v.Version != nil {
			supportedCount++
			if supportedCount == 2 {
				return *v.Version
			}
		}
	}

	return ""
}

// GetCreateMachineVersion returns the first supported machine image version (used for create).
func (s *SkeProviderOptions) GetCreateMachineVersion() string {
	if s.options == nil || s.options.MachineImages == nil {
		return ""
	}

	for _, mi := range *s.options.MachineImages {
		if mi.Name != nil && *mi.Name == s.nodePoolOsName && mi.Versions != nil {
			for _, v := range *mi.Versions {
				if v.State != nil && *v.State == "supported" && v.Version != nil {
					return *v.Version
				}
			}
		}
	}

	return ""
}

// GetUpdateMachineVersion returns the next supported version after the create version (used for update).
func (s *SkeProviderOptions) GetUpdateMachineVersion() string {
	if s.options == nil || s.options.MachineImages == nil {
		return ""
	}

	for _, mi := range *s.options.MachineImages {
		if mi.Name != nil && *mi.Name == s.nodePoolOsName && mi.Versions != nil {
			count := 0
			for _, v := range *mi.Versions {
				if v.State != nil && v.Version != nil {
					count++
					if count == 2 {
						return *v.Version
					}
				}
			}
		}
	}

	return ""
}
