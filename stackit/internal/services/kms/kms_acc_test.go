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

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
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
