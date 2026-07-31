package database

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	sdk "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *sdk.CreateDatabasePayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &Model{
				SharedModel: SharedModel{
					Name:          types.StringValue("db-name"),
					Owner:         types.StringValue("db-owner"),
					Collation:     types.StringValue("Latin1_General_CI_AS"),
					Compatibility: types.Int64Value(150),
				},
			},
			expected: &sdk.CreateDatabasePayload{
				Name:          "db-name",
				Owner:         "db-owner",
				Collation:     utils.Ptr("Latin1_General_CI_AS"),
				Compatibility: utils.Ptr(int32(150)),
			},
			isValid: true,
		},
		{
			description: "optional_fields_empty",
			input: &Model{
				SharedModel: SharedModel{
					Name:          types.StringValue("db-name"),
					Owner:         types.StringValue("db-owner"),
					Collation:     types.StringNull(),
					Compatibility: types.Int64Null(),
				},
			},
			expected: &sdk.CreateDatabasePayload{
				Name:  "db-name",
				Owner: "db-owner",
			},
			isValid: true,
		},
		{
			description: "nil_model",
			input:       nil,
			expected:    nil,
			isValid:     false,
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
	const testRegion = "region"
	tests := []struct {
		description string
		input       *sdk.GetDatabaseResponse
		region      string
		expected    SharedModel
		isValid     bool
	}{
		{
			"default_values",
			&sdk.GetDatabaseResponse{
				Id:                 123,
				Name:               "test_db",
				CollationName:      "test_collation",
				CompatibilityLevel: 100,
				Owner:              "test_owner",
			},
			testRegion,
			SharedModel{
				Id:            types.StringValue("pid,region,iid,test_db"),
				ProjectId:     types.StringValue("pid"),
				Region:        types.StringValue(testRegion),
				InstanceId:    types.StringValue("iid"),
				Name:          types.StringValue("test_db"),
				Collation:     types.StringValue("test_collation"),
				Compatibility: types.Int64Value(100),
				Owner:         types.StringValue("test_owner"),
				DatabaseId:    types.Int64Value(123),
			},
			true,
		},
		{
			"nil_response",
			nil,
			testRegion,
			SharedModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &SharedModel{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(context.Background(), tt.input, state, tt.region)
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
