package intake_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	sdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/intake"
	"github.com/stackitcloud/stackit-sdk-go/services/intake/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceMin string

//go:embed testdata/resource-max.tf
var resourceMax string

const intakeRunnerResource = "stackit_intake_runner.example"

const (
	intakeRunnerMinName        = "intake-min-runner"
	intakeRunnerMinNameUpdated = "intake-min-runner-upd"
	intakeRunnerMaxName        = "intake-max-runner"
	intakeRunnerMaxNameUpdated = "intake-max-runner-upd"
)

var testConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(intakeRunnerMinName),
}

var testConfigVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(intakeRunnerMaxName),
}

func testConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(tempConfig, testConfigVarsMin)
	tempConfig["name"] = config.StringVariable(intakeRunnerMinNameUpdated)
	return tempConfig
}

func testConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMax))
	maps.Copy(tempConfig, testConfigVarsMax)
	tempConfig["name"] = config.StringVariable(intakeRunnerMaxNameUpdated)
	return tempConfig
}

func TestAccIntakeRunnerMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakeRunnerDestroy,
		Steps: []resource.TestStep{
			// Create the minimum runner from the HCL file
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.IntakeProviderConfig() + "\n" + resourceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", intakeRunnerMinName),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", "1024"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", "1000"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "region"),
				),
			},
			// Data source check: creates config that includes resource and data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
					%s
					data "stackit_intake_runner" "example" {
						project_id = %s.project_id
						runner_id  = %s.runner_id
						region     = %s.region
					}`, testutil.IntakeProviderConfig()+"\n"+resourceMin, intakeRunnerResource, intakeRunnerResource, intakeRunnerResource),
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
				ConfigVariables:   testConfigVarsMin,
				Config:            testutil.IntakeProviderConfig() + "\n" + resourceMin,
				ResourceName:      intakeRunnerResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					// Construct ID string
					r, ok := s.RootModule().Resources[intakeRunnerResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", intakeRunnerResource)
					}
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["project_id"], r.Primary.Attributes["region"], r.Primary.Attributes["runner_id"]), nil
				},
			},
			// Update check: verifies API updated resource name without crashing
			{
				ConfigVariables: testConfigVarsMinUpdated(),
				Config:          testutil.IntakeProviderConfig() + "\n" + resourceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", intakeRunnerMinNameUpdated),
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
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.IntakeProviderConfig() + "\n" + resourceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "description", "An example runner for Intake"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", "1024"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", "1100"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.%", "2"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.env", "development"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.created_by", "terraform-provider-stackit"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "region"),
				),
			},
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
					%s
					data "stackit_intake_runner" "example" {
						project_id = %s.project_id
						runner_id  = %s.runner_id
					}`, testutil.IntakeProviderConfig()+"\n"+resourceMax, intakeRunnerResource, intakeRunnerResource),
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
			// Update and verify changes are reflected
			{
				ConfigVariables: testConfigVarsMaxUpdated(),
				Config:          testutil.IntakeProviderConfig() + "\n" + resourceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["name"])),
					// Ensure optional fields survived the update (didn't get wiped by a bad Update payload)
					resource.TestCheckResourceAttr(intakeRunnerResource, "description", "An example runner for Intake"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.env", "development"),
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
		client, err = intake.NewAPIClient()
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
