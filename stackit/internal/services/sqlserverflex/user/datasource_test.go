package sqlserverflex

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"
)

func TestMapDataSourceFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *sqlserverflex.GetUserResponse
		region      string
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			&sqlserverflex.GetUserResponse{},
			testRegion,
			DataSourceModel{
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
			true,
		},
		{
			"simple_values",
			&sqlserverflex.GetUserResponse{
				Roles: []string{
					"role_1",
					"role_2",
					"",
				},
				Username: "username",
				Host:     "host",
				Port:     int32(1234),
			},
			testRegion,
			DataSourceModel{
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
			true,
		},
		{
			"null_fields_and_int_conversions",
			&sqlserverflex.GetUserResponse{
				Id:    1,
				Roles: []string{},
				Port:  2123456789,
			},
			testRegion,
			DataSourceModel{
				Id:         types.StringValue("pid,region,iid,1"),
				UserId:     types.StringValue("1"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringValue(""),
				Roles:      types.SetValueMust(types.StringType, []attr.Value{}),
				Host:       types.StringValue(""),
				Port:       types.Int32Value(2123456789),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			nil,
			testRegion,
			DataSourceModel{},
			false,
		},
		{
			"nil_response_2",
			&sqlserverflex.GetUserResponse{},
			testRegion,
			DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			&sqlserverflex.GetUserResponse{},
			testRegion,
			DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &DataSourceModel{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
				UserId:     tt.expected.UserId,
			}
			err := mapDataSourceFields(tt.input, state, tt.region)
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
