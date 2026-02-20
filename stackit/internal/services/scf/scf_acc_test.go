package scf

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stackitcloud/stackit-sdk-go/services/scf"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceMin string

//go:embed testdata/resource-max.tf
var resourceMax string

var randName = acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
var nameMin = fmt.Sprintf("scf-min-%s-org", randName)
var nameMinUpdated = fmt.Sprintf("scf-min-%s-upd-org", randName)
var nameMax = fmt.Sprintf("scf-max-%s-org", randName)
var nameMaxUpdated = fmt.Sprintf("scf-max-%s-upd-org", randName)

const (
	platformName       = "Shared Cloud Foundry (public)"
	platformSystemId   = "01.cf.eu01"
	platformIdMax      = "0a3d1188-353a-4004-832c-53039c0e3868"
	platformApiUrl     = "https://api.system.01.cf.eu01.stackit.cloud"
	platformConsoleUrl = "https://console.apps.01.cf.eu01.stackit.cloud"
	quotaIdMax         = "e22cfe1a-0318-473f-88db-61d62dc629c0" // small
	quotaIdMaxUpdated  = "5ea6b9ab-4048-4bd9-8a8a-5dd7fc40745d" // medium
	suspendedMax       = true
	region             = "eu01"
)

var testConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(nameMin),
}

var testConfigVarsMax = config.Variables{
	"project_id":  config.StringVariable(testutil.ProjectId),
	"name":        config.StringVariable(nameMax),
	"platform_id": config.StringVariable(platformIdMax),
	"quota_id":    config.StringVariable(quotaIdMax),
	"suspended":   config.BoolVariable(suspendedMax),
	"region":      config.StringVariable(region),
}

func testScfOrgConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(tempConfig, testConfigVarsMin)
	// update scf organization to a new name
	tempConfig["name"] = config.StringVariable(nameMinUpdated)
	return tempConfig
}

func testScfOrgConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMax))
	maps.Copy(tempConfig, testConfigVarsMax)
	// update scf organization to a new name, unsuspend it and assign a new quota
	tempConfig["name"] = config.StringVariable(nameMaxUpdated)
	tempConfig["quota_id"] = config.StringVariable(quotaIdMaxUpdated)
	tempConfig["suspended"] = config.BoolVariable(!suspendedMax)
	return tempConfig
}

func TestAccScfOrganizationMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScfOrganizationDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.ScfProviderConfig() + resourceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "platform_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "org_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "quota_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "region"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "status"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "suspended"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "updated_at"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "org_id"),
					resource.TestCheckResourceAttr("stackit_scf_organization_manager.orgmanager", "platform_id", testutil.ConvertConfigVariable(testConfigVarsMax["platform_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization_manager.orgmanager", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "user_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "username"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "password"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "updated_at"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
					%s
					data "stackit_scf_organization" "org" {
						project_id  = stackit_scf_organization.org.project_id
						org_id = stackit_scf_organization.org.org_id
					}
					data "stackit_scf_organization_manager" "orgmanager" {
	                	org_id = stackit_scf_organization.org.org_id
	                	project_id = stackit_scf_organization.org.project_id
	                }
					`, testutil.ScfProviderConfig()+resourceMin,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_scf_organization.org", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "project_id",
						"data.stackit_scf_organization.org", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "created_at",
						"data.stackit_scf_organization.org", "created_at",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "name",
						"data.stackit_scf_organization.org", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "platform_id",
						"data.stackit_scf_organization.org", "platform_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "org_id",
						"data.stackit_scf_organization.org", "org_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "quota_id",
						"data.stackit_scf_organization.org", "quota_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "region",
						"data.stackit_scf_organization.org", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "status",
						"data.stackit_scf_organization.org", "status",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "suspended",
						"data.stackit_scf_organization.org", "suspended",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "updated_at",
						"data.stackit_scf_organization.org", "updated_at",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "region",
						"data.stackit_scf_organization_manager.orgmanager", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "platform_id",
						"data.stackit_scf_organization_manager.orgmanager", "platform_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "project_id",
						"data.stackit_scf_organization_manager.orgmanager", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "org_id",
						"data.stackit_scf_organization_manager.orgmanager", "org_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_scf_organization_manager.orgmanager", "user_id"),
					resource.TestCheckResourceAttrSet("data.stackit_scf_organization_manager.orgmanager", "username"),
					resource.TestCheckResourceAttrSet("data.stackit_scf_organization_manager.orgmanager", "created_at"),
					resource.TestCheckResourceAttrSet("data.stackit_scf_organization_manager.orgmanager", "updated_at"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_scf_organization.org",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_scf_organization.org"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_scf_organization.org")
					}
					orgId, ok := r.Primary.Attributes["org_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute org_id")
					}
					regionInAttributes, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, regionInAttributes, orgId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testScfOrgConfigVarsMinUpdated(),
				Config:          testutil.ScfProviderConfig() + resourceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "project_id", testutil.ConvertConfigVariable(testScfOrgConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "name", testutil.ConvertConfigVariable(testScfOrgConfigVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "platform_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "org_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "quota_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "region"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "suspended"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "updated_at"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccScfOrgMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScfOrganizationDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.ScfProviderConfig() + resourceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "platform_id", testutil.ConvertConfigVariable(testConfigVarsMax["platform_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "quota_id", testutil.ConvertConfigVariable(testConfigVarsMax["quota_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "suspended", testutil.ConvertConfigVariable(testConfigVarsMax["suspended"])),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "org_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "region"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "updated_at"),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.scf_platform", "platform_id", testutil.ConvertConfigVariable(testConfigVarsMax["platform_id"])),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.scf_platform", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.scf_platform", "display_name", platformName),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.scf_platform", "system_id", platformSystemId),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.scf_platform", "api_url", platformApiUrl),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.scf_platform", "console_url", platformConsoleUrl),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "org_id"),
					resource.TestCheckResourceAttr("stackit_scf_organization_manager.orgmanager", "platform_id", testutil.ConvertConfigVariable(testConfigVarsMax["platform_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization_manager.orgmanager", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "user_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "username"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "password"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization_manager.orgmanager", "updated_at"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
					%s
					data "stackit_scf_organization" "org" {
						project_id  = stackit_scf_organization.org.project_id
						org_id = stackit_scf_organization.org.org_id
						region = var.region
					}
					data "stackit_scf_organization_manager" "orgmanager" {
	                	org_id = stackit_scf_organization.org.org_id
	                	project_id = stackit_scf_organization.org.project_id
	                }
					data "stackit_scf_platform" "platform" {
	                	platform_id = stackit_scf_organization.org.platform_id
	                	project_id = stackit_scf_organization.org.project_id
	                }
					`, testutil.ScfProviderConfig()+resourceMax,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_scf_organization.org", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "project_id",
						"data.stackit_scf_organization.org", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "created_at",
						"data.stackit_scf_organization.org", "created_at",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "name",
						"data.stackit_scf_organization.org", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "platform_id",
						"data.stackit_scf_organization.org", "platform_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "org_id",
						"data.stackit_scf_organization.org", "org_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "quota_id",
						"data.stackit_scf_organization.org", "quota_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "region",
						"data.stackit_scf_organization.org", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "status",
						"data.stackit_scf_organization.org", "status",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "suspended",
						"data.stackit_scf_organization.org", "suspended",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "updated_at",
						"data.stackit_scf_organization.org", "updated_at",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "platform_id",
						"data.stackit_scf_platform.platform", "platform_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "project_id",
						"data.stackit_scf_platform.platform", "project_id",
					),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.platform", "display_name", platformName),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.platform", "system_id", platformSystemId),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.platform", "display_name", platformName),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.platform", "region", region),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.platform", "api_url", platformApiUrl),
					resource.TestCheckResourceAttr("data.stackit_scf_platform.platform", "console_url", platformConsoleUrl),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "region",
						"data.stackit_scf_organization_manager.orgmanager", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "platform_id",
						"data.stackit_scf_organization_manager.orgmanager", "platform_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "project_id",
						"data.stackit_scf_organization_manager.orgmanager", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_scf_organization.org", "org_id",
						"data.stackit_scf_organization_manager.orgmanager", "org_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_scf_organization_manager.orgmanager", "user_id"),
					resource.TestCheckResourceAttrSet("data.stackit_scf_organization_manager.orgmanager", "username"),
					resource.TestCheckResourceAttrSet("data.stackit_scf_organization_manager.orgmanager", "created_at"),
					resource.TestCheckResourceAttrSet("data.stackit_scf_organization_manager.orgmanager", "updated_at"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_scf_organization.org",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_scf_organization.org"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_scf_organization.org")
					}
					orgId, ok := r.Primary.Attributes["org_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute org_id")
					}
					regionInAttributes, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, regionInAttributes, orgId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testScfOrgConfigVarsMaxUpdated(),
				Config:          testutil.ScfProviderConfig() + resourceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "name", testutil.ConvertConfigVariable(testScfOrgConfigVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "platform_id", testutil.ConvertConfigVariable(testConfigVarsMax["platform_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "quota_id", testutil.ConvertConfigVariable(testScfOrgConfigVarsMaxUpdated()["quota_id"])),
					resource.TestCheckResourceAttr("stackit_scf_organization.org", "suspended", testutil.ConvertConfigVariable(testScfOrgConfigVarsMaxUpdated()["suspended"])),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "org_id"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "region"),
					resource.TestCheckResourceAttrSet("stackit_scf_organization.org", "updated_at"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

// Run apply and fail in the waiter. We expect that the IDs are saved in the state.
// Verify this in the second step by refreshing and checking the IDs in the URL.
func TestScfOrganizationSavesIDsOnError(t *testing.T) {
	var (
		projectId = uuid.NewString()
		guid      = uuid.NewString()
	)
	const name = "scf-org-error-test"
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
  default_region = "eu01"
  scf_custom_endpoint = "%s"
  service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_scf_organization" "org" {
  project_id = "%s"
  name       = "%s"
}
`, s.Server.URL, projectId, name)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create",
							ToJsonBody: &scf.OrganizationCreateResponse{
								Guid: utils.Ptr(guid),
							},
						},
						testutil.MockResponse{Description: "create waiter", StatusCode: http.StatusNotFound},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating scf organization.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/organizations/%s", projectId, region, guid)
								if req.URL.Path != expected {
									t.Errorf("Expected request to %s but got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete"},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusNotFound},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading scf organization.*"),
			},
		},
	})
}

func testAccCheckScfOrganizationDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *scf.APIClient
	var err error

	if testutil.ScfCustomEndpoint == "" {
		client, err = scf.NewAPIClient()
	} else {
		client, err = scf.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.ScfCustomEndpoint),
		)
	}

	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var orgsToDestroy []string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_scf_organization" {
			continue
		}
		orgId := strings.Split(rs.Primary.ID, core.Separator)[1]
		orgsToDestroy = append(orgsToDestroy, orgId)
	}

	organizationsList, err := client.ListOrganizations(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting scf organizations: %w", err)
	}

	scfOrgs := organizationsList.GetResources()
	for i := range scfOrgs {
		if scfOrgs[i].Guid == nil {
			continue
		}
		if utils.Contains(orgsToDestroy, *scfOrgs[i].Guid) {
			_, err := client.DeleteOrganizationExecute(ctx, testutil.ProjectId, testutil.Region, *scfOrgs[i].Guid)
			if err != nil {
				return fmt.Errorf("destroying scf organization %s during CheckDestroy: %w", *scfOrgs[i].Guid, err)
			}
		}
	}
	return nil
}
