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
	"name":                "token01",
	"description":         "my description",
	"description_updated": "my description updated",
	"region":              testutil.Region,
	"ttl_duration":        "1h",
}

func inputTokenConfig(name, description string) string {
	return fmt.Sprintf(`
		%s

		resource "stackit_modelserving_token" "token" {
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
				Config: inputTokenConfig(
					tokenResource["name"],
					tokenResource["description"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"stackit_modelserving_token.token",
						"project_id",
						tokenResource["project_id"],
					),
					resource.TestCheckResourceAttr(
						"stackit_modelserving_token.token",
						"region",
						tokenResource["region"],
					),
					resource.TestCheckResourceAttr(
						"stackit_modelserving_token.token",
						"name",
						tokenResource["name"],
					),
					resource.TestCheckResourceAttr(
						"stackit_modelserving_token.token",
						"description",
						tokenResource["description"],
					),
					resource.TestCheckResourceAttr(
						"stackit_modelserving_token.token",
						"ttl_duration",
						tokenResource["ttl_duration"],
					),
					resource.TestCheckResourceAttrSet(
						"stackit_modelserving_token.token",
						"token_id",
					),
					resource.TestCheckResourceAttrSet(
						"stackit_modelserving_token.token",
						"state",
					),
					resource.TestCheckResourceAttrSet(
						"stackit_modelserving_token.token",
						"valid_until",
					),
					resource.TestCheckResourceAttrSet(
						"stackit_modelserving_token.token",
						"content",
					),
				),
			},
			// Update
			{
				Config: inputTokenConfig(
					tokenResource["name"],
					tokenResource["description_updated"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"stackit_modelserving_token.token",
						"project_id",
						tokenResource["project_id"],
					),
					resource.TestCheckResourceAttr(
						"stackit_modelserving_token.token",
						"region",
						tokenResource["region"],
					),
					resource.TestCheckResourceAttr(
						"stackit_modelserving_token.token",
						"name",
						tokenResource["name"],
					),
					resource.TestCheckResourceAttr(
						"stackit_modelserving_token.token",
						"description",
						tokenResource["description_updated"],
					),
					resource.TestCheckResourceAttrSet(
						"stackit_modelserving_token.token",
						"token_id",
					),
					resource.TestCheckResourceAttrSet(
						"stackit_modelserving_token.token",
						"state",
					),
					resource.TestCheckResourceAttrSet(
						"stackit_modelserving_token.token",
						"valid_until",
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
		if rs.Type != "stackit_modelserving_token" {
			continue
		}
		// Token terraform ID: "[projectId],[region],[tokenId]"
		idParts := strings.Split(rs.Primary.ID, core.Separator)
		if len(idParts) != 3 {
			return fmt.Errorf("invalid ID: %s", rs.Primary.ID)
		}
		tokenId := idParts[2]

		_, err := client.GetToken(ctx, testutil.Region, testutil.ProjectId, tokenId).Execute()
		if err != nil {
			return fmt.Errorf("token %s still exists", tokenId)
		}
	}

	return nil
}
