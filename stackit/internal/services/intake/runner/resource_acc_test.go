package runner_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	sdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/intake"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// intakeRunnerResource is the name of the test resource
const intakeRunnerResource = "stackit_intake_runner.example"

func TestAccIntakeRunner(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakeRunnerDestroy,
		Steps: []resource.TestStep{
			// create the runner
			{
				Config: testutil.IntakeProviderConfig() + testAccIntakeRunnerConfigMinimal("example-runner-minimal"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", "example-runner-minimal"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "description", ""),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.%", "0"),
				),
			},
			// update the runner
			{
				Config: testutil.IntakeProviderConfig() + testAccIntakeRunnerConfigFull("example-runner-full", "An example runner for Intake", 1024, 1100),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", "example-runner-full"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "description", "An example runner for Intake"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", "1024"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", "1100"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.%", "2"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.created_by", "terraform-provider-stackit"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.env", "development"),
				),
			},
			// importing the runner
			{
				ResourceName:      intakeRunnerResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update to remove optional attributes
			{
				Config: testutil.IntakeProviderConfig() + testAccIntakeRunnerConfigUpdated("example-runner-updated", 1024, 1100),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", "example-runner-updated"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "description", ""),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.%", "0"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", "1024"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", "1100"),
				),
			},
		},
	})
}

func testAccIntakeRunnerConfigMinimal(name string) string {
	return fmt.Sprintf(`
        resource "stackit_intake_runner" "example" {
            project_id = "%s"
            name       = "%s"
            max_message_size_kib    = 1024
            max_messages_per_hour   = 1000
        }
        `,
		testutil.ProjectId,
		name,
	)
}

func testAccIntakeRunnerConfigFull(name, description string, maxKib, maxPerHour int) string {
	return fmt.Sprintf(`
        resource "stackit_intake_runner" "example" {
            project_id              = "%s"
            name                    = "%s"
            description             = "%s"
            max_message_size_kib    = %d
            max_messages_per_hour   = %d
            labels = {
                "created_by" = "terraform-provider-stackit"
                "env"        = "development"
            }
        }
        `,
		testutil.ProjectId,
		name,
		description,
		maxKib,
		maxPerHour,
	)
}

func testAccIntakeRunnerConfigUpdated(name string, maxKib, maxPerHour int) string {
	return fmt.Sprintf(`
        resource "stackit_intake_runner" "example" {
            project_id              = "%s"
            name                    = "%s"
            description             = ""
            max_message_size_kib    = %d
            max_messages_per_hour   = %d
            labels                  = {}
        }
        `,
		testutil.ProjectId,
		name,
		maxKib,
		maxPerHour,
	)
}

func testAccCheckIntakeRunnerDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *intake.APIClient
	var err error
	if testutil.IntakeCustomEndpoint == "" {
		client, err = intake.NewAPIClient()
	} else {
		client, err = intake.NewAPIClient(sdkConfig.WithEndpoint(testutil.IntakeCustomEndpoint))
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_intake_runner" {
			continue
		}
		// Try to find the runner
		_, err := client.GetIntakeRunner(ctx, rs.Primary.Attributes["project_id"], rs.Primary.Attributes["region"], rs.Primary.Attributes["runner_id"]).Execute()
		if err == nil {
			err = client.DeleteIntakeRunner(ctx, rs.Primary.Attributes["project_id"], rs.Primary.Attributes["region"], rs.Primary.Attributes["runner_id"]).Execute()
			if err != nil {
				return fmt.Errorf("intake runner with ID %s still existed, got an error removing", rs.Primary.ID, err)
			}

			return fmt.Errorf("intake runner with ID %s still existed", rs.Primary.ID)
		}
		var oapiErr *oapierror.GenericOpenAPIError
		if !errors.As(err, &oapiErr) || oapiErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("expected 404 not found, got error: %w", err)
		}
	}

	return nil
}
