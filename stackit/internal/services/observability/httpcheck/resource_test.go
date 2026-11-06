package httpcheck

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	observabilitySdk "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *observabilitySdk.HttpCheckChildResponse
		expected    Model
		isValid     bool
	}{
		{
			description: "ok",
			input: &observabilitySdk.HttpCheckChildResponse{
				Id:  "cid",
				Url: "https://www.example.com",
			},
			expected: Model{
				Id:          types.StringValue("pid,iid,cid"),
				ProjectId:   types.StringValue("pid"),
				InstanceId:  types.StringValue("iid"),
				HttpCheckId: types.StringValue("cid"),
				Url:         types.StringValue("https://www.example.com"),
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
			input: &observabilitySdk.HttpCheckChildResponse{
				Url: "https://www.example.com",
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
		Url: types.StringValue("https://www.example.com"),
	}
	expected := observabilitySdk.CreateHttpCheckPayload{
		Url: "https://www.example.com",
	}
	diff := cmp.Diff(toCreatePayload(model), expected)
	if diff != "" {
		t.Fatalf("Data does not match: %s", diff)
	}
}

func TestCreatedHttpCheck(t *testing.T) {
	const url = "https://www.example.com"
	tests := []struct {
		description string
		input       *observabilitySdk.HttpCheckResponse
		expectedId  string
		isValid     bool
	}{
		{
			description: "created check in response",
			input: &observabilitySdk.HttpCheckResponse{
				HttpCheck: &observabilitySdk.HttpCheckChildResponse{Id: "cid", Url: url},
				HttpChecks: []observabilitySdk.HttpCheckChildResponse{
					{Id: "other", Url: "https://other.example.com"},
					{Id: "cid", Url: url},
				},
			},
			expectedId: "cid",
			isValid:    true,
		},
		{
			description: "created check only in list",
			input: &observabilitySdk.HttpCheckResponse{
				HttpChecks: []observabilitySdk.HttpCheckChildResponse{
					{Id: "other", Url: "https://other.example.com"},
					{Id: "cid", Url: url},
				},
			},
			expectedId: "cid",
			isValid:    true,
		},
		{
			description: "created check only in list, ambiguous url",
			input: &observabilitySdk.HttpCheckResponse{
				HttpChecks: []observabilitySdk.HttpCheckChildResponse{
					{Id: "cid1", Url: url},
					{Id: "cid2", Url: url},
				},
			},
			isValid: false,
		},
		{
			description: "created check not found",
			input: &observabilitySdk.HttpCheckResponse{
				HttpChecks: []observabilitySdk.HttpCheckChildResponse{
					{Id: "other", Url: "https://other.example.com"},
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
			got, err := createdHttpCheck(tt.input, url)
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

func TestFindHttpCheck(t *testing.T) {
	checks := []observabilitySdk.HttpCheckChildResponse{
		{Id: "cid1", Url: "https://one.example.com"},
		{Id: "cid2", Url: "https://two.example.com"},
	}
	if got := findHttpCheck(checks, "cid2"); got == nil || got.Url != "https://two.example.com" {
		t.Fatalf("Expected to find cid2, got %v", got)
	}
	if got := findHttpCheck(checks, "missing"); got != nil {
		t.Fatalf("Expected nil, got %v", got)
	}
	if got := findHttpCheck(nil, "cid1"); got != nil {
		t.Fatalf("Expected nil, got %v", got)
	}
}
