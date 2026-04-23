package cdn_test

import (
	"context"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/pem"
	"fmt"
	"maps"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn/v1api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"

	cdnSdk "github.com/stackitcloud/stackit-sdk-go/services/cdn/v1api"
)

var (
	bucketName             = "acc-b" + acctest.RandStringFromCharSet(3, acctest.CharSetAlpha)
	bucketNameUpdated      = "acc-b-updated" + acctest.RandStringFromCharSet(3, acctest.CharSetAlpha)
	credentialsName        = "acc-c" + acctest.RandStringFromCharSet(3, acctest.CharSetAlpha)
	credentialsNameUpdated = "acc-c-updated" + acctest.RandStringFromCharSet(3, acctest.CharSetAlpha)
	httpTestName           = "acc-h" + acctest.RandStringFromCharSet(3, acctest.CharSetAlpha)

	// FIX: Reverted to stackit.gg as used in the working old code to avoid reserved domain rejection
	dnsNameHttp       = fmt.Sprintf("tf-acc-%s.stackit.gg", strings.Split(uuid.NewString(), "-")[0])
	dnsRecordNameHttp = uuid.NewString()

	// Build the full domain name here so we can use it to sign the certificate
	fullDomainNameHttp = fmt.Sprintf("%s.%s", dnsRecordNameHttp, dnsNameHttp)

	cert, key = makeCertAndKey(testutil.OrganizationId, fullDomainNameHttp)
)

var (
	//go:embed testdata/resource-bucket.tf
	resourceBucket string

	//go:embed testdata/resource-http-base.tf
	resourceHttpBase string

	//go:embed testdata/resource-http-custom-domain.tf
	resourceHttpCustomDomain string
)

var resourceHttpFull = resourceHttpBase + "\n" + resourceHttpCustomDomain

var testConfigVarsBucket = config.Variables{
	"project_id":          config.StringVariable(testutil.ProjectId),
	"bucket_name":         config.StringVariable(bucketName),
	"credentials_name":    config.StringVariable(credentialsName),
	"backend_bucket_type": config.StringVariable("bucket"),
	"regions":             config.ListVariable(config.StringVariable("EU"), config.StringVariable("US")),
	"region":              config.StringVariable("eu01"),
	"optimizer":           config.BoolVariable(true),
}

func configVarsBucketUpdated() config.Variables {
	updatedConfig := maps.Clone(testConfigVarsBucket)
	updatedConfig["bucket_name"] = config.StringVariable(bucketNameUpdated)
	updatedConfig["credentials_name"] = config.StringVariable(credentialsNameUpdated)

	return updatedConfig
}

var testConfigVarsHttp = config.Variables{
	"project_id":                    config.StringVariable(testutil.ProjectId),
	"name":                          config.StringVariable(httpTestName),
	"regions":                       config.ListVariable(config.StringVariable("EU"), config.StringVariable("US")),
	"dns_zone_name":                 config.StringVariable("acc_cdn_test_zone"),
	"dns_name":                      config.StringVariable(dnsNameHttp),
	"dns_record_name":               config.StringVariable(dnsRecordNameHttp),
	"optimizer":                     config.BoolVariable(true),
	"backend_http_type":             config.StringVariable("http"),
	"blocked_countries":             config.ListVariable(config.StringVariable("CU")),
	"backend_origin_url":            config.StringVariable("https://test-backend-1.cdn-dev.runs.onstackit.cloud"),
	"geofencing_list":               config.ListVariable(config.StringVariable("DE")),
	"origin_request_headers_name":   config.StringVariable("X-Custom-Header"),
	"origin_request_headers_value":  config.StringVariable("x-custom-value"),
	"certificate":                   config.StringVariable(string(cert)),
	"private_key":                   config.StringVariable(string(key)),
	"redirect_target_url":           config.StringVariable("https://example.com"),
	"redirect_status_code":          config.IntegerVariable(301),
	"redirect_matcher_value":        config.StringVariable("/shop/*"),
	"redirect_rule_description":     config.StringVariable("Acc test redirect"),
	"redirect_rule_enabled":         config.BoolVariable(true),
	"redirect_rule_match_condition": config.StringVariable("ANY"),
	"redirect_matcher_condition":    config.StringVariable("ANY"),
}

func configVarsHttpUpdated() config.Variables {
	updatedConfig := maps.Clone(testConfigVarsHttp)
	updatedConfig["regions"] = config.ListVariable(config.StringVariable("EU"), config.StringVariable("US"), config.StringVariable("ASIA"))
	updatedConfig["redirect_target_url"] = config.StringVariable("https://example.com/updated")
	return updatedConfig
}

func makeCertAndKey(organization, domain string) (cert, key []byte) {
	privateKey, err := rsa.GenerateKey(cryptoRand.Reader, 2048)
	if err != nil {
		fmt.Printf("failed to generate key: %s", err.Error())
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Issuer:       pkix.Name{CommonName: organization},
		Subject: pkix.Name{
			Organization: []string{organization},
			CommonName:   domain, // Required by most modern TLS validations
		},
		DNSNames:              []string{domain},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
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
		fmt.Printf("failed to generate cert: %s", err.Error())
	}

	return pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert,
		}), pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})
}

func TestAccCDNDistributionHttp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCDNDistributionDestroy,
		Steps: []resource.TestStep{
			// Distribution Create (Only Base config)
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceHttpBase,
				ConfigVariables: testConfigVarsHttp,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "distribution_id"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "updated_at"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "domains.0.name"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.0.type", "managed"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.0.status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.#", "1"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.target_url", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_target_url"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.status_code", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_status_code"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.description", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_rule_description"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.enabled", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_rule_enabled"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.rule_match_condition", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_rule_match_condition"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.matchers.#", "1"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.matchers.0.values.0", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_matcher_value"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.matchers.0.value_match_condition", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_matcher_condition"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.#", "2"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.0", "EU"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.1", "US"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.#", "1"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.0", "CU"),
					resource.TestCheckResourceAttr(
						"stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.origin_request_headers.%s", testutil.ConvertConfigVariable(testConfigVarsHttp["origin_request_headers_name"])),
						testutil.ConvertConfigVariable(testConfigVarsHttp["origin_request_headers_value"]),
					),
					resource.TestCheckResourceAttr(
						"stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.geofencing.%s.0", testutil.ConvertConfigVariable(testConfigVarsHttp["backend_origin_url"])),
						"DE",
					),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.optimizer.enabled", testutil.ConvertConfigVariable(testConfigVarsHttp["optimizer"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "status", "ACTIVE"),
				),
			},
			// Wait step, confirms the CNAME record has "propagated" before trying to add the custom domain
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceHttpBase,
				ConfigVariables: testConfigVarsHttp,
				Check: func(_ *terraform.State) error {
					_, err := blockUntilDomainResolves(fullDomainNameHttp)
					return err
				},
			},
			// Custom Domain Create (Now using Full config)
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceHttpFull,
				ConfigVariables: testConfigVarsHttp,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "name", fullDomainNameHttp),
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "certificate.version", "1"),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "distribution_id", "stackit_cdn_custom_domain.custom_domain", "distribution_id"),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "project_id", "stackit_cdn_custom_domain.custom_domain", "project_id"),
				),
			},
			// Import
			{
				ResourceName:    "stackit_cdn_distribution.distribution",
				ConfigVariables: testConfigVarsHttp,
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
				ResourceName:    "stackit_cdn_custom_domain.custom_domain",
				ConfigVariables: testConfigVarsHttp,
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
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceHttpFull,
				ConfigVariables: testConfigVarsHttp,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.distribution", "distribution_id"),
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.distribution", "created_at"),
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.distribution", "updated_at"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.#", "2"),
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.distribution", "domains.0.name"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.1.name", fullDomainNameHttp),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.0.status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.1.status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.0.type", "managed"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "domains.1.type", "custom"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.regions.#", "2"),
					resource.TestCheckResourceAttr(
						"data.stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.origin_request_headers.%s", testutil.ConvertConfigVariable(testConfigVarsHttp["origin_request_headers_name"])),
						testutil.ConvertConfigVariable(testConfigVarsHttp["origin_request_headers_value"]),
					),
					resource.TestCheckResourceAttr(
						"data.stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.geofencing.%s.0", testutil.ConvertConfigVariable(testConfigVarsHttp["backend_origin_url"])),
						"DE",
					),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.regions.0", "EU"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.regions.1", "US"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.blocked_countries.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.blocked_countries.0", "CU"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.optimizer.enabled", testutil.ConvertConfigVariable(testConfigVarsHttp["optimizer"])),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.redirects.rules.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.redirects.rules.0.target_url", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_target_url"])),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.redirects.rules.0.status_code", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_status_code"])),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.redirects.rules.0.description", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_rule_description"])),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.redirects.rules.0.enabled", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_rule_enabled"])),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.redirects.rules.0.rule_match_condition", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_rule_match_condition"])),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.redirects.rules.0.matchers.0.values.0", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_matcher_value"])),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.distribution", "config.redirects.rules.0.matchers.0.value_match_condition", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_matcher_condition"])),

					resource.TestCheckResourceAttr("data.stackit_cdn_custom_domain.custom_domain", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_custom_domain.custom_domain", "name", fullDomainNameHttp),
					resource.TestCheckResourceAttr("data.stackit_cdn_custom_domain.custom_domain", "certificate.version", "1"),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "distribution_id", "stackit_cdn_custom_domain.custom_domain", "distribution_id"),
				),
			},
			// Update
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceHttpFull,
				ConfigVariables: configVarsHttpUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "distribution_id"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "updated_at"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.#", "2"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "domains.0.name"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.1.name", fullDomainNameHttp),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.0.status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.1.status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.0.type", "managed"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "domains.1.type", "custom"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.#", "3"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.0", "EU"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.1", "US"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.regions.2", "ASIA"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.#", "1"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.0", "CU"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.optimizer.enabled", testutil.ConvertConfigVariable(testConfigVarsHttp["optimizer"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "status", "ACTIVE"),
					resource.TestCheckResourceAttr(
						"stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.origin_request_headers.%s", testutil.ConvertConfigVariable(testConfigVarsHttp["origin_request_headers_name"])),
						testutil.ConvertConfigVariable(testConfigVarsHttp["origin_request_headers_value"]),
					),
					resource.TestCheckResourceAttr(
						"stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.geofencing.%s.0", testutil.ConvertConfigVariable(testConfigVarsHttp["backend_origin_url"])),
						"DE",
					),

					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.#", "1"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.target_url", testutil.ConvertConfigVariable(configVarsHttpUpdated()["redirect_target_url"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.redirects.rules.0.status_code", testutil.ConvertConfigVariable(testConfigVarsHttp["redirect_status_code"])),

					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "name", fullDomainNameHttp),
					resource.TestCheckResourceAttr("stackit_cdn_custom_domain.custom_domain", "certificate.version", "1"),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "distribution_id", "stackit_cdn_custom_domain.custom_domain", "distribution_id"),
					resource.TestCheckResourceAttrPair("stackit_cdn_distribution.distribution", "project_id", "stackit_cdn_custom_domain.custom_domain", "project_id"),
				),
			},
		},
	})
}

func TestAccCDNDistributionBucket(t *testing.T) {
	expectedBucketUrl := fmt.Sprintf("https://%s.object.storage.eu01.onstackit.cloud", bucketName)
	expectedBucketUrlUpdated := fmt.Sprintf("https://%s.object.storage.eu01.onstackit.cloud", bucketNameUpdated)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCDNDistributionDestroy,
		Steps: []resource.TestStep{
			// Distribution Create
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceBucket,
				ConfigVariables: testConfigVarsBucket,
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
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.optimizer.enabled", testutil.ConvertConfigVariable(testConfigVarsBucket["optimizer"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "status", "ACTIVE"),

					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.backend.type", testutil.ConvertConfigVariable(testConfigVarsBucket["backend_bucket_type"])),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.backend.bucket_url", expectedBucketUrl),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.backend.region", testutil.ConvertConfigVariable(testConfigVarsBucket["region"])),

					// CRITICAL: Verify that the CDN keys match the Object Storage keys
					// We use AttrPair because the values are generated dynamically on the server side
					resource.TestCheckResourceAttrPair(
						"stackit_cdn_distribution.distribution", "config.backend.credentials.access_key_id",
						"stackit_objectstorage_credential.creds", "access_key",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_cdn_distribution.distribution", "config.backend.credentials.secret_access_key",
						"stackit_objectstorage_credential.creds", "secret_access_key",
					),
				),
			},
			// Import
			{
				ResourceName:    "stackit_cdn_distribution.distribution",
				ConfigVariables: testConfigVarsBucket,
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
				ImportState:       true,
				ImportStateVerify: true,
				// We MUST ignore credentials on import verification
				// 1. API doesn't return them (security).
				// 2. State has them (from resource creation).
				ImportStateVerifyIgnore: []string{
					"config.backend.credentials"},
			},
			// Data Source
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceBucket,
				ConfigVariables: testConfigVarsBucket,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.bucket_ds", "distribution_id"),
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.bucket_ds", "created_at"),
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.bucket_ds", "updated_at"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "domains.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_cdn_distribution.bucket_ds", "domains.0.name"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "domains.0.status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "domains.0.type", "managed"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "config.regions.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "config.regions.0", "EU"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "config.regions.1", "US"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "config.optimizer.enabled", testutil.ConvertConfigVariable(testConfigVarsBucket["optimizer"])),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "config.backend.type", testutil.ConvertConfigVariable(testConfigVarsBucket["backend_bucket_type"])),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "config.backend.bucket_url", expectedBucketUrl),
					resource.TestCheckResourceAttr("data.stackit_cdn_distribution.bucket_ds", "config.backend.region", testutil.ConvertConfigVariable(testConfigVarsBucket["region"])),

					// Security Check: Secrets should NOT be in Data Source
					resource.TestCheckNoResourceAttr("data.stackit_cdn_distribution.bucket_ds", "config.backend.credentials.access_key_id"),
					resource.TestCheckNoResourceAttr("data.stackit_cdn_distribution.bucket_ds", "config.backend.credentials.secret_access_key"),
				),
			},
			// Update
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceBucket,
				ConfigVariables: configVarsBucketUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.backend.bucket_url", expectedBucketUrlUpdated),

					// Verify that keys have been updated to the new credentials
					resource.TestCheckResourceAttrPair(
						"stackit_cdn_distribution.distribution", "config.backend.credentials.access_key_id",
						"stackit_objectstorage_credential.creds", "access_key",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_cdn_distribution.distribution", "config.backend.credentials.secret_access_key",
						"stackit_objectstorage_credential.creds", "secret_access_key",
					),
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
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + resourceBucket,
				ConfigVariables: configVarsBucketUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.#", "0"),
				),
			},
		},
	})
}

func testAccCheckCDNDistributionDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := cdnSdk.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.CdnCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	distributionsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_cdn_distribution" {
			continue
		}
		distributionId := strings.Split(rs.Primary.ID, core.Separator)[1]
		distributionsToDestroy = append(distributionsToDestroy, distributionId)
	}

	for _, dist := range distributionsToDestroy {
		_, err := client.DefaultAPI.DeleteDistribution(ctx, testutil.ProjectId, dist).Execute()
		if err != nil {
			return fmt.Errorf("destroying CDN distribution %s during CheckDestroy: %w", dist, err)
		}
		_, err = wait.DeleteDistributionWaitHandler(ctx, client.DefaultAPI, testutil.ProjectId, dist).WaitWithContext(ctx)
		if err != nil {
			return fmt.Errorf("destroying CDN distribution %s during CheckDestroy: waiting for deletion %w", dist, err)
		}
	}
	return nil
}

const (
	recordCheckInterval time.Duration = 3 * time.Second
	recordCheckAttempts               = 100 // wait up to 5 minutes for record to become available (normally takes less than 2 minutes)
)

func blockUntilDomainResolves(domain string) (net.IP, error) {
	// Create a custom resolver that bypasses the local system DNS settings/cache
	// and queries Google DNS (8.8.8.8) directly.
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			// Force query to Google DNS
			return d.DialContext(ctx, network, "8.8.8.8:53")
		},
	}

	// wait until it becomes ready
	isReady := func() (net.IP, error) {
		// Use a context for the individual query timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		ips, err := r.LookupIP(ctx, "ip", domain)
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
	for range attempts {
		dist, err := f()
		if err == nil {
			return dist, nil
		}
		errOuter = err
		time.Sleep(sleep)
	}
	return zero, fmt.Errorf("retry timed out, last error: %w", errOuter)
}
