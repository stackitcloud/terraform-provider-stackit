package token

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/modelserving"
)

func TestMapGetTokenFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		state       *Model
		input       *modelserving.GetTokenResponse
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
			input:       &modelserving.GetTokenResponse{Token: nil},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should error when state is nil in response",
			state:       nil,
			input: &modelserving.GetTokenResponse{
				Token: &modelserving.Token{},
			},
			expected: Model{},
			isValid:  false,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:        types.StringValue("pid,tid"),
				ProjectId: types.StringValue("pid"),
				TokenId:   types.StringValue("tid"),
			},
			input: &modelserving.GetTokenResponse{
				Token: &modelserving.Token{
					Id: utils.Ptr("tid"),
					ValidUntil: utils.Ptr(
						time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
					),
					State:       utils.Ptr("active"),
					Name:        utils.Ptr("name"),
					Description: utils.Ptr("desc"),
					Region:      utils.Ptr("eu01"),
				},
			},
			expected: Model{
				Id:                types.StringValue("pid,tid"),
				ProjectId:         types.StringValue("pid"),
				Region:            types.StringValue("eu01"),
				TokenId:           types.StringValue("tid"),
				Name:              types.StringValue("name"),
				Description:       types.StringValue("desc"),
				State:             types.StringValue("active"),
				ValidUntil:        types.StringValue("2099-01-01T00:00:00Z"),
				RotateWhenChanged: types.MapNull(types.StringType),
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

func TestMapCreateTokenFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description              string
		state                    *Model
		inputCreateTokenResponse *modelserving.CreateTokenResponse
		inputGetTokenResponse    *modelserving.GetTokenResponse
		expected                 Model
		isValid                  bool
	}{
		{
			description:              "should error when create token response is nil",
			state:                    &Model{},
			inputCreateTokenResponse: nil,
			inputGetTokenResponse:    nil,
			expected:                 Model{},
			isValid:                  false,
		},
		{
			description: "should error when token is nil in create token response",
			state:       &Model{},
			inputCreateTokenResponse: &modelserving.CreateTokenResponse{
				Token: nil,
			},
			inputGetTokenResponse: nil,
			expected:              Model{},
			isValid:               false,
		},
		{
			description: "should error when get token response is nil",
			state:       &Model{},
			inputCreateTokenResponse: &modelserving.CreateTokenResponse{
				Token: &modelserving.TokenCreated{},
			},
			inputGetTokenResponse: nil,
			expected:              Model{},
			isValid:               false,
		},
		{
			description: "should error when get token response is nil",
			state: &Model{
				Id:        types.StringValue("pid,tid"),
				ProjectId: types.StringValue("pid"),
			},
			inputCreateTokenResponse: &modelserving.CreateTokenResponse{
				Token: &modelserving.TokenCreated{
					Id: utils.Ptr("tid"),
					ValidUntil: utils.Ptr(
						time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
					),
					State:       utils.Ptr("active"),
					Name:        utils.Ptr("name"),
					Description: utils.Ptr("desc"),
					Region:      utils.Ptr("eu01"),
					Content:     utils.Ptr("content"),
				},
			},
			inputGetTokenResponse: nil,
			expected:              Model{},
			isValid:               false,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:        types.StringValue("pid,tid"),
				ProjectId: types.StringValue("pid"),
			},
			inputCreateTokenResponse: &modelserving.CreateTokenResponse{
				Token: &modelserving.TokenCreated{
					Id: utils.Ptr("tid"),
					ValidUntil: utils.Ptr(
						time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
					),
					State:       utils.Ptr("active"),
					Name:        utils.Ptr("name"),
					Description: utils.Ptr("desc"),
					Region:      utils.Ptr("eu01"),
					Content:     utils.Ptr("content"),
				},
			},
			inputGetTokenResponse: &modelserving.GetTokenResponse{
				Token: &modelserving.Token{
					State: utils.Ptr("active"),
				},
			},
			expected: Model{
				Id:                types.StringValue("pid,tid"),
				ProjectId:         types.StringValue("pid"),
				Region:            types.StringValue("eu01"),
				TokenId:           types.StringValue("tid"),
				Name:              types.StringValue("name"),
				Description:       types.StringValue("desc"),
				State:             types.StringValue("active"),
				ValidUntil:        types.StringValue("2099-01-01T00:00:00Z"),
				Content:           types.StringValue("content"),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			err := mapCreateResponse(
				tt.inputCreateTokenResponse,
				tt.inputGetTokenResponse,
				tt.state,
			)
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
		expected    *modelserving.CreateTokenPayload
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
			expected: &modelserving.CreateTokenPayload{
				Name:        utils.Ptr("name"),
				Description: utils.Ptr("desc"),
				TtlDuration: utils.Ptr("1h"),
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
		expected    *modelserving.PartialUpdateTokenPayload
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
			expected: &modelserving.PartialUpdateTokenPayload{
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
