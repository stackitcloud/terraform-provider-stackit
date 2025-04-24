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
	backend := types.ObjectValueMust(backendTypes, map[string]attr.Value{
		"type":                   types.StringValue("http"),
		"origin_url":             types.StringValue("https://www.mycoolapp.com"),
		"origin_request_headers": originRequestHeaders,
	})
	regions := []attr.Value{types.StringValue("EU"), types.StringValue("US")}
	regionsFixture := types.ListValueMust(types.StringType, regions)
	config := types.ObjectValueMust(configTypes, map[string]attr.Value{
		"backend": backend,
		"regions": regionsFixture,
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
				OriginUrl: cdn.PtrString("https://www.mycoolapp.com"),
				Regions:   &[]cdn.Region{"EU", "US"},
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
	backend := types.ObjectValueMust(backendTypes, map[string]attr.Value{
		"type":                   types.StringValue("http"),
		"origin_url":             types.StringValue("https://www.mycoolapp.com"),
		"origin_request_headers": originRequestHeaders,
	})
	regions := []attr.Value{types.StringValue("EU"), types.StringValue("US")}
	regionsFixture := types.ListValueMust(types.StringType, regions)
	config := types.ObjectValueMust(configTypes, map[string]attr.Value{
		"backend": backend,
		"regions": regionsFixture,
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
					},
				},
				Regions: &[]cdn.Region{"EU", "US"},
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
	})
	regions := []attr.Value{types.StringValue("EU"), types.StringValue("US")}
	regionsFixture := types.ListValueMust(types.StringType, regions)
	config := types.ObjectValueMust(configTypes, map[string]attr.Value{
		"backend": backend,
		"regions": regionsFixture,
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
			Config: &cdn.Config{Backend: &cdn.ConfigBackend{
				HttpBackend: &cdn.HttpBackend{
					OriginRequestHeaders: &map[string]string{
						"testHeader0": "testHeaderValue0",
						"testHeader1": "testHeaderValue1",
					},
					OriginUrl: cdn.PtrString("https://www.mycoolapp.com"),
					Type:      cdn.PtrString("http"),
				},
			},
				Regions: &[]cdn.Region{"EU", "US"},
			},
			CreatedAt: &createdAt,
			Domains: &[]cdn.Domain{
				{
					Name:   cdn.PtrString("test.stackit-cdn.com"),
					Status: cdn.DOMAINSTATUS_ACTIVE.Ptr(),
					Type:   cdn.PtrString("managed"),
				},
			},
			Id:        cdn.PtrString("test-distribution-id"),
			ProjectId: cdn.PtrString("test-project-id"),
			Status:    cdn.PtrString("ACTIVE"),
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
		"happy_path_status_error": {
			Expected: expectedModel(func(m *Model) {
				m.Status = types.StringValue("ERROR")
			}),
			Input: distributionFixture(func(d *cdn.Distribution) {
				d.Status = cdn.PtrString("ERROR")
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
						Type:   cdn.PtrString("managed"),
					},
					{
						Name:   cdn.PtrString("mycoolapp.info"),
						Status: cdn.DOMAINSTATUS_ACTIVE.Ptr(),
						Type:   cdn.PtrString("custom"),
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
			err := mapFields(tc.Input, model)
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
