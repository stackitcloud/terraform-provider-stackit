package cdn

import (
	"context"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
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

	tests := map[string]struct {
		Input          *cdn.GetCustomDomainResponse
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
		},
		"happy_path_managed_cert": {
			Expected: expectedModel(func(m *CustomDomainModel) {
				m.Certificate = types.ObjectNull(certificateTypes)
			}),
			Input: customDomainFixture(func(gcdr *cdn.GetCustomDomainResponse) {
				gcdr.Certificate = getRespManaged
			}),
			IsValid:      true,
			InitialModel: expectedModel(func(m *CustomDomainModel) { m.Certificate = types.ObjectNull(certificateTypes) }),
		},
		"happy_path_status_error": {
			Expected: expectedModel(func(m *CustomDomainModel) {
				m.Status = types.StringValue("ERROR")
				m.Certificate = certificateObj
			}),
			Input: customDomainFixture(func(d *cdn.GetCustomDomainResponse) {
				d.CustomDomain.Status = cdn.DOMAINSTATUS_ERROR.Ptr()
			}),
			IsValid: true,
			InitialModel: expectedModel(func(m *CustomDomainModel) {
				m.Certificate = basetypes.NewObjectValueMust(certificateTypes, map[string]attr.Value{
					"certificate": types.StringValue(dummyCert),
					"private_key": types.StringValue(dummyKey),
					"version":     types.Int64Null(),
				})
			}),
		},
		"sad_path_custom_domain_nil": {
			Expected:     expectedModel(),
			Input:        nil,
			IsValid:      false,
			InitialModel: &CustomDomainModel{},
		},
		"sad_path_name_missing": {
			Expected: expectedModel(),
			Input: customDomainFixture(func(d *cdn.GetCustomDomainResponse) {
				d.CustomDomain.Name = nil
			}),
			IsValid:      false,
			InitialModel: &CustomDomainModel{},
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			model := tc.InitialModel
			model.DistributionId = tc.Expected.DistributionId
			model.ProjectId = tc.Expected.ProjectId
			err := mapCustomDomainResourceFields(tc.Input, model)
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
func TestBuildCertificatePayload(t *testing.T) {
	// Redefine certificateTypes locally for testing, matching the updated schema
	certificateTypes := map[string]attr.Type{
		"version":     types.Int64Type,
		"certificate": types.StringType,
		"private_key": types.StringType,
	}
	organization := fmt.Sprintf("organization-%s", uuid.NewString())
	cert, key := makeCertAndKey(t, organization)
	certPEM := string(cert)
	keyPEM := string(key)
	certBase64 := base64.StdEncoding.EncodeToString(cert)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	tests := map[string]struct {
		model           *CustomDomainModel
		expectedPayload *cdn.PutCustomDomainPayloadCertificate
		expectErr       bool
		expectedErrMsg  string
	}{
		"success_managed_when_certificate_block_is_nil": {
			model: &CustomDomainModel{
				Certificate: types.ObjectNull(certificateTypes),
			},
			expectedPayload: &cdn.PutCustomDomainPayloadCertificate{
				PutCustomDomainManagedCertificate: cdn.NewPutCustomDomainManagedCertificate("managed"),
			},
			expectErr: false,
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
			expectErr: false,
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
			expectErr:      true,
			expectedErrMsg: "invalid certificate or private key. Please check if the string of the public certificate and private key in PEM format",
		},

		"success_managed_when_certificate_attributes_are_nil": {
			model: &CustomDomainModel{
				Certificate: basetypes.NewObjectValueMust(
					certificateTypes,
					map[string]attr.Value{
						"version":     types.Int64Null(),
						"certificate": types.StringNull(),
						"private_key": types.StringNull(),
					},
				),
			},
			expectErr:      true,
			expectedErrMsg: `"certificate" and "private_key" must be set`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			payload, err := buildCertificatePayload(context.Background(), tt.model)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected err, but got none")
				}
				if err.Error() != tt.expectedErrMsg {
					t.Fatalf("expected err '%s', got '%s'", tt.expectedErrMsg, err.Error())
				}
				return // Test ends here for failing cases
			}

			if err != nil {
				t.Fatalf("did not expect err, but got: %s", err.Error())
			}

			if diff := cmp.Diff(tt.expectedPayload, payload); diff != "" {
				t.Errorf("payload mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
