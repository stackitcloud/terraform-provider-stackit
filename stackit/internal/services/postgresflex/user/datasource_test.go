package postgresflex

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3beta1api"
)

func TestMapDataSourceFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description  string
		userResp     *postgresflex.GetUserResponse
		instanceResp *postgresflex.GetInstanceResponse
		region       string
		expected     DataSourceModel
		isValid      bool
	}{
		{
			description:  "default_values",
			userResp:     &postgresflex.GetUserResponse{},
			instanceResp: &postgresflex.GetInstanceResponse{},
			region:       testRegion,
			expected: DataSourceModel{
				Id:         types.StringValue("pid,region,iid,uid"),
				UserId:     types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringValue(""),
				Roles:      types.SetNull(types.StringType),
				Host:       types.StringValue(""),
				Port:       types.Int32Value(0),
				Region:     types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			userResp: &postgresflex.GetUserResponse{
				Roles: []string{
					"role_1",
					"role_2",
					"",
				},
				Name: "username",
			},
			instanceResp: &postgresflex.GetInstanceResponse{
				ConnectionInfo: postgresflex.InstanceConnectionInfo{
					Write: postgresflex.InstanceConnectionInfoWrite{
						Host: "host",
						Port: 1234,
					},
				},
			},
			region: testRegion,
			expected: DataSourceModel{
				Id:         types.StringValue("pid,region,iid,uid"),
				UserId:     types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringValue("username"),
				Roles: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("role_1"),
					types.StringValue("role_2"),
					types.StringValue(""),
				}),
				Host:   types.StringValue("host"),
				Port:   types.Int32Value(1234),
				Region: types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "null_fields_and_int_conversions",
			userResp: &postgresflex.GetUserResponse{
				Id:    123,
				Roles: []string{},
				Name:  "",
			},
			instanceResp: &postgresflex.GetInstanceResponse{
				ConnectionInfo: postgresflex.InstanceConnectionInfo{
					Write: postgresflex.InstanceConnectionInfoWrite{
						Host: "",
						Port: 2123456789,
					},
				},
			},
			region: testRegion,
			expected: DataSourceModel{
				Id:         types.StringValue("pid,region,iid,123"),
				UserId:     types.StringValue("123"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringValue(""),
				Roles:      types.SetValueMust(types.StringType, []attr.Value{}),
				Host:       types.StringValue(""),
				Port:       types.Int32Value(2123456789),
				Region:     types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description:  "nil_response",
			userResp:     nil,
			instanceResp: &postgresflex.GetInstanceResponse{},
			region:       testRegion,
			expected:     DataSourceModel{},
			isValid:      false,
		},
		{
			description:  "nil_response_2",
			userResp:     &postgresflex.GetUserResponse{},
			instanceResp: nil,
			region:       testRegion,
			expected:     DataSourceModel{},
			isValid:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &DataSourceModel{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
				UserId:     tt.expected.UserId,
			}
			err := mapDataSourceFields(tt.userResp, tt.instanceResp, state, tt.region)
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
