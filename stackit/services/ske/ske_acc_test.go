package ske_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/testutil"
)

var projectResource = map[string]string{
	"project_id": testutil.ProjectId,
}

var clusterResource = map[string]string{
	"project_id":                                       testutil.ProjectId,
	"name":                                             fmt.Sprintf("cl-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"name_min":                                         fmt.Sprintf("cl-min-%s", acctest.RandStringFromCharSet(3, acctest.CharSetAlphaNum)),
	"kubernetes_version":                               "1.24",
	"kubernetes_version_used":                          "1.24.17",
	"kubernetes_version_new":                           "1.25",
	"kubernetes_version_used_new":                      "1.25.13",
	"allowPrivilegedContainers":                        "true",
	"nodepool_name":                                    "np-acc-test",
	"nodepool_name_min":                                "np-acc-min-test",
	"nodepool_machine_type":                            "b1.2",
	"nodepool_os_version":                              "3510.2.5",
	"nodepool_os_version_min":                          "3510.2.5",
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
	"hibernations_start":                               "0 16 * * *",
	"hibernations_end":                                 "0 18 * * *",
	"hibernations_timezone":                            "Europe/Berlin",
	"maintenance_enable_kubernetes_version_updates":    "true",
	"maintenance_enable_machine_image_version_updates": "true",
	"maintenance_start":                                "01:23:45Z",
	"maintenance_end":                                  "05:00:00+02:00",
}

func getConfig(version string, apc *bool, maintenanceEnd *string) string {
	apcConfig := ""
	if apc != nil {
		if *apc {
			apcConfig = "allow_privileged_containers = true"
		} else {
			apcConfig = "allow_privileged_containers = false"
		}
	}
	maintenanceEndTF := clusterResource["maintenance_end"]
	if maintenanceEnd != nil {
		maintenanceEndTF = *maintenanceEnd
	}
	aux := fmt.Sprintf(`
		%s

		resource "stackit_ske_project" "project" {
			project_id = "%s"
		}

		resource "stackit_ske_cluster" "cluster" {
			project_id = stackit_ske_project.project.project_id
			name = "%s"
			kubernetes_version = "%s"
			%s
			node_pools = [{
				name = "%s"
				machine_type = "%s"
				minimum = "%s"
				maximum = "%s"
				max_surge = "%s"
				max_unavailable = "%s"
				os_name = "%s"
				os_version = "%s"
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

		resource "stackit_ske_cluster" "cluster_min" {
			project_id = stackit_ske_project.project.project_id
			name = "%s"
			kubernetes_version = "%s"
			node_pools = [{
				name = "%s"
				machine_type = "%s"
				os_version = "%s"
				minimum = "%s"
				maximum = "%s"
				availability_zones = ["%s"]
			}]
		}
		`,
		testutil.SKEProviderConfig(),
		projectResource["project_id"],
		clusterResource["name"],
		version,
		apcConfig,
		clusterResource["nodepool_name"],
		clusterResource["nodepool_machine_type"],
		clusterResource["nodepool_minimum"],
		clusterResource["nodepool_maximum"],
		clusterResource["nodepool_max_surge"],
		clusterResource["nodepool_max_unavailable"],
		clusterResource["nodepool_os_name"],
		clusterResource["nodepool_os_version"],
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
		clusterResource["hibernations_start"],
		clusterResource["hibernations_end"],
		clusterResource["hibernations_timezone"],
		clusterResource["maintenance_enable_kubernetes_version_updates"],
		clusterResource["maintenance_enable_machine_image_version_updates"],
		clusterResource["maintenance_start"],
		maintenanceEndTF,

		// Minimal
		clusterResource["name_min"],
		clusterResource["kubernetes_version_new"],
		clusterResource["nodepool_name_min"],
		clusterResource["nodepool_machine_type"],
		clusterResource["nodepool_os_version_min"],
		clusterResource["nodepool_minimum"],
		clusterResource["nodepool_maximum"],
		clusterResource["nodepool_zone"],
	)
	return aux
}

func getConfig2() string {
	aux := `
		provider "stackit" {
			ske_custom_endpoint = "http://localhost:3333"
			region = "eu01"
		}

		resource "stackit_ske_cluster" "cluster_min" {
			project_id         = "16f49d71-37ad-4137-8b97-44d9c55c4094"
			name               = "hs-min-3"
			kubernetes_version = "1.25"
			node_pools = [{
			name               = "np-acc-min-test"
			machine_type       = "b1.2"
			os_version         = "3510.2.5"
			minimum            = "2"
			maximum            = "3"
			availability_zones = ["eu01-3"]
			}]
		}
		`
	return aux
}

func TestAccSKE(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSKEDestroy,
		Steps: []resource.TestStep{

			// 1) Creation
			{
				Config: getConfig2(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Minimal cluster
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "kubernetes_version", clusterResource["kubernetes_version_new"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "kubernetes_version_used", clusterResource["kubernetes_version_used_new"]),
					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster_min", "allow_privileged_containers"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.name", clusterResource["nodepool_name_min"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.availability_zones.#", "1"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.availability_zones.0", clusterResource["nodepool_zone"]),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "node_pools.0.os_name"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.os_version", clusterResource["nodepool_os_version_min"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.machine_type", clusterResource["nodepool_machine_type"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.minimum", clusterResource["nodepool_minimum"]),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.maximum", clusterResource["nodepool_maximum"]),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "node_pools.0.max_surge"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "node_pools.0.max_unavailable"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "node_pools.0.volume_type"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.labels.%", "0"),
					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.taints"),
					resource.TestCheckResourceAttr("stackit_ske_cluster.cluster_min", "node_pools.0.cri", clusterResource["nodepool_cri"]),
					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster_min", "extensions"),
					resource.TestCheckNoResourceAttr("stackit_ske_cluster.cluster_min", "hibernations"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "maintenance.enable_kubernetes_version_updates"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "maintenance.enable_machine_image_version_updates"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "maintenance.start"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "maintenance.end"),
					resource.TestCheckResourceAttrSet("stackit_ske_cluster.cluster_min", "kube_config"),
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

	projectsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_ske_project" {
			continue
		}
		projectsToDestroy = append(projectsToDestroy, rs.Primary.ID)
	}
	for _, projectId := range projectsToDestroy {
		_, err := client.GetProject(ctx, projectId).Execute()
		if err != nil {
			oapiErr, ok := err.(*ske.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
			if !ok {
				return fmt.Errorf("could not convert error to GenericOpenApiError in acc test destruction, %w", err)
			}
			if oapiErr.StatusCode() == http.StatusNotFound || oapiErr.StatusCode() == http.StatusForbidden {
				// Already gone
				continue
			}
			return fmt.Errorf("getting project: %w", err)
		}

		_, err = client.DeleteProjectExecute(ctx, projectId)
		if err != nil {
			return fmt.Errorf("destroying project %s during CheckDestroy: %w", projectId, err)
		}
		_, err = ske.DeleteProjectWaitHandler(ctx, client, projectId).SetTimeout(15 * time.Minute).WaitWithContext(ctx)
		if err != nil {
			return fmt.Errorf("destroying project %s during CheckDestroy: waiting for deletion %w", projectId, err)
		}
	}
	return nil
}
