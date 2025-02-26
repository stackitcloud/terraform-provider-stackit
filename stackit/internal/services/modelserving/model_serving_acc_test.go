package modelserving_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/modelserving"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// Token resource data
var tokenResource = map[string]string{
	"project_id":          testutil.ProjectId,
	"name":                testutil.ResourceNameWithDateTime("token"),
	"description":         "my description",
	"description_updated": "my description updated",
	"region":              testutil.Region,
	"ttl_duration":        "1h",
}

func inputTokenConfig(name, description string) string {
	return fmt.Sprintf(`
		%s

		resource "stackit_model_serving_token" "token" {
			project_id = "%s"
			region = "%s"
			name = "%s"
			description = "%s"
			ttl_duration = "%s"
		}
		`,
		testutil.ModelServingProviderConfig(),
		tokenResource["project_id"],
		tokenResource["region"],
		name,
		description,
		tokenResource["ttl_duration"],
	)
}

func TestAccModelServingTokenResource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckModelServingTokenDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: inputTokenConfig(tokenResource["name"], tokenResource["description"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"stackit_model_serving_token.token",
						"project_id",
						tokenResource["project_id"],
					),
					resource.TestCheckResourceAttr(
						"stackit_model_serving_token.token",
						"region",
						tokenResource["region"],
					),
					resource.TestCheckResourceAttr(
						"stackit_model_serving_token.token",
						"name",
						tokenResource["name"],
					),
					resource.TestCheckResourceAttr(
						"stackit_model_serving_token.token",
						"description",
						tokenResource["description"],
					),
					resource.TestCheckResourceAttr(
						"stackit_model_serving_token.token",
						"ttl_duration",
						tokenResource["ttl_duration"],
					),
					resource.TestCheckResourceAttrSet(
						"stackit_model_serving_token.token",
						"token_id",
					),
					resource.TestCheckResourceAttrSet("stackit_model_serving_token.token", "state"),
					resource.TestCheckResourceAttrSet(
						"stackit_model_serving_token.token",
						"validUntil",
					),
					resource.TestCheckResourceAttrSet(
						"stackit_model_serving_token.token",
						"content",
					),
				),
			},
			// Data Source
			{
				Config: fmt.Sprintf(`
					%s

					data "stackit_model_serving_token" "token" {
						project_id = stackit_model_serving_token.token.project_id
						token_id = stackit_model_serving_token.token.token_id
						region = stackit_model_serving_token.token.region
					}`,
					inputTokenConfig(tokenResource["name"], tokenResource["description"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"stackit_model_serving_token.token", "project_id",
						"data.stackit_model_serving_token.token", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_model_serving_token.token", "token_id",
						"data.stackit_model_serving_token.token", "token_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_model_serving_token.token", "region",
						"data.stackit_model_serving_token.token", "region",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_model_serving_token.token", "name",
						"data.stackit_model_serving_token.token", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_model_serving_token.token", "description",
						"data.stackit_model_serving_token.token", "description",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_model_serving_token.token", "state",
						"data.stackit_model_serving_token.token", "state",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_model_serving_token.token", "validUntil",
						"data.stackit_model_serving_token.token", "validUntil",
					),
				),
			},
			// Import
			{
				ResourceName: "stackit_model_serving_token.token",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_model_serving_token.token"]
					if !ok {
						return "", fmt.Errorf(
							"couldn't find resource stackit_model_serving_token.token",
						)
					}
					tokenId, ok := r.Primary.Attributes["token_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute token_id")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, tokenId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: inputTokenConfig(
					tokenResource["name"],
					tokenResource["description_updated"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"stackit_model_serving_token.token",
						"project_id",
						tokenResource["project_id"],
					),
					resource.TestCheckResourceAttr(
						"stackit_model_serving_token.token",
						"region",
						tokenResource["region"],
					),
					resource.TestCheckResourceAttr(
						"stackit_model_serving_token.token",
						"name",
						tokenResource["name"],
					),
					resource.TestCheckResourceAttr(
						"stackit_model_serving_token.token",
						"description",
						tokenResource["description_updated"],
					),
					resource.TestCheckResourceAttrSet(
						"stackit_model_serving_token.token",
						"token_id",
					),
					resource.TestCheckResourceAttrSet("stackit_model_serving_token.token", "state"),
					resource.TestCheckResourceAttrSet(
						"stackit_model_serving_token.token",
						"validUntil",
					),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckModelServingTokenDestroy(s *terraform.State) error {
	ctx := context.Background()

	var client *modelserving.APIClient
	var err error
	if testutil.ModelServingCustomEndpoint == "" {
		client, err = modelserving.NewAPIClient()
	} else {
		client, err = modelserving.NewAPIClient(
			config.WithEndpoint(testutil.ModelServingCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_model_serving_token" {
			continue
		}
		// Token terraform ID: "[projectId],[tokenId]"
		idParts := strings.Split(rs.Primary.ID, core.Separator)
		if len(idParts) != 2 {
			return fmt.Errorf("invalid ID: %s", rs.Primary.ID)
		}
		tokenId := idParts[1]

		_, err := client.GetToken(ctx, testutil.Region, testutil.ProjectId, tokenId).Execute()
		if err == nil {
			return fmt.Errorf("token %s still exists", tokenId)
		}
	}

	return nil
}
