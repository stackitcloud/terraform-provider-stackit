package cdn

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
)

func TestMapDataSourceFields(t *testing.T) {
	// Define certificateTypes for the data source schema
	certificateDataSourceTypes := map[string]attr.Type{
		"version": types.Int64Type,
	}

	emtpyErrorsList := types.ListValueMust(types.StringType, []attr.Value{})

	// Expected certificate object when a custom certificate is returned
	certAttributes := map[string]attr.Value{
		"version": types.Int64Value(3),
	}
	certificateObj, _ := types.ObjectValue(certificateDataSourceTypes, certAttributes)

	// Helper to create expected model instances
	expectedModel := func(mods ...func(*customDomainDataSourceModel)) *customDomainDataSourceModel {
		model := &customDomainDataSourceModel{
			ID:             types.StringValue("test-project-id,test-distribution-id,https://testdomain.com"),
			DistributionId: types.StringValue("test-distribution-id"),
			ProjectId:      types.StringValue("test-project-id"),
			Name:           types.StringValue("https://testdomain.com"),
			Status:         types.StringValue("ACTIVE"),
			Errors:         emtpyErrorsList,
			Certificate:    types.ObjectUnknown(certificateDataSourceTypes),
		}
		for _, mod := range mods {
			mod(model)
		}
		return model
	}

	// API response fixtures for custom and managed certificates
	customType := "custom"
	customVersion := int64(3)
	getRespCustom := cdn.GetCustomDomainResponseGetCertificateAttributeType(&cdn.GetCustomDomainResponseCertificate{
		GetCustomDomainCustomCertificate: &cdn.GetCustomDomainCustomCertificate{
			Type:    &customType,
			Version: &customVersion,
		},
	})

	managedType := "managed"
	getRespManaged := cdn.GetCustomDomainResponseGetCertificateAttributeType(&cdn.GetCustomDomainResponseCertificate{
		GetCustomDomainManagedCertificate: &cdn.GetCustomDomainManagedCertificate{
			Type: &managedType,
		},
	})

	// Helper to create API response fixtures
	customDomainFixture := func(mods ...func(*cdn.GetCustomDomainResponse)) *cdn.GetCustomDomainResponse {
		distribution := &cdn.CustomDomain{
			Errors: &[]cdn.StatusError{},
			Name:   cdn.PtrString("https://testdomain.com"),
			Status: cdn.DOMAINSTATUS_ACTIVE.Ptr(),
		}
		customDomainResponse := &cdn.GetCustomDomainResponse{
			CustomDomain: distribution,
			Certificate:  getRespCustom,
		}

		for _, mod := range mods {
			mod(customDomainResponse)
		}
		return customDomainResponse
	}

	// Test cases
	tests := map[string]struct {
		Input    *cdn.GetCustomDomainResponse
		Expected *customDomainDataSourceModel
		IsValid  bool
	}{
		"happy_path_custom_cert": {
			Expected: expectedModel(func(m *customDomainDataSourceModel) {
				m.Certificate = certificateObj
			}),
			Input:   customDomainFixture(),
			IsValid: true,
		},
		"happy_path_managed_cert": {
			Expected: expectedModel(func(m *customDomainDataSourceModel) {
				m.Certificate = types.ObjectNull(certificateDataSourceTypes)
			}),
			Input: customDomainFixture(func(gcdr *cdn.GetCustomDomainResponse) {
				gcdr.Certificate = getRespManaged
			}),
			IsValid: true,
		},
		"happy_path_status_error": {
			Expected: expectedModel(func(m *customDomainDataSourceModel) {
				m.Status = types.StringValue("ERROR")
				m.Certificate = certificateObj
			}),
			Input: customDomainFixture(func(d *cdn.GetCustomDomainResponse) {
				d.CustomDomain.Status = cdn.DOMAINSTATUS_ERROR.Ptr()
			}),
			IsValid: true,
		},
		"sad_path_response_nil": {
			Expected: expectedModel(),
			Input:    nil,
			IsValid:  false,
		},
		"sad_path_name_missing": {
			Expected: expectedModel(),
			Input: customDomainFixture(func(d *cdn.GetCustomDomainResponse) {
				d.CustomDomain.Name = nil
			}),
			IsValid: false,
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			model := &customDomainDataSourceModel{}
			err := mapCustomDomainDataSourceFields(tc.Input, model, "test-project-id", "test-distribution-id")

			if err != nil && tc.IsValid {
				t.Fatalf("Error mapping fields: %v", err)
			}
			if err == nil && !tc.IsValid {
				t.Fatalf("Should have failed")
			}
			if tc.IsValid {
				diff := cmp.Diff(tc.Expected, model)
				if diff != "" {
					t.Fatalf("Mapped model not as expected (-want +got):\n%s", diff)
				}
			}
		})
	}
}
