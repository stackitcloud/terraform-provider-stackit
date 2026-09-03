package automation_test

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/datasource-templates.tf
var templatesDataSourceConfig string

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
