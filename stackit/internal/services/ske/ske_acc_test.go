package ske_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/stackit-sdk-go/services/ske/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var clusterResource = map[string]string{
	"project_id":                                       testutil.ProjectId,
	"name":                                             fmt.Sprintf("cl-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"name_min":                                         fmt.Sprintf("cl-min-%s", acctest.RandStringFromCharSet(3, acctest.CharSetAlphaNum)),
	"kubernetes_version_min":                           "1.26",
	"kubernetes_version_used":                          "1.26.15",
	"kubernetes_version_min_new":                       "1.27",
	"kubernetes_version_used_new":                      "1.27.13",
	"nodepool_name":                                    "np-acc-test",
	"nodepool_name_min":                                "np-acc-min-test",
	"nodepool_machine_type":                            "b1.2",
	"nodepool_os_version_min":                          "3815.2",
	"nodepool_os_version_used":                         "3815.2.1",
	"nodepool_os_version_min_new":                      "3815.2.1",
	"nodepool_os_version_used_new":                     "3815.2.1",
	"nodepool_os_name":                                 "flatcar",
	"nodepool_minimum":                                 "2",
	"nodepool_maximum":                                 "3",
	"nodepool_max_surge":                               "1",
	"nodepool_max_unavailable":                         "1",
	"nodepool_volume_size":                             "20",
	"nodepool_volume_type":                             "storage_premium_perf0",
	"nodepool_zone":                                    "eu01-3",
	"nodepool_cri":                                     "containerd",
	"nodepool_label_key":                               "key",
	"nodepool_label_value":                             "value",
	"nodepool_taints_effect":                           "PreferNoSchedule",
	"nodepool_taints_key":                              "tkey",
	"nodepool_taints_value":                            "tvalue",
	"extensions_acl_enabled":                           "true",
	"extensions_acl_cidrs":                             "192.168.0.0/24",
	"extensions_argus_enabled":                         "false",
	"extensions_argus_instance_id":                     "aaaaaaaa-1111-2222-3333-444444444444", // A not-existing Argus ID let the creation time-out.
	"extensions_dns_enabled":                           "true",
	"extensions_dns_zones":                             "foo.onstackit.cloud", // Dummy DNS zone, replace when running the tests!
	"hibernations_start":                               "0 16 * * *",
	"hibernations_end":                                 "0 18 * * *",
	"hibernations_timezone":                            "Europe/Berlin",
	"maintenance_enable_kubernetes_version_updates":    "true",
	"maintenance_enable_machine_image_version_updates": "true",
	"maintenance_start":                                "01:23:45Z",
	"maintenance_end":                                  "05:00:00+02:00",
	"maintenance_end_new":                              "03:03:03+00:00",
	"kubeconfig_expiration":                            "3600",
}

func getConfig(kubernetesVersion, nodePoolMachineOSVersion string, maintenanceEnd *string) string {
	maintenanceEndTF := clusterResource["maintenance_end"]
	if maintenanceEnd != nil {
		maintenanceEndTF = *maintenanceEnd
	}
	return fmt.Sprintf(`
		%s

		resource "stackit_ske_cluster" "cluster" {
			project_id = "%s"
			name = "%s"
			kubernetes_version_min = "%s"
			node_pools = [{
				name = "%s"
				machine_type = "%s"
				minimum = "%s"
				maximum = "%s"
				max_surge = "%s"
				max_unavailable = "%s"
				os_name = "%s"
				os_version_min = "%s"
				volume_size = "%s"
				volume_type = "%s"
				cri = "%s"
				availability_zones = ["%s"]
				labels = {
					%s = "%s"
				}
				taints = [{
					effect = "%s"
					key    = "%s"
					value  = "%s"
				}]
			}]
			extensions = {
				acl = {
					enabled = %s
					allowed_cidrs = ["%s"]
				}
				argus = {
					enabled = %s
					argus_instance_id = "%s"
				}
				dns = {
					enabled = %s
					zones = ["%s"]
				}
			}
			hibernations = [{
				start    = "%s"
				end      = "%s"
				timezone = "%s"
			}]
			maintenance = {
				enable_kubernetes_version_updates = %s
				enable_machine_image_version_updates = %s
				start = "%s"
				end = "%s"
			}
		}

		resource "stackit_ske_kubeconfig" "kubeconfig" {
			project_id = stackit_ske_cluster.cluster.project_id
			cluster_name = stackit_ske_cluster.cluster.name
			expiration = "%s"
		}

		resource "stackit_ske_cluster" "cluster_min" {
			project_id = "%s"
			name = "%s"
			node_pools = [{
				name = "%s"
				machine_type = "%s"
				minimum = "%s"
				maximum = "%s"
				availability_zones = ["%s"]
			}]
			maintenance = {
				enable_kubernetes_version_updates = %s
				enable_machine_image_version_updates = %s
				start = "%s"
				end = "%s"
			}
		}
		`,
		testutil.SKEProviderConfig(),
		clusterResource["project_id"],
		clusterResource["name"],
		kubernetesVersion,
		clusterResource["nodepool_name"],
		clusterResource["nodepool_machine_type"],
		clusterResource["nodepool_minimum"],
		clusterResource["nodepool_maximum"],
		clusterResource["nodepool_max_surge"],
		clusterResource["nodepool_max_unavailable"],
		clusterResource["nodepool_os_name"],
		nodePoolMachineOSVersion,
		clusterResource["nodepool_volume_size"],
		clusterResource["nodepool_volume_type"],
		clusterResource["nodepool_cri"],
		clusterResource["nodepool_zone"],
		clusterResource["nodepool_label_key"],
		clusterResource["nodepool_label_value"],
		clusterResource["nodepool_taints_effect"],
		clusterResource["nodepool_taints_key"],
		clusterResource["nodepool_taints_value"],
		clusterResource["extensions_acl_enabled"],
		clusterResource["extensions_acl_cidrs"],
		clusterResource["extensions_argus_enabled"],
		clusterResource["extensions_argus_instance_id"],
		clusterResource["extensions_dns_enabled"],
		clusterResource["extensions_dns_zones"],
		clusterResource["hibernations_start"],
		clusterResource["hibernations_end"],
		clusterResource["hibernations_timezone"],
		clusterResource["maintenance_enable_kubernetes_version_updates"],
		clusterResource["maintenance_enable_machine_image_version_updates"],
		clusterResource["maintenance_start"],
		maintenanceEndTF,

		// Kubeconfig
		clusterResource["kubeconfig_expiration"],

		// Minimal
		clusterResource["project_id"],
		clusterResource["name_min"],
		clusterResource["nodepool_name_min"],
		clusterResource["nodepool_machine_type"],
		clusterResource["nodepool_minimum"],
		clusterResource["nodepool_maximum"],
		clusterResource["nodepool_zone"],
		clusterResource["maintenance_enable_kubernetes_version_updates"],
		clusterResource["maintenance_enable_machine_image_version_updates"],
		clusterResource["maintenance_start"],
		maintenanceEndTF,
	)
}

func TestAccSKE(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSKEDestroy,
		Steps: []resource.TestStep{

			// 1) Creation
			{
				Config: getConfig(clusterResource["kubernetes_version_min"], clusterResource["nodepool_os_version_min"], nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					// cluster data
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "name", clusterResource["name"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "kubernetes_version_min", clusterResource["kubernetes_version_min"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "kubernetes_version_used", clusterResource["kubernetes_version_used"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.name", clusterResource["nodepool_name"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.0", clusterResource["nodepool_zone"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_name", clusterResource["nodepool_os_name"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_version_min", clusterResource["nodepool_os_version_min"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_version_used", clusterResource["nodepool_os_version_used"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.machine_type", clusterResource["nodepool_machine_type"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.minimum", clusterResource["nodepool_minimum"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.maximum", clusterResource["nodepool_maximum"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.max_surge", clusterResource["nodepool_max_surge"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.max_unavailable", clusterResource["nodepool_max_unavailable"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.volume_type", clusterResource["nodepool_volume_type"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.volume_size", clusterResource["nodepool_volume_size"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", fmt.Sprintf("node_pools.0.labels.%s", clusterResource["nodepool_label_key"]), clusterResource["nodepool_label_value"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.effect", clusterResource["nodepool_taints_effect"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.key", clusterResource["nodepool_taints_key"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.value", clusterResource["nodepool_taints_value"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.cri", clusterResource["nodepool_cri"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.enabled", clusterResource["extensions_acl_enabled"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.0", clusterResource["extensions_acl_cidrs"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.argus.enabled", clusterResource["extensions_argus_enabled"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.argus.argus_instance_id", clusterResource["extensions_argus_instance_id"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extenisons.dns.enabled", clusterResource["extensions_dns_enabled"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extenisons.dns.zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extenisons.dns.zones.0", clusterResource["extensions_dns_zones"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.start", clusterResource["hibernations_start"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.end", clusterResource["hibernations_end"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.timezone", clusterResource["hibernations_timezone"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_kubernetes_version_updates", clusterResource["maintenance_enable_kubernetes_version_updates"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_machine_image_version_updates", clusterResource["maintenance_enable_machine_image_version_updates"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.start", clusterResource["maintenance_start"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.end", clusterResource["maintenance_end"]),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster", "kube_config"),

					// Kubeconfig

					resource.TestCheckResourceAttrPair(
						"stackit_ske_kubeconfig.kubeconfig", "project_id",
						"stackit_ske_cluster.cluster", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_ske_kubeconfig.kubeconfig", "cluster_name",
						"stackit_ske_cluster.cluster", "name",
					),
					resource.TestCheckResourceAttr("stackit_ske_kubeconfig.kubeconfig", "expiration", clusterResource["kubeconfig_expiration"]),
					resource.TestCheckResourceAttrSet("stackit_ske_kubeconfig.kubeconfig", "kube_config"),
					resource.TestCheckResourceAttrSet("stackit_ske_kubeconfig.kubeconfig", "expires_at"),

					// Minimal cluster
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "name", clusterResource["name_min"]),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "kubernetes_version_used"),
					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster_min", "allow_privileged_containers"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.name", clusterResource["nodepool_name_min"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.availability_zones.0", clusterResource["nodepool_zone"]),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "node_pools.0.os_name"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.os_version_used", clusterResource["nodepool_os_version_used"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.machine_type", clusterResource["nodepool_machine_type"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.minimum", clusterResource["nodepool_minimum"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.maximum", clusterResource["nodepool_maximum"]),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "node_pools.0.max_surge"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "node_pools.0.max_unavailable"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "node_pools.0.volume_type"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.volume_size", clusterResource["nodepool_volume_size"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.labels.%", "0"),
					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.taints"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.cri", clusterResource["nodepool_cri"]),
					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster_min", "extensions"),
					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster_min", "hibernations"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "maintenance.enable_kubernetes_version_updates"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "maintenance.enable_machine_image_version_updates"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "maintenance.start"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "maintenance.end"),
					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster_min", "kube_config"),
				),
			},
			// 2) Data source
			{
				Config: fmt.Sprintf(`
					%s
		
					data "stackit_ske_cluster" "cluster" {
						project_id = "%s"
						name = "%s"
						depends_on = [stackit_ske_cluster.cluster]
					}

			        data "stackit_ske_cluster" "cluster_min" {
						project_id = "%s"
						name = "%s"
						depends_on = [stackit_ske_cluster.cluster_min]
					}

						`,
					getConfig(clusterResource["kubernetes_version_min"], clusterResource["nodepool_os_version_min"], nil),
					clusterResource["project_id"],
					clusterResource["name"],
					clusterResource["project_id"],
					clusterResource["name_min"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// cluster data
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "id", fmt.Sprintf("%s,%s",
						clusterResource["project_id"],
						clusterResource["name"],
					)),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "project_id", clusterResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "name", clusterResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "kubernetes_version_used", clusterResource["kubernetes_version_used"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.name", clusterResource["nodepool_name"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.availability_zones.0", clusterResource["nodepool_zone"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.os_name", clusterResource["nodepool_os_name"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.machine_type", clusterResource["nodepool_machine_type"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.minimum", clusterResource["nodepool_minimum"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.maximum", clusterResource["nodepool_maximum"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.max_surge", clusterResource["nodepool_max_surge"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.max_unavailable", clusterResource["nodepool_max_unavailable"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.volume_type", clusterResource["nodepool_volume_type"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.volume_size", clusterResource["nodepool_volume_size"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", fmt.Sprintf("node_pools.0.labels.%s", clusterResource["nodepool_label_key"]), clusterResource["nodepool_label_value"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.taints.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.taints.0.effect", clusterResource["nodepool_taints_effect"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.taints.0.key", clusterResource["nodepool_taints_key"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.taints.0.value", clusterResource["nodepool_taints_value"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.cri", clusterResource["nodepool_cri"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "extensions.acl.enabled", clusterResource["extensions_acl_enabled"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.0", clusterResource["extensions_acl_cidrs"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "hibernations.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "hibernations.0.start", clusterResource["hibernations_start"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "hibernations.0.end", clusterResource["hibernations_end"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "hibernations.0.timezone", clusterResource["hibernations_timezone"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "hibernations.0.end", clusterResource["hibernations_end"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "maintenance.enable_kubernetes_version_updates", clusterResource["maintenance_enable_kubernetes_version_updates"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "maintenance.enable_machine_image_version_updates", clusterResource["maintenance_enable_machine_image_version_updates"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "maintenance.start", clusterResource["maintenance_start"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "maintenance.end", clusterResource["maintenance_end"]),

					resource.TestCheckNoResourceAttr("data.stackit_ske_cluster.cluster", "kube_config"), // when using the kubeconfig resource, the kubeconfig field becomes null

					// Minimal cluster
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster_min", "name", clusterResource["name_min"]),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster_min", "kubernetes_version_used"),
					resource.TestCheckNoResourceAttr("data.stackit_ske_cluster.cluster_min", "allow_privileged_containers"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster_min", "node_pools.0.name", clusterResource["nodepool_name_min"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster_min", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster_min", "node_pools.0.availability_zones.0", clusterResource["nodepool_zone"]),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster_min", "node_pools.0.os_name"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster_min", "node_pools.0.machine_type", clusterResource["nodepool_machine_type"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster_min", "node_pools.0.minimum", clusterResource["nodepool_minimum"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster_min", "node_pools.0.maximum", clusterResource["nodepool_maximum"]),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster_min", "node_pools.0.max_surge"),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster_min", "node_pools.0.max_unavailable"),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster_min", "node_pools.0.volume_type"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster", "node_pools.0.volume_size", clusterResource["nodepool_volume_size"]),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster_min", "node_pools.0.labels.%", "0"),
					resource.TestCheckNoResourceAttr("data.stackit_ske_cluster.cluster_min", "node_pools.0.taints"),
					resource.TestCheckResourceAttr("data.stackit_ske_cluster.cluster_min", "node_pools.0.cri", clusterResource["nodepool_cri"]),
					resource.TestCheckNoResourceAttr("data.stackit_ske_cluster.cluster_min", "extensions"),
					resource.TestCheckNoResourceAttr("data.stackit_ske_cluster.cluster_min", "hibernations"),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster_min", "maintenance.enable_kubernetes_version_updates"),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster_min", "maintenance.enable_machine_image_version_updates"),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster_min", "maintenance.start"),
					resource.TestCheckResourceAttrSet("data.stackit_ske_cluster.cluster_min", "maintenance.end"),
					resource.TestCheckNoResourceAttr("data.stackit_ske_cluster.cluster_min", "kube_config"),
				),
			},
			// 3) Import cluster
			{
				ResourceName: "stackit_ske_cluster.cluster",
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
					return fmt.Sprintf("%s,%s", testutil.ProjectId, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// The fields are not provided in the SKE API when disabled, although set actively.
				ImportStateVerifyIgnore: []string{"kubernetes_version_min", "kube_config", "node_pools.0.os_version_min", "extensions.argus.%", "extensions.argus.argus_instance_id", "extensions.argus.enabled", "extensions.acl.enabled", "extensions.acl.allowed_cidrs", "extensions.acl.allowed_cidrs.#", "extensions.acl.%", "extensions.dns.enabled", "extensions.dns.zones", "extensions.dns.zones.#", "extensions.dns.zones.%"},
			},
			// 4) Import minimal cluster
			{
				ResourceName: "stackit_ske_cluster.cluster_min",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_ske_cluster.cluster_min"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_ske_cluster.cluster_min")
					}
					_, ok = r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, name), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"kubernetes_version_min", "kube_config", "node_pools.0.os_version_min"},
			},
			// 5) Update kubernetes version, OS version and maintenance end
			{
				Config: getConfig(clusterResource["kubernetes_version_min_new"], clusterResource["os_version_min_new"], utils.Ptr(clusterResource["maintenance_end_new"])),
				Check: resource.ComposeAggregateTestCheckFunc(
					// cluster data
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "project_id", clusterResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "name", clusterResource["name"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "kubernetes_version_min", clusterResource["kubernetes_version_min_new"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "kubernetes_version_used", clusterResource["kubernetes_version_used_new"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.name", clusterResource["nodepool_name"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.availability_zones.0", clusterResource["nodepool_zone"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_name", clusterResource["nodepool_os_name"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_version_min", clusterResource["nodepool_os_version_min_new"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_version_used", clusterResource["nodepool_os_version_used_new"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.machine_type", clusterResource["nodepool_machine_type"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.minimum", clusterResource["nodepool_minimum"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.maximum", clusterResource["nodepool_maximum"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.max_surge", clusterResource["nodepool_max_surge"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.max_unavailable", clusterResource["nodepool_max_unavailable"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.volume_type", clusterResource["nodepool_volume_type"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.volume_size", clusterResource["nodepool_volume_size"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", fmt.Sprintf("node_pools.0.labels.%s", clusterResource["nodepool_label_key"]), clusterResource["nodepool_label_value"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.effect", clusterResource["nodepool_taints_effect"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.key", clusterResource["nodepool_taints_key"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.taints.0.value", clusterResource["nodepool_taints_value"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.cri", clusterResource["nodepool_cri"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.enabled", clusterResource["extensions_acl_enabled"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.acl.allowed_cidrs.0", clusterResource["extensions_acl_cidrs"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.argus.enabled", clusterResource["extensions_argus_enabled"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.argus.argus_instance_id", clusterResource["extensions_argus_instance_id"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.dns.enabled", clusterResource["extensions_dns_enabled"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.dns.zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "extensions.dns.zones.0", clusterResource["extensions_dns_zones"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.start", clusterResource["hibernations_start"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.end", clusterResource["hibernations_end"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "hibernations.0.timezone", clusterResource["hibernations_timezone"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_kubernetes_version_updates", clusterResource["maintenance_enable_kubernetes_version_updates"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.enable_machine_image_version_updates", clusterResource["maintenance_enable_machine_image_version_updates"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.start", clusterResource["maintenance_start"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "maintenance.end", clusterResource["maintenance_end_new"]),

					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster", "kube_config"), // when using the kubeconfig resource, the kubeconfig field becomes null
				),
			},
			// 6) Downgrade kubernetes and nodepool machine OS version
			{
				Config: getConfig(clusterResource["kubernetes_version_min"], clusterResource["nodepool_os_version_min"], utils.Ptr(clusterResource["maintenance_end_new"])),
				Check: resource.ComposeAggregateTestCheckFunc(
					// cluster data
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "project_id", clusterResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "name", clusterResource["name"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "kubernetes_version_min", clusterResource["kubernetes_version_min"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "kubernetes_version_used", clusterResource["kubernetes_version_used_new"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_version_min", clusterResource["nodepool_os_version_min"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster", "node_pools.0.os_version_used", clusterResource["nodepool_os_version_used_new"]),
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
		client, err = ske.NewAPIClient()
	} else {
		client, err = ske.NewAPIClient(
			config.WithEndpoint(testutil.SKECustomEndpoint),
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
		// cluster terraform ID: = "[project_id],[cluster_name]"
		clusterName := strings.Split(rs.Primary.ID, core.Separator)[1]
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
