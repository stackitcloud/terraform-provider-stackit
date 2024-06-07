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
	"project_id":    testutil.ProjectId,
	"name":          testutil.ResourceNameWithDateTime("argus"),
	"plan_name":     "Monitoring-Basic-EU01",
	"new_plan_name": "Monitoring-Medium-EU01",
	"acl-0":         "1.2.3.4/32",
	"acl-1":         "111.222.111.222/32",
	"acl-1-updated": "111.222.111.125/32",
}

var scrapeConfigResource = map[string]string{
	"project_id":                           testutil.ProjectId,
	"name":                                 fmt.Sprintf("scrapeconfig-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum)),
	"urls":                                 fmt.Sprintf(`{urls = ["www.%s.de","%s.de"]}`, acctest.RandStringFromCharSet(15, acctest.CharSetAlphaNum), acctest.RandStringFromCharSet(15, acctest.CharSetAlphaNum)),
	"metrics_path":                         "/metrics",
	"scheme":                               "https",
	"scrape_interval":                      "4m", // non-default
	"sample_limit":                         "7",  // non-default
	"saml2_enable_url_parameters":          "false",
	"honor_labels":                         "false",
	"honor_timestamps":                     "false",
	"httpsdconfigs_refresh_interval":       "60s",
	"httpsdconfigs_tls_insecureskipverify": "false",
	"httpsdconfigs_url":                    fmt.Sprintf(`"http://%s.de"`, acctest.RandStringFromCharSet(15, acctest.CharSetAlphaNum)),
	"httpsdconfigs_oauth2_clientid":        "client",
	"httpsdconfigs_oauth2_secret":          "secret",
	"httpsdconfigs_oauth2_tokenurl":        fmt.Sprintf(`"http://%s.de"`, acctest.RandStringFromCharSet(15, acctest.CharSetAlphaNum)),
	"httpsdconfigs_oauth2_scopes":          "scope",
	"httpsdconfigs_oauth2_tls_isv":         "false",
	"metricsRC_action":                     "replace",
	"metricsRC_modulus":                    "2",
	"metricsRC_regex":                      ".*",
	"metricsRC_replacement":                "$1",
	"metricsRC_separator":                  ";",
	"metricsRC_targetLabel":                "target",
	"metricsRC_sourcelabels":               "source",
	"tlsconfig_insecureskipverify":         "false",
}

var credentialResource = map[string]string{
	"project_id": testutil.ProjectId,
}

func resourceConfig(acl *string, instanceName, planName, target, saml2EnableUrlParameters string) string {
	if acl == nil {
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
					sample_limit = %s
					saml2 = { 
						enable_url_parameters = %s
					}
					honor_labels = %s
					honor_timestamps = %s
					http_sd_configs = [{
						refresh_interval = "%s"
						tls_config = {
							insecure_skip_verify = %s
						}
						url = %s
						oauth2 = {
							client_id = "%s"
							client_secret = "%s"
							token_url = %s
							scopes = ["%s"]
							tls_config = {
								insecure_skip_verify = %s
							}							
						}
					}]
					metrics_relabel_configs = [{
						action = "%s"
						modulus = "%s"
						regex = "%s"
						replacement = "%s"
						separator = "%s"
						target_label = "%s"
						source_labels = ["%s"]
					}]
					tls_config = {
						insecure_skip_verify = %s		
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
			planName,
			scrapeConfigResource["name"],
			scrapeConfigResource["metrics_path"],
			target,
			scrapeConfigResource["scrape_interval"],
			scrapeConfigResource["sample_limit"],
			saml2EnableUrlParameters,
			scrapeConfigResource["honor_labels"],
			scrapeConfigResource["honor_timestamps"],
			scrapeConfigResource["httpsdconfigs_refresh_interval"],
			scrapeConfigResource["httpsdconfigs_tls_insecureskipverify"],
			scrapeConfigResource["httpsdconfigs_url"],
			scrapeConfigResource["httpsdconfigs_oauth2_clientid"],
			scrapeConfigResource["httpsdconfigs_oauth2_secret"],
			scrapeConfigResource["httpsdconfigs_oauth2_tokenurl"],
			scrapeConfigResource["httpsdconfigs_oauth2_scopes"],
			scrapeConfigResource["httpsdconfigs_oauth2_tls_isv"],
			scrapeConfigResource["metricsRC_action"],
			scrapeConfigResource["metricsRC_modulus"],
			scrapeConfigResource["metricsRC_regex"],
			scrapeConfigResource["metricsRC_replacement"],
			scrapeConfigResource["metricsRC_separator"],
			scrapeConfigResource["metricsRC_targetLabel"],
			scrapeConfigResource["metricsRC_sourcelabels"],
			scrapeConfigResource["tlsconfig_insecureskipverify"],
		)
	}
	return fmt.Sprintf(`
	%s

	resource "stackit_argus_instance" "instance" {
		project_id = "%s"
		name      = "%s"
		plan_name = "%s"
		acl       = %s
	}
	
	resource "stackit_argus_scrapeconfig" "scrapeconfig" {
		project_id = stackit_argus_instance.instance.project_id
		instance_id = stackit_argus_instance.instance.instance_id
		name = "%s"
		metrics_path = "%s"
		targets = [%s]
		scrape_interval = "%s"
		sample_limit = %s
		saml2 = { 
			enable_url_parameters = %s
		}
		honor_labels = %s
		honor_timestamps = %s
		http_sd_configs = [{
			refresh_interval = "%s"
			tls_config = {
				insecure_skip_verify = %s
			}
			url = %s
			oauth2 = {
				client_id = "%s"
				client_secret = "%s"
				token_url = %s
				scopes = ["%s"]
				tls_config = {
					insecure_skip_verify = %s
				}							
			}
		}]
		metrics_relabel_configs = [{
			action = "%s"
			modulus = "%s"
			regex = "%s"
			replacement = "%s"
			separator = "%s"
			target_label = "%s"
			source_labels = ["%s"]
		}]
		tls_config = {
			insecure_skip_verify = %s		
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
		planName,
		*acl,
		scrapeConfigResource["name"],
		scrapeConfigResource["metrics_path"],
		target,
		scrapeConfigResource["scrape_interval"],
		scrapeConfigResource["sample_limit"],
		saml2EnableUrlParameters,
		scrapeConfigResource["honor_labels"],
		scrapeConfigResource["honor_timestamps"],
		scrapeConfigResource["httpsdconfigs_refresh_interval"],
		scrapeConfigResource["httpsdconfigs_tls_insecureskipverify"],
		scrapeConfigResource["httpsdconfigs_url"],
		scrapeConfigResource["httpsdconfigs_oauth2_clientid"],
		scrapeConfigResource["httpsdconfigs_oauth2_secret"],
		scrapeConfigResource["httpsdconfigs_oauth2_tokenurl"],
		scrapeConfigResource["httpsdconfigs_oauth2_scopes"],
		scrapeConfigResource["httpsdconfigs_oauth2_tls_isv"],
		scrapeConfigResource["metricsRC_action"],
		scrapeConfigResource["metricsRC_modulus"],
		scrapeConfigResource["metricsRC_regex"],
		scrapeConfigResource["metricsRC_replacement"],
		scrapeConfigResource["metricsRC_separator"],
		scrapeConfigResource["metricsRC_targetLabel"],
		scrapeConfigResource["metricsRC_sourcelabels"],
		scrapeConfigResource["tlsconfig_insecureskipverify"],
	)
}

func TestAccResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckArgusDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: resourceConfig(
					utils.Ptr(fmt.Sprintf(
						"[%q, %q, %q]",
						instanceResource["acl-0"],
						instanceResource["acl-1"],
						instanceResource["acl-1"],
					)),
					instanceResource["name"],
					instanceResource["plan_name"],
					scrapeConfigResource["urls"],
					scrapeConfigResource["saml2_enable_url_parameters"],
				),
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

					// ACL
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "acl.0", instanceResource["acl-0"]),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "acl.1", instanceResource["acl-1"]),

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
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "sample_limit", scrapeConfigResource["sample_limit"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "saml2.enable_url_parameters", scrapeConfigResource["saml2_enable_url_parameters"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "honor_labels", scrapeConfigResource["honor_labels"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "honor_timestamps", scrapeConfigResource["honor_timestamps"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "http_sd_configs.url", scrapeConfigResource["httpsdconfigs_url"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "http_sd_configs.refresh_interval", scrapeConfigResource["httpsdconfigs_refresh_interval"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "http_sd_configs.tls.insecure_skip_verify", scrapeConfigResource["httpsdconfigs_tls_insecureskipverify"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "http_sd_configs.oauth2.url", scrapeConfigResource["httpsdconfigs_url"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "http_sd_configs.oauth2.client_id", scrapeConfigResource["httpsdconfigs_oauth2_clientid"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "http_sd_configs.oauth2.client_secret", scrapeConfigResource["httpsdconfigs_oauth2_secret"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "http_sd_configs.oauth2.token_url", scrapeConfigResource["httpsdconfigs_oauth2_tokenurl"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "http_sd_configs.oauth2.scopes", scrapeConfigResource["httpsdconfigs_oauth2_scopes"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "http_sd_configs.tls_config.insecure_skip_verify", scrapeConfigResource["httpsdconfigs_oauth2_tls_isv"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_relabel_configs.action", scrapeConfigResource["metricsRC_action"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_relabel_configs.modulus", scrapeConfigResource["metricsRC_modulus"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_relabel_configs.regex", scrapeConfigResource["metricsRC_regex"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_relabel_configs.replacement", scrapeConfigResource["metricsRC_replacement"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_relabel_configs.separator", scrapeConfigResource["metricsRC_separator"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_relabel_configs.target_label", scrapeConfigResource["metricsRC_targetLabel"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_relabel_configs.source_labels", scrapeConfigResource["metricsRC_sourcelabels"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "tls_config.insecure_skip_verify", scrapeConfigResource["tlsconfig_insecureskipverify"]),

					// credentials
					resource.TestCheckResourceAttr("stackit_argus_credential.credential", "project_id", credentialResource["project_id"]),
					resource.TestCheckResourceAttrPair(
						"stackit_argus_instance.instance", "instance_id",
						"stackit_argus_credential.credential", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_argus_credential.credential", "username"),
					resource.TestCheckResourceAttrSet("stackit_argus_credential.credential", "password"),
				),
			},
			// Creation without ACL
			{
				Config: resourceConfig(
					nil,
					instanceResource["name"],
					instanceResource["plan_name"],
					scrapeConfigResource["urls"],
					scrapeConfigResource["saml2_enable_url_parameters"],
				),
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

					// ACL
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "acl.#", "0"),

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
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "sample_limit", scrapeConfigResource["sample_limit"]),
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
			},
			// Creation with empty ACL
			{
				Config: resourceConfig(
					utils.Ptr("[]"),
					instanceResource["name"],
					instanceResource["plan_name"],
					scrapeConfigResource["urls"],
					scrapeConfigResource["saml2_enable_url_parameters"],
				),
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

					// ACL
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "acl.#", "0"),

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
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "sample_limit", scrapeConfigResource["sample_limit"]),
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
			},
			{
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
					resourceConfig(utils.Ptr(fmt.Sprintf(
						"[%q, %q]",
						instanceResource["acl-0"],
						instanceResource["acl-1"],
					)),
						instanceResource["name"],
						instanceResource["plan_name"],
						scrapeConfigResource["urls"],
						scrapeConfigResource["saml2_enable_url_parameters"],
					),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_argus_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("data.stackit_argus_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_argus_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("data.stackit_argus_instance.instance", "plan_name", instanceResource["plan_name"]),
					resource.TestCheckResourceAttr("data.stackit_argus_instance.instance", "acl.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_argus_instance.instance", "acl.0", instanceResource["acl-0"]),
					resource.TestCheckResourceAttr("data.stackit_argus_instance.instance", "acl.1", instanceResource["acl-1"]),
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
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "sample_limit", scrapeConfigResource["sample_limit"]),
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
				Config: resourceConfig(utils.Ptr(fmt.Sprintf(
					"[%q, %q]",
					instanceResource["acl-0"],
					instanceResource["acl-1-updated"],
				)),
					fmt.Sprintf("%s-new", instanceResource["name"]),
					instanceResource["new_plan_name"],
					"",
					"true",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "name", instanceResource["name"]+"-new"),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "plan_name", instanceResource["new_plan_name"]),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "acl.#", "2"),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "acl.0", instanceResource["acl-0"]),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "acl.1", instanceResource["acl-1-updated"]),

					// Scrape Config
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "name", scrapeConfigResource["name"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "targets.#", "0"),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_path", scrapeConfigResource["metrics_path"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scheme", scrapeConfigResource["scheme"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scrape_interval", scrapeConfigResource["scrape_interval"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "sample_limit", scrapeConfigResource["sample_limit"]),
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
					sample_limit = %s
					metrics_path = "%s"
					saml2 = {
						enable_url_parameters = false
					}
				}
				`,
					testutil.ArgusProviderConfig(),
					instanceResource["project_id"],
					instanceResource["name"],
					instanceResource["new_plan_name"],
					scrapeConfigResource["name"],
					scrapeConfigResource["urls"],
					scrapeConfigResource["scrape_interval"],
					scrapeConfigResource["sample_limit"],
					scrapeConfigResource["metrics_path"],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "project_id", instanceResource["project_id"]),
					resource.TestCheckResourceAttrSet("stackit_argus_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "name", instanceResource["name"]),
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "plan_name", instanceResource["new_plan_name"]),

					// ACL
					resource.TestCheckResourceAttr("stackit_argus_instance.instance", "acl.#", "0"),

					// Scrape Config
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "name", scrapeConfigResource["name"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "targets.#", "1"),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "metrics_path", scrapeConfigResource["metrics_path"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scheme", scrapeConfigResource["scheme"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "scrape_interval", scrapeConfigResource["scrape_interval"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "sample_limit", scrapeConfigResource["sample_limit"]),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "saml2.%", "1"),
					resource.TestCheckResourceAttr("stackit_argus_scrapeconfig.scrapeconfig", "saml2.enable_url_parameters", "false"),
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
		client, err = argus.NewAPIClient(
			config.WithRegion("eu01"),
		)
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
