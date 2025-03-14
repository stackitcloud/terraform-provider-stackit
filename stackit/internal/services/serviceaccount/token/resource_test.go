package token

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		inputRoles  []string
		expected    *serviceaccount.CreateAccessTokenPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{
				TtlDays: types.Int64Value(20),
			},
			[]string{},
			&serviceaccount.CreateAccessTokenPayload{
				TtlDays: types.Int64Value(20).ValueInt64Pointer(),
			},
			true,
		},
		{
			"nil_model",
			nil,
			[]string{},
			nil,
			false,
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

func TestMapCreateResponse(t *testing.T) {
	tests := []struct {
		description string
		input       *serviceaccount.AccessToken
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&serviceaccount.AccessToken{
				Id:    utils.Ptr("aid"),
				Token: utils.Ptr("token"),
			},
			Model{
				Id:            types.StringValue("pid,aid"),
				ProjectId:     types.StringValue("pid"),
				Token:         types.StringValue("token"),
				AccessTokenId: types.StringValue("aid"),
			},
			true,
		},
		{
			"complete_values",
			&serviceaccount.AccessToken{
				Id:         utils.Ptr("aid"),
				Token:      utils.Ptr("token"),
				CreatedAt:  utils.Ptr(time.Now()),
				ValidUntil: utils.Ptr(time.Now().Add(24 * time.Hour)),
			},
			Model{
				Id:            types.StringValue("pid,aid"),
				ProjectId:     types.StringValue("pid"),
				Token:         types.StringValue("token"),
				AccessTokenId: types.StringValue("aid"),
				CreatedAt:     types.StringValue(time.Now().Format(time.RFC3339)),                     // Adjust to the format used
				ValidUntil:    types.StringValue(time.Now().Add(24 * time.Hour).Format(time.RFC3339)), // Adjust format
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"nil_response_2",
			&serviceaccount.AccessToken{},
			Model{},
			false,
		},
		{
			"no_id",
			&serviceaccount.AccessToken{
				Token: utils.Ptr("token"),
			},
			Model{},
			false,
		},
		{
			"no_token",
			&serviceaccount.AccessToken{
				Id: utils.Ptr("id"),
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
			}
			err := mapCreateResponse(tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
		})
	}
}

func TestMapListResponse(t *testing.T) {
	tests := []struct {
		description string
		input       *serviceaccount.AccessTokenMetadata
		expected    Model
		isValid     bool
	}{
		{
			"valid_fields",
			&serviceaccount.AccessTokenMetadata{
				Id:         utils.Ptr("aid"),
				CreatedAt:  utils.Ptr(time.Now()),
				ValidUntil: utils.Ptr(time.Now().Add(24 * time.Hour)),
			},
			Model{
				Id:            types.StringValue("pid,aid"),
				ProjectId:     types.StringValue("pid"),
				AccessTokenId: types.StringValue("aid"),
				CreatedAt:     types.StringValue(time.Now().Format(time.RFC3339)),                     // Adjusted for test setup time
				ValidUntil:    types.StringValue(time.Now().Add(24 * time.Hour).Format(time.RFC3339)), // Adjust for format
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"nil_fields",
			&serviceaccount.AccessTokenMetadata{
				Id: nil,
			},
			Model{},
			false,
		},
		{
			"no_id",
			&serviceaccount.AccessTokenMetadata{
				CreatedAt:  utils.Ptr(time.Now()),
				ValidUntil: utils.Ptr(time.Now().Add(24 * time.Hour)),
			},
			Model{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
			}
			err := mapListResponse(tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Expected an error but did not get one")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Did not expect an error but got one: %v", err)
			}
		})
	}
}
