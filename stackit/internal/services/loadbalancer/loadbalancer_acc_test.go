package loadbalancer_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Instance resource data
var loadBalancerResource = map[string]string{
	"project_id":            testutil.ProjectId,
	"name":                  fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum)),
	"target_pool_name":      "example-target-pool",
	"target_port":           "5432",
	"target_port_updated":   "5431",
	"target_display_name":   "example-target",
	"healthy_threshold":     "3",
	"interval":              "10s",
	"interval_jitter":       "5s",
	"timeout":               "10s",
	"unhealthy_threshold":   "3",
	"use_source_ip_address": "true",
	"listener_display_name": "example-listener",
	"listener_port":         "5432",
	"listener_protocol":     "PROTOCOL_TLS_PASSTHROUGH",
	"network_role":          "ROLE_LISTENERS_AND_TARGETS",
	"private_network_only":  "false",
}

// Network resource data
var networkResource = map[string]string{
	"project_id":  testutil.ProjectId,
	"name":        fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"nameserver0": "8.8.8.8",
	"ipv4_prefix": "192.168.0.0/25",
	"routed":      "true",
}

// Server resource data
var serverResource = map[string]string{
	"project_id":            testutil.ProjectId,
	"availability_zone":     "eu01-1",
	"size":                  "32",
	"source_type":           "image",
	"source_id":             testutil.IaaSImageId,
	"name":                  fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)),
	"machine_type":          "t1.1",
	"user_data":             "#!/bin/bash",
	"delete_on_termination": "true",
}

// Public ip resource data
var publicIpResource = map[string]string{
	"project_id":           testutil.ProjectId,
	"network_interface_id": "stackit_network_interface.network_interface.network_interface_id",
}

func publicIpResourceConfig() string {
	return fmt.Sprintf(`
				resource "stackit_public_ip" "public_ip" {
							project_id = "%s"
							network_interface_id = %s
							lifecycle {
							ignore_changes = [
							network_interface_id
							]
						}
						}
				`,
		publicIpResource["project_id"],
		publicIpResource["network_interface_id"],
	)
}

func networkResourceConfig() string {
	return fmt.Sprintf(`
				resource "stackit_network" "network" {
					project_id = "%s"
					name       = "%s"
					ipv4_nameservers = ["%s"]
					ipv4_prefix = "%s"
					routed = "%s"
				}
				`,
		networkResource["project_id"],
		networkResource["name"],
		networkResource["nameserver0"],
		networkResource["ipv4_prefix"],
		networkResource["routed"],
	)
}

func networkInterfaceResourceConfig() string {
	return `
			resource "stackit_network_interface" "network_interface" {
				project_id = stackit_network.network.project_id
				network_id = stackit_network.network.network_id
				name       = "name"
			}
			`
}

// server config
func serverResourceConfig() string {
	return fmt.Sprintf(`
				resource "stackit_server" "server" {
					project_id = "%s"
					availability_zone = "%s"
					name = "%s"
					machine_type = "%s"
					boot_volume = {
						size = %s
						source_type = "%s"
						source_id = "%s"
						delete_on_termination = "%s"
					}
					network_interfaces = [stackit_network_interface.network_interface.network_interface_id]
					user_data = "%s"
				}
				`,
		serverResource["project_id"],
		serverResource["availability_zone"],
		serverResource["name"],
		serverResource["machine_type"],
		serverResource["size"],
		serverResource["source_type"],
		serverResource["source_id"],
		serverResource["delete_on_termination"],
		serverResource["user_data"],
	)
}

// loadbalancer config
func loadbalancerResourceConfig(targetPort string) string {
	return fmt.Sprintf(`
		%s

		%s

		%s

		%s

		%s

		resource "stackit_loadbalancer" "loadbalancer" {
			project_id = "%s"
			name       = "%s"
			target_pools = [
				{
				name        = "%s"
				target_port = %s
				targets = [
					{
					display_name = "%s"
					ip           = stackit_network_interface.network_interface.ipv4
					}
				]
				active_health_check = {
					healthy_threshold   = %s
					interval            = "%s"
					interval_jitter     = "%s"
					timeout             = "%s"
					unhealthy_threshold = %s
				}
				}
			]
			listeners = [
				{
				  display_name = "%s"
				  port         = %s
				  protocol     = "%s"
				  target_pool  = "%s"
				}
			]
			networks = [
				{
				network_id = stackit_network.network.network_id
				role       = "%s"
				}
			]
			external_address = stackit_public_ip.public_ip.ip
			options = {
				private_network_only = %s
			}
		}
		`,
		testutil.LoadBalancerProviderConfig(),
		networkResourceConfig(),
		networkInterfaceResourceConfig(),
		publicIpResourceConfig(),
		serverResourceConfig(),
		loadBalancerResource["project_id"],
		loadBalancerResource["name"],
		loadBalancerResource["target_pool_name"],
		targetPort,
		loadBalancerResource["target_display_name"],
		loadBalancerResource["healthy_threshold"],
		loadBalancerResource["interval"],
		loadBalancerResource["interval_jitter"],
		loadBalancerResource["timeout"],
		loadBalancerResource["unhealthy_threshold"],
		loadBalancerResource["listener_display_name"],
		loadBalancerResource["listener_port"],
		loadBalancerResource["listener_protocol"],
		loadBalancerResource["target_pool_name"],
		loadBalancerResource["network_role"],
		loadBalancerResource["private_network_only"],
	)
}

func TestAccLoadBalancerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLoadBalancerDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: loadbalancerResourceConfig(loadBalancerResource["target_port"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "project_id", loadBalancerResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "name", loadBalancerResource["name"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.name", loadBalancerResource["target_pool_name"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.target_port", loadBalancerResource["target_port"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.display_name", loadBalancerResource["target_display_name"]),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.healthy_threshold", loadBalancerResource["healthy_threshold"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval", loadBalancerResource["interval"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval_jitter", loadBalancerResource["interval_jitter"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.timeout", loadBalancerResource["timeout"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.unhealthy_threshold", loadBalancerResource["unhealthy_threshold"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.display_name", loadBalancerResource["listener_display_name"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.port", loadBalancerResource["listener_port"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.protocol", loadBalancerResource["listener_protocol"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", loadBalancerResource["target_pool_name"]),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "networks.0.role", loadBalancerResource["network_role"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.private_network_only", loadBalancerResource["private_network_only"]),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_loadbalancer" "loadbalancer" {
						project_id     = stackit_loadbalancer.loadbalancer.project_id
						name    = stackit_loadbalancer.loadbalancer.name
					}
					`,
					loadbalancerResourceConfig(loadBalancerResource["target_port"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "project_id", loadBalancerResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "name", loadBalancerResource["name"]),
					resource.TestCheckResourceAttrPair(
						"data.stackit_loadbalancer.loadbalancer", "project_id",
						"stackit_loadbalancer.loadbalancer", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_loadbalancer.loadbalancer", "name",
						"stackit_loadbalancer.loadbalancer", "name",
					),

					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.name", loadBalancerResource["target_pool_name"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.target_port", loadBalancerResource["target_port"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.display_name", loadBalancerResource["target_display_name"]),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "target_pools.0.targets.0.ip"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.healthy_threshold", loadBalancerResource["healthy_threshold"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval", loadBalancerResource["interval"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.interval_jitter", loadBalancerResource["interval_jitter"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.timeout", loadBalancerResource["timeout"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.active_health_check.unhealthy_threshold", loadBalancerResource["unhealthy_threshold"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.display_name", loadBalancerResource["listener_display_name"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.port", loadBalancerResource["listener_port"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.protocol", loadBalancerResource["listener_protocol"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", loadBalancerResource["target_pool_name"]),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "networks.0.role", loadBalancerResource["network_role"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_loadbalancer.loadbalancer",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_loadbalancer.loadbalancer"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_loadbalancer.loadbalancer")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, name), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"options.private_network_only"},
			},
			// Update
			{
				Config: loadbalancerResourceConfig(loadBalancerResource["target_port_updated"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "project_id", loadBalancerResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "name", loadBalancerResource["name"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.target_port", loadBalancerResource["target_port_updated"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckLoadBalancerDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *loadbalancer.APIClient
	var err error
	if testutil.LoadBalancerCustomEndpoint == "" {
		client, err = loadbalancer.NewAPIClient()
	} else {
		client, err = loadbalancer.NewAPIClient(
			config.WithEndpoint(testutil.LoadBalancerCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	region := "eu01"
	if testutil.Region != "" {
		region = testutil.Region
	}
	loadbalancersToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_loadbalancer" {
			continue
		}
		// loadbalancer terraform ID: = "[project_id],[name]"
		loadbalancerName := strings.Split(rs.Primary.ID, core.Separator)[1]
		loadbalancersToDestroy = append(loadbalancersToDestroy, loadbalancerName)
	}

	loadbalancersResp, err := client.ListLoadBalancers(ctx, testutil.ProjectId, region).Execute()
	if err != nil {
		return fmt.Errorf("getting loadbalancersResp: %w", err)
	}

	if loadbalancersResp.LoadBalancers == nil || (loadbalancersResp.LoadBalancers != nil && len(*loadbalancersResp.LoadBalancers) == 0) {
		fmt.Print("No load balancers found for project \n")
		return nil
	}

	items := *loadbalancersResp.LoadBalancers
	for i := range items {
		if items[i].Name == nil {
			continue
		}
		if utils.Contains(loadbalancersToDestroy, *items[i].Name) {
			_, err := client.DeleteLoadBalancerExecute(ctx, testutil.ProjectId, region, *items[i].Name)
			if err != nil {
				return fmt.Errorf("destroying load balancer %s during CheckDestroy: %w", *items[i].Name, err)
			}
			_, err = wait.DeleteLoadBalancerWaitHandler(ctx, client, testutil.ProjectId, region, *items[i].Name).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying load balancer %s during CheckDestroy: waiting for deletion %w", *items[i].Name, err)
			}
		}
	}
	return nil
}
