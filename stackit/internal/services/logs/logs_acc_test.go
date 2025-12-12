package logs_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	coreConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/logs"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceMin string

//go:embed testdata/resource-max.tf
var resourceMax string

var testConfigVarsMin = config.Variables{
	"project_id":     config.StringVariable(testutil.ProjectId),
	"region":         config.StringVariable(testutil.Region),
	"display_name":   config.StringVariable("tf-acc-test-logs-min"),
	"retention_days": config.IntegerVariable(7),
}

func testConfigVarsMinUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(newVars, testConfigVarsMin)
	newVars["display_name"] = config.StringVariable("tf-acc-test-logs-updated")
	newVars["retention_days"] = config.IntegerVariable(14)
	return newVars
}

var testConfigVarsMax = config.Variables{
	"project_id":     config.StringVariable(testutil.ProjectId),
	"region":         config.StringVariable(testutil.Region),
	"display_name":   config.StringVariable("tf-acc-test-logs-max"),
	"retention_days": config.IntegerVariable(7),
	"acl":            config.StringVariable("192.168.0.1/24"),
	"description":    config.StringVariable("Terraform Acceptance Test Logs Instance"),
}

func testConfigVarsMaxUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(newVars, testConfigVarsMin)
	newVars["display_name"] = config.StringVariable("tf-acc-test-logs-updated")
	newVars["retention_days"] = config.IntegerVariable(14)
	newVars["acl"] = config.StringVariable("192.168.0.1/16")
	newVars["description"] = config.StringVariable("Terraform Acceptance Test Logs Instance Updated")
	return newVars
}

func TestAccLogsInstanceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLogsInstanceDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.LogsProviderConfig() + resourceMin,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "region", testutil.ConvertConfigVariable(testConfigVarsMin["region"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "display_name", testutil.ConvertConfigVariable(testConfigVarsMin["display_name"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "retention_days", testutil.ConvertConfigVariable(testConfigVarsMin["retention_days"])),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "id"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "created"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "datasource_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "ingest_otlp_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "ingest_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "query_range_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "query_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "status"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigVarsMin,
				Config: testutil.LogsProviderConfig() + resourceMin + `
data "stackit_logs_instance" "logs" {
  project_id   = stackit_logs_instance.logs.project_id
  region       = stackit_logs_instance.logs.region
  instance_id  = stackit_logs_instance.logs.instance_id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_logs_instance.logs", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "region",
						"data.stackit_logs_instance.logs", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "display_name",
						"data.stackit_logs_instance.logs", "display_name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "retention_days",
						"data.stackit_logs_instance.logs", "retention_days",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "id",
						"data.stackit_logs_instance.logs", "id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "instance_id",
						"data.stackit_logs_instance.logs", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "created",
						"data.stackit_logs_instance.logs", "created",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "datasource_url",
						"data.stackit_logs_instance.logs", "datasource_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "ingest_otlp_url",
						"data.stackit_logs_instance.logs", "ingest_otlp_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "ingest_url",
						"data.stackit_logs_instance.logs", "ingest_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "query_range_url",
						"data.stackit_logs_instance.logs", "query_range_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "query_url",
						"data.stackit_logs_instance.logs", "query_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "status",
						"data.stackit_logs_instance.logs", "status",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_logs_instance.logs",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_logs_instance.logs"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_logs_instance.logs")
					}
					instanceId, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsMinUpdated(),
				Config:          testutil.LogsProviderConfig() + resourceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "project_id", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "region", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["region"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "display_name", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "retention_days", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["retention_days"])),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "id"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "created"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "datasource_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "ingest_otlp_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "ingest_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "query_range_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "query_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "status"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func TestAccLogsInstanceMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLogsInstanceDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.LogsProviderConfig() + resourceMax,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "region", testutil.ConvertConfigVariable(testConfigVarsMax["region"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "display_name", testutil.ConvertConfigVariable(testConfigVarsMax["display_name"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "retention_days", testutil.ConvertConfigVariable(testConfigVarsMax["retention_days"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "description", testutil.ConvertConfigVariable(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "id"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "created"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "datasource_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "ingest_otlp_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "ingest_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "query_range_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "query_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "status"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigVarsMax,
				Config: testutil.LogsProviderConfig() + resourceMax + `
data "stackit_logs_instance" "logs" {
  project_id   = stackit_logs_instance.logs.project_id
  region       = stackit_logs_instance.logs.region
  instance_id  = stackit_logs_instance.logs.instance_id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_logs_instance.logs", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "region",
						"data.stackit_logs_instance.logs", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "display_name",
						"data.stackit_logs_instance.logs", "display_name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "retention_days",
						"data.stackit_logs_instance.logs", "retention_days",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "id",
						"data.stackit_logs_instance.logs", "id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "instance_id",
						"data.stackit_logs_instance.logs", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "created",
						"data.stackit_logs_instance.logs", "created",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "datasource_url",
						"data.stackit_logs_instance.logs", "datasource_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "ingest_otlp_url",
						"data.stackit_logs_instance.logs", "ingest_otlp_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "ingest_url",
						"data.stackit_logs_instance.logs", "ingest_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "query_range_url",
						"data.stackit_logs_instance.logs", "query_range_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "query_url",
						"data.stackit_logs_instance.logs", "query_url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "status",
						"data.stackit_logs_instance.logs", "status",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "acl.0",
						"data.stackit_logs_instance.logs", "acl.0",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_logs_instance.logs", "description",
						"data.stackit_logs_instance.logs", "description",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_logs_instance.logs",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_logs_instance.logs"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_logs_instance.logs")
					}
					instanceId, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsMaxUpdated(),
				Config:          testutil.LogsProviderConfig() + resourceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "project_id", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "region", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["region"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "display_name", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "retention_days", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["retention_days"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["acl"])),
					resource.TestCheckResourceAttr("stackit_logs_instance.logs", "description", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["description"])),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "id"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "created"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "datasource_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "ingest_otlp_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "ingest_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "query_range_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "query_url"),
					resource.TestCheckResourceAttrSet("stackit_logs_instance.logs", "status"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func testAccCheckLogsInstanceDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *logs.APIClient
	var err error
	if testutil.LogsCustomEndpoint == "" {
		client, err = logs.NewAPIClient(
			coreConfig.WithRegion("eu01"),
		)
	} else {
		client, err = logs.NewAPIClient(
			coreConfig.WithEndpoint(testutil.LogsCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var instancesToDestroy []string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_logs_instance" {
			continue
		}
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[2]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	response, err := client.ListLogsInstances(ctx, testutil.ProjectId, "eu01").Execute()
	if err != nil {
		return fmt.Errorf("getting instances: %w", err)
	}
	for _, i := range *response.Instances {
		if !slices.Contains(instancesToDestroy, *i.Id) {
			continue
		}
		err := client.DeleteLogsInstance(ctx, testutil.ProjectId, "eu01", *i.Id).Execute()
		if err != nil {
			return fmt.Errorf("deleting instance %s: %w", *i.Id, err)
		}
	}
	return nil
}
