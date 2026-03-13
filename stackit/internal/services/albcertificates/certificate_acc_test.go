package alb_test

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	certSdk "github.com/stackitcloud/stackit-sdk-go/services/certificates"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testfiles/resource-min.tf
var resourceMinConfig string

//go:embed testfiles/resource-max.tf
var resourceMaxConfig string

var testConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"cert_name":  config.StringVariable(fmt.Sprintf("tf-acc-l%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
}

var testConfigVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"region":     config.StringVariable(testutil.Region),
	"cert_name":  config.StringVariable(fmt.Sprintf("tf-acc-l%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
}

func TestAccCertResourceMin(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"tls": {
				Source:            "hashicorp/tls",
				VersionConstraint: "4.0.4", // Use a specific version to avoid lock issues
			},
		},
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCertDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.CertProviderConfig() + resourceMinConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance resource
					resource.TestCheckResourceAttr("stackit_alb_certificate.certificate", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_alb_certificate.certificate", "name", testutil.ConvertConfigVariable(testConfigVarsMin["cert_name"])),
					resource.TestCheckResourceAttrSet("stackit_alb_certificate.certificate", "public_key"),
					resource.TestCheckResourceAttrSet("stackit_alb_certificate.certificate", "private_key"),
					resource.TestCheckResourceAttrSet("stackit_alb_certificate.certificate", "region"),
					resource.TestCheckResourceAttrSet("stackit_alb_certificate.certificate", "id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
						%s

						data "stackit_alb_certificate" "certificate" {
							project_id     = stackit_alb_certificate.certificate.project_id
							cert_id    = stackit_alb_certificate.certificate.cert_id
						}
						`,
					testutil.CertProviderConfig()+resourceMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance
					resource.TestCheckResourceAttr("data.stackit_alb_certificate.certificate", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_alb_certificate.certificate", "name", testutil.ConvertConfigVariable(testConfigVarsMin["cert_name"])),
					resource.TestCheckResourceAttrSet("data.stackit_alb_certificate.certificate", "public_key"),
					resource.TestCheckResourceAttrSet("data.stackit_alb_certificate.certificate", "region"),
					resource.TestCheckResourceAttrSet("data.stackit_alb_certificate.certificate", "id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_certificate.certificate", "project_id",
						"stackit_alb_certificate.certificate", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_certificate.certificate", "region",
						"stackit_alb_certificate.certificate", "region",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_certificate.certificate", "name",
						"stackit_alb_certificate.certificate", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_certificate.certificate", "public_key",
						"stackit_alb_certificate.certificate", "public_key",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_alb_certificate.certificate",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_alb_certificate.certificate"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_alb_certificate.certificate")
					}
					certID, ok := r.Primary.Attributes["cert_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, region, certID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore the sensitive field during verification, because the API doesn't return the key
				ImportStateVerifyIgnore: []string{"private_key"},
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccCertResourceMax(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"tls": {
				Source:            "hashicorp/tls",
				VersionConstraint: "4.0.4", // Use a specific version to avoid lock issues
			},
		},
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCertDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.CertProviderConfig() + resourceMaxConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance resource
					resource.TestCheckResourceAttr("stackit_alb_certificate.certificate", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_alb_certificate.certificate", "name", testutil.ConvertConfigVariable(testConfigVarsMax["cert_name"])),
					resource.TestCheckResourceAttr("stackit_alb_certificate.certificate", "region", testutil.ConvertConfigVariable(testConfigVarsMax["region"])),
					resource.TestCheckResourceAttrSet("stackit_alb_certificate.certificate", "public_key"),
					resource.TestCheckResourceAttrSet("stackit_alb_certificate.certificate", "private_key"),
					resource.TestCheckResourceAttrSet("stackit_alb_certificate.certificate", "id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
						%s

						data "stackit_alb_certificate" "certificate" {
							project_id     = stackit_alb_certificate.certificate.project_id
							cert_id    = stackit_alb_certificate.certificate.cert_id
						}
						`,
					testutil.CertProviderConfig()+resourceMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Load balancer instance
					resource.TestCheckResourceAttr("data.stackit_alb_certificate.certificate", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_alb_certificate.certificate", "name", testutil.ConvertConfigVariable(testConfigVarsMax["cert_name"])),
					resource.TestCheckResourceAttr("data.stackit_alb_certificate.certificate", "region", testutil.ConvertConfigVariable(testConfigVarsMax["region"])),
					resource.TestCheckResourceAttrSet("data.stackit_alb_certificate.certificate", "public_key"),
					resource.TestCheckResourceAttrSet("data.stackit_alb_certificate.certificate", "id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_certificate.certificate", "project_id",
						"stackit_alb_certificate.certificate", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_certificate.certificate", "region",
						"stackit_alb_certificate.certificate", "region",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_certificate.certificate", "name",
						"stackit_alb_certificate.certificate", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_alb_certificate.certificate", "public_key",
						"stackit_alb_certificate.certificate", "public_key",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_alb_certificate.certificate",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_alb_certificate.certificate"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_alb_certificate.certificate")
					}
					certID, ok := r.Primary.Attributes["cert_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, region, certID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore the sensitive field during verification, because the API doesn't return the key
				ImportStateVerifyIgnore: []string{"private_key"},
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckCertDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *certSdk.APIClient
	var err error
	if testutil.ALBCustomEndpoint == "" {
		client, err = certSdk.NewAPIClient()
	} else {
		client, err = certSdk.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.CertCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	region := "eu01"
	if testutil.Region != "" {
		region = testutil.Region
	}
	certificateToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_loadbalancer" {
			continue
		}
		// cetificate terraform ID: = "[project_id],[region],[name]"
		certificateName := strings.Split(rs.Primary.ID, core.Separator)[1]
		certificateToDestroy = append(certificateToDestroy, certificateName)
	}

	certificateResp, err := client.ListCertificates(ctx, testutil.ProjectId, region).Execute()
	if err != nil {
		return fmt.Errorf("getting certificateResp: %w", err)
	}

	if certificateResp.Items == nil || (certificateResp.Items != nil && len(*certificateResp.Items) == 0) {
		fmt.Print("No certificates found for project \n")
		return nil
	}

	items := *certificateResp.Items
	for i := range items {
		if items[i].Name == nil {
			continue
		}
		if utils.Contains(certificateToDestroy, *items[i].Name) {
			_, err := client.DeleteCertificateExecute(ctx, testutil.ProjectId, region, *items[i].Id)
			if err != nil {
				return fmt.Errorf("destroying certificate %s during CheckDestroy: %w", *items[i].Name, err)
			}
		}
	}
	return nil
}
