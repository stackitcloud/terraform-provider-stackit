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
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Network resource data
var networkResource = map[string]string{
	"project_id":         testutil.ProjectId,
	"name":               fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"ipv4_prefix_length": "24",
	"nameserver0":        "1.2.3.4",
	"nameserver1":        "5.6.7.8",
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
	"name":              "name",
	"description":       "description",
	"size":              "1",
	"label1":            "value",
	"performance_class": "storage_premium_perf1",
}

// Security Group resource data
var securityGroupResource = map[string]string{
	"project_id":  testutil.ProjectId,
	"name":        "name",
	"description": "description",
	"label1":      "value",
}

func networkResourceConfig(name, nameservers string) string {
	return fmt.Sprintf(`
				resource "stackit_network" "network" {
					project_id = "%s"
					name       = "%s"
					ipv4_prefix_length = "%s"
					nameservers = %s
				}
				`,
		networkResource["project_id"],
		name,
		networkResource["ipv4_prefix_length"],
		nameservers,
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

func networkAreaRouteResourceConfig() string {
	return fmt.Sprintf(`
				resource "stackit_network_area_route" "network_area_route" {
					organization_id = stackit_network_area.network_area.organization_id
					network_area_id = stackit_network_area.network_area.network_area_id
					prefix          = "%s"
					next_hop        = "%s"
				}
				`,
		networkAreaRouteResource["prefix"],
		networkAreaRouteResource["next_hop"],
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
		volumeResource["project_id"],
		name,
		volumeResource["description"],
		volumeResource["label1"],
	)
}

func resourceConfig(name, nameservers, areaname, networkranges string) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		testutil.IaaSProviderConfig(),
		networkResourceConfig(name, nameservers),
		networkAreaResourceConfig(areaname, networkranges),
		networkAreaRouteResourceConfig(),
		networkInterfaceResourceConfig(interfacename),
	)
}

func resourceConfigVolume(name, size string) string {
	return fmt.Sprintf("%s\n\n%s",
		testutil.IaaSProviderConfig(),
		volumeResourceConfig(name, size),
	)
}

func resourceConfigSecurityGroup(name string) string {
	return fmt.Sprintf("%s\n\n%s",
		testutil.IaaSProviderConfig(),
		securityGroupResourceConfig(name),
	)
}

func TestAccIaaS(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIaaSDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfig(
					networkResource["name"],
					fmt.Sprintf(
						"[%q]",
						networkResource["nameserver0"],
					),
					networkAreaResource["name"],
					networkAreaResource["networkrange0"],
					networkInterfaceResource["name"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", networkResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "name", networkResource["name"]),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.#", "1"),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.0", networkResource["nameserver0"]),

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
			// Data source
			{
				Config: fmt.Sprintf(`
					%s
			
					data "stackit_network" "network" {
						project_id  = stackit_network.network.project_id
						network_id = stackit_network.network.network_id
					}
			
					data "stackit_network_area" "network_area" {
						organization_id  = stackit_network_area.network_area.organization_id
						network_area_id  = stackit_network_area.network_area.network_area_id
					}
					
					data "stackit_network_area_route" "network_area_route" {
						organization_id  	  = stackit_network_area.network_area.organization_id
						network_area_id  	  = stackit_network_area.network_area.network_area_id
						network_area_route_id = stackit_network_area_route.network_area_route.network_area_route_id
					}

					data "stackit_network_interface" "network_interface" {
						project_id  	     = stackit_network.network.project_id
						network_id  	     = stackit_network.network.network_id
						network_interface_id = stackit_network_interface.network_interface.network_interface_id
					}
					`,
					resourceConfig(
						networkResource["name"],
						fmt.Sprintf(
							"[%q]",
							networkResource["nameserver0"],
						),
						networkAreaResource["name"],
						networkAreaResource["networkrange0"],
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
				ImportStateVerifyIgnore: []string{"ipv4_prefix_length"}, // Field is not returned by the API
			},
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
			// Update
			{
				Config: resourceConfig(
					fmt.Sprintf("%s-updated", networkResource["name"]),
					fmt.Sprintf(
						"[%q, %q]",
						networkResource["nameserver0"],
						networkResource["nameserver1"],
					),
					fmt.Sprintf("%s-updated", networkAreaResource["name"]),
					networkAreaResource["networkrange0"],
					fmt.Sprintf("%s-updated", networkInterfaceResource["name"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", networkResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "name", fmt.Sprintf("%s-updated", networkResource["name"])),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.#", "2"),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.0", networkResource["nameserver0"]),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.1", networkResource["nameserver1"]),

					// Network area
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "organization_id", networkAreaResource["organization_id"]),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_area_id"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "name", fmt.Sprintf("%s-updated", networkAreaResource["name"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.0.prefix", networkAreaResource["networkrange0"]),

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

func TestAccIaaSVolume(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIaaSVolumeDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfigVolume(volumeResource["name"], volumeResource["size"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_volume.volume", "project_id", volumeResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_volume.volume", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume", "name", volumeResource["name"]),
					resource.TestCheckResourceAttr("stackit_volume.volume", "availability_zone", volumeResource["availability_zone"]),
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
					resourceConfigVolume(volumeResource["name"], volumeResource["size"]),
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
				Config: resourceConfigVolume(
					fmt.Sprintf("%s-updated", volumeResource["name"]),
					"10",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_volume.volume", "project_id", volumeResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_volume.volume", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume", "name", fmt.Sprintf("%s-updated", volumeResource["name"])),
					resource.TestCheckResourceAttr("stackit_volume.volume", "availability_zone", volumeResource["availability_zone"]),
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

func TestAccIaaSSecurityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIaaSSecurityGroupDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfigSecurityGroup(securityGroupResource["name"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "project_id", securityGroupResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group", "security_group_id"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "name", securityGroupResource["name"]),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "labels.label1", securityGroupResource["label1"]),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "description", securityGroupResource["description"]),
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
					`,
					resourceConfigSecurityGroup(securityGroupResource["name"]),
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
			// Update
			{
				Config: resourceConfigSecurityGroup(
					fmt.Sprintf("%s-updated", securityGroupResource["name"]),
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

func testAccCheckIaaSDestroy(s *terraform.State) error {
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
	var client *iaasalpha.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaasalpha.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		client, err = iaasalpha.NewAPIClient(
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

func testAccCheckIaaSSecurityGroupDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaasalpha.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaasalpha.NewAPIClient(
			config.WithRegion("eu01"),
		)
	} else {
		client, err = iaasalpha.NewAPIClient(
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
