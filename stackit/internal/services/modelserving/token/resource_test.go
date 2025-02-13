package token

import (
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"testing"
)

func TestMapGetTokenFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		state       *Model
		input       *GetTokenResponse
		expected    Model
		isValid     bool
	}{
		{
			description: "should error when response is nil",
			state:       &Model{},
			input:       nil,
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should error when token is nil in response",
			state:       &Model{},
			input:       &GetTokenResponse{Token: nil},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should error when state is nil in response",
			state:       nil,
			input:       &GetTokenResponse{Token: &Token{}},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:        types.StringValue("pid,tid"),
				ProjectId: types.StringValue("pid"),
				TokenId:   types.StringValue("tid"),
			},
			input: &GetTokenResponse{
				Token: &Token{
					ID:          utils.Ptr("tid"),
					ValidUntil:  utils.Ptr("2021-01-01T00:00:00Z"),
					State:       utils.Ptr("active"),
					Name:        utils.Ptr("name"),
					Description: utils.Ptr("desc"),
					Region:      utils.Ptr("eu01"),
				},
			},
			expected: Model{
				Id:          types.StringValue("pid,tid"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue("eu01"),
				TokenId:     types.StringValue("tid"),
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				State:       types.StringValue("active"),
				ValidUntil:  types.StringValue("2021-01-01T00:00:00Z"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			err := mapGetResponse(tt.input, tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}

			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}

			if tt.isValid {
				diff := cmp.Diff(tt.state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapUpdateTokenFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		state       *Model
		input       *UpdateTokenResponse
		expected    Model
		isValid     bool
	}{
		{
			description: "should error when response is nil",
			state:       &Model{},
			input:       nil,
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should error when token is nil in response",
			state:       &Model{},
			input:       &UpdateTokenResponse{Token: nil},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should error when state is nil in response",
			state:       nil,
			input:       &UpdateTokenResponse{Token: &Token{}},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:        types.StringValue("pid,tid"),
				ProjectId: types.StringValue("pid"),
				TokenId:   types.StringValue("tid"),
			},
			input: &UpdateTokenResponse{
				Token: &Token{
					ID:          utils.Ptr("tid"),
					ValidUntil:  utils.Ptr("2021-01-01T00:00:00Z"),
					State:       utils.Ptr("active"),
					Name:        utils.Ptr("name"),
					Description: utils.Ptr("desc"),
					Region:      utils.Ptr("eu01"),
				},
			},
			expected: Model{
				Id:          types.StringValue("pid,tid"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue("eu01"),
				TokenId:     types.StringValue("tid"),
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				State:       types.StringValue("active"),
				ValidUntil:  types.StringValue("2021-01-01T00:00:00Z"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			err := mapUpdateResponse(tt.input, tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}

			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}

			if tt.isValid {
				diff := cmp.Diff(tt.state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapCreateTokenFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		state       *Model
		input       *CreateTokenResponse
		expected    Model
		isValid     bool
	}{
		{
			description: "should error when response is nil",
			state:       &Model{},
			input:       nil,
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should error when token is nil in response",
			state:       &Model{},
			input:       &CreateTokenResponse{Token: nil},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should error when state is nil in response",
			state:       nil,
			input:       &CreateTokenResponse{Token: &TokenCreated{}},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:        types.StringValue("pid,tid"),
				ProjectId: types.StringValue("pid"),
			},
			input: &CreateTokenResponse{
				Token: &TokenCreated{
					ID:          utils.Ptr("tid"),
					ValidUntil:  utils.Ptr("2021-01-01T00:00:00Z"),
					State:       utils.Ptr("active"),
					Name:        utils.Ptr("name"),
					Description: utils.Ptr("desc"),
					Region:      utils.Ptr("eu01"),
					Content:     utils.Ptr("content"),
				},
			},
			expected: Model{
				Id:          types.StringValue("pid,tid"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue("eu01"),
				TokenId:     types.StringValue("tid"),
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				State:       types.StringValue("active"),
				ValidUntil:  types.StringValue("2021-01-01T00:00:00Z"),
				Content:     types.StringValue("content"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			err := mapCreateResponse(tt.input, tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}

			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}

			if tt.isValid {
				diff := cmp.Diff(tt.state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		input       *Model
		expected    *CreateTokenPayload
		isValid     bool
	}{
		{
			description: "should error on nil input",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
		{
			description: "should convert correctly",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				TTLDuration: types.StringValue("1h"),
			},
			expected: &CreateTokenPayload{
				Name:        utils.Ptr("name"),
				Description: utils.Ptr("desc"),
				TTLDuration: utils.Ptr("1h"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

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

func TestToUpdatePayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		input       *Model
		expected    *UpdateTokenPayload
		isValid     bool
	}{
		{
			description: "should error on nil input",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
		{
			description: "should convert correctly",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
			},
			expected: &UpdateTokenPayload{
				Name:        utils.Ptr("name"),
				Description: utils.Ptr("desc"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			output, err := toUpdatePayload(tt.input)
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
