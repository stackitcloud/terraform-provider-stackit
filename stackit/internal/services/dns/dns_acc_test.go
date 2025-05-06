package dns_test

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"maps"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	core_config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stackitcloud/stackit-sdk-go/services/dns/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-min.tf
	resourceMinConfig string

	//go:embed testdata/resource-max.tf
	resourceMaxConfig string
)

var testConfigVarsMin = config.Variables{
	"project_id":     config.StringVariable(testutil.ProjectId),
	"name":           config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"dns_name":       config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha) + ".example.home"),
	"record_name":    config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"record_record1": config.StringVariable("1.2.3.4"),
	"record_type":    config.StringVariable("A"),
}

var testConfigVarsMax = config.Variables{
	"project_id":      config.StringVariable(testutil.ProjectId),
	"name":            config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"dns_name":        config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha) + ".example.home"),
	"acl":             config.StringVariable("0.0.0.0/0"),
	"active":          config.BoolVariable(true),
	"contact_email":   config.StringVariable("contact@example.com"),
	"default_ttl":     config.IntegerVariable(3600),
	"description":     config.StringVariable("a test description"),
	"expire_time":     config.IntegerVariable(1 * 24 * 60 * 60),
	"is_reverse_zone": config.BoolVariable(false),
	"negative_cache":  config.IntegerVariable(128),
	"primaries":       config.ListVariable(config.StringVariable("1.1.1.1")),
	"refresh_time":    config.IntegerVariable(3600),
	"retry_time":      config.IntegerVariable(600),
	"type":            config.StringVariable("primary"),

	"record_name":    config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"record_record1": config.StringVariable("1.2.3.4"),
	"record_active":  config.BoolVariable(true),
	"record_comment": config.StringVariable("a test comment"),
	"record_ttl":     config.IntegerVariable(3600),
	"record_type":    config.StringVariable("A"),
}

func configVarsInvalid(vars config.Variables) config.Variables {
	tempConfig := maps.Clone(vars)
	tempConfig["dns_name"] = config.StringVariable("foo")
	return tempConfig
}

func configVarsMinUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMin)
	tempConfig["record_record1"] = config.StringVariable("1.2.3.5")

	return tempConfig
}

func configVarsMaxUpdated() config.Variables {
	tempConfig := maps.Clone(testConfigVarsMax)
	tempConfig["record_record1"] = config.StringVariable("1.2.3.5")
	return tempConfig
}

func unwrap(v config.Variable) string {
	tmp, err := v.MarshalJSON()
	if err != nil {
		log.Panicf("cannot marshal variable %v: %v", v, err)
	}
	return strings.Trim(string(tmp), `"`)
}

func TestAccDnsMinResource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDnsDestroy,
		Steps: []resource.TestStep{
			// Creation fail
			{
				Config:          resourceMinConfig,
				ConfigVariables: configVarsInvalid(testConfigVarsMin),
				ExpectError:     regexp.MustCompile(`not a valid dns name. Need at least two levels`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),

					// Record set data
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "project_id",
						"stackit_dns_zone.zone", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "zone_id",
						"stackit_dns_zone.zone", "zone_id",
					),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", unwrap(testConfigVarsMin["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", unwrap(testConfigVarsMin["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", unwrap(testConfigVarsMin["record_type"])),
				),
			},
			{
				Config:          resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),

					// Record set data
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "project_id",
						"stackit_dns_zone.zone", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "zone_id",
						"stackit_dns_zone.zone", "zone_id",
					),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", unwrap(testConfigVarsMin["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", unwrap(testConfigVarsMin["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", unwrap(testConfigVarsMin["record_type"])),
				),
			},
			// Data sources
			{
				Config:          resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_zone.zone", "zone_id",
						"data.stackit_dns_zone.zone", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "zone_id",
						"data.stackit_dns_zone.zone", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "project_id",
						"data.stackit_dns_zone.zone", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "project_id",
						"stackit_dns_record_set.record_set", "project_id",
					),

					// Record set data
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", unwrap(testConfigVarsMin["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", unwrap(testConfigVarsMin["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", unwrap(testConfigVarsMin["record_type"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_dns_zone.zone",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_dns_zone.zone"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_dns_zone.recozonerd_set")
					}
					zoneId, ok := r.Primary.Attributes["zone_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute zone_id")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, zoneId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_dns_record_set.record_set",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_dns_record_set.record_set"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_dns_record_set.record_set")
					}
					zoneId, ok := r.Primary.Attributes["zone_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute zone_id")
					}
					recordSetId, ok := r.Primary.Attributes["record_set_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute record_set_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, zoneId, recordSetId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// Will be different because of the name vs fqdn problem, but the value is already tested in the datasource acc test
				ImportStateVerifyIgnore: []string{"name"},
			},
			// Update. The zone ttl should not be updated according to the DNS API.
			{
				Config:          resourceMinConfig,
				ConfigVariables: configVarsMinUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),

					// Record set data
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "project_id",
						"stackit_dns_zone.zone", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "zone_id",
						"stackit_dns_zone.zone", "zone_id",
					),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", unwrap(testConfigVarsMin["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", unwrap(configVarsMinUpdated()["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", unwrap(testConfigVarsMin["record_type"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccDnsMaxResource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDnsDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config:          resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),

					// Record set data
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "project_id",
						"stackit_dns_zone.zone", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "zone_id",
						"stackit_dns_zone.zone", "zone_id",
					),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "acl", unwrap(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "active", unwrap(testConfigVarsMax["active"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "contact_email", unwrap(testConfigVarsMax["contact_email"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "default_ttl", unwrap(testConfigVarsMax["default_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "description", unwrap(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "expire_time", unwrap(testConfigVarsMax["expire_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "is_reverse_zone", unwrap(testConfigVarsMax["is_reverse_zone"])),
					// resource.TestCheckResourceAttr("stackit_dns_zone.zone", "negative_cache", unwrap(testConfigVarsMax["negative_cache"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "primaries.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primaries.0"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "refresh_time", unwrap(testConfigVarsMax["refresh_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "retry_time", unwrap(testConfigVarsMax["retry_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "type", unwrap(testConfigVarsMax["type"])),

					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", unwrap(testConfigVarsMax["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", unwrap(testConfigVarsMax["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "active", unwrap(testConfigVarsMax["record_active"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "comment", unwrap(testConfigVarsMax["record_comment"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "ttl", unwrap(testConfigVarsMax["record_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", unwrap(testConfigVarsMax["record_type"])),
				),
			},
			// Data sources
			{
				Config:          resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_zone.zone", "zone_id",
						"data.stackit_dns_zone.zone", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "zone_id",
						"data.stackit_dns_zone.zone", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "project_id",
						"data.stackit_dns_zone.zone", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "project_id",
						"stackit_dns_record_set.record_set", "project_id",
					),

					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "acl", unwrap(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "active", unwrap(testConfigVarsMax["active"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "contact_email", unwrap(testConfigVarsMax["contact_email"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "default_ttl", unwrap(testConfigVarsMax["default_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "description", unwrap(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "expire_time", unwrap(testConfigVarsMax["expire_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "is_reverse_zone", unwrap(testConfigVarsMax["is_reverse_zone"])),
					// resource.TestCheckResourceAttr("stackit_dns_zone.zone", "negative_cache", unwrap(testConfigVarsMax["negative_cache"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "primaries.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primaries.0"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "refresh_time", unwrap(testConfigVarsMax["refresh_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "retry_time", unwrap(testConfigVarsMax["retry_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "type", unwrap(testConfigVarsMax["type"])),

					// Record set data
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", unwrap(testConfigVarsMax["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", unwrap(testConfigVarsMax["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "active", unwrap(testConfigVarsMax["record_active"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "comment", unwrap(testConfigVarsMax["record_comment"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "ttl", unwrap(testConfigVarsMax["record_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", unwrap(testConfigVarsMax["record_type"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_dns_zone.zone",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_dns_zone.zone"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_dns_zone.recozonerd_set")
					}
					zoneId, ok := r.Primary.Attributes["zone_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute zone_id")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, zoneId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_dns_record_set.record_set",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_dns_record_set.record_set"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_dns_record_set.record_set")
					}
					zoneId, ok := r.Primary.Attributes["zone_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute zone_id")
					}
					recordSetId, ok := r.Primary.Attributes["record_set_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute record_set_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, zoneId, recordSetId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// Will be different because of the name vs fqdn problem, but the value is already tested in the datasource acc test
				ImportStateVerifyIgnore: []string{"name"},
			},
			// Update. The zone ttl should not be updated according to the DNS API.
			{
				Config:          resourceMaxConfig,
				ConfigVariables: configVarsMaxUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "acl", unwrap(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "active", unwrap(testConfigVarsMax["active"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "contact_email", unwrap(testConfigVarsMax["contact_email"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "default_ttl", unwrap(testConfigVarsMax["default_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "description", unwrap(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "expire_time", unwrap(testConfigVarsMax["expire_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "is_reverse_zone", unwrap(testConfigVarsMax["is_reverse_zone"])),
					// resource.TestCheckResourceAttr("stackit_dns_zone.zone", "negative_cache", unwrap(testConfigVarsMax["negative_cache"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "primaries.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primaries.0"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "refresh_time", unwrap(testConfigVarsMax["refresh_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "retry_time", unwrap(testConfigVarsMax["retry_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "type", unwrap(testConfigVarsMax["type"])),

					// Record set data
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "project_id",
						"stackit_dns_zone.zone", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set", "zone_id",
						"stackit_dns_zone.zone", "zone_id",
					),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", unwrap(testConfigVarsMax["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", unwrap(configVarsMaxUpdated()["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "active", unwrap(testConfigVarsMax["record_active"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "comment", unwrap(testConfigVarsMax["record_comment"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "ttl", unwrap(testConfigVarsMax["record_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", unwrap(testConfigVarsMax["record_type"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckDnsDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *dns.APIClient
	var err error
	if testutil.DnsCustomEndpoint == "" {
		client, err = dns.NewAPIClient()
	} else {
		client, err = dns.NewAPIClient(
			core_config.WithEndpoint(testutil.DnsCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	zonesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_dns_zone" {
			continue
		}
		// zone terraform ID: "[projectId],[zoneId]"
		zoneId := strings.Split(rs.Primary.ID, core.Separator)[1]
		zonesToDestroy = append(zonesToDestroy, zoneId)
	}

	zonesResp, err := client.ListZones(ctx, testutil.ProjectId).ActiveEq(true).Execute()
	if err != nil {
		return fmt.Errorf("getting zonesResp: %w", err)
	}

	zones := *zonesResp.Zones
	for i := range zones {
		id := *zones[i].Id
		if utils.Contains(zonesToDestroy, id) {
			_, err := client.DeleteZoneExecute(ctx, testutil.ProjectId, id)
			if err != nil {
				return fmt.Errorf("destroying zone %s during CheckDestroy: %w", *zones[i].Id, err)
			}
			_, err = wait.DeleteZoneWaitHandler(ctx, client, testutil.ProjectId, id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying zone %s during CheckDestroy: waiting for deletion %w", *zones[i].Id, err)
			}
		}
	}
	return nil
}
