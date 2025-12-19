package postgresflexalpha

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/terraform-provider-stackit/pkg/postgresflexalpha"
)

func TestMapDataSourceFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *postgresflexalpha.GetUserResponse
		region      string
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			&postgresflexalpha.GetUserResponse{},
			testRegion,
			DataSourceModel{
				Id:         types.StringValue("pid,region,iid,1"),
				UserId:     types.Int64Value(1),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Roles:      types.SetNull(types.StringType),
				Host:       types.StringNull(),
				Port:       types.Int64Null(),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			&postgresflexalpha.GetUserResponse{
				Roles: &[]postgresflexalpha.UserRole{
					"role_1",
					"role_2",
					"",
				},
				Name: utils.Ptr("username"),
				Host: utils.Ptr("host"),
				Port: utils.Ptr(int64(1234)),
			},
			testRegion,
			DataSourceModel{
				Id:         types.StringValue("pid,region,iid,1"),
				UserId:     types.Int64Value(1),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringValue("username"),
				Roles: types.SetValueMust(
					types.StringType, []attr.Value{
						types.StringValue("role_1"),
						types.StringValue("role_2"),
						types.StringValue(""),
					},
				),
				Host:             types.StringValue("host"),
				Port:             types.Int64Value(1234),
				Region:           types.StringValue(testRegion),
				Status:           types.StringNull(),
				ConnectionString: types.StringNull(),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&postgresflexalpha.GetUserResponse{
				Id:               utils.Ptr(int64(1)),
				Roles:            &[]postgresflexalpha.UserRole{},
				Name:             nil,
				Host:             nil,
				Port:             utils.Ptr(int64(2123456789)),
				Status:           utils.Ptr("status"),
				ConnectionString: utils.Ptr("connection_string"),
			},
			testRegion,
			DataSourceModel{
				Id:               types.StringValue("pid,region,iid,1"),
				UserId:           types.Int64Value(1),
				InstanceId:       types.StringValue("iid"),
				ProjectId:        types.StringValue("pid"),
				Username:         types.StringNull(),
				Roles:            types.SetValueMust(types.StringType, []attr.Value{}),
				Host:             types.StringNull(),
				Port:             types.Int64Value(2123456789),
				Region:           types.StringValue(testRegion),
				Status:           types.StringValue("status"),
				ConnectionString: types.StringValue("connection_string"),
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
			&postgresflexalpha.GetUserResponse{},
			testRegion,
			DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			&postgresflexalpha.GetUserResponse{},
			testRegion,
			DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.description, func(t *testing.T) {
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
			},
		)
	}
}
