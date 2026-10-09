package ufw_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	ufw "github.com/stackitcloud/stackit-sdk-go/services/ufw/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/ufw_instance_min.tf
	resourceConfigMin string

	//go:embed testdata/ufw_instance_max.tf
	resourceConfigMax string
)

var testConfigUfwVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"source_ip":  config.StringVariable("192.168.0.0/24"),
}

var testConfigUfwVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigUfwVarsMin)
	updatedConfig["source_ip"] = config.StringVariable("10.0.0.0/8")
	return updatedConfig
}

var testConfigUfwVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"region":     config.StringVariable(testutil.Region),
	"source_ip":  config.StringVariable("192.168.0.0/24"),
}

var testConfigUfwVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigUfwVarsMax)
	updatedConfig["source_ip"] = config.StringVariable("10.0.0.0/8")
	return updatedConfig
}

func TestAccUfwInstanceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			{
				ConfigVariables: testConfigUfwVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceConfigMin),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "product", "edge-cloud"),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "source_ip", "192.168.0.0/24"),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "type", "ACL"),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "region", testutil.Region),
					resource.TestCheckResourceAttrSet("stackit_ufw_instance.example", "rule_id"),
					resource.TestCheckResourceAttrSet("stackit_ufw_instance.example", "instance_id"),
				),
			},
			{
				ConfigVariables: testConfigUfwVarsMin,
				Config: fmt.Sprintf(`
                %s
                %s

                data "stackit_ufw_instance" "example" {
                  project_id = stackit_ufw_instance.example.project_id
                  region     = stackit_ufw_instance.example.region
                  rule_id    = stackit_ufw_instance.example.rule_id
                }
                `,
					testutil.NewConfigBuilder().BuildProviderConfig(), resourceConfigMin,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "region", testutil.Region),
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "source_ip", "192.168.0.0/24"),
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "product", "edge-cloud"),
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "type", "ACL"),
					resource.TestCheckResourceAttrPair(
						"stackit_ufw_instance.example", "rule_id",
						"data.stackit_ufw_instance.example", "rule_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_ufw_instance.example", "instance_id",
						"data.stackit_ufw_instance.example", "instance_id",
					),
				),
			},
			{
				ConfigVariables: testConfigUfwVarsMin,
				ResourceName:    "stackit_ufw_instance.example",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_ufw_instance.example"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_ufw_instance.example")
					}
					ruleId, ok := r.Primary.Attributes["rule_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute rule_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, region, ruleId), nil
				},
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "rule_id",
			},
			{
				ConfigVariables: testConfigUfwVarsMinUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceConfigMin),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_ufw_instance.example", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "source_ip", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccUfwInstanceMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			{
				ConfigVariables: testConfigUfwVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceConfigMax),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "product", "edge-cloud"),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "source_ip", "192.168.0.0/24"),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "type", "ACL"),
					resource.TestCheckResourceAttrSet("stackit_ufw_instance.example", "rule_id"),
					resource.TestCheckResourceAttrSet("stackit_ufw_instance.example", "instance_id"),
				),
			},
			{
				ConfigVariables: testConfigUfwVarsMax,
				Config: fmt.Sprintf(`
                %s
                %s

                data "stackit_ufw_instance" "example" {
                  project_id = stackit_ufw_instance.example.project_id
                  region     = stackit_ufw_instance.example.region
                  rule_id    = stackit_ufw_instance.example.rule_id
                }
                `,
					testutil.NewConfigBuilder().BuildProviderConfig(), resourceConfigMax,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "region", testutil.Region),
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "source_ip", "192.168.0.0/24"),
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "product", "edge-cloud"),
					resource.TestCheckResourceAttr("data.stackit_ufw_instance.example", "type", "ACL"),
					resource.TestCheckResourceAttrPair(
						"stackit_ufw_instance.example", "rule_id",
						"data.stackit_ufw_instance.example", "rule_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_ufw_instance.example", "instance_id",
						"data.stackit_ufw_instance.example", "instance_id",
					),
					resource.TestCheckNoResourceAttr("data.stackit_ufw_instance.example", "description"),
					resource.TestCheckNoResourceAttr("data.stackit_ufw_instance.example", "direction"),
				),
			},
			{
				ConfigVariables: testConfigUfwVarsMax,
				ResourceName:    "stackit_ufw_instance.example",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_ufw_instance.example"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_ufw_instance.example")
					}
					ruleId, ok := r.Primary.Attributes["rule_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute rule_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, region, ruleId), nil
				},
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "rule_id",
			},
			{
				ConfigVariables: testConfigUfwVarsMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().BuildProviderConfig(), resourceConfigMax),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_ufw_instance.example", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "source_ip", "10.0.0.0/8"),
				),
			},
		},
	})
}

func createClient() (*ufw.APIClient, error) {
	client, err := ufw.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.UfwCustomEndpoint, false)...)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return client, nil
}

func testAccCheckDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := createClient()
	if err != nil {
		return err
	}

	var errs []error

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_ufw_instance" {
			continue
		}

		projectId := rs.Primary.Attributes["project_id"]
		region := rs.Primary.Attributes["region"]
		ruleId := rs.Primary.Attributes["rule_id"]

		if projectId == "" || region == "" || ruleId == "" {
			continue
		}

		_, err := client.DefaultAPI.GetRule(ctx, projectId, region, ruleId).Execute()
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			ok := errors.As(err, &oapiErr)
			if !ok || oapiErr.StatusCode != http.StatusNotFound {
				errs = append(errs, fmt.Errorf("getting rule %s during CheckDestroy: %w", ruleId, err))
			}
			continue
		}

		errs = append(errs, fmt.Errorf("UFW rule %s still exists", ruleId))
	}
	return errors.Join(errs...)
}
