// Copyright (c) STACKIT

package sqlserverflexalpha

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/terraform-provider-stackit/pkg/sqlserverflexalpha"
)

func TestMapFieldsCreate(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *sqlserverflexalpha.CreateUserResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&sqlserverflexalpha.CreateUserResponse{
				Id:       utils.Ptr(int64(1)),
				Password: utils.Ptr(""),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,1"),
				UserId:     types.Int64Value(1),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Roles:      types.SetNull(types.StringType),
				Password:   types.StringValue(""),
				Host:       types.StringNull(),
				Port:       types.Int64Null(),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			&sqlserverflexalpha.CreateUserResponse{
				Id: utils.Ptr(int64(2)),
				Roles: &[]sqlserverflexalpha.UserRole{
					"role_1",
					"role_2",
					"",
				},
				Username:        utils.Ptr("username"),
				Password:        utils.Ptr("password"),
				Host:            utils.Ptr("host"),
				Port:            utils.Ptr(int64(1234)),
				Status:          utils.Ptr("status"),
				DefaultDatabase: utils.Ptr("default_db"),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,2"),
				UserId:     types.Int64Value(2),
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
				Password:        types.StringValue("password"),
				Host:            types.StringValue("host"),
				Port:            types.Int64Value(1234),
				Region:          types.StringValue(testRegion),
				Status:          types.StringValue("status"),
				DefaultDatabase: types.StringValue("default_db"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&sqlserverflexalpha.CreateUserResponse{
				Id:       utils.Ptr(int64(3)),
				Roles:    &[]sqlserverflexalpha.UserRole{},
				Username: nil,
				Password: utils.Ptr(""),
				Host:     nil,
				Port:     utils.Ptr(int64(2123456789)),
			},
			testRegion,
			Model{
				Id:              types.StringValue("pid,region,iid,3"),
				UserId:          types.Int64Value(3),
				InstanceId:      types.StringValue("iid"),
				ProjectId:       types.StringValue("pid"),
				Username:        types.StringNull(),
				Roles:           types.SetValueMust(types.StringType, []attr.Value{}),
				Password:        types.StringValue(""),
				Host:            types.StringNull(),
				Port:            types.Int64Value(2123456789),
				Region:          types.StringValue(testRegion),
				DefaultDatabase: types.StringNull(),
				Status:          types.StringNull(),
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
			&sqlserverflexalpha.CreateUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&sqlserverflexalpha.CreateUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_password",
			&sqlserverflexalpha.CreateUserResponse{
				Id: utils.Ptr(int64(1)),
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
				err := mapFieldsCreate(tt.input, state, tt.region)
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
	const testRegion = "region"
	tests := []struct {
		description string
		input       *sqlserverflexalpha.GetUserResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&sqlserverflexalpha.GetUserResponse{},
			testRegion,
			Model{
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
			&sqlserverflexalpha.GetUserResponse{
				Roles: &[]sqlserverflexalpha.UserRole{
					"role_1",
					"role_2",
					"",
				},
				Username: utils.Ptr("username"),
				Host:     utils.Ptr("host"),
				Port:     utils.Ptr(int64(1234)),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,2"),
				UserId:     types.Int64Value(2),
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
				Host:   types.StringValue("host"),
				Port:   types.Int64Value(1234),
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&sqlserverflexalpha.GetUserResponse{
				Id:       utils.Ptr(int64(1)),
				Roles:    &[]sqlserverflexalpha.UserRole{},
				Username: nil,
				Host:     nil,
				Port:     utils.Ptr(int64(2123456789)),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,1"),
				UserId:     types.Int64Value(1),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Roles:      types.SetValueMust(types.StringType, []attr.Value{}),
				Host:       types.StringNull(),
				Port:       types.Int64Value(2123456789),
				Region:     types.StringValue(testRegion),
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
			&sqlserverflexalpha.GetUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&sqlserverflexalpha.GetUserResponse{},
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
		inputRoles  []sqlserverflexalpha.UserRole
		expected    *sqlserverflexalpha.CreateUserRequestPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			[]sqlserverflexalpha.UserRole{},
			&sqlserverflexalpha.CreateUserRequestPayload{
				Roles:    &[]sqlserverflexalpha.UserRole{},
				Username: nil,
			},
			true,
		},
		{
			"default_values",
			&Model{
				Username: types.StringValue("username"),
			},
			[]sqlserverflexalpha.UserRole{
				"role_1",
				"role_2",
			},
			&sqlserverflexalpha.CreateUserRequestPayload{
				Roles: &[]sqlserverflexalpha.UserRole{
					"role_1",
					"role_2",
				},
				Username: utils.Ptr("username"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Username: types.StringNull(),
			},
			[]sqlserverflexalpha.UserRole{
				"",
			},
			&sqlserverflexalpha.CreateUserRequestPayload{
				Roles: &[]sqlserverflexalpha.UserRole{
					"",
				},
				Username: nil,
			},
			true,
		},
		{
			"nil_model",
			nil,
			[]sqlserverflexalpha.UserRole{},
			nil,
			false,
		},
		{
			"nil_roles",
			&Model{
				Username: types.StringValue("username"),
			},
			[]sqlserverflexalpha.UserRole{},
			&sqlserverflexalpha.CreateUserRequestPayload{
				Roles:    &[]sqlserverflexalpha.UserRole{},
				Username: utils.Ptr("username"),
			},
			true,
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
