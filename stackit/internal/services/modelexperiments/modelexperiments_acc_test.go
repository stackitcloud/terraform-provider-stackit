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

const modelexperimentsInstanceTokenResource = "stackit_modelexperiments_token.example" // nolint:gosec // This is a TF resource name, not a credential
const modelexperimentsInstanceTokenDataResource = "data.stackit_modelexperiments_token.example"

var testModelexperimentsConfigVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"region":     config.StringVariable(testutil.Region),
	// Instance
	"name": config.StringVariable("tfAccTest-minInstance"),
	// Token
	"token_name": config.StringVariable("tfAccTest-minInstanceToken"),
}

var testModelexperimentsConfigVarsMax = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"region":     config.StringVariable(testutil.Region),
	// Instance
	"name":                         config.StringVariable("tfAccTest-maxInstance"),
	"description":                  config.StringVariable("instanceDescription"),
	"deleted_experiment_retention": config.StringVariable("30d"),
	"label_value":                  config.StringVariable("instanceLabel"),
	// Token
	"token_name":        config.StringVariable("tfAccTest-maxInstanceToken"),
	"token_description": config.StringVariable("tokenDescription"),
	"ttl_duration":      config.StringVariable("5h30m40s"),
	"token_label_value": config.StringVariable("tokenLabel"),
}

func testModelexperimentsInstanceConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testModelexperimentsConfigVarsMin))
	maps.Copy(tempConfig, testModelexperimentsConfigVarsMin)
	tempConfig["name"] = config.StringVariable("tfAccTest-minInstance-upd")
	tempConfig["token_name"] = config.StringVariable("tfAccTest-minInstanceToken-upd")
	return tempConfig
}

func testModelexperimentsInstanceConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testModelexperimentsConfigVarsMax))
	maps.Copy(tempConfig, testModelexperimentsConfigVarsMax)
	// Instance
	tempConfig["name"] = config.StringVariable("tfAccTest-maxInstance-upd")
	tempConfig["description"] = config.StringVariable("instanceDescription-upd")
	tempConfig["deleted_experiment_retention"] = config.StringVariable("2d")
	tempConfig["label_value"] = config.StringVariable("instanceLabel-upd")
	// Token
	tempConfig["token_name"] = config.StringVariable("tfAccTest-maxInstanceToken-upd")
	tempConfig["token_description"] = config.StringVariable("tokenDescription-upd")
	tempConfig["token_label_value"] = config.StringVariable("tokenLabel-upd")
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
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "name", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMin["name"])),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "deleted_experiment_retention"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "url"),

					// Token
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMin["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "name", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMin["token_name"])),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "token_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "token"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "valid_until"),
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
						modelexperimentsInstanceResource, "deleted_experiment_retention",
						modelexperimentsInstanceDataResource, "deleted_experiment_retention",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "url",
						modelexperimentsInstanceDataResource, "url",
					),
					// Token
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "project_id",
						modelexperimentsInstanceTokenDataResource, "project_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "region",
						modelexperimentsInstanceTokenDataResource, "region",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "instance_id",
						modelexperimentsInstanceTokenDataResource, "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "token_id",
						modelexperimentsInstanceTokenDataResource, "token_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "name",
						modelexperimentsInstanceTokenDataResource, "name",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "valid_until",
						modelexperimentsInstanceTokenDataResource, "valid_until",
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
					// Instance
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "name", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "deleted_experiment_retention"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "url"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "bucket_name"),

					// Token
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "name", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMinUpdated()["token_name"])),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "token_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "token"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "valid_until"),
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
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "name", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["name"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "description", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["description"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "deleted_experiment_retention", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["deleted_experiment_retention"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "labels.label", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["label_value"])),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "url"),

					// Token
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "name", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["token_name"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "description", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["token_description"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "ttl_duration", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["ttl_duration"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "labels.label", testutil.ConvertConfigVariable(testModelexperimentsConfigVarsMax["token_label_value"])),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "token_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "token"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "valid_until"),
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
						modelexperimentsInstanceResource, "deleted_experiment_retention",
						modelexperimentsInstanceDataResource, "deleted_experiment_retention",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceResource, "url",
						modelexperimentsInstanceDataResource, "url",
					),
					// Token
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "project_id",
						modelexperimentsInstanceTokenDataResource, "project_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "region",
						modelexperimentsInstanceTokenDataResource, "region",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "instance_id",
						modelexperimentsInstanceTokenDataResource, "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "token_id",
						modelexperimentsInstanceTokenDataResource, "token_id",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "name",
						modelexperimentsInstanceTokenDataResource, "name",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "description",
						modelexperimentsInstanceTokenDataResource, "description",
					),
					resource.TestCheckResourceAttrPair(
						modelexperimentsInstanceTokenResource, "valid_until",
						modelexperimentsInstanceTokenDataResource, "valid_until",
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
			// 4) Update
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceModelexperimentsInstanceMax,
				ConfigVariables: testModelexperimentsInstanceConfigVarsMaxUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "name", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "description", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["description"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "deleted_experiment_retention", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["deleted_experiment_retention"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceResource, "labels.label", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["label_value"])),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceResource, "url"),

					// Token
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "project_id", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "region", testutil.Region),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "name", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["token_name"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "description", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["token_description"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "ttl_duration", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["ttl_duration"])),
					resource.TestCheckResourceAttr(modelexperimentsInstanceTokenResource, "labels.label", testutil.ConvertConfigVariable(testModelexperimentsInstanceConfigVarsMaxUpdated()["token_label_value"])),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "instance_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "token_id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "id"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "token"),
					resource.TestCheckResourceAttrSet(modelexperimentsInstanceTokenResource, "valid_until"),
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

		// Token terraform ID: "[project_id],[region],[instance_id]"
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
