package postgresflex

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api"
)

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *postgresflex.GetDatabaseResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			description: "default_values",
			input: &postgresflex.GetDatabaseResponse{
				Id: 123,
			},
			region: testRegion,
			expected: Model{
				Id:         types.StringValue("pid,region,iid,123"),
				DatabaseId: types.StringValue("123"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue(""),
				Owner:      types.StringValue(""),
				Region:     types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			input: &postgresflex.GetDatabaseResponse{
				Id:    123,
				Name:  "dbname",
				Owner: "username",
			},
			region: testRegion,
			expected: Model{
				Id:         types.StringValue("pid,region,iid,123"),
				DatabaseId: types.StringValue("123"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("dbname"),
				Owner:      types.StringValue("username"),
				Region:     types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "null_fields_and_int_conversions",
			input: &postgresflex.GetDatabaseResponse{
				Id:    123,
				Name:  "",
				Owner: "",
			},
			region: testRegion,
			expected: Model{
				Id:         types.StringValue("pid,region,iid,123"),
				DatabaseId: types.StringValue("123"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue(""),
				Owner:      types.StringValue(""),
				Region:     types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "nil_response",
			input:       nil,
			region:      testRegion,
			expected:    Model{},
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(tt.input, state, tt.region)
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

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *postgresflex.CreateDatabasePayload
		isValid     bool
	}{
		{
			description: "default_values",
			input: &Model{
				Name:  types.StringValue("dbname"),
				Owner: types.StringValue("username"),
			},
			expected: &postgresflex.CreateDatabasePayload{
				Name:  "dbname",
				Owner: new("username"),
			},
			isValid: true,
		},
		{
			description: "null_fields",
			input: &Model{
				Name:  types.StringNull(),
				Owner: types.StringNull(),
			},
			expected: &postgresflex.CreateDatabasePayload{
				Name:  "",
				Owner: nil,
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
