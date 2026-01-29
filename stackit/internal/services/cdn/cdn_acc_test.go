package cdn_test

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// 1. Define the Configuration Struct
type distributionConfig struct {
	ProjectId          string
	BackendType        string
	OriginURL          string
	Geofencing         map[string][]string
	Regions            []string
	BlockedCountries   []string
	OptimizerEnabled   bool
	DNSName            string
	CustomDomainPrefix string
	Cert               string
	Key                string
}

// 2. Define the Template
const distributionTmpl = `
{{ .ProviderConfig }}

resource "stackit_cdn_distribution" "distribution" {
    project_id = "{{ .ProjectId }}"
    config = {
        backend = {
            type       = "{{ .BackendType }}"
            origin_url = "{{ .OriginURL }}"
            {{- if .Geofencing }}
            geofencing = {
                {{- range $url, $countries := .Geofencing }}
                "{{ $url }}" = {{ $countries | stringList }}
                {{- end }}
            }
            {{- end }}
        }
        regions = {{ .Regions | stringList }}

        {{- if .BlockedCountries }}
        blocked_countries = {{ .BlockedCountries | stringList }}
        {{- end }}

        optimizer = {
            enabled = {{ .OptimizerEnabled }}
        }
    }
}

resource "stackit_dns_zone" "dns_zone" {
    project_id    = "{{ .ProjectId }}"
    name          = "cdn_acc_test_zone"
    dns_name      = "{{ .DNSName }}"
    contact_email = "aa@bb.cc"
    type          = "primary"
    default_ttl   = 3600
}

resource "stackit_dns_record_set" "dns_record" {
    project_id = "{{ .ProjectId }}"
    zone_id    = stackit_dns_zone.dns_zone.zone_id
    name       = "{{ .CustomDomainPrefix }}"
    type       = "CNAME"
    records    = ["${stackit_cdn_distribution.distribution.domains[0].name}."]
}
`

const customDomainTmpl = `
resource "stackit_cdn_custom_domain" "custom_domain" {
    project_id      = stackit_cdn_distribution.distribution.project_id
    distribution_id = stackit_cdn_distribution.distribution.distribution_id
    name            = "${stackit_dns_record_set.dns_record.name}.${stackit_dns_zone.dns_zone.dns_name}"
    certificate = {
        certificate = {{ .Cert | printf "%q" }}
        private_key = {{ .Key | printf "%q" }}
    }
}
`

// 3. Implement the Renderer
func renderConfig(conf distributionConfig, includeCustomDomain bool) string {
	// Wrapper struct to include global provider config for the template
	type templateData struct {
		distributionConfig
		ProviderConfig string
	}

	data := templateData{
		distributionConfig: conf,
		ProviderConfig:     testutil.CdnProviderConfig(),
	}

	// Helper to format go slices into HCL lists
	funcMap := template.FuncMap{
		"stringList": func(s []string) string {
			if len(s) == 0 {
				return "[]"
			}
			var quoted []string
			for _, item := range s {
				quoted = append(quoted, fmt.Sprintf("%q", item))
			}
			return fmt.Sprintf("[%s]", strings.Join(quoted, ", "))
		},
	}

	// Parse distribution template
	tmpl, err := template.New("distribution").Funcs(funcMap).Parse(distributionTmpl)
	if err != nil {
		panic(fmt.Errorf("failed to parse distribution template: %w", err))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Errorf("failed to render distribution template: %w", err))
	}

	// Parse custom domain template if needed
	if includeCustomDomain {
		cdTmpl, err := template.New("customDomain").Funcs(funcMap).Parse(customDomainTmpl)
		if err != nil {
			panic(fmt.Errorf("failed to parse custom domain template: %w", err))
		}
		if err := cdTmpl.Execute(&buf, data); err != nil {
			panic(fmt.Errorf("failed to render custom domain template: %w", err))
		}
	}

	return buf.String()
}

func configDatasources(conf distributionConfig) string {
	baseConfig := renderConfig(conf, true) // Includes custom domain

	return fmt.Sprintf(`
        %s 

        data "stackit_cdn_distribution" "distribution" {
            project_id      = stackit_cdn_distribution.distribution.project_id
            distribution_id = stackit_cdn_distribution.distribution.distribution_id
        }
        
        data "stackit_cdn_custom_domain" "custom_domain" {
            project_id      = stackit_cdn_custom_domain.custom_domain.project_id
            distribution_id = stackit_cdn_custom_domain.custom_domain.distribution_id
            name            = stackit_cdn_custom_domain.custom_domain.name
        }
        `, baseConfig)
}
func makeCertAndKey(t *testing.T, organization string) (cert, key []byte) {
	privateKey, err := rsa.GenerateKey(cryptoRand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %s", err.Error())
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Issuer:       pkix.Name{CommonName: organization},
		Subject: pkix.Name{
			Organization: []string{organization},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	cert, err = x509.CreateCertificate(
		cryptoRand.Reader,
		&template,
		&template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		t.Fatalf("failed to generate cert: %s", err.Error())
	}

	return pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert,
		}), pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})
}
func TestAccCDNDistributionResource(t *testing.T) {
	// we use a different domain prefix each test run due to inconsistent upstream release of domains, which might impair consecutive test runs
	customDomainPrefix := uuid.NewString()
	dnsName := fmt.Sprintf("tf-acc-%s.stackit.gg", strings.Split(uuid.NewString(), "-")[0])
	geofencedOriginURL := "https://test-backend-2.cdn-dev.runs.onstackit.cloud"
	fullDomainName := fmt.Sprintf("%s.%s", customDomainPrefix, dnsName)
	organization := fmt.Sprintf("organization-%s", uuid.NewString())
	cert, key := makeCertAndKey(t, organization)

	// Setup Base Configuration
	baseConf := distributionConfig{
		ProjectId:          testutil.ProjectId,
		BackendType:        "http",
		OriginURL:          "https://test-backend-1.cdn-dev.runs.onstackit.cloud",
		DNSName:            dnsName,
		CustomDomainPrefix: customDomainPrefix,
		OptimizerEnabled:   true,
		Regions:            []string{"EU", "US"},
		BlockedCountries:   []string{"CU", "AQ"}, // Do NOT use DE or AT here, because the request might be blocked by bunny at the time of creation - don't lock yourself out
		Geofencing: map[string][]string{
			geofencedOriginURL: {"DE", "ES"},
		},
	}

	// Prepare updated config
	organization_updated := fmt.Sprintf("organization-updated-%s", uuid.NewString())
	cert_updated, key_updated := makeCertAndKey(t, organization_updated)

	updatedConf := baseConf
	updatedConf.Regions = []string{"EU", "US", "ASIA"}
	updatedConf.Cert = string(cert_updated)
	updatedConf.Key = string(key_updated)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCDNDistributionDestroy,
		Steps: []resource.TestStep{
			// Distribution Create
			{
				Config: renderConfig(baseConf, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "distribution_id"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "updated_at"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "domains.0.name"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.0.type", "managed"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.0.status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.#", "2"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.0", "EU"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.1", "US"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.#", "2"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.0", "CU"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.1", "AQ"),
					resource.TestCheckResourceAttr(
						"stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.geofencing.%s.0", geofencedOriginURL),
						"DE",
					),
					resource.TestCheckResourceAttr(
						"stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.geofencing.%s.1", geofencedOriginURL),
						"ES",
					),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.optimizer.enabled", "true"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "status", "ACTIVE"),
				),
			},
			// Wait step, that confirms the CNAME record has "propagated"
			{
				Config: renderConfig(baseConf, false),
				Check: func(_ *terraform.State) error {
					_, err := blockUntilDomainResolves(fullDomainName)
					return err
				},
			},
			// Custom Domain Create
			{
				Config: func() string {
					c := baseConf
					c.Cert = string(cert)
					c.Key = string(key)
					return renderConfig(c, true)
				}(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "name", fullDomainName),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "distribution_id", "stackit_cdn_custom_domain.custom_domain", "distribution_id"),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "project_id", "stackit_cdn_custom_domain.custom_domain", "project_id"),
				),
			},
			// Import
			{
				ResourceName: "stackit_cdn_distribution.distribution",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_cdn_distribution.distribution"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_cdn_distribution.distribution")
					}
					distributionId, ok := r.Primary.Attributes["distribution_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute distribution_id")
					}

					return fmt.Sprintf("%s,%s", testutil.ProjectId, distributionId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"domains"}, // we added a domain in the meantime...
			},
			{
				ResourceName: "stackit_cdn_custom_domain.custom_domain",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_cdn_custom_domain.custom_domain"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_cdn_custom_domain.custom_domain")
					}
					distributionId, ok := r.Primary.Attributes["distribution_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute distribution_id")
					}
					name, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, distributionId, name), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"certificate.certificate",
					"certificate.private_key",
				},
			},
			// Data Source
			{
				Config: func() string {
					c := baseConf
					c.Cert = string(cert)
					c.Key = string(key)
					return configDatasources(c)
				}(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.distribution", "distribution_id"),
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.distribution", "created_at"),
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.distribution", "updated_at"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.#", "2"),
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.distribution", "domains.0.name"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.1.name", fullDomainName),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.0.status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.1.status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.0.type", "managed"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.1.type", "custom"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.regions.#", "2"),
					resource.TestCheckResourceAttr(
						"data.stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.geofencing.%s.0", geofencedOriginURL),
						"DE",
					),
					resource.TestCheckResourceAttr(
						"data.stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.geofencing.%s.1", geofencedOriginURL),
						"ES",
					),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.regions.0", "EU"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.regions.1", "US"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.#", "2"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.0", "CU"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.1", "AQ"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.optimizer.enabled", "true"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_custom_domain.custom_domain", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_custom_domain.custom_domain", "certificate.version", "1"),
					resource.TestCheckResourceAttr("data.stackit_cdn_custom_domain.custom_domain", "name", fullDomainName),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "distribution_id", "stackit_cdn_custom_domain.custom_domain", "distribution_id"),
				),
			},
			// Update
			{
				Config: renderConfig(updatedConf, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "distribution_id"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "updated_at"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.#", "2"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "domains.0.name"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.1.name", fullDomainName),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.0.status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.1.status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.0.type", "managed"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.1.type", "custom"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.#", "3"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.0", "EU"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.1", "US"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.2", "ASIA"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.#", "2"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.0", "CU"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.1", "AQ"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.optimizer.enabled", "true"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "certificate.version", "2"),
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "name", fullDomainName),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "distribution_id", "stackit_cdn_custom_domain.custom_domain", "distribution_id"),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "project_id", "stackit_cdn_custom_domain.custom_domain", "project_id"),
				),
			},
			// Bug Fix Verification: Omitted Field Handling
			//
			// This step verifies that omitting 'blocked_countries' from the Terraform configuration
			// (by setting the pointer to nil) does not cause an "inconsistent result" error.
			//
			// Previously, omitting the field resulted in a 'null' config, but the API returned an
			// empty list '[]', causing a state mismatch. The 'Default' modifier in the schema now
			// ensures the missing config is treated as an empty list, matching the API response.
			{
				Config: func() string {
					c := baseConf
					c.BlockedCountries = nil // Empty list means it won't render
					return renderConfig(c, false)
				}(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.#", "0"),
				),
			},
		},
	})
}
func testAccCheckCDNDistributionDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *cdn.APIClient
	var err error
	if testutil.MongoDBFlexCustomEndpoint == "" {
		client, err = cdn.NewAPIClient()
	} else {
		client, err = cdn.NewAPIClient(
			config.WithEndpoint(testutil.MongoDBFlexCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	distributionsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_mongodbflex_instance" {
			continue
		}
		distributionId := strings.Split(rs.Primary.ID, core.Separator)[1]
		distributionsToDestroy = append(distributionsToDestroy, distributionId)
	}

	for _, dist := range distributionsToDestroy {
		_, err := client.DeleteDistribution(ctx, testutil.ProjectId, dist).Execute()
		if err != nil {
			return fmt.Errorf("destroying CDN distribution %s during CheckDestroy: %w", dist, err)
		}
		_, err = wait.DeleteDistributionWaitHandler(ctx, client, testutil.ProjectId, dist).WaitWithContext(ctx)
		if err != nil {
			return fmt.Errorf("destroying CDN distribution %s during CheckDestroy: waiting for deletion %w", dist, err)
		}
	}
	return nil
}

const (
	recordCheckInterval time.Duration = 3 * time.Second
	recordCheckAttempts               = 100 // wait up to 5 minutes for record to be come available (normally takes less than 2 minutes)
)

func blockUntilDomainResolves(domain string) (net.IP, error) {
	// wait until it becomes ready
	isReady := func() (net.IP, error) {
		ips, err := net.LookupIP(domain)
		if err != nil {
			return nil, fmt.Errorf("error looking up IP for domain %s: %w", domain, err)
		}
		for _, ip := range ips {
			if ip.String() != "<nil>" {
				return ip, nil
			}
		}
		return nil, fmt.Errorf("no IP for domain: %v", domain)
	}
	return retry(recordCheckAttempts, recordCheckInterval, isReady)
}

func retry[T any](attempts int, sleep time.Duration, f func() (T, error)) (T, error) {
	var zero T
	var errOuter error
	for i := 0; i < attempts; i++ {
		dist, err := f()
		if err == nil {
			return dist, nil
		}
		errOuter = err
		time.Sleep(sleep)
	}
	return zero, fmt.Errorf("retry timed out, last error: %w", errOuter)
}
