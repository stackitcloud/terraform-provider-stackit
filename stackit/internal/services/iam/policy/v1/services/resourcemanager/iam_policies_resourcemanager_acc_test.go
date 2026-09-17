package resourcemanager_test

import (
	_ "embed"
	"fmt"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testdestroy"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/folder.tf
	folderConfig string
)

func TestAccFolderIamPolicyV1Resource(t *testing.T) {
	configVars := config.Variables{
		"organization_id":    config.StringVariable(testutil.OrganizationId),
		"folder_name":        config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"folder_owner_email": config.StringVariable(testutil.TestProjectServiceAccountEmail),
		"role_bindings": config.ListVariable(
			config.ObjectVariable(map[string]config.Variable{
				"role":    config.StringVariable("owner"),
				"subject": config.StringVariable(testutil.TestProjectServiceAccountEmail),
			}),
		),
	}

	configVarsUpdated := func() config.Variables {
		tempConfig := maps.Clone(configVars)
		tempConfig["role_bindings"] = config.ListVariable() // empty list

		return tempConfig
	}

	const resourceIdentifier = "stackit_resourcemanager_folder_iam_policy_v1.iam_policy"
	const datasourceIdentifier = "data." + resourceIdentifier

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testdestroy.AccTestCheckDestroy,
		Steps: []resource.TestStep{
			// Create aka. initial update (IAM policies can only be *updated*, not created)
			{
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig(), folderConfig),
				ConfigVariables: configVars,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceIdentifier, "resource_id"),
					resource.TestCheckResourceAttr(resourceIdentifier, "role_bindings.#", "1"),
					resource.TestCheckResourceAttr(resourceIdentifier, "role_bindings.0.role", "owner"),
					resource.TestCheckResourceAttr(resourceIdentifier, "role_bindings.0.subject", testutil.TestProjectServiceAccountEmail),
				),
			},
			// Datasource
			{
				Config: fmt.Sprintf("%s\n%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig(), folderConfig, `
data "stackit_resourcemanager_folder_iam_policy_v1" "iam_policy" {
  resource_id = stackit_resourcemanager_folder.folder.folder_id
}
`),
				ConfigVariables: configVars,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceIdentifier, "resource_id", datasourceIdentifier, "resource_id"),
					resource.TestCheckResourceAttr(datasourceIdentifier, "role_bindings.#", "1"),
					resource.TestCheckResourceAttr(datasourceIdentifier, "role_bindings.0.role", testutil.ConvertConfigVariable(configVars["role"])),
					resource.TestCheckResourceAttr(datasourceIdentifier, "role_bindings.0.subject", testutil.ConvertConfigVariable(configVars["subject"])),
				),
			},
			// Update
			{
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig(), folderConfig),
				ConfigVariables: configVarsUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceIdentifier, "resource_id"),
					resource.TestCheckResourceAttr(resourceIdentifier, "role_bindings.#", "0"),
				),
			},
		},
	})
}
