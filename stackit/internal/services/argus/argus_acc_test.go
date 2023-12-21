package argus_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
	"github.com/stackitcloud/stackit-sdk-go/services/argus/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var instanceResource = map[string]string{
	"project_id": testutil.ProjectId,
	"name":       testutil.ResourceNameWithDateTime("argus"),
	"plan_name":  "Monitoring-Medium-EU01",
}

var scrapeConfigResource = map[string]string{
	"project_id":                  testutil.ProjectId,
	"name":                        fmt.Sprintf("scrapeconfig-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum)),
	"urls":                        fmt.Sprintf(`{urls = ["www.%s.de","%s.de"]}`, acctest.RandStringFromCharSet(15, acctest.CharSetAlphaNum), acctest.RandStringFromCharSet(15, acctest.CharSetAlphaNum)),
	"metrics_path":                "/metrics",
	"scheme":                      "https",
	"scrape_interval":             "4m", // non-default
	"saml2_enable_url_parameters": "false",
}

var credentialResource = map[string]string{
	"project_id": testutil.ProjectId,
}

func resourceConfig(instanceName, target, saml2EnableUrlParameters string) string {
	return fmt.Sprintf(`
				%s

				resource "stackit_argus_instance" "instance" {
					project_id = "%s"
					name      = "%s"
					plan_name = "%s"
				}
				
				resource "stackit_argus_scrapeconfig" "scrapeconfig" {
					project_id = stackit_argus_instance.instance.project_id
					instance_id = stackit_argus_instance.instance.instance_id
				    name = "%s"
					metrics_path = "%s"
				    targets = [%s]
					scrape_interval = "%s"
					saml2 = { 
						enable_url_parameters = %s
					}
				}

				resource "stackit_argus_credential" "credential" {
					project_id = stackit_argus_instance.instance.project_id
					instance_id = stackit_argus_instance.instance.instance_id
				}

				`,
		testutil.ArgusProviderConfig(),
		instanceResource["project_id"],
		instanceName,
		instanceResource["plan_name"],
		scrapeConfigResource["name"],
		scrapeConfigResource["metrics_path"],
		target,
		scrapeConfigResource["scrape_interval"],
		saml2EnableUrlParameters,
	)
}

func TestAccResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckArgusDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: resourceConfig(instanceResource["name"], scrapeConfigResource["urls"], scrapeConfigResource["saml2_enable_url_parameters"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "dashboard_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "is_updatable"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "grafana_public_read_access"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "grafana_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "grafana_initial_admin_user"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "grafana_initial_admin_password"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "metrics_retention_days"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "metrics_retention_days_5m_downsampling"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "metrics_retention_days_1h_downsampling"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "metrics_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "metrics_push_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "targets_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "alerting_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "logs_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "logs_push_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "jaeger_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "jaeger_ui_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "otlp_traces_url"),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "zipkin_spans_url"),

					// scrape config data
					resource.TestCheckResourceAttrPair(
						"stackit_argus_instance.instance", "project_id",
						"stackit_argus_scrapeconfig.scrapeconfig", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_argus_instance.instance", "instance_id",
						"stackit_argus_scrapeconfig.scrapeconfig", "instance_id",
					),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "name", scrapeConfigResource["name"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "targets.0.urls.#", "2"),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_path", scrapeConfigResource["metrics_path"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scheme", scrapeConfigResource["scheme"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scrape_interval", scrapeConfigResource["scrape_interval"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scrape_interval", scrapeConfigResource["scrape_interval"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "saml2.enable_url_parameters", scrapeConfigResource["saml2_enable_url_parameters"]),

					// credentials
					resource.TestCheckResourceAttr("stackit_argus_credential.credential", "project_id", credentialResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_argus_instance.instance", "instance_id",
						"stackit_argus_credential.credential", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_argus_credential.credential", "username"),
					resource.TestCheckResourceAttrSet("stackit_argus_credential.credential", "password"),
				),
			}, {
				// Data source
				Config: fmt.Sprintf(`
					%s

					data "stackit_argus_instance" "instance" {
					  	project_id  = stackit_argus_instance.instance.project_id
					  	instance_id = stackit_argus_instance.instance.instance_id
					}
					
					data "stackit_argus_scrapeconfig" "scrapeconfig" {
						project_id  = stackit_argus_scrapeconfig.scrapeconfig.project_id 
					  	instance_id = stackit_argus_scrapeconfig.scrapeconfig.instance_id
					  	name        = stackit_argus_scrapeconfig.scrapeconfig.name
					}
					`,
					resourceConfig(instanceResource["name"], scrapeConfigResource["urls"], scrapeConfigResource["saml2_enable_url_parameters"]),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_argus_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_argus_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_argus_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_argus_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttrPair(
						"stackit_argus_instance.instance", "project_id",
						"data.stackit_argus_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_argus_instance.instance", "instance_id",
						"data.stackit_argus_instance.instance", "instance_id",
					),
					// scrape config data
					resource.TestCheckResourceAttrPair(
						"stackit_argus_scrapeconfig.scrapeconfig", "project_id",
						"data.stackit_argus_scrapeconfig.scrapeconfig", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_argus_scrapeconfig.scrapeconfig", "instance_id",
						"data.stackit_argus_scrapeconfig.scrapeconfig", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_argus_scrapeconfig.scrapeconfig", "name",
						"data.stackit_argus_scrapeconfig.scrapeconfig", "name",
					),
					resource.TestCheckResourceAttr("data.stackit_argus_scrapeconfig.scrapeconfig", "name", scrapeConfigResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_argus_scrapeconfig.scrapeconfig", "targets.0.urls.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_argus_scrapeconfig.scrapeconfig", "metrics_path", scrapeConfigResource["metrics_path"]),
					resource.TestCheckResourceAttr("data.stackit_argus_scrapeconfig.scrapeconfig", "scheme", scrapeConfigResource["scheme"]),
					resource.TestCheckResourceAttr("data.stackit_argus_scrapeconfig.scrapeconfig", "scrape_interval", scrapeConfigResource["scrape_interval"]),
					resource.TestCheckResourceAttr("data.stackit_argus_scrapeconfig.scrapeconfig", "saml2.enable_url_parameters", scrapeConfigResource["saml2_enable_url_parameters"]),
				),
			},

			// Import
			{
				ResourceName: "stackit_argus_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_argus_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_argus_instance.instance")
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
			{
				ResourceName: "stackit_argus_scrapeconfig.scrapeconfig",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_argus_scrapeconfig.scrapeconfig"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_argus_scrapeconfig.scrapeconfig")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, instanceId, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: resourceConfig(fmt.Sprintf("%s-new", instanceResource["name"]), "", "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "name", instanceResource["name"]+"-new"),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "plan_name", instanceResource["plan_name"]),

					// Scrape Config
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "name", scrapeConfigResource["name"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "targets.#", "0"),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_path", scrapeConfigResource["metrics_path"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scheme", scrapeConfigResource["scheme"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scrape_interval", scrapeConfigResource["scrape_interval"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "saml2.%", "1"),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "saml2.enable_url_parameters", "true"),

					// Credentials
					resource.TestCheckResourceAttrSet("stackit_argus_credential.credential", "username"),
					resource.TestCheckResourceAttrSet("stackit_argus_credential.credential", "password"),
				),
			},
			// Update and remove saml2 attribute
			{
				Config: fmt.Sprintf(`
				%s

				resource "stackit_argus_instance" "instance" {
					project_id = "%s"
					name      = "%s"
					plan_name = "%s"
				}

				resource "stackit_argus_scrapeconfig" "scrapeconfig" {
					project_id = stackit_argus_instance.instance.project_id
					instance_id = stackit_argus_instance.instance.instance_id
				    name = "%s"
				    targets = [%s]
					scrape_interval = "%s"
					metrics_path = "%s"
				}
				`,
					testutil.ArgusProviderConfig(),
					instanceResource["project_id"],
					instanceResource["name"],
					instanceResource["plan_name"],
					scrapeConfigResource["name"],
					scrapeConfigResource["urls"],
					scrapeConfigResource["scrape_interval"],
					scrapeConfigResource["metrics_path"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "plan_name", instanceResource["plan_name"]),

					// Scrape Config
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "name", scrapeConfigResource["name"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "targets.#", "1"),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_path", scrapeConfigResource["metrics_path"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scheme", scrapeConfigResource["scheme"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scrape_interval", scrapeConfigResource["scrape_interval"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "saml2.%", "0"),
					resource.TestCheckNoResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "saml2.enable_url_parameters"),
				),
			},

			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckArgusDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *argus.APIClient
	var err error
	if testutil.ArgusCustomEndpoint == "" {
		client, err = argus.NewAPIClient()
	} else {
		client, err = argus.NewAPIClient(
			config.WithEndpoint(testutil.ArgusCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_argus_instance" {
			continue
		}
		// instance terraform ID: = "[project_id],[instance_id],[name]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.ListInstances(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	instances := *instancesResp.Instances
	for i := range instances {
		if utils.Contains(instancesToDestroy, *instances[i].Id) {
			if *instances[i].Status != wait.DeleteSuccess {
				_, err := client.DeleteInstanceExecute(ctx, testutil.ProjectId, *instances[i].Id)
				if err != nil {
					return fmt.Errorf("destroying instance %s during CheckDestroy: %w", *instances[i].Id, err)
				}
				_, err = wait.DeleteInstanceWaitHandler(ctx, client, testutil.ProjectId, *instances[i].Id).WaitWithContext(ctx)
				if err != nil {
					return fmt.Errorf("destroying instance %s during CheckDestroy: waiting for deletion %w", *instances[i].Id, err)
				}
			}
		}
	}
	return nil
}
