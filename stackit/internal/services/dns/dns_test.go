package dns

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestCreateTimeout(t *testing.T) {
	// only tests create timeout, read/update/delete would need a successful create beforehand. We could do this, but
	// these tests would be slow and flaky
	projectID := uuid.NewString()
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	providerConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "eu01"
	dns_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}
`, s.Server.URL)
	zoneResource := fmt.Sprintf(`
variable "name" {}

resource "stackit_dns_zone" "zone" {
	project_id = "%s"
	name = var.name
	dns_name = "dns.example.com"
	timeouts = {
		create = "10ms"
		read   = "10ms"
		update = "10ms"
		delete = "10ms"
	}
}
`, projectID)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// create fails
				PreConfig: func() {
					s.Reset(testutil.MockResponse{
						Handler: func(_ http.ResponseWriter, r *http.Request) {
							ctx := r.Context()
							select {
							case <-ctx.Done():
							case <-time.After(20 * time.Millisecond):
							}
						},
					})
				},
				Config:      providerConfig + "\n" + zoneResource,
				ExpectError: regexp.MustCompile("deadline exceeded"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable("create-zone"),
				},
			},
		},
	})
}
