package kms_test

import (
	_ "embed"
	"fmt"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/keyring-min.tf
	resourceKeyRingMinConfig string

	//go:embed testdata/keyring-max.tf
	resourceKeyRingMaxConfig string
)

var testConfigKeyRingVarsMin = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
}

var testConfigKeyRingVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigKeyRingVarsMin)
	updatedConfig["display_name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["display_name"])))
	return updatedConfig
}

var testConfigKeyRingVarsMax = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"description":  config.StringVariable("description"),
	"display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
}

var testConfigKeyRingVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigKeyRingVarsMax)
	updatedConfig["description"] = config.StringVariable("updated description")
	updatedConfig["display_name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["display_name"])))
	return updatedConfig
}

func TestAccKeyRingMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigKeyRingVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_kms_keyring.keyring", "keyring_id"),
					resource.TestCheckNoResourceAttr("stackit_kms_keyring.keyring", "description"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigKeyRingVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_kms_keyring" "keyring" {
						project_id = stackit_kms_keyring.keyring.project_id
						keyring_id = stackit_kms_keyring.keyring.keyring_id
					}
					`,
					testutil.KMSProviderConfig(), resourceKeyRingMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_kms_keyring.keyring", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_kms_keyring.keyring", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_keyring.keyring", "keyring_id",
						"data.stackit_kms_keyring.keyring", "keyring_id",
					),
					resource.TestCheckResourceAttr("data.stackit_kms_keyring.keyring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["display_name"])),
					resource.TestCheckNoResourceAttr("data.stackit_kms_keyring.keyring", "description"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigKeyRingVarsMin,
				ResourceName:    "stackit_kms_keyring.keyring",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_kms_keyring.keyring"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_kms_keyring.keyring")
					}
					keyRingId, ok := r.Primary.Attributes["keyring_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute keyring_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, keyRingId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigKeyRingVarsMinUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMinUpdated()["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_kms_keyring.keyring", "keyring_id"),
					resource.TestCheckNoResourceAttr("stackit_kms_keyring.keyring", "description"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccKeyRingMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigKeyRingVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "description", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["description"])),
					resource.TestCheckResourceAttrSet("stackit_kms_keyring.keyring", "keyring_id"),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["display_name"])),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigKeyRingVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_kms_keyring" "keyring" {
						project_id = stackit_kms_keyring.keyring.project_id
						keyring_id = stackit_kms_keyring.keyring.keyring_id
					}
					`,
					testutil.KMSProviderConfig(), resourceKeyRingMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.stackit_kms_keyring.keyring", "project_id", testutil.ProjectId),
						resource.TestCheckResourceAttr("data.stackit_kms_keyring.keyring", "region", testutil.Region),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_keyring.keyring", "keyring_id",
							"data.stackit_kms_keyring.keyring", "keyring_id",
						),
						resource.TestCheckResourceAttr("data.stackit_kms_keyring.keyring", "description", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["description"])),
						resource.TestCheckResourceAttr("data.stackit_kms_keyring.keyring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["display_name"])),
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigKeyRingVarsMax,
				ResourceName:    "stackit_kms_keyring.keyring",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_kms_keyring.keyring"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_kms_keyring.keyring")
					}
					keyRingId, ok := r.Primary.Attributes["keyring_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute keyring_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, keyRingId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigKeyRingVarsMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMaxUpdated()["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_kms_keyring.keyring", "keyring_id"),
					resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "description", testutil.ConvertConfigVariable(testConfigKeyRingVarsMaxUpdated()["description"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}
