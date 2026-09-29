package iplists

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	lbiplists "github.com/stackitcloud/stackit-sdk-go/services/lbiplists/v1alphaapi"
)

func fixtureDataSourceModel(mods ...func(m *DataSourceModel)) DataSourceModel {
	resp := DataSourceModel{
		Id:          types.StringValue(testProjectId + "," + testRegion + "," + "test-ip-list"),
		ProjectId:   types.StringValue(testProjectId),
		Region:      types.StringValue(testRegion),
		Name:        types.StringValue("test-ip-list"),
		Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
		NumberOfIPs: types.Int32Value(2),
		ContentHash: types.StringValue("5b2ed459546d7f3e1eeef5c2a07b1b1b50db749b3e04a3bc72d146fe51f1e21c"),
	}
	for _, mod := range mods {
		mod(&resp)
	}
	return resp
}

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *lbiplists.GetIPListResponse
		state       *DataSourceModel
		expected    DataSourceModel
		isValid     bool
	}{
		{
			description: "basic_ip_list",
			input:       fixtureIPListResponse(),
			state: &DataSourceModel{
				ProjectId: types.StringValue(testProjectId),
			},
			expected: fixtureDataSourceModel(),
			isValid:  true,
		},
		{
			description: "minimal_ip_list",
			input: &lbiplists.GetIPListResponse{
				Name: "test-ip-list",
			},
			state: &DataSourceModel{
				ProjectId: types.StringValue(testProjectId),
			},
			expected: fixtureDataSourceModel(func(m *DataSourceModel) {
				m.Labels = types.MapNull(types.StringType)
				m.NumberOfIPs = types.Int32Null()
				m.ContentHash = types.StringNull()
			}),
			isValid: true,
		},
		{
			description: "empty_labels",
			input: fixtureIPListResponse(func(m *lbiplists.GetIPListResponse) {
				m.Labels = &map[string]string{}
			}),
			state: &DataSourceModel{
				ProjectId: types.StringValue(testProjectId),
			},
			expected: fixtureDataSourceModel(func(m *DataSourceModel) {
				m.Labels = types.MapNull(types.StringType)
			}),
			isValid: true,
		},
		{
			description: "name_falls_back_to_model",
			input: fixtureIPListResponse(func(m *lbiplists.GetIPListResponse) {
				m.Name = ""
			}),
			state: &DataSourceModel{
				ProjectId: types.StringValue(testProjectId),
				Name:      types.StringValue("test-ip-list"),
			},
			expected: fixtureDataSourceModel(),
			isValid:  true,
		},
		{
			description: "nil_response",
			input:       nil,
			state: &DataSourceModel{
				ProjectId: types.StringValue(testProjectId),
			},
			expected: DataSourceModel{},
			isValid:  false,
		},
		{
			description: "missing_name",
			input: fixtureIPListResponse(func(m *lbiplists.GetIPListResponse) {
				m.Name = ""
			}),
			state: &DataSourceModel{
				ProjectId: types.StringValue(testProjectId),
			},
			expected: DataSourceModel{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceFields(context.Background(), tt.input, tt.state, testRegion)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.isValid {
				if diff := cmp.Diff(&tt.expected, tt.state); diff != "" {
					t.Fatalf("Data mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
