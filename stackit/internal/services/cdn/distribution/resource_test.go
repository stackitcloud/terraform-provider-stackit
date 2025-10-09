package cdn

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
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
		Expected *cdn.CreateDistributionPayload
		IsValid  bool
	}{
		"happy_path": {
			Input: modelFixture(),
			Expected: &cdn.CreateDistributionPayload{
				OriginRequestHeaders: &map[string]string{
					"testHeader0": "testHeaderValue0",
					"testHeader1": "testHeaderValue1",
				},
				OriginUrl:        cdn.PtrString("https://www.mycoolapp.com"),
				Regions:          &[]cdn.Region{"EU", "US"},
				BlockedCountries: &[]string{"XX", "YY", "ZZ"},
				Geofencing: &map[string][]string{
					"https://de.mycoolapp.com": {"DE", "FR"},
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
				})
			}),
			Expected: &cdn.CreateDistributionPayload{
				OriginRequestHeaders: &map[string]string{
					"testHeader0": "testHeaderValue0",
					"testHeader1": "testHeaderValue1",
				},
				OriginUrl:        cdn.PtrString("https://www.mycoolapp.com"),
				Regions:          &[]cdn.Region{"EU", "US"},
				Optimizer:        cdn.NewOptimizer(true),
				BlockedCountries: &[]string{"XX", "YY", "ZZ"},
				Geofencing: &map[string][]string{
					"https://de.mycoolapp.com": {"DE", "FR"},
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
		Expected *cdn.Config
		IsValid  bool
	}{
		"happy_path": {
			Input: modelFixture(),
			Expected: &cdn.Config{
				Backend: &cdn.ConfigBackend{
					HttpBackend: &cdn.HttpBackend{
						OriginRequestHeaders: &map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: cdn.PtrString("https://www.mycoolapp.com"),
						Type:      cdn.PtrString("http"),
						Geofencing: &map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:          &[]cdn.Region{"EU", "US"},
				BlockedCountries: &[]string{"XX", "YY", "ZZ"},
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
				})
			}),
			Expected: &cdn.Config{
				Backend: &cdn.ConfigBackend{
					HttpBackend: &cdn.HttpBackend{
						OriginRequestHeaders: &map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: cdn.PtrString("https://www.mycoolapp.com"),
						Type:      cdn.PtrString("http"),
						Geofencing: &map[string][]string{
							"https://de.mycoolapp.com": {"DE", "FR"},
						},
					},
				},
				Regions:          &[]cdn.Region{"EU", "US"},
				Optimizer:        cdn.NewOptimizer(true),
				BlockedCountries: &[]string{"XX", "YY", "ZZ"},
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
				diff := cmp.Diff(res, tc.Expected)
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
	config := types.ObjectValueMust(configTypes, map[string]attr.Value{
		"backend":           backend,
		"regions":           regionsFixture,
		"blocked_countries": blockedCountriesFixture,
		"optimizer":         types.ObjectNull(optimizerTypes),
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
	distributionFixture := func(mods ...func(*cdn.Distribution)) *cdn.Distribution {
		distribution := &cdn.Distribution{
			Config: &cdn.Config{
				Backend: &cdn.ConfigBackend{
					HttpBackend: &cdn.HttpBackend{
						OriginRequestHeaders: &map[string]string{
							"testHeader0": "testHeaderValue0",
							"testHeader1": "testHeaderValue1",
						},
						OriginUrl: cdn.PtrString("https://www.mycoolapp.com"),
						Type:      cdn.PtrString("http"),
					},
				},
				Regions:          &[]cdn.Region{"EU", "US"},
				BlockedCountries: &[]string{"XX", "YY", "ZZ"},
				Optimizer:        nil,
			},
			CreatedAt: &createdAt,
			Domains: &[]cdn.Domain{
				{
					Name:   cdn.PtrString("test.stackit-cdn.com"),
					Status: cdn.DOMAINSTATUS_ACTIVE.Ptr(),
					Type:   cdn.DOMAINTYPE_MANAGED.Ptr(),
				},
			},
			Id:        cdn.PtrString("test-distribution-id"),
			ProjectId: cdn.PtrString("test-project-id"),
			Status:    cdn.DISTRIBUTIONSTATUS_ACTIVE.Ptr(),
			UpdatedAt: &updatedAt,
		}
		for _, mod := range mods {
			mod(distribution)
		}
		return distribution
	}
	tests := map[string]struct {
		Input    *cdn.Distribution
		Expected *Model
		IsValid  bool
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
				})
			}),
			Input: distributionFixture(func(d *cdn.Distribution) {
				d.Config.Optimizer = &cdn.Optimizer{
					Enabled: cdn.PtrBool(true),
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
				})
				m.Config = types.ObjectValueMust(configTypes, map[string]attr.Value{
					"backend":           backendWithGeofencing,
					"regions":           regionsFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
					"blocked_countries": blockedCountriesFixture,
				})
			}),
			Input: distributionFixture(func(d *cdn.Distribution) {
				d.Config.Backend.HttpBackend.Geofencing = &geofencingInput
			}),
			IsValid: true,
		},
		"happy_path_status_error": {
			Expected: expectedModel(func(m *Model) {
				m.Status = types.StringValue("ERROR")
			}),
			Input: distributionFixture(func(d *cdn.Distribution) {
				d.Status = cdn.DISTRIBUTIONSTATUS_ERROR.Ptr()
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
			Input: distributionFixture(func(d *cdn.Distribution) {
				d.Domains = &[]cdn.Domain{
					{
						Name:   cdn.PtrString("test.stackit-cdn.com"),
						Status: cdn.DOMAINSTATUS_ACTIVE.Ptr(),
						Type:   cdn.DOMAINTYPE_MANAGED.Ptr(),
					},
					{
						Name:   cdn.PtrString("mycoolapp.info"),
						Status: cdn.DOMAINSTATUS_ACTIVE.Ptr(),
						Type:   cdn.DOMAINTYPE_CUSTOM.Ptr(),
					},
				}
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
			Input: distributionFixture(func(d *cdn.Distribution) {
				d.ProjectId = nil
			}),
			IsValid: false,
		},
		"sad_path_distribution_id_missing": {
			Expected: expectedModel(),
			Input: distributionFixture(func(d *cdn.Distribution) {
				d.Id = nil
			}),
			IsValid: false,
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			model := &Model{}
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
