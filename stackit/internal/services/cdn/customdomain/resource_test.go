package cdn

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
)

func TestMapFields(t *testing.T) {
	emtpyErrorsList := types.ListValueMust(types.StringType, []attr.Value{})
	certAttributes := map[string]attr.Value{
		"type":        types.StringValue("custom"),
		"version":     types.Int64Value(3),
		"certificate": types.StringNull(),
		"private_key": types.StringNull(),
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
		Input       *cdn.CustomDomain
		Certificate interface{}
		Expected    *CustomDomainModel
		IsValid     bool
	}{
		"happy_path": {
			Expected: expectedModel(func(m *CustomDomainModel) {
				m.Certificate = certificateObj
			}),
			Input:       customDomainFixture(),
			IsValid:     true,
			Certificate: getRespCustom,
		},
		"happy_path_status_error": {
			Expected: expectedModel(func(m *CustomDomainModel) {
				m.Status = types.StringValue("ERROR")
				m.Certificate = certificateObj
			}),
			Input: customDomainFixture(func(d *cdn.CustomDomain) {
				d.Status = cdn.DOMAINSTATUS_ERROR.Ptr()
			}),
			IsValid:     true,
			Certificate: getRespCustom,
		},
		"sad_path_custom_domain_nil": {
			Expected:    expectedModel(),
			Input:       nil,
			IsValid:     false,
			Certificate: getRespCustom,
		},
		"sad_path_name_missing": {
			Expected: expectedModel(),
			Input: customDomainFixture(func(d *cdn.CustomDomain) {
				d.Name = nil
			}),
			IsValid:     false,
			Certificate: getRespCustom,
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			if tn == "happy_path" {
				fmt.Println("AS")
			}
			model := &CustomDomainModel{}
			model.DistributionId = tc.Expected.DistributionId
			model.ProjectId = tc.Expected.ProjectId
			err := mapCustomDomainFields(tc.Input, model, tc.Certificate)
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
func TestBuildCertificatePayload(t *testing.T) {
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
		"success_managed_type": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"type":        types.StringValue("managed"),
						"version":     types.Int64Null(),
						"certificate": types.StringNull(),
						"private_key": types.StringNull(),
					},
				),
			},
			expectedPayload: &cdn.PutCustomDomainPayloadCertificate{
				PutCustomDomainManagedCertificate: cdn.NewPutCustomDomainManagedCertificate("managed"),
			},
			expectDiagnostics: false,
		},
		"success_nil_certificate_block": {
			model: &CustomDomainModel{
				Certificate: types.ObjectNull(certificateTypes),
			},
			expectedPayload: &cdn.PutCustomDomainPayloadCertificate{
				PutCustomDomainManagedCertificate: cdn.NewPutCustomDomainManagedCertificate("managed"),
			},
			expectDiagnostics: false,
		},
		"success_custom_type": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"type":        types.StringValue("custom"),
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
		"fail_custom_type_missing_cert": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"type":        types.StringValue("custom"),
						"version":     types.Int64Null(),
						"certificate": types.StringNull(), // Missing certificate
						"private_key": types.StringValue(keyPEM),
					},
				),
			},
			expectDiagnostics:   true,
			expectedDiagSummary: "Missing Certificate",
		},
		"fail_custom_type_missing_key": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"type":        types.StringValue("custom"),
						"version":     types.Int64Null(),
						"certificate": types.StringValue(certPEM),
						"private_key": types.StringNull(), // Missing key
					},
				),
			},
			expectDiagnostics:   true,
			expectedDiagSummary: "Missing Private Key",
		},
		"fail_custom_type_invalid_cert_pem": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"type":        types.StringValue("custom"),
						"version":     types.Int64Null(),
						"certificate": types.StringValue("this-is-not-pem"), // Invalid PEM
						"private_key": types.StringValue(keyPEM),
					},
				),
			},
			expectDiagnostics:   true,
			expectedDiagSummary: "Invalid Certificate Format",
		},
		"fail_custom_type_invalid_key_pem": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"type":        types.StringValue("custom"),
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
