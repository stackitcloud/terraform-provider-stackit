package edgecloud_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	coreConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/edge"
	"github.com/stackitcloud/stackit-sdk-go/services/edge/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	// Currently the API does not verify if the UUID belongs to a valid plan
	// This will change with GA, which will require replacing the random UUIDs with real plan UUIDs
	testPlanId                = uuid.NewString()
	testPlanIdUpdated         = uuid.NewString()
	minTestName               = "min-" + acctest.RandStringFromCharSet(4, acctest.CharSetAlpha)
	testDescription           = "test description"
	testDescriptionUpdated    = "test description updated"
	testExpiration            = 1800
	testRecreateBefore        = 120
	testRecreateBeforeUpdated = 100
)

//go:embed testdata/resource-min.tf
var resourceMin string

//go:embed testdata/resource-max.tf
var resourceMax string

// Minimal configuration
var testConfigVarsMin = config.Variables{
	// region is unset, to verify it is picked up from the provider config
	"project_id":   config.StringVariable(testutil.ProjectId),
	"display_name": config.StringVariable(minTestName),
	"plan_id":      config.StringVariable(testPlanId),
}

// Maximal configuration
func configVarsMax(displayName, planId, description string, expiration, recreateBefore int) config.Variables {
	return config.Variables{
		"project_id":      config.StringVariable(testutil.ProjectId),
		"region":          config.StringVariable(testutil.Region),
		"display_name":    config.StringVariable(displayName),
		"plan_id":         config.StringVariable(planId),
		"description":     config.StringVariable(description),
		"expiration":      config.IntegerVariable(expiration),
		"recreate_before": config.IntegerVariable(recreateBefore),
	}
}

func TestAccEdgeCloudInstanceMin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeCloudInstanceDestroy,
		Steps: []resource.TestStep{
			// resources
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMin,
				ConfigVariables: testConfigVarsMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					// instance
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "project_id", testutil.ProjectId),
					// testutil.Region is also used in testutils.EdgeCloudProviderConfig to define a default_region
					// this checks that this is successfully used for the resource, even if no region is specifically set
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "display_name", minTestName),
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "plan_id", testPlanId),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_instance.test_instance", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_instance.test_instance", "frontend_url"),
					// token
					resource.TestCheckResourceAttr("stackit_edgecloud_token.this", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_edgecloud_token.this", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.this", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.this", "token"),
					// kubeconfig
					resource.TestCheckResourceAttr("stackit_edgecloud_kubeconfig.this", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_edgecloud_kubeconfig.this", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.this", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.this", "kubeconfig"),
				),
			},
			// data sources
			{
				ConfigVariables: testConfigVarsMin,
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_edgecloud_instances.this", "id", fmt.Sprintf("%s,%s",
						testutil.ProjectId,
						testutil.Region,
					)),
					resource.TestCheckResourceAttr("data.stackit_edgecloud_plans.this", "id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_edgecloud_plans.this", "project_id", testutil.ProjectId),
				),
			},
			// import
			{
				ConfigVariables: testConfigVarsMin,
				ResourceName:    "stackit_edgecloud_instance.test_instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_edgecloud_instance.test_instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_edgecloud_instance.test_instance")
					}
					instanceID, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEdgeCloudMax(t *testing.T) {
	displayName := "mx-in-" + acctest.RandStringFromCharSet(2, acctest.CharSetAlpha)
	initialVars := configVarsMax(displayName, testPlanId, testDescription, testExpiration, testRecreateBefore)
	updatedVars := configVarsMax(displayName, testPlanIdUpdated, testDescriptionUpdated, testExpiration, testRecreateBeforeUpdated)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeCloudInstanceDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMax,
				ConfigVariables: initialVars,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "region", testutil.Region),
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "display_name", displayName),
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "plan_id", testPlanId),
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "description", testDescription),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_instance.test_instance", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_instance.test_instance", "frontend_url"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_instance.test_instance", "status"),
				),
			},
			// Data sources
			{
				ConfigVariables: initialVars,
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_edgecloud_instances.this", "id", fmt.Sprintf("%s,%s",
						testutil.ProjectId,
						testutil.Region,
					)),
					resource.TestCheckResourceAttrSet("data.stackit_edgecloud_instances.this", "instances.0.created"),
					// TestCheckResourceAttrSet fails if the value is "", which is an allowed value for the description. That's why it has to be checked via regex
					resource.TestMatchResourceAttr("data.stackit_edgecloud_instances.this", "instances.0.description", regexp.MustCompile("^.*$")),
					resource.TestCheckResourceAttrSet("data.stackit_edgecloud_instances.this", "instances.0.display_name"),
					resource.TestCheckResourceAttrSet("data.stackit_edgecloud_instances.this", "instances.0.frontend_url"),
					resource.TestCheckResourceAttrSet("data.stackit_edgecloud_instances.this", "instances.0.instance_id"),
					resource.TestCheckResourceAttrSet("data.stackit_edgecloud_instances.this", "instances.0.plan_id"),
					resource.TestCheckResourceAttrSet("data.stackit_edgecloud_instances.this", "instances.0.region"),
					resource.TestCheckResourceAttrSet("data.stackit_edgecloud_instances.this", "instances.0.status"),
					// check plans data source
					resource.TestCheckResourceAttr("data.stackit_edgecloud_plans.this", "id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_edgecloud_plans.this", "project_id", testutil.ProjectId),
				),
			},
			// Kubeconfig
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMax,
				ConfigVariables: initialVars,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Kubeconfig by name
					resource.TestCheckResourceAttr("stackit_edgecloud_kubeconfig.by_name", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_edgecloud_kubeconfig.by_name", "instance_name", displayName),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.by_name", "kubeconfig_id"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.by_name", "kubeconfig"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.by_name", "expires_at"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.by_name", "creation_time"),
					resource.TestCheckResourceAttr("stackit_edgecloud_kubeconfig.by_name", "region", testutil.Region),
					// Kubeconfig by id
					resource.TestCheckResourceAttr("stackit_edgecloud_kubeconfig.by_id", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrPair(
						"stackit_edgecloud_instance.test_instance", "instance_id",
						"stackit_edgecloud_kubeconfig.by_id", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.by_id", "kubeconfig_id"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.by_id", "kubeconfig"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.by_id", "expires_at"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_kubeconfig.by_id", "creation_time"),
					resource.TestCheckResourceAttr("stackit_edgecloud_kubeconfig.by_id", "region", testutil.Region),
				),
			},
			// Token
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMax,
				ConfigVariables: initialVars,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Token by name
					resource.TestCheckResourceAttr("stackit_edgecloud_token.by_name", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_edgecloud_token.by_name", "instance_name", displayName),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.by_name", "token_id"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.by_name", "token"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.by_name", "expires_at"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.by_name", "creation_time"),
					resource.TestCheckResourceAttr("stackit_edgecloud_token.by_name", "region", testutil.Region),
					// Token by id
					resource.TestCheckResourceAttr("stackit_edgecloud_token.by_id", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttrPair(
						"stackit_edgecloud_instance.test_instance", "instance_id",
						"stackit_edgecloud_token.by_id", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.by_id", "token_id"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.by_id", "token"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.by_id", "expires_at"),
					resource.TestCheckResourceAttrSet("stackit_edgecloud_token.by_id", "creation_time"),
					resource.TestCheckResourceAttr("stackit_edgecloud_token.by_id", "region", testutil.Region),
				),
			},
			// Update
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMax,
				ConfigVariables: updatedVars,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "plan_id", testPlanIdUpdated),
					resource.TestCheckResourceAttr("stackit_edgecloud_instance.test_instance", "description", testDescriptionUpdated),
				),
			},
			// Import
			{
				ConfigVariables: updatedVars,
				ResourceName:    "stackit_edgecloud_instance.test_instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_edgecloud_instance.test_instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_edgecloud_instance.test_instance")
					}
					instanceID, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEdgeCloudInstance_validation(t *testing.T) {
	validDisplayName := "mx-v-" + acctest.RandStringFromCharSet(2, acctest.CharSetAlpha)
	tooShortDisplayName := "abc"          // Invalid (3 chars)
	tooLongDisplayName := "too-long-name" // Invalid (13 chars)
	invalidUUID := "not-a-uuid"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Display Name Too Short
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMin,
				ConfigVariables: configVarsMax(tooShortDisplayName, testPlanId, testDescription, testExpiration, testRecreateBefore),
				ExpectError:     regexp.MustCompile(fmt.Sprintf(`string length must be between 4 and 8, got: %d`, len(tooShortDisplayName))),
			},
			// Display Name Too Long
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMin,
				ConfigVariables: configVarsMax(tooLongDisplayName, testPlanId, testDescription, testExpiration, testRecreateBefore),
				ExpectError:     regexp.MustCompile(fmt.Sprintf(`string length must be between 4 and 8, got: %d`, len(tooLongDisplayName))),
			},
			// Invalid Project ID
			{
				Config: testutil.EdgeCloudProviderConfig() + "\n" + resourceMin,
				ConfigVariables: config.Variables{
					"project_id":   config.StringVariable(invalidUUID),
					"region":       config.StringVariable(testutil.Region),
					"display_name": config.StringVariable(minTestName),
					"plan_id":      config.StringVariable(testPlanId),
				},
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Attribute project_id value must be an UUID, got: %s`, invalidUUID)),
			},
			// Invalid Plan ID
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMin,
				ConfigVariables: configVarsMax(validDisplayName, invalidUUID, testDescription, testExpiration, testRecreateBefore),
				ExpectError:     regexp.MustCompile(fmt.Sprintf(`Attribute plan_id value must be an UUID, got: %s`, invalidUUID)),
			},
			// Description Too Long
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMax,
				ConfigVariables: configVarsMax(validDisplayName, testPlanId, acctest.RandString(257), testExpiration, testRecreateBefore),
				ExpectError:     regexp.MustCompile(`Attribute description string length must be at most 256`),
			},
		},
	})
}

func TestAccEdgeCloudKubeconfigToken_validation(t *testing.T) {
	displayName := "mx-v-" + acctest.RandStringFromCharSet(2, acctest.CharSetAlpha)
	tooShortExpiration := 599
	tooLongExpiration := 15552001

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Expiration too short
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMax,
				ConfigVariables: configVarsMax(displayName, testPlanId, testDescription, tooShortExpiration, testRecreateBefore),
				ExpectError:     regexp.MustCompile(fmt.Sprintf(`Attribute expiration value must be between 600 and 15552000, got: %d`, tooShortExpiration)),
			},
			// Expiration Too Long
			{
				Config:          testutil.EdgeCloudProviderConfig() + "\n" + resourceMax,
				ConfigVariables: configVarsMax(displayName, testPlanId, testDescription, tooLongExpiration, testRecreateBefore),
				ExpectError:     regexp.MustCompile(fmt.Sprintf(`Attribute expiration value must be between 600 and 15552000, got: %d`, tooLongExpiration)),
			},
		},
	})
}

// testAccCheckEdgeCloudInstanceDestroy verifies that test resources are properly destroyed
func testAccCheckEdgeCloudInstanceDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *edge.APIClient
	var err error

	if testutil.EdgeCloudCustomEndpoint != "" {
		client, err = edge.NewAPIClient(coreConfig.WithEndpoint(testutil.EdgeCloudCustomEndpoint))
	} else {
		client, err = edge.NewAPIClient()
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_edgecloud_instance" {
			continue
		}
		idParts := strings.Split(rs.Primary.ID, core.Separator)
		if len(idParts) != 3 {
			return fmt.Errorf("invalid resource ID format: %s", rs.Primary.ID)
		}
		projectId, region, instanceId := idParts[0], idParts[1], idParts[2]

		_, err := client.GetInstance(ctx, projectId, region, instanceId).Execute()
		if err == nil {
			return fmt.Errorf("edge instance %q still exists", instanceId)
		}

		// If the error is a 404, the resource was successfully deleted
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if !ok || oapiErr.StatusCode != http.StatusNotFound {
			err := client.DeleteInstance(ctx, projectId, region, instanceId).Execute()
			if err != nil {
				return fmt.Errorf("deleting instance %s during CheckDestroy: %w", instanceId, err)
			}
			_, err = wait.DeleteInstanceWaitHandler(ctx, client, projectId, region, instanceId).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("waiting for instance deletion %s during CheckDestroy: %w", instanceId, err)
			}
		}
	}
	return nil
}
