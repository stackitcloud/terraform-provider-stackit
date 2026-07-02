package vpn

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Test can't be moved to resource test file because of dependency cycle
func TestConnectionResourceValidationExclusivePresharedKeyFields(t *testing.T) {
	config := func(tunnel1PresharedKeyConfig string) string {
		return fmt.Sprintf(`
		provider "stackit" {
			default_region = "eu01"
			service_account_token = "mock-server-needs-no-auth"
		}

		resource "stackit_vpn_connection" "connection" {
			project_id   = "4e684f79-a12c-449d-aa89-bcd9d8aafaf2"
			gateway_id   = "3dee3fb9-59f0-4f97-8eeb-a4da37d05a00"
			display_name = "foo"

			tunnel1 = {
				remote_address    = "203.0.113.1"
				%s
				phase1 = {
					dh_groups             = ["ecp384"]
					encryption_algorithms = ["aes256"]
					integrity_algorithms  = ["sha2_384"]
				}
				phase2 = {
					dh_groups             = ["ecp384"]
					encryption_algorithms = ["aes256"]
					integrity_algorithms  = ["sha2_384"]
				}
			}
								
			tunnel2 = {
				remote_address    = "203.0.113.2"
				pre_shared_key    = "secret-345-minimum-20-characters"
				phase1 = {
					dh_groups             = ["ecp384"]
					encryption_algorithms = ["aes256"]
					integrity_algorithms  = ["sha2_384"]
				}
				phase2 = {
					dh_groups             = ["ecp384"]
					encryption_algorithms = ["aes256"]
					integrity_algorithms  = ["sha2_384"]
					}
				}
			}
`, tunnel1PresharedKeyConfig)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// FAIL - pre-shared key via write-only field without version
				Config:      config(`pre_shared_key_wo = "secret-123-minimum-20-characters"`),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
			{
				// FAIL - pre-shared key via legacy field AND write-only field
				Config: config(
					`pre_shared_key = "secret-123-minimum-20-characters"
					pre_shared_key_wo = "secret-123-minimum-20-characters"
					pre_shared_key_wo_version = 1
				`),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
			{
				// FAIL - pre-shared key write-only field missing only version set
				Config:      config(`pre_shared_key_wo_version = 1`),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}
