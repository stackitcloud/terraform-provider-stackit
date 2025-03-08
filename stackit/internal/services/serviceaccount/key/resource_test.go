package key

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
)

func TestComputeValidUntil(t *testing.T) {
	tests := []struct {
		name     string
		ttlDays  *int
		isValid  bool
		expected time.Time
	}{
		{
			name:     "ttlDays is 10",
			ttlDays:  utils.Ptr(10),
			isValid:  true,
			expected: time.Now().UTC().Add(time.Duration(10) * 24 * time.Hour),
		},
		{
			name:     "ttlDays is 0",
			ttlDays:  utils.Ptr(0),
			isValid:  true,
			expected: time.Now().UTC().Add(time.Duration(0) * 24 * time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			int64TTlDays := int64(*tt.ttlDays)
			validUntil, err := computeValidUntil(&int64TTlDays)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				tolerance := 1 * time.Second
				if validUntil.Sub(tt.expected) > tolerance && tt.expected.Sub(validUntil) > tolerance {
					t.Fatalf("Times do not match. got: %v expected: %v", validUntil, tt.expected)
				}
			}
		})
	}
}

func TestMapResponse(t *testing.T) {
	tests := []struct {
		description string
		input       *serviceaccount.CreateServiceAccountKeyResponse
		expected    Model
		isValid     bool
	}{
		{
			description: "default_values",
			input: &serviceaccount.CreateServiceAccountKeyResponse{
				Id: utils.Ptr("id"),
			},
			expected: Model{
				Id:                  types.StringValue("pid,email,id"),
				KeyId:               types.StringValue("id"),
				ProjectId:           types.StringValue("pid"),
				ServiceAccountEmail: types.StringValue("email"),
				Json:                types.StringValue("{}"),
				RotateWhenChanged:   types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			isValid: true,
		},
		{
			description: "nil_response",
			input:       nil,
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "nil_response_2",
			input:       &serviceaccount.CreateServiceAccountKeyResponse{},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "no_id",
			input: &serviceaccount.CreateServiceAccountKeyResponse{
				Active: utils.Ptr(true),
			},
			expected: Model{},
			isValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId:           tt.expected.ProjectId,
				ServiceAccountEmail: tt.expected.ServiceAccountEmail,
				KeyId:               tt.expected.KeyId,
				Json:                types.StringValue("{}"),
				RotateWhenChanged:   types.MapValueMust(types.StringType, map[string]attr.Value{}),
			}
			err := mapCreateResponse(tt.input, model)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				model.Json = types.StringValue("{}")
				diff := cmp.Diff(*model, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
