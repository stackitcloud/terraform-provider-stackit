package ufw_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestAccUfwInstanceResource(t *testing.T) {
	projectId := testutil.ProjectId

	providerConfig := testutil.NewConfigBuilder().BuildProviderConfig()

	tfConfig := fmt.Sprintf(`
		%s

		resource "stackit_edgecloud_instance" "target" {
			project_id = "%s"
			display_name = "edge"
  			plan_id      = "4916c0e2-e719-445a-9920-58e491cd06c5"
  			description  = "cats live on the edge"
  			region       = "eu01"
		}

		resource "stackit_ufw_instance" "example" {
			project_id  = "%s"
			region      = "eu01"
			instance_id = stackit_edgecloud_instance.target.instance_id
			product     = "edge-cloud"
			source_ip   = "192.168.0.0/24"
			type        = "ACL"
		}
	`, providerConfig, projectId, projectId)

	tfConfigUpdated := fmt.Sprintf(`
		%s

		resource "stackit_edgecloud_instance" "target" {
			project_id = "%s"
			display_name = "edge"
  			plan_id      = "4916c0e2-e719-445a-9920-58e491cd06c5"
  			description  = "cats live on the edge"
  			region       = "eu01"
		}

		resource "stackit_ufw_instance" "example" {
			project_id  = "%s"
			region      = "eu01"
			instance_id = stackit_edgecloud_instance.target.instance_id
			product     = "edge-cloud"
			source_ip   = "10.0.0.0/8"
			type        = "ACL"
		}
	`, providerConfig, projectId, projectId)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tfConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "project_id", projectId),
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "source_ip", "192.168.0.0/24"),
					resource.TestCheckResourceAttrSet("stackit_ufw_instance.example", "rule_id"),
				),
			},
			{
				ResourceName:      "stackit_ufw_instance.example",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: tfConfigUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_ufw_instance.example", "source_ip", "10.0.0.0/8"),
				),
			},
		},
	})
}
