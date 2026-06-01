package dremio

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	dremioSdk "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi"
	dremioWaiter "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi/wait/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceDremioInstanceMin string

//go:embed testdata/resource-max.tf
var resourceDremioInstanceMax string

const dremioInstanceResource = "stackit_dremio_instance.example"
const dremioInstanceDataResource = "data.stackit_dremio_instance.example"

var testDremioInstanceConfigVarsMin = config.Variables{
	"project_id":          config.StringVariable(testutil.ProjectId),
	"region":              config.StringVariable(testutil.Region),
	"display_name":        config.StringVariable("dremioMinInstance"),
	"authentication_type": config.StringVariable("local-only"),
}

var testDremioInstanceConfigVarsMax = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"region":       config.StringVariable("eu01"),
	"display_name": config.StringVariable("dremioMaxInstance"),
	"description":  config.StringVariable("description"),

	"authentication_type": config.StringVariable("oauth"),

	"authentication_oauth_authority_url":               config.StringVariable("oauth-authority-url"),
	"authentication_oauth_client_id":                   config.StringVariable("oauth-client-id"),
	"authentication_oauth_client_secret":               config.StringVariable("oauth-client-secret"),
	"authentication_oauth_client_jwt_claims_user_name": config.StringVariable("oauth-jwt-claim-user"),
	"authentication_oauth_scope":                       config.StringVariable("oauth-scope"),
	"authentication_oauth_parameter_name":              config.StringVariable("oauth-parameter-name"),
	"authentication_oauth_parameter_value":             config.StringVariable("oauth-parameter-value"),
}

func testDremioInstanceConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testDremioInstanceConfigVarsMin))
	maps.Copy(tempConfig, testDremioInstanceConfigVarsMin)
	tempConfig["display_name"] = config.StringVariable("dremioMinInstanceUpd")
	return tempConfig
}

func testDremioInstanceConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testDremioInstanceConfigVarsMax))
	maps.Copy(tempConfig, testDremioInstanceConfigVarsMax)
	tempConfig["display_name"] = config.StringVariable("dremioMaxInstanceUpd")
	tempConfig["description"] = config.StringVariable("description-upd")

	// switching idp to azuread
	tempConfig["authentication_type"] = config.StringVariable("azuread")

	tempConfig["authentication_azuread_authority_url"] = config.StringVariable("azuread-authority-url-upd")
	tempConfig["authentication_azuread_client_id"] = config.StringVariable("azuread-client-id-upd")
	tempConfig["authentication_azuread_client_secret"] = config.StringVariable("azuread-client-secret-upd")

	return tempConfig
}

func TestDremioInstanceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccDremioInstanceDestroy,
		Steps: []resource.TestStep{
			// 1) Creation
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceDremioInstanceMin,
				ConfigVariables: testDremioInstanceConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dremioInstanceResource, "project_id", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(dremioInstanceResource, "display_name", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMin["display_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.type", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMin["authentication_type"])),

					resource.TestCheckResourceAttrSet(dremioInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.ui"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.arrow_flight"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.catalog"),
				),
			},
			// 2) Data Source
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceDremioInstanceMin,
				ConfigVariables: testDremioInstanceConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "project_id",
						dremioInstanceDataResource, "project_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "region",
						dremioInstanceDataResource, "region",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "instance_id",
						dremioInstanceDataResource, "instance_id",
					),

					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "display_name",
						dremioInstanceDataResource, "display_name",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.type",
						dremioInstanceDataResource, "authentication.type",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "endpoints.arrow_flight",
						dremioInstanceDataResource, "endpoints.arrow_flight",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "catalog",
						dremioInstanceDataResource, "catalog",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "endpoints.ui",
						dremioInstanceDataResource, "endpoints.ui",
					),
				),
			},
			// 3) Import
			{
				ConfigVariables:   testDremioInstanceConfigVarsMin,
				ResourceName:      dremioInstanceResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[dremioInstanceResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", dremioInstanceResource)
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instanceId")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
			},
			// 4) Update
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceDremioInstanceMin,
				ConfigVariables: testDremioInstanceConfigVarsMinUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dremioInstanceResource, "project_id", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(dremioInstanceResource, "display_name", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMin["display_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.type", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMin["authentication_type"])),

					resource.TestCheckResourceAttrSet(dremioInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.ui"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.arrow_flight"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.catalog"),
				),
			},
		},
	})
}

func TestDremioInstanceMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccDremioInstanceDestroy,
		Steps: []resource.TestStep{
			// 1) Creation
			{
				ConfigVariables: testDremioInstanceConfigVarsMax,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceDremioInstanceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dremioInstanceResource, "project_id", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(dremioInstanceResource, "display_name", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["display_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "description", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["description"])),

					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.type", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["authentication_type"])),

					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.authority_url", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["authentication_oauth_authority_url"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.client_id", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["authentication_oauth_client_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.client_secret", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["authentication_oauth_client_secret"])),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "authentication.oauth.redirect_url"),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.jwt_claims.user_name", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["authentication_oauth_client_jwt_claims_user_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.scope", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["authentication_oauth_scope"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.parameters.0.name", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["authentication_oauth_parameter_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.parameters.0.value", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMax["authentication_oauth_parameter_value"])),

					resource.TestCheckResourceAttrSet(dremioInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.ui"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.arrow_flight"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.catalog"),
				),
			},
			// 2) Data Source
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceDremioInstanceMax,
				ConfigVariables: testDremioInstanceConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "project_id",
						dremioInstanceDataResource, "project_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "region",
						dremioInstanceDataResource, "region",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "instance_id",
						dremioInstanceDataResource, "instance_id",
					),

					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "display_name",
						dremioInstanceDataResource, "display_name",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "description",
						dremioInstanceDataResource, "description",
					),

					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.type",
						dremioInstanceDataResource, "authentication.type",
					),
					// Authentication on the data source only shows the currently set IDP config,
					// which is oauth for the config here. Hence why we test for the oauth value here.
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.authority_url",
						dremioInstanceDataResource, "authentication.authority_url",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.client_id",
						dremioInstanceDataResource, "authentication.client_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.scope",
						dremioInstanceDataResource, "authentication.scope",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.parameters",
						dremioInstanceDataResource, "authentication.parameters",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.redirect_url",
						dremioInstanceDataResource, "authentication.redirect_url",
					),

					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "endpoints.arrow_flight",
						dremioInstanceDataResource, "endpoints.arrow_flight",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "catalog",
						dremioInstanceDataResource, "catalog",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "endpoints.ui",
						dremioInstanceDataResource, "endpoints.ui",
					),
				),
			},
			// 3) Import
			{
				ConfigVariables:   testDremioInstanceConfigVarsMax,
				ResourceName:      dremioInstanceResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[dremioInstanceResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", dremioInstanceResource)
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instanceId")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				}},
			// 4) Update
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceDremioInstanceMax,
				ConfigVariables: testDremioInstanceConfigVarsMaxUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dremioInstanceResource, "project_id", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(dremioInstanceResource, "display_name", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMaxUpdated()["display_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "description", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMaxUpdated()["description"])),

					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.type", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMaxUpdated()["authentication_type"])),

					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.azuread.authority_url", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMaxUpdated()["authentication_azuread_authority_url"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.azuread.client_id", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMaxUpdated()["authentication_azuread_client_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.azuread.client_secret", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMaxUpdated()["authentication_azuread_client_secret"])),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "authentication.azuread.redirect_url"),

					resource.TestCheckResourceAttrSet(dremioInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.ui"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.arrow_flight"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.catalog"),
				),
			},
		},
	})
}

func testAccDremioInstanceDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := dremioSdk.NewAPIClient(
		testutil.NewConfigBuilder().BuildClientOptions(testutil.DremioCustomEndpoint, true)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_dremio_instance" {
			continue
		}
		// Dremio internal ID: "[project_id],[region],[instance_id]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[2]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	// List all resources in the project/region to see what's left
	instancesResp, err := client.DefaultAPI.ListDremioInstances(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	// If the API returns a list of runners, check if our deleted ones are still there
	items := instancesResp.Dremios
	for i := range items {
		// If a runner we thought we deleted is found in the list
		if utils.Contains(instancesToDestroy, items[i].Id) {
			// Attempt a final delete and wait, just like Postgres
			err := client.DefaultAPI.DeleteDremioInstance(ctx, testutil.ProjectId, testutil.Region, items[i].Id).Execute()
			if err != nil {
				return fmt.Errorf("deleting Dremio instance %s during CheckDestroy: %w", items[i].Id, err)
			}

			// Using the wait handler for destruction verification
			_, err = dremioWaiter.DeleteDremioWaitHandler(ctx, client.DefaultAPI, testutil.ProjectId, testutil.Region, items[i].Id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("deleting Dremio instance %s during CheckDestroy: waiting for deletion %w", items[i].Id, err)
			}
		}
	}
	return nil
}
