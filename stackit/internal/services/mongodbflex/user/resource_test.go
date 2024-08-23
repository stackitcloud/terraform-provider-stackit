package mongodbflex

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
)

func TestMapFieldsCreate(t *testing.T) {
	tests := []struct {
		description string
		input       *mongodbflex.CreateUserResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{
					Id:       utils.Ptr("uid"),
					Password: utils.Ptr(""),
				},
			},
			Model{
				Id:         types.StringValue("pid,iid,uid"),
				UserId:     types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Database:   types.StringNull(),
				Roles:      types.SetNull(types.StringType),
				Password:   types.StringValue(""),
				Host:       types.StringNull(),
				Port:       types.Int64Null(),
				Uri:        types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{
					Id: utils.Ptr("uid"),
					Roles: &[]string{
						"role_1",
						"role_2",
						"",
					},
					Username: utils.Ptr("username"),
					Database: utils.Ptr("database"),
					Password: utils.Ptr("password"),
					Host:     utils.Ptr("host"),
					Port:     utils.Ptr(int64(1234)),
					Uri:      utils.Ptr("uri"),
				},
			},
			Model{
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
				Password: types.StringValue("password"),
				Host:     types.StringValue("host"),
				Port:     types.Int64Value(1234),
				Uri:      types.StringValue("uri"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{
					Id:       utils.Ptr("uid"),
					Roles:    &[]string{},
					Username: nil,
					Database: nil,
					Password: utils.Ptr(""),
					Host:     nil,
					Port:     utils.Ptr(int64(2123456789)),
					Uri:      nil,
				},
			},
			Model{
				Id:         types.StringValue("pid,iid,uid"),
				UserId:     types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Database:   types.StringNull(),
				Roles:      types.SetValueMust(types.StringType, []attr.Value{}),
				Password:   types.StringValue(""),
				Host:       types.StringNull(),
				Port:       types.Int64Value(2123456789),
				Uri:        types.StringNull(),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"nil_response_2",
			&mongodbflex.CreateUserResponse{},
			Model{},
			false,
		},
		{
			"no_resource_id",
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{},
			},
			Model{},
			false,
		},
		{
			"no_password",
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{
					Id: utils.Ptr("uid"),
				},
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFieldsCreate(tt.input, state)
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

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *mongodbflex.GetUserResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&mongodbflex.GetUserResponse{
				Item: &mongodbflex.InstanceResponseUser{},
			},
			Model{
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
					Port:     utils.Ptr(int64(1234)),
				},
			},
			Model{
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
					Port:     utils.Ptr(int64(2123456789)),
				},
			},
			Model{
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
			Model{},
			false,
		},
		{
			"nil_response_2",
			&mongodbflex.GetUserResponse{},
			Model{},
			false,
		},
		{
			"no_resource_id",
			&mongodbflex.GetUserResponse{
				Item: &mongodbflex.InstanceResponseUser{},
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
				UserId:     tt.expected.UserId,
			}
			err := mapFields(tt.input, state)
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
		inputRoles  []string
		expected    *mongodbflex.CreateUserPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&mongodbflex.CreateUserPayload{
				Roles:    &[]string{},
				Username: nil,
				Database: nil,
			},
			true,
		},
		{
			"default_values",
			&Model{
				Username: types.StringValue("username"),
				Database: types.StringValue("database"),
			},
			[]string{
				"role_1",
				"role_2",
			},
			&mongodbflex.CreateUserPayload{
				Roles: &[]string{
					"role_1",
					"role_2",
				},
				Username: utils.Ptr("username"),
				Database: utils.Ptr("database"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Username: types.StringNull(),
				Database: types.StringNull(),
			},
			[]string{
				"",
			},
			&mongodbflex.CreateUserPayload{
				Roles: &[]string{
					"",
				},
				Username: nil,
				Database: nil,
			},
			true,
		},
		{
			"nil_model",
			nil,
			[]string{},
			nil,
			false,
		},
		{
			"nil_roles",
			&Model{},
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputRoles)
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
