package vpn_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/gateway-min.tf
var gatewayMinConfig string

//go:embed testdata/gateway-max.tf
var gatewayMaxConfig string

//go:embed testdata/connection-min.tf
var connectionMinConfig string

//go:embed testdata/connection-max.tf
var connectionMaxConfig string

var gatewayMinVars = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"display_name": config.StringVariable("vpn-gw-acc-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"plan_id":      config.StringVariable("p100"),
	"routing_type": config.StringVariable("ROUTE_BASED"),
	"az_tunnel1":   config.StringVariable("eu01-1"),
	"az_tunnel2":   config.StringVariable("eu01-2"),
}

var gatewayMinVarsUpdated = func() config.Variables {
	updated := make(config.Variables, len(gatewayMinVars))
	maps.Copy(updated, gatewayMinVars)
	updated["display_name"] = config.StringVariable("vpn-gw-acc-test-updated-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	updated["plan_id"] = config.StringVariable("p500")
	return updated
}()

var gatewayMaxVars = config.Variables{
	"project_id":         config.StringVariable(testutil.ProjectId),
	"region":             config.StringVariable(testutil.Region),
	"display_name":       config.StringVariable("vpn-gw-acc-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"plan_id":            config.StringVariable("p500"),
	"routing_type":       config.StringVariable("BGP_ROUTE_BASED"),
	"az_tunnel1":         config.StringVariable("eu01-1"),
	"az_tunnel2":         config.StringVariable("eu01-2"),
	"local_asn":          config.IntegerVariable(65000),
	"advertised_route_1": config.StringVariable("10.0.0.0/16"),
	"advertised_route_2": config.StringVariable("192.168.0.0/24"),
	"label_key":          config.StringVariable("env"),
	"label_value":        config.StringVariable("test"),
}

var gatewayMaxVarsUpdated = func() config.Variables {
	updated := make(config.Variables, len(gatewayMaxVars))
	maps.Copy(updated, gatewayMaxVars)
	updated["display_name"] = config.StringVariable("vpn-gw-acc-test-updated-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	updated["local_asn"] = config.IntegerVariable(4294967294)
	updated["label_value"] = config.StringVariable("production")
	updated["advertised_route_1"] = config.StringVariable("10.10.0.0/16")
	updated["advertised_route_2"] = config.StringVariable("192.168.167.0/24")
	updated["advertised_route_3"] = config.StringVariable("172.16.10.0/24")
	return updated
}()

var gatewayMaxVarsUpdated2 = func() config.Variables {
	updated := make(config.Variables, len(gatewayMaxVarsUpdated))
	maps.Copy(updated, gatewayMaxVarsUpdated)
	updated["advertised_route_1"] = config.StringVariable("")
	updated["advertised_route_2"] = config.StringVariable("")
	updated["advertised_route_3"] = config.StringVariable("")
	updated["label_key"] = config.StringVariable("")
	updated["label_value"] = config.StringVariable("")
	return updated
}()

var connectionMinVars = func() config.Variables {
	vars := make(config.Variables, len(gatewayMinVars)+5)
	maps.Copy(vars, gatewayMinVars)
	vars["connection_display_name"] = config.StringVariable("vpn-conn-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	vars["tunnel1_remote_address"] = config.StringVariable("203.0.113.1")
	vars["tunnel1_psk"] = config.StringVariable("Super.Secret_$hared3Key_1")
	vars["tunnel2_remote_address"] = config.StringVariable("203.0.113.2")
	vars["tunnel2_psk"] = config.StringVariable("Super.Secret_$hared3Key_2")
	return vars
}()

var connectionMinVarsUpdated = func() config.Variables {
	updated := make(config.Variables, len(connectionMinVars))
	maps.Copy(updated, connectionMinVars)
	updated["connection_display_name"] = config.StringVariable("vpn-conn-updated-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	return updated
}()

var connectionMaxVars = func() config.Variables {
	vars := make(config.Variables)
	maps.Copy(vars, gatewayMaxVars) // BGP_ROUTE_BASED gateway with local_asn, labels, etc.
	vars["connection_display_name"] = config.StringVariable("vpn-conn-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	vars["tunnel1_remote_address"] = config.StringVariable("203.0.113.1")
	vars["tunnel1_psk"] = config.StringVariable("Super.Secret_$hared3Key_1")
	vars["tunnel1_psk_version"] = config.IntegerVariable(1)
	vars["tunnel1_bgp_remote_asn"] = config.IntegerVariable(65001)
	vars["tunnel2_remote_address"] = config.StringVariable("203.0.113.2")
	vars["tunnel2_psk"] = config.StringVariable("Super.Secret_$hared3Key_2")
	vars["tunnel2_psk_version"] = config.IntegerVariable(1)
	vars["tunnel2_bgp_remote_asn"] = config.IntegerVariable(65002)
	vars["remote_subnet"] = config.StringVariable("10.10.10.0/24")
	vars["local_subnet"] = config.StringVariable("192.168.0.0/24")
	vars["tunnel1_local_peering"] = config.StringVariable("192.168.0.1")
	vars["tunnel1_remote_peering"] = config.StringVariable("10.10.10.1")
	vars["tunnel2_local_peering"] = config.StringVariable("192.168.0.2")
	vars["tunnel2_remote_peering"] = config.StringVariable("10.10.10.2")
	return vars
}()

// connectionMaxVarsUpdated changes non-PSK mutable fields to exercise updates.
var connectionMaxVarsUpdated = func() config.Variables {
	updated := make(config.Variables)
	maps.Copy(updated, connectionMaxVars)
	updated["connection_display_name"] = config.StringVariable("vpn-conn-updated-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	updated["tunnel1_bgp_remote_asn"] = config.IntegerVariable(65003)
	updated["tunnel2_bgp_remote_asn"] = config.IntegerVariable(65004)
	return updated
}()

// connectionMaxVarsPskRotated exercises the write-only PSK rotation workflow:
// both tunnel PSKs are replaced and their versions incremented from 1 → 2.
var connectionMaxVarsPskRotated = func() config.Variables {
	rotated := make(config.Variables)
	maps.Copy(rotated, connectionMaxVarsUpdated)
	rotated["tunnel1_psk"] = config.StringVariable("Super.Secret_Rotated_$hared3Key_1!")
	rotated["tunnel1_psk_version"] = config.IntegerVariable(2)
	rotated["tunnel2_psk"] = config.StringVariable("Super.Secret_Rotated_$hared3Key_2!")
	rotated["tunnel2_psk_version"] = config.IntegerVariable(2)
	return rotated
}()

func TestAccVpnGatewayResourceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVpnResourcesDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: gatewayMinVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(gatewayMinVars["display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(gatewayMinVars["plan_id"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(gatewayMinVars["routing_type"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(gatewayMinVars["az_tunnel1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(gatewayMinVars["az_tunnel2"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "gateway_id"),
				),
			},
			// Data source
			{
				ConfigVariables: gatewayMinVars,
				Config: fmt.Sprintf(`
						%s
						%s

						data "stackit_vpn_gateway" "gateway" {
							project_id = stackit_vpn_gateway.gateway.project_id
                			gateway_id = stackit_vpn_gateway.gateway.gateway_id
						}
						`,
					testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(gatewayMinVars["display_name"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(gatewayMinVars["plan_id"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(gatewayMinVars["routing_type"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(gatewayMinVars["az_tunnel1"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(gatewayMinVars["az_tunnel2"])),

					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway.gateway", "gateway_id"),

					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway.gateway", "region", "stackit_vpn_gateway.gateway", "region"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway.gateway", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
				),
			},
			// Status data source
			{
				ConfigVariables: gatewayMinVars,
				Config: fmt.Sprintf(`
						%s
						%s

						data "stackit_vpn_gateway_status" "gateway" {
							project_id = stackit_vpn_gateway.gateway.project_id
                			gateway_id = stackit_vpn_gateway.gateway.gateway_id
						}
						`,
					testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway_status.gateway", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "region", testutil.Region),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "display_name", testutil.ConvertConfigVariable(gatewayMinVars["display_name"])),

					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "tunnels.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "tunnels.0.name", string(vpn.VPNTUNNELSNAME_TUNNEL1)),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway_status.gateway", "tunnels.0.internal_next_hop_ip"),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway_status.gateway", "tunnels.0.public_ip"),

					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "tunnels.1.name", string(vpn.VPNTUNNELSNAME_TUNNEL2)),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway_status.gateway", "tunnels.1.internal_next_hop_ip"),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway_status.gateway", "tunnels.1.public_ip"),

					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "connections.#", "0"),
				),
			},
			// Update
			{
				ConfigVariables: gatewayMinVarsUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(gatewayMinVarsUpdated["display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(gatewayMinVarsUpdated["plan_id"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(gatewayMinVarsUpdated["routing_type"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(gatewayMinVarsUpdated["az_tunnel1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(gatewayMinVarsUpdated["az_tunnel2"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "gateway_id"),
				),
			},
			// Import
			{
				ConfigVariables: gatewayMinVars,
				ResourceName:    "stackit_vpn_gateway.gateway",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpn_gateway.gateway"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_vpn_gateway.gateway")
					}
					gatewayId, ok := r.Primary.Attributes["gateway_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute gateway_id")
					}
					return fmt.Sprintf("%s,%s,%s",
						testutil.ProjectId,
						testutil.Region,
						gatewayId,
					), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccVpnGatewayResourceMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVpnResourcesDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: gatewayMaxVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "region", testutil.ConvertConfigVariable(gatewayMaxVars["region"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(gatewayMaxVars["display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(gatewayMaxVars["plan_id"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(gatewayMaxVars["routing_type"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(gatewayMaxVars["az_tunnel1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(gatewayMaxVars["az_tunnel2"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.local_asn", testutil.ConvertConfigVariable(gatewayMaxVars["local_asn"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.0", testutil.ConvertConfigVariable(gatewayMaxVars["advertised_route_1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.1", testutil.ConvertConfigVariable(gatewayMaxVars["advertised_route_2"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "labels."+testutil.ConvertConfigVariable(gatewayMaxVars["label_key"]), testutil.ConvertConfigVariable(gatewayMaxVars["label_value"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "gateway_id"),
				),
			},
			// Data source
			{
				ConfigVariables: gatewayMaxVars,
				Config: fmt.Sprintf(`
						%s
						%s

						data "stackit_vpn_gateway" "gateway" {
							project_id = stackit_vpn_gateway.gateway.project_id
							gateway_id = stackit_vpn_gateway.gateway.gateway_id
						}
						`,
					testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "project_id", testutil.ConvertConfigVariable(gatewayMaxVars["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(gatewayMaxVars["display_name"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(gatewayMaxVars["plan_id"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(gatewayMaxVars["routing_type"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(gatewayMaxVars["az_tunnel1"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(gatewayMaxVars["az_tunnel2"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "bgp.local_asn", testutil.ConvertConfigVariable(gatewayMaxVars["local_asn"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.0", testutil.ConvertConfigVariable(gatewayMaxVars["advertised_route_1"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.1", testutil.ConvertConfigVariable(gatewayMaxVars["advertised_route_2"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway.gateway", "labels."+testutil.ConvertConfigVariable(gatewayMaxVars["label_key"]), testutil.ConvertConfigVariable(gatewayMaxVars["label_value"])),

					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway.gateway", "gateway_id"),

					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway.gateway", "region", "stackit_vpn_gateway.gateway", "region"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway.gateway", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
				),
			},
			// Status data source
			{
				ConfigVariables: gatewayMaxVars,
				Config: fmt.Sprintf(`
						%s
						%s

						data "stackit_vpn_gateway_status" "gateway" {
							project_id = stackit_vpn_gateway.gateway.project_id
                			gateway_id = stackit_vpn_gateway.gateway.gateway_id
						}
						`,
					testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway_status.gateway", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "region", testutil.Region),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "display_name", testutil.ConvertConfigVariable(gatewayMaxVars["display_name"])),

					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "tunnels.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "tunnels.0.name", string(vpn.VPNTUNNELSNAME_TUNNEL1)),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway_status.gateway", "tunnels.0.internal_next_hop_ip"),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway_status.gateway", "tunnels.0.public_ip"),

					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "tunnels.1.name", string(vpn.VPNTUNNELSNAME_TUNNEL2)),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway_status.gateway", "tunnels.1.internal_next_hop_ip"),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway_status.gateway", "tunnels.1.public_ip"),

					resource.TestCheckResourceAttr("data.stackit_vpn_gateway_status.gateway", "connections.#", "0"),
				),
			},
			// Update
			{
				ConfigVariables: gatewayMaxVarsUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "region", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["region"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["plan_id"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["routing_type"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["az_tunnel1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["az_tunnel2"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.local_asn", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["local_asn"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.#", "3"),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.0", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["advertised_route_1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.1", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["advertised_route_2"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.2", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["advertised_route_3"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "labels."+testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["label_key"]), testutil.ConvertConfigVariable(gatewayMaxVarsUpdated["label_value"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "gateway_id"),
				),
			},
			// Update step 2 - test removal of optional fields
			{
				ConfigVariables: gatewayMaxVarsUpdated2,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "region", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated2["region"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated2["display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated2["plan_id"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated2["routing_type"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated2["az_tunnel1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated2["az_tunnel2"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.local_asn", testutil.ConvertConfigVariable(gatewayMaxVarsUpdated2["local_asn"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.#", "0"),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "labels.#", "0"),
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "gateway_id"),
				),
			},
			// Import
			{
				ConfigVariables: gatewayMaxVars,
				ResourceName:    "stackit_vpn_gateway.gateway",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpn_gateway.gateway"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_vpn_gateway.gateway")
					}
					gatewayId, ok := r.Primary.Attributes["gateway_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute gateway_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, gatewayId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccVpnConnectionResourceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVpnResourcesDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: connectionMinVars,
				Config:          fmt.Sprintf("%s\n%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMinConfig, connectionMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Gateway
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(connectionMinVars["display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(connectionMinVars["plan_id"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(connectionMinVars["routing_type"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(connectionMinVars["az_tunnel1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(connectionMinVars["az_tunnel2"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "gateway_id"),
					// Connection – identity & top-level
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "display_name", testutil.ConvertConfigVariable(connectionMinVars["connection_display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "enabled", "true"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "connection_id"),
					resource.TestCheckResourceAttrPair("stackit_vpn_connection.connection", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
					// Connection – tunnel1
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.remote_address", testutil.ConvertConfigVariable(connectionMinVars["tunnel1_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.0", "sha2_384"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase1.rekey_time"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.dh_groups.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.encryption_algorithms.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.integrity_algorithms.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.integrity_algorithms.0", "sha2_384"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase2.rekey_time"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase2.start_action"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase2.dpd_action"),
					// Connection – tunnel2
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.remote_address", testutil.ConvertConfigVariable(connectionMinVars["tunnel2_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.dh_groups.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.encryption_algorithms.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.integrity_algorithms.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.integrity_algorithms.0", "sha2_384"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase1.rekey_time"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.dh_groups.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.encryption_algorithms.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.integrity_algorithms.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.integrity_algorithms.0", "sha2_384"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase2.rekey_time"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase2.start_action"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase2.dpd_action"),
				),
			},
			// Data source
			{
				ConfigVariables: connectionMinVars,
				Config: fmt.Sprintf(`
						%s
						%s
						%s

						data "stackit_vpn_connection" "connection" {
							project_id    = stackit_vpn_connection.connection.project_id
							gateway_id    = stackit_vpn_connection.connection.gateway_id
							connection_id = stackit_vpn_connection.connection.connection_id
						}
						`,
					testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMinConfig, connectionMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "region", testutil.Region),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "display_name", testutil.ConvertConfigVariable(connectionMinVars["connection_display_name"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "enabled", "true"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.remote_address", testutil.ConvertConfigVariable(connectionMinVars["tunnel1_remote_address"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.0", "sha2_384"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.remote_address", testutil.ConvertConfigVariable(connectionMinVars["tunnel2_remote_address"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.phase1.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.phase1.integrity_algorithms.0", "sha2_384"),

					resource.TestCheckResourceAttrSet("data.stackit_vpn_connection.connection", "connection_id"),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_connection.connection", "gateway_id"),

					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "project_id", "stackit_vpn_connection.connection", "project_id"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "region", "stackit_vpn_connection.connection", "region"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "gateway_id", "stackit_vpn_connection.connection", "gateway_id"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "connection_id", "stackit_vpn_connection.connection", "connection_id"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "display_name", "stackit_vpn_connection.connection", "display_name"),
				),
			},
			// Update
			{
				ConfigVariables: connectionMinVarsUpdated,
				Config:          fmt.Sprintf("%s\n%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMinConfig, connectionMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Gateway unchanged
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(connectionMinVarsUpdated["display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(connectionMinVarsUpdated["plan_id"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(connectionMinVarsUpdated["routing_type"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(connectionMinVarsUpdated["az_tunnel1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(connectionMinVarsUpdated["az_tunnel2"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "gateway_id"),
					// Connection – all fields
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "display_name", testutil.ConvertConfigVariable(connectionMinVarsUpdated["connection_display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "enabled", "true"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "connection_id"),
					resource.TestCheckResourceAttrPair("stackit_vpn_connection.connection", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.remote_address", testutil.ConvertConfigVariable(connectionMinVarsUpdated["tunnel1_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.0", "sha2_384"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase1.rekey_time"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.integrity_algorithms.0", "sha2_384"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase2.rekey_time"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase2.start_action"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase2.dpd_action"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.remote_address", testutil.ConvertConfigVariable(connectionMinVarsUpdated["tunnel2_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.integrity_algorithms.0", "sha2_384"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase1.rekey_time"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.dh_groups.0", "ecp384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.integrity_algorithms.0", "sha2_384"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase2.rekey_time"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase2.start_action"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase2.dpd_action"),
				),
			},
			// Import
			{
				ConfigVariables: connectionMinVars,
				ResourceName:    "stackit_vpn_connection.connection",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpn_connection.connection"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_vpn_connection.connection")
					}
					connectionId, ok := r.Primary.Attributes["connection_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute connection_id")
					}
					gatewayId, ok := r.Primary.Attributes["gateway_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute gateway_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s",
						testutil.ProjectId,
						testutil.Region,
						gatewayId,
						connectionId,
					), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tunnel1.pre_shared_key_wo", "tunnel2.pre_shared_key_wo"},
			},
		},
	})
}

func TestAccVpnConnectionResourceMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVpnResourcesDestroy,
		Steps: []resource.TestStep{
			// Creation – BGP_ROUTE_BASED gateway + full connection config including BGP tunnel peers
			{
				ConfigVariables: connectionMaxVars,
				Config:          fmt.Sprintf("%s\n%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMaxConfig, connectionMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Gateway
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "region", testutil.ConvertConfigVariable(connectionMaxVars["region"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(connectionMaxVars["display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(connectionMaxVars["plan_id"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(connectionMaxVars["routing_type"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel1", testutil.ConvertConfigVariable(connectionMaxVars["az_tunnel1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "availability_zones.tunnel2", testutil.ConvertConfigVariable(connectionMaxVars["az_tunnel2"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.local_asn", testutil.ConvertConfigVariable(connectionMaxVars["local_asn"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.0", testutil.ConvertConfigVariable(connectionMaxVars["advertised_route_1"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.override_advertised_routes.1", testutil.ConvertConfigVariable(connectionMaxVars["advertised_route_2"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "labels."+testutil.ConvertConfigVariable(connectionMaxVars["label_key"]), testutil.ConvertConfigVariable(connectionMaxVars["label_value"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "gateway_id"),
					// Connection – identity & top-level
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "region", testutil.ConvertConfigVariable(connectionMaxVars["region"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "display_name", testutil.ConvertConfigVariable(connectionMaxVars["connection_display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "enabled", "true"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "remote_subnet.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "remote_subnet.0", testutil.ConvertConfigVariable(connectionMaxVars["remote_subnet"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "local_subnet.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "local_subnet.0", testutil.ConvertConfigVariable(connectionMaxVars["local_subnet"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "connection_id"),
					resource.TestCheckResourceAttrPair("stackit_vpn_connection.connection", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
					// Connection – tunnel1
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.remote_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel1_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.pre_shared_key_wo_version", testutil.ConvertConfigVariable(connectionMaxVars["tunnel1_psk_version"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.0", "modp2048"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.1", "ecp256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.1", "aes128gcm16"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.0", "sha2_256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.1", "sha2_384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.rekey_time", "25920"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.dh_groups.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.dh_groups.0", "modp2048"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.dh_groups.1", "ecp256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.encryption_algorithms.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.encryption_algorithms.1", "aes128gcm16"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.integrity_algorithms.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.integrity_algorithms.0", "sha2_256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.integrity_algorithms.1", "sha2_384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.rekey_time", "3240"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.start_action", "start"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase2.dpd_action"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.peering.local_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel1_local_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.peering.remote_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel1_remote_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.bgp.remote_asn", testutil.ConvertConfigVariable(connectionMaxVars["tunnel1_bgp_remote_asn"])),
					// Connection – tunnel2
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.remote_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel2_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.pre_shared_key_wo_version", testutil.ConvertConfigVariable(connectionMaxVars["tunnel2_psk_version"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.dh_groups.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.dh_groups.0", "modp2048"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.dh_groups.1", "ecp256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.encryption_algorithms.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.encryption_algorithms.1", "aes128gcm16"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.integrity_algorithms.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.integrity_algorithms.0", "sha2_256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.integrity_algorithms.1", "sha2_384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.rekey_time", "25920"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.dh_groups.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.dh_groups.0", "modp2048"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.dh_groups.1", "ecp256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.encryption_algorithms.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.encryption_algorithms.1", "aes128gcm16"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.integrity_algorithms.#", "2"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.integrity_algorithms.0", "sha2_256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.integrity_algorithms.1", "sha2_384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.rekey_time", "3240"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.start_action", "start"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase2.dpd_action"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.peering.local_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel2_local_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.peering.remote_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel2_remote_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.bgp.remote_asn", testutil.ConvertConfigVariable(connectionMaxVars["tunnel2_bgp_remote_asn"])),
				),
			},
			// Data source
			{
				ConfigVariables: connectionMaxVars,
				Config: fmt.Sprintf(`
						%s
						%s
						%s

						data "stackit_vpn_connection" "connection" {
							project_id    = stackit_vpn_connection.connection.project_id
							gateway_id    = stackit_vpn_connection.connection.gateway_id
							connection_id = stackit_vpn_connection.connection.connection_id
						}
						`,
					testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMaxConfig, connectionMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "region", testutil.ConvertConfigVariable(connectionMaxVars["region"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "display_name", testutil.ConvertConfigVariable(connectionMaxVars["connection_display_name"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "enabled", "true"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "remote_subnet.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "remote_subnet.0", testutil.ConvertConfigVariable(connectionMaxVars["remote_subnet"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "local_subnet.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "local_subnet.0", testutil.ConvertConfigVariable(connectionMaxVars["local_subnet"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.remote_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel1_remote_address"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.0", "modp2048"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.1", "ecp256"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.1", "aes128gcm16"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.0", "sha2_256"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.1", "sha2_384"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase1.rekey_time", "25920"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase2.rekey_time", "3240"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.phase2.start_action", "start"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.peering.local_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel1_local_peering"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.peering.remote_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel1_remote_peering"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel1.bgp.remote_asn", testutil.ConvertConfigVariable(connectionMaxVars["tunnel1_bgp_remote_asn"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.remote_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel2_remote_address"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.phase1.rekey_time", "25920"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.phase2.rekey_time", "3240"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.phase2.start_action", "start"),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.peering.local_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel2_local_peering"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.peering.remote_address", testutil.ConvertConfigVariable(connectionMaxVars["tunnel2_remote_peering"])),
					resource.TestCheckResourceAttr("data.stackit_vpn_connection.connection", "tunnel2.bgp.remote_asn", testutil.ConvertConfigVariable(connectionMaxVars["tunnel2_bgp_remote_asn"])),

					resource.TestCheckResourceAttrSet("data.stackit_vpn_connection.connection", "connection_id"),
					resource.TestCheckResourceAttrSet("data.stackit_vpn_connection.connection", "gateway_id"),

					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "project_id", "stackit_vpn_connection.connection", "project_id"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "region", "stackit_vpn_connection.connection", "region"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "gateway_id", "stackit_vpn_connection.connection", "gateway_id"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "connection_id", "stackit_vpn_connection.connection", "connection_id"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_connection.connection", "display_name", "stackit_vpn_connection.connection", "display_name"),
				),
			},
			// Update – change display name and BGP remote ASNs; verify no other drift
			{
				ConfigVariables: connectionMaxVarsUpdated,
				Config:          fmt.Sprintf("%s\n%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMaxConfig, connectionMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Gateway unchanged
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "region", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["region"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "display_name", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "plan_id", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["plan_id"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "routing_type", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["routing_type"])),
					resource.TestCheckResourceAttr("stackit_vpn_gateway.gateway", "bgp.local_asn", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["local_asn"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "gateway_id"),
					// Connection
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "region", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["region"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "display_name", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["connection_display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "enabled", "true"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "remote_subnet.0", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["remote_subnet"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "local_subnet.0", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["local_subnet"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "connection_id"),
					resource.TestCheckResourceAttrPair("stackit_vpn_connection.connection", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
					// tunnel1
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.remote_address", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel1_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.pre_shared_key_wo_version", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel1_psk_version"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.0", "modp2048"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.dh_groups.1", "ecp256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.encryption_algorithms.1", "aes128gcm16"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.0", "sha2_256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.integrity_algorithms.1", "sha2_384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.rekey_time", "25920"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.rekey_time", "3240"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.start_action", "start"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase2.dpd_action"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.peering.local_address", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel1_local_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.peering.remote_address", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel1_remote_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.bgp.remote_asn", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel1_bgp_remote_asn"])),
					// tunnel2
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.remote_address", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel2_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.pre_shared_key_wo_version", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel2_psk_version"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.dh_groups.0", "modp2048"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.dh_groups.1", "ecp256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.encryption_algorithms.0", "aes256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.encryption_algorithms.1", "aes128gcm16"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.integrity_algorithms.0", "sha2_256"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.integrity_algorithms.1", "sha2_384"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.rekey_time", "25920"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.rekey_time", "3240"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.start_action", "start"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase2.dpd_action"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.peering.local_address", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel2_local_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.peering.remote_address", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel2_remote_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.bgp.remote_asn", testutil.ConvertConfigVariable(connectionMaxVarsUpdated["tunnel2_bgp_remote_asn"])),
				),
			},
			// PSK rotation – increment pre_shared_key_wo_version 1 → 2 on both tunnels.
			// The write-only pre_shared_key_wo values are replaced; the provider reads the
			// version from state to detect the rotation and re-sends the new key to the API.
			// Verifying the new version value in state (and no unintended plan diff) is the
			// observable signal that the rotation was applied correctly.
			{
				ConfigVariables: connectionMaxVarsPskRotated,
				Config:          fmt.Sprintf("%s\n%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), gatewayMaxConfig, connectionMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Rotated version counters must be persisted in state
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.pre_shared_key_wo_version", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel1_psk_version"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.pre_shared_key_wo_version", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel2_psk_version"])),
					// All other fields must be unchanged – catches unintended drift
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "region", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["region"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "display_name", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["connection_display_name"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "enabled", "true"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "remote_subnet.0", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["remote_subnet"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "local_subnet.0", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["local_subnet"])),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "connection_id"),
					resource.TestCheckResourceAttrPair("stackit_vpn_connection.connection", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.remote_address", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel1_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase1.rekey_time", "25920"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.rekey_time", "3240"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.phase2.start_action", "start"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel1.phase2.dpd_action"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.peering.local_address", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel1_local_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.peering.remote_address", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel1_remote_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel1.bgp.remote_asn", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel1_bgp_remote_asn"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.remote_address", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel2_remote_address"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase1.rekey_time", "25920"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.rekey_time", "3240"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.phase2.start_action", "start"),
					resource.TestCheckResourceAttrSet("stackit_vpn_connection.connection", "tunnel2.phase2.dpd_action"),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.peering.local_address", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel2_local_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.peering.remote_address", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel2_remote_peering"])),
					resource.TestCheckResourceAttr("stackit_vpn_connection.connection", "tunnel2.bgp.remote_asn", testutil.ConvertConfigVariable(connectionMaxVarsPskRotated["tunnel2_bgp_remote_asn"])),
				),
			},
			// Import
			{
				ConfigVariables: connectionMaxVars,
				ResourceName:    "stackit_vpn_connection.connection",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpn_connection.connection"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_vpn_connection.connection")
					}
					connectionId, ok := r.Primary.Attributes["connection_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute connection_id")
					}
					gatewayId, ok := r.Primary.Attributes["gateway_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute gateway_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s",
						testutil.ProjectId,
						testutil.Region,
						gatewayId,
						connectionId,
					), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tunnel1.pre_shared_key_wo", "tunnel2.pre_shared_key_wo", "tunnel1.pre_shared_key_wo_version", "tunnel2.pre_shared_key_wo_version"},
			},
		},
	})
}

func testAccCheckVpnResourcesDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := vpn.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.VpnCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	gatewayIdsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		var gatewayId string
		switch rs.Type {
		case "stackit_vpn_gateway":
			// gateway terraform ID: "[project_id],[region],[gateway_id]"
			parts := strings.Split(rs.Primary.ID, core.Separator)
			if len(parts) > 2 {
				gatewayId = parts[2]
			} else if attrId, ok := rs.Primary.Attributes["gateway_id"]; ok && attrId != "" {
				gatewayId = attrId
			}
		case "stackit_vpn_connection":
			// connection terraform ID: "[project_id],[region],[gateway_id],[connection_id]"
			parts := strings.Split(rs.Primary.ID, core.Separator)
			if len(parts) > 2 {
				gatewayId = parts[2]
			} else if attrId, ok := rs.Primary.Attributes["gateway_id"]; ok && attrId != "" {
				gatewayId = attrId
			}
		default:
			continue
		}
		if gatewayId == "" {
			continue
		}
		if !slices.Contains(gatewayIdsToDestroy, gatewayId) {
			gatewayIdsToDestroy = append(gatewayIdsToDestroy, gatewayId)
		}
	}

	if len(gatewayIdsToDestroy) == 0 {
		return nil
	}

	gatewaysResp, err := client.DefaultAPI.ListGateways(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("listing gateways during CheckDestroy: %w", err)
	}

	for _, gateway := range gatewaysResp.Gateways {
		if gateway.Id == nil || !slices.Contains(gatewayIdsToDestroy, *gateway.Id) {
			continue
		}

		connectionsResp, err := client.DefaultAPI.ListGatewayConnections(ctx, testutil.ProjectId, testutil.Region, *gateway.Id).Execute()
		if err != nil {
			return fmt.Errorf("listing connections for gateway %s during CheckDestroy: %w", *gateway.Id, err)
		}
		for _, conn := range connectionsResp.Connections {
			if conn.Id == nil {
				continue
			}
			err := client.DefaultAPI.DeleteGatewayConnection(ctx, testutil.ProjectId, testutil.Region, *gateway.Id, *conn.Id).Execute()
			if err != nil {
				var oapiErr *oapierror.GenericOpenAPIError
				if errors.As(err, &oapiErr) && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusGone) {
					continue
				}
				return fmt.Errorf("destroying connection %s during CheckDestroy: %w", *conn.Id, err)
			}
		}

		err = client.DefaultAPI.DeleteGateway(ctx, testutil.ProjectId, testutil.Region, *gateway.Id).Execute()
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusGone) {
				continue
			}
			return fmt.Errorf("destroying gateway %s during CheckDestroy: %w", *gateway.Id, err)
		}
	}
	return nil
}
