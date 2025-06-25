package git

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/git"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceMin string

//go:embed testdata/resource-max.tf
var resourceMax string

var nameMin = fmt.Sprintf("git-min-%s-instance", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var nameMinUpdated = fmt.Sprintf("git-min-%s-instance", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var nameMax = fmt.Sprintf("git-max-%s-instance", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var nameMaxUpdated = fmt.Sprintf("git-max-%s-instance", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var aclUpdated = "192.168.1.0/32"

var testConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(nameMin),
}

var testConfigVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(nameMax),
	"acl":        config.StringVariable("192.168.0.0/16"),
	"flavor":     config.StringVariable("git-100"),
}

func testConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMin))
	maps.Copy(tempConfig, testConfigVarsMin)
	// update git instance to a new name
	// should trigger creating a new instance
	tempConfig["name"] = config.StringVariable(nameMinUpdated)
	return tempConfig
}

func testConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigVarsMax))
	maps.Copy(tempConfig, testConfigVarsMax)
	// update git instance to a new name
	// should trigger creating a new instance
	tempConfig["name"] = config.StringVariable(nameMaxUpdated)
	tempConfig["acl"] = config.StringVariable(aclUpdated)

	return tempConfig
}

func TestAccGitMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitInstanceDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.GitProviderConfig() + resourceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_git.git", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_git.git", "name", testutil.ConvertConfigVariable(testConfigVarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_git.git", "url"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "version"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "created"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "consumed_object_storage"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "consumed_disk"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "flavor"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMin,
				Config: fmt.Sprintf(`
					%s

					data "stackit_git" "git" {
						project_id  = stackit_git.git.project_id
						instance_id = stackit_git.git.instance_id
					}
					`, testutil.GitProviderConfig()+resourceMin,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_git.git", "project_id", testutil.ConvertConfigVariable(testConfigVarsMin["project_id"])),
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
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "created",
						"data.stackit_git.git", "created",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "consumed_object_storage",
						"data.stackit_git.git", "consumed_object_storage",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "consumed_disk",
						"data.stackit_git.git", "consumed_disk",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "flavor",
						"data.stackit_git.git", "flavor",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMin,
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
				ConfigVariables: testConfigVarsMinUpdated(),
				Config:          testutil.GitProviderConfig() + resourceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_git.git", "project_id", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_git.git", "name", testutil.ConvertConfigVariable(testConfigVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttrSet("stackit_git.git", "url"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "version"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "created"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "consumed_object_storage"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "consumed_disk"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "flavor"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccGitMax(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitInstanceDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVarsMax,
				Config:          testutil.GitProviderConfig() + resourceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_git.git", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_git.git", "name", testutil.ConvertConfigVariable(testConfigVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_git.git", "flavor", testutil.ConvertConfigVariable(testConfigVarsMax["flavor"])),
					resource.TestCheckResourceAttr("stackit_git.git", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMax["acl"])),
					resource.TestCheckResourceAttrSet("stackit_git.git", "url"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "version"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "created"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "consumed_object_storage"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "consumed_disk"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVarsMax,
				Config: fmt.Sprintf(`
					%s

					data "stackit_git" "git" {
						project_id  = stackit_git.git.project_id
						instance_id = stackit_git.git.instance_id
					}
					`, testutil.GitProviderConfig()+resourceMax,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_git.git", "project_id", testutil.ConvertConfigVariable(testConfigVarsMax["project_id"])),
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
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "created",
						"data.stackit_git.git", "created",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "consumed_object_storage",
						"data.stackit_git.git", "consumed_object_storage",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "consumed_disk",
						"data.stackit_git.git", "consumed_disk",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "flavor",
						"data.stackit_git.git", "flavor",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_git.git", "acl",
						"data.stackit_git.git", "acl",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMax,
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
				ConfigVariables: testConfigVarsMaxUpdated(),
				Config:          testutil.GitProviderConfig() + resourceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_git.git", "project_id", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr("stackit_git.git", "name", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_git.git", "flavor", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["flavor"])),
					resource.TestCheckResourceAttr("stackit_git.git", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMaxUpdated()["acl"])),
					resource.TestCheckResourceAttrSet("stackit_git.git", "url"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "version"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "created"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "consumed_object_storage"),
					resource.TestCheckResourceAttrSet("stackit_git.git", "consumed_disk"),
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
