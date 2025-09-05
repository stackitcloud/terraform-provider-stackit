package serverbackup_test

import (
	"context"
	_ "embed"
	"fmt"
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
	"github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-min.tf
	resourceMinConfig string

	//go:embed testdata/resource-max.tf
	resourceMaxConfig string
)

var testConfigVarsMin = config.Variables{
	"project_id":       config.StringVariable(testutil.ProjectId),
	"server_id":        config.StringVariable(testutil.ServerId),
	"schedule_name":    config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"rrule":            config.StringVariable("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
	"enabled":          config.BoolVariable(true),
	"backup_name":      config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"retention_period": config.IntegerVariable(14),
}

var testConfigVarsMax = config.Variables{
	"project_id":       config.StringVariable(testutil.ProjectId),
	"server_id":        config.StringVariable(testutil.ServerId),
	"schedule_name":    config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"rrule":            config.StringVariable("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
	"enabled":          config.BoolVariable(true),
	"backup_name":      config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"retention_period": config.IntegerVariable(14),
	"region":           config.StringVariable("eu01"),
}

func configVarsInvalid(vars config.Variables) config.Variables {
	tempConfig := maps.Clone(vars)
	tempConfig["retention_period"] = config.IntegerVariable(0)
	return tempConfig
}

func configVarsMinUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMin)
	tempConfig["retention_period"] = config.IntegerVariable(12)
	tempConfig["rrule"] = config.StringVariable("DTSTART;TZID=Europe/Berlin:20250430T010000 RRULE:FREQ=DAILY;INTERVAL=3")

	return tempConfig
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMax)
	tempConfig["retention_period"] = config.IntegerVariable(12)
	tempConfig["rrule"] = config.StringVariable("DTSTART;TZID=Europe/Berlin:20250430T010000 RRULE:FREQ=DAILY;INTERVAL=3")
	return tempConfig
}

func TestAccServerBackupScheduleMinResource(t *testing.T) {
	if testutil.ServerId == "" {
		fmt.Println("TF_ACC_SERVER_ID not set, skipping test")
		return
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerBackupScheduleDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.ServerBackupProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: configVarsInvalid(testConfigVarsMin),
				ExpectError:     regexp.MustCompile(`.*backup_properties.retention_period value must be at least 1*`),
			},
			// Creation
			{
				Config:          testutil.ServerBackupProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Backup schedule data
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "server_id", testutil.ConvertConfigVariable(testConfigVarsMin["server_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "backup_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "name", testutil.ConvertConfigVariable(testConfigVarsMin["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(testConfigVarsMin["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "enabled", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "backup_properties.name", testutil.ConvertConfigVariable(testConfigVarsMin["backup_name"])),
				),
			},
			// data source
			{
				Config:          testutil.ServerBackupProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server backup schedule data
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "server_id", testutil.ConvertConfigVariable(testConfigVarsMin["server_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_backup_schedule.schedule_data_test", "backup_schedule_id"),
					resource.TestCheckResourceAttrSet("data.stackit_server_backup_schedule.schedule_data_test", "id"),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "name", testutil.ConvertConfigVariable(testConfigVarsMin["schedule_name"])),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "rrule", testutil.ConvertConfigVariable(testConfigVarsMin["rrule"])),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "enabled", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "backup_properties.name", testutil.ConvertConfigVariable(testConfigVarsMin["backup_name"])),

					// Server backup schedules data
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedules.schedules_data_test", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedules.schedules_data_test", "server_id", testutil.ConvertConfigVariable(testConfigVarsMin["server_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_backup_schedules.schedules_data_test", "id"),
				),
			},
			// Import
			{
				ResourceName:    "stackit_server_backup_schedule.test_schedule",
				ConfigVariables: testConfigVarsMin,
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
				Config:          testutil.ServerBackupProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: configVarsMinUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Backup schedule data
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "server_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["server_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "backup_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "name", testutil.ConvertConfigVariable(configVarsMinUpdated()["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(configVarsMinUpdated()["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "enabled", testutil.ConvertConfigVariable(configVarsMinUpdated()["enabled"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "backup_properties.retention_period", testutil.ConvertConfigVariable(configVarsMinUpdated()["retention_period"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "backup_properties.name", testutil.ConvertConfigVariable(configVarsMinUpdated()["backup_name"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccServerBackupScheduleMaxResource(t *testing.T) {
	if testutil.ServerId == "" {
		fmt.Println("TF_ACC_SERVER_ID not set, skipping test")
		return
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerBackupScheduleDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.ServerBackupProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: configVarsInvalid(testConfigVarsMax),
				ExpectError:     regexp.MustCompile(`.*backup_properties.retention_period value must be at least 1*`),
			},
			// Creation
			{
				Config:          testutil.ServerBackupProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Backup schedule data
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "server_id", testutil.ConvertConfigVariable(testConfigVarsMax["server_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "backup_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "name", testutil.ConvertConfigVariable(testConfigVarsMax["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(testConfigVarsMax["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "enabled", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "backup_properties.name", testutil.ConvertConfigVariable(testConfigVarsMax["backup_name"])),
				),
			},
			// data source
			{
				Config:          testutil.ServerBackupProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server backup schedule data
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "server_id", testutil.ConvertConfigVariable(testConfigVarsMax["server_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_backup_schedule.schedule_data_test", "backup_schedule_id"),
					resource.TestCheckResourceAttrSet("data.stackit_server_backup_schedule.schedule_data_test", "id"),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "name", testutil.ConvertConfigVariable(testConfigVarsMax["schedule_name"])),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "rrule", testutil.ConvertConfigVariable(testConfigVarsMax["rrule"])),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "enabled", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedule.schedule_data_test", "backup_properties.name", testutil.ConvertConfigVariable(testConfigVarsMax["backup_name"])),

					// Server backup schedules data
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedules.schedules_data_test", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_server_backup_schedules.schedules_data_test", "server_id", testutil.ConvertConfigVariable(testConfigVarsMax["server_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_backup_schedules.schedules_data_test", "id"),
				),
			},
			// Import
			{
				ResourceName:    "stackit_server_backup_schedule.test_schedule",
				ConfigVariables: testConfigVarsMax,
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
				Config:          testutil.ServerBackupProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: configVarsMaxUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Backup schedule data
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "server_id", testutil.ConvertConfigVariable(configVarsMaxUpdated()["server_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "backup_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_backup_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(configVarsMaxUpdated()["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "enabled", testutil.ConvertConfigVariable(configVarsMaxUpdated()["enabled"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "backup_properties.retention_period", testutil.ConvertConfigVariable(configVarsMaxUpdated()["retention_period"])),
					resource.TestCheckResourceAttr("stackit_server_backup_schedule.test_schedule", "backup_properties.name", testutil.ConvertConfigVariable(configVarsMaxUpdated()["backup_name"])),
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
			core_config.WithEndpoint(testutil.ServerBackupCustomEndpoint),
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
