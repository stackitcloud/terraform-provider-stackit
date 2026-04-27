package telemetryrouter_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/instance-min.tf
	instanceMin string

	//go:embed testdata/instance-max.tf
	instanceMax string

	//go:embed testdata/access-token-min.tf
	accessTokenMinConfig string

	//go:embed testdata/access-token-max.tf
	accessTokenMaxConfig string

	//go:embed testdata/destination-otlp-basic-auth.tf
	destinationOTLPBasicAuthConfig string

	//go:embed testdata/destination-otlp-bearer-token.tf
	destinationOTLPBearerTokenConfig string

	//go:embed testdata/destination-s3.tf
	destinationS3Config string
)

var testConfigVarsMin = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"region":       config.StringVariable(testutil.Region),
	"display_name": config.StringVariable("tf-acc-test-telemetryrouter-min"),
}

func testConfigVarsMinUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(newVars, testConfigVarsMin)
	newVars["display_name"] = config.StringVariable("tf-acc-test-telemetryrouter-upd")
	return newVars
}

var testConfigVarsMax = config.Variables{
	"project_id":     config.StringVariable(testutil.ProjectId),
	"region":         config.StringVariable(testutil.Region),
	"display_name":   config.StringVariable("tf-acc-test-telemetryrouter-max"),
	"description":    config.StringVariable("Terraform Acceptance Test TelemetryRouter Instance"),
	"filter_key":     config.StringVariable("key"),
	"filter_level":   config.StringVariable("logRecord"),
	"filter_matcher": config.StringVariable("="),
	"filter_value0":  config.StringVariable("value1"),
	"filter_value1":  config.StringVariable("value2"),
}

func testConfigVarsMaxUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(newVars, testConfigVarsMin)
	newVars["display_name"] = config.StringVariable("tf-acc-test-telemetryrouter-upd")
	newVars["description"] = config.StringVariable("Terraform Acceptance Test TelemetryRouter Instance Updated")
	newVars["filter_key"] = config.StringVariable("other")
	newVars["filter_level"] = config.StringVariable("resource")
	newVars["filter_matcher"] = config.StringVariable("!=")
	newVars["filter_value0"] = config.StringVariable("value3")
	newVars["filter_value1"] = config.StringVariable("value4")

	return newVars
}

var testConfigAccessTokenVarsMin = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"region":       config.StringVariable(testutil.Region),
	"display_name": config.StringVariable("tf-acc-test-acc-token-min"),
	"status":       config.StringVariable("active"),
}

func testConfigAccessTokenVarsMinUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigAccessTokenVarsMin))
	maps.Copy(newVars, testConfigAccessTokenVarsMin)
	newVars["display_name"] = config.StringVariable("tf-acc-test-token-updated")
	return newVars
}

var testConfigAccessTokenVarsMax = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"region":       config.StringVariable(testutil.Region),
	"display_name": config.StringVariable("tf-acc-test-acc-token-max"),
	"description":  config.StringVariable("Terraform Acceptance Test TelemetryRouter Access Token"),
	"ttl":          config.IntegerVariable(7),
	"status":       config.StringVariable("active"),
}

func testConfigAccessTokenVarsMaxUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigAccessTokenVarsMax))
	maps.Copy(newVars, testConfigAccessTokenVarsMax)
	newVars["display_name"] = config.StringVariable("tf-acc-test-token-updated")
	newVars["description"] = config.StringVariable("tf-acc-test-token-decription-updated")
	return newVars
}

var testConfigDestinationVarsOTLPBasicAuth = config.Variables{
	"project_id":                    config.StringVariable(testutil.ProjectId),
	"region":                        config.StringVariable(testutil.Region),
	"display_name":                  config.StringVariable("tf-acc-test-tlmr-dest"),
	"description":                   config.StringVariable("Terraform Acceptance Test TelemetryRouter OTLP Destination"),
	"config_filter_key":             config.StringVariable("key"),
	"config_filter_level":           config.StringVariable("logRecord"),
	"config_filter_matcher":         config.StringVariable("="),
	"config_filter_value0":          config.StringVariable("value1"),
	"config_filter_value1":          config.StringVariable("value2"),
	"config_opentelemetry_username": config.StringVariable("user"),
	"config_opentelemetry_password": config.StringVariable("password"),
	"config_opentelemetry_uri":      config.StringVariable("https://localhost:8116"),
}

func testConfigDestinationVarsOTLPBasicAuthUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigDestinationVarsOTLPBasicAuth))
	maps.Copy(newVars, testConfigDestinationVarsOTLPBasicAuth)
	newVars["display_name"] = config.StringVariable("tf-acc-test-tlmr-dest-upd")
	newVars["description"] = config.StringVariable("Terraform Acceptance Test TelemetryRouter Destination Updated")
	newVars["config_filter_key"] = config.StringVariable("other")
	newVars["config_filter_level"] = config.StringVariable("resource")
	newVars["config_filter_matcher"] = config.StringVariable("!=")
	newVars["config_filter_value0"] = config.StringVariable("value3")
	newVars["config_filter_value1"] = config.StringVariable("value4")
	newVars["config_opentelemetry_username"] = config.StringVariable("user1")
	newVars["config_opentelemetry_password"] = config.StringVariable("pass1")
	newVars["config_opentelemetry_uri"] = config.StringVariable("https://localhost:8117")

	return newVars
}

var testConfigDestinationVarsOTLPBearerToken = config.Variables{
	"project_id":                        config.StringVariable(testutil.ProjectId),
	"region":                            config.StringVariable(testutil.Region),
	"display_name":                      config.StringVariable("tf-acc-test-tlmr-dest"),
	"description":                       config.StringVariable("Terraform Acceptance Test TelemetryRouter OTLP Destination"),
	"config_filter_key":                 config.StringVariable("key"),
	"config_filter_level":               config.StringVariable("logRecord"),
	"config_filter_matcher":             config.StringVariable("="),
	"config_filter_value0":              config.StringVariable("value1"),
	"config_filter_value1":              config.StringVariable("value2"),
	"config_opentelemetry_bearer_token": config.StringVariable("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.KMUFsIDTnFmyG3nMiGM6H9FNFUROf3wh7SmqJp-QV30"),
	"config_opentelemetry_uri":          config.StringVariable("https://localhost:8116"),
}

func testConfigDestinationVarsOTLPBearerTokenUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigDestinationVarsOTLPBearerToken))
	maps.Copy(newVars, testConfigDestinationVarsOTLPBearerToken)
	newVars["display_name"] = config.StringVariable("tf-acc-test-tlmr-dest-upd")
	newVars["description"] = config.StringVariable("Terraform Acceptance Test TelemetryRouter Destination Updated")
	newVars["config_filter_key"] = config.StringVariable("other")
	newVars["config_filter_level"] = config.StringVariable("resource")
	newVars["config_filter_matcher"] = config.StringVariable("!=")
	newVars["config_filter_value0"] = config.StringVariable("value3")
	newVars["config_filter_value1"] = config.StringVariable("value4")
	newVars["config_opentelemetry_bearer_token"] = config.StringVariable("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.KMUFsIDTnFmyG3nMiGM6H9FNFUROf3wh7SmqJp-QV31")
	newVars["config_opentelemetry_uri"] = config.StringVariable("https://localhost:8117")

	return newVars
}

var testConfigDestinationVarsS3 = config.Variables{
	"project_id":            config.StringVariable(testutil.ProjectId),
	"region":                config.StringVariable(testutil.Region),
	"display_name":          config.StringVariable("tf-acc-test-tlmr-dest"),
	"description":           config.StringVariable("Terraform Acceptance Test TelemetryRouter OTLP Destination"),
	"config_filter_key":     config.StringVariable("key"),
	"config_filter_level":   config.StringVariable("logRecord"),
	"config_filter_matcher": config.StringVariable("="),
	"config_filter_value0":  config.StringVariable("value1"),
	"config_filter_value1":  config.StringVariable("value2"),
	"config_s3_id":          config.StringVariable("id"),
	"config_s3_secret":      config.StringVariable("secret"),
	"config_s3_bucket":      config.StringVariable("bucket"),
	"config_s3_endpoint":    config.StringVariable("https://localhost:8116"),
}

func testConfigDestinationVarsS3Updated() config.Variables {
	newVars := make(config.Variables, len(testConfigDestinationVarsS3))
	maps.Copy(newVars, testConfigDestinationVarsS3)
	newVars["display_name"] = config.StringVariable("tf-acc-test-tlmr-dest-upd")
	newVars["description"] = config.StringVariable("Terraform Acceptance Test TelemetryRouter Destination Updated")
	newVars["config_filter_key"] = config.StringVariable("other")
	newVars["config_filter_level"] = config.StringVariable("resource")
	newVars["config_filter_matcher"] = config.StringVariable("!=")
	newVars["config_filter_value0"] = config.StringVariable("value3")
	newVars["config_filter_value1"] = config.StringVariable("value4")
	newVars["config_s3_id"] = config.StringVariable("id1")
	newVars["config_s3_secret"] = config.StringVariable("secret1")
	newVars["config_s3_bucket"] = config.StringVariable("bucket1")
	newVars["config_s3_endpoint"] = config.StringVariable("https://localhost:8117")

	return newVars
}

func TestTelemetryRouterInstanceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + instanceMin,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "region", testutil.ConvertConfigVariable(testConfigVarsMin["region"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "display_name", testutil.ConvertConfigVariable(testConfigVarsMin["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "uri"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "status"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigVarsMin,
				Config: testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + instanceMin + `
			data "stackit_telemetryrouter_instance" "router" {
			 project_id   = stackit_telemetryrouter_instance.router.project_id
			 region       = stackit_telemetryrouter_instance.router.region
			 instance_id  = stackit_telemetryrouter_instance.router.instance_id
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "region",
						"data.stackit_telemetryrouter_instance.router", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "display_name",
						"data.stackit_telemetryrouter_instance.router", "display_name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "id",
						"data.stackit_telemetryrouter_instance.router", "id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "instance_id",
						"data.stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "creation_time",
						"data.stackit_telemetryrouter_instance.router", "creation_time",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "uri",
						"data.stackit_telemetryrouter_instance.router", "uri",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "status",
						"data.stackit_telemetryrouter_instance.router", "status",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_telemetryrouter_instance.router",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_telemetryrouter_instance.router"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_telemetryrouter_instance.router")
					}
					instanceId, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsMinUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + instanceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "region", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["region"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "display_name", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "uri"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "status"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func TestTelemetryRouterInstanceMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + instanceMax,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "region", testutil.ConvertConfigVariable(testConfigVarsMax["region"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "display_name", testutil.ConvertConfigVariable(testConfigVarsMax["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "description", testutil.ConvertConfigVariable(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigVarsMax["filter_key"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigVarsMax["filter_level"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigVarsMax["filter_matcher"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigVarsMax["filter_value0"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigVarsMax["filter_value1"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "uri"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "status"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigVarsMax,
				Config: testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + instanceMax + `
			data "stackit_telemetryrouter_instance" "router" {
			project_id   = stackit_telemetryrouter_instance.router.project_id
			region       = stackit_telemetryrouter_instance.router.region
			instance_id  = stackit_telemetryrouter_instance.router.instance_id
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "region",
						"data.stackit_telemetryrouter_instance.router", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "display_name",
						"data.stackit_telemetryrouter_instance.router", "display_name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "id",
						"data.stackit_telemetryrouter_instance.router", "id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "instance_id",
						"data.stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "creation_time",
						"data.stackit_telemetryrouter_instance.router", "creation_time",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "uri",
						"data.stackit_telemetryrouter_instance.router", "uri",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "status",
						"data.stackit_telemetryrouter_instance.router", "status",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "description",
						"data.stackit_telemetryrouter_instance.router", "description",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "filter.attributes.0.key",
						"data.stackit_telemetryrouter_instance.router", "filter.attributes.0.key",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "filter.attributes.0.level",
						"data.stackit_telemetryrouter_instance.router", "filter.attributes.0.level",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "filter.attributes.0.matcher",
						"data.stackit_telemetryrouter_instance.router", "filter.attributes.0.matcher",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "filter.attributes.0.value.0",
						"data.stackit_telemetryrouter_instance.router", "filter.attributes.0.value.0",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_instance.router", "filter.attributes.0.value.1",
						"data.stackit_telemetryrouter_instance.router", "filter.attributes.0.value.1",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_telemetryrouter_instance.router",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_telemetryrouter_instance.router"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_telemetryrouter_instance.router")
					}
					instanceId, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVarsMaxUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + instanceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "region", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["region"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "display_name", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "description", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["description"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["filter_key"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["filter_level"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["filter_matcher"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["filter_value0"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["filter_value1"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "uri"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "status"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func TestAccTelemetryRouterAccessTokenMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigAccessTokenVarsMin,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + accessTokenMinConfig,
				Check: resource.ComposeTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "region", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["region"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "display_name", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "instance_id"),

					// Access token data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "project_id", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "region", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_access_token.accessToken", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "display_name", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "access_token_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "creator_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "access_token"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "status", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["status"])),
					resource.TestCheckNoResourceAttr("stackit_telemetryrouter_access_token.accessToken", "expiration_time"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigAccessTokenVarsMin,
				Config: testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + accessTokenMinConfig + `
					data "stackit_telemetryrouter_access_token" "accessToken" {
					  project_id   = stackit_telemetryrouter_access_token.accessToken.project_id
					  region       = stackit_telemetryrouter_access_token.accessToken.region
					  instance_id  = stackit_telemetryrouter_access_token.accessToken.instance_id
					  access_token_id  = stackit_telemetryrouter_access_token.accessToken.access_token_id
					}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_access_token.accessToken", "project_id", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_access_token.accessToken", "region", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_access_token.accessToken", "instance_id",
						"data.stackit_telemetryrouter_access_token.accessToken", "instance_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_access_token.accessToken", "access_token_id"),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_access_token.accessToken", "display_name", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["display_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_access_token.accessToken", "creator_id"),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_access_token.accessToken", "status", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMin["status"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigAccessTokenVarsMin,
				ResourceName:    "stackit_telemetryrouter_access_token.accessToken",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_telemetryrouter_access_token.accessToken"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_telemetryrouter_access_token.accessToken")
					}
					instanceId, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					tokenId, ok := rs.Primary.Attributes["access_token_id"]
					if !ok {
						return "", fmt.Errorf("access_token_id not set")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, tokenId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ttl", "access_token"},
			},
			// Update
			{
				ConfigVariables: testConfigAccessTokenVarsMinUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + accessTokenMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "project_id", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "region", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMinUpdated()["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_access_token.accessToken", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "display_name", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMinUpdated()["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "access_token_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "creator_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "status"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "access_token"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func TestAccTelemetryRouterAccessTokenMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigAccessTokenVarsMax,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + accessTokenMaxConfig,
				Check: resource.ComposeTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "region", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["region"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "display_name", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "instance_id"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "description", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["description"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "uri"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "status"),

					// Access token data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "project_id", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "region", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_access_token.accessToken", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "access_token_id"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "display_name", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "description", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "ttl", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["ttl"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "access_token_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "creator_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "access_token"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "status", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["status"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "expiration_time"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigAccessTokenVarsMax,
				Config: testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + accessTokenMaxConfig + `
					data "stackit_telemetryrouter_access_token" "accessToken" {
					  project_id   = stackit_telemetryrouter_access_token.accessToken.project_id
					  region       = stackit_telemetryrouter_access_token.accessToken.region
					  instance_id  = stackit_telemetryrouter_access_token.accessToken.instance_id
					  access_token_id  = stackit_telemetryrouter_access_token.accessToken.access_token_id
					}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_access_token.accessToken", "project_id", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_access_token.accessToken", "region", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_access_token.accessToken", "instance_id",
						"data.stackit_telemetryrouter_access_token.accessToken", "instance_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_access_token.accessToken", "access_token_id"),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_access_token.accessToken", "display_name", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["display_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_access_token.accessToken", "creator_id"),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_access_token.accessToken", "status", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMax["status"])),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_access_token.accessToken", "expiration_time"),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_access_token.accessToken", "description"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigAccessTokenVarsMax,
				ResourceName:    "stackit_telemetryrouter_access_token.accessToken",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_telemetryrouter_access_token.accessToken"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_telemetryrouter_access_token.accessToken")
					}
					instanceId, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					tokenId, ok := rs.Primary.Attributes["access_token_id"]
					if !ok {
						return "", fmt.Errorf("access_token_id not set")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, tokenId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ttl", "access_token"},
			},
			// Update
			{
				ConfigVariables: testConfigAccessTokenVarsMaxUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + accessTokenMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "project_id", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "region", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMaxUpdated()["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_access_token.accessToken", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "display_name", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMaxUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_access_token.accessToken", "description", testutil.ConvertConfigVariable(testConfigAccessTokenVarsMaxUpdated()["description"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "access_token_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "creator_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "status"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_access_token.accessToken", "access_token"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func TestAccTelemetryRouterDestinationOTLPBasicAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigDestinationVarsOTLPBasicAuth,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + destinationOTLPBasicAuthConfig,
				Check: resource.ComposeTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["region"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "uri"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "status"),

					// Destination data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_destination.destination", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "description", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["description"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_key"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_level"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_matcher"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_value0"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_value1"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.basic_auth.username", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_opentelemetry_username"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.basic_auth.password", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_opentelemetry_password"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.uri", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_opentelemetry_uri"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "destination_id"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.config_type", "OpenTelemetry"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "status"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigDestinationVarsOTLPBasicAuth,
				Config: testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + destinationOTLPBasicAuthConfig + `
					data "stackit_telemetryrouter_destination" "destination" {
					  project_id   = stackit_telemetryrouter_destination.destination.project_id
					  region       = stackit_telemetryrouter_destination.destination.region
					  instance_id  = stackit_telemetryrouter_destination.destination.instance_id
					  destination_id  = stackit_telemetryrouter_destination.destination.destination_id
					}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_destination.destination", "instance_id",
						"data.stackit_telemetryrouter_destination.destination", "instance_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_destination.destination", "destination_id"),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["display_name"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "description", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["description"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_key"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_level"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_matcher"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_value0"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_filter_value1"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.opentelemetry.uri", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuth["config_opentelemetry_uri"])),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_destination.destination", "status"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigDestinationVarsOTLPBasicAuth,
				ResourceName:    "stackit_telemetryrouter_destination.destination",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_telemetryrouter_destination.destination"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_telemetryrouter_destination.destination")
					}
					instanceId, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					destinationId, ok := rs.Primary.Attributes["destination_id"]
					if !ok {
						return "", fmt.Errorf("destination_id not set")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, destinationId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.opentelemetry.basic_auth.password"},
			},
			// Update
			{
				ConfigVariables: testConfigDestinationVarsOTLPBasicAuthUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + destinationOTLPBasicAuthConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_destination.destination", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "description", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["description"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["config_filter_key"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["config_filter_level"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["config_filter_matcher"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["config_filter_value0"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["config_filter_value1"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.basic_auth.username", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["config_opentelemetry_username"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.basic_auth.password", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["config_opentelemetry_password"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.uri", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBasicAuthUpdated()["config_opentelemetry_uri"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "destination_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "status"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.config_type", "OpenTelemetry"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func TestAccTelemetryRouterDestinationOTLPBearerToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigDestinationVarsOTLPBearerToken,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + destinationOTLPBearerTokenConfig,
				Check: resource.ComposeTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["region"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "uri"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "status"),

					// Destination data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_destination.destination", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "description", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["description"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_key"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_level"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_matcher"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_value0"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_value1"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.bearer_token", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_opentelemetry_bearer_token"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.uri", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_opentelemetry_uri"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "destination_id"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.config_type", "OpenTelemetry"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "status"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigDestinationVarsOTLPBearerToken,
				Config: testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + destinationOTLPBearerTokenConfig + `
					data "stackit_telemetryrouter_destination" "destination" {
					  project_id   = stackit_telemetryrouter_destination.destination.project_id
					  region       = stackit_telemetryrouter_destination.destination.region
					  instance_id  = stackit_telemetryrouter_destination.destination.instance_id
					  destination_id  = stackit_telemetryrouter_destination.destination.destination_id
					}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_destination.destination", "instance_id",
						"data.stackit_telemetryrouter_destination.destination", "instance_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_destination.destination", "destination_id"),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["display_name"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "description", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["description"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_key"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_level"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_matcher"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_value0"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_filter_value1"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.opentelemetry.uri", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerToken["config_opentelemetry_uri"])),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_destination.destination", "status"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigDestinationVarsOTLPBearerToken,
				ResourceName:    "stackit_telemetryrouter_destination.destination",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_telemetryrouter_destination.destination"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_telemetryrouter_destination.destination")
					}
					instanceId, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					destinationId, ok := rs.Primary.Attributes["destination_id"]
					if !ok {
						return "", fmt.Errorf("destination_id not set")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, destinationId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.opentelemetry.bearer_token"},
			},
			// Update
			{
				ConfigVariables: testConfigDestinationVarsOTLPBearerTokenUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + destinationOTLPBearerTokenConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_destination.destination", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "description", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["description"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["config_filter_key"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["config_filter_level"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["config_filter_matcher"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["config_filter_value0"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["config_filter_value1"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.bearer_token", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["config_opentelemetry_bearer_token"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.opentelemetry.uri", testutil.ConvertConfigVariable(testConfigDestinationVarsOTLPBearerTokenUpdated()["config_opentelemetry_uri"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "destination_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "status"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.config_type", "OpenTelemetry"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func TestAccTelemetryRouterDestinationS3(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigDestinationVarsS3,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + destinationS3Config,
				Check: resource.ComposeTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["region"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_instance.router", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "creation_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "uri"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_instance.router", "status"),

					// Destination data
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_destination.destination", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "description", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["description"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_key"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_level"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_matcher"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_value0"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_value1"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.s3.access_key.id", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_s3_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.s3.access_key.secret", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_s3_secret"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.s3.bucket", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_s3_bucket"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.s3.endpoint", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_s3_endpoint"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "destination_id"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.config_type", "S3"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "status"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigDestinationVarsS3,
				Config: testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + destinationS3Config + `
					data "stackit_telemetryrouter_destination" "destination" {
					  project_id   = stackit_telemetryrouter_destination.destination.project_id
					  region       = stackit_telemetryrouter_destination.destination.region
					  instance_id  = stackit_telemetryrouter_destination.destination.instance_id
					  destination_id  = stackit_telemetryrouter_destination.destination.destination_id
					}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_destination.destination", "instance_id",
						"data.stackit_telemetryrouter_destination.destination", "instance_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_destination.destination", "destination_id"),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["display_name"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "description", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["description"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_key"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_level"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_matcher"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_value0"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_filter_value1"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.s3.bucket", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_s3_bucket"])),
					resource.TestCheckResourceAttr("data.stackit_telemetryrouter_destination.destination", "config.s3.endpoint", testutil.ConvertConfigVariable(testConfigDestinationVarsS3["config_s3_endpoint"])),
					resource.TestCheckResourceAttrSet("data.stackit_telemetryrouter_destination.destination", "status"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigDestinationVarsS3,
				ResourceName:    "stackit_telemetryrouter_destination.destination",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_telemetryrouter_destination.destination"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_telemetryrouter_destination.destination")
					}
					instanceId, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					destinationId, ok := rs.Primary.Attributes["destination_id"]
					if !ok {
						return "", fmt.Errorf("destination_id not set")
					}
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, destinationId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.s3.access_key"},
			},
			// Update
			{
				ConfigVariables: testConfigDestinationVarsS3Updated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + destinationS3Config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "project_id", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "region", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["region"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetryrouter_destination.destination", "instance_id",
						"stackit_telemetryrouter_instance.router", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "display_name", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "description", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["description"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.key", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["config_filter_key"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.level", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["config_filter_level"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.matcher", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["config_filter_matcher"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.0", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["config_filter_value0"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.filter.attributes.0.values.1", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["config_filter_value1"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.s3.access_key.id", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["config_s3_id"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.s3.access_key.secret", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["config_s3_secret"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.s3.bucket", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["config_s3_bucket"])),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.s3.endpoint", testutil.ConvertConfigVariable(testConfigDestinationVarsS3Updated()["config_s3_endpoint"])),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "destination_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetryrouter_destination.destination", "status"),
					resource.TestCheckResourceAttr("stackit_telemetryrouter_destination.destination", "config.config_type", "S3"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAccCheckTelemetryRouterInstanceDestroy,
		testAccCheckTelemetryRouterAccessTokenDestroy,
		testAccCheckTelemetryRouterDestinationDestroy,
	}

	var errs []error

	wg := sync.WaitGroup{}
	wg.Add(len(checkFunctions))

	for _, f := range checkFunctions {
		go func() {
			err := f(s)
			if err != nil {
				errs = append(errs, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

func testAccCheckTelemetryRouterInstanceDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := telemetryrouter.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.TelemetryRouterCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var instancesToDestroy []string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_telemetryrouter_instance" {
			continue
		}
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[2]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	response, err := client.DefaultAPI.ListTelemetryRouters(ctx, testutil.ProjectId, "eu01").Execute()
	if err != nil {
		return fmt.Errorf("getting instances: %w", err)
	}
	for i := range response.TelemetryRouters {
		if !slices.Contains(instancesToDestroy, response.TelemetryRouters[i].Id) {
			continue
		}

		err := client.DefaultAPI.DeleteTelemetryRouter(ctx, testutil.ProjectId, "eu01", response.TelemetryRouters[i].Id).Execute()
		if err != nil {
			return fmt.Errorf("deleting instance %s: %w", response.TelemetryRouters[i].Id, err)
		}
	}
	return nil
}

func testAccCheckTelemetryRouterAccessTokenDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := telemetryrouter.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.TelemetryRouterCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error
	// access tokens
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_telemetryrouter_access_token" {
			continue
		}
		accessTokenId := strings.Split(rs.Primary.ID, core.Separator)[3]
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[2]
		region := strings.Split(rs.Primary.ID, core.Separator)[1]

		err := client.DefaultAPI.DeleteAccessToken(ctx, testutil.ProjectId, region, instanceId, accessTokenId).Execute()
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger access token deletion %q: %w", accessTokenId, err))
		}
	}

	return errors.Join(errs...)
}

func testAccCheckTelemetryRouterDestinationDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := telemetryrouter.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.TelemetryRouterCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error
	// access tokens
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_telemetryrouter_destination" {
			continue
		}
		destinationId := strings.Split(rs.Primary.ID, core.Separator)[3]
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[2]
		region := strings.Split(rs.Primary.ID, core.Separator)[1]

		err := client.DefaultAPI.DeleteDestination(ctx, testutil.ProjectId, region, instanceId, destinationId).Execute()
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger destination deletion %q: %w", destinationId, err))
		}
	}

	return errors.Join(errs...)
}
