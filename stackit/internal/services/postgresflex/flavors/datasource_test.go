package flavors

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		name     string
		response *postgresflex.ListFlavorsResponse
		model    *model
		expected *model
		valid    bool
	}{
		{
			name: "maps and sorts flavors and storage classes",
			response: &postgresflex.ListFlavorsResponse{
				Flavors: []postgresflex.ListFlavors{
					{
						Id:          "flavor-2",
						Description: "second",
						Cpu:         4,
						Memory:      8,
						MinGB:       20,
						MaxGB:       200,
						NodeType:    "Replica",
						StorageClasses: []postgresflex.FlavorStorageClassesStorageClass{
							{
								Class:          "class-2",
								MaxIoPerSec:    2000,
								MaxThroughInMb: 200,
							},
							{
								Class:          "class-1",
								MaxIoPerSec:    1000,
								MaxThroughInMb: 100,
							},
						},
					},
					{
						Id:          "flavor-1",
						Description: "first",
						Cpu:         2,
						Memory:      4,
						MinGB:       10,
						MaxGB:       100,
						NodeType:    "Single",
						StorageClasses: []postgresflex.FlavorStorageClassesStorageClass{
							{
								Class:          "class-3",
								MaxIoPerSec:    3000,
								MaxThroughInMb: 300,
							},
						},
					},
				},
			},
			model: &model{
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
			},
			expected: &model{
				ID:        types.StringValue("project-id,eu01"),
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
				Flavors: []flavor{
					{
						Id:          types.StringValue("flavor-1"),
						Description: types.StringValue("first"),
						CPU:         types.Int64Value(2),
						Memory:      types.Int64Value(4),
						MinGB:       types.Int32Value(10),
						MaxGB:       types.Int32Value(100),
						NodeType:    types.StringValue("Single"),
						StorageClasses: []storageClass{
							{
								Class:          types.StringValue("class-3"),
								MaxIOPerSec:    types.Int32Value(3000),
								MaxThroughInMB: types.Int32Value(300),
							},
						},
					},
					{
						Id:          types.StringValue("flavor-2"),
						Description: types.StringValue("second"),
						CPU:         types.Int64Value(4),
						Memory:      types.Int64Value(8),
						MinGB:       types.Int32Value(20),
						MaxGB:       types.Int32Value(200),
						NodeType:    types.StringValue("Replica"),
						StorageClasses: []storageClass{
							{
								Class:          types.StringValue("class-1"),
								MaxIOPerSec:    types.Int32Value(1000),
								MaxThroughInMB: types.Int32Value(100),
							},
							{
								Class:          types.StringValue("class-2"),
								MaxIOPerSec:    types.Int32Value(2000),
								MaxThroughInMB: types.Int32Value(200),
							},
						},
					},
				},
			},
			valid: true,
		},
		{
			name:     "maps empty response",
			response: &postgresflex.ListFlavorsResponse{},
			model: &model{
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
			},
			expected: &model{
				ID:        types.StringValue("project-id,eu01"),
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
				Flavors:   []flavor{},
			},
			valid: true,
		},
		{
			name: "rejects nil response",
			model: &model{
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
			},
			valid: false,
		},
		{
			name:     "rejects nil model",
			response: &postgresflex.ListFlavorsResponse{},
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapFields(tt.response, tt.model)
			if (err == nil) != tt.valid {
				t.Fatalf("mapFields() error = %v, valid = %t", err, tt.valid)
			}
			if tt.valid {
				if diff := cmp.Diff(tt.expected, tt.model); diff != "" {
					t.Fatalf("mapFields() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
