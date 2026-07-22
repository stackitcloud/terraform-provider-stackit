package token

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
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
				Id:                types.StringValue("pid,eu01,id,tid"),
				ProjectId:         types.StringValue("pid"),
				InstanceId:        types.StringValue("id"),
				Region:            types.StringValue("eu01"),
				TokenId:           types.StringValue("tid"),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			input: modelexperiments.TokenMetadata{
				Id:          "tid",
				Description: new("description"),
				Labels:      &map[string]string{"key": "value"},
				State:       "active",
				Name:        "name",
				ValidUntil:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: Model{
				Id:                types.StringValue("pid,eu01,id,tid"),
				ProjectId:         types.StringValue("pid"),
				Region:            types.StringValue("eu01"),
				InstanceId:        types.StringValue("id"),
				Name:              types.StringValue("name"),
				Description:       types.StringValue("description"),
				ValidUntil:        types.StringValue("2099-01-01T00:00:00Z"),
				Labels:            types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				TokenId:           types.StringValue("tid"),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := mapToken(ctx, &tt.input, tt.state, "eu01", "id")
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
		inputCreateResponse *modelexperiments.CreateInstanceTokenResponse
		expected            Model
		isValid             bool
	}{
		{
			description:         "should error when state is nil",
			state:               nil,
			inputCreateResponse: &modelexperiments.CreateInstanceTokenResponse{},
			expected:            Model{},
			isValid:             false,
		},
		{
			description: "should error when token id is not present",
			state:       &Model{},
			inputCreateResponse: &modelexperiments.CreateInstanceTokenResponse{
				Token: modelexperiments.Token{},
			},
			expected: Model{},
			isValid:  false,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:                types.StringValue("pid,eu01,id,tid"),
				ProjectId:         types.StringValue("pid"),
				InstanceId:        types.StringValue("id"),
				Region:            types.StringValue("eu01"),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			inputCreateResponse: &modelexperiments.CreateInstanceTokenResponse{
				Token: modelexperiments.Token{
					Id:          "tid",
					Content:     "token",
					Description: new("description"),
					Labels:      &map[string]string{"key": "value"},
					State:       "active",
					Name:        "name",
					ValidUntil:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expected: Model{
				Id:                types.StringValue("pid,eu01,id,tid"),
				ProjectId:         types.StringValue("pid"),
				Region:            types.StringValue("eu01"),
				InstanceId:        types.StringValue("id"),
				Name:              types.StringValue("name"),
				Description:       types.StringValue("description"),
				Token:             types.StringValue("token"),
				TokenId:           types.StringValue("tid"),
				Labels:            types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				ValidUntil:        types.StringValue("2099-01-01T00:00:00Z"),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "should map fields correctly with label nil",
			state: &Model{
				Id:                types.StringValue("pid,eu01,id,tid"),
				ProjectId:         types.StringValue("pid"),
				InstanceId:        types.StringValue("id"),
				Region:            types.StringValue("eu01"),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			inputCreateResponse: &modelexperiments.CreateInstanceTokenResponse{
				Token: modelexperiments.Token{
					Id:          "tid",
					Content:     "token",
					Description: new("description"),
					Labels:      nil,
					State:       "active",
					Name:        "name",
					ValidUntil:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expected: Model{
				Id:                types.StringValue("pid,eu01,id,tid"),
				ProjectId:         types.StringValue("pid"),
				Region:            types.StringValue("eu01"),
				InstanceId:        types.StringValue("id"),
				Name:              types.StringValue("name"),
				Description:       types.StringValue("description"),
				Token:             types.StringValue("token"),
				TokenId:           types.StringValue("tid"),
				Labels:            types.MapNull(types.StringType),
				ValidUntil:        types.StringValue("2099-01-01T00:00:00Z"),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := mapCreateResponse(ctx, &tt.inputCreateResponse.Token, tt.state, "eu01", "id")
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
