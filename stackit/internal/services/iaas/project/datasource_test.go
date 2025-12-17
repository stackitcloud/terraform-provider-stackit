package project

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

const (
	testTimestampValue = "2006-01-02T15:04:05Z"
)

func testTimestamp() time.Time {
	timestamp, _ := time.Parse(time.RFC3339, testTimestampValue)
	return timestamp
}

func TestMapDataSourceFields(t *testing.T) {
	const projectId = "pid"
	tests := []struct {
		description string
		state       *DatasourceModel
		input       *iaas.Project
		expected    *DatasourceModel
		isValid     bool
	}{
		{
			description: "default_values",
			state: &DatasourceModel{
				ProjectId: types.StringValue(projectId),
			},
			input: &iaas.Project{
				Id: utils.Ptr(projectId),
			},
			expected: &DatasourceModel{
				Id:        types.StringValue(projectId),
				ProjectId: types.StringValue(projectId),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			state: &DatasourceModel{
				ProjectId: types.StringValue(projectId),
			},
			input: &iaas.Project{
				AreaId:         utils.Ptr(iaas.AreaId{String: utils.Ptr("aid")}),
				CreatedAt:      utils.Ptr(testTimestamp()),
				InternetAccess: utils.Ptr(true),
				Id:             utils.Ptr(projectId),
				Status:         utils.Ptr("CREATED"),
				UpdatedAt:      utils.Ptr(testTimestamp()),
			},
			expected: &DatasourceModel{
				Id:             types.StringValue(projectId),
				ProjectId:      types.StringValue(projectId),
				AreaId:         types.StringValue("aid"),
				InternetAccess: types.BoolValue(true),
				State:          types.StringValue("CREATED"),
				Status:         types.StringValue("CREATED"),
				CreatedAt:      types.StringValue(testTimestampValue),
				UpdatedAt:      types.StringValue(testTimestampValue),
			},
			isValid: true,
		},
		{
			description: "static_area_id",
			state: &DatasourceModel{
				ProjectId: types.StringValue(projectId),
			},
			input: &iaas.Project{
				AreaId: utils.Ptr(iaas.AreaId{
					StaticAreaID: iaas.STATICAREAID_PUBLIC.Ptr(),
				}),
				Id: utils.Ptr(projectId),
			},
			expected: &DatasourceModel{
				Id:        types.StringValue(projectId),
				ProjectId: types.StringValue(projectId),
				AreaId:    types.StringValue("PUBLIC"),
			},
			isValid: true,
		},
		{
			description: "response_nil_fail",
			state:       &DatasourceModel{},
			input:       nil,
			expected:    &DatasourceModel{},
			isValid:     false,
		},
		{
			description: "no_project_id_fail",
			state:       &DatasourceModel{},
			input:       &iaas.Project{},
			expected:    &DatasourceModel{},
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceFields(tt.input, tt.state)
			if !tt.isValid && err == nil {
				t.Fatal("should have failed")
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
