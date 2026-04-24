package cdn

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	cdnSdk "github.com/stackitcloud/stackit-sdk-go/services/cdn/v1api"
)

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

	config := types.ObjectValueMust(configTypes, map[string]attr.Value{
		"backend":           backend,
		"regions":           regionsFixture,
		"blocked_countries": blockedCountriesFixture,
		"optimizer":         types.ObjectNull(optimizerTypes),
		"redirects":         types.ObjectNull(redirectsTypes),
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
			},
			IsValid: true,
		},
		"happy_path_with_optimizer": {
			Input: modelFixture(func(m *Model) {
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           backend,
					"regions":           regionsFixture,
					"optimizer":         optimizer,
					"blocked_countries": blockedCountriesFixture,
					"redirects":         types.ObjectNull(redirectsTypes),
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
				Regions:          []cdnSdk.Region{"EU", "US"},
				Optimizer:        cdnSdk.NewOptimizer(true),
				BlockedCountries: []string{"XX", "YY", "ZZ"},
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
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           backend,
					"regions":           regionsFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"blocked_countries": blockedCountriesFixture,
					"redirects":         redirectsConfigVal,
				})
			}),
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
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           bucketBackend,
					"regions":           regionsFixture, // reusing the existing one
					"blocked_countries": blockedCountriesFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"redirects":         types.ObjectNull(redirectsTypes),
				})
			}),
			Expected: &cdnSdk.CreateDistributionPayload{
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

				diff := cmp.Diff(res, tc.Expected)
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

	config := types.ObjectValueMust(configTypes, map[string]attr.Value{
		"backend":           backend,
		"regions":           regionsFixture,
		"optimizer":         types.ObjectNull(optimizerTypes),
		"blocked_countries": blockedCountriesFixture,
		"redirects":         types.ObjectNull(redirectsTypes),
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
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           backend,
					"regions":           regionsFixture,
					"optimizer":         optimizer,
					"blocked_countries": blockedCountriesFixture,
					"redirects":         types.ObjectNull(redirectsTypes),
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
		"happy_path_with_redirects": {
			Input: modelFixture(func(m *Model) {
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           backend,
					"regions":           regionsFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"blocked_countries": blockedCountriesFixture,
					"redirects":         redirectsConfigVal,
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
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           bucketBackend,
					"regions":           regionsFixture,
					"blocked_countries": blockedCountriesFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"redirects":         types.ObjectNull(redirectsTypes),
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
					))
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

	config := types.ObjectValueMust(configTypes, map[string]attr.Value{
		"backend":           backend,
		"regions":           regionsFixture,
		"blocked_countries": blockedCountriesFixture,
		"optimizer":         types.ObjectNull(optimizerTypes),
		"redirects":         types.ObjectNull(redirectsAttrTypes),
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
	configOld := types.ObjectValueMust(configTypes, map[string]attr.Value{
		"backend":           bucketBackendOld,
		"regions":           regionsFixture,
		"blocked_countries": blockedCountriesFixture,
		"optimizer":         types.ObjectNull(optimizerTypes),
		"redirects":         types.ObjectNull(redirectsAttrTypes),
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
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           backend,
					"regions":           regionsFixture,
					"optimizer":         optimizer,
					"blocked_countries": blockedCountriesFixture,
					"redirects":         types.ObjectNull(redirectsAttrTypes),
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
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           backendWithGeofencing,
					"regions":           regionsFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"blocked_countries": blockedCountriesFixture,
					"redirects":         types.ObjectNull(redirectsAttrTypes),
				})
			}),
			Input: distributionFixture(func(d *cdnSdk.Distribution) {
				d.Config.Backend.HttpBackend.Geofencing = geofencingInput
			}),
			IsValid: true,
		},
		"happy_path_with_redirects": {
			Expected: expectedModel(func(m *Model) {
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           backend,
					"regions":           regionsFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"blocked_countries": blockedCountriesFixture,
					"redirects":         redirectsConfigExpected,
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
				m.Config = configOld
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
