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

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	coreConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-key-ring-min.tf
	resourceKeyRingMinConfig string

	//go:embed testdata/resource-key-ring-max.tf
	resourceKeyRingMaxConfig string

	//go:embed testdata/resource-key-min.tf
	resourceKeyMinConfig string

	//go:embed testdata/resource-wrapping-key-min.tf
	resourceWrappingKeyMinConfig string
)

var testConfigKeyRingVarsMin = config.Variables{
	"display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"project_id":   config.StringVariable(testutil.ProjectId),
}

var testConfigKeyRingVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigKeyRingVarsMin)
	updatedConfig["display_name"] = config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	return updatedConfig
}

var testConfigKeyRingVarsMax = config.Variables{
	"description":  config.StringVariable("description"),
	"display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"project_id":   config.StringVariable(testutil.ProjectId),
	"region":       config.StringVariable(testutil.Region),
}

var testConfigKeyRingVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigKeyRingVarsMax {
		updatedConfig[k] = v
	}
	updatedConfig["description"] = config.StringVariable("updated description")
	updatedConfig["display_name"] = config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	return updatedConfig
}

var testConfigKeyVarsMin = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"algorithm":    config.StringVariable("aes_256_gcm"),
	"display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"protection":   config.StringVariable("software"),
	"purpose":      config.StringVariable("symmetric_encrypt_decrypt"),
}

var testConfigWrappingKeyVarsMin = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"algorithm":    config.StringVariable("rsa_2048_oaep_sha256"),
	"display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"protection":   config.StringVariable("software"),
	"purpose":      config.StringVariable("wrap_symmetric_key"),
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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_key_ring.key_ring", "project_id", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_kms_key_ring.key_ring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_kms_key_ring.key_ring", "key_ring_id"),
					resource.TestCheckResourceAttrSet("stackit_kms_key_ring.key_ring", "region"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigKeyRingVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_kms_key_ring.key_ring", "project_id", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_key_ring.key_ring", "key_ring_id",
						"data.stackit_kms_key_ring.key_ring", "key_ring_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_key_ring.key_ring", "region",
						"data.stackit_kms_key_ring.key_ring", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_key_ring.key_ring", "project_id",
						"data.stackit_kms_key_ring.key_ring", "project_id",
					),
					resource.TestCheckResourceAttr("data.stackit_kms_key_ring.key_ring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["display_name"])),
				),
			},
		},
	})
}

func TestAccKeyRingMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			//Creation
			{
				ConfigVariables: testConfigKeyRingVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_key_ring.key_ring", "project_id", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_kms_key_ring.key_ring", "description", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["description"])),
					resource.TestCheckResourceAttrSet("stackit_kms_key_ring.key_ring", "key_ring_id"),
					resource.TestCheckResourceAttr("stackit_kms_key_ring.key_ring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_kms_key_ring.key_ring", "region"),
				),
			},
			// Data Source
			{
				ConfigVariables: testConfigKeyRingVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.stackit_kms_key_ring.key_ring", "project_id", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["project_id"])),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_key_ring.key_ring", "key_ring_id",
							"data.stackit_kms_key_ring.key_ring", "key_ring_id",
						),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_key_ring.key_ring", "region",
							"data.stackit_kms_key_ring.key_ring", "region",
						),
						resource.TestCheckResourceAttrPair(
							"stackit_kms_key_ring.key_ring", "project_id",
							"data.stackit_kms_key_ring.key_ring", "project_id",
						),
						resource.TestCheckResourceAttr("data.stackit_kms_key_ring.key_ring", "description", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["description"])),
						resource.TestCheckResourceAttr("data.stackit_kms_key_ring.key_ring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMax["display_name"])),
					),
				),
			},
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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_key.key", "algorithm", testutil.ConvertConfigVariable(testConfigKeyVarsMin["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "display_name", testutil.ConvertConfigVariable(testConfigKeyVarsMin["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "purpose", testutil.ConvertConfigVariable(testConfigKeyVarsMin["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_key.key", "protection", testutil.ConvertConfigVariable(testConfigKeyVarsMin["protection"])),
				),
			},
		},
	})
}

func TestAccWrappingKeyMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			{
				ConfigVariables: testConfigWrappingKeyVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceWrappingKeyMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "algorithm", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["algorithm"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "display_name", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["display_name"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "purpose", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["purpose"])),
					resource.TestCheckResourceAttr("stackit_kms_wrapping_key.wrapping_key", "protection", testutil.ConvertConfigVariable(testConfigWrappingKeyVarsMin["protection"])),
				),
			},
		},
	})
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAccCheckKeyDestroy,
		testAccCheckWrappingKeyDestroy,
		// no automatic destroy of key rings possible since they can't be deleted or scheduled for deletion as long as they have keys inside them
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
