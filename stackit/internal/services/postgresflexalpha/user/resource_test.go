package postgresflexalpha

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

func TestMapFieldsCreate(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *postgresflexalpha.CreateUserResponse
		updateRoles *postgresflexalpha.UpdateUserRequestPayload
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&postgresflexalpha.CreateUserResponse{
				Id:       utils.Ptr(int64(1)),
				Password: utils.Ptr(""),
			},
			&postgresflexalpha.UpdateUserRequestPayload{
				Roles: &[]postgresflexalpha.UserRole{},
			},

			testRegion,
			Model{
				Id:               types.StringValue("pid,region,iid,1"),
				UserId:           types.Int64Value(1),
				InstanceId:       types.StringValue("iid"),
				ProjectId:        types.StringValue("pid"),
				Username:         types.StringNull(),
				Roles:            types.SetValueMust(types.StringType, []attr.Value{}),
				Password:         types.StringValue(""),
				Host:             types.StringNull(),
				Port:             types.Int64Null(),
				Uri:              types.StringNull(),
				Region:           types.StringValue(testRegion),
				Status:           types.StringNull(),
				ConnectionString: types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&postgresflexalpha.CreateUserResponse{
				Id:               utils.Ptr(int64(1)),
				Name:             utils.Ptr("username"),
				Password:         utils.Ptr("password"),
				ConnectionString: utils.Ptr("connection_string"),
				Status:           utils.Ptr("status"),
			},
			&postgresflexalpha.UpdateUserRequestPayload{
				Roles: &[]postgresflexalpha.UserRole{},
			},
			testRegion,
			Model{
				Id:               types.StringValue("pid,region,iid,1"),
				UserId:           types.Int64Value(1),
				InstanceId:       types.StringValue("iid"),
				ProjectId:        types.StringValue("pid"),
				Username:         types.StringValue("username"),
				Roles:            types.SetValueMust(types.StringType, []attr.Value{}),
				Password:         types.StringValue("password"),
				Host:             types.StringNull(),
				Port:             types.Int64Null(),
				Uri:              types.StringNull(),
				Region:           types.StringValue(testRegion),
				Status:           types.StringValue("status"),
				ConnectionString: types.StringValue("connection_string"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&postgresflexalpha.CreateUserResponse{
				Id:               utils.Ptr(int64(1)),
				Name:             nil,
				Password:         utils.Ptr(""),
				ConnectionString: nil,
				Status:           nil,
			},
			&postgresflexalpha.UpdateUserRequestPayload{
				Roles: &[]postgresflexalpha.UserRole{},
			},
			testRegion,
			Model{
				Id:               types.StringValue("pid,region,iid,1"),
				UserId:           types.Int64Value(1),
				InstanceId:       types.StringValue("iid"),
				ProjectId:        types.StringValue("pid"),
				Username:         types.StringNull(),
				Roles:            types.SetValueMust(types.StringType, []attr.Value{}),
				Password:         types.StringValue(""),
				Host:             types.StringNull(),
				Port:             types.Int64Null(),
				Uri:              types.StringNull(),
				Region:           types.StringValue(testRegion),
				Status:           types.StringNull(),
				ConnectionString: types.StringNull(),
			},
			true,
		},
		{
			"nil_response",
			nil,
			nil,
			testRegion,
			Model{},
			false,
		},
		{
			"nil_response_2",
			&postgresflexalpha.CreateUserResponse{},
			&postgresflexalpha.UpdateUserRequestPayload{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&postgresflexalpha.CreateUserResponse{},
			&postgresflexalpha.UpdateUserRequestPayload{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_password",
			&postgresflexalpha.CreateUserResponse{
				Id: utils.Ptr(int64(1)),
			},
			&postgresflexalpha.UpdateUserRequestPayload{
				Roles: &[]postgresflexalpha.UserRole{},
			},
			testRegion,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.description, func(t *testing.T) {
				state := &Model{
					ProjectId:  tt.expected.ProjectId,
					InstanceId: tt.expected.InstanceId,
				}
				var roles *[]postgresflexalpha.UserRole
				if tt.updateRoles != nil {
					roles = tt.updateRoles.Roles
				}
				err := mapFieldsCreate(tt.input, roles, state, tt.region)
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

func TestMapFields(t *testing.T) {
	t.Skip("Skipping - needs refactoring")
	const testRegion = "region"
	tests := []struct {
		description string
		input       *postgresflexalpha.GetUserResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&postgresflexalpha.GetUserResponse{},
			testRegion,
			Model{
				Id:               types.StringValue("pid,region,iid,1"),
				UserId:           types.Int64Value(int64(1)),
				InstanceId:       types.StringValue("iid"),
				ProjectId:        types.StringValue("pid"),
				Username:         types.StringNull(),
				Roles:            types.SetNull(types.StringType),
				Host:             types.StringNull(),
				Port:             types.Int64Null(),
				Region:           types.StringValue(testRegion),
				Status:           types.StringNull(),
				ConnectionString: types.StringNull(),
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
			Model{
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
				Id:   utils.Ptr(int64(1)),
				Name: nil,
				Host: nil,
				Port: utils.Ptr(int64(2123456789)),
			},
			testRegion,
			Model{
				Id:               types.StringValue("pid,region,iid,1"),
				UserId:           types.Int64Value(1),
				InstanceId:       types.StringValue("iid"),
				ProjectId:        types.StringValue("pid"),
				Username:         types.StringNull(),
				Roles:            types.SetValueMust(types.StringType, []attr.Value{}),
				Host:             types.StringNull(),
				Port:             types.Int64Value(2123456789),
				Region:           types.StringValue(testRegion),
				Status:           types.StringNull(),
				ConnectionString: types.StringNull(),
			},
			true,
		},
		{
			"nil_response",
			nil,
			testRegion,
			Model{},
			false,
		},
		{
			"nil_response_2",
			&postgresflexalpha.GetUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&postgresflexalpha.GetUserResponse{},
			testRegion,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.description, func(t *testing.T) {
				state := &Model{
					ProjectId:  tt.expected.ProjectId,
					InstanceId: tt.expected.InstanceId,
					UserId:     tt.expected.UserId,
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
			},
		)
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		inputRoles  *[]string
		expected    *postgresflexalpha.CreateUserRequestPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&[]string{},
			&postgresflexalpha.CreateUserRequestPayload{
				Name:  nil,
				Roles: &[]postgresflexalpha.UserRole{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				Username: types.StringValue("username"),
			},
			&[]string{
				"role_1",
				"role_2",
			},
			&postgresflexalpha.CreateUserRequestPayload{
				Name: utils.Ptr("username"),
				Roles: &[]postgresflexalpha.UserRole{
					"role_1",
					"role_2",
				},
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Username: types.StringNull(),
			},
			&[]string{
				"",
			},
			&postgresflexalpha.CreateUserRequestPayload{
				Roles: &[]postgresflexalpha.UserRole{
					"",
				},
				Name: nil,
			},
			true,
		},
		{
			"nil_model",
			nil,
			&[]string{},
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
		t.Run(
			tt.description, func(t *testing.T) {
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
			},
		)
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		inputRoles  *[]string
		expected    *postgresflexalpha.UpdateUserRequestPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&[]string{},
			&postgresflexalpha.UpdateUserRequestPayload{
				Roles: &[]postgresflexalpha.UserRole{},
			},
			true,
		},
		{
			"default_values",
			&Model{
				Username: types.StringValue("username"),
			},
			&[]string{
				"role_1",
				"role_2",
			},
			&postgresflexalpha.UpdateUserRequestPayload{
				Roles: &[]postgresflexalpha.UserRole{
					"role_1",
					"role_2",
				},
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Username: types.StringNull(),
			},
			&[]string{
				"",
			},
			&postgresflexalpha.UpdateUserRequestPayload{
				Roles: &[]postgresflexalpha.UserRole{
					"",
				},
			},
			true,
		},
		{
			"nil_model",
			nil,
			&[]string{},
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
		t.Run(
			tt.description, func(t *testing.T) {
				output, err := toUpdatePayload(tt.input, tt.inputRoles)
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
			},
		)
	}
}
