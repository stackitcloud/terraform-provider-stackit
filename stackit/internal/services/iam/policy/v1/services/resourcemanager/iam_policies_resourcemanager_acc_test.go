package resourcemanager_test

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

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
		"role":               config.StringVariable("owner"),
		"subject":            config.StringVariable(testutil.TestProjectServiceAccountEmail),
	}

	//configVarsUpdated := func() config.Variables {
	//	tempConfig := maps.Clone(configVars)
	//	tempConfig["record_record1"] = config.StringVariable("1.2.3.5")

	//	return tempConfig
	//}

	const resourceIdentifier = "stackit_resourcemanager_folder_iam_policy_v1.iam_policy"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		// TODO: check destroy
		Steps: []resource.TestStep{
			// Create
			{
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig(), folderConfig),
				ConfigVariables: configVars,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceIdentifier, "resource_id"),
					resource.TestCheckResourceAttr(resourceIdentifier, "role_bindings.#", ""),
					resource.TestCheckResourceAttr(resourceIdentifier, "role_bindings.0.role", testutil.ConvertConfigVariable(configVars["role"])),
					resource.TestCheckResourceAttr(resourceIdentifier, "role_bindings.0.subject", testutil.ConvertConfigVariable(configVars["subject"])),
				),
			},
		},
	})
}
