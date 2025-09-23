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
	core_config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-key-ring-min.tf
	resourceKeyRingMinConfig string
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

func TestAccKeyRingMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		//CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigKeyRingVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_kms_key_ring.key_ring", "project_id", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_kms_key_ring.key_ring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["display_name"])),
					resource.TestCheckResourceAttrSet("stackit_kms_key_ring", "key_ring_id"),
				),
			},
			// Data source
			/*{
				ConfigVariables: testConfigKeyRingVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.KMSProviderConfig(), resourceKeyRingMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_kms_key_ring.key_ring", "project_id", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_kms_key_ring.key_ring", "key_ring_id",
						"data.stackit_kms_key_ring.key_ring", "key_ring_id",
					),
					resource.TestCheckResourceAttr("data.stackit_kms_key_ring.key_ring", "display_name", testutil.ConvertConfigVariable(testConfigKeyRingVarsMin["display_name"])),
				),
			},*/
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
		client, err = kms.NewAPIClient(
			core_config.WithRegion("eu01"),
		)
	} else {
		client, err = kms.NewAPIClient(
			core_config.WithEndpoint(testutil.KMSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_kms_key_ring" {
			continue
		}
		keyRingId := strings.Split(rs.Primary.ID, core.Separator)[1]
		err := client.DeleteKeyRingExecute(ctx, testutil.ProjectId, testutil.Region, keyRingId)
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger key ring deletion %q: %w", keyRingId, err))
		}
	}

	return errors.Join(errs...)
}
