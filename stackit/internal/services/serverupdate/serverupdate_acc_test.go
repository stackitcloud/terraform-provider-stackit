package serverupdate_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serverupdate"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Server update schedule resource data
var serverUpdateScheduleResource = map[string]string{
	"project_id":         testutil.ProjectId,
	"server_id":          testutil.ServerId,
	"name":               testutil.ResourceNameWithDateTime("server-update-schedule"),
	"rrule":              "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1",
	"maintenance_window": "1",
}

func resourceConfig(maintenanceWindow int64, region *string) string {
	var regionConfig string
	if region != nil {
		regionConfig = fmt.Sprintf(`region = %q`, *region)
	}
	return fmt.Sprintf(`
				%s

				resource "stackit_server_update_schedule" "test_schedule" {
					project_id = "%s"
					server_id  = "%s"
					name  = "%s"
 				 	rrule = "%s"
                    enabled = true
                    maintenance_window = %d
					%s
				}
				`,
		testutil.ServerUpdateProviderConfig(),
		serverUpdateScheduleResource["project_id"],
		serverUpdateScheduleResource["server_id"],
		serverUpdateScheduleResource["name"],
		serverUpdateScheduleResource["rrule"],
		maintenanceWindow,
		regionConfig,
	)
}

func TestAccServerUpdateScheduleResource(t *testing.T) {
	if testutil.ServerId == "" {
		fmt.Println("TF_ACC_SERVER_ID not set, skipping test")
		return
	}
	var invalidMaintenanceWindow int64 = 0
	var validMaintenanceWindow int64 = 15
	var updatedMaintenanceWindow int64 = 8
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerUpdateScheduleDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:      resourceConfig(invalidMaintenanceWindow, &testutil.Region),
				ExpectError: regexp.MustCompile(`.*maintenance_window value must be at least 1*`),
			},
			// Creation
			{
				Config: resourceConfig(validMaintenanceWindow, &testutil.Region),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", serverUpdateScheduleResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "server_id", serverUpdateScheduleResource["server_id"]),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "name", serverUpdateScheduleResource["name"]),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", serverUpdateScheduleResource["rrule"]),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "region", testutil.Region),
				),
			},
			// data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_server_update_schedules" "schedules_data_test" {
						project_id  = stackit_server_update_schedule.test_schedule.project_id
						server_id  = stackit_server_update_schedule.test_schedule.server_id
					}

					data "stackit_server_update_schedule" "schedule_data_test" {
						project_id  = stackit_server_update_schedule.test_schedule.project_id
						server_id  = stackit_server_update_schedule.test_schedule.server_id
                        update_schedule_id = stackit_server_update_schedule.test_schedule.update_schedule_id
					}`,
					resourceConfig(validMaintenanceWindow, &testutil.Region),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server update schedule data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.schedule_data_test", "project_id", serverUpdateScheduleResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.schedule_data_test", "server_id", serverUpdateScheduleResource["server_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.schedule_data_test", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.schedule_data_test", "id"),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.schedule_data_test", "name", serverUpdateScheduleResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.schedule_data_test", "rrule", serverUpdateScheduleResource["rrule"]),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.schedule_data_test", "enabled", strconv.FormatBool(true)),

					// Server update schedules data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedules.schedules_data_test", "project_id", serverUpdateScheduleResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedules.schedules_data_test", "server_id", serverUpdateScheduleResource["server_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedules.schedules_data_test", "id"),
				),
			},
			// Import
			{
				ResourceName: "stackit_server_update_schedule.test_schedule",
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
			// Update
			{
				Config: resourceConfig(updatedMaintenanceWindow, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", serverUpdateScheduleResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "server_id", serverUpdateScheduleResource["server_id"]),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "name", serverUpdateScheduleResource["name"]),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", serverUpdateScheduleResource["rrule"]),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "maintenance_window", strconv.FormatInt(8, 10)),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckServerUpdateScheduleDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *serverupdate.APIClient
	var err error
	if testutil.ServerUpdateCustomEndpoint == "" {
		client, err = serverupdate.NewAPIClient()
	} else {
		client, err = serverupdate.NewAPIClient(
			config.WithEndpoint(testutil.ServerUpdateCustomEndpoint),
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
