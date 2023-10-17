package mongodbflex

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
)

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *mongodbflex.GetUserResponse
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			&mongodbflex.GetUserResponse{
				Item: &mongodbflex.InstanceResponseUser{},
			},
			DataSourceModel{
				Id:         types.StringValue("pid,iid,uid"),
				UserId:     types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Database:   types.StringNull(),
				Roles:      types.SetNull(types.StringType),
				Host:       types.StringNull(),
				Port:       types.Int64Null(),
			},
			true,
		},
		{
			"simple_values",
			&mongodbflex.GetUserResponse{
				Item: &mongodbflex.InstanceResponseUser{
					Roles: &[]string{
						"role_1",
						"role_2",
						"",
					},
					Username: utils.Ptr("username"),
					Database: utils.Ptr("database"),
					Host:     utils.Ptr("host"),
					Port:     utils.Ptr(int32(1234)),
				},
			},
			DataSourceModel{
				Id:         types.StringValue("pid,iid,uid"),
				UserId:     types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringValue("username"),
				Database:   types.StringValue("database"),
				Roles: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("role_1"),
					types.StringValue("role_2"),
					types.StringValue(""),
				}),
				Host: types.StringValue("host"),
				Port: types.Int64Value(1234),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&mongodbflex.GetUserResponse{
				Item: &mongodbflex.InstanceResponseUser{
					Id:       utils.Ptr("uid"),
					Roles:    &[]string{},
					Username: nil,
					Database: nil,
					Host:     nil,
					Port:     utils.Ptr(int32(2123456789)),
				},
			},
			DataSourceModel{
				Id:         types.StringValue("pid,iid,uid"),
				UserId:     types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Database:   types.StringNull(),
				Roles:      types.SetValueMust(types.StringType, []attr.Value{}),
				Host:       types.StringNull(),
				Port:       types.Int64Value(2123456789),
			},
			true,
		},
		{
			"nil_response",
			nil,
			DataSourceModel{},
			false,
		},
		{
			"nil_response_2",
			&mongodbflex.GetUserResponse{},
			DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			&mongodbflex.GetUserResponse{
				Item: &mongodbflex.InstanceResponseUser{},
			},
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
			err := mapDataSourceFields(tt.input, state)
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
