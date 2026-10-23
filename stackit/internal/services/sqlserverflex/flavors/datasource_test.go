package flavors

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *sqlserverflex.ListFlavorsResponse
		state       *model
		expected    *model
		isValid     bool
	}{
		{
			description: "default_values_and_sorting",
			input: &sqlserverflex.ListFlavorsResponse{
				Flavors: []sqlserverflex.ListFlavors{
					{
						Id:          "id2",
						Description: "desc2",
						Cpu:         4,
						Memory:      8,
						MinGB:       20,
						MaxGB:       200,
						NodeType:    "ha",
						StorageClasses: []sqlserverflex.FlavorStorageClassesStorageClass{
							{
								Class:          "class2",
								MaxIoPerSec:    2000,
								MaxThroughInMb: 200,
							},
							{
								Class:          "class1",
								MaxIoPerSec:    1000,
								MaxThroughInMb: 100,
							},
						},
					},
					{
						Id:          "id1",
						Description: "desc1",
						Cpu:         2,
						Memory:      4,
						MinGB:       10,
						MaxGB:       100,
						NodeType:    "single",
						StorageClasses: []sqlserverflex.FlavorStorageClassesStorageClass{
							{
								Class:          "class3",
								MaxIoPerSec:    3000,
								MaxThroughInMb: 300,
							},
						},
					},
				},
			},
			state: &model{
				ProjectId: types.StringValue("project_id"),
				Region:    types.StringValue("region"),
			},
			expected: &model{
				ID:        types.StringValue("project_id,region"),
				ProjectId: types.StringValue("project_id"),
				Region:    types.StringValue("region"),
				Flavors: []flavor{
					{
						Id:          types.StringValue("id1"),
						Description: types.StringValue("desc1"),
						CPU:         types.Int64Value(2),
						Memory:      types.Int64Value(4),
						MinGB:       types.Int32Value(10),
						MaxGB:       types.Int32Value(100),
						NodeType:    types.StringValue("single"),
						StorageClasses: []storageClass{
							{
								Class:          types.StringValue("class3"),
								MaxIOPerSec:    types.Int32Value(3000),
								MaxThroughInMB: types.Int32Value(300),
							},
						},
					},
					{
						Id:          types.StringValue("id2"),
						Description: types.StringValue("desc2"),
						CPU:         types.Int64Value(4),
						Memory:      types.Int64Value(8),
						MinGB:       types.Int32Value(20),
						MaxGB:       types.Int32Value(200),
						NodeType:    types.StringValue("ha"),
						StorageClasses: []storageClass{
							{
								Class:          types.StringValue("class1"),
								MaxIOPerSec:    types.Int32Value(1000),
								MaxThroughInMB: types.Int32Value(100),
							},
							{
								Class:          types.StringValue("class2"),
								MaxIOPerSec:    types.Int32Value(2000),
								MaxThroughInMB: types.Int32Value(200),
							},
						},
					},
				},
			},
			isValid: true,
		},
		{
			description: "empty_response",
			input:       &sqlserverflex.ListFlavorsResponse{},
			state: &model{
				ProjectId: types.StringValue("project_id"),
				Region:    types.StringValue("region"),
			},
			expected: &model{
				ID:        types.StringValue("project_id,region"),
				ProjectId: types.StringValue("project_id"),
				Region:    types.StringValue("region"),
			},
			isValid: true,
		},
		{
			description: "nil_response",
			input:       nil,
			state: &model{
				ProjectId: types.StringValue("project_id"),
				Region:    types.StringValue("region"),
			},
			expected: &model{
				ProjectId: types.StringValue("project_id"),
				Region:    types.StringValue("region"),
			},
			isValid: false,
		},
		{
			description: "nil_model",
			input:       &sqlserverflex.ListFlavorsResponse{},
			state:       nil,
			expected:    nil,
			isValid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(tt.input, tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.expected, tt.state)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
