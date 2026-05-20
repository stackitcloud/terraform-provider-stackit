package vpn_test

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
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/gateway-min.tf
var gatewayMinConfig string

//go:embed testdata/gateway-max.tf
var gatewayMaxConfig string

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

func TestAccVpnGatewayResourceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVpnGatewayDestroy,
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
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "state"),
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
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway.gateway", "state"),

					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway.gateway", "region", "stackit_vpn_gateway.gateway", "region"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway.gateway", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
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
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "state"),
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
		CheckDestroy:             testAccCheckVpnGatewayDestroy,
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
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "state"),
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
					resource.TestCheckResourceAttrSet("data.stackit_vpn_gateway.gateway", "state"),

					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway.gateway", "region", "stackit_vpn_gateway.gateway", "region"),
					resource.TestCheckResourceAttrPair("data.stackit_vpn_gateway.gateway", "gateway_id", "stackit_vpn_gateway.gateway", "gateway_id"),
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
					resource.TestCheckResourceAttrSet("stackit_vpn_gateway.gateway", "state"),
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

func testAccCheckVpnGatewayDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := vpn.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.VpnCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	gatewaysToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_vpn_gateway" {
			continue
		}
		// gateway terraform ID: "[project_id],[region],[gateway_id]"
		gatewayId := strings.Split(rs.Primary.ID, core.Separator)[2]
		gatewaysToDestroy = append(gatewaysToDestroy, gatewayId)
	}

	gatewaysResp, err := client.DefaultAPI.ListGateways(ctx, testutil.ProjectId, vpn.Region(testutil.Region)).Execute()
	if err != nil {
		return fmt.Errorf("getting gateways: %w", err)
	}

	gateways := gatewaysResp.Gateways
	for _, gateway := range gateways {
		if gateway.Id == nil {
			continue
		}
		for _, gatewayId := range gatewaysToDestroy {
			if *gateway.Id == gatewayId {
				err := client.DefaultAPI.DeleteGateway(ctx, testutil.ProjectId, vpn.Region(testutil.Region), *gateway.Id).Execute()
				if err != nil {
					return fmt.Errorf("destroying gateway %s during CheckDestroy: %w", gatewayId, err)
				}
			}
		}
	}
	return nil
}
