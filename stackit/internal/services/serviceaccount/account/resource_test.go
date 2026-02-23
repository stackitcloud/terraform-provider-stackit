package account

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *serviceaccount.CreateServiceAccountPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&serviceaccount.CreateServiceAccountPayload{
				Name: nil,
			},
			true,
		},
		{
			"default_values",
			&Model{
				Name: types.StringValue("example-name1"),
			},
			&serviceaccount.CreateServiceAccountPayload{
				Name: utils.Ptr("example-name1"),
			},
			true,
		},
		{
			"nil_model",
			nil,
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

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *serviceaccount.ServiceAccount
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&serviceaccount.ServiceAccount{
				ProjectId: utils.Ptr("pid"),
				Email:     utils.Ptr("mail"),
			},
			Model{
				Id:        types.StringValue("pid,mail"),
				ProjectId: types.StringValue("pid"),
				Email:     types.StringValue("mail"),
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
			&serviceaccount.ServiceAccount{},
			Model{},
			false,
		},
		{
			"no_id",
			&serviceaccount.ServiceAccount{
				ProjectId: utils.Ptr("pid"),
				Internal:  utils.Ptr(true),
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
			err := mapFields(tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestParseNameFromEmail(t *testing.T) {
	testCases := []struct {
		email       string
		expected    string
		shouldError bool
	}{
		// Standard SA domain (Positive: 7 to 10 random characters)
		{"foo-vshp191@sa.stackit.cloud", "foo", false},           // 7 chars
		{"bar-8565oq12@sa.stackit.cloud", "bar", false},          // 8 chars
		{"foo-bar-acfj2s123@sa.stackit.cloud", "foo-bar", false}, // 9 chars
		{"baz-abcdefghij@sa.stackit.cloud", "baz", false},        // 10 chars

		// Standard SA domain (Negative: 6 and 11 random characters)
		{"foo-vshp19@sa.stackit.cloud", "", true},      // 6 chars (Too short)
		{"bar-8565oq12345@sa.stackit.cloud", "", true}, // 11 chars (Too long)

		// SKE SA domain (Positive: 7 to 10 random characters)
		{"foo-qnmbwo1@ske.sa.stackit.cloud", "foo", false},           // 7 chars
		{"bar-qnmbwo12@ske.sa.stackit.cloud", "bar", false},          // 8 chars
		{"foo-bar-qnmbwo123@ske.sa.stackit.cloud", "foo-bar", false}, // 9 chars
		{"baz-abcdefghij@ske.sa.stackit.cloud", "baz", false},        // 10 chars

		// SKE SA domain (Negative: 6 and 11 random characters)
		{"foo-qnmbwo@ske.sa.stackit.cloud", "", true},      // 6 chars (Too short)
		{"bar-qnmbwo12345@ske.sa.stackit.cloud", "", true}, // 11 chars (Too long)

		// Invalid cases (Formatting & Unknown Domains)
		{"invalid-email@sa.stackit.cloud", "", true},
		{"missingcode-@sa.stackit.cloud", "", true},
		{"nohyphen8565oq1@sa.stackit.cloud", "", true},
		{"eu01-qnmbwo1@unknown.stackit.cloud", "", true},
		{"eu01-qnmbwo1@ske.stackit.com", "", true}, // Missing .sa. and ends in .com
		{"someotherformat@sa.stackit.cloud", "", true},
		{"invalid-format@ske.sa.stackit.cloud", "", true}, // SKE domain but missing the character suffix completely
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			name, err := parseNameFromEmail(tc.email)
			if tc.shouldError {
				if err == nil {
					t.Errorf("expected an error for email: %s, but got none", tc.email)
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error for email: %s, but got: %v", tc.email, err)
				}
				if name != tc.expected {
					t.Errorf("expected name: %s, got: %s for email: %s", tc.expected, name, tc.email)
				}
			}
		})
	}
}
