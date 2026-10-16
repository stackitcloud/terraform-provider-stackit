package certcheck

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	observabilitySdk "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *observabilitySdk.CertCheckChildResponse
		expected    Model
		isValid     bool
	}{
		{
			description: "ok",
			input: &observabilitySdk.CertCheckChildResponse{
				Id:     "cid",
				Source: "tcp://www.example.com:443",
			},
			expected: Model{
				Id:          types.StringValue("pid,iid,cid"),
				ProjectId:   types.StringValue("pid"),
				InstanceId:  types.StringValue("iid"),
				CertCheckId: types.StringValue("cid"),
				Source:      types.StringValue("tcp://www.example.com:443"),
			},
			isValid: true,
		},
		{
			description: "nil response",
			input:       nil,
			isValid:     false,
		},
		{
			description: "missing id",
			input: &observabilitySdk.CertCheckChildResponse{
				Source: "tcp://www.example.com:443",
			},
			isValid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := Model{
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("iid"),
			}
			err := mapFields(tt.input, &model)
			if !tt.isValid {
				if err == nil {
					t.Fatalf("Should have failed")
				}
				return
			}
			if err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			diff := cmp.Diff(model, tt.expected)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	model := &Model{
		Source: types.StringValue("tcp://www.example.com:443"),
	}
	expected := observabilitySdk.CreateCertCheckPayload{
		Source: "tcp://www.example.com:443",
	}
	diff := cmp.Diff(toCreatePayload(model), expected)
	if diff != "" {
		t.Fatalf("Data does not match: %s", diff)
	}
}

func TestCreatedCertCheck(t *testing.T) {
	const source = "tcp://www.example.com:443"
	tests := []struct {
		description string
		input       *observabilitySdk.CertCheckResponse
		expectedId  string
		isValid     bool
	}{
		{
			description: "created check in response",
			input: &observabilitySdk.CertCheckResponse{
				CertCheck: &observabilitySdk.CertCheckChildResponse{Id: "cid", Source: source},
				CertChecks: []observabilitySdk.CertCheckChildResponse{
					{Id: "other", Source: "tcp://other.example.com:443"},
					{Id: "cid", Source: source},
				},
			},
			expectedId: "cid",
			isValid:    true,
		},
		{
			description: "created check only in list",
			input: &observabilitySdk.CertCheckResponse{
				CertChecks: []observabilitySdk.CertCheckChildResponse{
					{Id: "other", Source: "tcp://other.example.com:443"},
					{Id: "cid", Source: source},
				},
			},
			expectedId: "cid",
			isValid:    true,
		},
		{
			description: "created check only in list, ambiguous source",
			input: &observabilitySdk.CertCheckResponse{
				CertChecks: []observabilitySdk.CertCheckChildResponse{
					{Id: "cid1", Source: source},
					{Id: "cid2", Source: source},
				},
			},
			isValid: false,
		},
		{
			description: "created check not found",
			input: &observabilitySdk.CertCheckResponse{
				CertChecks: []observabilitySdk.CertCheckChildResponse{
					{Id: "other", Source: "tcp://other.example.com:443"},
				},
			},
			isValid: false,
		},
		{
			description: "nil response",
			input:       nil,
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := createdCertCheck(tt.input, source)
			if !tt.isValid {
				if err == nil {
					t.Fatalf("Should have failed")
				}
				return
			}
			if err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if got.Id != tt.expectedId {
				t.Fatalf("Expected id %q, got %q", tt.expectedId, got.Id)
			}
		})
	}
}

func TestFindCertCheck(t *testing.T) {
	checks := []observabilitySdk.CertCheckChildResponse{
		{Id: "cid1", Source: "tcp://one.example.com:443"},
		{Id: "cid2", Source: "tcp://two.example.com:443"},
	}
	if got := findCertCheck(checks, "cid2"); got == nil || got.Source != "tcp://two.example.com:443" {
		t.Fatalf("Expected to find cid2, got %v", got)
	}
	if got := findCertCheck(checks, "missing"); got != nil {
		t.Fatalf("Expected nil, got %v", got)
	}
	if got := findCertCheck(nil, "cid1"); got != nil {
		t.Fatalf("Expected nil, got %v", got)
	}
}
