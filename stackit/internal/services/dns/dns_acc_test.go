package dns_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Zone resource data
var zoneResource = map[string]string{
	"project_id":          testutil.ProjectId,
	"name":                testutil.ResourceNameWithDateTime("zone"),
	"dns_name":            fmt.Sprintf("www.%s.com", acctest.RandStringFromCharSet(20, acctest.CharSetAlpha)),
	"dns_name_min":        fmt.Sprintf("www.%s.com", acctest.RandStringFromCharSet(20, acctest.CharSetAlpha)),
	"description":         "my description",
	"description_updated": "my description updated",
	"acl":                 "192.168.0.0/24",
	"active":              "true",
	"contact_email":       "aa@bb.cc",
	"ttl":                 "120",
	"ttl_updated":         "4440",
	"expire_time":         "123456",
	"is_reverse_zone":     "false",
	"negative_cache":      "60",
	"primaries":           "1.2.3.4",
	"refresh_time":        "500",
	"retry_time":          "700",
	"type":                "primary",
}

// Record set resource data
var recordSetResource = map[string]string{
	"name":            fmt.Sprintf("tf-acc-%s.%s.", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha), zoneResource["dns_name"]),
	"name_min":        fmt.Sprintf("tf-acc-%s.%s.", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha), zoneResource["dns_name_min"]),
	"records":         `"1.2.3.4"`,
	"records_updated": `"5.6.7.8", "9.10.11.12"`,
	"ttl":             "3700",
	"type":            "A",
	"active":          "true",
	"comment":         "a comment",
}

func inputConfig(zoneName, description, ttl, records string) string {
	return fmt.Sprintf(`
		%s

		resource "stackit_dns_zone" "zone" {
			project_id = "%s"
			name    = "%s"
			dns_name = "%s"
			description = "%s"
			acl = "%s"
			active = %s
			contact_email = "%s"
			default_ttl = %s
			expire_time = %s
			is_reverse_zone = %s
			negative_cache = %s
			primaries = ["%s"]
			refresh_time = %s
			retry_time = %s
			type = "%s"
		}

		resource "stackit_dns_record_set" "record_set" {
			project_id = stackit_dns_zone.zone.project_id
			zone_id    = stackit_dns_zone.zone.zone_id
			name       = "%s"
			records    = [%s]
			type       = "%s"
			ttl 	   =  %s
			comment    = "%s"
			active     =  %s

		}
		`,
		testutil.DnsProviderConfig(),
		zoneResource["project_id"],
		zoneName,
		zoneResource["dns_name"],
		description,
		zoneResource["acl"],
		zoneResource["active"],
		zoneResource["contact_email"],
		ttl,
		zoneResource["expire_time"],
		zoneResource["is_reverse_zone"],
		zoneResource["negative_cache"],
		zoneResource["primaries"],
		zoneResource["refresh_time"],
		zoneResource["retry_time"],
		zoneResource["type"],
		recordSetResource["name"],
		records,
		recordSetResource["type"],
		recordSetResource["ttl"],
		recordSetResource["comment"],
		recordSetResource["active"],
	)
}

func TestAccDnsResource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDnsDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: inputConfig(zoneResource["name"], zoneResource["description"], zoneResource["ttl"], recordSetResource["records"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", zoneResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "name", zoneResource["name"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "dns_name", zoneResource["dns_name"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "description", zoneResource["description"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "acl", zoneResource["acl"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "active", zoneResource["active"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "contact_email", zoneResource["contact_email"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "default_ttl", zoneResource["ttl"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "expire_time", zoneResource["expire_time"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "is_reverse_zone", zoneResource["is_reverse_zone"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "negative_cache", zoneResource["negative_cache"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "primaries.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "primaries.0", zoneResource["primaries"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "refresh_time", zoneResource["refresh_time"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "retry_time", zoneResource["retry_time"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "type", zoneResource["type"]),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primary_name_server"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "serial_number"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "visibility"),
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
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", recordSetResource["name"]),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.0", strings.ReplaceAll(recordSetResource["records"], "\"", "")),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", recordSetResource["type"]),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "ttl", recordSetResource["ttl"]),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "comment", recordSetResource["comment"]),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "active", recordSetResource["active"]),
				),
			},
			// Data sources
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_dns_zone" "zone" {
						project_id = stackit_dns_zone.zone.project_id
						zone_id    = stackit_dns_zone.zone.zone_id
					}

					data "stackit_dns_record_set" "record_set" {
						project_id = stackit_dns_zone.zone.project_id
						zone_id    = stackit_dns_zone.zone.zone_id
						record_set_id = stackit_dns_record_set.record_set.record_set_id
					}`,
					inputConfig(zoneResource["name"], zoneResource["description"], zoneResource["ttl"], recordSetResource["records"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "project_id", zoneResource["project_id"]),
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
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "name", zoneResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "default_ttl", zoneResource["ttl"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "dns_name", zoneResource["dns_name"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "description", zoneResource["description"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "acl", zoneResource["acl"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "active", zoneResource["active"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "contact_email", zoneResource["contact_email"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "default_ttl", zoneResource["ttl"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "expire_time", zoneResource["expire_time"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "is_reverse_zone", zoneResource["is_reverse_zone"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "negative_cache", zoneResource["negative_cache"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "primaries.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "primaries.0", zoneResource["primaries"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "refresh_time", zoneResource["refresh_time"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "retry_time", zoneResource["retry_time"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "type", zoneResource["type"]),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone", "primary_name_server"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone", "serial_number"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone", "visibility"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone", "state"),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone", "record_count", "4"),

					// Record set data
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set", "record_set_id"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "name", recordSetResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "records.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "type", recordSetResource["type"]),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "ttl", recordSetResource["ttl"]),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "comment", recordSetResource["comment"]),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set", "active", recordSetResource["active"]),
				),
			},
			// Import
			{
				ResourceName: "stackit_dns_zone.zone",
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
				ResourceName: "stackit_dns_record_set.record_set",
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
			},
			// Update. The zone ttl should not be updated according to the DNS API.
			{
				Config: inputConfig(zoneResource["name"], zoneResource["description_updated"], zoneResource["ttl"], recordSetResource["records_updated"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "project_id", zoneResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "zone_id"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "name", zoneResource["name"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "dns_name", zoneResource["dns_name"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "description", zoneResource["description_updated"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "acl", zoneResource["acl"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "active", zoneResource["active"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "contact_email", zoneResource["contact_email"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "default_ttl", zoneResource["ttl"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "expire_time", zoneResource["expire_time"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "is_reverse_zone", zoneResource["is_reverse_zone"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "negative_cache", zoneResource["negative_cache"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "primaries.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "primaries.0", zoneResource["primaries"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "refresh_time", zoneResource["refresh_time"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "retry_time", zoneResource["retry_time"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone", "type", zoneResource["type"]),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "primary_name_server"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "serial_number"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone", "visibility"),
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
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "name", recordSetResource["name"]),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "records.#", "2"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "type", recordSetResource["type"]),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "ttl", recordSetResource["ttl"]),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "comment", recordSetResource["comment"]),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set", "active", recordSetResource["active"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func inputConfigMinimal() string {
	return fmt.Sprintf(`
		%s

		resource "stackit_dns_zone" "zone_min" {
			project_id = "%s"
			name    = "%s"
			dns_name = "%s"
			contact_email = "%s"
			type = "%s"
			acl = "%s"
		}

		resource "stackit_dns_record_set" "record_set_min" {
			project_id = stackit_dns_zone.zone_min.project_id
			zone_id    = stackit_dns_zone.zone_min.zone_id
			name       = "%s"
			records    = [%s]
			type       = "%s"
		}
		`,
		testutil.DnsProviderConfig(),
		zoneResource["project_id"],
		zoneResource["name"],
		zoneResource["dns_name_min"],
		zoneResource["contact_email"],
		zoneResource["type"],
		zoneResource["acl"],
		recordSetResource["name_min"],
		recordSetResource["records"],
		recordSetResource["type"],
	)
}

func TestAccDnsMinimalResource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDnsDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: inputConfigMinimal(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone
					resource.TestCheckResourceAttr("stackit_dns_zone.zone_min", "project_id", zoneResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "zone_id"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone_min", "name", zoneResource["name"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone_min", "dns_name", zoneResource["dns_name_min"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone_min", "contact_email", zoneResource["contact_email"]),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone_min", "type", zoneResource["type"]),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "acl"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone_min", "active", "true"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "default_ttl"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "expire_time"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone_min", "is_reverse_zone", "false"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "negative_cache"),
					resource.TestCheckResourceAttr("stackit_dns_zone.zone_min", "primaries.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "refresh_time"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "retry_time"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "primary_name_server"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "serial_number"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "visibility"),
					resource.TestCheckResourceAttrSet("stackit_dns_zone.zone_min", "state"),

					// Record set
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set_min", "project_id",
						"stackit_dns_zone.zone_min", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_record_set.record_set_min", "zone_id",
						"stackit_dns_zone.zone_min", "zone_id",
					),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set_min", "record_set_id"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set_min", "name", recordSetResource["name_min"]),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set_min", "records.#", "1"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set_min", "records.0", strings.ReplaceAll(recordSetResource["records"], "\"", "")),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set_min", "type", recordSetResource["type"]),
					resource.TestCheckResourceAttrSet("stackit_dns_record_set.record_set_min", "ttl"),
					resource.TestCheckNoResourceAttr("stackit_dns_record_set.record_set_min", "comment"),
					resource.TestCheckResourceAttr("stackit_dns_record_set.record_set_min", "active", "true"),
				),
			},
			// Data sources
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_dns_zone" "zone_min" {
						project_id = stackit_dns_zone.zone_min.project_id
						zone_id    = stackit_dns_zone.zone_min.zone_id
					}

					data "stackit_dns_record_set" "record_set_min" {
						project_id = stackit_dns_zone.zone_min.project_id
						zone_id    = stackit_dns_zone.zone_min.zone_id
						record_set_id = stackit_dns_record_set.record_set_min.record_set_id
					}`,
					inputConfigMinimal(),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Zone data
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "project_id", zoneResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_dns_zone.zone_min", "zone_id",
						"data.stackit_dns_zone.zone_min", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set_min", "zone_id",
						"data.stackit_dns_zone.zone_min", "zone_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set_min", "project_id",
						"data.stackit_dns_zone.zone_min", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_dns_record_set.record_set_min", "project_id",
						"stackit_dns_record_set.record_set_min", "project_id",
					),

					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "project_id", zoneResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_min", "zone_id"),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "name", zoneResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "dns_name", zoneResource["dns_name_min"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "contact_email", zoneResource["contact_email"]),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "type", zoneResource["type"]),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_min", "acl"),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "active", "true"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_min", "default_ttl"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_min", "expire_time"),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "is_reverse_zone", "false"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_min", "negative_cache"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_min", "primary_name_server"),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "primaries.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_min", "refresh_time"),
					resource.TestCheckResourceAttrSet("data.stackit_dns_zone.zone_min", "retry_time"),
					resource.TestCheckResourceAttr("data.stackit_dns_zone.zone_min", "record_count", "4"),

					// Record set data
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set_min", "record_set_id"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set_min", "name", recordSetResource["name_min"]),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set_min", "records.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set_min", "records.0", strings.ReplaceAll(recordSetResource["records"], "\"", "")),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set_min", "type", recordSetResource["type"]),
					resource.TestCheckResourceAttrSet("data.stackit_dns_record_set.record_set_min", "ttl"),
					resource.TestCheckNoResourceAttr("data.stackit_dns_record_set.record_set_min", "comment"),
					resource.TestCheckResourceAttr("data.stackit_dns_record_set.record_set_min", "active", "true"),
				),
			},
			// Import
			{
				ResourceName: "stackit_dns_zone.zone_min",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_dns_zone.zone_min"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_dns_zone.zone_min")
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
				ResourceName: "stackit_dns_record_set.record_set_min",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_dns_record_set.record_set_min"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_dns_record_set.record_set_min")
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
			config.WithEndpoint(testutil.DnsCustomEndpoint),
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

	zonesResp, err := client.GetZones(ctx, testutil.ProjectId).ActiveEq(true).Execute()
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
			_, err = dns.DeleteZoneWaitHandler(ctx, client, testutil.ProjectId, id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying zone %s during CheckDestroy: waiting for deletion %w", *zones[i].Id, err)
			}
		}
	}
	return nil
}
