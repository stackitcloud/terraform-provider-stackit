package iaasalpha

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	// VPC

	//go:embed testdata/resource-vpc-min.tf
	resourceVpcMinConfig string

	//go:embed testdata/resource-vpc-max.tf
	resourceVpcMaxConfig string

	// VPC Routing Table

	//go:embed testdata/resource-vpc-routingtable-min.tf
	resourceVpcRoutingTableMinConfig string

	//go:embed testdata/resource-vpc-routingtable-max.tf
	resourceVpcRoutingTableMaxConfig string

	// VPC Region

	//go:embed testdata/resource-vpc-region-min.tf
	resourceVpcRegionMinConfig string

	// no max test, currently there are no optional attributes

	// VPC Routing Table Static Route

	//go:embed testdata/resource-vpc-routingtable-static-route-min.tf
	resourceVpcRoutingTableStaticRouteMinConfig string

	//go:embed testdata/resource-vpc-routingtable-static-route-max.tf
	resourceVpcRoutingTableStaticRouteMaxConfig string

	// VPC Network Range

	//go:embed testdata/resource-vpc-network-range-min.tf
	resourceVpcNetworkRangeMinConfig string

	//go:embed testdata/resource-vpc-network-range-max.tf
	resourceVpcNetworkRangeMaxConfig string
)

// VPC - MIN

var testConfigVPCVarsMin = config.Variables{
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"description":         config.StringVariable(""),
	"project_name": config.StringVariable(fmt.Sprintf(
		"tf-acc-vpc-min-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),

	"name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),
}

// VPC - MAX

var testConfigVPCVarsMax = config.Variables{
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"project_name": config.StringVariable(fmt.Sprintf(
		"tf-acc-vpc-max-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),

	"name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),
	"description": config.StringVariable("terraform acceptance test"),
	"label_key":   config.StringVariable("stage"),
	"label_value": config.StringVariable("qa"),
}

var testConfigVPCVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigVPCVarsMax)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf(
		"%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"]),
	))
	updatedConfig["description"] = config.StringVariable("terraform acceptance test updated")
	updatedConfig["label_key"] = config.StringVariable("stage-updated")
	updatedConfig["label_value"] = config.StringVariable("qa-updated")
	return updatedConfig
}()

// VPC Routing Table - MIN

var testConfigVPCRoutingTableVarsMin = config.Variables{
	"project_name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),
}

var testConfigVPCRoutingTableVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigVPCRoutingTableVarsMin)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf(
		"%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"]),
	))
	return updatedConfig
}()

// VPC Routing Table - MAX

var testConfigVPCRoutingTableVarsMax = config.Variables{
	"project_name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),
	"description":    config.StringVariable("terraform acceptance test"),
	"system_routes":  config.BoolVariable(false),
	"dynamic_routes": config.BoolVariable(false),
	"region":         config.StringVariable(testutil.Region),
	"label_key":      config.StringVariable("stage"),
	"label_value":    config.StringVariable("qa"),
}

var testConfigVPCRoutingTableVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigVPCRoutingTableVarsMax)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf(
		"%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"]),
	))
	updatedConfig["description"] = config.StringVariable("terraform acceptance test updated")
	updatedConfig["label_key"] = config.StringVariable("stage-updated")
	updatedConfig["label_value"] = config.StringVariable("qa-updated")
	updatedConfig["system_routes"] = config.BoolVariable(true)
	updatedConfig["dynamic_routes"] = config.BoolVariable(true)
	return updatedConfig
}()

// VPC Routing Table Static Route - MIN

var testConfigVPCRoutingTableStaticRouteVarsMin = config.Variables{
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"project_name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),
	"destination_type":  config.StringVariable("cidrv4"),
	"destination_value": config.StringVariable("10.0.0.0/8"),
	"nexthop_type":      config.StringVariable("ipv4"),
	"nexthop_value":     config.StringVariable("10.0.0.1"),
}

var testConfigVPCRoutingTableStaticRouteVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigVPCRoutingTableStaticRouteVarsMin)
	// currently only cidrv4 supported
	updatedConfig["nexthop_type"] = config.StringVariable("blackhole")
	updatedConfig["nexthop_value"] = nil
	return updatedConfig
}()

// VPC Routing Table Static Route - MAX

var testConfigVPCRoutingTableStaticRouteVarsMax = config.Variables{
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"project_name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),
	"destination_type":  config.StringVariable("cidrv4"),
	"destination_value": config.StringVariable("10.0.0.0/8"),
	"nexthop_type":      config.StringVariable("ipv4"),
	"nexthop_value":     config.StringVariable("10.0.0.1"),
	"labels": config.MapVariable(map[string]config.Variable{
		"key1": config.StringVariable("value1"),
	}),
}

var testConfigVPCRoutingTableStaticRouteVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigVPCRoutingTableStaticRouteVarsMax)
	// currently only cidrv4 supported
	updatedConfig["nexthop_type"] = config.StringVariable("blackhole")
	updatedConfig["nexthop_value"] = nil
	updatedConfig["labels"] = config.MapVariable(map[string]config.Variable{
		"key1": config.StringVariable("value1-updated"),
		"key2": config.StringVariable("value2"),
	})
	return updatedConfig
}()

// VPC Region - MIN

var testConfigVPCRegionVarsMin = config.Variables{
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),
}

// VPC Network range - MIN

var testConfigVPCNetworkRangeVarsMin = config.Variables{
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"testing_setup_name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),

	"ip_version":  config.StringVariable(string(iaas.NETWORKRANGEIPV4REQUESTIPVERSION_IPV4)),
	"prefix":      config.StringVariable("192.168.1.0/24"),
	"description": config.StringVariable("network range acc test"),
}

var testConfigVPCNetworkRangeVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigVPCNetworkRangeVarsMin)
	updatedConfig["description"] = config.StringVariable("network range acc test update")
	return updatedConfig
}()

// VPC Network range - MAX

var testConfigVPCNetworkRangeVarsMax = config.Variables{
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"testing_setup_name": config.StringVariable(fmt.Sprintf(
		"tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
	)),

	"ip_version":            config.StringVariable(string(iaas.NETWORKRANGEIPV4REQUESTIPVERSION_IPV4)),
	"prefix":                config.StringVariable("192.168.1.0/24"),
	"description":           config.StringVariable("network range max acc test"),
	"default_prefix_length": config.IntegerVariable(26),
	"max_prefix_length":     config.IntegerVariable(27),
	"min_prefix_length":     config.IntegerVariable(17),
	"nameserver":            config.StringVariable("1.1.1.1"),
	"region":                config.StringVariable(testutil.Region),
	"label_key":             config.StringVariable("acc"),
	"label_value":           config.StringVariable("test"),
}

var testConfigVPCNetworkRangeVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigVPCNetworkRangeVarsMax)
	updatedConfig["description"] = config.StringVariable("network range acc test update")
	updatedConfig["default_prefix_length"] = config.IntegerVariable(27)
	updatedConfig["max_prefix_length"] = config.IntegerVariable(27)
	updatedConfig["min_prefix_length"] = config.IntegerVariable(20)
	updatedConfig["nameserver"] = config.StringVariable("8.8.8.8")
	updatedConfig["label_key"] = config.StringVariable("acc-update")
	updatedConfig["label_value"] = config.StringVariable("test-update")
	return updatedConfig
}()

func TestAccVPCMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVPCVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", "name", testutil.ConvertConfigVariable(testConfigVPCVarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_vpc.vpc", "vpc_id"),
					resource.TestCheckResourceAttrSet("stackit_vpc.vpc", "id"),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", "description", testutil.ConvertConfigVariable(testConfigVPCVarsMin["description"])),
					resource.TestCheckNoResourceAttr("stackit_vpc.vpc", "labels"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVPCVarsMin,
				Config: fmt.Sprintf(`
				%s
				%s

				data "stackit_vpc" "vpc" {
					project_id = stackit_vpc.vpc.project_id
					vpc_id     = stackit_vpc.vpc.vpc_id
				}
				`, testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.stackit_vpc.vpc", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttr("data.stackit_vpc.vpc", "name", testutil.ConvertConfigVariable(testConfigVPCVarsMin["name"])),
					resource.TestCheckResourceAttrSet("data.stackit_vpc.vpc", "vpc_id"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc.vpc", "id"),
					resource.TestCheckResourceAttr("data.stackit_vpc.vpc", "description", testutil.ConvertConfigVariable(testConfigVPCVarsMin["description"])),
					resource.TestCheckNoResourceAttr("data.stackit_vpc.vpc", "labels"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVPCVarsMin,
				ResourceName:    "stackit_vpc.vpc",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpc.vpc"]
					if !ok {
						return "", errors.New("couldn't find resource stackit_vpc.vpc")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", errors.New("couldn't find attribute project_id")
					}
					vpcId, ok := r.Primary.Attributes["vpc_id"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					return fmt.Sprintf("%s,%s", projectId, vpcId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// In the minimal config it's not possible to do an update, because only the project_id is set.
		},
	})
}

func TestAccVPCMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVPCVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrSet("stackit_vpc.vpc", "vpc_id"),
					resource.TestCheckResourceAttrSet("stackit_vpc.vpc", "id"),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", "name", testutil.ConvertConfigVariable(testConfigVPCVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", "description", testutil.ConvertConfigVariable(testConfigVPCVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", fmt.Sprintf("labels.%s", testutil.ConvertConfigVariable(testConfigVPCVarsMax["label_key"])), testutil.ConvertConfigVariable(testConfigVPCVarsMax["label_value"])),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", "labels.%", "1"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVPCVarsMax,
				Config: fmt.Sprintf(`
				%s
				%s

				data "stackit_vpc" "vpc" {
					project_id = stackit_vpc.vpc.project_id
					vpc_id     = stackit_vpc.vpc.vpc_id
				}
				`, testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.stackit_vpc.vpc", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_vpc.vpc", "vpc_id"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc.vpc", "id"),
					resource.TestCheckResourceAttr("data.stackit_vpc.vpc", "name", testutil.ConvertConfigVariable(testConfigVPCVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_vpc.vpc", "description", testutil.ConvertConfigVariable(testConfigVPCVarsMax["description"])),
					resource.TestCheckResourceAttr("data.stackit_vpc.vpc", fmt.Sprintf("labels.%s", testutil.ConvertConfigVariable(testConfigVPCVarsMax["label_key"])), testutil.ConvertConfigVariable(testConfigVPCVarsMax["label_value"])),
					resource.TestCheckResourceAttr("data.stackit_vpc.vpc", "labels.%", "1"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVPCVarsMax,
				ResourceName:    "stackit_vpc.vpc",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpc.vpc"]
					if !ok {
						return "", errors.New("couldn't find resource stackit_vpc.vpc")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", errors.New("couldn't find attribute project_id")
					}
					vpcId, ok := r.Primary.Attributes["vpc_id"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					return fmt.Sprintf("%s,%s", projectId, vpcId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVPCVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrSet("stackit_vpc.vpc", "vpc_id"),
					resource.TestCheckResourceAttrSet("stackit_vpc.vpc", "id"),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", "name", testutil.ConvertConfigVariable(testConfigVPCVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", "description", testutil.ConvertConfigVariable(testConfigVPCVarsMaxUpdated["description"])),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", fmt.Sprintf("labels.%s", testutil.ConvertConfigVariable(testConfigVPCVarsMaxUpdated["label_key"])), testutil.ConvertConfigVariable(testConfigVPCVarsMaxUpdated["label_value"])),
					resource.TestCheckResourceAttr("stackit_vpc.vpc", "labels.%", "1"),
				),
			},
		},
	})
}

func TestAccVPCRoutingTableMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVPCRoutingTableVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "routing_table_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"stackit_vpc_routing_table.routing_table", "vpc_id",
					),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMin["name"])),
					resource.TestCheckNoResourceAttr("stackit_vpc_routing_table.routing_table", "description"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "labels.%", "0"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "region"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "dynamic_routes", "true"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "system_routes", "true"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVPCRoutingTableVarsMin,
				Config: fmt.Sprintf(`
				%s
				%s

				data "stackit_vpc_routing_table" "routing_table" {
					project_id       = stackit_vpc.vpc.project_id
					vpc_id           = stackit_vpc.vpc.vpc_id
					routing_table_id = stackit_vpc_routing_table.routing_table.routing_table_id
				}
				`, testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table.routing_table", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table.routing_table", "routing_table_id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_vpc_routing_table.routing_table", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"data.stackit_vpc_routing_table.routing_table", "vpc_id",
					),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMin["name"])),
					resource.TestCheckNoResourceAttr("data.stackit_vpc_routing_table.routing_table", "description"),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "labels.%", "0"),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "region", testutil.Region),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "dynamic_routes", "true"),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "system_routes", "true"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVPCRoutingTableVarsMin,
				ResourceName:    "stackit_vpc_routing_table.routing_table",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpc_routing_table.routing_table"]
					if !ok {
						return "", errors.New("couldn't find resource stackit_vpc.vpc")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", errors.New("couldn't find attribute project_id")
					}
					vpcId, ok := r.Primary.Attributes["vpc_id"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					routingTableId, ok := r.Primary.Attributes["routing_table_id"]
					if !ok {
						return "", errors.New("couldn't find attribute routing_table_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", projectId, vpcId, region, routingTableId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVPCRoutingTableVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "routing_table_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"stackit_vpc_routing_table.routing_table", "vpc_id",
					),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMinUpdated["name"])),
					resource.TestCheckNoResourceAttr("stackit_vpc_routing_table.routing_table", "description"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "labels.%", "0"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "region"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "dynamic_routes", "true"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "system_routes", "true"),
				),
			},
		},
	})
}

func TestAccVPCRoutingTableMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVPCRoutingTableVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "routing_table_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"stackit_vpc_routing_table.routing_table", "vpc_id",
					),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "description", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "region", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["region"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "dynamic_routes", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["dynamic_routes"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "system_routes", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["system_routes"])),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVPCRoutingTableVarsMax,
				Config: fmt.Sprintf(`
				%s
				%s

				data "stackit_vpc_routing_table" "routing_table" {
					project_id       = stackit_vpc.vpc.project_id
					vpc_id           = stackit_vpc.vpc.vpc_id
					routing_table_id = stackit_vpc_routing_table.routing_table.routing_table_id
				}
				`, testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table.routing_table", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table.routing_table", "routing_table_id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_vpc_routing_table.routing_table", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"data.stackit_vpc_routing_table.routing_table", "vpc_id",
					),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "description", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["description"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "region", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["region"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "dynamic_routes", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["dynamic_routes"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table.routing_table", "system_routes", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMax["system_routes"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVPCRoutingTableVarsMax,
				ResourceName:    "stackit_vpc_routing_table.routing_table",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpc_routing_table.routing_table"]
					if !ok {
						return "", errors.New("couldn't find resource stackit_vpc.vpc")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", errors.New("couldn't find attribute project_id")
					}
					vpcId, ok := r.Primary.Attributes["vpc_id"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					routingTableId, ok := r.Primary.Attributes["routing_table_id"]
					if !ok {
						return "", errors.New("couldn't find attribute routing_table_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", projectId, vpcId, region, routingTableId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVPCRoutingTableVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table.routing_table", "routing_table_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"stackit_vpc_routing_table.routing_table", "vpc_id",
					),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "description", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMaxUpdated["description"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "region", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMaxUpdated["region"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "dynamic_routes", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMaxUpdated["dynamic_routes"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table.routing_table", "system_routes", testutil.ConvertConfigVariable(testConfigVPCRoutingTableVarsMaxUpdated["system_routes"])),
				),
			},
		},
	})
}

func TestAccVPCRegionMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigVPCRegionVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRegionMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_region.region", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_region.region", "id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_region.region", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"stackit_vpc_region.region", "vpc_id",
					),
					resource.TestCheckResourceAttrSet("stackit_vpc_region.region", "region"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigVPCRegionVarsMin,
				Config: fmt.Sprintf(`
				%s
				%s

				data "stackit_vpc_region" "region" {
					project_id = stackit_vpc.vpc.project_id
					vpc_id = stackit_vpc.vpc.vpc_id
					region = "eu01"
				}
				`, testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRegionMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_region.region", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_vpc_region.region", "id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_vpc_region.region", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"data.stackit_vpc_region.region", "vpc_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_region.region", "region"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVPCRegionVarsMin,
				ResourceName:    "stackit_vpc_region.region",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpc_region.region"]
					if !ok {
						return "", errors.New("couldn't find resource stackit_vpc_region.region")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", errors.New("couldn't find attribute project_id")
					}
					vpcId, ok := r.Primary.Attributes["vpc_id"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", errors.New("couldn't find attribute region")
					}
					return fmt.Sprintf("%s,%s,%s", projectId, vpcId, region), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// No update for min config
		},
	})
}

func TestAccVPCRoutingTableStaticRouteMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVPCRoutingTableStaticRouteVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableStaticRouteMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table_static_route.static_route", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "route_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table_static_route.static_route", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"stackit_vpc_routing_table_static_route.static_route", "vpc_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "routing_table_id",
						"stackit_vpc_routing_table_static_route.static_route", "routing_table_id",
					),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "destination.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMin["destination_type"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "destination.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMin["destination_value"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "nexthop.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMin["nexthop_type"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "nexthop.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMin["nexthop_value"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "labels.%", "0"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "region"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVPCRoutingTableStaticRouteVarsMin,
				Config: fmt.Sprintf(`
				%s
				%s

				data "stackit_vpc_routing_table_static_route" "static_route" {
					project_id       = stackit_vpc.vpc.project_id
					vpc_id           = stackit_vpc.vpc.vpc_id
					routing_table_id = stackit_vpc_routing_table.routing_table.routing_table_id
					route_id         = stackit_vpc_routing_table_static_route.static_route.route_id
				}
				`, testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableStaticRouteMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table_static_route.static_route", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table_static_route.static_route", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table_static_route.static_route", "route_id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_vpc_routing_table_static_route.static_route", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"data.stackit_vpc_routing_table_static_route.static_route", "vpc_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "routing_table_id",
						"data.stackit_vpc_routing_table_static_route.static_route", "routing_table_id",
					),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "destination.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMin["destination_type"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "destination.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMin["destination_value"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "nexthop.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMin["nexthop_type"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "nexthop.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMin["nexthop_value"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "labels.%", "0"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table_static_route.static_route", "region"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVPCRoutingTableStaticRouteVarsMin,
				ResourceName:    "stackit_vpc_routing_table_static_route.static_route",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpc_routing_table_static_route.static_route"]
					if !ok {
						return "", errors.New("couldn't find resource stackit_vpc_routing_table_static_route.static_route")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", errors.New("couldn't find attribute project_id")
					}
					vpcId, ok := r.Primary.Attributes["vpc_id"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", errors.New("couldn't find attribute region")
					}
					routingTableId, ok := r.Primary.Attributes["routing_table_id"]
					if !ok {
						return "", errors.New("couldn't find attribute routing_table_id")
					}
					routeId, ok := r.Primary.Attributes["route_id"]
					if !ok {
						return "", errors.New("couldn't find attribute route_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s,%s", projectId, vpcId, region, routingTableId, routeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVPCRoutingTableStaticRouteVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableStaticRouteMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table_static_route.static_route", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "route_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table_static_route.static_route", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"stackit_vpc_routing_table_static_route.static_route", "vpc_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "routing_table_id",
						"stackit_vpc_routing_table_static_route.static_route", "routing_table_id",
					),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "destination.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMinUpdated["destination_type"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "destination.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMinUpdated["destination_value"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "nexthop.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMinUpdated["nexthop_type"])),
					resource.TestCheckNoResourceAttr("stackit_vpc_routing_table_static_route.static_route", "nexthop.value"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "labels.%", "0"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "region"),
				),
			},
		},
	})
}

func TestAccVPCRoutingTableStaticRouteMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVPCRoutingTableStaticRouteVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableStaticRouteMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table_static_route.static_route", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "route_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table_static_route.static_route", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"stackit_vpc_routing_table_static_route.static_route", "vpc_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "routing_table_id",
						"stackit_vpc_routing_table_static_route.static_route", "routing_table_id",
					),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "destination.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMax["destination_type"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "destination.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMax["destination_value"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "nexthop.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMax["nexthop_type"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "nexthop.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMax["nexthop_value"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "labels.key1", "value1"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "region"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVPCRoutingTableStaticRouteVarsMax,
				Config: fmt.Sprintf(`
				%s
				%s

				data "stackit_vpc_routing_table_static_route" "static_route" {
					project_id       = stackit_vpc.vpc.project_id
					vpc_id           = stackit_vpc.vpc.vpc_id
					routing_table_id = stackit_vpc_routing_table.routing_table.routing_table_id
					route_id         = stackit_vpc_routing_table_static_route.static_route.route_id
				}
				`, testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableStaticRouteMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table_static_route.static_route", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table_static_route.static_route", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table_static_route.static_route", "route_id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_vpc_routing_table_static_route.static_route", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"data.stackit_vpc_routing_table_static_route.static_route", "vpc_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "routing_table_id",
						"data.stackit_vpc_routing_table_static_route.static_route", "routing_table_id",
					),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "destination.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMax["destination_type"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "destination.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMax["destination_value"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "nexthop.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMax["nexthop_type"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "nexthop.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMax["nexthop_value"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_vpc_routing_table_static_route.static_route", "labels.key1", "value1"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_routing_table_static_route.static_route", "region"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVPCRoutingTableStaticRouteVarsMax,
				ResourceName:    "stackit_vpc_routing_table_static_route.static_route",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpc_routing_table_static_route.static_route"]
					if !ok {
						return "", errors.New("couldn't find resource stackit_vpc_routing_table_static_route.static_route")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", errors.New("couldn't find attribute project_id")
					}
					vpcId, ok := r.Primary.Attributes["vpc_id"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", errors.New("couldn't find attribute region")
					}
					routingTableId, ok := r.Primary.Attributes["routing_table_id"]
					if !ok {
						return "", errors.New("couldn't find attribute routing_table_id")
					}
					routeId, ok := r.Primary.Attributes["route_id"]
					if !ok {
						return "", errors.New("couldn't find attribute route_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s,%s", projectId, vpcId, region, routingTableId, routeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVPCRoutingTableStaticRouteVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcRoutingTableStaticRouteMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table.routing_table", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_routing_table_static_route.static_route", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "route_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table_static_route.static_route", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc.vpc", "vpc_id",
						"stackit_vpc_routing_table_static_route.static_route", "vpc_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_routing_table.routing_table", "routing_table_id",
						"stackit_vpc_routing_table_static_route.static_route", "routing_table_id",
					),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "destination.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMaxUpdated["destination_type"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "destination.value", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMaxUpdated["destination_value"])),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "nexthop.type", testutil.ConvertConfigVariable(testConfigVPCRoutingTableStaticRouteVarsMaxUpdated["nexthop_type"])),
					resource.TestCheckNoResourceAttr("stackit_vpc_routing_table_static_route.static_route", "nexthop.value"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "labels.%", "2"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "labels.key1", "value1-updated"),
					resource.TestCheckResourceAttr("stackit_vpc_routing_table_static_route.static_route", "labels.key2", "value2"),
					resource.TestCheckResourceAttrSet("stackit_vpc_routing_table_static_route.static_route", "region"),
				),
			},
		},
	})
}

func TestAccVPCNetworkRangeMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigVPCNetworkRangeVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcNetworkRangeMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_region.region", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_network_range.network_range", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_network_range.network_range", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "network_range_id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "region"),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "default_prefix_length"),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "max_prefix_length"),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "min_prefix_length"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "nameservers.#", "0"),
					resource.TestCheckNoResourceAttr("stackit_vpc_network_range.network_range", "labels"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "ip_version", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMin["ip_version"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "prefix", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMin["prefix"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "description", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMin["description"])),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigVPCNetworkRangeVarsMin,
				Config: fmt.Sprintf(`
				%s
				%s
			
				data "stackit_vpc_network_range" "network_range" {
					project_id       = stackit_vpc.vpc.project_id
					vpc_id           = stackit_vpc.vpc.vpc_id
					network_range_id = stackit_vpc_network_range.network_range.network_range_id
				}
				`, testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcNetworkRangeMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_region.region", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_network_range.network_range", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_vpc_network_range.network_range", "id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_vpc_network_range.network_range", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_network_range.network_range", "network_range_id"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_network_range.network_range", "region"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_network_range.network_range", "default_prefix_length"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_network_range.network_range", "max_prefix_length"),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_network_range.network_range", "min_prefix_length"),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "nameservers.#", "0"),
					resource.TestCheckNoResourceAttr("data.stackit_vpc_network_range.network_range", "labels"),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "ip_version", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMin["ip_version"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "prefix", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMin["prefix"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "description", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMin["description"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVPCNetworkRangeVarsMin,
				ResourceName:    "stackit_vpc_network_range.network_range",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpc_network_range.network_range"]
					if !ok {
						return "", errors.New("couldn't find resource stackit_vpc_network_range.network_range")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", errors.New("couldn't find attribute project_id")
					}
					vpcId, ok := r.Primary.Attributes["vpc_id"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", errors.New("couldn't find attribute region")
					}
					networkRangeId, ok := r.Primary.Attributes["network_range_id"]
					if !ok {
						return "", errors.New("couldn't find attribute network_range_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", projectId, vpcId, region, networkRangeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVPCNetworkRangeVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcNetworkRangeMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_region.region", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_network_range.network_range", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_network_range.network_range", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "network_range_id"),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "region"),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "default_prefix_length"),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "max_prefix_length"),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "min_prefix_length"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "nameservers.#", "0"),
					resource.TestCheckNoResourceAttr("stackit_vpc_network_range.network_range", "labels"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "ip_version", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMinUpdated["ip_version"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "prefix", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMinUpdated["prefix"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "description", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMinUpdated["description"])),
				),
			},
		},
	})
}

func TestAccVPCNetworkRangeMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigVPCNetworkRangeVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcNetworkRangeMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_region.region", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_vpc_network_range.network_range", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_network_range.network_range", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "network_range_id"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "region", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["region"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "default_prefix_length", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["default_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "max_prefix_length", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["max_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "min_prefix_length", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["min_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "nameservers.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "nameservers.0", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["nameserver"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", fmt.Sprintf("labels.%s", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["label_key"])), testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["label_value"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "ip_version", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["ip_version"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "prefix", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["prefix"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "description", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["description"])),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigVPCNetworkRangeVarsMax,
				Config: fmt.Sprintf(`
				%s
				%s
			
				data "stackit_vpc_network_range" "network_range" {
					project_id       = stackit_vpc.vpc.project_id
					vpc_id           = stackit_vpc.vpc.vpc_id
					network_range_id = stackit_vpc_network_range.network_range.network_range_id
				}
				`, testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcNetworkRangeMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_region.region", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_network_range.network_range", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_vpc_network_range.network_range", "id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_vpc_network_range.network_range", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_vpc_network_range.network_range", "network_range_id"),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "region", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["region"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "default_prefix_length", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["default_prefix_length"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "max_prefix_length", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["max_prefix_length"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "min_prefix_length", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["min_prefix_length"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "nameservers.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "nameservers.0", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["nameserver"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", fmt.Sprintf("labels.%s", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["label_key"])), testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["label_value"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "ip_version", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["ip_version"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "prefix", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["prefix"])),
					resource.TestCheckResourceAttr("data.stackit_vpc_network_range.network_range", "description", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMax["description"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVPCNetworkRangeVarsMax,
				ResourceName:    "stackit_vpc_network_range.network_range",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_vpc_network_range.network_range"]
					if !ok {
						return "", errors.New("couldn't find resource stackit_vpc_network_range.network_range")
					}
					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", errors.New("couldn't find attribute project_id")
					}
					vpcId, ok := r.Primary.Attributes["vpc_id"]
					if !ok {
						return "", errors.New("couldn't find attribute vpc_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", errors.New("couldn't find attribute region")
					}
					networkRangeId, ok := r.Primary.Attributes["network_range_id"]
					if !ok {
						return "", errors.New("couldn't find attribute network_range_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", projectId, vpcId, region, networkRangeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVPCNetworkRangeVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildProviderConfig(), resourceVpcNetworkRangeMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_vpc.vpc", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_region.region", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_vpc_network_range.network_range", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "id"),
					resource.TestCheckResourceAttrPair(
						"stackit_vpc_network_range.network_range", "project_id",
						"stackit_resourcemanager_project.project", "project_id",
					),
					resource.TestCheckResourceAttrSet("stackit_vpc_network_range.network_range", "network_range_id"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "region", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["region"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "default_prefix_length", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["default_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "max_prefix_length", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["max_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "min_prefix_length", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["min_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "nameservers.#", "1"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "nameservers.0", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["nameserver"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", fmt.Sprintf("labels.%s", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["label_key"])), testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["label_value"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "ip_version", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["ip_version"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "prefix", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["prefix"])),
					resource.TestCheckResourceAttr("stackit_vpc_network_range.network_range", "description", testutil.ConvertConfigVariable(testConfigVPCNetworkRangeVarsMaxUpdated["description"])),
				),
			},
		},
	})
}

func testCheckDestroy(s *terraform.State) error {
	checkDestroyFuncs := []resource.TestCheckFunc{
		testVpcNetworkRangeDestroy,
		testVpcRoutingTableStaticRouteDestroy,
		testVpcRoutingTableCheckDestroy,
		testVpcRegionDestroy,
		testVpcCheckDestroy,
	}

	var errs []error
	for _, checkDestroyFunc := range checkDestroyFuncs {
		err := checkDestroyFunc(s)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func testVpcRoutingTableCheckDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := iaas.NewAPIClient(testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildClientOptions(testutil.IaaSCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type routingTableIds struct {
		projectId      string
		vpcId          string
		region         string
		routingTableId string
	}

	rtToDestroy := []routingTableIds{}
	var errs []error
	// vpc
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_vpc_routing_table" {
			continue
		}
		projectId, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", rs.Primary))
			continue
		}
		vpcId, ok := rs.Primary.Attributes["vpc_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no vpc_id found in %s", rs.Primary))
			continue
		}
		region, ok := rs.Primary.Attributes["region"]
		if !ok {
			errs = append(errs, fmt.Errorf("no region found in %s", rs.Primary))
		}
		routingTableId, ok := rs.Primary.Attributes["routing_table_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no routing_table_id found in %s", rs.Primary))
		}
		rtToDestroy = append(rtToDestroy, routingTableIds{
			projectId:      projectId,
			vpcId:          vpcId,
			region:         region,
			routingTableId: routingTableId,
		})
	}

	for _, rt := range rtToDestroy {
		_, err := client.DefaultAPI.GetVPCRoutingTable(ctx, rt.projectId, rt.vpcId, rt.region, rt.routingTableId).Execute()
		if err == nil {
			err := client.DefaultAPI.DeleteVPCRoutingTable(ctx, rt.projectId, rt.vpcId, rt.region, rt.routingTableId).Execute()
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting rt with ID %q in project %q, vpc %q, region %q : %w", rt.routingTableId, rt.projectId, rt.vpcId, rt.region, err))
			}
			continue
		}

		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if oapiErr.StatusCode == 404 || oapiErr.StatusCode == 403 {
				continue
			}
		}
		errs = append(errs, fmt.Errorf("deleting rt: %w", err))
	}

	return errors.Join(errs...)
}

func testVpcCheckDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := iaas.NewAPIClient(testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildClientOptions(testutil.IaaSCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type vpcIds struct {
		projectID string
		vpcID     string
	}

	vpcsToDestroy := []vpcIds{}
	var errs []error
	// vpc
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_vpc" {
			continue
		}
		projectId, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", rs.Primary))
			continue
		}
		vpcId, ok := rs.Primary.Attributes["vpc_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no vpc_id found in %s", rs.Primary))
			continue
		}
		vpcsToDestroy = append(vpcsToDestroy, vpcIds{
			projectID: projectId,
			vpcID:     vpcId,
		})
	}

	for _, vpc := range vpcsToDestroy {
		_, err := client.DefaultAPI.GetVPC(ctx, vpc.projectID, vpc.vpcID).Execute()
		if err == nil {
			err := client.DefaultAPI.DeleteVPC(ctx, vpc.projectID, vpc.vpcID).Execute()
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting vpc with ID %q: %w", vpc.vpcID, err))
			}
			continue
		}
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if oapiErr.StatusCode == 404 || oapiErr.StatusCode == 403 {
				continue
			}
		}
		errs = append(errs, fmt.Errorf("deleting vpc: %w", err))
	}

	return errors.Join(errs...)
}

func testVpcRegionDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := iaas.NewAPIClient(testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildClientOptions(testutil.IaaSCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	type regionIds struct {
		projectId, vpcId, region string
	}

	var toDestroy []regionIds
	var errs []error
	for _, r := range s.RootModule().Resources {
		if r.Type != "stackit_vpc_region" {
			continue
		}
		projectId, ok := r.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", r.Primary))
			continue
		}
		vpcId, ok := r.Primary.Attributes["vpc_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no vpc_id found in %s", r.Primary))
			continue
		}
		region, ok := r.Primary.Attributes["region"]
		if !ok {
			errs = append(errs, fmt.Errorf("no region found in %s", r.Primary))
			continue
		}
		toDestroy = append(toDestroy, regionIds{
			projectId: projectId,
			vpcId:     vpcId,
			region:    region,
		})
	}
	for _, id := range toDestroy {
		_, err := client.DefaultAPI.GetVPCRegion(ctx, id.projectId, id.vpcId, id.region).Execute()
		if err == nil {
			err := client.DefaultAPI.DeleteVPCRegion(ctx, id.projectId, id.vpcId, id.region).Execute()
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting region with ID %q in project %q, vpc %q: %w", id.region, id.projectId, id.vpcId, err))
			}
			continue
		}
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if oapiErr.StatusCode == 404 || oapiErr.StatusCode == 403 {
				continue
			}
		}
		errs = append(errs, fmt.Errorf("deleting region: %w", err))
	}
	return errors.Join(errs...)
}

func testVpcRoutingTableStaticRouteDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := iaas.NewAPIClient(testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildClientOptions(testutil.IaaSCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type staticRouteIds struct {
		projectId      string
		vpcId          string
		region         string
		routingTableId string
		routeId        string
	}

	routesToDestroy := []staticRouteIds{}
	var errs []error
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_vpc_routing_table_static_route" {
			continue
		}
		projectId, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", rs.Primary))
			continue
		}
		vpcId, ok := rs.Primary.Attributes["vpc_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no vpc_id found in %s", rs.Primary))
			continue
		}
		region, ok := rs.Primary.Attributes["region"]
		if !ok {
			errs = append(errs, fmt.Errorf("no region found in %s", rs.Primary))
			continue
		}
		routingTableId, ok := rs.Primary.Attributes["routing_table_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no routing_table_id found in %s", rs.Primary))
			continue
		}
		routeId, ok := rs.Primary.Attributes["route_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no route_id found in %s", rs.Primary))
			continue
		}
		routesToDestroy = append(routesToDestroy, staticRouteIds{
			projectId:      projectId,
			vpcId:          vpcId,
			region:         region,
			routingTableId: routingTableId,
			routeId:        routeId,
		})
	}

	for _, route := range routesToDestroy {
		_, err := client.DefaultAPI.GetVPCStaticRoute(ctx, route.projectId, route.vpcId, route.region, route.routingTableId, route.routeId).Execute()
		if err == nil {
			err := client.DefaultAPI.DeleteVPCStaticRoute(ctx, route.projectId, route.vpcId, route.region, route.routingTableId, route.routeId).Execute()
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting static route with ID %q in project %q, vpc %q, region %q, routing table %q : %w", route.routeId, route.projectId, route.vpcId, route.region, route.routingTableId, err))
			}
			continue
		}

		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if oapiErr.StatusCode == 404 || oapiErr.StatusCode == 403 {
				continue
			}
		}
		errs = append(errs, fmt.Errorf("deleting static route: %w", err))
	}

	return errors.Join(errs...)
}

func testVpcNetworkRangeDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := iaas.NewAPIClient(testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildClientOptions(testutil.IaaSCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type networkRangeIds struct {
		projectId      string
		vpcId          string
		region         string
		networkRangeId string
	}

	networkRangesToDestroy := []networkRangeIds{}
	var errs []error
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_vpc_network_range" {
			continue
		}
		projectId, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", rs.Primary))
			continue
		}
		vpcId, ok := rs.Primary.Attributes["vpc_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no vpc_id found in %s", rs.Primary))
			continue
		}
		region, ok := rs.Primary.Attributes["region"]
		if !ok {
			errs = append(errs, fmt.Errorf("no region found in %s", rs.Primary))
			continue
		}
		networkRangeId, ok := rs.Primary.Attributes["network_range_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no network_range_id found in %s", rs.Primary))
			continue
		}
		networkRangesToDestroy = append(networkRangesToDestroy, networkRangeIds{
			projectId:      projectId,
			vpcId:          vpcId,
			region:         region,
			networkRangeId: networkRangeId,
		})
	}

	for _, networkRange := range networkRangesToDestroy {
		_, err := client.DefaultAPI.GetVPCNetworkRange(ctx, networkRange.projectId, networkRange.vpcId, networkRange.region, networkRange.networkRangeId).Execute()
		if err == nil {
			err := client.DefaultAPI.DeleteVPCNetworkRange(ctx, networkRange.projectId, networkRange.vpcId, networkRange.region, networkRange.networkRangeId).Execute()
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting network range with ID %q in project %q, vpc %q, region %q : %w", networkRange.networkRangeId, networkRange.projectId, networkRange.vpcId, networkRange.region, err))
			}
			continue
		}

		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if oapiErr.StatusCode == 404 || oapiErr.StatusCode == 403 {
				continue
			}
		}
		errs = append(errs, fmt.Errorf("deleting static route: %w", err))
	}

	return errors.Join(errs...)
}
