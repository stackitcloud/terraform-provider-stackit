package modelexperiments_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//nolint:all
var instanceResource = map[string]string{
	"project_id":                testutil.ProjectId,
	"name":                      "tf acc test instance01",
	"description":               "my description",
	"description_updated":       "my description updated",
	"region":                    testutil.Region,
	"token_name":                "tf acc test token01",
	"token_description":         "my token description",
	"token_description_updated": "my token description updated",
}

func inputInstanceConfig(instanceName, instanceDescription, token_name, token_description string) string {
	return fmt.Sprintf(`
		%s

		resource "stackit_modelexperiments_instance" "instance" {
  			project_id   = "%s"
  			name         = "%s"
  			region       = "%s"
  			description = "%s"
		}

		resource "stackit_modelexperiments_token" "token" {
  			project_id   = "%s"
  			name         = "%s"
  			region       = "%s"
  			instance_id = stackit_modelexperiments_instance.instance.instance_id
  			description =  "%s"
		}
		`,
		testutil.NewConfigBuilder().BuildProviderConfig(),
		instanceResource["project_id"],
		instanceName,
		instanceResource["region"],
		instanceDescription,
		instanceResource["project_id"],
		token_name,
		instanceResource["region"],
		token_description,
	)
}

func TestAccModelExperimentsInstanceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckModelExperimentsInstanceDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: inputInstanceConfig(
					instanceResource["name"],
					instanceResource["description"],
					instanceResource["token_name"],
					instanceResource["token_description"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_modelexperiments_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_instance.instance", "region", instanceResource["region"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_instance.instance", "description", instanceResource["description"]),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "state"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "bucket_name"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "deleted_experiment_retention"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "url"),
					resource.TestCheckResourceAttr("stackit_modelexperiments_token.token", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_token.token", "region", instanceResource["region"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_token.token", "name", instanceResource["token_name"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_token.token", "description", instanceResource["token_description"]),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "token_id"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "state"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "token"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "valid_until"),
				),
			},
			// Update
			{
				Config: inputInstanceConfig(
					instanceResource["name"],
					instanceResource["description_updated"],
					instanceResource["token_name"],
					instanceResource["token_description_updated"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_modelexperiments_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_instance.instance", "region", instanceResource["region"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_instance.instance", "description", instanceResource["description_updated"]),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "state"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "bucket_name"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "deleted_experiment_retention"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_instance.instance", "url"),
					resource.TestCheckResourceAttr("stackit_modelexperiments_token.token", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_token.token", "region", instanceResource["region"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_token.token", "name", instanceResource["token_name"]),
					resource.TestCheckResourceAttr("stackit_modelexperiments_token.token", "description", instanceResource["token_description_updated"]),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "token_id"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "state"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "token"),
					resource.TestCheckResourceAttrSet("stackit_modelexperiments_token.token", "valid_until"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckModelExperimentsInstanceDestroy(s *terraform.State) error {
	fmt.Println("destroying resources")
	ctx := context.Background()
	client, err := modelexperiments.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.ModelExperimentsCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_modelexperiments_instance" {
			continue
		}

		// Token terraform ID: "[project_id],[region],[token_id]"
		idParts := strings.Split(rs.Primary.ID, core.Separator)
		if len(idParts) != 3 {
			return fmt.Errorf("invalid ID: %s", rs.Primary.ID)
		}
		if idParts[2] != "" {
			instancesToDestroy = append(instancesToDestroy, idParts[2])
		}
	}

	if len(instancesToDestroy) == 0 {
		return nil
	}

	instancesResp, err := client.DefaultAPI.ListInstances(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("getting instanceResp: %w", err)
	}

	if len(instancesResp.Instances) == 0 {
		fmt.Print("No instances found for project \n")
		return nil
	}

	items := instancesResp.Instances
	for i := range items {
		if slices.Contains(instancesToDestroy, items[i].Name) {
			_, err := client.DefaultAPI.DeleteInstance(ctx, testutil.ProjectId, testutil.Region, items[i].Id).Execute()
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: %w", items[i].Name, err)
			}
			_, err = wait.DeleteInstanceWaitHandler(ctx, client.DefaultAPI, testutil.Region, testutil.ProjectId, items[i].Id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying token %s during CheckDestroy: waiting for deletion %w", items[i].Name, err)
			}
		}
	}
	return nil
}
