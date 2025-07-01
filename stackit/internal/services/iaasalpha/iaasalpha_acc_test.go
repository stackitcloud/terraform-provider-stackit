package iaasalpha_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

const (
	testNetworkAreaId = "25bbf23a-8134-4439-9f5e-1641caf8354e"
)

var (
	//go:embed testdata/resource-routingtable-min.tf
	resourceRoutingTableMinConfig string

	//go:embed testdata/resource-routingtable-max.tf
	resourceRoutingTableMaxConfig string

	//go:embed testdata/resource-routingtable-route-min.tf
	resourceRoutingTableRouteMinConfig string

	//go:embed testdata/resource-routingtable-route-max.tf
	resourceRoutingTableRouteMaxConfig string
)

var testConfigRoutingTableMin = config.Variables{
	"organization_id": config.StringVariable(testutil.OrganizationId),
	"network_area_id": config.StringVariable(testNetworkAreaId),
	"name":            config.StringVariable(fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
}

var testConfigRoutingTableMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigRoutingTableMin {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)))
	return updatedConfig
}()

var testConfigRoutingTableMax = config.Variables{
	"organization_id": config.StringVariable(testutil.OrganizationId),
	"network_area_id": config.StringVariable(testNetworkAreaId),
	"name":            config.StringVariable(fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"description":     config.StringVariable("This is the description of the routing table."),
	"label":           config.StringVariable("routing-table-label-01"),
	"system_routes":   config.BoolVariable(false),
	"region":          config.StringVariable(testutil.Region),
}

var testConfigRoutingTableMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigRoutingTableMax {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("acc-test-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)))
	updatedConfig["description"] = config.StringVariable("This is the updated description of the routing table.")
	updatedConfig["label"] = config.StringVariable("routing-table-updated-label-01")
	return updatedConfig
}()

// execute min and max tests with t.Run() to prevent parallel runs
func TestAccRoutingTable(t *testing.T) {
	t.Run("min", func(t *testing.T) {
		t.Logf("TestAccRoutingTableMin name: %s", testutil.ConvertConfigVariable(testConfigRoutingTableMin["name"]))
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
			CheckDestroy:             testAccCheckDestroy,
			Steps: []resource.TestStep{
				// Creation
				{
					ConfigVariables: testConfigRoutingTableMin,
					Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceRoutingTableMinConfig),
					Check: resource.ComposeAggregateTestCheckFunc(
						// Routing table
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "organization_id", testutil.ConvertConfigVariable(testConfigRoutingTableMin["organization_id"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "network_area_id", testutil.ConvertConfigVariable(testConfigRoutingTableMin["network_area_id"])),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "routing_table_id"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigRoutingTableMin["name"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "labels.%", "0"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "region", testutil.Region),
						resource.TestCheckNoResourceAttr("stackit_routing_table.routing_table", "description"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "system_routes", "true"),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "created_at"),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "updated_at"),
					),
				},
				// Data sources
				{
					ConfigVariables: testConfigRoutingTableMin,
					Config: fmt.Sprintf(`
					%s
					%s
					
					# single routing table
					data "stackit_routing_table" "routing_table" {
						organization_id  = stackit_routing_table.routing_table.organization_id
						network_area_id  = stackit_routing_table.routing_table.network_area_id
						routing_table_id  = stackit_routing_table.routing_table.routing_table_id
					}
					
					# all routing tables in network area
					data "stackit_routing_tables" "routing_tables" {
						organization_id  = stackit_routing_table.routing_table.organization_id
						network_area_id  = stackit_routing_table.routing_table.network_area_id
					}
					`,
						testutil.IaaSProviderConfig(), resourceRoutingTableMinConfig,
					),
					Check: resource.ComposeAggregateTestCheckFunc(
						// Routing table
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "organization_id", testutil.ConvertConfigVariable(testConfigRoutingTableMin["organization_id"])),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "network_area_id", testutil.ConvertConfigVariable(testConfigRoutingTableMin["network_area_id"])),
						resource.TestCheckResourceAttrPair(
							"stackit_routing_table.routing_table", "routing_table_id",
							"data.stackit_routing_table.routing_table", "routing_table_id",
						),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigRoutingTableMin["name"])),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "labels.%", "0"),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "region", testutil.Region),
						resource.TestCheckNoResourceAttr("data.stackit_routing_table.routing_table", "description"),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "system_routes", "true"),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "default", "false"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_table.routing_table", "created_at"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_table.routing_table", "updated_at"),

						// Routing tables
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "organization_id", testutil.ConvertConfigVariable(testConfigRoutingTableMin["organization_id"])),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "network_area_id", testutil.ConvertConfigVariable(testConfigRoutingTableMin["network_area_id"])),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "region", testutil.Region),
						// there will be always two routing tables because of the main routing table of the network area
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.#", "2"),

						// default routing table
						resource.TestCheckResourceAttrPair(
							"stackit_routing_table.routing_table", "items.0.routing_table_id",
							"data.stackit_routing_tables.routing_tables", "routing_table_id",
						),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.0.default", "true"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_tables.routing_tables", "items.0.created_at"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_tables.routing_tables", "items.0.updated_at"),

						// second routing table managed via terraform
						resource.TestCheckResourceAttrPair(
							"stackit_routing_table.routing_table", "items.1.routing_table_id",
							"data.stackit_routing_tables.routing_tables", "routing_table_id",
						),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.name", testutil.ConvertConfigVariable(testConfigRoutingTableMin["name"])),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.labels.%", "0"),
						resource.TestCheckNoResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.description"),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.system_routes", "true"),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.default", "false"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_tables.routing_tables", "items.1.created_at"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_tables.routing_tables", "items.1.updated_at"),
					),
				},
				// Import
				{
					ConfigVariables: testConfigRoutingTableMinUpdated,
					ResourceName:    "stackit_routing_table.routing_table",
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						r, ok := s.RootModule().Resources["stackit_routing_table.routing_table"]
						if !ok {
							return "", fmt.Errorf("couldn't find resource stackit_routing_table.routing_table")
						}
						region, ok := r.Primary.Attributes["region"]
						if !ok {
							return "", fmt.Errorf("couldn't find attribute region")
						}
						networkAreaId, ok := r.Primary.Attributes["network_area_id"]
						if !ok {
							return "", fmt.Errorf("couldn't find attribute network_area_id")
						}
						routingTableId, ok := r.Primary.Attributes["routing_table_id"]
						if !ok {
							return "", fmt.Errorf("couldn't find attribute routing_table_id")
						}
						return fmt.Sprintf("%s,%s,%s,%s", testutil.OrganizationId, region, networkAreaId, routingTableId), nil
					},
					ImportState:       true,
					ImportStateVerify: true,
				},
				// Update
				{
					ConfigVariables: testConfigRoutingTableMinUpdated,
					Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceRoutingTableMinConfig),
					Check: resource.ComposeAggregateTestCheckFunc(
						// Routing table
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "organization_id", testutil.ConvertConfigVariable(testConfigRoutingTableMinUpdated["organization_id"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "network_area_id", testutil.ConvertConfigVariable(testConfigRoutingTableMinUpdated["network_area_id"])),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "routing_table_id"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigRoutingTableMinUpdated["name"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "labels.%", "0"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "region", testutil.Region),
						resource.TestCheckNoResourceAttr("stackit_routing_table.routing_table", "description"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "system_routes", "true"),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "created_at"),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "updated_at"),
					),
				},
				// Deletion is done by the framework implicitly
			},
		})
	})

	t.Run("max", func(t *testing.T) {
		t.Logf("TestAccRoutingTableMax name: %s", testutil.ConvertConfigVariable(testConfigRoutingTableMax["name"]))
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
			CheckDestroy:             testAccCheckDestroy,
			Steps: []resource.TestStep{
				// Creation
				{
					ConfigVariables: testConfigRoutingTableMax,
					Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceRoutingTableMaxConfig),
					Check: resource.ComposeAggregateTestCheckFunc(
						// Routing table
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "organization_id", testutil.ConvertConfigVariable(testConfigRoutingTableMax["organization_id"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "network_area_id", testutil.ConvertConfigVariable(testConfigRoutingTableMax["network_area_id"])),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "routing_table_id"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigRoutingTableMax["name"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "labels.%", "1"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "labels.acc-test", testutil.ConvertConfigVariable(testConfigRoutingTableMax["label"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "region", testutil.ConvertConfigVariable(testConfigRoutingTableMax["region"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "description", testutil.ConvertConfigVariable(testConfigRoutingTableMax["description"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "system_routes", testutil.ConvertConfigVariable(testConfigRoutingTableMax["system_routes"])),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "created_at"),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "updated_at"),
					),
				},
				// Data sources
				{
					ConfigVariables: testConfigRoutingTableMax,
					Config: fmt.Sprintf(`
					%s
					%s
					
					# single routing table
					data "stackit_routing_table" "routing_table" {
						organization_id  = stackit_routing_table.routing_table.organization_id
						network_area_id  = stackit_routing_table.routing_table.network_area_id
						routing_table_id  = stackit_routing_table.routing_table.routing_table_id
					}
					
					# all routing tables in network area
					data "stackit_routing_tables" "routing_tables" {
						organization_id  = stackit_routing_table.routing_table.organization_id
						network_area_id  = stackit_routing_table.routing_table.network_area_id
					}
					`,
						testutil.IaaSProviderConfig(), resourceRoutingTableMaxConfig,
					),
					Check: resource.ComposeAggregateTestCheckFunc(
						// Routing table
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "organization_id", testutil.ConvertConfigVariable(testConfigRoutingTableMax["organization_id"])),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "network_area_id", testutil.ConvertConfigVariable(testConfigRoutingTableMax["network_area_id"])),
						resource.TestCheckResourceAttrPair(
							"stackit_routing_table.routing_table", "routing_table_id",
							"data.stackit_routing_table.routing_table", "routing_table_id",
						),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigRoutingTableMax["name"])),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "labels.%", "1"),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "labels.acc-test", testutil.ConvertConfigVariable(testConfigRoutingTableMax["label"])),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "region", testutil.ConvertConfigVariable(testConfigRoutingTableMax["region"])),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "description", testutil.ConvertConfigVariable(testConfigRoutingTableMax["description"])),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "system_routes", testutil.ConvertConfigVariable(testConfigRoutingTableMax["system_routes"])),
						resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "default", "false"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_table.routing_table", "created_at"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_table.routing_table", "updated_at"),

						// Routing tables
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "organization_id", testutil.ConvertConfigVariable(testConfigRoutingTableMax["organization_id"])),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "network_area_id", testutil.ConvertConfigVariable(testConfigRoutingTableMax["network_area_id"])),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "region", testutil.ConvertConfigVariable(testConfigRoutingTableMax["region"])),
						// there will be always two routing tables because of the main routing table of the network area
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.#", "2"),

						// default routing table
						resource.TestCheckResourceAttrPair(
							"stackit_routing_table.routing_table", "items.0.routing_table_id",
							"data.stackit_routing_tables.routing_tables", "routing_table_id",
						),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.0.default", "true"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_tables.routing_tables", "items.0.created_at"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_tables.routing_tables", "items.0.updated_at"),

						// second routing table managed via terraform
						resource.TestCheckResourceAttrPair(
							"stackit_routing_table.routing_table", "items.1.routing_table_id",
							"data.stackit_routing_tables.routing_tables", "routing_table_id",
						),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.name", testutil.ConvertConfigVariable(testConfigRoutingTableMax["name"])),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.labels.%", "1"),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.labels.acc-test", testutil.ConvertConfigVariable(testConfigRoutingTableMax["label"])),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.description", testutil.ConvertConfigVariable(testConfigRoutingTableMax["description"])),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.system_routes", testutil.ConvertConfigVariable(testConfigRoutingTableMax["system_routes"])),
						resource.TestCheckResourceAttr("data.stackit_routing_tables.routing_tables", "items.1.default", "false"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_tables.routing_tables", "items.1.created_at"),
						resource.TestCheckResourceAttrSet("data.stackit_routing_tables.routing_tables", "items.1.updated_at"),
					),
				},
				// Import
				{
					ConfigVariables: testConfigRoutingTableMaxUpdated,
					ResourceName:    "stackit_routing_table.routing_table",
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						r, ok := s.RootModule().Resources["stackit_routing_table.routing_table"]
						if !ok {
							return "", fmt.Errorf("couldn't find resource stackit_routing_table.routing_table")
						}
						region, ok := r.Primary.Attributes["region"]
						if !ok {
							return "", fmt.Errorf("couldn't find attribute region")
						}
						networkAreaId, ok := r.Primary.Attributes["network_area_id"]
						if !ok {
							return "", fmt.Errorf("couldn't find attribute network_area_id")
						}
						routingTableId, ok := r.Primary.Attributes["routing_table_id"]
						if !ok {
							return "", fmt.Errorf("couldn't find attribute routing_table_id")
						}
						return fmt.Sprintf("%s,%s,%s,%s", testutil.OrganizationId, region, networkAreaId, routingTableId), nil
					},
					ImportState:       true,
					ImportStateVerify: true,
				},
				// Update
				{
					ConfigVariables: testConfigRoutingTableMaxUpdated,
					Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceRoutingTableMaxConfig),
					Check: resource.ComposeAggregateTestCheckFunc(
						// Routing table
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "organization_id", testutil.ConvertConfigVariable(testConfigRoutingTableMaxUpdated["organization_id"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "network_area_id", testutil.ConvertConfigVariable(testConfigRoutingTableMaxUpdated["network_area_id"])),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "routing_table_id"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigRoutingTableMaxUpdated["name"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "labels.%", "1"),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "labels.acc-test", testutil.ConvertConfigVariable(testConfigRoutingTableMaxUpdated["label"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "region", testutil.ConvertConfigVariable(testConfigRoutingTableMaxUpdated["region"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "description", testutil.ConvertConfigVariable(testConfigRoutingTableMaxUpdated["description"])),
						resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "system_routes", testutil.ConvertConfigVariable(testConfigRoutingTableMaxUpdated["system_routes"])),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "created_at"),
						resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "updated_at"),
					),
				},
				// Deletion is done by the framework implicitly
			},
		})
	})
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAccCheckRoutingTableDestroy,
		testAccCheckRoutingTableRouteDestroy,
	}
	var errs []error

	wg := sync.WaitGroup{}
	wg.Add(len(checkFunctions))

	for _, f := range checkFunctions {
		go func() {
			err := f(s)
			if err != nil {
				errs = append(errs, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

func testAccCheckRoutingTableDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaasalpha.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaasalpha.NewAPIClient()
	} else {
		client, err = iaasalpha.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error
	// routing tables
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_routing_table" {
			continue
		}
		routingTableId := strings.Split(rs.Primary.ID, core.Separator)[3]
		region := strings.Split(rs.Primary.ID, core.Separator)[1]
		err := client.DeleteRoutingTableFromAreaExecute(ctx, testutil.OrganizationId, testNetworkAreaId, region, routingTableId)
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}
				if oapiErr.StatusCode == http.StatusInternalServerError {
					// TODO
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger routing table deletion %q: %w", routingTableId, err))
		}
	}

	return errors.Join(errs...)
}

func testAccCheckRoutingTableRouteDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaasalpha.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaasalpha.NewAPIClient()
	} else {
		client, err = iaasalpha.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error
	// routes
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_routing_table_route" {
			continue
		}
		routingTableRouteId := strings.Split(rs.Primary.ID, core.Separator)[4]
		routingTableId := strings.Split(rs.Primary.ID, core.Separator)[3]
		region := strings.Split(rs.Primary.ID, core.Separator)[1]
		err := client.DeleteRouteFromRoutingTableExecute(ctx, testutil.OrganizationId, testNetworkAreaId, region, routingTableId, routingTableRouteId)
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}
				if oapiErr.StatusCode == http.StatusInternalServerError {
					// TODO
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger routing table route deletion %q: %w", routingTableId, err))
		}
	}

	return errors.Join(errs...)
}
