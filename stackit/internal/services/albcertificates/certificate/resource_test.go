package certificate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	certSdk "github.com/stackitcloud/stackit-sdk-go/services/certificates/v2api"
)

const (
	projectID      = "b8c3fbaa-3ab4-4a8e-9584-de22453d046f"
	region         = "eu01"
	certName       = "example-cert-2"
	certID         = "example-cert-2-v1-dfa816b3184f63f43d918ea5f9493f5359f6c2404b69afbb0b60fb1af69d0bc0"
	tfID           = projectID + "," + region + "," + certID
	certPrivateKey = "dummy-private-pem-key"
	certPublicKey  = "dummy-public-pem-key"
)

func fixtureModel(mods ...func(m *Model)) *Model {
	resp := &Model{
		DataSourceModel: DataSourceModel{
			Id:        types.StringValue(tfID),
			ProjectId: types.StringValue(projectID),
			Region:    types.StringValue(region),
			CertID:    types.StringValue(certID),
			Name:      types.StringValue(certName),
			PublicKey: types.StringValue(certPublicKey),
		},
		PrivateKey: types.StringValue(certPrivateKey),
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func fixtureModelNull(mods ...func(m *Model)) *Model {
	resp := &Model{
		DataSourceModel: DataSourceModel{
			Id:        types.StringNull(),
			ProjectId: types.StringNull(),
			Name:      types.StringNull(),
			Region:    types.StringNull(),
		},
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func fixtureCertificate(mods ...func(c *certSdk.GetCertificateResponse)) *certSdk.GetCertificateResponse {
	resp := &certSdk.GetCertificateResponse{
		Id:        utils.Ptr(certID),
		Name:      utils.Ptr(certName),
		PublicKey: utils.Ptr(certPublicKey),
		Region:    utils.Ptr(region),
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *certSdk.CreateCertificatePayload
		isValid     bool
	}{
		{
			description: "valid",
			input:       fixtureModel(),
			expected: &certSdk.CreateCertificatePayload{
				Name:       utils.Ptr(certName),
				PrivateKey: utils.Ptr(certPrivateKey),
				PublicKey:  utils.Ptr(certPublicKey),
			},
			isValid: true,
		},
		{
			description: "valid empty",
			input:       fixtureModelNull(),
			expected:    &certSdk.CreateCertificatePayload{},
			isValid:     true,
		},
		{
			description: "model nil",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	tests := []struct {
		description string
		input       *certSdk.GetCertificateResponse
		output      *Model
		region      string
		expected    *Model
		isValid     bool
	}{
		{
			description: "valid full model",
			input:       fixtureCertificate(),
			output: &Model{
				DataSourceModel: DataSourceModel{ProjectId: types.StringValue(projectID)},
			},
			region: testRegion,
			expected: fixtureModel(func(m *Model) {
				m.PrivateKey = types.StringNull()
			}),
			isValid: true,
		},
		{
			description: "error input nil",
			input:       nil,
			output: &Model{
				DataSourceModel: DataSourceModel{ProjectId: types.StringValue(projectID)},
			},
			region:   testRegion,
			expected: fixtureModel(),
			isValid:  false,
		},
		{
			description: "error model nil",
			input:       fixtureCertificate(),
			output:      nil,
			region:      testRegion,
			expected:    fixtureModel(),
			isValid:     false,
		},
		{
			description: "error no cert ID",
			input: fixtureCertificate(func(m *certSdk.GetCertificateResponse) {
				m.Id = nil
			}),
			output: &Model{
				DataSourceModel: DataSourceModel{
					ProjectId: types.StringValue(projectID),
					CertID:    types.StringValue(""),
				},
			},
			region:   testRegion,
			expected: fixtureModel(),
			isValid:  false,
		},
		{
			description: "valid name in model",
			input:       fixtureCertificate(),
			output: &Model{
				DataSourceModel: DataSourceModel{
					ProjectId: types.StringValue(projectID),
					CertID:    types.StringValue(certID),
				},
			},
			region: testRegion,
			expected: fixtureModel(func(m *Model) {
				m.PrivateKey = types.StringNull()
			}),
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(tt.input, tt.output, tt.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
