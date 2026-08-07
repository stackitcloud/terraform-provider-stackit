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
	intake "github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-runner-min.tf
var resourceIntakeRunnerMin string

//go:embed testdata/resource-runner-max.tf
var resourceIntakeRunnerMax string

//go:embed testdata/resource-intake-min.tf
var resourceIntakesMin string

//go:embed testdata/resource-intake-max.tf
var resourceIntakesMax string

const intakeRunnerResource = "stackit_intake_runner.example"
const intakesResource = "stackit_intakes.example"

var testIntakeRunnerConfigVarsMin = config.Variables{
	"project_id":            config.StringVariable(testutil.ProjectId),
	"name":                  config.StringVariable("tf-acc-runner-min"),
	"max_message_size_kib":  config.IntegerVariable(1024),
	"max_messages_per_hour": config.IntegerVariable(1000),
}

var testIntakeRunnerConfigVarsMax = config.Variables{
	"project_id":            config.StringVariable(testutil.ProjectId),
	"name":                  config.StringVariable("tf-acc-runner-max"),
	"region":                config.StringVariable(testutil.Region),
	"description":           config.StringVariable("An example runner for Intake"),
	"max_message_size_kib":  config.IntegerVariable(1024),
	"max_messages_per_hour": config.IntegerVariable(1100),
}

// TODO: Intakes creation acceptance tests require a valid Dremio PAT.
var testIntakesConfigVarsMin = config.Variables{
	"project_id":                   config.StringVariable(testutil.ProjectId),
	"runner_name":                  config.StringVariable("tf-acc-runner-min"),
	"intake_name":                  config.StringVariable("tf-acc-intake-min"),
	"max_message_size_kib":         config.IntegerVariable(1024),
	"max_messages_per_hour":        config.IntegerVariable(1000),
	"dremio_display_name":          config.StringVariable("tfAccDremioIntakeMin"),
	"dremio_user_email":            config.StringVariable("tf-acc-intake-min@example.com"),
	"dremio_user_first_name":       config.StringVariable("Intake"),
	"dremio_user_last_name":        config.StringVariable("Min"),
	"dremio_user_name":             config.StringVariable("tf_acc_intakeminuser"),
	"dremio_user_password":         config.StringVariable("TestAcceptance12345!@"),
	"dremio_personal_access_token": config.StringVariable("pending-dremio-pat"),
}

var testIntakesConfigVarsMax = config.Variables{
	"project_id":                   config.StringVariable(testutil.ProjectId),
	"region":                       config.StringVariable(testutil.Region),
	"runner_name":                  config.StringVariable("tf-acc-runner-for-max"),
	"intake_name":                  config.StringVariable("tf-acc-intake-max"),
	"description":                  config.StringVariable("An example full intake with dynamic Dremio"),
	"max_message_size_kib":         config.IntegerVariable(1024),
	"max_messages_per_hour":        config.IntegerVariable(1000),
	"dremio_display_name":          config.StringVariable("tfAccDremioIntakeMax"),
	"dremio_user_email":            config.StringVariable("tf-acc-test@example.com"),
	"dremio_user_first_name":       config.StringVariable("Acc"),
	"dremio_user_last_name":        config.StringVariable("Test"),
	"dremio_user_name":             config.StringVariable("tf_acc_dremio_user"),
	"dremio_user_password":         config.StringVariable("TestAcceptance12345!@"),
	"dremio_personal_access_token": config.StringVariable("pending-dremio-pat"),
}

func testIntakeRunnerConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testIntakeRunnerConfigVarsMin))
	maps.Copy(tempConfig, testIntakeRunnerConfigVarsMin)
	tempConfig["name"] = config.StringVariable("tf-acc-runner-min-upd")
	return tempConfig
}

func testIntakeRunnerConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testIntakeRunnerConfigVarsMax))
	maps.Copy(tempConfig, testIntakeRunnerConfigVarsMax)
	tempConfig["name"] = config.StringVariable("tf-acc-runner-max-upd")
	return tempConfig
}

func testIntakesConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testIntakesConfigVarsMin))
	maps.Copy(tempConfig, testIntakesConfigVarsMin)
	tempConfig["intake_name"] = config.StringVariable("tf-acc-intake-min-upd")
	return tempConfig
}

func testIntakesConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testIntakesConfigVarsMax))
	maps.Copy(tempConfig, testIntakesConfigVarsMax)
	tempConfig["intake_name"] = config.StringVariable("tf-acc-intake-max-upd")
	tempConfig["description"] = config.StringVariable("Updated full intake description")
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
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceIntakeRunnerMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["name"])),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "description"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "labels"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["max_messages_per_hour"])),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "uri"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "create_time"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.Region),
				),
			},
			// Data source check: creates config that includes resource and data source
			{
				ConfigVariables: testIntakeRunnerConfigVarsMin,
				Config: fmt.Sprintf(`
				%s
				%s
				data "stackit_intake_runner" "example" {
					project_id = %s.project_id
					runner_id  = %s.runner_id
					region     = %s.region
				}`, testutil.NewConfigBuilder().BuildProviderConfig(), resourceIntakeRunnerMin, intakeRunnerResource, intakeRunnerResource, intakeRunnerResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Make sure it's correctly found resource by comparing runner_id attribute
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "project_id", "data.stackit_intake_runner.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "runner_id", "data.stackit_intake_runner.example", "runner_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "name", "data.stackit_intake_runner.example", "name"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "region", "data.stackit_intake_runner.example", "region"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "description"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "labels"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "uri", "data.stackit_intake_runner.example", "uri"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "create_time", "data.stackit_intake_runner.example", "create_time"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "max_messages_per_hour", "data.stackit_intake_runner.example", "max_messages_per_hour"),
				),
			},
			// Simulate terraform import
			{
				ConfigVariables:   testIntakeRunnerConfigVarsMin,
				Config:            testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMin,
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
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMin["max_messages_per_hour"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.Region),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "description"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "labels"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "uri"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "create_time"),
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
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMax,
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
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "uri"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "create_time"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["region"])),
				),
			},
			{
				ConfigVariables: testIntakeRunnerConfigVarsMax,
				Config: fmt.Sprintf(`
				%s
				%s
				data "stackit_intake_runner" "example" {
					project_id = %s.project_id
					runner_id  = %s.runner_id
				}`, testutil.NewConfigBuilder().BuildProviderConfig(), resourceIntakeRunnerMax, intakeRunnerResource, intakeRunnerResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "project_id", "data.stackit_intake_runner.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "runner_id", "data.stackit_intake_runner.example", "runner_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "name", "data.stackit_intake_runner.example", "name"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "description", "data.stackit_intake_runner.example", "description"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "region", "data.stackit_intake_runner.example", "region"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "uri", "data.stackit_intake_runner.example", "uri"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "create_time", "data.stackit_intake_runner.example", "create_time"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "labels.env", "data.stackit_intake_runner.example", "labels.env"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "max_messages_per_hour", "data.stackit_intake_runner.example", "max_messages_per_hour"),
				),
			},
			// Simulate terraform import
			{
				ConfigVariables:   testIntakeRunnerConfigVarsMax,
				Config:            testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMax,
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
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMax,
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
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "uri"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "create_time"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMax["region"])),
				),
			},
		},
	})
}

// TODO: Intakes acceptance tests (TestAccIntakesMin and TestAccIntakesMax)
func TestAccIntakesMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakesDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create minimal intake
			{
				ConfigVariables: testIntakesConfigVarsMin,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + resourceIntakesMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakesResource, "project_id", testutil.ConvertConfigVariable(testIntakesConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(intakesResource, "name", testutil.ConvertConfigVariable(testIntakesConfigVarsMin["intake_name"])),
					resource.TestCheckResourceAttrSet(intakesResource, "intake_id"),
					resource.TestCheckResourceAttrSet(intakesResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakesResource, "id"),
					resource.TestCheckResourceAttrSet(intakesResource, "uri"),
					resource.TestCheckResourceAttrSet(intakesResource, "create_time"),
					resource.TestCheckResourceAttr(intakesResource, "region", testutil.Region),
				),
			},
			// Step 2: Data source check
			{
				ConfigVariables: testIntakesConfigVarsMin,
				Config: fmt.Sprintf(`
				%s
				%s
				data "stackit_intakes" "example" {
					project_id = %s.project_id
					intake_id  = %s.intake_id
					region     = %s.region
				}`, testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceIntakesMin, intakesResource, intakesResource, intakesResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(intakesResource, "project_id", "data.stackit_intakes.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "intake_id", "data.stackit_intakes.example", "intake_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "name", "data.stackit_intakes.example", "name"),
					resource.TestCheckResourceAttrPair(intakesResource, "runner_id", "data.stackit_intakes.example", "runner_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "region", "data.stackit_intakes.example", "region"),
					resource.TestCheckResourceAttrPair(intakesResource, "uri", "data.stackit_intakes.example", "uri"),
					resource.TestCheckResourceAttrPair(intakesResource, "create_time", "data.stackit_intakes.example", "create_time"),
				),
			},
			// Step 3: Import state check
			{
				ConfigVariables:   testIntakesConfigVarsMin,
				Config:            testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceIntakesMin,
				ResourceName:      intakesResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[intakesResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", intakesResource)
					}
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["project_id"], r.Primary.Attributes["region"], r.Primary.Attributes["intake_id"]), nil
				},
			},
			// Step 4: Update check
			{
				ConfigVariables: testIntakesConfigVarsMinUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceIntakesMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakesResource, "project_id", testutil.ConvertConfigVariable(testIntakesConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr(intakesResource, "name", testutil.ConvertConfigVariable(testIntakesConfigVarsMinUpdated()["intake_name"])),
					resource.TestCheckResourceAttrSet(intakesResource, "intake_id"),
				),
			},
		},
	})
}

// TODO: Intakes acceptance tests are put on hold pending STACKIT Dremio team clarification.
func TestAccIntakesMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakesDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create full intake with dynamic Dremio
			{
				ConfigVariables: testIntakesConfigVarsMax,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceIntakesMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakesResource, "project_id", testutil.ConvertConfigVariable(testIntakesConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(intakesResource, "name", testutil.ConvertConfigVariable(testIntakesConfigVarsMax["intake_name"])),
					resource.TestCheckResourceAttr(intakesResource, "description", testutil.ConvertConfigVariable(testIntakesConfigVarsMax["description"])),
					resource.TestCheckResourceAttr(intakesResource, "labels.env", "development"),
					resource.TestCheckResourceAttr(intakesResource, "labels.created_by", "terraform-provider-stackit"),
					resource.TestCheckResourceAttr(intakesResource, "catalog_auth_type", "dremio"),
					resource.TestCheckResourceAttr(intakesResource, "catalog_namespace", "intake"),
					resource.TestCheckResourceAttr(intakesResource, "catalog_warehouse", "default"),
					resource.TestCheckResourceAttrSet(intakesResource, "intake_id"),
					resource.TestCheckResourceAttrSet(intakesResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakesResource, "catalog_uri"),
					resource.TestCheckResourceAttrSet(intakesResource, "catalog_table_name"),
				),
			},
			// Step 2: Data source check
			{
				ConfigVariables: testIntakesConfigVarsMax,
				Config: fmt.Sprintf(`
				%s
				%s
				data "stackit_intakes" "example" {
					project_id = %s.project_id
					intake_id  = %s.intake_id
				}`, testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), resourceIntakesMax, intakesResource, intakesResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(intakesResource, "project_id", "data.stackit_intakes.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "intake_id", "data.stackit_intakes.example", "intake_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "name", "data.stackit_intakes.example", "name"),
					resource.TestCheckResourceAttrPair(intakesResource, "description", "data.stackit_intakes.example", "description"),
					resource.TestCheckResourceAttrPair(intakesResource, "catalog_auth_type", "data.stackit_intakes.example", "catalog_auth_type"),
					resource.TestCheckResourceAttrPair(intakesResource, "catalog_namespace", "data.stackit_intakes.example", "catalog_namespace"),
					resource.TestCheckResourceAttrPair(intakesResource, "catalog_warehouse", "data.stackit_intakes.example", "catalog_warehouse"),
				),
			},
			// Step 3: Import state check (ignore write-only PAT)
			{
				ConfigVariables:         testIntakesConfigVarsMax,
				Config:                  testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceIntakesMax,
				ResourceName:            intakesResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"dremio_personal_access_token"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[intakesResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", intakesResource)
					}
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["project_id"], r.Primary.Attributes["region"], r.Primary.Attributes["intake_id"]), nil
				},
			},
			// Step 4: Update check
			{
				ConfigVariables: testIntakesConfigVarsMaxUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + resourceIntakesMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakesResource, "project_id", testutil.ConvertConfigVariable(testIntakesConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(intakesResource, "name", testutil.ConvertConfigVariable(testIntakesConfigVarsMaxUpdated()["intake_name"])),
					resource.TestCheckResourceAttr(intakesResource, "description", testutil.ConvertConfigVariable(testIntakesConfigVarsMaxUpdated()["description"])),
					resource.TestCheckResourceAttrSet(intakesResource, "intake_id"),
				),
			},
		},
	})
}

// testAccCheckIntakeRunnerDestroy act as independent auditor to verify destroy operation
func testAccCheckIntakeRunnerDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := intake.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.GitCustomEndpoint, false)...)
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
	instancesResp, err := client.DefaultAPI.ListIntakeRunners(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	// If the API returns a list of runners, check if our deleted ones are still there
	items := instancesResp.IntakeRunners
	for i := range items {
		// If a runner we thought we deleted is found in the list
		if utils.Contains(instancesToDestroy, items[i].Id) {
			// Attempt a final delete and wait
			err := client.DefaultAPI.DeleteIntakeRunner(ctx, testutil.ProjectId, testutil.Region, items[i].Id).Execute()
			if err != nil {
				return fmt.Errorf("deleting runner %s during CheckDestroy: %w", items[i].Id, err)
			}

			// Using the wait handler for destruction verification
			_, err = wait.DeleteIntakeRunnerWaitHandler(ctx, client.DefaultAPI, testutil.ProjectId, testutil.Region, items[i].Id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("deleting runner %s during CheckDestroy: waiting for deletion %w", items[i].Id, err)
			}
		}
	}
	return nil
}

// testAccCheckIntakesDestroy act as independent auditor to verify destroy operation for intakes
func testAccCheckIntakesDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := intake.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.GitCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_intakes" {
			continue
		}
		// Intake internal ID: "[project_id],[region],[intake_id]"
		idParts := strings.Split(rs.Primary.ID, core.Separator)
		if len(idParts) < 3 {
			continue
		}
		intakeId := idParts[2]
		instancesToDestroy = append(instancesToDestroy, intakeId)
	}

	instancesResp, err := client.DefaultAPI.ListIntakes(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	items := instancesResp.Intakes
	for i := range items {
		if utils.Contains(instancesToDestroy, items[i].Id) {
			err := client.DefaultAPI.DeleteIntake(ctx, testutil.ProjectId, testutil.Region, items[i].Id).Execute()
			if err != nil {
				return fmt.Errorf("deleting intake %s during CheckDestroy: %w", items[i].Id, err)
			}

			_, err = wait.DeleteIntakeWaitHandler(ctx, client.DefaultAPI, testutil.ProjectId, testutil.Region, items[i].Id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("deleting intake %s during CheckDestroy: waiting for deletion %w", items[i].Id, err)
			}
		}
	}
	return nil
}
