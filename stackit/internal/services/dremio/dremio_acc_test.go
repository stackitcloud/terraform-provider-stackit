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
	dremioWaiter "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceDremioInstanceMin string

//go:embed testdata/resource-max.tf
var resourceDremioInstanceMax string

const dremioInstanceResource = "stackit_dremio_instance.example"
const dremioInstanceDataResource = "data.stackit_dremio_instance.example"

const dremioUserResource = "stackit_dremio_user.example"
const dremioUserDataResource = "data.stackit_dremio_user.example"

var testDremioConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"region":     config.StringVariable(testutil.Region),
	//Instance
	"display_name":        config.StringVariable("dremioMinInstance"),
	"authentication_type": config.StringVariable(string(dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY)),
	//User
	"email":      config.StringVariable("minInstanceUser@example.com"),
	"first_name": config.StringVariable("Min"),
	"last_name":  config.StringVariable("InstanceUser"),
	"name":       config.StringVariable("minInstanceUser"),
	"password":   config.StringVariable("minInstanceUserPassword!23"),
}

var testDremioConfigVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"region":     config.StringVariable("eu01"),
	//Instance
	"display_name":                                     config.StringVariable("dremioMaxInstance"),
	"description":                                      config.StringVariable("description"),
	"authentication_type":                              config.StringVariable(string(dremioSdk.AUTHENTICATIONTYPE_OAUTH)),
	"authentication_oauth_authority_url":               config.StringVariable("oauth-authority-url"),
	"authentication_oauth_client_id":                   config.StringVariable("oauth-client-id"),
	"authentication_oauth_client_secret":               config.StringVariable("oauth-client-secret"),
	"authentication_oauth_client_jwt_claims_user_name": config.StringVariable("oauth-jwt-claim-user"),
	"authentication_oauth_scope":                       config.StringVariable("oauth-scope"),
	"authentication_oauth_parameter_name":              config.StringVariable("oauth-parameter-name"),
	"authentication_oauth_parameter_value":             config.StringVariable("oauth-parameter-value"),
	//User
	"email":            config.StringVariable("maxInstanceUser@example.com"),
	"user_description": config.StringVariable("Max Instance User Description"),
	"first_name":       config.StringVariable("Max"),
	"last_name":        config.StringVariable("InstanceUser"),
	"name":             config.StringVariable("maxInstanceUser"),
	"password":         config.StringVariable("maxInstanceUserPassword!23"),
}

func testDremioInstanceConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testDremioConfigVarsMin))
	maps.Copy(tempConfig, testDremioConfigVarsMin)
	tempConfig["display_name"] = config.StringVariable("dremioMinInstanceUpd")
	return tempConfig
}

func testDremioInstanceConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testDremioConfigVarsMax))
	maps.Copy(tempConfig, testDremioConfigVarsMax)
	tempConfig["display_name"] = config.StringVariable("dremioMaxInstanceUpd")
	tempConfig["description"] = config.StringVariable("description-upd")

	// switching idp to azuread
	tempConfig["authentication_type"] = config.StringVariable(string(dremioSdk.AUTHENTICATIONTYPE_AZUREAD))

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
				ConfigVariables: testDremioConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr(dremioInstanceResource, "project_id", testutil.ConvertConfigVariable(testDremioConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "id"),

					resource.TestCheckResourceAttr(dremioInstanceResource, "display_name", testutil.ConvertConfigVariable(testDremioConfigVarsMin["display_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.type", testutil.ConvertConfigVariable(testDremioConfigVarsMin["authentication_type"])),

					resource.TestCheckResourceAttrSet(dremioInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.ui"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.arrow_flight"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.catalog"),

					// User
					resource.TestCheckResourceAttr(dremioUserResource, "project_id", testutil.ConvertConfigVariable(testDremioConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(dremioUserResource, "region", testutil.Region),
					resource.TestCheckResourceAttrSet(dremioUserResource, "instance_id"),
					resource.TestCheckResourceAttrSet(dremioUserResource, "user_id"),
					resource.TestCheckResourceAttrSet(dremioUserResource, "id"),

					resource.TestCheckResourceAttr(dremioUserResource, "email", testutil.ConvertConfigVariable(testDremioConfigVarsMin["email"])),
					resource.TestCheckResourceAttr(dremioUserResource, "first_name", testutil.ConvertConfigVariable(testDremioConfigVarsMin["first_name"])),
					resource.TestCheckResourceAttr(dremioUserResource, "last_name", testutil.ConvertConfigVariable(testDremioConfigVarsMin["last_name"])),
					resource.TestCheckResourceAttr(dremioUserResource, "name", testutil.ConvertConfigVariable(testDremioConfigVarsMin["name"])),
					resource.TestCheckResourceAttr(dremioUserResource, "password", testutil.ConvertConfigVariable(testDremioConfigVarsMin["password"])),

					resource.TestCheckResourceAttrSet(dremioUserResource, "state"),
				),
			},
			// 2) Data Source
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceDremioInstanceMin,
				ConfigVariables: testDremioConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
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
						dremioInstanceResource, "endpoints.catalog",
						dremioInstanceDataResource, "endpoints.catalog",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "endpoints.ui",
						dremioInstanceDataResource, "endpoints.ui",
					),
					// User
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "project_id",
						dremioUserDataResource, "project_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "region",
						dremioUserDataResource, "region",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "instance_id",
						dremioUserDataResource, "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "user_id",
						dremioUserDataResource, "user_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "email",
						dremioUserDataResource, "email",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "first_name",
						dremioUserDataResource, "first_name",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "last_name",
						dremioUserDataResource, "last_name",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "name",
						dremioUserDataResource, "name",
					),
				),
			},
			// 3) Import
			{
				ConfigVariables:   testDremioConfigVarsMin,
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
			{
				ConfigVariables:   testDremioConfigVarsMin,
				ResourceName:      dremioUserResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[dremioUserResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", dremioUserResource)
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instanceId")
					}
					userId, ok := r.Primary.Attributes["user_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute userId")
					}

					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, userId), nil
				},
			},
			// 4) Update
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceDremioInstanceMin,
				ConfigVariables: testDremioInstanceConfigVarsMinUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dremioInstanceResource, "project_id", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(dremioInstanceResource, "display_name", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMinUpdated()["display_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.type", testutil.ConvertConfigVariable(testDremioInstanceConfigVarsMinUpdated()["authentication_type"])),

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
				ConfigVariables: testDremioConfigVarsMax,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceDremioInstanceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr(dremioInstanceResource, "project_id", testutil.ConvertConfigVariable(testDremioConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(dremioInstanceResource, "display_name", testutil.ConvertConfigVariable(testDremioConfigVarsMax["display_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "description", testutil.ConvertConfigVariable(testDremioConfigVarsMax["description"])),

					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.type", testutil.ConvertConfigVariable(testDremioConfigVarsMax["authentication_type"])),

					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.authority_url", testutil.ConvertConfigVariable(testDremioConfigVarsMax["authentication_oauth_authority_url"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.client_id", testutil.ConvertConfigVariable(testDremioConfigVarsMax["authentication_oauth_client_id"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.client_secret", testutil.ConvertConfigVariable(testDremioConfigVarsMax["authentication_oauth_client_secret"])),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "authentication.oauth.redirect_url"),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.jwt_claims.user_name", testutil.ConvertConfigVariable(testDremioConfigVarsMax["authentication_oauth_client_jwt_claims_user_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.scope", testutil.ConvertConfigVariable(testDremioConfigVarsMax["authentication_oauth_scope"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.parameters.0.name", testutil.ConvertConfigVariable(testDremioConfigVarsMax["authentication_oauth_parameter_name"])),
					resource.TestCheckResourceAttr(dremioInstanceResource, "authentication.oauth.parameters.0.value", testutil.ConvertConfigVariable(testDremioConfigVarsMax["authentication_oauth_parameter_value"])),

					resource.TestCheckResourceAttrSet(dremioInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.ui"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.arrow_flight"),
					resource.TestCheckResourceAttrSet(dremioInstanceResource, "endpoints.catalog"),

					// User
					resource.TestCheckResourceAttr(dremioUserResource, "project_id", testutil.ConvertConfigVariable(testDremioConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(dremioUserResource, "region", testutil.Region),
					resource.TestCheckResourceAttrSet(dremioUserResource, "instance_id"),
					resource.TestCheckResourceAttrSet(dremioUserResource, "user_id"),
					resource.TestCheckResourceAttrSet(dremioUserResource, "id"),

					resource.TestCheckResourceAttr(dremioUserResource, "email", testutil.ConvertConfigVariable(testDremioConfigVarsMax["email"])),
					resource.TestCheckResourceAttr(dremioUserResource, "user_description", testutil.ConvertConfigVariable(testDremioConfigVarsMax["user_description"])),
					resource.TestCheckResourceAttr(dremioUserResource, "first_name", testutil.ConvertConfigVariable(testDremioConfigVarsMax["first_name"])),
					resource.TestCheckResourceAttr(dremioUserResource, "last_name", testutil.ConvertConfigVariable(testDremioConfigVarsMax["last_name"])),
					resource.TestCheckResourceAttr(dremioUserResource, "name", testutil.ConvertConfigVariable(testDremioConfigVarsMax["name"])),
					resource.TestCheckResourceAttr(dremioUserResource, "password", testutil.ConvertConfigVariable(testDremioConfigVarsMax["password"])),

					resource.TestCheckResourceAttrSet(dremioUserResource, "state"),
				),
			},
			// 2) Data Source
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceDremioInstanceMax,
				ConfigVariables: testDremioConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
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
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.authority_url",
						dremioInstanceDataResource, "authentication.oauth.authority_url",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.client_id",
						dremioInstanceDataResource, "authentication.oauth.client_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.scope",
						dremioInstanceDataResource, "authentication.oauth.scope",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.parameters",
						dremioInstanceDataResource, "authentication.oauth.parameters",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "authentication.oauth.redirect_url",
						dremioInstanceDataResource, "authentication.oauth.redirect_url",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "endpoints.arrow_flight",
						dremioInstanceDataResource, "endpoints.arrow_flight",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "endpoints.catalog",
						dremioInstanceDataResource, "endpoints.catalog",
					),
					resource.TestCheckResourceAttrPair(
						dremioInstanceResource, "endpoints.ui",
						dremioInstanceDataResource, "endpoints.ui",
					),
					// User
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "project_id",
						dremioUserDataResource, "project_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "region",
						dremioUserDataResource, "region",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "instance_id",
						dremioUserDataResource, "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "user_id",
						dremioUserDataResource, "user_id",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "email",
						dremioUserDataResource, "email",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "user_description",
						dremioUserDataResource, "user_description",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "first_name",
						dremioUserDataResource, "first_name",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "last_name",
						dremioUserDataResource, "last_name",
					),
					resource.TestCheckResourceAttrPair(
						dremioUserResource, "name",
						dremioUserDataResource, "name",
					),
				),
			},
			// 3) Import
			{
				ConfigVariables:   testDremioConfigVarsMax,
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
