package serviceenablement

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testdestroy"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/min.tf
	minConfig string
)

func TestAccServiceEnablementMin(t *testing.T) {
	configVars := config.Variables{
		"organization_id":       config.StringVariable(testutil.OrganizationId),
		"project_name":          config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"service_id":            config.StringVariable("cloud.stackit.ske"),
		"service_account_email": config.StringVariable(testutil.TestProjectServiceAccountEmail),
	}

	const (
		resourceIdentifier   = "stackit_service_enablement.service_enablement"
		datasourceIdentifier = "data.stackit_service_enablement.service_enablement"
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testdestroy.AccTestCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: configVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), minConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceIdentifier, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"stackit_resourcemanager_project.project", "project_id",
						resourceIdentifier, "project_id",
					),
					resource.TestCheckResourceAttr(resourceIdentifier, "region", testutil.Region),
					resource.TestCheckResourceAttr(resourceIdentifier, "service_id", testutil.ConvertConfigVariable(configVars["service_id"])),
				),
			},
			// Data source
			{
				ConfigVariables: configVars,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_service_enablement" "service_enablement" {
						project_id = stackit_resourcemanager_project.project.project_id
						service_id = "%s"
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), minConfig, testutil.ConvertConfigVariable(configVars["service_id"]),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceIdentifier, plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"stackit_resourcemanager_project.project", "project_id",
						datasourceIdentifier, "project_id",
					),
					resource.TestCheckResourceAttr(datasourceIdentifier, "region", testutil.Region),
					resource.TestCheckResourceAttr(datasourceIdentifier, "enabled", "true"),
					resource.TestCheckResourceAttr(datasourceIdentifier, "service_id", testutil.ConvertConfigVariable(configVars["service_id"])),
				),
			},
			// Import
			{
				ConfigVariables: configVars,
				ResourceName:    resourceIdentifier,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[resourceIdentifier]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", resourceIdentifier)
					}

					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}

					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}

					serviceId, ok := r.Primary.Attributes["service_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute service_id")
					}

					return fmt.Sprintf("%s,%s,%s", projectId, region, serviceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Deletion is done by the framework implicitly
		},
	})
}
