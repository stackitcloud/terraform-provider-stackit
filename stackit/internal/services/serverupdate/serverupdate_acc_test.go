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
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/serverupdate"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-min.tf
	resourceMinConfig string

	//go:embed testdata/resource-max.tf
	resourceMaxConfig string

	//go:embed testdata/datasource.tf
	datasourceConfig string
)

var testConfigVarsMin = config.Variables{
	"project_id":         config.StringVariable(testutil.ProjectId),
	"schedule_name":      config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"rrule":              config.StringVariable("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
	"enabled":            config.BoolVariable(true),
	"maintenance_window": config.IntegerVariable(1),
	"server_name":        config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)),
	"network_name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)),
	"machine_type":       config.StringVariable("t1.1"),
	// image needs to contain the STACKIT Server Agent
	"image_id": config.StringVariable("fb5b3fa8-5e20-478a-929a-2b7da1676b18"),
}

var testConfigVarsMax = config.Variables{
	"project_id":         config.StringVariable(testutil.ProjectId),
	"schedule_name":      config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"rrule":              config.StringVariable("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
	"enabled":            config.BoolVariable(true),
	"maintenance_window": config.IntegerVariable(1),
	"region":             config.StringVariable("eu01"),
	"server_name":        config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)),
	"network_name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)),
	"machine_type":       config.StringVariable("t1.1"),
	// image needs to contain the STACKIT Server Agent
	"image_id": config.StringVariable("fb5b3fa8-5e20-478a-929a-2b7da1676b18"),
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
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckServerUpdateScheduleDestroy,
			testAccCheckServerDestroy,
		),
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: configVarsInvalid(configVarsMinUpdated()),
				ExpectError:     regexp.MustCompile(`.*maintenance_window value must be at least 1*`),
			},
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "name", testutil.ConvertConfigVariable(testConfigVarsMin["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(testConfigVarsMin["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", testutil.ConvertConfigVariable(testConfigVarsMin["enabled"])),

					// server
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "server_id"),

					// enable
					resource.TestCheckResourceAttrSet("stackit_server_update_enable.enable", "server_id"),
					resource.TestCheckResourceAttr("stackit_server_update_enable.enable", "enabled", "true"),
				),
			},
			// data source
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig + "\n" + datasourceConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server update schedule data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "name", testutil.ConvertConfigVariable(testConfigVarsMin["schedule_name"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(testConfigVarsMin["rrule"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "enabled", testutil.ConvertConfigVariable(testConfigVarsMin["enabled"])),

					// Server update schedules data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedules.schedules_data_test", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedules.schedules_data_test", "id"),

					// server
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedules.schedules_data_test", "server_id"),

					// enable
					resource.TestCheckResourceAttrSet("data.stackit_server_update_enable.enable_test", "server_id"),
					resource.TestCheckResourceAttr("data.stackit_server_update_enable.enable_test", "enabled", "true"),
				),
			},
			// Import
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
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute server_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, serverId, scheduleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: configVarsMinUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "name", testutil.ConvertConfigVariable(configVarsMinUpdated()["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(configVarsMinUpdated()["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", testutil.ConvertConfigVariable(configVarsMinUpdated()["enabled"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "maintenance_window", testutil.ConvertConfigVariable(configVarsMinUpdated()["maintenance_window"])),

					// server
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "server_id"),

					// enable
					resource.TestCheckResourceAttrSet("stackit_server_update_enable.enable", "server_id"),
					resource.TestCheckResourceAttr("stackit_server_update_enable.enable", "enabled", "true"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccServerUpdateScheduleMaxResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckServerUpdateScheduleDestroy,
			testAccCheckServerDestroy,
		),
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: configVarsInvalid(testConfigVarsMax),
				ExpectError:     regexp.MustCompile(`.*maintenance_window value must be at least 1*`),
			},
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "server_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "name", testutil.ConvertConfigVariable(testConfigVarsMax["schedule_name"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(testConfigVarsMax["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", testutil.ConvertConfigVariable(testConfigVarsMax["enabled"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "region", testutil.Region),

					// server
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "server_id"),

					// enable
					resource.TestCheckResourceAttrSet("stackit_server_update_enable.enable", "server_id"),
					resource.TestCheckResourceAttr("stackit_server_update_enable.enable", "enabled", "true"),
				),
			},
			// data source
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMaxConfig + "\n" + datasourceConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server update schedule data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "name", testutil.ConvertConfigVariable(testConfigVarsMax["schedule_name"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(testConfigVarsMax["rrule"])),
					resource.TestCheckResourceAttr("data.stackit_server_update_schedule.test_schedule", "enabled", testutil.ConvertConfigVariable(testConfigVarsMax["enabled"])),

					// Server update schedules data
					resource.TestCheckResourceAttr("data.stackit_server_update_schedules.schedules_data_test", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedules.schedules_data_test", "id"),

					// server
					resource.TestCheckResourceAttrSet("data.stackit_server_update_schedules.schedules_data_test", "server_id"),

					// enable
					resource.TestCheckResourceAttrSet("data.stackit_server_update_enable.enable_test", "server_id"),
					resource.TestCheckResourceAttr("data.stackit_server_update_enable.enable_test", "enabled", "true"),
				),
			},
			// Import
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
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute server_id")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, serverId, scheduleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: configVarsMaxUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Update schedule data
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "project_id", testutil.ConvertConfigVariable(configVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "update_schedule_id"),
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "rrule", testutil.ConvertConfigVariable(configVarsMinUpdated()["rrule"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "enabled", testutil.ConvertConfigVariable(configVarsMinUpdated()["enabled"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "maintenance_window", testutil.ConvertConfigVariable(configVarsMinUpdated()["maintenance_window"])),
					resource.TestCheckResourceAttr("stackit_server_update_schedule.test_schedule", "region", testutil.Region),

					// server
					resource.TestCheckResourceAttrSet("stackit_server_update_schedule.test_schedule", "server_id"),

					// enable
					resource.TestCheckResourceAttrSet("stackit_server_update_enable.enable", "server_id"),
					resource.TestCheckResourceAttr("stackit_server_update_enable.enable", "enabled", "true"),
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
	client, err := serverupdate.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.ServerUpdateCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var serverId string
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "stackit_server" {
			// server terraform ID: "[project_id],[region],[server_id]"
			serverId = strings.Split(rs.Primary.ID, core.Separator)[2]
			break
		}
	}

	if serverId == "" {
		return fmt.Errorf("could not find server ID in state")
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

	schedulesResp, err := client.ListUpdateSchedules(ctx, testutil.ProjectId, serverId, testutil.Region).Execute()
	// The destroy functions are called after all resources are cleaned up.
	// If the server was successfully destroyed we should see a 404 here.
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Server not found") {
			return nil
		}
		return fmt.Errorf("getting schedulesResp: %w", err)
	}

	schedules := *schedulesResp.Items
	for i := range schedules {
		if schedules[i].Id == nil {
			continue
		}
		scheduleId := strconv.FormatInt(*schedules[i].Id, 10)
		if utils.Contains(schedulesToDestroy, scheduleId) {
			err := client.DeleteUpdateScheduleExecute(ctx, testutil.ProjectId, serverId, scheduleId, testutil.Region)
			if err != nil {
				return fmt.Errorf("destroying server update schedule %s during CheckDestroy: %w", scheduleId, err)
			}
		}
	}
	return nil
}

// Additional function to check if the server was deleted if something went wrong in the first case.
func testAccCheckServerDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := iaas.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.IaaSCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	serversToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_server" {
			continue
		}
		// server terraform ID: "[project_id],[region],[server_id]"
		serverId := strings.Split(rs.Primary.ID, core.Separator)[2]
		serversToDestroy = append(serversToDestroy, serverId)
	}

	serversResp, err := client.ListServersExecute(ctx, testutil.ProjectId, testutil.Region)
	if err != nil {
		return fmt.Errorf("getting serversResp: %w", err)
	}

	servers := *serversResp.Items
	for i := range servers {
		if servers[i].Id == nil {
			continue
		}
		if utils.Contains(serversToDestroy, *servers[i].Id) {
			err := client.DeleteServerExecute(ctx, testutil.ProjectId, testutil.Region, *servers[i].Id)
			if err != nil {
				return fmt.Errorf("destroying server %s during CheckDestroy: %w", *servers[i].Id, err)
			}
		}
	}
	return nil
}
