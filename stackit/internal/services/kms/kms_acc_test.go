package kms_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	coreConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

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

	//go:embed testdata/key-min.tf
	resourceKeyMinConfig string

	//go:embed testdata/key-max.tf
	resourceKeyMaxConfig string

	//go:embed testdata/wrapping-key-min.tf
	resourceWrappingKeyMinConfig string

	//go:embed testdata/wrapping-key-max.tf
	resourceWrappingKeyMaxConfig string
)

// KEY RING - MIN

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

// KEY RING - MAX

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

// KEY - MIN

var testConfigKeyVarsMin = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"keyring_display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"display_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"algorithm":            config.StringVariable(string(kms.ALGORITHM_AES_256_GCM)),
	"protection":           config.StringVariable("software"),
	"purpose":              config.StringVariable(string(kms.PURPOSE_SYMMETRIC_ENCRYPT_DECRYPT)),
}

var testConfigKeyVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigKeyVarsMin)
	updatedConfig["display_name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["display_name"])))
	updatedConfig["algorithm"] = config.StringVariable(string(kms.ALGORITHM_RSA_3072_OAEP_SHA256))
	updatedConfig["purpose"] = config.StringVariable(string(kms.PURPOSE_ASYMMETRIC_ENCRYPT_DECRYPT))
	return updatedConfig
}

// KEY - MAX

var testConfigKeyVarsMax = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"keyring_display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"display_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"algorithm":            config.StringVariable(string(kms.ALGORITHM_AES_256_GCM)),
	"protection":           config.StringVariable("software"),
	"purpose":              config.StringVariable(string(kms.PURPOSE_SYMMETRIC_ENCRYPT_DECRYPT)),
	"access_scope":         config.StringVariable(string(kms.ACCESSSCOPE_PUBLIC)),
	"import_only":          config.BoolVariable(true),
	"description":          config.StringVariable("kms-key-description"),
}

var testConfigKeyVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigKeyVarsMax)
	updatedConfig["display_name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["display_name"])))
	updatedConfig["algorithm"] = config.StringVariable(string(kms.ALGORITHM_RSA_3072_OAEP_SHA256))
	updatedConfig["purpose"] = config.StringVariable(string(kms.PURPOSE_ASYMMETRIC_ENCRYPT_DECRYPT))
	updatedConfig["import_only"] = config.BoolVariable(true)
	updatedConfig["description"] = config.StringVariable("kms-key-description-updated")
	return updatedConfig
}

// WRAPPING KEY - MIN

var testConfigWrappingKeyVarsMin = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"keyring_display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"display_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"algorithm":            config.StringVariable(string(kms.WRAPPINGALGORITHM__2048_OAEP_SHA256)),
	"protection":           config.StringVariable(string(kms.PROTECTION_SOFTWARE)),
	"purpose":              config.StringVariable(string(kms.WRAPPINGPURPOSE_SYMMETRIC_KEY)),
}

var testConfigWrappingKeyVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigWrappingKeyVarsMin)
	updatedConfig["display_name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["display_name"])))
	updatedConfig["algorithm"] = config.StringVariable(string(kms.WRAPPINGALGORITHM__4096_OAEP_SHA256_AES_256_KEY_WRAP))
	updatedConfig["purpose"] = config.StringVariable(string(kms.WRAPPINGPURPOSE_ASYMMETRIC_KEY))
	return updatedConfig
}

// WRAPPING KEY - MAX

var testConfigWrappingKeyVarsMax = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"keyring_display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"display_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"algorithm":            config.StringVariable(string(kms.WRAPPINGALGORITHM__2048_OAEP_SHA256)),
	"protection":           config.StringVariable(string(kms.PROTECTION_SOFTWARE)),
	"purpose":              config.StringVariable(string(kms.WRAPPINGPURPOSE_SYMMETRIC_KEY)),
	"description":          config.StringVariable("kms-wrapping-key-description"),
	"access_scope":         config.StringVariable(string(kms.ACCESSSCOPE_PUBLIC)),
}

var testConfigWrappingKeyVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigWrappingKeyVarsMax)
	updatedConfig["display_name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["display_name"])))
	updatedConfig["algorithm"] = config.StringVariable(string(kms.WRAPPINGALGORITHM__4096_OAEP_SHA256_AES_256_KEY_WRAP))
	updatedConfig["purpose"] = config.StringVariable(string(kms.WRAPPINGPURPOSE_ASYMMETRIC_KEY))
	updatedConfig["description"] = config.StringVariable("kms-wrapping-key-description-updated")
	return updatedConfig
}

func TestAccKeyRingMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigKeyRingVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionCreate),
					},
				},
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
					},
				},
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionReplace),
					},
				},
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
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigKeyRingVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionCreate),
					},
				},
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
					},
				},
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionReplace),
					},
				},
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

func TestAccKeyMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigKeyVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_kms_key.key", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_key.key", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_keyring.keyring", "keyring_id",
						"stackit_kms_key.key", "keyring_id",
					),
					resource.TestCheckResourceAttrSet("stackit_kms_key.key", "key_id"),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "algorithm", testutil.ConvertConfigVariable(testConfigKeyVarsMin["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "display_name", testutil.ConvertConfigVariable(testConfigKeyVarsMin["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "purpose", testutil.ConvertConfigVariable(testConfigKeyVarsMin["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "protection", testutil.ConvertConfigVariable(testConfigKeyVarsMin["protection"])),
					resource.TestCheckNoResourceAttr("stackit_kms_key.key", "description"),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "access_scope", string(kms.ACCESSSCOPE_PUBLIC)),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "import_only", "false"),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigKeyVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_kms_key" "key" {
						project_id = stackit_kms_key.key.project_id
						keyring_id = stackit_kms_key.key.keyring_id
						key_id = stackit_kms_key.key.key_id
					}
					`,
					testutil.KMSProviderConfig(), resourceKeyMinConfig,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_kms_key.key", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "project_id", testutil.ProjectId),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "region", testutil.Region),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_keyring.keyring", "keyring_id",
							"data.stackit_kms_key.key", "keyring_id",
						),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_key.key", "key_id",
							"data.stackit_kms_key.key", "key_id",
						),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "algorithm", testutil.ConvertConfigVariable(testConfigKeyVarsMin["algorithm"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "display_name", testutil.ConvertConfigVariable(testConfigKeyVarsMin["display_name"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "purpose", testutil.ConvertConfigVariable(testConfigKeyVarsMin["purpose"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "protection", testutil.ConvertConfigVariable(testConfigKeyVarsMin["protection"])),
						resource.TestCheckNoResourceAttr("data.stackit_kms_key.key", "description"),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "access_scope", string(kms.ACCESSSCOPE_PUBLIC)),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "import_only", "false"),
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigKeyVarsMin,
				ResourceName:    "stackit_kms_key.key",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_kms_key.key"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_kms_key.key")
					}
					keyRingId, ok := r.Primary.Attributes["keyring_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute keyring_id")
					}
					keyId, ok := r.Primary.Attributes["key_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute key_id")
					}

					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, keyRingId, keyId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigKeyVarsMinUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_kms_key.key", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_key.key", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_keyring.keyring", "keyring_id",
						"stackit_kms_key.key", "keyring_id",
					),
					resource.TestCheckResourceAttrSet("stackit_kms_key.key", "key_id"),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "algorithm", testutil.ConvertConfigVariable(testConfigKeyVarsMinUpdated()["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "display_name", testutil.ConvertConfigVariable(testConfigKeyVarsMinUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "purpose", testutil.ConvertConfigVariable(testConfigKeyVarsMinUpdated()["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "protection", testutil.ConvertConfigVariable(testConfigKeyVarsMinUpdated()["protection"])),
					resource.TestCheckNoResourceAttr("stackit_kms_key.key", "description"),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "access_scope", string(kms.ACCESSSCOPE_PUBLIC)),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "import_only", "false"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccKeyMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigKeyVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_kms_key.key", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_key.key", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_keyring.keyring", "keyring_id",
						"stackit_kms_key.key", "keyring_id",
					),
					resource.TestCheckResourceAttrSet("stackit_kms_key.key", "key_id"),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "algorithm", testutil.ConvertConfigVariable(testConfigKeyVarsMax["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "display_name", testutil.ConvertConfigVariable(testConfigKeyVarsMax["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "purpose", testutil.ConvertConfigVariable(testConfigKeyVarsMax["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "protection", testutil.ConvertConfigVariable(testConfigKeyVarsMax["protection"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "description", testutil.ConvertConfigVariable(testConfigKeyVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "access_scope", testutil.ConvertConfigVariable(testConfigKeyVarsMax["access_scope"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "import_only", testutil.ConvertConfigVariable(testConfigKeyVarsMax["import_only"])),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigKeyVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_kms_key" "key" {
						project_id = stackit_kms_key.key.project_id
						keyring_id = stackit_kms_key.key.keyring_id
						key_id = stackit_kms_key.key.key_id
					}
					`,
					testutil.KMSProviderConfig(), resourceKeyMaxConfig,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_kms_key.key", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "project_id", testutil.ProjectId),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "region", testutil.Region),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_keyring.keyring", "keyring_id",
							"data.stackit_kms_key.key", "keyring_id",
						),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_key.key", "key_id",
							"data.stackit_kms_key.key", "key_id",
						),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "algorithm", testutil.ConvertConfigVariable(testConfigKeyVarsMax["algorithm"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "display_name", testutil.ConvertConfigVariable(testConfigKeyVarsMax["display_name"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "purpose", testutil.ConvertConfigVariable(testConfigKeyVarsMax["purpose"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "protection", testutil.ConvertConfigVariable(testConfigKeyVarsMax["protection"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "description", testutil.ConvertConfigVariable(testConfigKeyVarsMax["description"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "access_scope", testutil.ConvertConfigVariable(testConfigKeyVarsMax["access_scope"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key.key", "import_only", testutil.ConvertConfigVariable(testConfigKeyVarsMax["import_only"])),
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigKeyVarsMax,
				ResourceName:    "stackit_kms_key.key",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_kms_key.key"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_kms_key.key")
					}
					keyRingId, ok := r.Primary.Attributes["keyring_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute keyring_id")
					}
					keyId, ok := r.Primary.Attributes["key_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute key_id")
					}

					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, keyRingId, keyId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigKeyVarsMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_kms_key.key", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_key.key", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_keyring.keyring", "keyring_id",
						"stackit_kms_key.key", "keyring_id",
					),
					resource.TestCheckResourceAttrSet("stackit_kms_key.key", "key_id"),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "algorithm", testutil.ConvertConfigVariable(testConfigKeyVarsMaxUpdated()["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "display_name", testutil.ConvertConfigVariable(testConfigKeyVarsMaxUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "purpose", testutil.ConvertConfigVariable(testConfigKeyVarsMaxUpdated()["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "protection", testutil.ConvertConfigVariable(testConfigKeyVarsMaxUpdated()["protection"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "description", testutil.ConvertConfigVariable(testConfigKeyVarsMaxUpdated()["description"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "access_scope", testutil.ConvertConfigVariable(testConfigKeyVarsMaxUpdated()["access_scope"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "import_only", testutil.ConvertConfigVariable(testConfigKeyVarsMaxUpdated()["import_only"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccWrappingKeyMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigWrappingKeyVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceWrappingKeyMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_kms_wrapping_key.wrapping_key", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_keyring.keyring", "keyring_id",
						"stackit_kms_wrapping_key.wrapping_key", "keyring_id",
					),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "wrapping_key_id"),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "algorithm", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "display_name", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "purpose", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "protection", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["protection"])),
					resource.TestCheckNoResourceAttr("stackit_kms_wrapping_key.wrapping_key", "description"),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "access_scope", string(kms.ACCESSSCOPE_PUBLIC)),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "public_key"),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "expires_at"),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "created_at"),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigWrappingKeyVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_kms_wrapping_key" "wrapping_key" {
						project_id = stackit_kms_wrapping_key.wrapping_key.project_id
						keyring_id = stackit_kms_wrapping_key.wrapping_key.keyring_id
						wrapping_key_id = stackit_kms_wrapping_key.wrapping_key.wrapping_key_id
					}
					`,
					testutil.KMSProviderConfig(), resourceWrappingKeyMinConfig,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_kms_wrapping_key.wrapping_key", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "project_id", testutil.ProjectId),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "region", testutil.Region),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_keyring.keyring", "keyring_id",
							"data.stackit_kms_wrapping_key.wrapping_key", "keyring_id",
						),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_wrapping_key.wrapping_key", "wrapping_key_id",
							"data.stackit_kms_wrapping_key.wrapping_key", "wrapping_key_id",
						),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "algorithm", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["algorithm"])),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "display_name", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["display_name"])),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "purpose", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["purpose"])),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "protection", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["protection"])),
						resource.TestCheckNoResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "description"),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "access_scope", string(kms.ACCESSSCOPE_PUBLIC)),
						resource.TestCheckResourceAttrSet("data.stackit_kms_wrapping_key.wrapping_key", "public_key"),
						resource.TestCheckResourceAttrSet("data.stackit_kms_wrapping_key.wrapping_key", "expires_at"),
						resource.TestCheckResourceAttrSet("data.stackit_kms_wrapping_key.wrapping_key", "created_at"),
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigWrappingKeyVarsMin,
				ResourceName:    "stackit_kms_wrapping_key.wrapping_key",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_kms_wrapping_key.wrapping_key"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_kms_wrapping_key.wrapping_key")
					}
					keyRingId, ok := r.Primary.Attributes["keyring_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute keyring_id")
					}
					wrappingKeyId, ok := r.Primary.Attributes["wrapping_key_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute wrapping_key_id")
					}

					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, keyRingId, wrappingKeyId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigWrappingKeyVarsMinUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceWrappingKeyMinConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_kms_wrapping_key.wrapping_key", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_keyring.keyring", "keyring_id",
						"stackit_kms_wrapping_key.wrapping_key", "keyring_id",
					),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "wrapping_key_id"),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "algorithm", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMinUpdated()["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "display_name", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMinUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "purpose", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMinUpdated()["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "protection", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMinUpdated()["protection"])),
					resource.TestCheckNoResourceAttr("stackit_kms_wrapping_key.wrapping_key", "description"),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "access_scope", string(kms.ACCESSSCOPE_PUBLIC)),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "public_key"),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "expires_at"),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "created_at"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccWrappingKeyMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigWrappingKeyVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceWrappingKeyMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_kms_wrapping_key.wrapping_key", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_keyring.keyring", "keyring_id",
						"stackit_kms_wrapping_key.wrapping_key", "keyring_id",
					),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "wrapping_key_id"),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "algorithm", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "display_name", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "purpose", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "protection", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["protection"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "description", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "access_scope", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["access_scope"])),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "public_key"),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "expires_at"),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "created_at"),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigWrappingKeyVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_kms_wrapping_key" "wrapping_key" {
						project_id = stackit_kms_wrapping_key.wrapping_key.project_id
						keyring_id = stackit_kms_wrapping_key.wrapping_key.keyring_id
						wrapping_key_id = stackit_kms_wrapping_key.wrapping_key.wrapping_key_id
					}
					`,
					testutil.KMSProviderConfig(), resourceWrappingKeyMaxConfig,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_kms_wrapping_key.wrapping_key", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "project_id", testutil.ProjectId),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "region", testutil.Region),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_keyring.keyring", "keyring_id",
							"data.stackit_kms_wrapping_key.wrapping_key", "keyring_id",
						),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_wrapping_key.wrapping_key", "wrapping_key_id",
							"data.stackit_kms_wrapping_key.wrapping_key", "wrapping_key_id",
						),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "algorithm", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["algorithm"])),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "display_name", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["display_name"])),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "purpose", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["purpose"])),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "protection", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["protection"])),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "description", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["description"])),
						resource.TestCheckResourceAttr("data.stackit_kms_wrapping_key.wrapping_key", "access_scope", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMax["access_scope"])),
						resource.TestCheckResourceAttrSet("data.stackit_kms_wrapping_key.wrapping_key", "public_key"),
						resource.TestCheckResourceAttrSet("data.stackit_kms_wrapping_key.wrapping_key", "expires_at"),
						resource.TestCheckResourceAttrSet("data.stackit_kms_wrapping_key.wrapping_key", "created_at"),
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigWrappingKeyVarsMax,
				ResourceName:    "stackit_kms_wrapping_key.wrapping_key",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_kms_wrapping_key.wrapping_key"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_kms_wrapping_key.wrapping_key")
					}
					keyRingId, ok := r.Primary.Attributes["keyring_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute keyring_id")
					}
					wrappingKeyId, ok := r.Primary.Attributes["wrapping_key_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute wrapping_key_id")
					}

					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, keyRingId, wrappingKeyId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigWrappingKeyVarsMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceWrappingKeyMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_kms_keyring.keyring", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("stackit_kms_wrapping_key.wrapping_key", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "region", testutil.Region),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_keyring.keyring", "keyring_id",
						"stackit_kms_wrapping_key.wrapping_key", "keyring_id",
					),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "wrapping_key_id"),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "algorithm", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMaxUpdated()["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "display_name", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMaxUpdated()["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "purpose", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMaxUpdated()["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "protection", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMaxUpdated()["protection"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "description", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMaxUpdated()["description"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "access_scope", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMaxUpdated()["access_scope"])),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "public_key"),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "expires_at"),
					resource.TestCheckResourceAttrSet("stackit_kms_wrapping_key.wrapping_key", "created_at"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAccCheckKeyDestroy,
		testAccCheckWrappingKeyDestroy,
		testAccCheckKeyRingDestroy,
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

func testAccCheckKeyRingDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *kms.APIClient
	var err error
	if testutil.KMSCustomEndpoint == "" {
		client, err = kms.NewAPIClient()
	} else {
		client, err = kms.NewAPIClient(
			coreConfig.WithEndpoint(testutil.KMSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_kms_keyring" {
			continue
		}
		keyRingId := strings.Split(rs.Primary.ID, core.Separator)[2]
		err := client.DeleteKeyRingExecute(ctx, testutil.ProjectId, testutil.Region, keyRingId)
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}

				// Workaround: when the delete endpoint is called for a keyring which has keys inside it (no matter if
				// they are scheduled for deletion or not, it will throw an HTTP 400 error and the keyring can't be
				// deleted then).
				// But at least we can delete all empty keyrings created by the keyring acc tests this way.
				if oapiErr.StatusCode == http.StatusBadRequest {
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger keyring deletion %q: %w", keyRingId, err))
		}
	}

	return errors.Join(errs...)
}

func testAccCheckKeyDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *kms.APIClient
	var err error
	if testutil.KMSCustomEndpoint == "" {
		client, err = kms.NewAPIClient()
	} else {
		client, err = kms.NewAPIClient(
			coreConfig.WithEndpoint(testutil.KMSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_kms_key" {
			continue
		}
		keyRingId := strings.Split(rs.Primary.ID, core.Separator)[2]
		keyId := strings.Split(rs.Primary.ID, core.Separator)[3]
		err := client.DeleteKeyExecute(ctx, testutil.ProjectId, testutil.Region, keyRingId, keyId)
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}

				// workaround: when the delete endpoint is called a second time for a key which is already scheduled
				// for deletion, one will get an HTTP 400 error which we have to ignore here
				if oapiErr.StatusCode == http.StatusBadRequest {
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger key deletion %q: %w", keyRingId, err))
		}
	}

	return errors.Join(errs...)
}

func testAccCheckWrappingKeyDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *kms.APIClient
	var err error
	if testutil.KMSCustomEndpoint == "" {
		client, err = kms.NewAPIClient()
	} else {
		client, err = kms.NewAPIClient(
			coreConfig.WithEndpoint(testutil.KMSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_kms_wrapping_key" {
			continue
		}
		keyRingId := strings.Split(rs.Primary.ID, core.Separator)[2]
		wrappingKeyId := strings.Split(rs.Primary.ID, core.Separator)[3]
		err := client.DeleteWrappingKeyExecute(ctx, testutil.ProjectId, testutil.Region, keyRingId, wrappingKeyId)
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger wrapping key deletion %q: %w", keyRingId, err))
		}
	}

	return errors.Join(errs...)
}
