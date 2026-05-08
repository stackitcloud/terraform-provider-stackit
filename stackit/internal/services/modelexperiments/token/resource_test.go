package token

import (
	"context"
	"testing"
	"time"

	modelexperiments "dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/modelexperiments/v1api"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMapTokenFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		state       *Model
		input       modelexperiments.TokenMetadata
		expected    Model
		isValid     bool
	}{
		{
			description: "should error when state is nil",
			state:       nil,
			input: modelexperiments.TokenMetadata{
				Id: "id",
			},
			expected: Model{},
			isValid:  false,
		},
		{
			description: "should error when token id is not present",
			state:       &Model{},
			input:       modelexperiments.TokenMetadata{},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:         types.StringValue("pid,eu01,id"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
				TokenId:    types.StringValue("id"),
			},
			input: modelexperiments.TokenMetadata{
				Id:          "id",
				Description: new("description"),
				Labels:      &map[string]string{"key": "value"},
				State:       "active",
				Name:        "name",
				ValidUntil:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: Model{
				Id:          types.StringValue("pid,eu01,id"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue("eu01"),
				InstanceId:  types.StringValue("id"),
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				State:       types.StringValue("active"),
				ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
				Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				TokenId:     types.StringValue("id"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := mapToken(ctx, tt.input, tt.state)
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

func TestMapCreateResponseFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description         string
		state               *Model
		inputCreateResponse *modelexperiments.CreateTokenResponse
		inputGetResponse    *modelexperiments.GetTokenResponse
		expected            Model
		isValid             bool
	}{
		{
			description:         "should error when token create response is nil",
			state:               &Model{},
			inputCreateResponse: nil,
			inputGetResponse:    &modelexperiments.GetTokenResponse{},
			expected:            Model{},
			isValid:             false,
		},
		{
			description:         "should error when state is nil",
			state:               nil,
			inputCreateResponse: &modelexperiments.CreateTokenResponse{},
			inputGetResponse:    &modelexperiments.GetTokenResponse{},
			expected:            Model{},
			isValid:             false,
		},
		{
			description: "should error when token id is not present",
			state:       &Model{},
			inputCreateResponse: &modelexperiments.CreateTokenResponse{
				Token: modelexperiments.Token{},
			},
			inputGetResponse: &modelexperiments.GetTokenResponse{},
			expected:         Model{},
			isValid:          false,
		},
		{
			description: "should map fields correctly even if Get Response is nil",
			state: &Model{
				Id:         types.StringValue("pid,eu01,id"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
			},
			inputCreateResponse: &modelexperiments.CreateTokenResponse{
				Token: modelexperiments.Token{
					Id:          "id",
					Content:     "token",
					Description: new("description"),
					Labels:      &map[string]string{"key": "value"},
					State:       "active",
					Name:        "name",
					ValidUntil:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
				}},
			inputGetResponse: nil,
			expected: Model{
				Id:          types.StringValue("pid,eu01,id"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue("eu01"),
				InstanceId:  types.StringValue("id"),
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				State:       types.StringValue("unknown"),
				Token:       types.StringValue("token"),
				TokenId:     types.StringValue("id"),
				Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
			},
			isValid: true,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:         types.StringValue("pid,eu01,id"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
			},
			inputCreateResponse: &modelexperiments.CreateTokenResponse{
				Token: modelexperiments.Token{
					Id:          "id",
					Content:     "token",
					Description: new("description"),
					Labels:      &map[string]string{"key": "value"},
					State:       "active",
					Name:        "name",
					ValidUntil:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
				}},
			inputGetResponse: &modelexperiments.GetTokenResponse{
				Token: modelexperiments.TokenMetadata{
					State: "active",
				},
			},
			expected: Model{
				Id:          types.StringValue("pid,eu01,id"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue("eu01"),
				InstanceId:  types.StringValue("id"),
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				State:       types.StringValue("active"),
				Token:       types.StringValue("token"),
				TokenId:     types.StringValue("id"),
				Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
			},
			isValid: true,
		},
		{
			description: "should map fields correctly with label nil",
			state: &Model{
				Id:         types.StringValue("pid,eu01,id"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
			},
			inputCreateResponse: &modelexperiments.CreateTokenResponse{
				Token: modelexperiments.Token{
					Id:          "id",
					Content:     "token",
					Description: new("description"),
					Labels:      nil,
					State:       "active",
					Name:        "name",
					ValidUntil:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
				}},
			inputGetResponse: &modelexperiments.GetTokenResponse{
				Token: modelexperiments.TokenMetadata{
					State: "active",
				},
			},
			expected: Model{
				Id:          types.StringValue("pid,eu01,id"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue("eu01"),
				InstanceId:  types.StringValue("id"),
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				State:       types.StringValue("active"),
				Token:       types.StringValue("token"),
				TokenId:     types.StringValue("id"),
				Labels:      types.MapNull(types.StringType),
				ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := mapCreateResponse(ctx, tt.inputCreateResponse, tt.inputGetResponse, tt.state, "eu01")
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
		expected    *modelexperiments.CreateInstanceTokenPayload
		isValid     bool
	}{
		{
			description: "should error on nil input",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
		{
			description: "should error when map is not correct",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				Labels:      types.MapValueMust(types.Int64Type, map[string]attr.Value{"key": types.Int64Value(33)}),
				TTLDuration: types.StringValue("30d"),
			},
			expected: nil,
			isValid:  false,
		},
		{
			description: "should convert correctly",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				TTLDuration: types.StringValue("30d"),
			},
			expected: &modelexperiments.CreateInstanceTokenPayload{
				Name:        "name",
				Description: new("desc"),
				Labels:      &map[string]string{"key": "value"},
				TtlDuration: new("30d"),
			},
			isValid: true,
		},
		{
			description: "should convert correctly without labels",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				Labels:      types.MapNull(types.StringType),
				TTLDuration: types.StringValue("30d"),
			},
			expected: &modelexperiments.CreateInstanceTokenPayload{
				Name:        "name",
				Description: new("desc"),
				TtlDuration: new("30d"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
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
		expected    *modelexperiments.PartialUpdateInstanceTokenPayload
		isValid     bool
	}{
		{
			description: "should error on nil input",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
		{
			description: "should error when map is not correct",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				Labels:      types.MapValueMust(types.Int64Type, map[string]attr.Value{"key": types.Int64Value(33)}),
			},
			expected: nil,
			isValid:  false,
		},
		{
			description: "should convert correctly",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
			},
			expected: &modelexperiments.PartialUpdateInstanceTokenPayload{
				Name:        new("name"),
				Description: new("desc"),
				Labels:      &map[string]string{"key": "value"},
			},
			isValid: true,
		},
		{
			description: "should convert correctly without labels",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("desc"),
				Labels:      types.MapNull(types.StringType),
			},
			expected: &modelexperiments.PartialUpdateInstanceTokenPayload{
				Name:        new("name"),
				Description: new("desc"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
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
