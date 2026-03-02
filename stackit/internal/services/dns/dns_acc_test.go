package dns_test

import (
	"context"
	_ "embed"
	"fmt"
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
	// "negative_cache":  config.IntegerVariable(128),
	"primaries":    config.ListVariable(config.StringVariable("1.1.1.1")),
	"refresh_time": config.IntegerVariable(3600),
	"retry_time":   config.IntegerVariable(600),
	"type":         config.StringVariable("primary"),

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
			},
			// Creation fail: trailing dot is rejected on purpose
			{
				Config: resourceMinConfig,
				ConfigVariables: func() config.Variables {
					vars := maps.Clone(testConfigVarsMin)

					// Ensure it ends with a dot (even if the random value already had one, be explicit)
					base := testutil.ConvertConfigVariable(vars["dns_name"])
					vars["dns_name"] = config.StringVariable(base + ".")

					return vars
				}(),
				ExpectError: regexp.MustCompile(`dns_name must not end with a trailing dot`),
			},
			// creation
			{
				Config:          resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),

					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primary_name_server"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "serial_number"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "visibility"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
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
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "name"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", testutil.ConvertConfigVariable(testConfigVarsMin["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", testutil.ConvertConfigVariable(testConfigVarsMin["record_type"])),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "fqdn"),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "state"),
				),
			},
			// Data sources
			{
				Config:          resourceMinConfig,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data by zone_id
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "project_id", testutil.ProjectId),
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
						"stackit_dns_record_set.record_set", "project_id",
						"data.stackit_dns_record_set.record_set", "project_id",
					),

					// Zone data by dns_name
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_zone.zone", "zone_id",
						"data.stackit_dns_zone.zone_name", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "zone_id",
						"data.stackit_dns_zone.zone_name", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "project_id",
						"data.stackit_dns_zone.zone_name", "project_id",
					),

					// Record set data
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set", "name"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "records.0", testutil.ConvertConfigVariable(testConfigVarsMin["record_record1"])),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "type", testutil.ConvertConfigVariable(testConfigVarsMin["record_type"])),
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
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primary_name_server"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "serial_number"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "visibility"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
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
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", testutil.ConvertConfigVariable(testConfigVarsMin["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", testutil.ConvertConfigVariable(configVarsMinUpdated()["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", testutil.ConvertConfigVariable(testConfigVarsMin["record_type"])),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "fqdn"),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "state")),
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
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "acl", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "active", testutil.ConvertConfigVariable(testConfigVarsMax["active"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "contact_email", testutil.ConvertConfigVariable(testConfigVarsMax["contact_email"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "default_ttl", testutil.ConvertConfigVariable(testConfigVarsMax["default_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "description", testutil.ConvertConfigVariable(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "expire_time", testutil.ConvertConfigVariable(testConfigVarsMax["expire_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "is_reverse_zone", testutil.ConvertConfigVariable(testConfigVarsMax["is_reverse_zone"])),
					//  resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "negative_cache"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "primaries.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primaries.0"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "refresh_time", testutil.ConvertConfigVariable(testConfigVarsMax["refresh_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "retry_time", testutil.ConvertConfigVariable(testConfigVarsMax["retry_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "type", testutil.ConvertConfigVariable(testConfigVarsMax["type"])),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primary_name_server"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "serial_number"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "visibility"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),

					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", testutil.ConvertConfigVariable(testConfigVarsMax["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", testutil.ConvertConfigVariable(testConfigVarsMax["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "active", testutil.ConvertConfigVariable(testConfigVarsMax["record_active"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "comment", testutil.ConvertConfigVariable(testConfigVarsMax["record_comment"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "ttl", testutil.ConvertConfigVariable(testConfigVarsMax["record_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", testutil.ConvertConfigVariable(testConfigVarsMax["record_type"])),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "fqdn"),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "state"),
				),
			},
			// Data sources
			{
				Config:          resourceMaxConfig,
				ConfigVariables: testConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data by zone_id
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "project_id", testutil.ProjectId),
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

					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "acl", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "active", testutil.ConvertConfigVariable(testConfigVarsMax["active"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "contact_email", testutil.ConvertConfigVariable(testConfigVarsMax["contact_email"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "default_ttl", testutil.ConvertConfigVariable(testConfigVarsMax["default_ttl"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "description", testutil.ConvertConfigVariable(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "expire_time", testutil.ConvertConfigVariable(testConfigVarsMax["expire_time"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "is_reverse_zone", testutil.ConvertConfigVariable(testConfigVarsMax["is_reverse_zone"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "primaries.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone", "primaries.0"),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "refresh_time", testutil.ConvertConfigVariable(testConfigVarsMax["refresh_time"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "retry_time", testutil.ConvertConfigVariable(testConfigVarsMax["retry_time"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "type", testutil.ConvertConfigVariable(testConfigVarsMax["type"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "dns_name", testutil.ConvertConfigVariable(testConfigVarsMax["dns_name"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					// resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone", "negative_cache"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone", "serial_number"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone", "state"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone", "visibility"),

					// Zone data by dns_name
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_zone.zone", "zone_id",
						"data.stackit_dns_zone.zone_name", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "zone_id",
						"data.stackit_dns_zone.zone_name", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set", "project_id",
						"data.stackit_dns_zone.zone_name", "project_id",
					),

					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "acl", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "active", testutil.ConvertConfigVariable(testConfigVarsMax["active"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "contact_email", testutil.ConvertConfigVariable(testConfigVarsMax["contact_email"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "default_ttl", testutil.ConvertConfigVariable(testConfigVarsMax["default_ttl"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "description", testutil.ConvertConfigVariable(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "expire_time", testutil.ConvertConfigVariable(testConfigVarsMax["expire_time"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "is_reverse_zone", testutil.ConvertConfigVariable(testConfigVarsMax["is_reverse_zone"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "primaries.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_name", "primaries.0"),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "refresh_time", testutil.ConvertConfigVariable(testConfigVarsMax["refresh_time"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "retry_time", testutil.ConvertConfigVariable(testConfigVarsMax["retry_time"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "type", testutil.ConvertConfigVariable(testConfigVarsMax["type"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "dns_name", testutil.ConvertConfigVariable(testConfigVarsMax["dns_name"])),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_name", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					// resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_name", "negative_cache"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_name", "serial_number"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_name", "state"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_name", "visibility"),

					// Record set data
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set", "name"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "active", testutil.ConvertConfigVariable(testConfigVarsMax["active"])),
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set", "fqdn"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set", "state"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "records.0", testutil.ConvertConfigVariable(testConfigVarsMax["record_record1"])),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "active", testutil.ConvertConfigVariable(testConfigVarsMax["record_active"])),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "comment", testutil.ConvertConfigVariable(testConfigVarsMax["record_comment"])),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "ttl", testutil.ConvertConfigVariable(testConfigVarsMax["record_ttl"])),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "type", testutil.ConvertConfigVariable(testConfigVarsMax["record_type"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
				ResourceName:    "stackit_dns_zone.zone",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_dns_zone.zone"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_dns_zone.record_set")
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
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "acl", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "active", testutil.ConvertConfigVariable(testConfigVarsMax["active"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "contact_email", testutil.ConvertConfigVariable(testConfigVarsMax["contact_email"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "default_ttl", testutil.ConvertConfigVariable(testConfigVarsMax["default_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "description", testutil.ConvertConfigVariable(testConfigVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "expire_time", testutil.ConvertConfigVariable(testConfigVarsMax["expire_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "is_reverse_zone", testutil.ConvertConfigVariable(testConfigVarsMax["is_reverse_zone"])),
					// resource.TestCheckResourceAttr("stackit_dns_zone.zone", "negative_cache", testutil.ConvertConfigVariable(testConfigVarsMax["negative_cache"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "primaries.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primaries.0"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "refresh_time", testutil.ConvertConfigVariable(testConfigVarsMax["refresh_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "retry_time", testutil.ConvertConfigVariable(testConfigVarsMax["retry_time"])),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "type", testutil.ConvertConfigVariable(testConfigVarsMax["type"])),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primary_name_server"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "serial_number"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "state"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "visibility"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
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
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", testutil.ConvertConfigVariable(testConfigVarsMax["record_name"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", testutil.ConvertConfigVariable(configVarsMaxUpdated()["record_record1"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "active", testutil.ConvertConfigVariable(testConfigVarsMax["record_active"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "comment", testutil.ConvertConfigVariable(testConfigVarsMax["record_comment"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "ttl", testutil.ConvertConfigVariable(testConfigVarsMax["record_ttl"])),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", testutil.ConvertConfigVariable(testConfigVarsMax["record_type"])),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "fqdn"),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set", "state")),
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
