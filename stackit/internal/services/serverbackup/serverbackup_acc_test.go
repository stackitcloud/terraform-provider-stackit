package serverbackup_test

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
	"github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Server backup schedule resource data
var serverBackupScheduleResource = map[string]string{
	"project_id":           testutil.ProjectId,
	"server_id":            testutil.ServerId,
	"backup_schedule_name": testutil.ResourceNameWithDateTime("server-backup-schedule"),
	"rrule":                "DTSTART;TZID=Europe/Berlin:20250325T080000 RRULE:FREQ=DAILY;INTERVAL=1;COUNT=3",
	"backup_name":          testutil.ResourceNameWithDateTime("server-backup-schedule-backup"),
}

func resourceConfig(retentionPeriod int64) string {
	return fmt.Sprintf(`
				%s

				resource "stackit_server_backup_schedule" "test_schedule" {
					project_id = "%s"
					server_id  = "%s"
					name  = "%s"
 				 	rrule = "%s"
                    enabled = true
                    backup_properties = {
                        name = "%s"
                        retention_period = %d
                        volume_ids = null
                    }
				}
				`,
		testutil.ServerBackupProviderConfig(),
		serverBackupScheduleResource["project_id"],
		serverBackupScheduleResource["server_id"],
		serverBackupScheduleResource["backup_schedule_name"],
		serverBackupScheduleResource["rrule"],
		serverBackupScheduleResource["backup_name"],
		retentionPeriod,
	)
}

func resourceConfigWithUpdate() string {
	return fmt.Sprintf(`
				%s

				resource "stackit_server_backup_schedule" "test_schedule" {
					project_id = "%s"
					server_id  = "%s"
					name  = "%s"
 				 	rrule = "%s"
                    enabled = false
                    backup_properties = {
                        name = "%s"
                        retention_period = 20 
                        volume_ids = null
                    }
				}
				`,
		testutil.ServerBackupProviderConfig(),
		serverBackupScheduleResource["project_id"],
		serverBackupScheduleResource["server_id"],
		serverBackupScheduleResource["backup_schedule_name"],
		serverBackupScheduleResource["rrule"],
		serverBackupScheduleResource["backup_name"],
	)
}

func TestAccServerBackupScheduleResource(t *testing.T) {
	if testutil.ServerId == "" {
		fmt.Println("TF_ACC_SERVER_ID not set, skipping test")
		return
	}
	var invalidRetentionPeriod int64 = 0
	var validRetentionPeriod int64 = 15
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerBackupScheduleDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:      resourceConfig(invalidRetentionPeriod),
				ExpectError: regexp.MustCompile(`.*backup_properties.retention_period value must be at least 1*`),
			},
			// Creation
			{
				Config: resourceConfig(validRetentionPeriod),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Backup schedule data
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "project_id", serverBackupScheduleResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "server_id", serverBackupScheduleResource["server_id"]),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "backup_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "name", serverBackupScheduleResource["backup_schedule_name"]),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "rrule", serverBackupScheduleResource["rrule"]),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "enabled", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "backup_properties.name", serverBackupScheduleResource["backup_name"]),
				),
			},
			// data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_server_backup_schedules" "schedules_data_test" {
						project_id  = stackit_server_backup_schedule.test_schedule.project_id
						server_id  = stackit_server_backup_schedule.test_schedule.server_id
						region = stackit_server_backup_schedule.test_schedule.region
					}

					data "stackit_server_backup_schedule" "schedule_data_test" {
						project_id  = stackit_server_backup_schedule.test_schedule.project_id
						server_id  = stackit_server_backup_schedule.test_schedule.server_id
                        backup_schedule_id = stackit_server_backup_schedule.test_schedule.backup_schedule_id
						region = stackit_server_backup_schedule.test_schedule.region
					}`,
					resourceConfig(validRetentionPeriod),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server backup schedule data
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "project_id", serverBackupScheduleResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "server_id", serverBackupScheduleResource["server_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_server_backup_schedule.schedule_data_test", "backup_schedule_id"),
					resource.TestCheckResourceAttrSet("data.stackit_server_backup_schedule.schedule_data_test", "id"),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "name", serverBackupScheduleResource["backup_schedule_name"]),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "rrule", serverBackupScheduleResource["rrule"]),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "enabled", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "backup_properties.name", serverBackupScheduleResource["backup_name"]),

					// Server backup schedules data
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedules.schedules_data_test", "project_id", serverBackupScheduleResource["project_id"]),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedules.schedules_data_test", "server_id", serverBackupScheduleResource["server_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_server_backup_schedules.schedules_data_test", "id"),
				),
			},
			// Import
			{
				ResourceName: "stackit_server_backup_schedule.test_schedule",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server_backup_schedule.test_schedule"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server_backup_schedule.test_schedule")
					}
					scheduleId, ok := r.Primary.Attributes["backup_schedule_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute backup_schedule_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, testutil.ServerId, scheduleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: resourceConfigWithUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Backup schedule data
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "project_id", serverBackupScheduleResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "server_id", serverBackupScheduleResource["server_id"]),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "backup_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "name", serverBackupScheduleResource["backup_schedule_name"]),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "rrule", serverBackupScheduleResource["rrule"]),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "enabled", strconv.FormatBool(false)),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "backup_properties.retention_period", strconv.FormatInt(20, 10)),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "backup_properties.name", serverBackupScheduleResource["backup_name"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckServerBackupScheduleDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *serverbackup.APIClient
	var err error
	if testutil.ServerBackupCustomEndpoint == "" {
		client, err = serverbackup.NewAPIClient()
	} else {
		client, err = serverbackup.NewAPIClient(
			config.WithEndpoint(testutil.ServerBackupCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	schedulesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_server_backup_schedule" {
			continue
		}
		// server backup schedule terraform ID: "[project_id],[server_id],[backup_schedule_id]"
		scheduleId := strings.Split(rs.Primary.ID, core.Separator)[3]
		schedulesToDestroy = append(schedulesToDestroy, scheduleId)
	}

	schedulesResp, err := client.ListBackupSchedules(ctx, testutil.ProjectId, testutil.ServerId, testutil.Region).Execute()
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
			err := client.DeleteBackupScheduleExecute(ctx, testutil.ProjectId, testutil.ServerId, scheduleId, testutil.Region)
			if err != nil {
				return fmt.Errorf("destroying server backup schedule %s during CheckDestroy: %w", scheduleId, err)
			}
		}
	}
	return nil
}
