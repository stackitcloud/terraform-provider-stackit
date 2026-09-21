package automation_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	automation "github.com/stackitcloud/stackit-sdk-go/services/automation/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/datasource-templates.tf
	templatesDataSourceConfig string

	//go:embed testdata/resource-min.tf
	resourceMinConfig string

	//go:embed testdata/resource-max.tf
	resourceMaxConfig string

	//go:embed testdata/resource-max-cleared.tf
	resourceMaxClearedConfig string

	//go:embed testdata/datasource.tf
	datasourceConfig string
)

// lookupVolumeRecoveryPointManagementTemplateID looks up the system-provided automation
// template of kind "VolumeRecoveryPointManagement". Volume automation templates are not
// created via Terraform, so acceptance tests have to discover a valid template_id at runtime.
func lookupVolumeRecoveryPointManagementTemplateID(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	client, err := automation.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.AutomationCustomEndpoint, false)...)
	if err != nil {
		t.Fatalf("creating automation client: %v", err)
	}

	templatesResp, err := client.DefaultAPI.ListVolumeTemplates(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		t.Fatalf("listing volume templates: %v", err)
	}

	for _, tpl := range templatesResp.Items {
		templateResp, err := client.DefaultAPI.GetVolumeTemplate(ctx, testutil.ProjectId, testutil.Region, tpl.Id).Execute()
		if err != nil {
			t.Fatalf("getting volume template %q: %v", tpl.Id, err)
		}
		if templateResp.Input != nil && templateResp.Input.Kind == automation.VOLUMETEMPLATEAUTOMATIONINPUTKIND_VOLUME_RECOVERY_POINT_MANAGEMENT {
			return tpl.Id
		}
	}
	t.Fatal("no volume automation template of kind VolumeRecoveryPointManagement found")
	return ""
}

// futureRrule builds an rrule with a DTSTART relative to now, instead of a hardcoded date that
// would eventually be in the past.
func futureRrule(afterDuration time.Duration, intervalDays int) string {
	dtstart := time.Now().UTC().Add(afterDuration).Format("20060102T150405")
	return fmt.Sprintf("DTSTART;TZID=UTC:%s RRULE:FREQ=DAILY;INTERVAL=%d", dtstart, intervalDays)
}

func testConfigVarsMin(templateID string) config.Variables {
	// the input variable needs a json decoded string
	inputMap := map[string]interface{}{
		"kind": "VolumeRecoveryPointManagement",
		"snapshotRetentionPolicy": map[string]interface{}{
			"kind": "indefinitely",
		},
	}
	inputJson, err := json.Marshal(inputMap)
	if err != nil {
		return nil
	}

	return config.Variables{
		"project_id":  config.StringVariable(testutil.ProjectId),
		"template_id": config.StringVariable(templateID),
		"name":        config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"rrule":       config.StringVariable(futureRrule(time.Hour, 1)),
		"input":       config.StringVariable(string(inputJson)),
	}
}

func testConfigVarsMax(templateID string) config.Variables {
	// the input variable needs a json decoded string
	inputMap := map[string]interface{}{
		"kind":                "VolumeRecoveryPointManagement",
		"inheritVolumeLabels": true,
		"recoveryPointLabels": map[string]interface{}{
			"created-by": "tf-acc-test",
		},
		"snapshotRetentionPolicy": map[string]interface{}{
			"kind":  "count",
			"value": 4,
		},
	}
	inputJson, err := json.Marshal(inputMap)
	if err != nil {
		return nil
	}

	return config.Variables{
		"project_id":  config.StringVariable(testutil.ProjectId),
		"region":      config.StringVariable(testutil.Region),
		"template_id": config.StringVariable(templateID),
		"name":        config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"description": config.StringVariable("tf-acc-test description"),
		"rrule":       config.StringVariable(futureRrule(time.Hour, 1)),
		"input":       config.StringVariable(string(inputJson)),
	}
}

func configVarsMinUpdated(base config.Variables) config.Variables {
	tempConfig := maps.Clone(base)
	tempConfig["name"] = config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	tempConfig["rrule"] = config.StringVariable(futureRrule(2*time.Hour, 3))
	return tempConfig
}

func configVarsMaxUpdated(base config.Variables) config.Variables {
	// the input variable needs a json decoded string
	inputMap := map[string]interface{}{
		"kind": "VolumeRecoveryPointManagement",
		"recoveryPointLabels": map[string]interface{}{
			"created-by": "tf-acc-test-updated",
		},
		"snapshotRetentionPolicy": map[string]interface{}{
			"kind": "indefinitely",
		},
	}
	inputJson, err := json.Marshal(inputMap)
	if err != nil {
		return nil
	}

	tempConfig := maps.Clone(base)
	tempConfig["description"] = config.StringVariable("tf-acc-test description updated")
	tempConfig["retention_count"] = config.IntegerVariable(5)
	tempConfig["input"] = config.StringVariable(string(inputJson))
	return tempConfig
}

func TestAccAutomationTemplatesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigVariables: config.Variables{
					"project_id": config.StringVariable(testutil.ProjectId),
				},
				Config: testutil.NewConfigBuilder().Region(testutil.Region).EnableBetaResources(true).BuildProviderConfig() + "\n" + templatesDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_automation_templates.templates", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_automation_templates.templates", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("data.stackit_automation_templates.templates", "id"),
					resource.TestCheckResourceAttrSet("data.stackit_automation_templates.templates", "templates.0.template_id"),
					resource.TestCheckResourceAttrSet("data.stackit_automation_templates.templates", "templates.0.name"),
					resource.TestCheckResourceAttrSet("data.stackit_automation_templates.templates", "templates.0.description"),
					resource.TestCheckResourceAttrSet("data.stackit_automation_templates.templates", "templates.0.create_time"),
				),
			},
		},
	})
}

func TestAccVolumeAutomationMinResource(t *testing.T) {
	varsMin := testConfigVarsMin(lookupVolumeRecoveryPointManagementTemplateID(t))
	varsMinUpdated := configVarsMinUpdated(varsMin)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVolumeAutomationDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: varsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "project_id", testutil.ConvertConfigVariable(varsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "region"),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "template_id", testutil.ConvertConfigVariable(varsMin["template_id"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "name", testutil.ConvertConfigVariable(varsMin["name"])),
					resource.TestCheckNoResourceAttr("stackit_volume_automation.test", "description"),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "input", testutil.ConvertConfigVariable(varsMin["input"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "triggers.schedule.rrule", testutil.ConvertConfigVariable(varsMin["rrule"])),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "automation_id"),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "id"),
				),
			},
			// data source
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig + "\n" + datasourceConfig,
				ConfigVariables: varsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "project_id", testutil.ConvertConfigVariable(varsMin["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_volume_automation.test_data", "region"),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "template_id", testutil.ConvertConfigVariable(varsMin["template_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_volume_automation.test_data", "automation_id"),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "name", testutil.ConvertConfigVariable(varsMin["name"])),
					resource.TestCheckNoResourceAttr("data.stackit_volume_automation.test_data", "description"),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "input", testutil.ConvertConfigVariable(varsMin["input"])),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "triggers.schedule.rrule", testutil.ConvertConfigVariable(varsMin["rrule"])),
					resource.TestCheckResourceAttrSet("data.stackit_volume_automation.test_data", "id"),
				),
			},
			// Import
			{
				ResourceName:    "stackit_volume_automation.test",
				ConfigVariables: varsMin,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_volume_automation.test"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_volume_automation.test")
					}
					automationId, ok := r.Primary.Attributes["automation_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute automation_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, automationId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: varsMinUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "project_id", testutil.ConvertConfigVariable(varsMinUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "region"),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "template_id", testutil.ConvertConfigVariable(varsMinUpdated["template_id"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "name", testutil.ConvertConfigVariable(varsMinUpdated["name"])),
					resource.TestCheckNoResourceAttr("stackit_volume_automation.test", "description"),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "input", testutil.ConvertConfigVariable(varsMinUpdated["input"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "triggers.schedule.rrule", testutil.ConvertConfigVariable(varsMinUpdated["rrule"])),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "automation_id"),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccVolumeAutomationMaxResource(t *testing.T) {
	varsMax := testConfigVarsMax(lookupVolumeRecoveryPointManagementTemplateID(t))
	varsMaxUpdated := configVarsMaxUpdated(varsMax)
	varsMaxCleared := maps.Clone(varsMaxUpdated)
	delete(varsMaxCleared, "name")
	delete(varsMaxCleared, "description")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVolumeAutomationDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: varsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "project_id", testutil.ConvertConfigVariable(varsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "region", testutil.ConvertConfigVariable(varsMax["region"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "template_id", testutil.ConvertConfigVariable(varsMax["template_id"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "name", testutil.ConvertConfigVariable(varsMax["name"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "description", testutil.ConvertConfigVariable(varsMax["description"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "input", testutil.ConvertConfigVariable(varsMax["input"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "triggers.schedule.rrule", testutil.ConvertConfigVariable(varsMax["rrule"])),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "automation_id"),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "id"),
				),
			},
			// data source
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMaxConfig + "\n" + datasourceConfig,
				ConfigVariables: varsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "project_id", testutil.ConvertConfigVariable(varsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "region", testutil.ConvertConfigVariable(varsMax["region"])),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "template_id", testutil.ConvertConfigVariable(varsMax["template_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_volume_automation.test_data", "automation_id"),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "name", testutil.ConvertConfigVariable(varsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "description", testutil.ConvertConfigVariable(varsMax["description"])),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "input", testutil.ConvertConfigVariable(varsMax["input"])),
					resource.TestCheckResourceAttr("data.stackit_volume_automation.test_data", "triggers.schedule.rrule", testutil.ConvertConfigVariable(varsMax["rrule"])),
					resource.TestCheckResourceAttrSet("data.stackit_volume_automation.test_data", "id"),
				),
			},
			// Import
			{
				ResourceName:    "stackit_volume_automation.test",
				ConfigVariables: varsMax,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_volume_automation.test"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_volume_automation.test")
					}
					automationId, ok := r.Primary.Attributes["automation_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute automation_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, automationId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMaxConfig,
				ConfigVariables: varsMaxUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "project_id", testutil.ConvertConfigVariable(varsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "region", testutil.ConvertConfigVariable(varsMaxUpdated["region"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "template_id", testutil.ConvertConfigVariable(varsMaxUpdated["template_id"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "name", testutil.ConvertConfigVariable(varsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "description", testutil.ConvertConfigVariable(varsMaxUpdated["description"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "input", testutil.ConvertConfigVariable(varsMaxUpdated["input"])),
					resource.TestCheckResourceAttr("stackit_volume_automation.test", "triggers.schedule.rrule", testutil.ConvertConfigVariable(varsMaxUpdated["rrule"])),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "automation_id"),
					resource.TestCheckResourceAttrSet("stackit_volume_automation.test", "id"),
				),
			},
			// Clear optional name and description
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMaxClearedConfig,
				ConfigVariables: varsMaxCleared,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("stackit_volume_automation.test", "name"),
					resource.TestCheckNoResourceAttr("stackit_volume_automation.test", "description"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckVolumeAutomationDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := automation.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.AutomationCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating automation client: %w", err)
	}

	var errs []error
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_volume_automation" {
			continue
		}
		projectId := rs.Primary.Attributes["project_id"]
		if projectId == "" {
			continue
		}
		region := rs.Primary.Attributes["region"]
		if region == "" {
			continue
		}
		automationId := rs.Primary.Attributes["automation_id"]
		if automationId == "" {
			continue
		}

		_, err = client.DefaultAPI.GetVolumeAutomation(ctx, projectId, region, automationId).Execute()
		if err == nil {
			errs = append(errs, fmt.Errorf("volume automation %s still exists", automationId))
			continue
		}
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			continue
		}
		errs = append(errs, fmt.Errorf("checking volume automation %s destruction: %w", automationId, err))
	}
	return errors.Join(errs...)
}
