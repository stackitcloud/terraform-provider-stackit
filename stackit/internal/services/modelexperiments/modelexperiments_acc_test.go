package modelexperiments_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-min.tf
var resourceModelexperimentsInstanceMin string

//go:embed testdata/resource-max.tf
var resourceModelexperimentsInstanceMax string

const modelexperimentsInstanceResource = "stackit_modelexperiments_instance.example"
const modelexperimentsInstanceDataResource = "data.stackit_modelexperiments_instance.example"

const modelexperimentsTokenResource = "stackit_modelexperiments_token.example"

var testModelexperimentsConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"region":     config.StringVariable(testutil.Region),
	// Instance
	"name": config.StringVariable("minInstance"),
	// Token
	"token_name": config.StringVariable("minInstanceToken"),
}

var testModelexperimentsConfigVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"region":     config.StringVariable(testutil.Region),
	// Instance
	"name":                         config.StringVariable("maxInstance"),
	"description":                  config.StringVariable("instanceDescription"),
	"deleted_experiment_retention": config.StringVariable("30d"),
	"label_value":                  config.StringVariable("instanceLabel"),
	// Token
	"token_name":        config.StringVariable("maxInstanceToken"),
	"token_description": config.StringVariable("tokenDescription"),
	"ttl_duration":      config.StringVariable("5h30m40s"),
	"token_label_value": config.StringVariable("tokenLabel"),
}

func testModelexperimentsInstanceConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testModelexperimentsConfigVarsMin))
	maps.Copy(tempConfig, testModelexperimentsConfigVarsMin)
	tempConfig["name"] = config.StringVariable("tfAccModelexperimentsMinInstanceUpd")
	tempConfig["description"] = config.StringVariable("description-upd")
	return tempConfig
}

func testModelexperimentsInstanceConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testModelexperimentsConfigVarsMax))
	maps.Copy(tempConfig, testModelexperimentsConfigVarsMax)
	tempConfig["name"] = config.StringVariable("tfAccModelexperimentsMaxInstanceUpd")
	tempConfig["description"] = config.StringVariable("description-upd")
	return tempConfig
}

func TestAccModelExperimentsInstanceMin(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckModelExperimentsInstanceDestroy,
		Steps: []resource.TestStep{
			// 1) Creation
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceModelexperimentsInstanceMin,
				ConfigVariables: testModelexperimentsConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "id"),

					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "name", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMin["name"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "description", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMin["description"])),

					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "bucket_name"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "deleted_experiment_retention"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "url"),

					// Token
					resource.TestCheckResourceAttr(modelexperimentsTokenResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsTokenResource, "region", testutil.Region),
					resource.TestCheckResourceAttrSet(modelexperimentsTokenResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsTokenResource, "token_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsTokenResource, "id"),

					resource.TestCheckResourceAttr(modelexperimentsTokenResource, "name", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMin["token_name"])),
					resource.TestCheckResourceAttr(modelexperimentsTokenResource, "description", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMin["token_description"])),
				),
			},
			// 2) Data Source
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceModelexperimentsInstanceMin,
				ConfigVariables: testModelexperimentsConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "project_id",
						modelexperimentsInstanceDataResource, "project_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "region",
						modelexperimentsInstanceDataResource, "region",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "instance_id",
						modelexperimentsInstanceDataResource, "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "name",
						modelexperimentsInstanceDataResource, "name",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "description",
						modelexperimentsInstanceDataResource, "description",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "state",
						modelexperimentsInstanceDataResource, "state",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "bucket_name",
						modelexperimentsInstanceDataResource, "bucket_name",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "deleted_experiment_retention",
						modelexperimentsInstanceDataResource, "deleted_experiment_retention",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "url",
						modelexperimentsInstanceDataResource, "url",
					),
				),
			},
			// 3) Import
			{
				ConfigVariables:   testModelexperimentsConfigVarsMin,
				ResourceName:      modelexperimentsInstanceResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[modelexperimentsInstanceResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", modelexperimentsInstanceResource)
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instanceId")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
			},
			// 4) Update
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceModelexperimentsInstanceMin,
				ConfigVariables: testModelexperimentsInstanceConfigVarsMinUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "name", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "description", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMinUpdated()["description"])),

					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "bucket_name"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "deleted_experiment_retention"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "url"),
				),
			},
		},
	})
}

func TestAccModelExperimentsInstanceMax(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckModelExperimentsInstanceDestroy,
		Steps: []resource.TestStep{
			// 1) Creation
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceModelexperimentsInstanceMax,
				ConfigVariables: testModelexperimentsConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "id"),

					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "name", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["name"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "description", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["description"])),

					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "bucket_name"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "deleted_experiment_retention"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "url"),

					// Token
					resource.TestCheckResourceAttr(modelexperimentsTokenResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsTokenResource, "region", testutil.Region),
					resource.TestCheckResourceAttrSet(modelexperimentsTokenResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsTokenResource, "token_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsTokenResource, "id"),

					resource.TestCheckResourceAttr(modelexperimentsTokenResource, "name", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["token_name"])),
					resource.TestCheckResourceAttr(modelexperimentsTokenResource, "description", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["token_description"])),
				),
			},
			// 2) Data Source
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceModelexperimentsInstanceMax,
				ConfigVariables: testModelexperimentsConfigVarsMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "project_id",
						modelexperimentsInstanceDataResource, "project_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "region",
						modelexperimentsInstanceDataResource, "region",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "instance_id",
						modelexperimentsInstanceDataResource, "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "name",
						modelexperimentsInstanceDataResource, "name",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "description",
						modelexperimentsInstanceDataResource, "description",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "state",
						modelexperimentsInstanceDataResource, "state",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "bucket_name",
						modelexperimentsInstanceDataResource, "bucket_name",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "deleted_experiment_retention",
						modelexperimentsInstanceDataResource, "deleted_experiment_retention",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "url",
						modelexperimentsInstanceDataResource, "url",
					),
				),
			},
			// 3) Import
			{
				ConfigVariables:   testModelexperimentsConfigVarsMax,
				ResourceName:      modelexperimentsInstanceResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[modelexperimentsInstanceResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", modelexperimentsInstanceResource)
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instanceId")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
			},
			{
				ConfigVariables:   testModelexperimentsConfigVarsMax,
				ResourceName:      modelexperimentsTokenResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[modelexperimentsTokenResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", modelexperimentsTokenResource)
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instanceId")
					}
					tokenId, ok := r.Primary.Attributes["token_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute tokenId")
					}

					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, tokenId), nil
				},
				ImportStateVerifyIgnore: []string{"token"},
			},
			// 4) Update
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceModelexperimentsInstanceMax,
				ConfigVariables: testModelexperimentsInstanceConfigVarsMaxUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "name", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "description", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["description"])),

					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "state"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "bucket_name"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "deleted_experiment_retention"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "url"),
				),
			},
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
