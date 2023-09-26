package opensearch_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/testutil"
)

// Instance resource data
var instanceResource = map[string]string{
	"project_id": testutil.ProjectId,
	"name":       testutil.ResourceNameWithDateTime("opensearch"),
	"plan_id":    "9e4eac4b-b03d-4d7b-b01b-6d1224aa2d68",
	"plan_name":  "stackit-qa-opensearch-1.2.10-replica",
	"version":    "2",
	"sgw_acl":    "192.168.0.0/24",
}

func resourceConfig() string {
	return fmt.Sprintf(`
				%s

				resource "stackit_opensearch_instance" "instance" {
					project_id = "%s"
					name    = "%s"
					plan_name  = "%s"
 				 	version    = "%s"
				}

				resource "stackit_opensearch_credentials" "credentials" {
					project_id = stackit_opensearch_instance.instance.project_id
					instance_id = stackit_opensearch_instance.instance.instance_id
				}
				`,
		testutil.OpenSearchProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["plan_name"],
		instanceResource["version"],
	)
}

func resourceConfigUpdate() string {
	return fmt.Sprintf(`
				%s

				resource "stackit_opensearch_instance" "instance" {
					project_id = "%s"
					name    = "%s"
					plan_name  = "%s"
 				 	version    = "%s"
					parameters = {
						sgw_acl = "%s"
					}
				}

				resource "stackit_opensearch_credentials" "credentials" {
					project_id = stackit_opensearch_instance.instance.project_id
					instance_id = stackit_opensearch_instance.instance.instance_id
				}
				`,
		testutil.OpenSearchProviderConfig(),
		instanceResource["project_id"],
		instanceResource["name"],
		instanceResource["plan_name"],
		instanceResource["version"],
		instanceResource["sgw_acl"],
	)
}

func TestAccOpenSearchResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckOpenSearchDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				Config: resourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "parameters.sgw_acl"),

					// Credentials data
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credentials.credentials", "project_id",
						"stackit_opensearch_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_opensearch_credentials.credentials", "instance_id",
						"stackit_opensearch_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credentials.credentials", "credentials_id"),
					resource.TestCheckResourceAttrSet("stackit_opensearch_credentials.credentials", "host"),
				),
			},
			// Data source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_opensearch_instance" "instance" {
						project_id  = stackit_opensearch_instance.instance.project_id
						instance_id = stackit_opensearch_instance.instance.instance_id
					}

					data "stackit_opensearch_credentials" "credentials" {
						project_id     = stackit_opensearch_credentials.credentials.project_id
						instance_id    = stackit_opensearch_credentials.credentials.instance_id
					    credentials_id = stackit_opensearch_credentials.credentials.credentials_id
					}`,
					resourceConfig(),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrPair("stackit_opensearch_instance.instance", "instance_id",
						"data.stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrPair("stackit_opensearch_credentials.credentials", "credentials_id",
						"data.stackit_opensearch_credentials.credentials", "credentials_id"),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("data.stackit_opensearch_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_instance.instance", "parameters.sgw_acl"),

					// Credentials data
					resource.TestCheckResourceAttr("data.stackit_opensearch_credentials.credentials", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credentials.credentials", "credentials_id"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credentials.credentials", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credentials.credentials", "port"),
					resource.TestCheckResourceAttrSet("data.stackit_opensearch_credentials.credentials", "uri"),
				),
			},
			// Import
			{
				ResourceName: "stackit_opensearch_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_opensearch_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_opensearch_instance.instance")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: "stackit_opensearch_credentials.credentials",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_opensearch_credentials.credentials"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_opensearch_credentials.credentials")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					credentialsId, ok := r.Primary.Attributes["credentials_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute credentials_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, credentialsId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: resourceConfigUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_opensearch_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_id", instanceResource["plan_id"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "version", instanceResource["version"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_opensearch_instance.instance", "parameters.sgw_acl", instanceResource["sgw_acl"]),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckOpenSearchDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *opensearch.APIClient
	var err error
	if testutil.OpenSearchCustomEndpoint == "" {
		client, err = opensearch.NewAPIClient()
	} else {
		client, err = opensearch.NewAPIClient(
			config.WithEndpoint(testutil.OpenSearchCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_opensearch_instance" {
			continue
		}
		// instance terraform ID: "[project_id],[instance_id]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.GetInstances(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	instances := *instancesResp.Instances
	for i := range instances {
		if instances[i].InstanceId == nil {
			continue
		}
		if utils.Contains(instancesToDestroy, *instances[i].InstanceId) {
			if !checkInstanceDeleteSuccess(&instances[i]) {
				err := client.DeleteInstanceExecute(ctx, testutil.ProjectId, *instances[i].InstanceId)
				if err != nil {
					return fmt.Errorf("destroying instance %s during CheckDestroy: %w", *instances[i].InstanceId, err)
				}
				_, err = opensearch.DeleteInstanceWaitHandler(ctx, client, testutil.ProjectId, *instances[i].InstanceId).WaitWithContext(ctx)
				if err != nil {
					return fmt.Errorf("destroying instance %s during CheckDestroy: waiting for deletion %w", *instances[i].InstanceId, err)
				}
			}
		}
	}
	return nil
}

func checkInstanceDeleteSuccess(i *opensearch.Instance) bool {
	if *i.LastOperation.Type != opensearch.InstanceTypeDelete {
		return false
	}

	if *i.LastOperation.Type == opensearch.InstanceTypeDelete {
		if *i.LastOperation.State != opensearch.InstanceStateSuccess {
			return false
		} else if strings.Contains(*i.LastOperation.Description, "DeleteFailed") || strings.Contains(*i.LastOperation.Description, "failed") {
			return false
		}
	}
	return true
}
