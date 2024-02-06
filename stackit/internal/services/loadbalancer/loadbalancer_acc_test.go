package loadbalancer_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
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
	"serverNameIndicator":   "domain.com",
	"network_role":          "ROLE_LISTENERS_AND_TARGETS",
	"private_network_only":  "true",
}

func configResources(targetPort string) string {
	return fmt.Sprintf(`
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
					ip           = openstack_compute_instance_v2.example.network.0.fixed_ip_v4
					}
				]
				session_persistence = {
					use_source_ip_address = %s
				}
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
				serverNameIndictors = [
					{
						"name": "%s"
					}
				]
				target_pool  = "%s"
				}
			]
			networks = [
				{
				network_id = openstack_networking_network_v2.example.id
				role       = "%s"
				}
			]
			options = {
				private_network_only = %s
			}
		}

		resource "stackit_loadbalancer_credential" "credential" {
			project_id   = "%s"
			display_name = "%s"
			username     = "%s"
			password     = "%s"
		}
		`,
		supportingInfraResources(loadBalancerResource["name"], OpenStack{
			userDomainName: testutil.OSUserDomainName,
			userName:       testutil.OSUserName,
			password:       testutil.OSPassword,
		}),
		testutil.LoadBalancerProviderConfig(),
		loadBalancerResource["project_id"],
		loadBalancerResource["name"],
		loadBalancerResource["target_pool_name"],
		targetPort,
		loadBalancerResource["target_display_name"],
		loadBalancerResource["use_source_ip_address"],
		loadBalancerResource["healthy_threshold"],
		loadBalancerResource["interval"],
		loadBalancerResource["interval_jitter"],
		loadBalancerResource["timeout"],
		loadBalancerResource["unhealthy_threshold"],
		loadBalancerResource["listener_display_name"],
		loadBalancerResource["listener_port"],
		loadBalancerResource["listener_protocol"],
		loadBalancerResource["serverNameIndicator"],
		loadBalancerResource["target_pool_name"],
		loadBalancerResource["network_role"],
		loadBalancerResource["private_network_only"],
		loadBalancerResource["project_id"],
		loadBalancerResource["credential_display_name"],
		loadBalancerResource["credential_username"],
		loadBalancerResource["credential_password"],
	)
}

func supportingInfraResources(name string, os OpenStack) string {
	return fmt.Sprintf(`
		provider "openstack" {
			user_domain_name = "%s"
			user_name        = "%s"
			password         = "%s"
			region           = "RegionOne"
			auth_url         = "https://keystone.api.iaas.eu01.stackit.cloud/v3"
		}

		# Create a network
		resource "openstack_networking_network_v2" "example" {
			name = "%s_network"
		}

		resource "openstack_networking_subnet_v2" "example" {
			name            = "%s_subnet"
			cidr            = "192.168.0.0/25"
			dns_nameservers = ["8.8.8.8"]
			network_id      = openstack_networking_network_v2.example.id
		}

		data "openstack_networking_network_v2" "public" {
			name = "floating-net"
		}

		resource "openstack_networking_floatingip_v2" "example_ip" {
			pool = data.openstack_networking_network_v2.public.name
		}

		# Create an instance
		data "openstack_compute_flavor_v2" "example" {
			name = "g1.1"
		}

		resource "openstack_compute_instance_v2" "example" {
			depends_on      = [openstack_networking_subnet_v2.example]
			name            = "%s_instance"
			flavor_id       = data.openstack_compute_flavor_v2.example.id
			admin_pass      = "example"
			security_groups = ["default"]

			block_device {
				uuid                  = "4364cdb2-dacd-429b-803e-f0f7cfde1c24" // Ubuntu 22.04
				volume_size           = 32
				source_type           = "image"
				destination_type      = "volume"
				delete_on_termination = true
			}

			network {
				name = openstack_networking_network_v2.example.name
			}

			lifecycle {
				ignore_changes = [security_groups]
			}
		}

		resource "openstack_networking_router_v2" "example_router" {
			name                = "%s_router"
			admin_state_up      = "true"
			external_network_id = data.openstack_networking_network_v2.public.id
		}

		resource "openstack_networking_router_interface_v2" "example_interface" {
			router_id = openstack_networking_router_v2.example_router.id
			subnet_id = openstack_networking_subnet_v2.example.id
		}

		`,
		os.userDomainName, os.userName, os.password, name, name, name, name)
}

func TestAccLoadBalancerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				VersionConstraint: "= 1.52.1",
				Source:            "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testAccCheckLoadBalancerDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: configResources(loadBalancerResource["target_port"]),
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
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "target_pools.0.session_persistence.use_source_ip_address", loadBalancerResource["use_source_ip_address"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.display_name", loadBalancerResource["listener_display_name"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.port", loadBalancerResource["listener_port"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.protocol", loadBalancerResource["listener_protocol"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.serverNameIndicators.0.name", loadBalancerResource["serverNameIndicator"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", loadBalancerResource["target_pool_name"]),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "networks.0.role", loadBalancerResource["network_role"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer.loadbalancer", "options.private_network_only", loadBalancerResource["private_network_only"]),

					// Credential
					resource.TestCheckResourceAttrPair(
						"stackit_loadbalancer_credential.credential", "project_id",
						"stackit_loadbalancer.loadbalancer", "project_id",
					),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_credential.credential", "credentials_ref"),
					resource.TestCheckResourceAttr("stackit_loadbalancer_credential.credential", "display_name", loadBalancerResource["credential_display_name"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer_credential.credential", "username", loadBalancerResource["credential_username"]),
					resource.TestCheckResourceAttr("stackit_loadbalancer_credential.credential", "password", loadBalancerResource["credential_password"]),
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
					configResources(loadBalancerResource["target_port"]),
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
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "target_pools.0.session_persistence.use_source_ip_address", loadBalancerResource["use_source_ip_address"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.display_name", loadBalancerResource["listener_display_name"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.port", loadBalancerResource["listener_port"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.protocol", loadBalancerResource["listener_protocol"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.serverNameIndicators.0.name", loadBalancerResource["serverNameIndicator"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "listeners.0.target_pool", loadBalancerResource["target_pool_name"]),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer.loadbalancer", "networks.0.network_id"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "networks.0.role", loadBalancerResource["network_role"]),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer.loadbalancer", "options.private_network_only", loadBalancerResource["private_network_only"]),
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

					return fmt.Sprintf("%s,%s", testutil.ProjectId, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: "stackit_loadbalancer_credential.credential",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_loadbalancer_credential.credential"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_loadbalancer_credential.credential")
					}
					credentialsRef, ok := r.Primary.Attributes["credentials_ref"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute credentials_ref")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, credentialsRef), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update
			{
				Config: configResources(loadBalancerResource["target_port_updated"]),
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

type OpenStack struct {
	userDomainName string
	userName       string
	password       string
}

func testAccCheckLoadBalancerDestroy(_ *terraform.State) error {
	ctx := context.Background()
	var client *loadbalancer.APIClient
	var err error
	if testutil.LoadBalancerCustomEndpoint == "" {
		client, err = loadbalancer.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		client, err = loadbalancer.NewAPIClient(
			config.WithEndpoint(testutil.LoadBalancerCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	// Disabling loadbalancer functionality will delete all load balancers
	_, err = client.DisableServiceExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("disabling loadbalancer functionality: %w", err)
	}

	return nil
}
