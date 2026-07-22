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

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		state       *InstanceTokenDataSourceModel
		input       modelexperiments.TokenMetadata
		expected    InstanceTokenDataSourceModel
		isValid     bool
	}{
		{
			description: "should error when state is nil",
			state:       nil,
			input: modelexperiments.TokenMetadata{
				Id: "id",
			},
			expected: InstanceTokenDataSourceModel{},
			isValid:  false,
		},
		{
			description: "should error when token id is not present",
			state:       &InstanceTokenDataSourceModel{},
			input:       modelexperiments.TokenMetadata{},
			expected:    InstanceTokenDataSourceModel{},
			isValid:     false,
		},
		{
			description: "map min values",
			state: &InstanceTokenDataSourceModel{
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
				TokenId:    types.StringValue("tid"),
			},
			input: modelexperiments.TokenMetadata{
				Id:         "tid",
				State:      "active",
				Name:       "name",
				ValidUntil: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: InstanceTokenDataSourceModel{
				Id:         types.StringValue("pid,eu01,id,tid"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
				TokenId:    types.StringValue("tid"),
				Name:       types.StringValue("name"),
				ValidUntil: types.StringValue("2099-01-01T00:00:00Z"),
				Labels:     types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "map max values",
			state: &InstanceTokenDataSourceModel{
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
				TokenId:    types.StringValue("tid"),
			},
			input: modelexperiments.TokenMetadata{
				Id:          "tid",
				State:       "active",
				Name:        "name",
				Description: new("description"),
				Labels:      &map[string]string{"key": "value"},
				ValidUntil:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: InstanceTokenDataSourceModel{
				Id:          types.StringValue("pid,eu01,id,tid"),
				ProjectId:   types.StringValue("pid"),
				InstanceId:  types.StringValue("id"),
				Region:      types.StringValue("eu01"),
				TokenId:     types.StringValue("tid"),
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
				Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := mapDataSourceFields(ctx, &tt.input, tt.state, "eu01", "id")
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
