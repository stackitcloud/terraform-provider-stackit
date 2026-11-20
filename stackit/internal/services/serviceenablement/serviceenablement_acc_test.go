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

	//go:embed testdata/max.tf
	maxConfig string
)

// buildImportStateIdFunc returns a ImportStateIdFunc which can be reused for all service
// enablement resource acceptance test implementations.
func buildImportStateIdFunc(t *testing.T, resourceId string) resource.ImportStateIdFunc {
	t.Helper()

	return func(s *terraform.State) (string, error) {
		r, ok := s.RootModule().Resources[resourceId]
		if !ok {
			return "", fmt.Errorf("couldn't find resource %s", resourceId)
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
	}
}

func TestAccServiceEnablementMin(t *testing.T) {
	// NOTE: This testcase is considered complete. It tests the service enablement resource
	// implementation for the ske service in a single region. When onboarding new services,
	// only add them to the "Max" test. There is no need to add them here.
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
				ConfigVariables:   configVars,
				ResourceName:      resourceIdentifier,
				ImportStateIdFunc: buildImportStateIdFunc(t, resourceIdentifier),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccServiceEnablementMax(t *testing.T) {
	// Tests the service enablement in all combinations of allowed service IDs and regions.
	configVars := config.Variables{
		"organization_id":       config.StringVariable(testutil.OrganizationId),
		"project_name":          config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"service_account_email": config.StringVariable(testutil.TestProjectServiceAccountEmail),
	}

	const (
		skeEu01ResourceIdentifier   = "stackit_service_enablement.ske_eu01"
		skeEu02ResourceIdentifier   = "stackit_service_enablement.ske_eu02"
		skeEu01DatasourceIdentifier = "data.stackit_service_enablement.ske_eu01"
		skeEu02DatasourceIdentifier = "data.stackit_service_enablement.ske_eu02"
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testdestroy.AccTestCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: configVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), maxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(skeEu01ResourceIdentifier, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(skeEu02ResourceIdentifier, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// SKE eu01
					resource.TestCheckResourceAttrPair(
						"stackit_resourcemanager_project.project", "project_id",
						skeEu01ResourceIdentifier, "project_id",
					),
					resource.TestCheckResourceAttr(skeEu01ResourceIdentifier, "region", "eu01"),
					resource.TestCheckResourceAttr(skeEu01ResourceIdentifier, "service_id", "cloud.stackit.ske"),

					// SKE eu02
					resource.TestCheckResourceAttrPair(
						"stackit_resourcemanager_project.project", "project_id",
						skeEu02ResourceIdentifier, "project_id",
					),
					resource.TestCheckResourceAttr(skeEu02ResourceIdentifier, "region", "eu02"),
					resource.TestCheckResourceAttr(skeEu02ResourceIdentifier, "service_id", "cloud.stackit.ske"),
				),
			},
			// Data source
			{
				ConfigVariables: configVars,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_service_enablement" "ske_eu01" {
						project_id = stackit_resourcemanager_project.project.project_id
						service_id = "cloud.stackit.ske"
						region     = "eu01"
					}
					
					data "stackit_service_enablement" "ske_eu02" {
						project_id = stackit_resourcemanager_project.project.project_id
						service_id = "cloud.stackit.ske"
						region     = "eu02"
					}
					`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), maxConfig,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(skeEu01ResourceIdentifier, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(skeEu02ResourceIdentifier, plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// SKE eu01
					resource.TestCheckResourceAttrPair(
						"stackit_resourcemanager_project.project", "project_id",
						skeEu01DatasourceIdentifier, "project_id",
					),
					resource.TestCheckResourceAttr(skeEu01DatasourceIdentifier, "region", "eu01"),
					resource.TestCheckResourceAttr(skeEu01DatasourceIdentifier, "enabled", "true"),
					resource.TestCheckResourceAttr(skeEu01ResourceIdentifier, "service_id", "cloud.stackit.ske"),

					// SKE eu02
					resource.TestCheckResourceAttrPair(
						"stackit_resourcemanager_project.project", "project_id",
						skeEu02DatasourceIdentifier, "project_id",
					),
					resource.TestCheckResourceAttr(skeEu02DatasourceIdentifier, "region", "eu02"),
					resource.TestCheckResourceAttr(skeEu02DatasourceIdentifier, "enabled", "true"),
					resource.TestCheckResourceAttr(skeEu02ResourceIdentifier, "service_id", "cloud.stackit.ske"),
				),
			},
			// Import - SKE eu01
			{
				ConfigVariables:   configVars,
				ResourceName:      skeEu01ResourceIdentifier,
				ImportStateIdFunc: buildImportStateIdFunc(t, skeEu01ResourceIdentifier),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Import - SKE eu02
			{
				ConfigVariables:   configVars,
				ResourceName:      skeEu02ResourceIdentifier,
				ImportStateIdFunc: buildImportStateIdFunc(t, skeEu02ResourceIdentifier),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Deletion is done by the framework implicitly
		},
	})
}
