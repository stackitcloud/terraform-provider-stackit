package serverupdate_test

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"maps"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	core_config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serverupdate"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-min.tf
	resourceMinConfig string

	//go:embed testdata/resource-max.tf
	resourceMaxConfig string
)

func unwrap(v config.Variable) string {
	tmp, err := v.MarshalJSON()
	if err != nil {
		log.Panicf("cannot marshal variable %v: %v", v, err)
	}
	return strings.Trim(string(tmp), `"`)
}

var testConfigVarsMin = config.Variables{
	"project_id":         config.StringVariable(testutil.ProjectId),
	"server_name":        config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"schedule_name":      config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"rrule":              config.StringVariable("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
	"enabled":            config.BoolVariable(true),
	"maintenance_window": config.IntegerVariable(1),
	"server_id":          config.StringVariable(testutil.ServerId),
}

var testConfigVarsMax = config.Variables{
	"project_id":         config.StringVariable(testutil.ProjectId),
	"server_name":        config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"schedule_name":      config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"rrule":              config.StringVariable("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
	"enabled":            config.BoolVariable(true),
	"maintenance_window": config.IntegerVariable(1),
	"region":             config.StringVariable("eu01"),
	"server_id":          config.StringVariable(testutil.ServerId),
}

func configVarsInvalid(vars config.Variables) config.Variables {
	tempConfig := maps.Clone(vars)
	tempConfig["maintenance_window"] = config.IntegerVariable(0)
	return tempConfig
}

func configVarsMinUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMin)
	tempConfig["maintenance_window"] = config.IntegerVariable(12)
	tempConfig["rrule"] = config.StringVariable("DTSTART;TZID=Europe/Berlin:20250430T010000 RRULE:FREQ=DAILY;INTERVAL=3")

	return tempConfig
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMax)
	tempConfig["maintenance_window"] = config.IntegerVariable(12)
	tempConfig["rrule"] = config.StringVariable("DTSTART;TZID=Europe/Berlin:20250430T010000 RRULE:FREQ=DAILY;INTERVAL=3")
	return tempConfig
}

func TestAccServerUpdateScheduleMinResource(t *testing.T) {
	if testutil.ServerId == "" {
		fmt.Println("TF_ACC_SERVER_ID not set, skipping test")
		return
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerUpdateScheduleDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.ServerUpdateProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: configVarsInvalid(configVarsMinUpdated()),
				ExpectError:     regexp.MustCompile(`.*maintenance_window value must be at least 1*`),
			},
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", unwrap(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "server_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "name", unwrap(testConfigVarsMin["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", unwrap(testConfigVarsMin["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", unwrap(testConfigVarsMin["enabled"])),
				),
			},
			// data source
			{
				Config:          testutil.ServerUpdateProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server update schedule data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "project_id", unwrap(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "name", unwrap(testConfigVarsMin["schedule_name"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "rrule", unwrap(testConfigVarsMin["rrule"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "enabled", unwrap(testConfigVarsMin["enabled"])),

					// Server update schedules data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedules.schedules_data_test", "project_id", unwrap(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedules.schedules_data_test", "server_id", unwrap(testConfigVarsMin["server_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedules.schedules_data_test", "id"),
				),
			},
			// // Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_server_update_schedule.test_schedule",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server_update_schedule.test_schedule"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server_update_schedule.test_schedule")
					}
					scheduleId, ok := r.Primary.Attributes["update_schedule_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute update_schedule_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, testutil.ServerId, scheduleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// // Update
			{
				ConfigVariables: configVarsMinUpdated(),
				Config:          testutil.ServerUpdateProviderConfig() + "\n" + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", unwrap(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "name", unwrap(configVarsMinUpdated()["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", unwrap(configVarsMinUpdated()["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", unwrap(configVarsMinUpdated()["enabled"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "maintenance_window", unwrap(configVarsMinUpdated()["maintenance_window"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccServerUpdateScheduleMaxResource(t *testing.T) {
	if testutil.ServerId == "" {
		fmt.Println("TF_ACC_SERVER_ID not set, skipping test")
		return
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerUpdateScheduleDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.ServerUpdateProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: configVarsInvalid(testConfigVarsMax),
				ExpectError:     regexp.MustCompile(`.*maintenance_window value must be at least 1*`),
			},
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", unwrap(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "server_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "name", unwrap(testConfigVarsMax["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", unwrap(testConfigVarsMax["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", unwrap(testConfigVarsMax["enabled"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "region", testutil.Region),
				),
			},
			// data source
			{
				Config:          testutil.ServerUpdateProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server update schedule data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "project_id", unwrap(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "name", unwrap(testConfigVarsMax["schedule_name"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "rrule", unwrap(testConfigVarsMax["rrule"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "enabled", unwrap(testConfigVarsMax["enabled"])),

					// Server update schedules data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedules.schedules_data_test", "project_id", unwrap(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedules.schedules_data_test", "server_id", unwrap(testConfigVarsMax["server_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedules.schedules_data_test", "id"),
				),
			},
			// // Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_server_update_schedule.test_schedule",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server_update_schedule.test_schedule"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server_update_schedule.test_schedule")
					}
					scheduleId, ok := r.Primary.Attributes["update_schedule_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute update_schedule_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, testutil.ServerId, scheduleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// // Update
			{
				ConfigVariables: configVarsMaxUpdated(),
				Config:          testutil.ServerUpdateProviderConfig() + "\n" + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", unwrap(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", unwrap(configVarsMinUpdated()["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", unwrap(configVarsMinUpdated()["enabled"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "maintenance_window", unwrap(configVarsMinUpdated()["maintenance_window"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "region", testutil.Region),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckServerUpdateScheduleDestroy(s *terraform.State) error {
	ctx := context.Background()
	if err := deleteSchedule(ctx, s); err != nil {
		log.Printf("cannot delete schedule: %v", err)
	}

	return nil
}

func deleteSchedule(ctx context.Context, s *terraform.State) error {
	var client *serverupdate.APIClient
	var err error
	if testutil.ServerUpdateCustomEndpoint == "" {
		client, err = serverupdate.NewAPIClient()
	} else {
		client, err = serverupdate.NewAPIClient(
			core_config.WithEndpoint(testutil.ServerUpdateCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	schedulesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_server_update_schedule" {
			continue
		}
		// server update schedule terraform ID: "[project_id],[server_id],[update_schedule_id]"
		scheduleId := strings.Split(rs.Primary.ID, core.Separator)[2]
		schedulesToDestroy = append(schedulesToDestroy, scheduleId)
	}

	schedulesResp, err := client.ListUpdateSchedules(ctx, testutil.ProjectId, testutil.ServerId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting schedulesResp: %w", err)
	}

	schedules := *schedulesResp.Items
	for i := range schedules {
		if schedules[i].Id == nil {
			continue
		}
		scheduleId := strconv.FormatInt(*schedules[i].Id, 10)
		if utils.Contains(schedulesToDestroy, scheduleId) {
			err := client.DeleteUpdateScheduleExecute(ctx, testutil.ProjectId, testutil.ServerId, scheduleId, testutil.Region)
			if err != nil {
				return fmt.Errorf("destroying server update schedule %s during CheckDestroy: %w", scheduleId, err)
			}
		}
	}
	return nil
}
