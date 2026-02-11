package cdn_test

import (
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

var instanceResource = map[string]string{
	"project_id":                testutil.ProjectId,
	"config_backend_type":       "http",
	"config_backend_origin_url": "https://test-backend-1.cdn-dev.runs.onstackit.cloud",
	"config_regions":            "\"EU\", \"US\"",
	"config_regions_updated":    "\"EU\", \"US\", \"ASIA\"",
	"blocked_countries":         "\"CU\", \"AQ\"", // Do NOT use DE or AT here, because the request might be blocked by bunny at the time of creation - don't lock yourself out
	"custom_domain_prefix":      uuid.NewString(), // we use a different domain prefix each test run due to inconsistent upstream release of domains, which might impair consecutive test runs
	"dns_name":                  fmt.Sprintf("tf-acc-%s.stackit.gg", strings.Split(uuid.NewString(), "-")[0]),
}

func configResources(regions string, geofencingCountries []string, blockedCountries *string) string {
	var quotedCountries []string
	for _, country := range geofencingCountries {
		quotedCountries = append(quotedCountries, fmt.Sprintf(`%q`, country))
	}

	geofencingList := strings.Join(quotedCountries, ",")

	blockedCountriesConfig := ""
	if blockedCountries != nil {
		blockedCountriesConfig = fmt.Sprintf("blocked_countries = [%s]", *blockedCountries)
	}

	return fmt.Sprintf(`
				%s

				resource "stackit_cdn_distribution" "distribution" {
					project_id = "%s"
		            config = {
						backend = {
							type = "http"
							origin_url = "%s"
							geofencing = {
								"%s" = [%s]  
							}
						}
						regions           = [%s]
						%s
            
						optimizer = {
							enabled = true
						}
					}
				}

				resource "stackit_dns_zone" "dns_zone" {
					project_id    = "%s"
					name          = "cdn_acc_test_zone"
					dns_name      = "%s"
					contact_email = "aa@bb.cc"
					type          = "primary"
					default_ttl   = 3600
				}
				resource "stackit_dns_record_set" "dns_record" {
					project_id = "%s"
					zone_id    = stackit_dns_zone.dns_zone.zone_id
					name       = "%s"
					type       = "CNAME"
					records    = ["${stackit_cdn_distribution.distribution.domains[0].name}."]
				}
		`, testutil.CdnProviderConfig(), testutil.ProjectId, instanceResource["config_backend_origin_url"], instanceResource["config_backend_origin_url"], geofencingList,
		regions, blockedCountriesConfig, testutil.ProjectId, instanceResource["dns_name"],
		testutil.ProjectId, instanceResource["custom_domain_prefix"])
}

func configCustomDomainResources(regions, cert, key string, geofencingCountries []string, blockedCountries *string) string {
	return fmt.Sprintf(`
				%s

		        resource "stackit_cdn_custom_domain" "custom_domain" {
					project_id = stackit_cdn_distribution.distribution.project_id
					distribution_id = stackit_cdn_distribution.distribution.distribution_id
		            name = "${stackit_dns_record_set.dns_record.name}.${stackit_dns_zone.dns_zone.dns_name}"
					certificate = {
						certificate = %q
						private_key = %q
					}
				}
`, configResources(regions, geofencingCountries, blockedCountries), cert, key)
}

func configDatasources(regions, cert, key string, geofencingCountries []string, blockedCountries *string) string {
	return fmt.Sprintf(`
        %s 

        data "stackit_cdn_distribution" "distribution" {
					project_id = stackit_cdn_distribution.distribution.project_id
            distribution_id = stackit_cdn_distribution.distribution.distribution_id
        }
        
        data "stackit_cdn_custom_domain" "custom_domain" {
					project_id = stackit_cdn_custom_domain.custom_domain.project_id
					distribution_id = stackit_cdn_custom_domain.custom_domain.distribution_id
					name = stackit_cdn_custom_domain.custom_domain.name

        }
		`, configCustomDomainResources(regions, cert, key, geofencingCountries, blockedCountries))
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
	fullDomainName := fmt.Sprintf("%s.%s", instanceResource["custom_domain_prefix"], instanceResource["dns_name"])
	organization := fmt.Sprintf("organization-%s", uuid.NewString())
	cert, key := makeCertAndKey(t, organization)
	geofencing := []string{"DE", "ES"}

	organization_updated := fmt.Sprintf("organization-updated-%s", uuid.NewString())
	cert_updated, key_updated := makeCertAndKey(t, organization_updated)

	// Helper for default blocked countries
	defaultBlockedCountries := cdn.PtrString(instanceResource["blocked_countries"])

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCDNDistributionDestroy,
		Steps: []resource.TestStep{
			// Distribution Create
			{
				Config: configResources(instanceResource["config_regions"], geofencing, defaultBlockedCountries),
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
						fmt.Sprintf("config.backend.geofencing.%s.0", instanceResource["config_backend_origin_url"]),
						"DE",
					),
					resource.TestCheckResourceAttr(
						"stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.geofencing.%s.1", instanceResource["config_backend_origin_url"]),
						"ES",
					),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.optimizer.enabled", "true"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "status", "ACTIVE"),
				),
			},
			// Wait step, that confirms the CNAME record has "propagated"
			{
				Config: configResources(instanceResource["config_regions"], geofencing, defaultBlockedCountries),
				Check: func(_ *terraform.State) error {
					_, err := blockUntilDomainResolves(fullDomainName)
					return err
				},
			},
			// Custom Domain Create
			{
				Config: configCustomDomainResources(instanceResource["config_regions"], string(cert), string(key), geofencing, defaultBlockedCountries),
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
				Config: configDatasources(instanceResource["config_regions"], string(cert), string(key), geofencing, defaultBlockedCountries),
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
						fmt.Sprintf("config.backend.geofencing.%s.0", instanceResource["config_backend_origin_url"]),
						"DE",
					),
					resource.TestCheckResourceAttr(
						"data.stackit_cdn_distribution.distribution",
						fmt.Sprintf("config.backend.geofencing.%s.1", instanceResource["config_backend_origin_url"]),
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
				Config: configCustomDomainResources(instanceResource["config_regions_updated"], string(cert_updated), string(key_updated), geofencing, defaultBlockedCountries),
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
				Config: configResources(instanceResource["config_regions"], geofencing, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.blocked_countries.#", "0"),
				),
			},
		},
	})
}
func configBucketResources(bucketUrl, region, accessKey, secretKey string) string {
	return fmt.Sprintf(`
                %s

                resource "stackit_cdn_distribution" "distribution" {
                    project_id = "%s"
                    config = {
                        backend = {
                            type       = "bucket"
                            bucket_url = "%s"
                            region     = "%s"
                            access_key_id = "%s"
                            secret_key = "%s"
                        }
                        regions           = ["EU", "US"]
                        blocked_countries = ["CN", "RU"] 
            
                        optimizer = {
                            enabled = false
                        }
                    }
                }
        `, testutil.CdnProviderConfig(), testutil.ProjectId, bucketUrl, region, accessKey, secretKey)
}
func randomString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes := make([]byte, n)
	if _, err := cryptoRand.Read(bytes); err != nil {
		panic("failed to generate random string: " + err.Error())
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes)
}
func TestAccCDNDistributionBucketResource(t *testing.T) {
	// Define dummy bucket data
	// Note: We use dummy credentials. The CDN API might validate format,
	// but usually accepts valid-looking strings for async creation.
	bucketUrl := "https://my-private-bucket.s3.eu-central-1.amazonaws.com"
	bucketRegion := "eu01"
	accessKey := strings.ToUpper(randomString(16))
	secretKey := randomString(40)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCDNDistributionDestroy,
		Steps: []resource.TestStep{
			// 1. Distribution Create with Bucket Backend
			{
				Config: configBucketResources(bucketUrl, bucketRegion, accessKey, secretKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Basic Identity Checks
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "distribution_id"),
					resource.TestCheckResourceAttrSet("stackit_cdn_distribution.distribution", "created_at"),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "status", "ACTIVE"),

					// Backend Type Check
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.backend.type", "bucket"),

					// Bucket Specific Field Checks
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.backend.bucket_url", bucketUrl),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.backend.region", bucketRegion),

					// CRITICAL: Check Credentials Persistence
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.backend.access_key_id", accessKey),
					resource.TestCheckResourceAttr("stackit_cdn_distribution.distribution", "config.backend.secret_key", secretKey),

					// Ensure HTTP fields are not set
					resource.TestCheckNoResourceAttr("stackit_cdn_distribution.distribution", "config.backend.origin_url"),
				),
			},
			// 2. Import Test
			// This verifies that importing a bucket distribution works,
			// although note that imported resources usually lose secrets (access_key_id/secret_key)
			// because they are not in the API response.
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
				ImportState:       true,
				ImportStateVerify: true,
				// We MUST ignore credentials on import verification
				// because the API does not return them, so Terraform will see them as null/empty
				// in the new state, differing from the config.
				ImportStateVerifyIgnore: []string{
					"config.backend.access_key_id",
					"config.backend.secret_key",
				},
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
