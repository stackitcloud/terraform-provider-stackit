package iaas_test

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
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

const (
	serverMachineType        = "t1.1"
	updatedServerMachineType = "t1.2"
)

// Network resource data
var networkResource = map[string]string{
	"project_id":         testutil.ProjectId,
	"name":               fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"ipv4_prefix_length": "24",
	"nameserver0":        "1.2.3.4",
	"nameserver1":        "5.6.7.8",
	"ipv4_gateway":       "10.1.2.1",
	"ipv4_prefix":        "10.1.2.1/24",
	"routed":             "false",
}

var networkAreaResource = map[string]string{
	"organization_id":  testutil.OrganizationId,
	"name":             fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"networkrange0":    "10.0.0.0/16",
	"transfer_network": "10.1.2.0/24",
}

var networkAreaRouteResource = map[string]string{
	"organization_id": networkAreaResource["organization_id"],
	"network_area_id": networkAreaResource["network_area_id"],
	"prefix":          "1.1.1.0/24",
	"next_hop":        "1.1.1.1",
	"label1":          "value1",
	"label1-updated":  "value1-updated",
}

var networkInterfaceResource = map[string]string{
	"project_id": testutil.ProjectId,
	"network_id": networkResource["network_id"],
	"name":       "name",
}

// Volume resource data
var volumeResource = map[string]string{
	"project_id":        testutil.ProjectId,
	"availability_zone": "eu01-1",
	"name":              fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)),
	"description":       "description",
	"size":              "1",
	"label1":            "value",
	"performance_class": "storage_premium_perf1",
}

// Server resource data
var serverResource = map[string]string{
	"project_id":            testutil.ProjectId,
	"availability_zone":     "eu01-1",
	"size":                  "64",
	"source_type":           "image",
	"source_id":             testutil.IaaSImageId,
	"name":                  fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)),
	"machine_type":          serverMachineType,
	"label1":                "value",
	"user_data":             "#!/bin/bash",
	"delete_on_termination": "true",
}

// Security Group resource data
var securityGroupResource = map[string]string{
	"project_id":  testutil.ProjectId,
	"name":        "name",
	"description": "description",
	"label1":      "value",
}

// Security Group rule resource data
var securityGroupRuleResource = map[string]string{
	"project_id":  testutil.ProjectId,
	"direction":   "ingress",
	"description": "description",
}

// Public IP resource data
var publicIpResource = map[string]string{
	"project_id":           testutil.ProjectId,
	"label1":               "value",
	"network_interface_id": "stackit_network_interface.network_interface.network_interface_id",
}

// Key pair resource data
var keyPairResource = map[string]string{
	"name":           "key-pair-name",
	"public_key":     `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIDsPd27M449akqCtdFg2+AmRVJz6eWio0oMP9dVg7XZ`,
	"label1":         "value1",
	"label1-updated": "value1-updated",
}

// Image resource data
var imageResource = map[string]string{
	"project_id":      testutil.ProjectId,
	"name":            fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)),
	"disk_format":     "qcow2",
	"local_file_path": testutil.TestImageLocalFilePath,
	"min_disk_size":   "1",
	"min_ram":         "1",
	"label1":          "value1",
	"boot_menu":       "true",
}

func networkResourceConfig(name, nameservers string) string {
	return fmt.Sprintf(`
				resource "stackit_network" "network" {
					project_id = "%s"
					name       = "%s"
					ipv4_prefix_length = "%s"
					ipv4_nameservers = %s
					ipv4_gateway = "%s"
					ipv4_prefix = "%s"
					routed = "%s"
				}
				`,
		networkResource["project_id"],
		name,
		networkResource["ipv4_prefix_length"],
		nameservers,
		networkResource["ipv4_gateway"],
		networkResource["ipv4_prefix"],
		networkResource["routed"],
	)
}

// routed: true, gateway not present
func networkResourceConfigRouted(name, nameservers string) string {
	return fmt.Sprintf(`
				resource "stackit_network" "network" {
					project_id = "%s"
					name       = "%s"
					ipv4_prefix_length = "%s"
					ipv4_nameservers = %s
					ipv4_prefix = "%s"
					routed = "true"
				}
				`,
		networkResource["project_id"],
		name,
		networkResource["ipv4_prefix_length"],
		nameservers,
		networkResource["ipv4_prefix"],
	)
}

func networkAreaResourceConfig(areaname, networkranges string) string {
	return fmt.Sprintf(`
				resource "stackit_network_area" "network_area" {
					organization_id = "%s"
					name       = "%s"
					network_ranges = [{
						prefix = "%s"
					}]
					transfer_network = "%s"
				}
				`,
		networkAreaResource["organization_id"],
		areaname,
		networkranges,
		networkAreaResource["transfer_network"],
	)
}

func networkAreaRouteResourceConfig(labelValue string) string {
	return fmt.Sprintf(`
				resource "stackit_network_area_route" "network_area_route" {
					organization_id = stackit_network_area.network_area.organization_id
					network_area_id = stackit_network_area.network_area.network_area_id
					prefix          = "%s"
					next_hop        = "%s"
					labels = {
						"label1" = "%s"
					}
				}
				`,
		networkAreaRouteResource["prefix"],
		networkAreaRouteResource["next_hop"],
		labelValue,
	)
}

func networkInterfaceResourceConfig(name string) string {
	return fmt.Sprintf(`
				resource "stackit_network_interface" "network_interface" {
					project_id = stackit_network.network.project_id
					network_id = stackit_network.network.network_id
					name       = "%s"
				}
				`,
		name,
	)
}

func volumeResourceConfig(name, size string) string {
	return fmt.Sprintf(`
				resource "stackit_volume" "volume" {
					project_id = "%s"
					availability_zone = "%s"
					name = "%s"
					description = "%s"
					size = %s
					labels = {
						"label1" = "%s"
					}
					performance_class = "%s"
				}
				`,
		volumeResource["project_id"],
		volumeResource["availability_zone"],
		name,
		volumeResource["description"],
		size,
		volumeResource["label1"],
		volumeResource["performance_class"],
	)
}

func serverResourceConfig(name, machineType string) string {
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
					labels = {
						"label1" = "%s"
					}
					user_data = "%s"
				}
				`,
		serverResource["project_id"],
		serverResource["availability_zone"],
		name,
		machineType,
		serverResource["size"],
		serverResource["source_type"],
		serverResource["source_id"],
		serverResource["delete_on_termination"],
		serverResource["label1"],
		serverResource["user_data"],
	)
}

func securityGroupResourceConfig(name string) string {
	return fmt.Sprintf(`
				resource "stackit_security_group" "security_group" {
					project_id = "%s"
					name = "%s"
					description = "%s"
					labels = {
						"label1" = "%s"
					}
				}
				`,
		securityGroupResource["project_id"],
		name,
		securityGroupResource["description"],
		securityGroupResource["label1"],
	)
}

func securityGroupRuleResourceConfig(direction string) string {
	return fmt.Sprintf(`
				resource "stackit_security_group_rule" "security_group_rule" {
					project_id 		  = "%s"
					security_group_id = stackit_security_group.security_group.security_group_id
					direction 		  = "%s"
					description 	  = "%s"
				}
				`,
		securityGroupRuleResource["project_id"],
		direction,
		securityGroupRuleResource["description"],
	)
}

func volumeAttachmentResourceConfig() string {
	return fmt.Sprintf(`
				resource "stackit_server_volume_attach" "attach_volume" {
					project_id 		  = "%s"
					server_id = stackit_server.server.server_id
					volume_id = stackit_volume.volume.volume_id
				}
				`,
		testutil.ProjectId,
	)
}

func serviceAccountAttachmentResourceConfig() string {
	return fmt.Sprintf(`
				resource "stackit_server_service_account_attach" "attach_sa" {
					project_id 		  = "%s"
					server_id = stackit_server.server.server_id
					service_account_email = "%s"
				}
				`,
		testutil.ProjectId,
		testutil.TestProjectServiceAccountEmail,
	)
}

func imageResourceConfig(name string) string {
	return fmt.Sprintf(`
				resource "stackit_image" "image" {
					project_id = "%s"
					name = "%s"
					disk_format = "%s"
					local_file_path = "%s"
					min_disk_size = %s
					min_ram = %s
					labels = {
						"label1" = "%s"
					}
					config = {
						boot_menu = %s
					}
				}
				`,
		imageResource["project_id"],
		name,
		imageResource["disk_format"],
		imageResource["local_file_path"],
		imageResource["min_disk_size"],
		imageResource["min_ram"],
		imageResource["label1"],
		imageResource["boot_menu"],
	)
}

func testAccNetworkAreaConfig(areaname, networkranges, routeLabelValue string) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s",
		testutil.IaaSProviderConfig(),
		networkAreaResourceConfig(areaname, networkranges),
		networkAreaRouteResourceConfig(routeLabelValue),
	)
}

func testAccVolumeConfig(name, size string) string {
	return fmt.Sprintf("%s\n\n%s",
		testutil.IaaSProviderConfig(),
		volumeResourceConfig(name, size),
	)
}

func testAccServerConfig(name, nameservers, serverName, machineType, interfacename string) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s",
		testutil.IaaSProviderConfig(),
		networkResourceConfig(name, nameservers),
		serverResourceConfig(serverName, machineType),
		volumeResourceConfig(volumeResource["name"], volumeResource["size"]),
		networkInterfaceResourceConfig(interfacename),
		volumeAttachmentResourceConfig(),
		serviceAccountAttachmentResourceConfig(),
	)
}

func resourceConfigSecurityGroup(name, direction string) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s",
		testutil.IaaSProviderConfig(),
		securityGroupResourceConfig(name),
		securityGroupRuleResourceConfig(direction),
	)
}

func testAccPublicIpConfig(nameNetwork, nameservers, nameNetworkInterface, publicIpResourceConfig string) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		testutil.IaaSProviderConfig(),
		networkResourceConfigRouted(nameNetwork, nameservers),
		networkInterfaceResourceConfig(nameNetworkInterface),
		publicIpResourceConfig,
	)
}

func testAccKeyPairConfig(keyPairResourceConfig string) string {
	return fmt.Sprintf("%s\n\n%s",
		testutil.IaaSProviderConfig(),
		keyPairResourceConfig,
	)
}

func testAccImageConfig(name string) string {
	return fmt.Sprintf("%s\n\n%s",
		testutil.IaaSProviderConfig(),
		imageResourceConfig(name),
	)
}

func TestAccNetworkArea(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNetworkAreaDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: testAccNetworkAreaConfig(
					networkAreaResource["name"],
					networkAreaResource["networkrange0"],
					networkAreaRouteResource["label1"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network Area
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "organization_id", networkAreaResource["organization_id"]),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_area_id"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "name", networkAreaResource["name"]),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.0.prefix", networkAreaResource["networkrange0"]),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_ranges.0.network_range_id"),

					// Network Area Route
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "organization_id",
						"stackit_network_area.network_area", "organization_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "network_area_id",
						"stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_area_route.network_area_route", "network_area_route_id"),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "prefix", networkAreaRouteResource["prefix"]),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "next_hop", networkAreaRouteResource["next_hop"]),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "labels.label1", networkAreaRouteResource["label1"]),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s
						
					data "stackit_network_area" "network_area" {
						organization_id  = stackit_network_area.network_area.organization_id
						network_area_id  = stackit_network_area.network_area.network_area_id
					}
					
					data "stackit_network_area_route" "network_area_route" {
						organization_id  	  = stackit_network_area.network_area.organization_id
						network_area_id  	  = stackit_network_area.network_area.network_area_id
						network_area_route_id = stackit_network_area_route.network_area_route.network_area_route_id
					}
					`,
					testAccNetworkAreaConfig(
						networkAreaResource["name"],
						networkAreaResource["networkrange0"],
						networkAreaRouteResource["label1"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(

					// Network area
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "organization_id", networkAreaResource["organization_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_network_area.network_area", "network_area_id",
						"data.stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "name", networkAreaResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "network_ranges.0.prefix", networkAreaResource["networkrange0"]),

					// Network area route
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "organization_id",
						"stackit_network_area.network_area", "organization_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "network_area_id",
						"stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_area_route.network_area_route", "network_area_route_id"),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "prefix", networkAreaRouteResource["prefix"]),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "next_hop", networkAreaRouteResource["next_hop"]),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "labels.label1", networkAreaRouteResource["label1"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_network_area.network_area",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_area.network_area"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_area.network_area")
					}
					networkAreaId, ok := r.Primary.Attributes["network_area_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_area_id")
					}
					return fmt.Sprintf("%s,%s", testutil.OrganizationId, networkAreaId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: "stackit_network_area_route.network_area_route",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_area_route.network_area_route"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_area_route.network_area_route")
					}
					networkAreaId, ok := r.Primary.Attributes["network_area_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_area_id")
					}
					networkAreaRouteId, ok := r.Primary.Attributes["network_area_route_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_area_route_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.OrganizationId, networkAreaId, networkAreaRouteId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccNetworkAreaConfig(
					fmt.Sprintf("%s-updated", networkAreaResource["name"]),
					networkAreaResource["networkrange0"],
					networkAreaRouteResource["label1-updated"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network area
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "organization_id", networkAreaResource["organization_id"]),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_area_id"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "name", fmt.Sprintf("%s-updated", networkAreaResource["name"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.0.prefix", networkAreaResource["networkrange0"]),

					// Network area route
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "organization_id",
						"stackit_network_area.network_area", "organization_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "network_area_id",
						"stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_area_route.network_area_route", "network_area_route_id"),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "prefix", networkAreaRouteResource["prefix"]),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "next_hop", networkAreaRouteResource["next_hop"]),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "labels.label1", networkAreaRouteResource["label1-updated"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccVolume(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIaaSVolumeDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: testAccVolumeConfig(volumeResource["name"], volumeResource["size"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_volume.volume", "project_id", volumeResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_volume.volume", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume", "name", volumeResource["name"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "availability_zone", volumeResource["availability_zone"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "labels.label1", volumeResource["label1"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "description", volumeResource["description"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "performance_class", volumeResource["performance_class"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "size", volumeResource["size"]),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s
			
					data "stackit_volume" "volume" {
						project_id  = stackit_volume.volume.project_id
						volume_id = stackit_volume.volume.volume_id
					}
					`,
					testAccVolumeConfig(volumeResource["name"], volumeResource["size"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_volume.volume", "project_id", networkResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.volume", "volume_id",
						"data.stackit_volume.volume", "volume_id",
					),
					resource.TestCheckResourceAttr("data.stackit_volume.volume", "name", volumeResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_volume.volume", "availability_zone", volumeResource["availability_zone"]),
					resource.TestCheckResourceAttr("data.stackit_volume.volume", "availability_zone", volumeResource["availability_zone"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "labels.label1", volumeResource["label1"]),
					resource.TestCheckResourceAttr("data.stackit_volume.volume", "description", volumeResource["description"]),
					resource.TestCheckResourceAttr("data.stackit_volume.volume", "performance_class", volumeResource["performance_class"]),
					resource.TestCheckResourceAttr("data.stackit_volume.volume", "size", volumeResource["size"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_volume.volume",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_volume.volume"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_volume.volume")
					}
					volumeId, ok := r.Primary.Attributes["volume_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, volumeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccVolumeConfig(
					fmt.Sprintf("%s-updated", volumeResource["name"]),
					"10",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_volume.volume", "project_id", volumeResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_volume.volume", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume", "name", fmt.Sprintf("%s-updated", volumeResource["name"])),
					resource.TestCheckResourceAttr("stackit_volume.volume", "availability_zone", volumeResource["availability_zone"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "labels.label1", volumeResource["label1"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "description", volumeResource["description"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "performance_class", volumeResource["performance_class"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "size", "10"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: testAccServerConfig(
					networkResource["name"],
					fmt.Sprintf(
						"[%q]",
						networkResource["nameserver0"],
					),
					serverResource["name"],
					serverResource["machine_type"],
					networkInterfaceResource["name"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(

					// Network
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", networkResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "name", networkResource["name"]),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.#", "1"),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.0", networkResource["nameserver0"]),
					resource.TestCheckResourceAttr("stackit_network.network", "ipv4_gateway", networkResource["ipv4_gateway"]),
					resource.TestCheckResourceAttr("stackit_network.network", "ipv4_prefix", networkResource["ipv4_prefix"]),
					resource.TestCheckResourceAttr("stackit_network.network", "routed", networkResource["routed"]),

					// Server
					resource.TestCheckResourceAttr("stackit_server.server", "project_id", serverResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_server.server", "server_id"),
					resource.TestCheckResourceAttr("stackit_server.server", "name", serverResource["name"]),
					resource.TestCheckResourceAttr("stackit_server.server", "availability_zone", serverResource["availability_zone"]),
					resource.TestCheckResourceAttr("stackit_server.server", "machine_type", serverResource["machine_type"]),
					resource.TestCheckResourceAttr("stackit_server.server", "labels.label1", serverResource["label1"]),
					resource.TestCheckResourceAttr("stackit_server.server", "user_data", serverResource["user_data"]),
					resource.TestCheckResourceAttrSet("stackit_server.server", "network_interfaces.0"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.size", serverResource["size"]),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_type", serverResource["source_type"]),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_id", serverResource["source_id"]),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.delete_on_termination", serverResource["delete_on_termination"]),

					// Network Interface
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface", "project_id",
						"stackit_network.network", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "network_interface_id"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "name", networkInterfaceResource["name"]),

					// Volume attachment
					resource.TestCheckResourceAttrPair(
						"stackit_server_volume_attach.attach_volume", "project_id",
						"stackit_server.server", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_server_volume_attach.attach_volume", "server_id",
						"stackit_server.server", "server_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_server_volume_attach.attach_volume", "volume_id",
						"stackit_volume.volume", "volume_id",
					),

					// Service account attachment
					resource.TestCheckResourceAttrPair(
						"stackit_server_service_account_attach.attach_sa", "project_id",
						"stackit_server.server", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_server_service_account_attach.attach_sa", "server_id",
						"stackit_server.server", "server_id",
					),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
						%s

						data "stackit_network" "network" {
							project_id  = stackit_network.network.project_id
							network_id = stackit_network.network.network_id
						}

						data "stackit_server" "server" {
							project_id  = stackit_server.server.project_id
							server_id = stackit_server.server.server_id
						}

						data "stackit_network_interface" "network_interface" {
							project_id  	     = stackit_network.network.project_id
							network_id  	     = stackit_network.network.network_id
							network_interface_id = stackit_network_interface.network_interface.network_interface_id
						}
						`,
					testAccServerConfig(
						networkResource["name"],
						fmt.Sprintf(
							"[%q]",
							networkResource["nameserver0"],
						),
						serverResource["name"],
						serverResource["machine_type"],
						networkInterfaceResource["name"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_network.network", "project_id", networkResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_network.network", "network_id",
						"data.stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttr("data.stackit_network.network", "name", networkResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_network.network", "nameservers.0", networkResource["nameserver0"]),
					resource.TestCheckResourceAttr("data.stackit_network.network", "ipv4_gateway", networkResource["ipv4_gateway"]),
					resource.TestCheckResourceAttr("data.stackit_network.network", "routed", networkResource["routed"]),

					// Server
					resource.TestCheckResourceAttr("data.stackit_server.server", "project_id", serverResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "server_id",
						"data.stackit_server.server", "server_id",
					),
					resource.TestCheckResourceAttr("data.stackit_server.server", "name", serverResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_server.server", "availability_zone", serverResource["availability_zone"]),
					resource.TestCheckResourceAttr("data.stackit_server.server", "machine_type", serverResource["machine_type"]),
					resource.TestCheckResourceAttr("data.stackit_server.server", "labels.label1", serverResource["label1"]),

					// Network Interface
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface", "project_id",
						"stackit_network.network", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "network_interface_id"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "name", networkInterfaceResource["name"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_network.network",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network.network"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network.network")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, networkId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ipv4_prefix_length", "ipv4_prefix"}, // Field is not returned by the API
			},
			{
				ResourceName: "stackit_server.server",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server.server"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server.server")
					}
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute server_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, serverId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"boot_volume", "user_data"}, // Field is not mapped as it is only relevant on creation
			},
			{
				ResourceName: "stackit_network_interface.network_interface",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_interface.network_interface"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_interface.network_interface")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					networkInterfaceId, ok := r.Primary.Attributes["network_interface_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_interface_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, networkId, networkInterfaceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: "stackit_server_volume_attach.attach_volume",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server_volume_attach.attach_volume"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server_volume_attach.attach_volume")
					}
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute server_id")
					}
					volumeId, ok := r.Primary.Attributes["volume_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, serverId, volumeId), nil
				},
				ImportState:       true,
				ImportStateVerify: false,
			},
			{
				ResourceName: "stackit_server_service_account_attach.attach_sa",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server_service_account_attach.attach_sa"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server_service_account_attach.attach_sa")
					}
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute server_id")
					}
					serviceAccountEmail, ok := r.Primary.Attributes["service_account_email"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, serverId, serviceAccountEmail), nil
				},
				ImportState:       true,
				ImportStateVerify: false,
			},
			// Update
			{
				Config: testAccServerConfig(
					fmt.Sprintf("%s-updated", networkResource["name"]),
					fmt.Sprintf(
						"[%q, %q]",
						networkResource["nameserver0"],
						networkResource["nameserver1"],
					),
					fmt.Sprintf("%s-updated", serverResource["name"]),
					updatedServerMachineType,
					fmt.Sprintf("%s-updated", networkInterfaceResource["name"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", networkResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "name", fmt.Sprintf("%s-updated", networkResource["name"])),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.#", "2"),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.0", networkResource["nameserver0"]),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.1", networkResource["nameserver1"]),
					resource.TestCheckResourceAttr("stackit_network.network", "ipv4_gateway", networkResource["ipv4_gateway"]),

					// Server
					resource.TestCheckResourceAttr("stackit_server.server", "project_id", serverResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_server.server", "server_id"),
					resource.TestCheckResourceAttr("stackit_server.server", "name", fmt.Sprintf("%s-updated", serverResource["name"])),
					resource.TestCheckResourceAttr("stackit_server.server", "availability_zone", serverResource["availability_zone"]),
					resource.TestCheckResourceAttr("stackit_server.server", "machine_type", updatedServerMachineType),
					resource.TestCheckResourceAttr("stackit_server.server", "labels.label1", serverResource["label1"]),
					resource.TestCheckResourceAttr("stackit_server.server", "user_data", serverResource["user_data"]),
					resource.TestCheckResourceAttrSet("stackit_server.server", "network_interfaces.0"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.size", serverResource["size"]),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_type", serverResource["source_type"]),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_id", serverResource["source_id"]),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.delete_on_termination", serverResource["delete_on_termination"]),

					// Network interface
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface", "project_id",
						"stackit_network.network", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "network_interface_id"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "name", fmt.Sprintf("%s-updated", networkInterfaceResource["name"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccIaaSSecurityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIaaSSecurityGroupDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfigSecurityGroup(
					securityGroupResource["name"],
					securityGroupRuleResource["direction"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Security Group
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "project_id", securityGroupResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group", "security_group_id"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "name", securityGroupResource["name"]),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "labels.label1", securityGroupResource["label1"]),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "description", securityGroupResource["description"]),

					// Security Group Rule
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "direction", securityGroupRuleResource["direction"]),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "description", securityGroupRuleResource["description"]),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s
			
					data "stackit_security_group" "security_group" {
						project_id  = stackit_security_group.security_group.project_id
						security_group_id = stackit_security_group.security_group.security_group_id
					}

					data "stackit_security_group_rule" "security_group_rule" {
						project_id             = stackit_security_group.security_group.project_id
						security_group_id      = stackit_security_group.security_group.security_group_id
						security_group_rule_id = stackit_security_group_rule.security_group_rule.security_group_rule_id
					}
					`,
					resourceConfigSecurityGroup(
						securityGroupResource["name"],
						securityGroupRuleResource["direction"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "project_id", securityGroupResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group.security_group", "security_group_id",
						"data.stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "name", securityGroupResource["name"]),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "labels.label1", securityGroupResource["label1"]),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "description", securityGroupResource["description"]),

					// Security Group Rule
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "direction", securityGroupRuleResource["direction"]),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "description", securityGroupRuleResource["description"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_security_group.security_group",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_security_group.security_group"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_security_group.security_group")
					}
					securityGroupId, ok := r.Primary.Attributes["security_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute security_group_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, securityGroupId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: "stackit_security_group_rule.security_group_rule",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_security_group_rule.security_group_rule"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_security_group_rule.security_group_rule")
					}
					securityGroupId, ok := r.Primary.Attributes["security_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute security_group_id")
					}
					securityGroupRuleId, ok := r.Primary.Attributes["security_group_rule_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute security_group_rule_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, securityGroupId, securityGroupRuleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: resourceConfigSecurityGroup(
					fmt.Sprintf("%s-updated", securityGroupResource["name"]),
					securityGroupRuleResource["direction"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "project_id", securityGroupResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group", "security_group_id"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "name", fmt.Sprintf("%s-updated", securityGroupResource["name"])),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "labels.label1", securityGroupResource["label1"]),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "description", securityGroupResource["description"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccPublicIp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIaaSPublicIpDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: testAccPublicIpConfig(
					networkResource["name"],
					fmt.Sprintf(
						"[%q]",
						networkResource["nameserver0"],
					),
					networkInterfaceResource["name"],
					fmt.Sprintf(`
						resource "stackit_public_ip" "public_ip" {
							project_id = "%s"
							labels = {
								"label1" = "%s"
							}
							network_interface_id = %s
						}
					`,
						publicIpResource["project_id"],
						publicIpResource["label1"],
						publicIpResource["network_interface_id"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "project_id", publicIpResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip", "public_ip_id"),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "labels.label1", publicIpResource["label1"]),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "network_interface_id"),
				),
			},

			// Data source
			{
				Config: fmt.Sprintf(`
						%s

						data "stackit_public_ip" "public_ip" {
							project_id   		 = stackit_public_ip.public_ip.project_id
							public_ip_id 		 = stackit_public_ip.public_ip.public_ip_id
						}
						`,
					testAccPublicIpConfig(
						networkResource["name"],
						fmt.Sprintf(
							"[%q]",
							networkResource["nameserver0"],
						),
						networkInterfaceResource["name"],
						fmt.Sprintf(`
								resource "stackit_public_ip" "public_ip" {
									project_id = "%s"
									labels = {
										"label1" = "%s"
									}
									network_interface_id = %s
								}
							`,
							publicIpResource["project_id"],
							publicIpResource["label1"],
							publicIpResource["network_interface_id"],
						),
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_public_ip.public_ip", "project_id", publicIpResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip.public_ip", "public_ip_id",
						"data.stackit_public_ip.public_ip", "public_ip_id",
					),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "labels.label1", publicIpResource["label1"]),
					resource.TestCheckResourceAttrSet("data.stackit_public_ip.public_ip", "network_interface_id"),
				),
			},
			// Import
			{
				ResourceName: "stackit_public_ip.public_ip",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_public_ip.public_ip"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_public_ip.public_ip")
					}
					publicIpId, ok := r.Primary.Attributes["public_ip_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute public_ip_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, publicIpId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccPublicIpConfig(
					networkResource["name"],
					fmt.Sprintf(
						"[%q]",
						networkResource["nameserver0"],
					),
					networkInterfaceResource["name"],
					fmt.Sprintf(`
								resource "stackit_public_ip" "public_ip" {
									project_id = "%s"
									labels = {
										"label1" = "%s"
									}
								}
							`,
						publicIpResource["project_id"],
						publicIpResource["label1"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "project_id", publicIpResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip", "public_ip_id"),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "labels.label1", publicIpResource["label1"]),
					resource.TestCheckNoResourceAttr("stackit_public_ip.public_ip", "network_interface_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccKeyPair(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIaaSKeyPairDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: testAccKeyPairConfig(
					fmt.Sprintf(`
						resource "stackit_key_pair" "key_pair" {
							name = "%s"
							public_key = "%s"
							labels = {
								"label1" = "%s"
							}
						}
					`,
						keyPairResource["name"],
						keyPairResource["public_key"],
						keyPairResource["label1"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "name", keyPairResource["name"]),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "labels.label1", keyPairResource["label1"]),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "public_key", keyPairResource["public_key"]),
					resource.TestCheckResourceAttrSet("stackit_key_pair.key_pair", "fingerprint"),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_key_pair" "key_pair" {
						name = stackit_key_pair.key_pair.name
					}
					`,
					testAccKeyPairConfig(
						fmt.Sprintf(`
							resource "stackit_key_pair" "key_pair" {
								name = "%s"
								public_key = "%s"
								labels = {
									"label1" = "%s"
								}
						}
						`,
							keyPairResource["name"],
							keyPairResource["public_key"],
							keyPairResource["label1"],
						),
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_key_pair.key_pair", "name", keyPairResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_key_pair.key_pair", "public_key", keyPairResource["public_key"]),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "labels.label1", keyPairResource["label1"]),
					resource.TestCheckResourceAttrPair(
						"stackit_key_pair.key_pair", "fingerprint",
						"data.stackit_key_pair.key_pair", "fingerprint",
					),
				),
			},
			// Import
			{
				ResourceName: "stackit_key_pair.key_pair",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_key_pair.key_pair"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_key_pair.key_pair")
					}
					keyPairName, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return keyPairName, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccKeyPairConfig(
					fmt.Sprintf(`
							resource "stackit_key_pair" "key_pair" {
								name = "%s"
								public_key = "%s"
								labels = {
									"label1" = "%s"
								}
						}
						`,
						keyPairResource["name"],
						keyPairResource["public_key"],
						keyPairResource["label1-updated"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "name", keyPairResource["name"]),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "labels.label1", keyPairResource["label1-updated"]),
					resource.TestCheckResourceAttrSet("stackit_key_pair.key_pair", "fingerprint"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIaaSImageDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: testAccImageConfig(imageResource["name"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_image.image", "project_id", imageResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_image.image", "image_id"),
					resource.TestCheckResourceAttr("stackit_image.image", "name", imageResource["name"]),
					resource.TestCheckResourceAttr("stackit_image.image", "disk_format", imageResource["disk_format"]),
					resource.TestCheckResourceAttr("stackit_image.image", "min_disk_size", imageResource["min_disk_size"]),
					resource.TestCheckResourceAttr("stackit_image.image", "min_ram", imageResource["min_ram"]),
					resource.TestCheckResourceAttrSet("stackit_image.image", "local_file_path"),
					resource.TestCheckResourceAttr("stackit_image.image", "local_file_path", imageResource["local_file_path"]),
					resource.TestCheckResourceAttr("stackit_image.image", "labels.label1", imageResource["label1"]),
					resource.TestCheckResourceAttr("stackit_image.image", "config.boot_menu", imageResource["boot_menu"]),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.digest"),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_image" "image" {
						project_id = stackit_image.image.project_id
						image_id = stackit_image.image.image_id
					}
					`,
					testAccImageConfig(imageResource["name"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_image.image", "project_id", imageResource["project_id"]),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "image_id", "stackit_image.image", "image_id"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "name", "stackit_image.image", "name"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "disk_format", "stackit_image.image", "disk_format"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "min_disk_size", "stackit_image.image", "min_disk_size"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "min_ram", "stackit_image.image", "min_ram"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "protected", "stackit_image.image", "protected"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "labels.label1", "stackit_image.image", "labels.label1"),
				),
			},
			// Import
			{
				ResourceName: "stackit_image.image",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_image.image"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_image.image")
					}
					imageId, ok := r.Primary.Attributes["image_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute image_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, imageId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"local_file_path"},
			},
			// Update
			{
				Config: testAccImageConfig(fmt.Sprintf("%s-updated", imageResource["name"])),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_image.image", "name", fmt.Sprintf("%s-updated", imageResource["name"])),
					resource.TestCheckResourceAttr("stackit_image.image", "project_id", imageResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_image.image", "labels.label1", imageResource["label1"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckNetworkAreaDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			config.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	// network areas
	networkAreasToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_network_area" {
			continue
		}
		networkAreaId := strings.Split(rs.Primary.ID, core.Separator)[1]
		networkAreasToDestroy = append(networkAreasToDestroy, networkAreaId)
	}

	networkAreasResp, err := client.ListNetworkAreasExecute(ctx, testutil.OrganizationId)
	if err != nil {
		return fmt.Errorf("getting networkAreasResp: %w", err)
	}

	networkAreas := *networkAreasResp.Items
	for i := range networkAreas {
		if networkAreas[i].AreaId == nil {
			continue
		}
		if utils.Contains(networkAreasToDestroy, *networkAreas[i].AreaId) {
			err := client.DeleteNetworkAreaExecute(ctx, testutil.OrganizationId, *networkAreas[i].AreaId)
			if err != nil {
				return fmt.Errorf("destroying network area %s during CheckDestroy: %w", *networkAreas[i].AreaId, err)
			}
		}
	}
	return nil
}

func testAccCheckIaaSVolumeDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			config.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	volumesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_volume" {
			continue
		}
		// volume terraform ID: "[project_id],[volume_id]"
		volumeId := strings.Split(rs.Primary.ID, core.Separator)[1]
		volumesToDestroy = append(volumesToDestroy, volumeId)
	}

	volumesResp, err := client.ListVolumesExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting volumesResp: %w", err)
	}

	volumes := *volumesResp.Items
	for i := range volumes {
		if volumes[i].Id == nil {
			continue
		}
		if utils.Contains(volumesToDestroy, *volumes[i].Id) {
			err := client.DeleteVolumeExecute(ctx, testutil.ProjectId, *volumes[i].Id)
			if err != nil {
				return fmt.Errorf("destroying volume %s during CheckDestroy: %w", *volumes[i].Id, err)
			}
		}
	}
	return nil
}

func testAccCheckServerDestroy(s *terraform.State) error {
	ctx := context.Background()
	var alphaClient *iaas.APIClient
	var client *iaas.APIClient
	var err error
	var alphaErr error
	if testutil.IaaSCustomEndpoint == "" {
		alphaClient, alphaErr = iaas.NewAPIClient(
			config.WithRegion("eu01"),
		)
		client, err = iaas.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		alphaClient, alphaErr = iaas.NewAPIClient(
			config.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
		client, err = iaas.NewAPIClient(
			config.WithRegion("eu01"),
		)
	}
	if err != nil || alphaErr != nil {
		return fmt.Errorf("creating client: %w, %w", err, alphaErr)
	}

	// Servers

	serversToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_server" {
			continue
		}
		// server terraform ID: "[project_id],[server_id]"
		serverId := strings.Split(rs.Primary.ID, core.Separator)[1]
		serversToDestroy = append(serversToDestroy, serverId)
	}

	serversResp, err := alphaClient.ListServersExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting serversResp: %w", err)
	}

	servers := *serversResp.Items
	for i := range servers {
		if servers[i].Id == nil {
			continue
		}
		if utils.Contains(serversToDestroy, *servers[i].Id) {
			err := alphaClient.DeleteServerExecute(ctx, testutil.ProjectId, *servers[i].Id)
			if err != nil {
				return fmt.Errorf("destroying server %s during CheckDestroy: %w", *servers[i].Id, err)
			}
		}
	}

	// Networks

	networksToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_network" {
			continue
		}
		// network terraform ID: "[project_id],[network_id]"
		networkId := strings.Split(rs.Primary.ID, core.Separator)[1]
		networksToDestroy = append(networksToDestroy, networkId)
	}

	networksResp, err := client.ListNetworksExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting networksResp: %w", err)
	}

	networks := *networksResp.Items
	for i := range networks {
		if networks[i].NetworkId == nil {
			continue
		}
		if utils.Contains(networksToDestroy, *networks[i].NetworkId) {
			err := client.DeleteNetworkExecute(ctx, testutil.ProjectId, *networks[i].NetworkId)
			if err != nil {
				return fmt.Errorf("destroying network %s during CheckDestroy: %w", *networks[i].NetworkId, err)
			}
		}
	}

	return nil
}

func testAccCheckIaaSSecurityGroupDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			config.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	securityGroupsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_security_group" {
			continue
		}
		// security group terraform ID: "[project_id],[security_group_id]"
		securityGroupId := strings.Split(rs.Primary.ID, core.Separator)[1]
		securityGroupsToDestroy = append(securityGroupsToDestroy, securityGroupId)
	}

	securityGroupsResp, err := client.ListSecurityGroupsExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting securityGroupsResp: %w", err)
	}

	securityGroups := *securityGroupsResp.Items
	for i := range securityGroups {
		if securityGroups[i].Id == nil {
			continue
		}
		if utils.Contains(securityGroupsToDestroy, *securityGroups[i].Id) {
			err := client.DeleteSecurityGroupExecute(ctx, testutil.ProjectId, *securityGroups[i].Id)
			if err != nil {
				return fmt.Errorf("destroying security group %s during CheckDestroy: %w", *securityGroups[i].Id, err)
			}
		}
	}
	return nil
}

func testAccCheckIaaSPublicIpDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			config.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	publicIpsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_public_ip" {
			continue
		}
		// public IP terraform ID: "[project_id],[public_ip_id]"
		publicIpId := strings.Split(rs.Primary.ID, core.Separator)[1]
		publicIpsToDestroy = append(publicIpsToDestroy, publicIpId)
	}

	publicIpsResp, err := client.ListPublicIPsExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting publicIpsResp: %w", err)
	}

	publicIps := *publicIpsResp.Items
	for i := range publicIps {
		if publicIps[i].Id == nil {
			continue
		}
		if utils.Contains(publicIpsToDestroy, *publicIps[i].Id) {
			err := client.DeletePublicIPExecute(ctx, testutil.ProjectId, *publicIps[i].Id)
			if err != nil {
				return fmt.Errorf("destroying public IP %s during CheckDestroy: %w", *publicIps[i].Id, err)
			}
		}
	}
	return nil
}

func testAccCheckIaaSKeyPairDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			config.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	keyPairsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_key_pair" {
			continue
		}
		// Key pair terraform ID: "[name]"
		keyPairsToDestroy = append(keyPairsToDestroy, rs.Primary.ID)
	}

	keyPairsResp, err := client.ListKeyPairsExecute(ctx)
	if err != nil {
		return fmt.Errorf("getting key pairs: %w", err)
	}

	keyPairs := *keyPairsResp.Items
	for i := range keyPairs {
		if keyPairs[i].Name == nil {
			continue
		}
		if utils.Contains(keyPairsToDestroy, *keyPairs[i].Name) {
			err := client.DeleteKeyPairExecute(ctx, *keyPairs[i].Name)
			if err != nil {
				return fmt.Errorf("destroying key pair %s during CheckDestroy: %w", *keyPairs[i].Name, err)
			}
		}
	}
	return nil
}

func testAccCheckIaaSImageDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			config.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	imagesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_image" {
			continue
		}
		// Image terraform ID: "[project_id],[image_id]"
		imageId := strings.Split(rs.Primary.ID, core.Separator)[1]
		imagesToDestroy = append(imagesToDestroy, imageId)
	}

	imagesResp, err := client.ListImagesExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting images: %w", err)
	}

	images := *imagesResp.Items
	for i := range images {
		if images[i].Id == nil {
			continue
		}
		if utils.Contains(imagesToDestroy, *images[i].Id) {
			err := client.DeleteImageExecute(ctx, testutil.ProjectId, *images[i].Id)
			if err != nil {
				return fmt.Errorf("destroying image %s during CheckDestroy: %w", *images[i].Id, err)
			}
		}
	}
	return nil
}
