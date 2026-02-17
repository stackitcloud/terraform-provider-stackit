package intake_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	sdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/intake"
	"github.com/stackitcloud/stackit-sdk-go/services/intake/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceIntakeRunnerMin string

//go:embed testdata/resource-max.tf
var resourceIntakeRunnerMax string

const intakeRunnerResource = "stackit_intake_runner.example"

var testIntakeRunnerConfigVarsMin = config.Variables{
	"project_id":            config.StringVariable(testutil.ProjectId),
	"name":                  config.StringVariable("intake-min-runner"),
	"region":                config.StringVariable(testutil.Region),
	"max_message_size_kib":  config.IntegerVariable(1024),
	"max_messages_per_hour": config.IntegerVariable(1000),
}

var testIntakeRunnerConfigVarsMax = config.Variables{
	"project_id":            config.StringVariable(testutil.ProjectId),
	"name":                  config.StringVariable("intake-max-runner"),
	"region":                config.StringVariable(testutil.Region),
	"description":           config.StringVariable("An example runner for Intake"),
	"max_message_size_kib":  config.IntegerVariable(1024),
	"max_messages_per_hour": config.IntegerVariable(1100),
}

func testIntakeRunnerConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testIntakeRunnerConfigVarsMin))
	maps.Copy(tempConfig, testIntakeRunnerConfigVarsMin)
	tempConfig["name"] = config.StringVariable("intake-min-runner-upd")
	return tempConfig
}

func testIntakeRunnerConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testIntakeRunnerConfigVarsMax))
	maps.Copy(tempConfig, testIntakeRunnerConfigVarsMax)
	tempConfig["name"] = config.StringVariable("intake-max-runner-upd")
	return tempConfig
}

func TestAccIntakeRunnerMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakeRunnerDestroy,
		Steps: []resource.TestStep{
			// Create the minimum runner from the HCL file
			{
				ConfigVariables: testIntakeRunnerConfigVarsMin,
				Config:          testutil.IntakeProviderConfig() + "\n" + resourceIntakeRunnerMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["name"])),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["max_messages_per_hour"])),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["region"])),
				),
			},
			// Data source check: creates config that includes resource and data source
			{
				ConfigVariables: testIntakeRunnerConfigVarsMin,
				Config: fmt.Sprintf(`
				%s
				data "stackit_intake_runner" "example" {
					project_id = %s.project_id
					runner_id  = %s.runner_id
					region     = %s.region
				}`, testutil.IntakeProviderConfig()+"\n"+resourceIntakeRunnerMin, intakeRunnerResource, intakeRunnerResource, intakeRunnerResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Make sure it's correctly found resource by comparing runner_id attribute
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "project_id", "data.stackit_intake_runner.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "runner_id", "data.stackit_intake_runner.example", "runner_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "name", "data.stackit_intake_runner.example", "name"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "region", "data.stackit_intake_runner.example", "region"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "max_messages_per_hour", "data.stackit_intake_runner.example", "max_messages_per_hour"),
				),
			},
			// Simulate terraform import
			{
				ConfigVariables:   testIntakeRunnerConfigVarsMin,
				Config:            testutil.IntakeProviderConfig() + "\n" + resourceIntakeRunnerMin,
				ResourceName:      intakeRunnerResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					// Construct ID string
					r, ok := s.RootModule().Resources[intakeRunnerResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", intakeRunnerResource)
					}
					// ID structure: project_id, region, runner_id
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["project_id"], r.Primary.Attributes["region"], r.Primary.Attributes["runner_id"]), nil
				},
			},
			// Update check: verifies API updated resource name without crashing
			{
				ConfigVariables: testIntakeRunnerConfigVarsMinUpdated(),
				Config:          testutil.IntakeProviderConfig() + "\n" + resourceIntakeRunnerMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["max_messages_per_hour"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["region"])),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
				),
			},
		},
	})
}

func TestAccIntakeRunnerMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakeRunnerDestroy,
		Steps: []resource.TestStep{
			// Create the max intake runner from HCL files and verify comparison
			{
				ConfigVariables: testIntakeRunnerConfigVarsMax,
				Config:          testutil.IntakeProviderConfig() + "\n" + resourceIntakeRunnerMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["name"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "description", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["description"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["max_messages_per_hour"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.%", "2"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.env", "development"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.created_by", "terraform-provider-stackit"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["region"])),
				),
			},
			{
				ConfigVariables: testIntakeRunnerConfigVarsMax,
				Config: fmt.Sprintf(`
				%s
				data "stackit_intake_runner" "example" {
					project_id = %s.project_id
					runner_id  = %s.runner_id
				}`, testutil.IntakeProviderConfig()+"\n"+resourceIntakeRunnerMax, intakeRunnerResource, intakeRunnerResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "project_id", "data.stackit_intake_runner.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "runner_id", "data.stackit_intake_runner.example", "runner_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "name", "data.stackit_intake_runner.example", "name"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "description", "data.stackit_intake_runner.example", "description"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "region", "data.stackit_intake_runner.example", "region"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "labels.env", "data.stackit_intake_runner.example", "labels.env"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "max_messages_per_hour", "data.stackit_intake_runner.example", "max_messages_per_hour"),
				),
			},
			// Simulate terraform import
			{
				ConfigVariables:   testIntakeRunnerConfigVarsMax,
				Config:            testutil.IntakeProviderConfig() + "\n" + resourceIntakeRunnerMax,
				ResourceName:      intakeRunnerResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					// Construct ID string
					r, ok := s.RootModule().Resources[intakeRunnerResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", intakeRunnerResource)
					}
					// ID structure: project_id, region, runner_id
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["project_id"], r.Primary.Attributes["region"], r.Primary.Attributes["runner_id"]), nil
				},
			},
			// Update and verify changes are reflected
			{
				ConfigVariables: testIntakeRunnerConfigVarsMaxUpdated(),
				Config:          testutil.IntakeProviderConfig() + "\n" + resourceIntakeRunnerMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "description", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["description"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["max_messages_per_hour"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.%", "2"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.env", "development"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.created_by", "terraform-provider-stackit"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["region"])),
				),
			},
		},
	})
}

// testAccCheckIntakeRunnerDestroy act as independent auditor to verify destroy operation
func testAccCheckIntakeRunnerDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *intake.APIClient
	var err error

	if testutil.IntakeCustomEndpoint == "" {
		client, err = intake.NewAPIClient(
			sdkConfig.WithRegion(testutil.Region),
		)
	} else {
		client, err = intake.NewAPIClient(
			sdkConfig.WithEndpoint(testutil.IntakeCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_intake_runner" {
			continue
		}
		// Intake internal ID: "[project_id],[region],[runner_id]"
		runnerId := strings.Split(rs.Primary.ID, core.Separator)[2]
		instancesToDestroy = append(instancesToDestroy, runnerId)
	}

	// List all resources in the project/region to see what's left
	instancesResp, err := client.ListIntakeRunners(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	// If the API returns a list of runners, check if our deleted ones are still there
	items := *instancesResp.IntakeRunners
	for i := range items {
		if items[i].Id == nil {
			continue
		}

		// If a runner we thought we deleted is found in the list
		if utils.Contains(instancesToDestroy, *items[i].Id) {
			// Attempt a final delete and wait, just like Postgres
			err := client.DeleteIntakeRunner(ctx, testutil.ProjectId, testutil.Region, *items[i].Id).Execute()
			if err != nil {
				return fmt.Errorf("deleting runner %s during CheckDestroy: %w", *items[i].Id, err)
			}

			// Using the wait handler for destruction verification
			_, err = wait.DeleteIntakeRunnerWaitHandler(ctx, client, testutil.ProjectId, testutil.Region, *items[i].Id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("deleting runner %s during CheckDestroy: waiting for deletion %w", *items[i].Id, err)
			}
		}
	}
	return nil
}
