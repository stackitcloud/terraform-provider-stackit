package telemetrylink_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	telemetrylink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-min.tf
	resourceMin string

	//go:embed testdata/resource-max.tf
	resourceMax string
)

var testConfigVarsMin = config.Variables{
	"resource_type":       config.StringVariable("project"),
	"resource_id":         config.StringVariable(testutil.ProjectId),
	"region":              config.StringVariable(testutil.Region),
	"display_name":        config.StringVariable("tf-acc-test-link-min"),
	"access_token":        config.StringVariable("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.KMUFsIDTnFmyG3nMiGM6H9FNFUROf3wh7SmqJp-QV30"),
	"telemetry_router_id": config.StringVariable("97272f10-87ec-4715-b280-195a4ab1856c"),
}

func testConfigVarsMinUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(newVars, testConfigVarsMin)
	newVars["display_name"] = config.StringVariable("tf-acc-test-link-updated")
	return newVars
}

var testConfigVarsMax = config.Variables{
	"resource_type":       config.StringVariable("project"),
	"resource_id":         config.StringVariable(testutil.ProjectId),
	"region":              config.StringVariable(testutil.Region),
	"display_name":        config.StringVariable("tf-acc-test-link-max"),
	"description":         config.StringVariable("tf-acc-test-link-description"),
	"access_token":        config.StringVariable("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.KMUFsIDTnFmyG3nMiGM6H9FNFUROf3wh7SmqJp-QV30"),
	"telemetry_router_id": config.StringVariable("97272f10-87ec-4715-b280-195a4ab1856c"),
}

func testConfigVarsMaxUpdated() config.Variables {
	newVars := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(newVars, testConfigVarsMin)
	newVars["display_name"] = config.StringVariable("tf-acc-test-link-updated")
	newVars["description"] = config.StringVariable("Terraform Acceptance Test TelemetryLink Link Updated")
	return newVars
}

func TestAccTelemetryLinkLinkMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceMin,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "resource_type", testutil.ConvertConfigVariable(testConfigVarsMin["resource_type"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "resource_id", testutil.ConvertConfigVariable(testConfigVarsMin["resource_id"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "region", testutil.ConvertConfigVariable(testConfigVarsMin["region"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "display_name", testutil.ConvertConfigVariable(testConfigVarsMin["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "access_token", testutil.ConvertConfigVariable(testConfigVarsMin["access_token"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "telemetry_router_id", testutil.ConvertConfigVariable(testConfigVarsMin["telemetry_router_id"])),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "link_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "create_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "status"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigVarsMin,
				Config: testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceMin + `
			data "stackit_telemetrylink_link" "link" {
			 resource_type = stackit_telemetrylink_link.link.resource_type
			 resource_id   = stackit_telemetrylink_link.link.resource_id
			 region        = stackit_telemetrylink_link.link.region
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_telemetrylink_link.link", "resource_type", testutil.ConvertConfigVariable(testConfigVarsMin["resource_type"])),
					resource.TestCheckResourceAttr("data.stackit_telemetrylink_link.link", "resource_id", testutil.ConvertConfigVariable(testConfigVarsMin["resource_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "region",
						"data.stackit_telemetrylink_link.link", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "id",
						"data.stackit_telemetrylink_link.link", "id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "link_id",
						"data.stackit_telemetrylink_link.link", "link_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "display_name",
						"data.stackit_telemetrylink_link.link", "display_name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "create_time",
						"data.stackit_telemetrylink_link.link", "create_time",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "telemetry_router_id",
						"data.stackit_telemetrylink_link.link", "telemetry_router_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "status",
						"data.stackit_telemetrylink_link.link", "status",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_telemetrylink_link.link",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_telemetrylink_link.link"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_telemetrylink_link.link")
					}
					resourceType, ok := rs.Primary.Attributes["resource_type"]
					if !ok {
						return "", fmt.Errorf("resource_type not set")
					}
					return fmt.Sprintf("%s,%s,%s", resourceType, testutil.ProjectId, testutil.Region), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"access_token"},
			},
			// Update
			{
				ConfigVariables: testConfigVarsMinUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "resource_type", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["resource_type"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "resource_id", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["resource_id"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "region", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["region"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "display_name", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "access_token", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["access_token"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "telemetry_router_id", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["telemetry_router_id"])),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "link_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "create_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "status"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func TestAccTelemetryLinkLinkMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceMax,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "resource_type", testutil.ConvertConfigVariable(testConfigVarsMax["resource_type"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "resource_id", testutil.ConvertConfigVariable(testConfigVarsMax["resource_id"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "region", testutil.ConvertConfigVariable(testConfigVarsMax["region"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "display_name", testutil.ConvertConfigVariable(testConfigVarsMax["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "description", testutil.ConvertConfigVariable(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "access_token", testutil.ConvertConfigVariable(testConfigVarsMax["access_token"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "telemetry_router_id", testutil.ConvertConfigVariable(testConfigVarsMax["telemetry_router_id"])),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "link_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "create_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "status"),
				),
			},
			// Datasource
			{
				ConfigVariables: testConfigVarsMax,
				Config: testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceMax + `
			data "stackit_telemetrylink_link" "link" {
			 resource_type = stackit_telemetrylink_link.link.resource_type
			 resource_id   = stackit_telemetrylink_link.link.resource_id
			 region        = stackit_telemetrylink_link.link.region
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_telemetrylink_link.link", "resource_type", testutil.ConvertConfigVariable(testConfigVarsMax["resource_type"])),
					resource.TestCheckResourceAttr("data.stackit_telemetrylink_link.link", "resource_id", testutil.ConvertConfigVariable(testConfigVarsMax["resource_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "region",
						"data.stackit_telemetrylink_link.link", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "id",
						"data.stackit_telemetrylink_link.link", "id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "link_id",
						"data.stackit_telemetrylink_link.link", "link_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "display_name",
						"data.stackit_telemetrylink_link.link", "display_name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "description",
						"data.stackit_telemetrylink_link.link", "description",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "create_time",
						"data.stackit_telemetrylink_link.link", "create_time",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "telemetry_router_id",
						"data.stackit_telemetrylink_link.link", "telemetry_router_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_telemetrylink_link.link", "status",
						"data.stackit_telemetrylink_link.link", "status",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_telemetrylink_link.link",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_telemetrylink_link.link"]
					if !ok {
						return "", fmt.Errorf("not found: %s", "stackit_telemetrylink_link.link")
					}
					resourceType, ok := rs.Primary.Attributes["resource_type"]
					if !ok {
						return "", fmt.Errorf("resource_type not set")
					}
					return fmt.Sprintf("%s,%s,%s", resourceType, testutil.ProjectId, testutil.Region), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"access_token"},
			},
			// Update
			{
				ConfigVariables: testConfigVarsMaxUpdated(),
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + resourceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "resource_type", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["resource_type"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "resource_id", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["resource_id"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "region", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["region"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "display_name", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "description", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["description"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "access_token", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["access_token"])),
					resource.TestCheckResourceAttr("stackit_telemetrylink_link.link", "telemetry_router_id", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["telemetry_router_id"])),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "id"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "link_id"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "create_time"),
					resource.TestCheckResourceAttrSet("stackit_telemetrylink_link.link", "status"),
				),
			},
			// Deletion handled by framework
		},
	})
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAccCheckLogsInstanceDestroy,
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

func testAccCheckLogsInstanceDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := telemetrylink.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.LogsCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type link struct {
		resourceType string
		resourceId   string
		region       string
	}

	var linksToDestroy []link
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_telemetrylink_link" {
			continue
		}
		parts := strings.Split(rs.Primary.ID, core.Separator)
		linksToDestroy = append(linksToDestroy, link{
			resourceType: parts[0],
			resourceId:   parts[1],
			region:       parts[2],
		})
	}

	for _, l := range linksToDestroy {
		var err error
		switch l.resourceType {
		case "organization":
			err = client.DefaultAPI.DeleteOrganizationTelemetryLink(ctx, l.resourceId, l.region).Execute()
		case "folder":
			err = client.DefaultAPI.DeleteFolderTelemetryLink(ctx, l.resourceId, l.region).Execute()
		case "project":
			err = client.DefaultAPI.DeleteProjectTelemetryLink(ctx, l.resourceId, l.region).Execute()
		}
		if err != nil {
			return fmt.Errorf("deleting link %s %s: %w", l.resourceType, l.resourceId, err)
		}
	}
	return nil
}
