package sca_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	scaSdk "github.com/stackitcloud/stackit-sdk-go/services/sca/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceMinConfig string

var (
	appName = "tf-acc-app-" + acctest.RandStringFromCharSet(3, acctest.CharSetAlphaNum)
)

var testConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"region":     config.StringVariable(testutil.Region),
	"app_name":   config.StringVariable(appName),
	"stopped":    config.BoolVariable(false),
	"instances":  config.IntegerVariable(1),
}

func configVarsInvalid(vars config.Variables) config.Variables {
	tempConfig := maps.Clone(vars)
	delete(tempConfig, "app_name")
	return tempConfig
}

func configVarsApplicationUpdated() config.Variables {
	updatedConfig := maps.Clone(testConfigVarsMin)
	updatedConfig["instances"] = config.IntegerVariable(2)

	return updatedConfig
}

func TestAccSCAApplication(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSCAApplicationDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: configVarsInvalid(testConfigVarsMin),
				ExpectError:     regexp.MustCompile(`input variable "app_name" is not set`),
			},
			// Creation
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_sca_application.app", "application_id"),
					resource.TestCheckResourceAttrSet("stackit_sca_application.app", "environment_id"),
					resource.TestCheckResourceAttrSet("stackit_sca_application.app", "urls.#"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "stopped", "false"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "containers.#", "1"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "containers.0.name", "nginx"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "containers.0.image", "nginxinc/nginx-unprivileged:latest"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "containers.0.cpu_millis", "1000"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "containers.0.memory_mb", "512"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "containers.0.env.from_value.TEST_KEY", "test_value"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "network.public.enabled", "true"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "network.public.port", "8080"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "scaling.manual.instances", "1"),
				),
			},
			// Import
			{
				ResourceName:    "stackit_sca_application.app",
				ConfigVariables: testConfigVarsMin,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sca_application.app"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sca_application.app")
					}

					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					envId, ok := r.Primary.Attributes["environment_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute environment_id")
					}
					appId, ok := r.Primary.Attributes["application_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute application_id")
					}

					// internal ID structure: project_id,region,environment_id,application_id
					return fmt.Sprintf("%s,%s,%s,%s", projectId, region, envId, appId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
					"status",
				},
			},
			// Update
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceMinConfig,
				ConfigVariables: configVarsApplicationUpdated(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sca_application.app", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// resource.TestCheckResourceAttr("stackit_sca_application.app", "stopped", "true"),
					resource.TestCheckResourceAttr("stackit_sca_application.app", "scaling.manual.instances", "2"),
				),
			},
		},
	})
}

func testAccCheckSCAApplicationDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := scaSdk.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.ScaCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_sca_application" {
			continue
		}

		// internal ID structure: project_id,region,environment_id,application_id
		idParts := strings.Split(rs.Primary.ID, core.Separator)
		if len(idParts) != 4 {
			return fmt.Errorf("unexpected resource ID format: %s", rs.Primary.ID)
		}

		projectId := idParts[0]
		envId := idParts[2]
		appId := idParts[3]

		_, err := client.DefaultAPI.GetApplication(ctx, projectId, envId, appId).Execute()
		if err == nil {
			return fmt.Errorf("SCA application %s still exists after destroy", appId)
		}
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			continue
		}
		return fmt.Errorf("checking if SCA application %s is destroyed: %w", appId, err)
	}
	return nil
}
