package intake_test

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	intake "github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-runner-min.tf
var resourceIntakeRunnerMin string

//go:embed testdata/resource-runner-max.tf
var resourceIntakeRunnerMax string

//go:embed testdata/resource-intake-min.tf
var resourceIntakesMin string

//go:embed testdata/resource-intake-max.tf
var resourceIntakesMax string

const intakeRunnerResource = "stackit_intake_runner.example"
const intakesResource = "stackit_intakes.example"

var runnerNameMin = fmt.Sprintf("tf-acc-runner-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var runnerNameMinUpd = fmt.Sprintf("tf-acc-runner-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var runnerNameMax = fmt.Sprintf("tf-acc-runner-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var runnerNameMaxUpd = fmt.Sprintf("tf-acc-runner-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var runnerNameMaxPrereq = fmt.Sprintf("tf-acc-runner-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var intakeNameMin = fmt.Sprintf("tf-acc-intake-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var intakeNameMinUpd = fmt.Sprintf("tf-acc-intake-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var intakeNameMax = fmt.Sprintf("tf-acc-intake-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var intakeNameMaxUpd = fmt.Sprintf("tf-acc-intake-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var dremioUserMin = fmt.Sprintf("tfAcc%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
var dremioUserMax = fmt.Sprintf("tfAcc%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

func testIntakeRunnerConfigVarsMin() config.Variables {
	return config.Variables{
		"project_id":            config.StringVariable(testutil.ProjectId),
		"name":                  config.StringVariable(runnerNameMin),
		"max_message_size_kib":  config.IntegerVariable(1024),
		"max_messages_per_hour": config.IntegerVariable(1000),
	}
}

func testIntakeRunnerConfigVarsMax() config.Variables {
	return config.Variables{
		"project_id":            config.StringVariable(testutil.ProjectId),
		"name":                  config.StringVariable(runnerNameMax),
		"region":                config.StringVariable(testutil.Region),
		"description":           config.StringVariable("An example runner for Intake"),
		"max_message_size_kib":  config.IntegerVariable(1024),
		"max_messages_per_hour": config.IntegerVariable(1100),
	}
}

func testIntakesConfigVarsMin() config.Variables {
	return config.Variables{
		"project_id":                   config.StringVariable(testutil.ProjectId),
		"runner_name":                  config.StringVariable(runnerNameMin),
		"intake_name":                  config.StringVariable(intakeNameMin),
		"max_message_size_kib":         config.IntegerVariable(1024),
		"max_messages_per_hour":        config.IntegerVariable(1000),
		"dremio_display_name":          config.StringVariable(fmt.Sprintf("tfAccDremio%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))),
		"dremio_user_email":            config.StringVariable(fmt.Sprintf("tf-acc-%s@example.com", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))),
		"dremio_user_first_name":       config.StringVariable("Intake"),
		"dremio_user_last_name":        config.StringVariable("Min"),
		"dremio_user_name":             config.StringVariable(dremioUserMin),
		"dremio_user_password":         config.StringVariable(fmt.Sprintf("TestAcc12!@%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))),
		"dremio_personal_access_token": config.StringVariable("pending-dremio-pat"),
	}
}

func testIntakesConfigVarsMax() config.Variables {
	return config.Variables{
		"project_id":                   config.StringVariable(testutil.ProjectId),
		"region":                       config.StringVariable(testutil.Region),
		"runner_name":                  config.StringVariable(runnerNameMaxPrereq),
		"intake_name":                  config.StringVariable(intakeNameMax),
		"description":                  config.StringVariable("An example full intake with dynamic Dremio"),
		"max_message_size_kib":         config.IntegerVariable(1024),
		"max_messages_per_hour":        config.IntegerVariable(1000),
		"dremio_display_name":          config.StringVariable(fmt.Sprintf("tfAccDremio%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))),
		"dremio_user_email":            config.StringVariable(fmt.Sprintf("tf-acc-%s@example.com", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))),
		"dremio_user_first_name":       config.StringVariable("Acc"),
		"dremio_user_last_name":        config.StringVariable("Test"),
		"dremio_user_name":             config.StringVariable(dremioUserMax),
		"dremio_user_password":         config.StringVariable(fmt.Sprintf("TestAcc12!@%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))),
		"dremio_personal_access_token": config.StringVariable("pending-dremio-pat"),
	}
}

func testIntakeRunnerConfigVarsMinUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testIntakeRunnerConfigVarsMin()))
	maps.Copy(tempConfig, testIntakeRunnerConfigVarsMin())
	tempConfig["name"] = config.StringVariable(runnerNameMinUpd)
	return tempConfig
}

func testIntakeRunnerConfigVarsMaxUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testIntakeRunnerConfigVarsMax()))
	maps.Copy(tempConfig, testIntakeRunnerConfigVarsMax())
	tempConfig["name"] = config.StringVariable(runnerNameMaxUpd)
	return tempConfig
}

func testIntakesConfigVarsMinUpdated(base config.Variables) config.Variables {
	tempConfig := make(config.Variables, len(base))
	maps.Copy(tempConfig, base)
	tempConfig["intake_name"] = config.StringVariable(intakeNameMinUpd)
	return tempConfig
}

func testIntakesConfigVarsMaxUpdated(base config.Variables) config.Variables {
	tempConfig := make(config.Variables, len(base))
	maps.Copy(tempConfig, base)
	tempConfig["intake_name"] = config.StringVariable(intakeNameMaxUpd)
	tempConfig["description"] = config.StringVariable("Updated full intake description")
	tempConfig["max_messages_per_hour"] = config.IntegerVariable(1100)
	return tempConfig
}

// getDremioPAT authenticates against Dremio UI API, enables PAT support key, resolves user UUID, and issues a PAT
func getDremioPAT(ctx context.Context, uiEndpoint, username, password string) (string, error) {
	if !strings.HasPrefix(uiEndpoint, "http://") && !strings.HasPrefix(uiEndpoint, "https://") {
		uiEndpoint = "https://" + uiEndpoint
	}
	uiEndpoint = strings.TrimSuffix(uiEndpoint, "/")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // acceptance test TLS skip
	}
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}

	// 1. authenticate with retry loop for service startup readiness POST /oauth/token
	tokenURL := fmt.Sprintf("%s/oauth/token", uiEndpoint)
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("scope", "dremio.all")
	form.Set("username", username)
	form.Set("password", password)

	var accessToken string
	var lastErr error

	maxRetries := 18 // 3 minutes total retry time
	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return "", fmt.Errorf("creating login request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := httpClient.Do(req)
		if err == nil {
			respBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var tokenResp struct {
					AccessToken string `json:"access_token"`
				}
				if jsonErr := json.Unmarshal(respBytes, &tokenResp); jsonErr == nil && tokenResp.AccessToken != "" {
					accessToken = tokenResp.AccessToken
					break
				}
			} else {
				lastErr = fmt.Errorf("login status %d: %s", resp.StatusCode, string(respBytes))
			}
		} else {
			lastErr = err
		}

		time.Sleep(10 * time.Second)
	}

	if accessToken == "" {
		return "", fmt.Errorf("failed to authenticate against Dremio at %s after retries: %w", tokenURL, lastErr)
	}

	// 2. enable PAT support key PUT /apiv2/settings/auth.personal-access-tokens.enabled
	settingsURL := fmt.Sprintf("%s/apiv2/settings/auth.personal-access-tokens.enabled", uiEndpoint)
	settingsBodyMap := map[string]interface{}{
		"type":  "BOOLEAN",
		"id":    "auth.personal-access-tokens.enabled",
		"value": true,
	}
	settingsJSON, err := json.Marshal(settingsBodyMap)
	if err != nil {
		return "", fmt.Errorf("marshaling settings body: %w", err)
	}

	settingsReq, err := http.NewRequestWithContext(ctx, http.MethodPut, settingsURL, bytes.NewBuffer(settingsJSON))
	if err != nil {
		return "", fmt.Errorf("creating settings request: %w", err)
	}
	settingsReq.Header.Set("Authorization", "Bearer "+accessToken)
	settingsReq.Header.Set("Content-Type", "application/json")

	settingsResp, err := httpClient.Do(settingsReq)
	if err != nil {
		return "", fmt.Errorf("enabling PAT support key at %s: %w", settingsURL, err)
	}
	_ = settingsResp.Body.Close()

	// 3. resolve user UUID GET /api/v3/user/by-name/{username}
	userURL := fmt.Sprintf("%s/api/v3/user/by-name/%s", uiEndpoint, url.PathEscape(username))
	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("creating user lookup request: %w", err)
	}
	userReq.Header.Set("Authorization", "Bearer "+accessToken)

	userResp, err := httpClient.Do(userReq)
	if err != nil {
		return "", fmt.Errorf("looking up user UUID at %s: %w", userURL, err)
	}
	defer func() { _ = userResp.Body.Close() }()

	userBytes, err := io.ReadAll(userResp.Body)
	if err != nil {
		return "", fmt.Errorf("reading user response: %w", err)
	}

	if userResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("user lookup failed with status %d: %s", userResp.StatusCode, string(userBytes))
	}

	var userObj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(userBytes, &userObj); err != nil || userObj.ID == "" {
		return "", fmt.Errorf("failed to parse user UUID from response: %s", string(userBytes))
	}

	// 4. issue Personal Access Token POST /api/v3/user/{id}/token
	patURL := fmt.Sprintf("%s/api/v3/user/%s/token", uiEndpoint, userObj.ID)
	patBodyMap := map[string]interface{}{
		"label":                "acceptance-test",
		"millisecondsToExpire": 86400000, // 24 hours
	}
	patBodyJSON, err := json.Marshal(patBodyMap)
	if err != nil {
		return "", fmt.Errorf("marshaling PAT request body: %w", err)
	}

	patReq, err := http.NewRequestWithContext(ctx, http.MethodPost, patURL, bytes.NewBuffer(patBodyJSON))
	if err != nil {
		return "", fmt.Errorf("creating PAT request: %w", err)
	}
	patReq.Header.Set("Authorization", "Bearer "+accessToken)
	patReq.Header.Set("Content-Type", "application/json")

	patResp, err := httpClient.Do(patReq)
	if err != nil {
		return "", fmt.Errorf("requesting PAT at %s: %w", patURL, err)
	}
	defer func() { _ = patResp.Body.Close() }()

	patBytes, err := io.ReadAll(patResp.Body)
	if err != nil {
		return "", fmt.Errorf("reading PAT response: %w", err)
	}

	if patResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PAT creation failed with status %d: %s", patResp.StatusCode, string(patBytes))
	}

	rawToken := strings.TrimSpace(string(patBytes))
	rawToken = strings.Trim(rawToken, `"`)

	var patObj struct {
		Token string `json:"token"`
		PAT   string `json:"pat"`
	}
	if err := json.Unmarshal(patBytes, &patObj); err == nil {
		if patObj.Token != "" {
			return patObj.Token, nil
		}
		if patObj.PAT != "" {
			return patObj.PAT, nil
		}
	}

	return rawToken, nil
}

func TestAccIntakeRunnerMin(t *testing.T) {
	cfg := testIntakeRunnerConfigVarsMin()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakeRunnerDestroy,
		Steps: []resource.TestStep{
			// Create the minimum runner from the HCL file
			{
				ConfigVariables: cfg,
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + resourceIntakeRunnerMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(cfg["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(cfg["name"])),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "description"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "labels"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(cfg["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(cfg["max_messages_per_hour"])),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "uri"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "create_time"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.Region),
				),
			},
			// Data source check
			{
				ConfigVariables: cfg,
				Config: fmt.Sprintf(`
				%s
				%s
				data "stackit_intake_runner" "example" {
					project_id = %s.project_id
					runner_id  = %s.runner_id
					region     = %s.region
				}`, testutil.NewConfigBuilder().BuildProviderConfig(), resourceIntakeRunnerMin, intakeRunnerResource, intakeRunnerResource, intakeRunnerResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "project_id", "data.stackit_intake_runner.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "runner_id", "data.stackit_intake_runner.example", "runner_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "name", "data.stackit_intake_runner.example", "name"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "region", "data.stackit_intake_runner.example", "region"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "description"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "labels"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "uri", "data.stackit_intake_runner.example", "uri"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "create_time", "data.stackit_intake_runner.example", "create_time"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "max_messages_per_hour", "data.stackit_intake_runner.example", "max_messages_per_hour"),
				),
			},
			// Simulate terraform import
			{
				ConfigVariables:   cfg,
				Config:            testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMin,
				ResourceName:      intakeRunnerResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[intakeRunnerResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", intakeRunnerResource)
					}
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["project_id"], r.Primary.Attributes["region"], r.Primary.Attributes["runner_id"]), nil
				},
			},
			// Update check
			{
				ConfigVariables: testIntakeRunnerConfigVarsMinUpdated(),
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(cfg["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(cfg["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(cfg["max_messages_per_hour"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.Region),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "description"),
					resource.TestCheckNoResourceAttr(intakeRunnerResource, "labels"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "uri"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "create_time"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
				),
			},
		},
	})
}

func TestAccIntakeRunnerMax(t *testing.T) {
	cfg := testIntakeRunnerConfigVarsMax()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakeRunnerDestroy,
		Steps: []resource.TestStep{
			// Create the max intake runner from HCL files and verify comparison
			{
				ConfigVariables: cfg,
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(cfg["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(cfg["name"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "description", testutil.ConvertConfigVariable(cfg["description"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(cfg["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(cfg["max_messages_per_hour"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.%", "2"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.env", "development"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.created_by", "terraform-provider-stackit"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "uri"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "create_time"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.ConvertConfigVariable(cfg["region"])),
				),
			},
			// Data source check
			{
				ConfigVariables: cfg,
				Config: fmt.Sprintf(`
				%s
				%s
				data "stackit_intake_runner" "example" {
					project_id = %s.project_id
					runner_id  = %s.runner_id
				}`, testutil.NewConfigBuilder().BuildProviderConfig(), resourceIntakeRunnerMax, intakeRunnerResource, intakeRunnerResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "project_id", "data.stackit_intake_runner.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "runner_id", "data.stackit_intake_runner.example", "runner_id"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "name", "data.stackit_intake_runner.example", "name"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "description", "data.stackit_intake_runner.example", "description"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "region", "data.stackit_intake_runner.example", "region"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "uri", "data.stackit_intake_runner.example", "uri"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "create_time", "data.stackit_intake_runner.example", "create_time"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "labels.env", "data.stackit_intake_runner.example", "labels.env"),
					resource.TestCheckResourceAttrPair(intakeRunnerResource, "max_messages_per_hour", "data.stackit_intake_runner.example", "max_messages_per_hour"),
				),
			},
			// Simulate terraform import
			{
				ConfigVariables:   cfg,
				Config:            testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMax,
				ResourceName:      intakeRunnerResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[intakeRunnerResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", intakeRunnerResource)
					}
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["project_id"], r.Primary.Attributes["region"], r.Primary.Attributes["runner_id"]), nil
				},
			},
			// Update and verify changes are reflected
			{
				ConfigVariables: testIntakeRunnerConfigVarsMaxUpdated(),
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourceIntakeRunnerMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakeRunnerResource, "project_id", testutil.ConvertConfigVariable(cfg["project_id"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "name", testutil.ConvertConfigVariable(testIntakeRunnerConfigVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "description", testutil.ConvertConfigVariable(cfg["description"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_message_size_kib", testutil.ConvertConfigVariable(cfg["max_message_size_kib"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "max_messages_per_hour", testutil.ConvertConfigVariable(cfg["max_messages_per_hour"])),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.%", "2"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.env", "development"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "labels.created_by", "terraform-provider-stackit"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "id"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "uri"),
					resource.TestCheckResourceAttrSet(intakeRunnerResource, "create_time"),
					resource.TestCheckResourceAttr(intakeRunnerResource, "region", testutil.ConvertConfigVariable(cfg["region"])),
				),
			},
		},
	})
}

func TestAccIntakesMin(t *testing.T) {
	cfg := testIntakesConfigVarsMin()
	cfgUpdated := testIntakesConfigVarsMinUpdated(cfg)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakesDestroy,
		Steps: []resource.TestStep{
			// Step 1: Provision prerequisites and dynamically acquire Dremio PAT
			{
				ConfigVariables: cfg,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + strings.Split(resourceIntakesMin, "resource \"stackit_intakes\"")[0],
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_dremio_instance.dremio", "endpoints.ui"),
					resource.TestCheckResourceAttrSet("stackit_dremio_user.dremio_user", "user_id"),
					func(s *terraform.State) error {
						dremioRes, ok := s.RootModule().Resources["stackit_dremio_instance.dremio"]
						if !ok {
							return fmt.Errorf("could not find stackit_dremio_instance.dremio in state")
						}
						uiEndpoint := dremioRes.Primary.Attributes["endpoints.ui"]
						if uiEndpoint == "" {
							return fmt.Errorf("dremio instance endpoints.ui is empty")
						}

						username := testutil.ConvertConfigVariable(cfg["dremio_user_name"])
						password := testutil.ConvertConfigVariable(cfg["dremio_user_password"])

						dremioPAT, err := getDremioPAT(context.Background(), uiEndpoint, username, password)
						if err != nil {
							return fmt.Errorf("failed to obtain Dremio PAT: %w", err)
						}

						cfg["dremio_personal_access_token"] = config.StringVariable(dremioPAT)
						cfgUpdated["dremio_personal_access_token"] = config.StringVariable(dremioPAT)
						return nil
					},
				),
			},
			// Step 2: Create minimal intake using the generated PAT
			{
				ConfigVariables: cfg,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + resourceIntakesMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakesResource, "project_id", testutil.ConvertConfigVariable(cfg["project_id"])),
					resource.TestCheckResourceAttr(intakesResource, "name", testutil.ConvertConfigVariable(cfg["intake_name"])),
					resource.TestCheckResourceAttrSet(intakesResource, "intake_id"),
					resource.TestCheckResourceAttrSet(intakesResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakesResource, "id"),
					resource.TestCheckResourceAttrSet(intakesResource, "uri"),
					resource.TestCheckResourceAttrSet(intakesResource, "create_time"),
					resource.TestCheckResourceAttr(intakesResource, "region", testutil.Region),
				),
			},
			// Step 3: Data source check
			{
				ConfigVariables: cfg,
				Config: fmt.Sprintf(`
				%s
				%s
				data "stackit_intakes" "example" {
					project_id = %s.project_id
					intake_id  = %s.intake_id
					region     = %s.region
				}`, testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig(), resourceIntakesMin, intakesResource, intakesResource, intakesResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(intakesResource, "project_id", "data.stackit_intakes.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "intake_id", "data.stackit_intakes.example", "intake_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "name", "data.stackit_intakes.example", "name"),
					resource.TestCheckResourceAttrPair(intakesResource, "runner_id", "data.stackit_intakes.example", "runner_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "region", "data.stackit_intakes.example", "region"),
					resource.TestCheckResourceAttrPair(intakesResource, "uri", "data.stackit_intakes.example", "uri"),
					resource.TestCheckResourceAttrPair(intakesResource, "create_time", "data.stackit_intakes.example", "create_time"),
				),
			},
			// Step 4: Import state check
			{
				ConfigVariables:         cfg,
				Config:                  testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + resourceIntakesMin,
				ResourceName:            intakesResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"dremio_personal_access_token"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[intakesResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", intakesResource)
					}
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["project_id"], r.Primary.Attributes["region"], r.Primary.Attributes["intake_id"]), nil
				},
			},
			// Step 5: Update check
			{
				ConfigVariables: cfgUpdated,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + resourceIntakesMin,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakesResource, "project_id", testutil.ConvertConfigVariable(cfg["project_id"])),
					resource.TestCheckResourceAttr(intakesResource, "name", testutil.ConvertConfigVariable(cfgUpdated["intake_name"])),
					resource.TestCheckResourceAttrSet(intakesResource, "intake_id"),
				),
			},
		},
	})
}

func TestAccIntakesMax(t *testing.T) {
	cfg := testIntakesConfigVarsMax()
	cfgUpdated := testIntakesConfigVarsMaxUpdated(cfg)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntakesDestroy,
		Steps: []resource.TestStep{
			// Step 1: Provision prerequisites and dynamically acquire Dremio PAT
			{
				ConfigVariables: cfg,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + strings.Split(resourceIntakesMax, "resource \"stackit_intakes\"")[0],
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_dremio_instance.dremio", "endpoints.ui"),
					resource.TestCheckResourceAttrSet("stackit_dremio_user.dremio_user", "user_id"),
					func(s *terraform.State) error {
						dremioRes, ok := s.RootModule().Resources["stackit_dremio_instance.dremio"]
						if !ok {
							return fmt.Errorf("could not find stackit_dremio_instance.dremio in state")
						}
						uiEndpoint := dremioRes.Primary.Attributes["endpoints.ui"]
						if uiEndpoint == "" {
							return fmt.Errorf("dremio instance endpoints.ui is empty")
						}

						username := testutil.ConvertConfigVariable(cfg["dremio_user_name"])
						password := testutil.ConvertConfigVariable(cfg["dremio_user_password"])

						dremioPAT, err := getDremioPAT(context.Background(), uiEndpoint, username, password)
						if err != nil {
							return fmt.Errorf("failed to obtain Dremio PAT: %w", err)
						}

						cfg["dremio_personal_access_token"] = config.StringVariable(dremioPAT)
						cfgUpdated["dremio_personal_access_token"] = config.StringVariable(dremioPAT)
						return nil
					},
				),
			},
			// Step 2: Create full intake with generated Dremio PAT
			{
				ConfigVariables: cfg,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + resourceIntakesMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakesResource, "project_id", testutil.ConvertConfigVariable(cfg["project_id"])),
					resource.TestCheckResourceAttr(intakesResource, "name", testutil.ConvertConfigVariable(cfg["intake_name"])),
					resource.TestCheckResourceAttr(intakesResource, "description", testutil.ConvertConfigVariable(cfg["description"])),
					resource.TestCheckResourceAttr(intakesResource, "labels.env", "development"),
					resource.TestCheckResourceAttr(intakesResource, "labels.created_by", "terraform-provider-stackit"),
					resource.TestCheckResourceAttr(intakesResource, "catalog_auth_type", "dremio"),
					resource.TestCheckResourceAttr(intakesResource, "catalog_namespace", "intake"),
					resource.TestCheckResourceAttr(intakesResource, "catalog_warehouse", "default"),
					resource.TestCheckResourceAttrSet(intakesResource, "intake_id"),
					resource.TestCheckResourceAttrSet(intakesResource, "runner_id"),
					resource.TestCheckResourceAttrSet(intakesResource, "catalog_uri"),
					resource.TestCheckResourceAttrSet(intakesResource, "catalog_table_name"),
				),
			},
			// Step 3: Data source check
			{
				ConfigVariables: cfg,
				Config: fmt.Sprintf(`
				%s
				%s
				data "stackit_intakes" "example" {
					project_id = %s.project_id
					intake_id  = %s.intake_id
				}`, testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig(), resourceIntakesMax, intakesResource, intakesResource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(intakesResource, "project_id", "data.stackit_intakes.example", "project_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "intake_id", "data.stackit_intakes.example", "intake_id"),
					resource.TestCheckResourceAttrPair(intakesResource, "name", "data.stackit_intakes.example", "name"),
					resource.TestCheckResourceAttrPair(intakesResource, "description", "data.stackit_intakes.example", "description"),
					resource.TestCheckResourceAttrPair(intakesResource, "catalog_auth_type", "data.stackit_intakes.example", "catalog_auth_type"),
					resource.TestCheckResourceAttrPair(intakesResource, "catalog_namespace", "data.stackit_intakes.example", "catalog_namespace"),
					resource.TestCheckResourceAttrPair(intakesResource, "catalog_warehouse", "data.stackit_intakes.example", "catalog_warehouse"),
				),
			},
			// Step 4: Import state check (ignore write-only PAT)
			{
				ConfigVariables:         cfg,
				Config:                  testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + resourceIntakesMax,
				ResourceName:            intakesResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"dremio_personal_access_token"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources[intakesResource]
					if !ok {
						return "", fmt.Errorf("couldn't find resource %s", intakesResource)
					}
					return fmt.Sprintf("%s,%s,%s", r.Primary.Attributes["project_id"], r.Primary.Attributes["region"], r.Primary.Attributes["intake_id"]), nil
				},
			},
			// Step 5: Update check
			{
				ConfigVariables: cfgUpdated,
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).Experiments(testutil.ExperimentDremio).BuildProviderConfig() + "\n" + resourceIntakesMax,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(intakesResource, "project_id", testutil.ConvertConfigVariable(cfg["project_id"])),
					resource.TestCheckResourceAttr(intakesResource, "name", testutil.ConvertConfigVariable(cfgUpdated["intake_name"])),
					resource.TestCheckResourceAttr(intakesResource, "description", testutil.ConvertConfigVariable(cfgUpdated["description"])),
					resource.TestCheckResourceAttrSet(intakesResource, "intake_id"),
				),
			},
		},
	})
}

// testAccCheckIntakeRunnerDestroy verifies all runners are destroyed, actively deleting any leftovers
func testAccCheckIntakeRunnerDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := intake.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.IntakeCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var instancesToDestroy []string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_intake_runner" {
			continue
		}
		runnerId := strings.Split(rs.Primary.ID, core.Separator)[2]
		instancesToDestroy = append(instancesToDestroy, runnerId)
	}

	instancesResp, err := client.DefaultAPI.ListIntakeRunners(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("listing intake runners: %w", err)
	}

	for i := range instancesResp.IntakeRunners {
		if utils.Contains(instancesToDestroy, instancesResp.IntakeRunners[i].Id) {
			err := client.DefaultAPI.DeleteIntakeRunner(ctx, testutil.ProjectId, testutil.Region, instancesResp.IntakeRunners[i].Id).Execute()
			if err != nil {
				return fmt.Errorf("destroying runner %s during CheckDestroy: %w", instancesResp.IntakeRunners[i].Id, err)
			}

			_, err = wait.DeleteIntakeRunnerWaitHandler(ctx, client.DefaultAPI, testutil.ProjectId, testutil.Region, instancesResp.IntakeRunners[i].Id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying runner %s during CheckDestroy: waiting for deletion %w", instancesResp.IntakeRunners[i].Id, err)
			}
		}
	}
	return nil
}

// testAccCheckIntakesDestroy verifies all intakes are destroyed, actively deleting any leftovers
func testAccCheckIntakesDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := intake.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.IntakeCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var instancesToDestroy []string
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_intakes" {
			continue
		}
		idParts := strings.Split(rs.Primary.ID, core.Separator)
		if len(idParts) < 3 {
			continue
		}
		intakeId := idParts[2]
		instancesToDestroy = append(instancesToDestroy, intakeId)
	}

	instancesResp, err := client.DefaultAPI.ListIntakes(ctx, testutil.ProjectId, testutil.Region).Execute()
	if err != nil {
		return fmt.Errorf("listing intakes: %w", err)
	}

	for i := range instancesResp.Intakes {
		if utils.Contains(instancesToDestroy, instancesResp.Intakes[i].Id) {
			err := client.DefaultAPI.DeleteIntake(ctx, testutil.ProjectId, testutil.Region, instancesResp.Intakes[i].Id).Execute()
			if err != nil {
				return fmt.Errorf("destroying intake %s during CheckDestroy: %w", instancesResp.Intakes[i].Id, err)
			}

			_, err = wait.DeleteIntakeWaitHandler(ctx, client.DefaultAPI, testutil.ProjectId, testutil.Region, instancesResp.Intakes[i].Id).WaitWithContext(ctx)
			if err != nil {
				return fmt.Errorf("destroying intake %s during CheckDestroy: waiting for deletion %w", instancesResp.Intakes[i].Id, err)
			}
		}
	}
	return nil
}
