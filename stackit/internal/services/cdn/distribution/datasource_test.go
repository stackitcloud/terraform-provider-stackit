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

func TestMapDataSourceFields(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	headers := map[string]attr.Value{
		"testHeader0": types.StringValue("testHeaderValue0"),
		"testHeader1": types.StringValue("testHeaderValue1"),
	}
	originRequestHeaders := types.MapValueMust(types.StringType, headers)
	backend := types.ObjectValueMust(dataSourceBackendTypes, map[string]attr.Value{
		"type":                   types.StringValue("http"),
		"origin_url":             types.StringValue("https://www.mycoolapp.com"),
		"origin_request_headers": originRequestHeaders,
		"geofencing":             types.MapNull(geofencingTypes.ElemType),
		"bucket_url":             types.StringNull(),
		"region":                 types.StringNull(),
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
	config := types.ObjectValueMust(dataSourceConfigTypes, map[string]attr.Value{
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

	bucketBackendExpected := types.ObjectValueMust(dataSourceBackendTypes, map[string]attr.Value{
		"type":                   types.StringValue("bucket"),
		"bucket_url":             types.StringValue("https://s3.example.com"),
		"region":                 types.StringValue("eu01"),
		"origin_url":             types.StringNull(),
		"origin_request_headers": types.MapNull(types.StringType),
		"geofencing":             types.MapNull(geofencingTypes.ElemType),
	})
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
				m.Config = types.ObjectValueMust(dataSourceConfigTypes, map[string]attr.Value{
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
		"happy_path_bucket": {
			Input: distributionFixture(func(d *cdn.Distribution) {
				d.Config.Backend = &cdn.ConfigBackend{
					BucketBackend: &cdn.BucketBackend{
						Type:      cdn.PtrString("bucket"),
						BucketUrl: cdn.PtrString("https://s3.example.com"),
						Region:    cdn.PtrString("eu01"),
					},
				}
			}),
			Expected: expectedModel(func(m *Model) {
				m.Config = types.ObjectValueMust(dataSourceConfigTypes, map[string]attr.Value{
					"backend":           bucketBackendExpected,
					"regions":           regionsFixture,
					"blocked_countries": blockedCountriesFixture,
					"optimizer":         types.ObjectNull(optimizerTypes),
				})
			}),
			IsValid: true,
		},
		"happy_path_with_geofencing": {
			Expected: expectedModel(func(m *Model) {
				backendWithGeofencing := types.ObjectValueMust(dataSourceBackendTypes, map[string]attr.Value{
					"type":                   types.StringValue("http"),
					"origin_url":             types.StringValue("https://www.mycoolapp.com"),
					"origin_request_headers": originRequestHeaders,
					"geofencing":             geofencing,
					"bucket_url":             types.StringNull(),
					"region":                 types.StringNull(),
				})
				m.Config = types.ObjectValueMust(dataSourceConfigTypes, map[string]attr.Value{
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

			err := mapDataSourceFields(context.Background(), tc.Input, model)
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
