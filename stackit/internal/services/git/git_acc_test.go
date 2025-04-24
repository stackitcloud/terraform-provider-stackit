package git

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/git"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource.tf
var resourceConfig string

var name = fmt.Sprintf("git-%s-instance", testutil.GenerateRandomString(5))
var nameUpdated = fmt.Sprintf("git-%s-instance", testutil.GenerateRandomString(5))

var testConfigVars = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(name),
}

func testConfigVarsUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVars))
	maps.Copy(tempConfig, testConfigVars)
	// update git instance to a new name
	// should trigger creating a new instance
	tempConfig["name"] = config.StringVariable(nameUpdated)
	return tempConfig
}

func TestGitInstance(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitInstanceDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVars,
				Config:          testutil.GitProviderConfig() + resourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_git.git", "project_id", testutil.ConvertConfigVariable(testConfigVars["project_id"])),
					resource.TestCheckResourceAttr("stackit_git.git", "name", testutil.ConvertConfigVariable(testConfigVars["name"])),
					resource.TestCheckResourceAttrSet("stackit_git.git", "name"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "url"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "version"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "instance_id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVars,
				Config: fmt.Sprintf(`
					%s

					data "stackit_git" "git" {
						project_id  = stackit_git.git.project_id
						instance_id = stackit_git.git.instance_id
					}
					`, testutil.GitProviderConfig()+resourceConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_git.git", "project_id", testutil.ConvertConfigVariable(testConfigVars["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "project_id",
						"data.stackit_git.git", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "instance_id",
						"data.stackit_git.git", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "name",
						"data.stackit_git.git", "name",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "url",
						"data.stackit_git.git", "url",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "version",
						"data.stackit_git.git", "version",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVars,
				ResourceName:    "stackit_git.git",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_git.git"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_git.git")
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
			// Update
			{
				ConfigVariables: testConfigVarsUpdated(),
				Config:          testutil.GitProviderConfig() + resourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_git.git", "project_id", testutil.ConvertConfigVariable(testConfigVars["project_id"])),
					resource.TestCheckResourceAttr("stackit_git.git", "name", testutil.ConvertConfigVariable(testConfigVarsUpdated()["name"])),
					resource.TestCheckResourceAttrSet("stackit_git.git", "name"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "url"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "version"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "instance_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckGitInstanceDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *git.APIClient
	var err error

	if testutil.GitCustomEndpoint == "" {
		client, err = git.NewAPIClient()
	} else {
		client, err = git.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.GitCustomEndpoint),
		)
	}

	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var instancesToDestroy []string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_git" {
			continue
		}
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.ListInstances(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting git instances: %w", err)
	}

	gitInstances := *instancesResp.Instances
	for i := range gitInstances {
		if gitInstances[i].Id == nil {
			continue
		}
		if utils.Contains(instancesToDestroy, *gitInstances[i].Id) {
			err := client.DeleteInstance(ctx, testutil.ProjectId, *gitInstances[i].Id).Execute()
			if err != nil {
				return fmt.Errorf("destroying git instance %s during CheckDestroy: %w", *gitInstances[i].Id, err)
			}
		}
	}
	return nil
}
