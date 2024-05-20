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

// Network resource data
var networkResource = map[string]string{
	"project_id":         testutil.ProjectId,
	"name":               fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)),
	"ipv4_prefix_length": "24",
	"nameserver0":        "1.2.3.4",
	"nameserver1":        "5.6.7.8",
}

func resourceConfig(name, nameservers string) string {
	return fmt.Sprintf(`
				%s

				resource "stackit_network" "network" {
					project_id = "%s"
					name       = "%s"
					ipv4_prefix_length = "%s"
					nameservers = %s
				}
				`,
		testutil.IaaSProviderConfig(),
		networkResource["project_id"],
		name,
		networkResource["ipv4_prefix_length"],
		nameservers,
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
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", networkResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "name", networkResource["name"]),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.#", "1"),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.0", networkResource["nameserver0"]),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_network" "network" {
						project_id  = stackit_network.network.project_id
						network_id = stackit_network.network.network_id
					}`,
					resourceConfig(
						networkResource["name"],
						fmt.Sprintf(
							"[%q]",
							networkResource["nameserver0"],
						),
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
			// Update
			{
				Config: resourceConfig(
					fmt.Sprintf("%s-updated", networkResource["name"]),
					fmt.Sprintf(
						"[%q, %q]",
						networkResource["nameserver0"],
						networkResource["nameserver1"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", networkResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "name", fmt.Sprintf("%s-updated", networkResource["name"])),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.#", "2"),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.0", networkResource["nameserver0"]),
					resource.TestCheckResourceAttr("stackit_network.network", "nameservers.1", networkResource["nameserver1"]),
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
	return nil
}
