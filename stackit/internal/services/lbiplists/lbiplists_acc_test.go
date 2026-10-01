package lbiplists_test

// Requirements to run this acceptance test:
// - Terraform CLI >= 1.11 (the resource uses the write-only attribute `file_content`)

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	lbiplists "github.com/stackitcloud/stackit-sdk-go/services/lbiplists/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/iplist.tf
var ipListConfig string

var ipListRegion = testutil.Region

var ipListVars = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"name":                 config.StringVariable(fmt.Sprintf("acc-test-%s", strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)))),
	"file_content":         config.StringVariable("10.0.0.0/16\n192.168.0.0/24"),
	"file_content_version": config.IntegerVariable(1),
	"label_key":            config.StringVariable("env"),
	"label_value":          config.StringVariable("test"),
}

// ipListVarsLabelsUpdated changes only the label value - the file_content_version stays the same.
// This exercises the labels-only update path
var ipListVarsLabelsUpdated = func() config.Variables {
	updated := make(config.Variables, len(ipListVars))
	maps.Copy(updated, ipListVars)
	updated["label_value"] = config.StringVariable("production")
	return updated
}()

// ipListVarsContentRotated replaces the file_content and increments file_content_version 1 → 2.
var ipListVarsContentRotated = func() config.Variables {
	rotated := make(config.Variables, len(ipListVarsLabelsUpdated))
	maps.Copy(rotated, ipListVarsLabelsUpdated)
	rotated["file_content"] = config.StringVariable("10.10.0.0/16\n172.16.0.0/12")
	rotated["file_content_version"] = config.IntegerVariable(2)
	return rotated
}()

func TestAccIpListResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIpListsDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: ipListVars,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), ipListConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "region", ipListRegion),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "name", testutil.ConvertConfigVariable(ipListVars["name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "file_content_version", testutil.ConvertConfigVariable(ipListVars["file_content_version"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "labels."+testutil.ConvertConfigVariable(ipListVars["label_key"]), testutil.ConvertConfigVariable(ipListVars["label_value"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "number_of_ips", "2"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_ip_list.iplist", "content_hash"),
				),
			},
			// Data source
			{
				ConfigVariables: ipListVars,
				Config: fmt.Sprintf(`
						%s
						%s

						data "stackit_loadbalancer_ip_list" "iplist" {
							project_id = stackit_loadbalancer_ip_list.iplist.project_id
							name       = stackit_loadbalancer_ip_list.iplist.name
						}
						`,
					testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), ipListConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_loadbalancer_ip_list.iplist", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer_ip_list.iplist", "region", ipListRegion),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer_ip_list.iplist", "name", testutil.ConvertConfigVariable(ipListVars["name"])),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer_ip_list.iplist", "number_of_ips", "2"),
					resource.TestCheckResourceAttrSet("data.stackit_loadbalancer_ip_list.iplist", "content_hash"),
					resource.TestCheckResourceAttr("data.stackit_loadbalancer_ip_list.iplist", "labels."+testutil.ConvertConfigVariable(ipListVars["label_key"]), testutil.ConvertConfigVariable(ipListVars["label_value"])),

					resource.TestCheckResourceAttrPair("data.stackit_loadbalancer_ip_list.iplist", "id", "stackit_loadbalancer_ip_list.iplist", "id"),
					resource.TestCheckResourceAttrPair("data.stackit_loadbalancer_ip_list.iplist", "region", "stackit_loadbalancer_ip_list.iplist", "region"),
					resource.TestCheckResourceAttrPair("data.stackit_loadbalancer_ip_list.iplist", "content_hash", "stackit_loadbalancer_ip_list.iplist", "content_hash"),
					resource.TestCheckResourceAttrPair("data.stackit_loadbalancer_ip_list.iplist", "number_of_ips", "stackit_loadbalancer_ip_list.iplist", "number_of_ips"),
				),
			},
			// Labels-only update - file_content_version unchanged
			{
				ConfigVariables: ipListVarsLabelsUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), ipListConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "name", testutil.ConvertConfigVariable(ipListVarsLabelsUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "file_content_version", testutil.ConvertConfigVariable(ipListVarsLabelsUpdated["file_content_version"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "labels."+testutil.ConvertConfigVariable(ipListVarsLabelsUpdated["label_key"]), testutil.ConvertConfigVariable(ipListVarsLabelsUpdated["label_value"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "number_of_ips", "2"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_ip_list.iplist", "content_hash"),
				),
			},
			// Content rotation - file_content replaced, version bumped 1 → 2
			{
				ConfigVariables: ipListVarsContentRotated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig(), ipListConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "file_content_version", testutil.ConvertConfigVariable(ipListVarsContentRotated["file_content_version"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "number_of_ips", "2"),
					resource.TestCheckResourceAttrSet("stackit_loadbalancer_ip_list.iplist", "content_hash"),
					// all other fields must be unchanged
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "name", testutil.ConvertConfigVariable(ipListVarsContentRotated["name"])),
					resource.TestCheckResourceAttr("stackit_loadbalancer_ip_list.iplist", "labels."+testutil.ConvertConfigVariable(ipListVarsContentRotated["label_key"]), testutil.ConvertConfigVariable(ipListVarsContentRotated["label_value"])),
				),
			},
			// Import
			{
				ConfigVariables: ipListVars,
				ResourceName:    "stackit_loadbalancer_ip_list.iplist",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_loadbalancer_ip_list.iplist"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_loadbalancer_ip_list.iplist")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s",
						testutil.ProjectId,
						ipListRegion,
						name,
					), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// file_content is write-only and never stored in state; file_content_version can not be
				// recovered on import since the API never returns it
				ImportStateVerifyIgnore: []string{"file_content", "file_content_version"},
			},
		},
	})
}

func testAccCheckIpListsDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := lbiplists.NewAPIClient(
		testutil.NewConfigBuilder().BuildClientOptions(testutil.LoadBalancerCustomEndpoint, false)...,
	)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_loadbalancer_ip_list" {
			continue
		}

		// terraform ID: "[project_id],[region],[name]"
		parts := strings.Split(rs.Primary.ID, core.Separator)
		if len(parts) != 3 {
			return fmt.Errorf("unexpected terraform ID %q for stackit_loadbalancer_ip_list", rs.Primary.ID)
		}
		projectId, region, name := parts[0], parts[1], parts[2]

		_, err := client.DefaultAPI.GetIPList(ctx, projectId, region, name).Execute()
		if err == nil {
			return fmt.Errorf("IP list %q still exists", rs.Primary.ID)
		}
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); !ok || oapiErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("checking IP list %q: %w", rs.Primary.ID, err)
		}
	}

	return nil
}
