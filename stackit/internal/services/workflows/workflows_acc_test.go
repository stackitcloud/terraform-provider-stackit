package workflows_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// runSuffix is shared across all tests in this process so a re-run won't
// collide with an instance left behind by a previous failed run. Six chars
// keeps room under the server's 25-char display_name cap.
var runSuffix = acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)

func instanceDisplayName(kind string) string {
	return fmt.Sprintf("tf-%s-%s", runSuffix, kind)
}

var (
	//go:embed testdata/instance.tf
	instanceConfig string

	//go:embed testdata/instance-no-description.tf
	instanceNoDescriptionConfig string

	//go:embed testdata/instance-stackit-idp.tf
	instanceStackITIdPConfig string

	//go:embed testdata/dagbundle-git.tf
	dagBundleGitConfig string

	//go:embed testdata/dagbundle-git-no-subdir.tf
	dagBundleGitNoSubdirConfig string

	//go:embed testdata/dagbundle-s3.tf
	dagBundleS3Config string
)

// requireEnv collects a set of required env vars for these tests; if any are
// missing, skips the test with a clear message. Keeps test runs cheap when
// workflows-specific secrets aren't provisioned in the runner.
func requireEnv(t *testing.T, keys ...string) map[string]string {
	t.Helper()
	out := make(map[string]string, len(keys))
	missing := []string{}
	for _, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			missing = append(missing, k)
		}
		out[k] = v
	}
	if len(missing) > 0 {
		t.Skipf("Skipping: missing required env var(s): %v", missing)
	}
	return out
}

func baseInstanceVars(t *testing.T, displayName, description string) config.Variables {
	env := requireEnv(t,
		"TF_ACC_WORKFLOWS_VERSION",
		"TF_ACC_WORKFLOWS_IDP_NAME",
		"TF_ACC_WORKFLOWS_IDP_CLIENT_ID",
		"TF_ACC_WORKFLOWS_IDP_CLIENT_SECRET",
		"TF_ACC_WORKFLOWS_IDP_SCOPE",
		"TF_ACC_WORKFLOWS_IDP_DISCOVERY_ENDPOINT",
	)
	return config.Variables{
		"project_id":             config.StringVariable(testutil.ProjectId),
		"region":                 config.StringVariable(testutil.Region),
		"display_name":           config.StringVariable(displayName),
		"description":            config.StringVariable(description),
		"instance_version":       config.StringVariable(env["TF_ACC_WORKFLOWS_VERSION"]),
		"idp_name":               config.StringVariable(env["TF_ACC_WORKFLOWS_IDP_NAME"]),
		"idp_client_id":          config.StringVariable(env["TF_ACC_WORKFLOWS_IDP_CLIENT_ID"]),
		"idp_client_secret":      config.StringVariable(env["TF_ACC_WORKFLOWS_IDP_CLIENT_SECRET"]),
		"idp_scope":              config.StringVariable(env["TF_ACC_WORKFLOWS_IDP_SCOPE"]),
		"idp_discovery_endpoint": config.StringVariable(env["TF_ACC_WORKFLOWS_IDP_DISCOVERY_ENDPOINT"]),
	}
}

func bundleVars(t *testing.T, base config.Variables, bundleName, subdir string) config.Variables {
	env := requireEnv(t,
		"TF_ACC_WORKFLOWS_DAGS_GIT_URL",
		"TF_ACC_WORKFLOWS_DAGS_GIT_BRANCH",
		"TF_ACC_WORKFLOWS_DAGS_GIT_USER",
		"TF_ACC_WORKFLOWS_DAGS_GIT_PAT",
	)
	out := make(config.Variables, len(base)+5)
	for k, v := range base {
		out[k] = v
	}
	out["bundle_name"] = config.StringVariable(bundleName)
	out["bundle_url"] = config.StringVariable(env["TF_ACC_WORKFLOWS_DAGS_GIT_URL"])
	out["bundle_branch"] = config.StringVariable(env["TF_ACC_WORKFLOWS_DAGS_GIT_BRANCH"])
	out["bundle_username"] = config.StringVariable(env["TF_ACC_WORKFLOWS_DAGS_GIT_USER"])
	out["bundle_password"] = config.StringVariable(env["TF_ACC_WORKFLOWS_DAGS_GIT_PAT"])
	out["bundle_subdir"] = config.StringVariable(subdir)
	return out
}

func TestAccWorkflowsInstance(t *testing.T) {
	vars := baseInstanceVars(t, instanceDisplayName("wf"), "Acceptance test instance")
	updated := cloneVars(vars)
	updated["description"] = config.StringVariable("Acceptance test instance — updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowsInstanceDestroy,
		Steps: []resource.TestStep{
			{
				ConfigVariables: vars,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + instanceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_workflows_instance.workflow", "project_id", testutil.ConvertConfigVariable(vars["project_id"])),
					resource.TestCheckResourceAttr("stackit_workflows_instance.workflow", "region", testutil.ConvertConfigVariable(vars["region"])),
					resource.TestCheckResourceAttr("stackit_workflows_instance.workflow", "display_name", testutil.ConvertConfigVariable(vars["display_name"])),
					resource.TestCheckResourceAttr("stackit_workflows_instance.workflow", "description", testutil.ConvertConfigVariable(vars["description"])),
					resource.TestCheckResourceAttr("stackit_workflows_instance.workflow", "version", testutil.ConvertConfigVariable(vars["instance_version"])),
					resource.TestCheckResourceAttr("stackit_workflows_instance.workflow", "identity_provider.type", "oauth2"),
					resource.TestCheckResourceAttr("stackit_workflows_instance.workflow", "identity_provider.client_secret", testutil.ConvertConfigVariable(vars["idp_client_secret"])),
					resource.TestCheckResourceAttrSet("stackit_workflows_instance.workflow", "instance_id"),
					resource.TestCheckResourceAttrSet("stackit_workflows_instance.workflow", "endpoints.url"),
					resource.TestCheckResourceAttrSet("stackit_workflows_instance.workflow", "endpoints.redirect_url"),
					resource.TestCheckResourceAttrSet("stackit_workflows_instance.workflow", "status"),
					resource.TestCheckResourceAttrSet("stackit_workflows_instance.workflow", "created_at"),
				),
			},
			{
				ConfigVariables: vars,
				Config: testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + instanceConfig + `
				data "stackit_workflows_instance" "workflow" {
				  project_id   = stackit_workflows_instance.workflow.project_id
				  region       = stackit_workflows_instance.workflow.region
				  instance_id  = stackit_workflows_instance.workflow.instance_id
				}
				data "stackit_workflows_instances" "all" {
				  project_id = stackit_workflows_instance.workflow.project_id
				  region     = stackit_workflows_instance.workflow.region
				}
				data "stackit_workflows_provider_options" "options" {
				  region = stackit_workflows_instance.workflow.region
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("stackit_workflows_instance.workflow", "instance_id", "data.stackit_workflows_instance.workflow", "instance_id"),
					resource.TestCheckResourceAttrPair("stackit_workflows_instance.workflow", "display_name", "data.stackit_workflows_instance.workflow", "display_name"),
					resource.TestCheckResourceAttrPair("stackit_workflows_instance.workflow", "endpoints.url", "data.stackit_workflows_instance.workflow", "endpoints.url"),
					resource.TestCheckResourceAttrPair("stackit_workflows_instance.workflow", "status", "data.stackit_workflows_instance.workflow", "status"),
					resource.TestCheckResourceAttrSet("data.stackit_workflows_instances.all", "instances.#"),
					resource.TestCheckResourceAttrSet("data.stackit_workflows_provider_options.options", "versions.#"),
				),
			},
			// client_secret is not returned by the API; expect it to be absent on import.
			{
				ConfigVariables: vars,
				ResourceName:    "stackit_workflows_instance.workflow",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_workflows_instance.workflow"]
					if !ok {
						return "", fmt.Errorf("not found: stackit_workflows_instance.workflow")
					}
					instanceID, ok := rs.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("instance_id not set")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceID), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"identity_provider.client_secret"},
			},
			{
				ConfigVariables: updated,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + instanceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_workflows_instance.workflow", "description", testutil.ConvertConfigVariable(updated["description"])),
				),
			},
			// Verify clearing description: server treats "" as the clear sentinel; provider sends it via ClearableString.
			// Server treats "" as the clear sentinel; provider sends it via clearableString.
			{
				ConfigVariables: updated,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + instanceNoDescriptionConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("stackit_workflows_instance.workflow", "description"),
				),
			},
			// Verify client_secret rotation: server requires the secret on every IdP update (credential-leak defense).
			// Server requires the secret on every IdP update (credential-leak defense).
			{
				ConfigVariables: rotatedIdPSecret(updated, "rotated-secret-value"),
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + instanceNoDescriptionConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_workflows_instance.workflow", "identity_provider.client_secret", "rotated-secret-value"),
				),
			},
		},
	})
}

func bundleS3Vars(t *testing.T, base config.Variables, bundleName, prefix string) config.Variables {
	env := requireEnv(t,
		"TF_ACC_WORKFLOWS_DAGS_S3_BUCKET",
		"TF_ACC_WORKFLOWS_DAGS_S3_ENDPOINT",
		"TF_ACC_WORKFLOWS_DAGS_S3_ACCESS_KEY_ID",
		"TF_ACC_WORKFLOWS_DAGS_S3_SECRET_ACCESS_KEY",
	)
	out := make(config.Variables, len(base)+6)
	for k, v := range base {
		out[k] = v
	}
	out["bundle_name"] = config.StringVariable(bundleName)
	out["bucket_name"] = config.StringVariable(env["TF_ACC_WORKFLOWS_DAGS_S3_BUCKET"])
	out["endpoint"] = config.StringVariable(env["TF_ACC_WORKFLOWS_DAGS_S3_ENDPOINT"])
	out["prefix"] = config.StringVariable(prefix)
	out["access_key_id"] = config.StringVariable(env["TF_ACC_WORKFLOWS_DAGS_S3_ACCESS_KEY_ID"])
	out["secret_access_key"] = config.StringVariable(env["TF_ACC_WORKFLOWS_DAGS_S3_SECRET_ACCESS_KEY"])
	return out
}

func TestAccWorkflowsDagBundleS3(t *testing.T) {
	base := baseInstanceVars(t, instanceDisplayName("wfs3"), "Acceptance test instance for S3 bundles")
	vars := bundleS3Vars(t, base, "backup-dags", "dags/")
	updated := bundleS3Vars(t, base, "backup-dags", "dags/v2/")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowsInstanceDestroy,
		Steps: []resource.TestStep{
			{
				ConfigVariables: vars,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + dagBundleS3Config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "name", testutil.ConvertConfigVariable(vars["bundle_name"])),
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "s3.bucket_name", testutil.ConvertConfigVariable(vars["bucket_name"])),
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "s3.auth.type", "access_key"),
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "s3.auth.access_key_id", testutil.ConvertConfigVariable(vars["access_key_id"])),
					resource.TestCheckNoResourceAttr("stackit_workflows_dag_bundle.bundle", "git"),
				),
			},
			{
				ConfigVariables:         vars,
				ResourceName:            "stackit_workflows_dag_bundle.bundle",
				ImportStateIdFunc:       importDagBundleID,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"s3.auth.secret_access_key"},
			},
			{
				ConfigVariables: updated,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + dagBundleS3Config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "s3.prefix", testutil.ConvertConfigVariable(updated["prefix"])),
				),
			},
			// Rotate secret_access_key — exercise UpdateDagBundle credential path.
			{
				ConfigVariables: rotateS3Secret(updated, "rotated-s3-secret"),
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + dagBundleS3Config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "s3.auth.secret_access_key", "rotated-s3-secret"),
				),
			},
		},
	})
}

func importDagBundleID(state *terraform.State) (string, error) {
	rs, ok := state.RootModule().Resources["stackit_workflows_dag_bundle.bundle"]
	if !ok {
		return "", fmt.Errorf("not found: stackit_workflows_dag_bundle.bundle")
	}
	instanceID := rs.Primary.Attributes["instance_id"]
	name := rs.Primary.Attributes["name"]
	return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceID, name), nil
}

func TestAccWorkflowsDagBundle(t *testing.T) {
	base := baseInstanceVars(t, instanceDisplayName("wfbn"), "Acceptance test instance for bundles")
	vars := bundleVars(t, base, "main-dags", "dags")
	updated := bundleVars(t, base, "main-dags", "dags/v2")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowsInstanceDestroy,
		Steps: []resource.TestStep{
			{
				ConfigVariables: vars,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + dagBundleGitConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_workflows_instance.workflow", "instance_id"),
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "name", testutil.ConvertConfigVariable(vars["bundle_name"])),
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "git.url", testutil.ConvertConfigVariable(vars["bundle_url"])),
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "git.branch", testutil.ConvertConfigVariable(vars["bundle_branch"])),
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "git.auth.type", "basic"),
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "git.auth.username", testutil.ConvertConfigVariable(vars["bundle_username"])),
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "git.auth.password", testutil.ConvertConfigVariable(vars["bundle_password"])),
					resource.TestCheckNoResourceAttr("stackit_workflows_dag_bundle.bundle", "s3"),
				),
			},
			{
				ConfigVariables: vars,
				Config: testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + dagBundleGitConfig + `
				data "stackit_workflows_dag_bundle" "bundle" {
				  project_id  = stackit_workflows_dag_bundle.bundle.project_id
				  region      = stackit_workflows_dag_bundle.bundle.region
				  instance_id = stackit_workflows_dag_bundle.bundle.instance_id
				  name        = stackit_workflows_dag_bundle.bundle.name
				}
				data "stackit_workflows_dag_bundles" "all" {
				  project_id  = stackit_workflows_dag_bundle.bundle.project_id
				  region      = stackit_workflows_dag_bundle.bundle.region
				  instance_id = stackit_workflows_dag_bundle.bundle.instance_id
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("stackit_workflows_dag_bundle.bundle", "git.url", "data.stackit_workflows_dag_bundle.bundle", "git.url"),
					resource.TestCheckResourceAttrPair("stackit_workflows_dag_bundle.bundle", "git.branch", "data.stackit_workflows_dag_bundle.bundle", "git.branch"),
					resource.TestCheckResourceAttr("data.stackit_workflows_dag_bundles.all", "dag_bundles.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_workflows_dag_bundles.all", "dag_bundles.0.type", "git"),
				),
			},
			{
				ConfigVariables: vars,
				ResourceName:    "stackit_workflows_dag_bundle.bundle",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["stackit_workflows_dag_bundle.bundle"]
					if !ok {
						return "", fmt.Errorf("not found: stackit_workflows_dag_bundle.bundle")
					}
					instanceID := rs.Primary.Attributes["instance_id"]
					name := rs.Primary.Attributes["name"]
					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceID, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"git.auth.password",
				},
			},
			{
				ConfigVariables: updated,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + dagBundleGitConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "git.subdir", testutil.ConvertConfigVariable(updated["bundle_subdir"])),
				),
			},
			// Clear subdir by switching to a config that omits it. Server uses
			// "" as the clear sentinel; provider sends it via clearableString.
			{
				ConfigVariables: updated,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + dagBundleGitNoSubdirConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("stackit_workflows_dag_bundle.bundle", "git.subdir"),
				),
			},
			// Rotate password — exercises UpdateDagBundle credential path.
			{
				ConfigVariables: rotateBundlePassword(updated, "rotated-pat-value"),
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + dagBundleGitConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_workflows_dag_bundle.bundle", "git.auth.password", "rotated-pat-value"),
				),
			},
		},
	})
}

// TestAccWorkflowsDagBundle_RejectsSubdirWithSlashes verifies that
// `subdir = "/dags/"` is rejected at plan time so the user is forced to write
// the canonical form. Without this guard, the server's "" normalization would
// cause a perpetual diff against the user's literal value.
func TestAccWorkflowsDagBundle_RejectsSubdirWithSlashes(t *testing.T) {
	base := baseInstanceVars(t, instanceDisplayName("wfnm"), "Acceptance test subdir validation")
	vars := bundleVars(t, base, "norm-dags", "/dags/")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigVariables: vars,
				Config:          testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + dagBundleGitConfig,
				ExpectError:     regexp.MustCompile(`(?s)subdir.*must not have leading or trailing slashes`),
			},
		},
	})
}

// TestAccWorkflowsInstance_RequiresExperimentFlag verifies the experiment
// gate: a workflows resource declared without `experiments = ["workflows"]`
// in the provider block fails Configure with an actionable error. No API
// call is made because the failure happens before plan.
func TestAccWorkflowsInstance_RequiresExperimentFlag(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// NOTE: provider block intentionally omits .Experiments(...).
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + `
				resource "stackit_workflows_instance" "x" {
				  project_id   = "00000000-0000-0000-0000-000000000000"
				  region       = "eu01"
				  display_name = "x"
				  version      = "workflows-3.0-airflow-3.1"
				  identity_provider = {
				    type               = "oauth2"
				    name               = "n"
				    client_id          = "id"
				    client_secret      = "s"
				    scope              = "openid"
				    discovery_endpoint = "https://idp.example.com/.well-known/openid-configuration"
				  }
				}
				`,
				ExpectError: regexp.MustCompile(`(?s)workflows experiment.*disabled by default`),
			},
		},
	})
}

// TestAccWorkflowsInstance_StackITIdPRejected verifies that requesting the
// `stackit` IdP type is rejected at plan time by the schema validator (the
// backend doesn't yet support it).
func TestAccWorkflowsInstance_StackITIdPRejected(t *testing.T) {
	// Plan-time validator check — no API call, so no env gating.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigVariables: config.Variables{
					"project_id":       config.StringVariable("00000000-0000-0000-0000-000000000000"),
					"region":           config.StringVariable("eu01"),
					"display_name":     config.StringVariable("tf-stackit-idp"),
					"instance_version": config.StringVariable("workflows-3.0-airflow-3.1"),
				},
				Config:      testutil.NewConfigBuilder().Experiments(testutil.ExperimentWorkflows).BuildProviderConfig() + "\n" + instanceStackITIdPConfig,
				ExpectError: regexp.MustCompile(`(?s)(must be one of|value must be one of).*oauth2`),
			},
		},
	})
}

func cloneVars(in config.Variables) config.Variables {
	out := make(config.Variables, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func rotatedIdPSecret(in config.Variables, newSecret string) config.Variables {
	out := cloneVars(in)
	out["idp_client_secret"] = config.StringVariable(newSecret)
	return out
}

func rotateBundlePassword(in config.Variables, newPassword string) config.Variables {
	out := cloneVars(in)
	out["bundle_password"] = config.StringVariable(newPassword)
	return out
}

func rotateS3Secret(in config.Variables, newSecret string) config.Variables {
	out := cloneVars(in)
	out["secret_access_key"] = config.StringVariable(newSecret)
	return out
}

func testAccCheckWorkflowsInstanceDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := workflows.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.WorkflowsCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_workflows_instance" {
			continue
		}
		instancesToDestroy = append(instancesToDestroy, rs.Primary.Attributes["instance_id"])
	}

	for _, id := range instancesToDestroy {
		_, err := client.DefaultAPI.GetInstance(ctx, testutil.ProjectId, testutil.Region, id).Execute()
		if err == nil {
			return fmt.Errorf("Workflows instance %s still exists", id)
		}
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			continue
		}
		return fmt.Errorf("unexpected error checking instance %s: %w", id, err)
	}
	return nil
}
