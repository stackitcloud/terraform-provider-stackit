package cdn

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	cdnSdk "github.com/stackitcloud/stackit-sdk-go/services/cdn/v1api"
)

func createTestConfig(vals map[string]attr.Value) types.Object {
	if _, ok := vals["blocked_ips"]; !ok {
		vals["blocked_ips"] = types.ListValueMust(types.StringType, []attr.Value{})
	}
	if _, ok := vals["default_cache_duration"]; !ok {
		vals["default_cache_duration"] = types.StringNull()
	}
	if _, ok := vals["monthly_limit_bytes"]; !ok {
		vals["monthly_limit_bytes"] = types.Int64Null()
	}
	return types.ObjectValueMust(configTypes, vals)
}

// configFixture builds a config Object with fully-null defaults for every
// attribute. Callers opt in to specific values via mod funcs. Follows the
// fixture pattern used elsewhere in the repo (see fixtureModel in
// stackit/internal/services/telemetryrouter/accesstoken/resource_test.go).
func configFixture(mods ...func(vals map[string]attr.Value)) types.Object {
	vals := map[string]attr.Value{
		"backend":                types.ObjectNull(backendTypes),
		"regions":                types.ListNull(types.StringType),
		"blocked_countries":      types.ListValueMust(types.StringType, []attr.Value{}),
		"blocked_ips":            types.ListValueMust(types.StringType, []attr.Value{}),
		"default_cache_duration": types.StringNull(),
		"monthly_limit_bytes":    types.Int64Null(),
		"optimizer":              types.ObjectNull(optimizerTypes),
		"redirects":              types.ObjectNull(redirectsTypes),
		"waf":                    types.ObjectNull(wafTypes),
		"tls":                    types.ObjectNull(tlsTypes),
		"strip_response_cookies": types.BoolUnknown(),
		"forward_host_header":    types.BoolUnknown(),
	}
	for _, mod := range mods {
		mod(vals)
	}
	return types.ObjectValueMust(configTypes, vals)
}

func TestToCreatePayload(t *testing.T) {
	headers := map[string]attr.Value{
		"testHeader0": types.StringValue("testHeaderValue0"),
		"testHeader1": types.StringValue("testHeaderValue1"),
	}
	originRequestHeaders := types.MapValueMust(types.StringType, headers)
	geofencingCountries := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("DE"),
		types.StringValue("FR"),
	})
	geofencing := types.MapValueMust(geofencingTypes.ElemType, map[string]attr.Value{
		"https://de.mycoolapp.com": geofencingCountries,
	})

	backend := types.ObjectValueMust(backendTypes, map[string]attr.Value{
		"type":                   types.StringValue("http"),
		"origin_url":             types.StringValue("https://www.mycoolapp.com"),
		"origin_request_headers": originRequestHeaders,
		"geofencing":             geofencing,
		"bucket_url":             types.StringNull(),
		"region":                 types.StringNull(),
		"credentials":            types.ObjectNull(backendCredentialsTypes),
	})
	regions := []attr.Value{types.StringValue("EU"), types.StringValue("US")}
	regionsFixture := types.ListValueMust(types.StringType, regions)
	blockedCountries := []attr.Value{types.StringValue("XX"), types.StringValue("YY"), types.StringValue("ZZ")}
	blockedCountriesFixture := types.ListValueMust(types.StringType, blockedCountries)
	optimizer := types.ObjectValueMust(optimizerTypes, map[string]attr.Value{
		"enabled": types.BoolValue(true),
	})
	emptyWafSet := types.SetValueMust(types.StringType, []attr.Value{})
	expectedDefaultWafConfig := cdnSdk.WafConfig{
		Mode:                       cdnSdk.WafMode("DISABLED"),
		Type:                       cdnSdk.WafType("FREE"),
		AllowedHttpVersions:        []string{},
		AllowedRequestContentTypes: []string{},
		AllowedHttpMethods:         []string{},
		EnabledRuleIds:             []string{},
		DisabledRuleIds:            []string{},
		LogOnlyRuleIds:             []string{},
		EnabledRuleGroupIds:        []string{},
		DisabledRuleGroupIds:       []string{},
		LogOnlyRuleGroupIds:        []string{},
		EnabledRuleCollectionIds:   []string{},
		DisabledRuleCollectionIds:  []string{},
		LogOnlyRuleCollectionIds:   []string{},
	}
	defaultWaf := types.ObjectValueMust(wafTypes, map[string]attr.Value{
		"mode":                          types.StringValue("DISABLED"),
		"type":                          types.StringValue("FREE"),
		"paranoia_level":                types.StringNull(),
		"allowed_http_versions":         emptyWafSet,
		"allowed_request_content_types": emptyWafSet,
		"allowed_http_methods":          emptyWafSet,
		"enabled_rule_ids":              emptyWafSet,
		"disabled_rule_ids":             emptyWafSet,
		"log_only_rule_ids":             emptyWafSet,
		"enabled_rule_group_ids":        emptyWafSet,
		"disabled_rule_group_ids":       emptyWafSet,
		"log_only_rule_group_ids":       emptyWafSet,
		"enabled_rule_collection_ids":   emptyWafSet,
		"disabled_rule_collection_ids":  emptyWafSet,
		"log_only_rule_collection_ids":  emptyWafSet,
	})

	redirectsObjType, ok := configTypes["redirects"].(basetypes.ObjectType)
	if !ok {
		t.Fatalf("configTypes[\"redirects\"] is not of type basetypes.ObjectType")
	}
	redirectsAttrTypes := redirectsObjType.AttrTypes

	config := createTestConfig(map[string]attr.Value{
		"backend":                backend,
		"regions":                regionsFixture,
		"blocked_countries":      blockedCountriesFixture,
		"optimizer":              types.ObjectNull(optimizerTypes),
		"redirects":              types.ObjectNull(redirectsTypes),
		"waf":                    defaultWaf,
		"tls":                    types.ObjectNull(tlsTypes),
		"strip_response_cookies": types.BoolUnknown(),
		"forward_host_header":    types.BoolUnknown(),
	})

	matcherValues := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("/shop/*"),
	})
	matcherVal := types.ObjectValueMust(matcherTypes, map[string]attr.Value{
		"values":                matcherValues,
		"value_match_condition": types.StringValue("ALL"),
	})
	matchersList := types.ListValueMust(types.ObjectType{AttrTypes: matcherTypes}, []attr.Value{matcherVal})

	ruleVal := types.ObjectValueMust(redirectRuleTypes, map[string]attr.Value{
		"description":          types.StringValue("Test redirect"),
		"enabled":              types.BoolValue(true),
		"target_url":           types.StringValue("https://example.com/redirect"),
		"status_code":          types.Int32Value(301),
		"rule_match_condition": types.StringValue("ALL"),
		"matchers":             matchersList,
	})
	rulesList := types.ListValueMust(types.ObjectType{AttrTypes: redirectRuleTypes}, []attr.Value{ruleVal})

	redirectsConfigVal := types.ObjectValueMust(redirectsTypes, map[string]attr.Value{
		"rules": rulesList,
	})
	populatedWafSet := types.SetValueMust(types.StringType, []attr.Value{
		types.StringValue("rule1"),
		types.StringValue("rule2"),
	})
	populatedWaf := types.ObjectValueMust(wafTypes, map[string]attr.Value{
		"mode":                          types.StringValue("ENABLED"),
		"type":                          types.StringValue("PREMIUM"),
		"paranoia_level":                types.StringValue("L2"),
		"allowed_http_versions":         populatedWafSet,
		"allowed_request_content_types": populatedWafSet,
		"allowed_http_methods":          populatedWafSet,
		"enabled_rule_ids":              populatedWafSet,
		"disabled_rule_ids":             populatedWafSet,
		"log_only_rule_ids":             populatedWafSet,
		"enabled_rule_group_ids":        populatedWafSet,
		"disabled_rule_group_ids":       populatedWafSet,
		"log_only_rule_group_ids":       populatedWafSet,
		"enabled_rule_collection_ids":   populatedWafSet,
		"disabled_rule_collection_ids":  populatedWafSet,
		"log_only_rule_collection_ids":  populatedWafSet,
	})

	expectedParanoiaLevel := cdnSdk.WafParanoiaLevel("L2")
	expectedWafConfig := cdnSdk.WafConfig{
		Mode:                       cdnSdk.WafMode("ENABLED"),
		Type:                       cdnSdk.WafType("PREMIUM"),
		ParanoiaLevel:              &expectedParanoiaLevel,
		AllowedHttpVersions:        []string{"rule1", "rule2"},
		AllowedRequestContentTypes: []string{"rule1", "rule2"},
		AllowedHttpMethods:         []string{"rule1", "rule2"},
		EnabledRuleIds:             []string{"rule1", "rule2"},
		DisabledRuleIds:            []string{"rule1", "rule2"},
		LogOnlyRuleIds:             []string{"rule1", "rule2"},
		EnabledRuleGroupIds:        []string{"rule1", "rule2"},
		DisabledRuleGroupIds:       []string{"rule1", "rule2"},
		LogOnlyRuleGroupIds:        []string{"rule1", "rule2"},
		EnabledRuleCollectionIds:   []string{"rule1", "rule2"},
		DisabledRuleCollectionIds:  []string{"rule1", "rule2"},
		LogOnlyRuleCollectionIds:   []string{"rule1", "rule2"},
	}

	modelFixture := func(mods ...func(*Model)) *Model {
		model := &Model{
			DistributionId: types.StringValue("test-distribution-id"),
			ProjectId:      types.StringValue("test-project-id"),
			Config:         config,
		}
		for _, mod := range mods {
			mod(model)
		}
		return model
	}
	tests := map[string]struct {
		Input    *Model
		Expected *cdnSdk.CreateDistributionPayload
		IsValid  bool
	}{
		"happy_path": {
			Input: modelFixture(),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					HttpBackendCreate: &cdnSdk.HttpBackendCreate{
						Geofencing:           &map[string][]string{"https://de.mycoolapp.com": {"DE", "FR"}},
						OriginRequestHeaders: &map[string]string{"testHeader0": "testHeaderValue0", "testHeader1": "testHeaderValue1"},
						OriginUrl:            "https://www.mycoolapp.com",
						Type:                 "http",
					},
				},
				Waf: &expectedDefaultWafConfig,
			},
			IsValid: true,
		},
		"happy_path_with_optimizer": {
			Input: modelFixture(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              optimizer,
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsTypes),
					"waf":                    defaultWaf,
					"tls":                    types.ObjectNull(tlsTypes),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:          []cdnSdk.Region{"EU", "US"},
				Optimizer:        cdnSdk.NewOptimizer(true),
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				Waf:              &expectedDefaultWafConfig,
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					HttpBackendCreate: &cdnSdk.HttpBackendCreate{
						Geofencing:           &map[string][]string{"https://de.mycoolapp.com": {"DE", "FR"}},
						OriginRequestHeaders: &map[string]string{"testHeader0": "testHeaderValue0", "testHeader1": "testHeaderValue1"},
						OriginUrl:            "https://www.mycoolapp.com",
						Type:                 "http",
					},
				},
			},
			IsValid: true,
		},
		"happy_path_with_redirects": {
			Input: modelFixture(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              redirectsConfigVal,
					"waf":                    defaultWaf,
					"tls":                    types.ObjectNull(tlsTypes),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				Waf:              &expectedDefaultWafConfig,
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					HttpBackendCreate: &cdnSdk.HttpBackendCreate{
						Geofencing:           &map[string][]string{"https://de.mycoolapp.com": {"DE", "FR"}},
						OriginRequestHeaders: &map[string]string{"testHeader0": "testHeaderValue0", "testHeader1": "testHeaderValue1"},
						OriginUrl:            "https://www.mycoolapp.com",
						Type:                 "http",
					},
				},
				Redirects: &cdnSdk.RedirectConfig{
					Rules: []cdnSdk.RedirectRule{
						{
							Description:        cdnSdk.PtrString("Test redirect"),
							Enabled:            cdnSdk.PtrBool(true),
							TargetUrl:          "https://example.com/redirect",
							StatusCode:         301,
							RuleMatchCondition: cdnSdk.MATCHCONDITION_ALL.Ptr(),
							Matchers: []cdnSdk.Matcher{
								{
									Values:              []string{"/shop/*"},
									ValueMatchCondition: cdnSdk.MATCHCONDITION_ALL.Ptr(),
								},
							},
						},
					},
				},
			},
			IsValid: true,
		},
		"happy_path_bucket": {
			Input: modelFixture(func(m *Model) {
				creds := types.ObjectValueMust(backendCredentialsTypes, map[string]attr.Value{
					"access_key_id":     types.StringValue("my-access"),
					"secret_access_key": types.StringValue("my-secret"),
				})
				bucketBackend := types.ObjectValueMust(backendTypes, map[string]attr.Value{
					"type":                   types.StringValue("bucket"),
					"bucket_url":             types.StringValue("https://s3.example.com"),
					"region":                 types.StringValue("eu01"),
					"credentials":            creds,
					"origin_url":             types.StringNull(),
					"origin_request_headers": types.MapNull(types.StringType),
					"geofencing":             types.MapNull(geofencingTypes.ElemType),
				})
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                bucketBackend,
					"regions":                regionsFixture, // reusing the existing one
					"blocked_countries":      blockedCountriesFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"redirects":              types.ObjectNull(redirectsTypes),
					"waf":                    defaultWaf,
					"tls":                    types.ObjectNull(tlsTypes),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Waf: &expectedDefaultWafConfig,
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					BucketBackendCreate: &cdnSdk.BucketBackendCreate{
						Type:      "bucket",
						BucketUrl: "https://s3.example.com",
						Region:    "eu01",
						Credentials: cdnSdk.BucketCredentials{
							AccessKeyId:     "my-access",
							SecretAccessKey: "my-secret",
						},
					},
				},
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
			},
			IsValid: true,
		},
		"happy_path_with_waf": {
			Input: modelFixture(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    populatedWaf,
					"tls":                    types.ObjectNull(tlsTypes),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				Waf:              &expectedWafConfig,
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					HttpBackendCreate: &cdnSdk.HttpBackendCreate{
						Geofencing:           &map[string][]string{"https://de.mycoolapp.com": {"DE", "FR"}},
						OriginRequestHeaders: &map[string]string{"testHeader0": "testHeaderValue0", "testHeader1": "testHeaderValue1"},
						OriginUrl:            "https://www.mycoolapp.com",
						Type:                 "http",
					},
				},
			},
			IsValid: true,
		},
		"happy_path_with_strip_response_and_cookies_forward": {
			Input: modelFixture(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    types.ObjectNull(wafTypes),
					"tls":                    types.ObjectNull(tlsTypes),
					"strip_response_cookies": types.BoolValue(true),
					"forward_host_header":    types.BoolValue(true),
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:              []cdnSdk.Region{"EU", "US"},
				BlockedCountries:     []string{"XX", "YY", "ZZ"},
				Waf:                  nil,
				StripResponseCookies: new(true),
				ForwardHostHeader:    new(true),
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					HttpBackendCreate: &cdnSdk.HttpBackendCreate{
						Geofencing:           &map[string][]string{"https://de.mycoolapp.com": {"DE", "FR"}},
						OriginRequestHeaders: &map[string]string{"testHeader0": "testHeaderValue0", "testHeader1": "testHeaderValue1"},
						OriginUrl:            "https://www.mycoolapp.com",
						Type:                 "http",
					},
				},
			},
			IsValid: true,
		},
		"happy_path_with_tls": {
			Input: modelFixture(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":           backend,
					"regions":           regionsFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"blocked_countries": blockedCountriesFixture,
					"redirects":         types.ObjectNull(redirectsAttrTypes),
					"waf":               types.ObjectNull(wafTypes),
					"tls": types.ObjectValueMust(tlsTypes, map[string]attr.Value{
						"enable_tls_10": types.BoolValue(true),
						"enable_tls_11": types.BoolValue(true),
					}),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				Waf:              nil,
				Tls: &cdnSdk.TlsConfig{
					EnableTls10: true,
					EnableTls11: true,
				},
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					HttpBackendCreate: &cdnSdk.HttpBackendCreate{
						Geofencing:           &map[string][]string{"https://de.mycoolapp.com": {"DE", "FR"}},
						OriginRequestHeaders: &map[string]string{"testHeader0": "testHeaderValue0", "testHeader1": "testHeaderValue1"},
						OriginUrl:            "https://www.mycoolapp.com",
						Type:                 "http",
					},
				},
			},
			IsValid: true,
		},
		"happy_path_with_blocked_ips": {
			Input: modelFixture(func(m *Model) {
				m.Config = configFixture(func(v map[string]attr.Value) {
					v["backend"] = backend
					v["regions"] = regionsFixture
					v["blocked_countries"] = blockedCountriesFixture
					v["waf"] = defaultWaf
					v["blocked_ips"] = types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("1.2.3.4"),
						types.StringValue("5.6.7.8"),
					})
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				BlockedIps:       []string{"1.2.3.4", "5.6.7.8"},
				Waf:              &expectedDefaultWafConfig,
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					HttpBackendCreate: &cdnSdk.HttpBackendCreate{
						Geofencing:           &map[string][]string{"https://de.mycoolapp.com": {"DE", "FR"}},
						OriginRequestHeaders: &map[string]string{"testHeader0": "testHeaderValue0", "testHeader1": "testHeaderValue1"},
						OriginUrl:            "https://www.mycoolapp.com",
						Type:                 "http",
					},
				},
			},
			IsValid: true,
		},
		"happy_path_with_default_cache_duration": {
			Input: modelFixture(func(m *Model) {
				m.Config = configFixture(func(v map[string]attr.Value) {
					v["backend"] = backend
					v["regions"] = regionsFixture
					v["blocked_countries"] = blockedCountriesFixture
					v["waf"] = defaultWaf
					v["default_cache_duration"] = types.StringValue("P1DT2H30M")
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:              []cdnSdk.Region{"EU", "US"},
				BlockedCountries:     []string{"XX", "YY", "ZZ"},
				DefaultCacheDuration: cdnSdk.PtrString("P1DT2H30M"),
				Waf:                  &expectedDefaultWafConfig,
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					HttpBackendCreate: &cdnSdk.HttpBackendCreate{
						Geofencing:           &map[string][]string{"https://de.mycoolapp.com": {"DE", "FR"}},
						OriginRequestHeaders: &map[string]string{"testHeader0": "testHeaderValue0", "testHeader1": "testHeaderValue1"},
						OriginUrl:            "https://www.mycoolapp.com",
						Type:                 "http",
					},
				},
			},
			IsValid: true,
		},
		"happy_path_with_monthly_limit_bytes": {
			Input: modelFixture(func(m *Model) {
				m.Config = configFixture(func(v map[string]attr.Value) {
					v["backend"] = backend
					v["regions"] = regionsFixture
					v["blocked_countries"] = blockedCountriesFixture
					v["waf"] = defaultWaf
					v["monthly_limit_bytes"] = types.Int64Value(1073741824)
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:           []cdnSdk.Region{"EU", "US"},
				BlockedCountries:  []string{"XX", "YY", "ZZ"},
				MonthlyLimitBytes: cdnSdk.PtrInt64(1073741824),
				Waf:               &expectedDefaultWafConfig,
				Backend: cdnSdk.CreateDistributionPayloadBackend{
					HttpBackendCreate: &cdnSdk.HttpBackendCreate{
						Geofencing:           &map[string][]string{"https://de.mycoolapp.com": {"DE", "FR"}},
						OriginRequestHeaders: &map[string]string{"testHeader0": "testHeaderValue0", "testHeader1": "testHeaderValue1"},
						OriginUrl:            "https://www.mycoolapp.com",
						Type:                 "http",
					},
				},
			},
			IsValid: true,
		},
		"sad_path_model_nil": {
			Input:    nil,
			Expected: nil,
			IsValid:  false,
		},
		"sad_path_config_error": {
			Input: modelFixture(func(m *Model) {
				m.Config = types.ObjectNull(configTypes)
			}),
			Expected: nil,
			IsValid:  false,
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			res, err := toCreatePayload(context.Background(), tc.Input)
			if err != nil && tc.IsValid {
				t.Fatalf("Error converting model to create payload: %v", err)
			}
			if err == nil && !tc.IsValid {
				t.Fatalf("Should have failed")
			}
			if tc.IsValid {
				// set generated ID before diffing
				tc.Expected.IntentId = res.IntentId

				diff := cmp.Diff(res, tc.Expected, cmpopts.EquateEmpty())
				if diff != "" {
					t.Fatalf("Create Payload not as expected: %s", diff)
				}
			}
		})
	}
}

func TestConvertConfig(t *testing.T) {
	headers := map[string]attr.Value{
		"testHeader0": types.StringValue("testHeaderValue0"),
		"testHeader1": types.StringValue("testHeaderValue1"),
	}
	originRequestHeaders := types.MapValueMust(types.StringType, headers)
	geofencingCountries := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("DE"),
		types.StringValue("FR"),
	})
	geofencing := types.MapValueMust(geofencingTypes.ElemType, map[string]attr.Value{
		"https://de.mycoolapp.com": geofencingCountries,
	})
	backend := types.ObjectValueMust(backendTypes, map[string]attr.Value{
		"type":                   types.StringValue("http"),
		"origin_url":             types.StringValue("https://www.mycoolapp.com"),
		"origin_request_headers": originRequestHeaders,
		"geofencing":             geofencing,
		"bucket_url":             types.StringNull(),
		"region":                 types.StringNull(),
		"credentials":            types.ObjectNull(backendCredentialsTypes),
	})
	regions := []attr.Value{types.StringValue("EU"), types.StringValue("US")}
	regionsFixture := types.ListValueMust(types.StringType, regions)
	blockedCountries := []attr.Value{types.StringValue("XX"), types.StringValue("YY"), types.StringValue("ZZ")}
	blockedCountriesFixture := types.ListValueMust(types.StringType, blockedCountries)
	optimizer := types.ObjectValueMust(optimizerTypes, map[string]attr.Value{"enabled": types.BoolValue(true)})

	config := createTestConfig(map[string]attr.Value{
		"backend":                backend,
		"regions":                regionsFixture,
		"optimizer":              types.ObjectNull(optimizerTypes),
		"blocked_countries":      blockedCountriesFixture,
		"redirects":              types.ObjectNull(redirectsTypes),
		"waf":                    types.ObjectNull(wafTypes),
		"tls":                    types.ObjectNull(tlsTypes),
		"strip_response_cookies": types.BoolUnknown(),
		"forward_host_header":    types.BoolUnknown(),
	})

	matcherValues := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("/shop/*"),
	})
	matcherVal := types.ObjectValueMust(matcherTypes, map[string]attr.Value{
		"values":                matcherValues,
		"value_match_condition": types.StringValue("ALL"),
	})
	matchersList := types.ListValueMust(types.ObjectType{AttrTypes: matcherTypes}, []attr.Value{matcherVal})

	ruleVal := types.ObjectValueMust(redirectRuleTypes, map[string]attr.Value{
		"description":          types.StringValue("Test redirect"),
		"enabled":              types.BoolValue(true),
		"target_url":           types.StringValue("https://example.com/redirect"),
		"status_code":          types.Int32Value(301),
		"rule_match_condition": types.StringValue("ALL"),
		"matchers":             matchersList,
	})
	rulesList := types.ListValueMust(types.ObjectType{AttrTypes: redirectRuleTypes}, []attr.Value{ruleVal})
	populatedWafSet := types.SetValueMust(types.StringType, []attr.Value{
		types.StringValue("rule1"),
		types.StringValue("rule2"),
	})

	redirectsConfigVal := types.ObjectValueMust(redirectsTypes, map[string]attr.Value{
		"rules": rulesList,
	})
	populatedWaf := types.ObjectValueMust(wafTypes, map[string]attr.Value{
		"mode":                          types.StringValue("ENABLED"),
		"type":                          types.StringValue("PREMIUM"),
		"paranoia_level":                types.StringValue("L2"),
		"allowed_http_versions":         populatedWafSet,
		"allowed_request_content_types": populatedWafSet,
		"allowed_http_methods":          populatedWafSet,
		"enabled_rule_ids":              populatedWafSet,
		"disabled_rule_ids":             populatedWafSet,
		"log_only_rule_ids":             populatedWafSet,
		"enabled_rule_group_ids":        populatedWafSet,
		"disabled_rule_group_ids":       populatedWafSet,
		"log_only_rule_group_ids":       populatedWafSet,
		"enabled_rule_collection_ids":   populatedWafSet,
		"disabled_rule_collection_ids":  populatedWafSet,
		"log_only_rule_collection_ids":  populatedWafSet,
	})

	expectedParanoiaLevel := cdnSdk.WafParanoiaLevel("L2")
	expectedWafConfig := cdnSdk.WafConfig{
		Mode:                       cdnSdk.WafMode("ENABLED"),
		Type:                       cdnSdk.WafType("PREMIUM"),
		ParanoiaLevel:              &expectedParanoiaLevel,
		AllowedHttpVersions:        []string{"rule1", "rule2"},
		AllowedRequestContentTypes: []string{"rule1", "rule2"},
		AllowedHttpMethods:         []string{"rule1", "rule2"},
		EnabledRuleIds:             []string{"rule1", "rule2"},
		DisabledRuleIds:            []string{"rule1", "rule2"},
		LogOnlyRuleIds:             []string{"rule1", "rule2"},
		EnabledRuleGroupIds:        []string{"rule1", "rule2"},
		DisabledRuleGroupIds:       []string{"rule1", "rule2"},
		LogOnlyRuleGroupIds:        []string{"rule1", "rule2"},
		EnabledRuleCollectionIds:   []string{"rule1", "rule2"},
		DisabledRuleCollectionIds:  []string{"rule1", "rule2"},
		LogOnlyRuleCollectionIds:   []string{"rule1", "rule2"},
	}

	modelFixture := func(mods ...func(*Model)) *Model {
		model := &Model{
			DistributionId: types.StringValue("test-distribution-id"),
			ProjectId:      types.StringValue("test-project-id"),
			Config:         config,
		}
		for _, mod := range mods {
			mod(model)
		}
		return model
	}

	tests := map[string]struct {
		Input    *Model
		Expected *cdnSdk.Config
		IsValid  bool
	}{
		"happy_path": {
			Input: modelFixture(),
			Expected: &cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					HttpBackend: &cdnSdk.HttpBackend{
						OriginRequestHeaders: map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: "https://www.mycoolapp.com",
						Type:      "http",
						Geofencing: map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
			},
			IsValid: true,
		},
		"happy_path_with_optimizer": {
			Input: modelFixture(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              optimizer,
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsTypes),
					"waf":                    types.ObjectNull(wafTypes),
					"tls":                    types.ObjectNull(tlsTypes),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					HttpBackend: &cdnSdk.HttpBackend{
						OriginRequestHeaders: map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: "https://www.mycoolapp.com",
						Type:      "http",
						Geofencing: map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:          []cdnSdk.Region{"EU", "US"},
				Optimizer:        cdnSdk.NewOptimizer(true),
				BlockedCountries: []string{"XX", "YY", "ZZ"},
			},
			IsValid: true,
		},
		"happy_path_with_tls": {
			Input: modelFixture(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":           backend,
					"regions":           regionsFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"blocked_countries": blockedCountriesFixture,
					"redirects":         types.ObjectNull(redirectsTypes),
					"waf":               types.ObjectNull(wafTypes),
					"tls": types.ObjectValueMust(tlsTypes, map[string]attr.Value{
						"enable_tls_10": types.BoolValue(true),
						"enable_tls_11": types.BoolValue(true),
					}),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					HttpBackend: &cdnSdk.HttpBackend{
						OriginRequestHeaders: map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: "https://www.mycoolapp.com",
						Type:      "http",
						Geofencing: map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				Tls: cdnSdk.TlsConfig{
					EnableTls10: true,
					EnableTls11: true,
				},
			},
			IsValid: true,
		},
		"happy_path_with_waf": {
			Input: modelFixture(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsTypes),
					"waf":                    populatedWaf,
					"tls":                    types.ObjectNull(tlsTypes),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					HttpBackend: &cdnSdk.HttpBackend{
						OriginRequestHeaders: map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: "https://www.mycoolapp.com",
						Type:      "http",
						Geofencing: map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				Waf:              expectedWafConfig,
			},
			IsValid: true,
		},
		"happy_path_with_redirects": {
			Input: modelFixture(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              redirectsConfigVal,
					"waf":                    types.ObjectNull(wafTypes),
					"tls":                    types.ObjectNull(tlsTypes),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					HttpBackend: &cdnSdk.HttpBackend{
						OriginRequestHeaders: map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: "https://www.mycoolapp.com",
						Type:      "http",
						Geofencing: map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				Redirects: &cdnSdk.RedirectConfig{
					Rules: []cdnSdk.RedirectRule{
						{
							Description:        cdnSdk.PtrString("Test redirect"),
							Enabled:            cdnSdk.PtrBool(true),
							TargetUrl:          "https://example.com/redirect",
							StatusCode:         301,
							RuleMatchCondition: cdnSdk.MATCHCONDITION_ALL.Ptr(),
							Matchers: []cdnSdk.Matcher{
								{
									Values:              []string{"/shop/*"},
									ValueMatchCondition: cdnSdk.MATCHCONDITION_ALL.Ptr(),
								},
							},
						},
					},
				},
			},
			IsValid: true,
		},
		"happy_path_bucket": {
			Input: modelFixture(func(m *Model) {
				creds := types.ObjectValueMust(backendCredentialsTypes, map[string]attr.Value{
					"access_key_id":     types.StringValue("my-access"),
					"secret_access_key": types.StringValue("my-secret"),
				})
				bucketBackend := types.ObjectValueMust(backendTypes, map[string]attr.Value{
					"type":                   types.StringValue("bucket"),
					"bucket_url":             types.StringValue("https://s3.example.com"),
					"region":                 types.StringValue("eu01"),
					"credentials":            creds,
					"origin_url":             types.StringNull(),
					"origin_request_headers": types.MapNull(types.StringType),
					"geofencing":             types.MapNull(geofencingTypes.ElemType),
				})
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                bucketBackend,
					"regions":                regionsFixture,
					"blocked_countries":      blockedCountriesFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"redirects":              types.ObjectNull(redirectsTypes),
					"waf":                    types.ObjectNull(wafTypes),
					"tls":                    types.ObjectNull(tlsTypes),
					"strip_response_cookies": types.BoolUnknown(),
					"forward_host_header":    types.BoolUnknown(),
				})
			}),
			Expected: &cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					BucketBackend: &cdnSdk.BucketBackend{
						Type:      "bucket",
						BucketUrl: "https://s3.example.com",
						Region:    "eu01",
						// Note: config does not return credentials

					},
				},
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
			},
			IsValid: true,
		},
		"happy_path_with_blocked_ips": {
			Input: modelFixture(func(m *Model) {
				m.Config = configFixture(func(v map[string]attr.Value) {
					v["backend"] = backend
					v["regions"] = regionsFixture
					v["blocked_countries"] = blockedCountriesFixture
					v["blocked_ips"] = types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("1.2.3.4"),
						types.StringValue("5.6.7.8"),
					})
				})
			}),
			Expected: &cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					HttpBackend: &cdnSdk.HttpBackend{
						OriginRequestHeaders: map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: "https://www.mycoolapp.com",
						Type:      "http",
						Geofencing: map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				BlockedIps:       []string{"1.2.3.4", "5.6.7.8"},
			},
			IsValid: true,
		},
		"happy_path_with_default_cache_duration": {
			Input: modelFixture(func(m *Model) {
				m.Config = configFixture(func(v map[string]attr.Value) {
					v["backend"] = backend
					v["regions"] = regionsFixture
					v["blocked_countries"] = blockedCountriesFixture
					v["default_cache_duration"] = types.StringValue("P1DT2H30M")
				})
			}),
			Expected: &cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					HttpBackend: &cdnSdk.HttpBackend{
						OriginRequestHeaders: map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: "https://www.mycoolapp.com",
						Type:      "http",
						Geofencing: map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:              []cdnSdk.Region{"EU", "US"},
				BlockedCountries:     []string{"XX", "YY", "ZZ"},
				DefaultCacheDuration: *cdnSdk.NewNullableString(cdnSdk.PtrString("P1DT2H30M")),
			},
			IsValid: true,
		},
		"happy_path_with_monthly_limit_bytes": {
			Input: modelFixture(func(m *Model) {
				m.Config = configFixture(func(v map[string]attr.Value) {
					v["backend"] = backend
					v["regions"] = regionsFixture
					v["blocked_countries"] = blockedCountriesFixture
					v["monthly_limit_bytes"] = types.Int64Value(1073741824)
				})
			}),
			Expected: &cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					HttpBackend: &cdnSdk.HttpBackend{
						OriginRequestHeaders: map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: "https://www.mycoolapp.com",
						Type:      "http",
						Geofencing: map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:           []cdnSdk.Region{"EU", "US"},
				BlockedCountries:  []string{"XX", "YY", "ZZ"},
				MonthlyLimitBytes: *cdnSdk.NewNullableInt64(cdnSdk.PtrInt64(1073741824)),
			},
			IsValid: true,
		},
		"sad_path_model_nil": {
			Input:    nil,
			Expected: nil,
			IsValid:  false,
		},
		"sad_path_config_error": {
			Input: modelFixture(func(m *Model) {
				m.Config = types.ObjectNull(configTypes)
			}),
			Expected: nil,
			IsValid:  false,
		},
	}

	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			res, err := convertConfig(context.Background(), tc.Input)
			if err != nil && tc.IsValid {
				t.Fatalf("Error converting model to create payload: %v", err)
			}
			if err == nil && !tc.IsValid {
				t.Fatalf("Should have failed")
			}
			if tc.IsValid {
				diff := cmp.Diff(res, tc.Expected,
					// The struct contains now a NullableString and NullableInt64.
					// Previously those were pointers which could be compared but the value of those
					// are unexported and therefore cmp cannot compare them.
					cmpopts.IgnoreUnexported(
						cdnSdk.NullableString{},
						cdnSdk.NullableInt64{},
					),
					cmpopts.EquateEmpty(),
				)
				if diff != "" {
					t.Fatalf("Create Payload not as expected: %s", diff)
				}
			}
		})
	}
}

func TestMapFields(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	headers := map[string]attr.Value{
		"testHeader0": types.StringValue("testHeaderValue0"),
		"testHeader1": types.StringValue("testHeaderValue1"),
	}
	originRequestHeaders := types.MapValueMust(types.StringType, headers)
	backend := types.ObjectValueMust(backendTypes, map[string]attr.Value{
		"type":                   types.StringValue("http"),
		"origin_url":             types.StringValue("https://www.mycoolapp.com"),
		"origin_request_headers": originRequestHeaders,
		"geofencing":             types.MapNull(geofencingTypes.ElemType),
		"bucket_url":             types.StringNull(),
		"region":                 types.StringNull(),
		"credentials":            types.ObjectNull(backendCredentialsTypes),
	})
	regions := []attr.Value{types.StringValue("EU"), types.StringValue("US")}
	regionsFixture := types.ListValueMust(types.StringType, regions)
	blockedCountries := []attr.Value{types.StringValue("XX"), types.StringValue("YY"), types.StringValue("ZZ")}
	blockedCountriesFixture := types.ListValueMust(types.StringType, blockedCountries)
	geofencingCountries := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("DE"), types.StringValue("BR")})
	geofencing := types.MapValueMust(geofencingTypes.ElemType, map[string]attr.Value{
		"test/": geofencingCountries,
	})
	geofencingInput := map[string][]string{"test/": {"DE", "BR"}}
	optimizer := types.ObjectValueMust(optimizerTypes, map[string]attr.Value{
		"enabled": types.BoolValue(true),
	})

	redirectsObjType, ok := configTypes["redirects"].(basetypes.ObjectType)
	if !ok {
		t.Fatalf("configTypes[\"redirects\"] is not of type basetypes.ObjectType")
	}
	redirectsAttrTypes := redirectsObjType.AttrTypes
	populatedWafSet := types.SetValueMust(types.StringType, []attr.Value{
		types.StringValue("rule1"),
		types.StringValue("rule2"),
	})
	emptyWafSet := types.SetValueMust(types.StringType, []attr.Value{})
	populatedWaf := types.ObjectValueMust(wafTypes, map[string]attr.Value{
		"mode":                          types.StringValue("ENABLED"),
		"type":                          types.StringValue("PREMIUM"),
		"paranoia_level":                types.StringValue("L2"),
		"allowed_http_versions":         populatedWafSet,
		"allowed_request_content_types": populatedWafSet,
		"allowed_http_methods":          populatedWafSet,
		"enabled_rule_ids":              populatedWafSet,
		"disabled_rule_ids":             populatedWafSet,
		"log_only_rule_ids":             populatedWafSet,
		"enabled_rule_group_ids":        populatedWafSet,
		"disabled_rule_group_ids":       populatedWafSet,
		"log_only_rule_group_ids":       populatedWafSet,
		"enabled_rule_collection_ids":   populatedWafSet,
		"disabled_rule_collection_ids":  populatedWafSet,
		"log_only_rule_collection_ids":  populatedWafSet,
	})

	expectedParanoiaLevel := cdnSdk.WafParanoiaLevel("L2")
	expectedWafConfig := cdnSdk.WafConfig{
		Mode:                       cdnSdk.WafMode("ENABLED"),
		Type:                       cdnSdk.WafType("PREMIUM"),
		ParanoiaLevel:              &expectedParanoiaLevel,
		AllowedHttpVersions:        []string{"rule1", "rule2"},
		AllowedRequestContentTypes: []string{"rule1", "rule2"},
		AllowedHttpMethods:         []string{"rule1", "rule2"},
		EnabledRuleIds:             []string{"rule1", "rule2"},
		DisabledRuleIds:            []string{"rule1", "rule2"},
		LogOnlyRuleIds:             []string{"rule1", "rule2"},
		EnabledRuleGroupIds:        []string{"rule1", "rule2"},
		DisabledRuleGroupIds:       []string{"rule1", "rule2"},
		LogOnlyRuleGroupIds:        []string{"rule1", "rule2"},
		EnabledRuleCollectionIds:   []string{"rule1", "rule2"},
		DisabledRuleCollectionIds:  []string{"rule1", "rule2"},
		LogOnlyRuleCollectionIds:   []string{"rule1", "rule2"},
	}
	defaultWaf := types.ObjectValueMust(wafTypes, map[string]attr.Value{
		"mode":                          types.StringValue("DISABLED"),
		"type":                          types.StringValue("FREE"),
		"paranoia_level":                types.StringNull(),
		"allowed_http_versions":         types.SetNull(types.StringType),
		"allowed_request_content_types": types.SetNull(types.StringType),
		"allowed_http_methods":          types.SetNull(types.StringType),
		"enabled_rule_ids":              types.SetNull(types.StringType),
		"disabled_rule_ids":             types.SetNull(types.StringType),
		"log_only_rule_ids":             types.SetNull(types.StringType),
		"enabled_rule_group_ids":        types.SetNull(types.StringType),
		"disabled_rule_group_ids":       types.SetNull(types.StringType),
		"log_only_rule_group_ids":       types.SetNull(types.StringType),
		"enabled_rule_collection_ids":   types.SetNull(types.StringType),
		"disabled_rule_collection_ids":  types.SetNull(types.StringType),
		"log_only_rule_collection_ids":  types.SetNull(types.StringType),
	})

	// defaultWafEmptyRules is the WAF model produced by a plan in which the user explicitly set
	// every rule list to an empty set (e.g. disabled_rule_ids = []) while the WAF stays disabled.
	defaultWafEmptyRules := types.ObjectValueMust(wafTypes, map[string]attr.Value{
		"mode":                          types.StringValue("DISABLED"),
		"type":                          types.StringValue("FREE"),
		"paranoia_level":                types.StringNull(),
		"allowed_http_versions":         types.SetNull(types.StringType),
		"allowed_request_content_types": types.SetNull(types.StringType),
		"allowed_http_methods":          types.SetNull(types.StringType),
		"enabled_rule_ids":              emptyWafSet,
		"disabled_rule_ids":             emptyWafSet,
		"log_only_rule_ids":             emptyWafSet,
		"enabled_rule_group_ids":        emptyWafSet,
		"disabled_rule_group_ids":       emptyWafSet,
		"log_only_rule_group_ids":       emptyWafSet,
		"enabled_rule_collection_ids":   emptyWafSet,
		"disabled_rule_collection_ids":  emptyWafSet,
		"log_only_rule_collection_ids":  emptyWafSet,
	})

	defaultTls := types.ObjectValueMust(tlsTypes, map[string]attr.Value{
		"enable_tls_10": types.BoolValue(false),
		"enable_tls_11": types.BoolValue(false),
	})
	config := createTestConfig(map[string]attr.Value{
		"backend":                backend,
		"regions":                regionsFixture,
		"blocked_countries":      blockedCountriesFixture,
		"optimizer":              types.ObjectNull(optimizerTypes),
		"redirects":              types.ObjectNull(redirectsAttrTypes),
		"waf":                    defaultWaf,
		"tls":                    defaultTls,
		"strip_response_cookies": types.BoolValue(false),
		"forward_host_header":    types.BoolValue(false),
	})

	redirectsInput := &cdnSdk.RedirectConfig{
		Rules: []cdnSdk.RedirectRule{
			{
				Description:        cdnSdk.PtrString("Test redirect"),
				Enabled:            cdnSdk.PtrBool(true),
				TargetUrl:          "https://example.com/redirect",
				StatusCode:         301,
				RuleMatchCondition: cdnSdk.MATCHCONDITION_ALL.Ptr(),
				Matchers: []cdnSdk.Matcher{
					{
						Values:              []string{"/shop/*"},
						ValueMatchCondition: cdnSdk.MATCHCONDITION_ALL.Ptr(),
					},
				},
			},
		},
	}

	matcherValuesExpected := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("/shop/*"),
	})
	matcherValExpected := types.ObjectValueMust(matcherTypes, map[string]attr.Value{
		"values":                matcherValuesExpected,
		"value_match_condition": types.StringValue("ALL"),
	})
	matchersListExpected := types.ListValueMust(types.ObjectType{AttrTypes: matcherTypes}, []attr.Value{matcherValExpected})

	ruleValExpected := types.ObjectValueMust(redirectRuleTypes, map[string]attr.Value{
		"description":          types.StringValue("Test redirect"),
		"enabled":              types.BoolValue(true),
		"target_url":           types.StringValue("https://example.com/redirect"),
		"status_code":          types.Int32Value(301),
		"rule_match_condition": types.StringValue("ALL"),
		"matchers":             matchersListExpected,
	})
	rulesListExpected := types.ListValueMust(types.ObjectType{AttrTypes: redirectRuleTypes}, []attr.Value{ruleValExpected})

	redirectsConfigExpected := types.ObjectValueMust(redirectsTypes, map[string]attr.Value{
		"rules": rulesListExpected,
	})

	emtpyErrorsList := types.ListValueMust(types.StringType, []attr.Value{})
	managedDomain := types.ObjectValueMust(domainTypes, map[string]attr.Value{
		"name":   types.StringValue("test.stackit-cdn.com"),
		"status": types.StringValue("ACTIVE"),
		"type":   types.StringValue("managed"),
		"errors": types.ListValueMust(types.StringType, []attr.Value{}),
	})
	domains := types.ListValueMust(types.ObjectType{AttrTypes: domainTypes}, []attr.Value{managedDomain})
	expectedModel := func(mods ...func(*Model)) *Model {
		model := &Model{
			ID:             types.StringValue("test-project-id,test-distribution-id"),
			DistributionId: types.StringValue("test-distribution-id"),
			ProjectId:      types.StringValue("test-project-id"),
			Config:         config,
			Status:         types.StringValue("ACTIVE"),
			CreatedAt:      types.StringValue(createdAt.String()),
			UpdatedAt:      types.StringValue(updatedAt.String()),
			Errors:         emtpyErrorsList,
			Domains:        domains,
		}
		for _, mod := range mods {
			mod(model)
		}
		return model
	}
	distributionFixture := func(mods ...func(*cdnSdk.Distribution)) *cdnSdk.Distribution {
		distribution := &cdnSdk.Distribution{
			Config: cdnSdk.Config{
				Backend: cdnSdk.ConfigBackend{
					HttpBackend: &cdnSdk.HttpBackend{
						OriginRequestHeaders: map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: "https://www.mycoolapp.com",
						Type:      "http",
					},
				},
				Regions:          []cdnSdk.Region{"EU", "US"},
				BlockedCountries: []string{"XX", "YY", "ZZ"},
				Optimizer:        nil,
				Waf: cdnSdk.WafConfig{
					Mode: cdnSdk.WAFMODE_DISABLED,
					Type: cdnSdk.WAFTYPE_FREE,
				},
				Tls: cdnSdk.TlsConfig{
					EnableTls10: false,
					EnableTls11: false,
				},
			},
			CreatedAt: createdAt,
			Domains: []cdnSdk.Domain{
				{
					Name:   "test.stackit-cdn.com",
					Status: cdnSdk.DOMAINSTATUS_ACTIVE,
					Type:   "managed",
				},
			},
			Id:        "test-distribution-id",
			ProjectId: "test-project-id",
			Status:    "ACTIVE",
			UpdatedAt: updatedAt,
		}
		for _, mod := range mods {
			mod(distribution)
		}
		return distribution
	}
	// define old state with the secrets
	oldCreds := types.ObjectValueMust(backendCredentialsTypes, map[string]attr.Value{
		"access_key_id":     types.StringValue("old-access"),
		"secret_access_key": types.StringValue("old-secret"),
	})
	bucketBackendOld := types.ObjectValueMust(backendTypes, map[string]attr.Value{
		"type":                   types.StringValue("bucket"),
		"bucket_url":             types.StringValue("https://s3.example.com"),
		"region":                 types.StringValue("eu01"),
		"credentials":            oldCreds,
		"origin_url":             types.StringNull(),
		"origin_request_headers": types.MapNull(types.StringType),
		"geofencing":             types.MapNull(geofencingTypes.ElemType),
	})
	configOld := createTestConfig(map[string]attr.Value{
		"backend":                bucketBackendOld,
		"regions":                regionsFixture,
		"blocked_countries":      blockedCountriesFixture,
		"optimizer":              types.ObjectNull(optimizerTypes),
		"redirects":              types.ObjectNull(redirectsAttrTypes),
		"waf":                    types.ObjectNull(wafTypes),
		"tls":                    types.ObjectNull(tlsTypes),
		"strip_response_cookies": types.BoolUnknown(),
		"forward_host_header":    types.BoolUnknown(),
	})
	tests := map[string]struct {
		Input        *cdnSdk.Distribution
		Expected     *Model
		InitialState *Model
		IsValid      bool
	}{
		"happy_path": {
			Expected: expectedModel(),
			Input:    distributionFixture(),
			IsValid:  true,
		},
		"happy_path_with_optimizer": {
			Expected: expectedModel(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              optimizer,
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    defaultWaf,
					"tls":                    defaultTls,
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.Optimizer = &cdnSdk.Optimizer{
					Enabled: true,
				}
			}),
			IsValid: true,
		},
		"happy_path_with_geofencing": {
			Expected: expectedModel(func(m *Model) {
				backendWithGeofencing := types.ObjectValueMust(backendTypes, map[string]attr.Value{
					"type":                   types.StringValue("http"),
					"origin_url":             types.StringValue("https://www.mycoolapp.com"),
					"origin_request_headers": originRequestHeaders,
					"geofencing":             geofencing,
					"bucket_url":             types.StringNull(),
					"region":                 types.StringNull(),
					"credentials":            types.ObjectNull(backendCredentialsTypes),
				})
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backendWithGeofencing,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    defaultWaf,
					"tls":                    defaultTls,
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.Backend.HttpBackend.Geofencing = geofencingInput
			}),
			IsValid: true,
		},
		"happy_path_with_redirects": {
			Expected: expectedModel(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              redirectsConfigExpected,
					"waf":                    defaultWaf,
					"tls":                    defaultTls,
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.Redirects = redirectsInput
			}),
			IsValid: true,
		},
		"happy_path_status_error": {
			Expected: expectedModel(func(m *Model) {
				m.Status = types.StringValue("ERROR")
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Status = "ERROR"
			}),
			IsValid: true,
		},
		"happy_path_with_waf": {
			Expected: expectedModel(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    populatedWaf,
					"tls":                    defaultTls,
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.Waf = expectedWafConfig
			}),
			IsValid: true,
		},
		// Regression test for https://github.com/stackitcloud/terraform-provider-stackit/issues/1630:
		// when the API omits a WAF rule-list field (returns nil) but the model (plan) holds an
		// explicitly empty set, the state must keep the empty set instead of becoming null.
		"waf_rule_list_api_nil_model_empty_set": {
			InitialState: expectedModel(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    defaultWafEmptyRules,
					"tls":                    defaultTls,
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			// The API omits all rule-list fields (nil), as happens when the WAF is disabled.
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.Waf = cdnSdk.WafConfig{
					Mode: cdnSdk.WAFMODE_DISABLED,
					Type: cdnSdk.WAFTYPE_FREE,
				}
			}),
			Expected: expectedModel(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    defaultWafEmptyRules,
					"tls":                    defaultTls,
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			IsValid: true,
		},
		// When the API returns an explicit empty list, the state must be an empty set regardless
		// of the prior model value.
		"waf_rule_list_api_empty_list": {
			InitialState: expectedModel(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    populatedWaf,
					"tls":                    defaultTls,
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.Waf = cdnSdk.WafConfig{
					Mode:                       cdnSdk.WAFMODE_ENABLED,
					Type:                       cdnSdk.WAFTYPE_PREMIUM,
					EnabledRuleIds:             []string{},
					DisabledRuleIds:            []string{},
					LogOnlyRuleIds:             []string{},
					EnabledRuleGroupIds:        []string{},
					DisabledRuleGroupIds:       []string{},
					LogOnlyRuleGroupIds:        []string{},
					EnabledRuleCollectionIds:   []string{},
					DisabledRuleCollectionIds:  []string{},
					LogOnlyRuleCollectionIds:   []string{},
					AllowedHttpVersions:        []string{"rule1", "rule2"},
					AllowedRequestContentTypes: []string{"rule1", "rule2"},
					AllowedHttpMethods:         []string{"rule1", "rule2"},
				}
			}),
			Expected: expectedModel(func(m *Model) {
				emptyRulesWaf := types.ObjectValueMust(wafTypes, map[string]attr.Value{
					"mode":                          types.StringValue("ENABLED"),
					"type":                          types.StringValue("PREMIUM"),
					"paranoia_level":                types.StringNull(),
					"allowed_http_versions":         populatedWafSet,
					"allowed_request_content_types": populatedWafSet,
					"allowed_http_methods":          populatedWafSet,
					"enabled_rule_ids":              emptyWafSet,
					"disabled_rule_ids":             emptyWafSet,
					"log_only_rule_ids":             emptyWafSet,
					"enabled_rule_group_ids":        emptyWafSet,
					"disabled_rule_group_ids":       emptyWafSet,
					"log_only_rule_group_ids":       emptyWafSet,
					"enabled_rule_collection_ids":   emptyWafSet,
					"disabled_rule_collection_ids":  emptyWafSet,
					"log_only_rule_collection_ids":  emptyWafSet,
				})
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                backend,
					"regions":                regionsFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"blocked_countries":      blockedCountriesFixture,
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    emptyRulesWaf,
					"tls":                    defaultTls,
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			IsValid: true,
		},
		"happy_path_with_tls_and_strip_response_and_cookies_forward": {
			Expected: expectedModel(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":           backend,
					"regions":           regionsFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"blocked_countries": blockedCountriesFixture,
					"redirects":         types.ObjectNull(redirectsAttrTypes),
					"waf":               defaultWaf,
					"tls": types.ObjectValueMust(tlsTypes, map[string]attr.Value{
						"enable_tls_10": types.BoolValue(true),
						"enable_tls_11": types.BoolValue(true),
					}),
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.Tls = cdnSdk.TlsConfig{
					EnableTls10: true,
					EnableTls11: true,
				}
				d.Config.ForwardHostHeader = false
				d.Config.StripResponseCookies = false
			}),
			IsValid: true,
		},
		"happy_path_custom_domain": {
			Expected: expectedModel(func(m *Model) {
				managedDomain := types.ObjectValueMust(domainTypes, map[string]attr.Value{
					"name":   types.StringValue("test.stackit-cdn.com"),
					"status": types.StringValue("ACTIVE"),
					"type":   types.StringValue("managed"),
					"errors": types.ListValueMust(types.StringType, []attr.Value{}),
				})
				customDomain := types.ObjectValueMust(domainTypes, map[string]attr.Value{
					"name":   types.StringValue("mycoolapp.info"),
					"status": types.StringValue("ACTIVE"),
					"type":   types.StringValue("custom"),
					"errors": types.ListValueMust(types.StringType, []attr.Value{}),
				})
				domains := types.ListValueMust(types.ObjectType{AttrTypes: domainTypes}, []attr.Value{managedDomain, customDomain})
				m.Domains = domains
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Domains = []cdnSdk.Domain{
					{
						Name:   "test.stackit-cdn.com",
						Status: cdnSdk.DOMAINSTATUS_ACTIVE,
						Type:   "managed",
					},
					{
						Name:   "mycoolapp.info",
						Status: cdnSdk.DOMAINSTATUS_ACTIVE,
						Type:   "custom",
					},
				}
			}),
			IsValid: true,
		},
		"happy_path_bucket_restore_creds": {
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.Backend = cdnSdk.ConfigBackend{
					BucketBackend: &cdnSdk.BucketBackend{
						Type:      "bucket",
						BucketUrl: "https://s3.example.com",
						Region:    "eu01",
					},
				}
			}),
			InitialState: expectedModel(func(m *Model) {
				m.Config = configOld
			}),
			Expected: expectedModel(func(m *Model) {
				m.Config = createTestConfig(map[string]attr.Value{
					"backend":                bucketBackendOld,
					"regions":                regionsFixture,
					"blocked_countries":      blockedCountriesFixture,
					"optimizer":              types.ObjectNull(optimizerTypes),
					"redirects":              types.ObjectNull(redirectsAttrTypes),
					"waf":                    defaultWaf,
					"tls":                    defaultTls,
					"strip_response_cookies": types.BoolValue(false),
					"forward_host_header":    types.BoolValue(false),
				})
			}),
			IsValid: true,
		},
		"happy_path_with_blocked_ips": {
			Expected: expectedModel(func(m *Model) {
				m.Config = configFixture(func(v map[string]attr.Value) {
					v["backend"] = backend
					v["regions"] = regionsFixture
					v["blocked_countries"] = blockedCountriesFixture
					v["waf"] = defaultWaf
					v["tls"] = defaultTls
					v["strip_response_cookies"] = types.BoolValue(false)
					v["forward_host_header"] = types.BoolValue(false)
					v["blocked_ips"] = types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("1.2.3.4"),
						types.StringValue("5.6.7.8"),
					})
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.BlockedIps = []string{"1.2.3.4", "5.6.7.8"}
			}),
			IsValid: true,
		},
		"happy_path_with_default_cache_duration": {
			Expected: expectedModel(func(m *Model) {
				m.Config = configFixture(func(v map[string]attr.Value) {
					v["backend"] = backend
					v["regions"] = regionsFixture
					v["blocked_countries"] = blockedCountriesFixture
					v["waf"] = defaultWaf
					v["tls"] = defaultTls
					v["strip_response_cookies"] = types.BoolValue(false)
					v["forward_host_header"] = types.BoolValue(false)
					v["default_cache_duration"] = types.StringValue("P1DT2H30M")
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.DefaultCacheDuration = *cdnSdk.NewNullableString(cdnSdk.PtrString("P1DT2H30M"))
			}),
			IsValid: true,
		},
		"happy_path_with_monthly_limit_bytes": {
			Expected: expectedModel(func(m *Model) {
				m.Config = configFixture(func(v map[string]attr.Value) {
					v["backend"] = backend
					v["regions"] = regionsFixture
					v["blocked_countries"] = blockedCountriesFixture
					v["waf"] = defaultWaf
					v["tls"] = defaultTls
					v["strip_response_cookies"] = types.BoolValue(false)
					v["forward_host_header"] = types.BoolValue(false)
					v["monthly_limit_bytes"] = types.Int64Value(1073741824)
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.MonthlyLimitBytes = *cdnSdk.NewNullableInt64(cdnSdk.PtrInt64(1073741824))
			}),
			IsValid: true,
		},
		"sad_path_distribution_nil": {
			Expected: nil,
			Input:    nil,
			IsValid:  false,
		},
		"sad_path_project_id_missing": {
			Expected: expectedModel(),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.ProjectId = ""
			}),
			IsValid: false,
		},
		"sad_path_distribution_id_missing": {
			Expected: expectedModel(),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Id = ""
			}),
			IsValid: false,
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			model := &Model{}
			if tc.InitialState != nil {
				model = tc.InitialState
			} else {
				model.Config = types.ObjectNull(configTypes)
			}

			err := mapFields(context.Background(), tc.Input, model)
			if err != nil && tc.IsValid {
				t.Fatalf("Error mapping fields: %v", err)
			}
			if err == nil && !tc.IsValid {
				t.Fatalf("Should have failed")
			}
			if tc.IsValid {
				diff := cmp.Diff(model, tc.Expected)
				if diff != "" {
					t.Fatalf("Create Payload not as expected: %s", diff)
				}
			}
		})
	}
}

// TestValidateCountryCode tests the validateCountryCode function with a variety of inputs.
func TestValidateCountryCode(t *testing.T) {
	testCases := []struct {
		name          string
		inputCountry  string
		wantOutput    string
		expectError   bool
		expectedError string
	}{
		// Happy Path
		{
			name:         "Valid lowercase",
			inputCountry: "us",
			wantOutput:   "US",
			expectError:  false,
		},
		{
			name:         "Valid uppercase",
			inputCountry: "DE",
			wantOutput:   "DE",
			expectError:  false,
		},
		{
			name:         "Valid mixed case",
			inputCountry: "cA",
			wantOutput:   "CA",
			expectError:  false,
		},
		{
			name:         "Valid country code FR",
			inputCountry: "fr",
			wantOutput:   "FR",
			expectError:  false,
		},

		// Error Scenarios
		{
			name:          "Invalid length - too short",
			inputCountry:  "a",
			wantOutput:    "",
			expectError:   true,
			expectedError: "country code must be exactly 2 characters long",
		},
		{
			name:          "Invalid length - too long",
			inputCountry:  "USA",
			wantOutput:    "",
			expectError:   true,
			expectedError: "country code must be exactly 2 characters long",
		},
		{
			name:          "Invalid characters - contains number",
			inputCountry:  "U1",
			wantOutput:    "",
			expectError:   true,
			expectedError: "country code 'U1' must consist of two alphabetical letters (A-Z or a-z)",
		},
		{
			name:          "Invalid characters - contains symbol",
			inputCountry:  "D!",
			wantOutput:    "",
			expectError:   true,
			expectedError: "country code 'D!' must consist of two alphabetical letters (A-Z or a-z)",
		},
		{
			name:          "Invalid characters - both are numbers",
			inputCountry:  "42",
			wantOutput:    "",
			expectError:   true,
			expectedError: "country code '42' must consist of two alphabetical letters (A-Z or a-z)",
		},
		{
			name:          "Empty string",
			inputCountry:  "",
			wantOutput:    "",
			expectError:   true,
			expectedError: "country code must be exactly 2 characters long",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotOutput, err := validateCountryCode(tc.inputCountry)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected an error for input '%s', but got none", tc.inputCountry)
				} else if err.Error() != tc.expectedError {
					t.Errorf("for input '%s', expected error '%s', but got '%s'", tc.inputCountry, tc.expectedError, err.Error())
				}
				if gotOutput != "" {
					t.Errorf("expected empty string on error, but got '%s'", gotOutput)
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error for input '%s', but got: %v", tc.inputCountry, err)
				}
				if gotOutput != tc.wantOutput {
					t.Errorf("for input '%s', expected output '%s', but got '%s'", tc.inputCountry, tc.wantOutput, gotOutput)
				}
			}
		})
	}
}

// wafObjForModifyPlan builds a waf object value for ModifyPlan tests, allowing null sets.
func wafObjForModifyPlan(mode string, disabledIds, enabledCollectionIds types.Set) types.Object {
	return types.ObjectValueMust(wafTypes, map[string]attr.Value{
		"mode":                          types.StringValue(mode),
		"type":                          types.StringValue("FREE"),
		"paranoia_level":                types.StringNull(),
		"allowed_http_versions":         types.SetNull(types.StringType),
		"allowed_request_content_types": types.SetNull(types.StringType),
		"allowed_http_methods":          types.SetNull(types.StringType),
		"enabled_rule_ids":              types.SetNull(types.StringType),
		"disabled_rule_ids":             disabledIds,
		"log_only_rule_ids":             types.SetNull(types.StringType),
		"enabled_rule_group_ids":        types.SetNull(types.StringType),
		"disabled_rule_group_ids":       types.SetNull(types.StringType),
		"log_only_rule_group_ids":       types.SetNull(types.StringType),
		"enabled_rule_collection_ids":   enabledCollectionIds,
		"disabled_rule_collection_ids":  types.SetNull(types.StringType),
		"log_only_rule_collection_ids":  types.SetNull(types.StringType),
	})
}

func modifyPlanModel(updatedAt types.String, waf types.Object) *Model {
	return &Model{
		ID:             types.StringValue("test-project-id,test-distribution-id"),
		DistributionId: types.StringValue("test-distribution-id"),
		ProjectId:      types.StringValue("test-project-id"),
		Status:         types.StringValue("ACTIVE"),
		CreatedAt:      types.StringValue("2026-09-04 09:00:00 +0000 UTC"),
		UpdatedAt:      updatedAt,
		Errors:         types.ListValueMust(types.StringType, []attr.Value{}),
		Domains:        types.ListValueMust(types.ObjectType{AttrTypes: domainTypes}, []attr.Value{}),
		Config: configFixture(func(v map[string]attr.Value) {
			v["backend"] = types.ObjectValueMust(backendTypes, map[string]attr.Value{
				"type":                   types.StringValue("http"),
				"origin_url":             types.StringValue("https://www.mycoolapp.com"),
				"origin_request_headers": types.MapNull(types.StringType),
				"geofencing":             types.MapNull(geofencingTypes.ElemType),
				"bucket_url":             types.StringNull(),
				"region":                 types.StringNull(),
				"credentials":            types.ObjectNull(backendCredentialsTypes),
			})
			v["regions"] = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("EU")})
			v["waf"] = waf
			v["tls"] = types.ObjectNull(tlsTypes)
			v["strip_response_cookies"] = types.BoolValue(false)
			v["forward_host_header"] = types.BoolValue(false)
		}),
	}
}

func modifyPlanRequest(ctx context.Context, schemaResp *resource.SchemaResponse, state, plan, config *Model) (resource.ModifyPlanRequest, *resource.ModifyPlanResponse) {
	req := resource.ModifyPlanRequest{}
	req.State = tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tftypes.DynamicPseudoType, nil)}
	req.Plan = tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tftypes.DynamicPseudoType, nil)}
	req.State.Set(ctx, state)
	req.Plan.Set(ctx, plan)

	// tfsdk.Config has no Set method; marshal the config model via a scratch plan and reuse its Raw.
	configScratch := tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tftypes.DynamicPseudoType, nil)}
	configScratch.Set(ctx, config)
	req.Config = tfsdk.Config{Schema: schemaResp.Schema, Raw: configScratch.Raw}

	resp := &resource.ModifyPlanResponse{}
	resp.Plan = tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tftypes.DynamicPseudoType, nil)}
	resp.Plan.Set(ctx, plan)
	return req, resp
}

// TestModifyPlan verifies that server-managed attributes (updated_at and unconfigured WAF rule
// lists) are planned as unknown during updates, and that no-op plans are left untouched to avoid
// perpetual diffs. Regression test for the inconsistent-result errors reported in
// https://github.com/stackitcloud/terraform-provider-stackit/issues/1630.
func TestModifyPlan(t *testing.T) {
	ctx := context.Background()
	r := &distributionResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	tsOld := types.StringValue("2026-09-04 09:02:37 +0000 UTC")

	disabledPopulated := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("@builtin/crs/request/942140")})
	disabledEmpty := types.SetValueMust(types.StringType, []attr.Value{})
	nullSet := types.SetNull(types.StringType)

	// helper to read a nested waf set from a plan model's config
	getWafSet := func(t *testing.T, m Model, name string) types.Set {
		t.Helper()
		var cfg distributionConfig
		diags := m.Config.As(ctx, &cfg, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			t.Fatalf("reading config: %v", diags)
		}
		var w wafConfig
		diags = cfg.Waf.As(ctx, &w, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			t.Fatalf("reading waf: %v", diags)
		}
		switch name {
		case "disabled_rule_ids":
			return w.DisabledRuleIds
		case "enabled_rule_collection_ids":
			return w.EnabledRuleCollectionIds
		}
		t.Fatalf("unknown waf attr %q", name)
		return types.SetNull(types.StringType)
	}

	t.Run("update via waf removal: updated_at and unconfigured lists become unknown", func(t *testing.T) {
		// state: disabled_rule_ids populated, enabled_rule_collection_ids null (never configured)
		state := modifyPlanModel(tsOld, wafObjForModifyPlan("ENABLED", disabledPopulated, nullSet))
		// plan: EmptyOnRemoval already emptied disabled_rule_ids; enabled_rule_collection_ids null
		plan := modifyPlanModel(tsOld, wafObjForModifyPlan("ENABLED", disabledEmpty, nullSet))
		// config: user removed disabled_rule_ids (null), enabled_rule_collection_ids null
		config := modifyPlanModel(tsOld, wafObjForModifyPlan("ENABLED", nullSet, nullSet))

		req, resp := modifyPlanRequest(ctx, schemaResp, state, plan, config)
		r.ModifyPlan(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ModifyPlan diagnostics: %v", resp.Diagnostics)
		}

		var got Model
		resp.Plan.Get(ctx, &got)

		if !got.UpdatedAt.IsUnknown() {
			t.Errorf("expected updated_at to be unknown, got %v", got.UpdatedAt)
		}
		if gotSet := getWafSet(t, got, "enabled_rule_collection_ids"); !gotSet.IsUnknown() {
			t.Errorf("expected enabled_rule_collection_ids to be unknown, got %v", gotSet)
		}
		if gotSet := getWafSet(t, got, "disabled_rule_ids"); !gotSet.Equal(disabledEmpty) {
			t.Errorf("expected disabled_rule_ids to remain empty set, got %v", gotSet)
		}
	})

	t.Run("no-op plan: left untouched (no perpetual diff)", func(t *testing.T) {
		state := modifyPlanModel(tsOld, wafObjForModifyPlan("ENABLED", disabledPopulated, nullSet))
		plan := modifyPlanModel(tsOld, wafObjForModifyPlan("ENABLED", disabledPopulated, nullSet))
		config := modifyPlanModel(tsOld, wafObjForModifyPlan("ENABLED", disabledPopulated, nullSet))

		req, resp := modifyPlanRequest(ctx, schemaResp, state, plan, config)
		r.ModifyPlan(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ModifyPlan diagnostics: %v", resp.Diagnostics)
		}

		var got Model
		resp.Plan.Get(ctx, &got)

		if !got.UpdatedAt.Equal(tsOld) {
			t.Errorf("expected updated_at to remain %v, got %v", tsOld, got.UpdatedAt)
		}
		if gotSet := getWafSet(t, got, "enabled_rule_collection_ids"); !gotSet.IsNull() {
			t.Errorf("expected enabled_rule_collection_ids to remain null, got %v", gotSet)
		}
	})

	t.Run("create: skipped", func(t *testing.T) {
		plan := modifyPlanModel(types.StringUnknown(), wafObjForModifyPlan("ENABLED", disabledPopulated, nullSet))
		config := modifyPlanModel(types.StringUnknown(), wafObjForModifyPlan("ENABLED", disabledPopulated, nullSet))

		req := resource.ModifyPlanRequest{}
		req.State = tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)} // null state = create
		req.Plan = tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tftypes.DynamicPseudoType, nil)}
		req.Plan.Set(ctx, plan)

		configScratch := tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tftypes.DynamicPseudoType, nil)}
		configScratch.Set(ctx, config)
		req.Config = tfsdk.Config{Schema: schemaResp.Schema, Raw: configScratch.Raw}

		resp := &resource.ModifyPlanResponse{}
		resp.Plan = tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tftypes.DynamicPseudoType, nil)}
		resp.Plan.Set(ctx, plan)

		r.ModifyPlan(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ModifyPlan diagnostics: %v", resp.Diagnostics)
		}

		var got Model
		resp.Plan.Get(ctx, &got)
		if !got.UpdatedAt.IsUnknown() {
			t.Errorf("expected updated_at to remain unknown on create, got %v", got.UpdatedAt)
		}
	})
}
