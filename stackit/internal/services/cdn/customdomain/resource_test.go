package cdn

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
)

func TestMapFields(t *testing.T) {
	// Redefine certificateTypes locally for testing, matching the updated schema
	certificateTypes := map[string]attr.Type{
		"version":     types.Int64Type,
		"certificate": types.StringType,
		"private_key": types.StringType,
	}

	const dummyCert = "dummy-cert-pem"
	const dummyKey = "dummy-key-pem"

	emtpyErrorsList := types.ListValueMust(types.StringType, []attr.Value{})

	// Expected object when a custom certificate is returned
	certAttributes := map[string]attr.Value{
		"version":     types.Int64Value(3),
		"certificate": types.StringValue(dummyCert),
		"private_key": types.StringValue(dummyKey),
	}
	certificateObj, _ := types.ObjectValue(certificateTypes, certAttributes)

	expectedModel := func(mods ...func(*CustomDomainModel)) *CustomDomainModel {
		model := &CustomDomainModel{
			ID:             types.StringValue("test-project-id,test-distribution-id,https://testdomain.com"),
			DistributionId: types.StringValue("test-distribution-id"),
			ProjectId:      types.StringValue("test-project-id"),
			Status:         types.StringValue("ACTIVE"),
			Errors:         emtpyErrorsList,
			Certificate:    types.ObjectUnknown(certificateTypes),
		}
		for _, mod := range mods {
			mod(model)
		}
		return model
	}

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

	customDomainFixture := func(mods ...func(*cdn.CustomDomain)) *cdn.CustomDomain {
		distribution := &cdn.CustomDomain{
			Errors: &[]cdn.StatusError{},
			Name:   cdn.PtrString("https://testdomain.com"),
			Status: cdn.DOMAINSTATUS_ACTIVE.Ptr(),
		}
		for _, mod := range mods {
			mod(distribution)
		}
		return distribution
	}

	tests := map[string]struct {
		Input          *cdn.CustomDomain
		Certificate    interface{}
		Expected       *CustomDomainModel
		InitialModel   *CustomDomainModel
		IsValid        bool
		SkipInitialNil bool
	}{
		"happy_path_custom_cert": {
			Expected: expectedModel(func(m *CustomDomainModel) {
				m.Certificate = certificateObj
			}),
			Input:   customDomainFixture(),
			IsValid: true,
			InitialModel: expectedModel(func(m *CustomDomainModel) {
				m.Certificate = basetypes.NewObjectValueMust(certificateTypes, map[string]attr.Value{
					"certificate": types.StringValue(dummyCert),
					"private_key": types.StringValue(dummyKey),
					"version":     types.Int64Null(),
				})
			}),
			Certificate: getRespCustom,
		},
		"happy_path_managed_cert": {
			Expected: expectedModel(func(m *CustomDomainModel) {
				m.Certificate = types.ObjectNull(certificateTypes)
			}),
			Input:        customDomainFixture(),
			IsValid:      true,
			InitialModel: expectedModel(func(m *CustomDomainModel) { m.Certificate = types.ObjectNull(certificateTypes) }),
			Certificate:  getRespManaged,
		},
		"happy_path_status_error": {
			Expected: expectedModel(func(m *CustomDomainModel) {
				m.Status = types.StringValue("ERROR")
				m.Certificate = certificateObj
			}),
			Input: customDomainFixture(func(d *cdn.CustomDomain) {
				d.Status = cdn.DOMAINSTATUS_ERROR.Ptr()
			}),
			IsValid: true,
			InitialModel: expectedModel(func(m *CustomDomainModel) {
				m.Certificate = basetypes.NewObjectValueMust(certificateTypes, map[string]attr.Value{
					"certificate": types.StringValue(dummyCert),
					"private_key": types.StringValue(dummyKey),
					"version":     types.Int64Null(),
				})
			}),
			Certificate: getRespCustom,
		},
		"sad_path_custom_domain_nil": {
			Expected:     expectedModel(),
			Input:        nil,
			IsValid:      false,
			InitialModel: &CustomDomainModel{},
			Certificate:  getRespCustom,
		},
		"sad_path_name_missing": {
			Expected: expectedModel(),
			Input: customDomainFixture(func(d *cdn.CustomDomain) {
				d.Name = nil
			}),
			IsValid:      false,
			InitialModel: &CustomDomainModel{},
			Certificate:  getRespCustom,
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			model := tc.InitialModel
			model.DistributionId = tc.Expected.DistributionId
			model.ProjectId = tc.Expected.ProjectId
			err := mapCustomDomainFields(context.Background(), tc.Input, model, tc.Certificate)
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
func TestBuildCertificatePayload(t *testing.T) {
	// Redefine certificateTypes locally for testing, matching the updated schema
	certificateTypes := map[string]attr.Type{
		"version":     types.Int64Type,
		"certificate": types.StringType,
		"private_key": types.StringType,
	}

	// Dummy PEM and their expected raw Base64 counterparts for testing.
	const certPEM = `-----BEGIN CERTIFICATE-----
Y2VydGlmaWNhdGVfZGF0YQ==
-----END CERTIFICATE-----`
	const certBase64 = "Y2VydGlmaWNhdGVfZGF0YQ==" // "certificate_data"

	const keyPEM = `-----BEGIN PRIVATE KEY-----
cHJpdmF0ZV9rZXlfZGF0YQ==
-----END PRIVATE KEY-----`
	const keyBase64 = "cHJpdmF0ZV9rZXlfZGF0YQ==" // "private_key_data"

	tests := map[string]struct {
		model               *CustomDomainModel
		expectedPayload     *cdn.PutCustomDomainPayloadCertificate
		expectDiagnostics   bool
		expectedDiagSummary string
	}{
		"success_managed_when_certificate_block_is_nil": {
			model: &CustomDomainModel{
				Certificate: types.ObjectNull(certificateTypes),
			},
			expectedPayload: &cdn.PutCustomDomainPayloadCertificate{
				PutCustomDomainManagedCertificate: cdn.NewPutCustomDomainManagedCertificate("managed"),
			},
			expectDiagnostics: false,
		},
		"success_custom_certificate": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"version":     types.Int64Null(),
						"certificate": types.StringValue(certPEM),
						"private_key": types.StringValue(keyPEM),
					},
				),
			},
			expectedPayload: &cdn.PutCustomDomainPayloadCertificate{
				PutCustomDomainCustomCertificate: cdn.NewPutCustomDomainCustomCertificate(certBase64, keyBase64, "custom"),
			},
			expectDiagnostics: false,
		},
		"fail_custom_missing_cert_value": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"version":     types.Int64Null(),
						"certificate": types.StringValue(""), // Empty certificate
						"private_key": types.StringValue(keyPEM),
					},
				),
			},
			expectDiagnostics:   true,
			expectedDiagSummary: "Invalid Certificate Format",
		},
		"fail_custom_missing_key_value": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"version":     types.Int64Null(),
						"certificate": types.StringValue(certPEM),
						"private_key": types.StringValue(""), // Empty key
					},
				),
			},
			expectDiagnostics:   true,
			expectedDiagSummary: "Invalid Private Key Format",
		},
		"fail_custom_invalid_cert_pem": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"version":     types.Int64Null(),
						"certificate": types.StringValue("this-is-not-pem"), // Invalid PEM
						"private_key": types.StringValue(keyPEM),
					},
				),
			},
			expectDiagnostics:   true,
			expectedDiagSummary: "Invalid Certificate Format",
		},
		"fail_custom_invalid_key_pem": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"version":     types.Int64Null(),
						"certificate": types.StringValue(certPEM),
						"private_key": types.StringValue("this-is-not-pem-either"), // Invalid PEM
					},
				),
			},
			expectDiagnostics:   true,
			expectedDiagSummary: "Invalid Private Key Format",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			payload, diags := buildCertificatePayload(context.Background(), tt.model)
			if tt.expectDiagnostics {
				if !diags.HasError() {
					t.Fatalf("expected diagnostics, but got none")
				}
				if summary := diags.Errors()[0].Summary(); summary != tt.expectedDiagSummary {
					t.Fatalf("expected diagnostic summary '%s', got '%s'", tt.expectedDiagSummary, summary)
				}
				return // Test ends here for failing cases
			}

			if diags.HasError() {
				t.Fatalf("did not expect diagnostics, but got: %v", diags)
			}

			if diff := cmp.Diff(tt.expectedPayload, payload); diff != "" {
				t.Errorf("payload mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
