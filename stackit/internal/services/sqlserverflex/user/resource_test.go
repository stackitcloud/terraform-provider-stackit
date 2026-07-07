package sqlserverflex

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3beta2api"
)

func TestMapFieldsCreate(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *sqlserverflex.CreateUserResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&sqlserverflex.CreateUserResponse{
				Id:       1,
				Password: "secret",
			},
			testRegion,
			Model{
				Id:                types.StringValue("pid,region,iid,1"),
				UserId:            types.StringValue("1"),
				InstanceId:        types.StringValue("iid"),
				ProjectId:         types.StringValue("pid"),
				Username:          types.StringValue(""),
				Roles:             types.SetNull(types.StringType),
				Password:          types.StringValue("secret"),
				Host:              types.StringValue(""),
				Port:              types.Int32Value(0),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"simple_values",
			&sqlserverflex.CreateUserResponse{
				Id: 1,
				Roles: []string{
					"role_1",
					"role_2",
					"",
				},
				Username: "username",
				Password: "password",
				Host:     "host",
				Port:     1234,
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,1"),
				UserId:     types.StringValue("1"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringValue("username"),
				Roles: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("role_1"),
					types.StringValue("role_2"),
					types.StringValue(""),
				}),
				Password:          types.StringValue("password"),
				Host:              types.StringValue("host"),
				Port:              types.Int32Value(1234),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&sqlserverflex.CreateUserResponse{
				Id:       1,
				Roles:    []string{},
				Password: "",
				Port:     2123456789,
			},
			testRegion,
			Model{
				Id:                types.StringValue("pid,region,iid,1"),
				UserId:            types.StringValue("1"),
				InstanceId:        types.StringValue("iid"),
				ProjectId:         types.StringValue("pid"),
				Username:          types.StringValue(""),
				Roles:             types.SetValueMust(types.StringType, []attr.Value{}),
				Password:          types.StringValue(""),
				Host:              types.StringValue(""),
				Port:              types.Int32Value(2123456789),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
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
			&sqlserverflex.CreateUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&sqlserverflex.CreateUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_password",
			&sqlserverflex.CreateUserResponse{},
			testRegion,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:         tt.expected.ProjectId,
				InstanceId:        tt.expected.InstanceId,
				RotateWhenChanged: types.MapNull(types.StringType),
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
		})
	}
}

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *sqlserverflex.GetUserResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&sqlserverflex.GetUserResponse{},
			testRegion,
			Model{
				Id:                types.StringValue("pid,region,iid,uid"),
				UserId:            types.StringValue("uid"),
				InstanceId:        types.StringValue("iid"),
				ProjectId:         types.StringValue("pid"),
				Username:          types.StringValue(""),
				Roles:             types.SetNull(types.StringType),
				Host:              types.StringValue(""),
				Port:              types.Int32Value(0),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
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
				Port:     1234,
			},
			testRegion,
			Model{
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
				Host:              types.StringValue("host"),
				Port:              types.Int32Value(1234),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
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
			Model{
				Id:                types.StringValue("pid,region,iid,1"),
				UserId:            types.StringValue("1"),
				InstanceId:        types.StringValue("iid"),
				ProjectId:         types.StringValue("pid"),
				Username:          types.StringValue(""),
				Roles:             types.SetValueMust(types.StringType, []attr.Value{}),
				Host:              types.StringValue(""),
				Port:              types.Int32Value(2123456789),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
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
			&sqlserverflex.GetUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&sqlserverflex.GetUserResponse{},
			testRegion,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:         tt.expected.ProjectId,
				InstanceId:        tt.expected.InstanceId,
				UserId:            tt.expected.UserId,
				RotateWhenChanged: types.MapNull(types.StringType),
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
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		inputRoles  []string
		expected    *sqlserverflex.CreateUserPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&sqlserverflex.CreateUserPayload{
				Roles:    []string{},
				Username: "",
			},
			true,
		},
		{
			"default_values",
			&Model{
				Username: types.StringValue("username"),
			},
			[]string{
				"role_1",
				"role_2",
			},
			&sqlserverflex.CreateUserPayload{
				Roles: []string{
					"role_1",
					"role_2",
				},
				Username: "username",
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Username: types.StringNull(),
			},
			[]string{
				"",
			},
			&sqlserverflex.CreateUserPayload{
				Roles: []string{
					"",
				},
				Username: "",
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
			&Model{
				Username: types.StringValue("username"),
			},
			[]string{},
			&sqlserverflex.CreateUserPayload{
				Roles:    []string{},
				Username: "username",
			},
			true,
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
