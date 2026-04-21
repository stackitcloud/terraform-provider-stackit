package postgresflex

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v2api"
)

func TestMapFieldsCreate(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *postgresflex.CreateUserResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&postgresflex.CreateUserResponse{
				Item: &postgresflex.User{
					Id:       new("uid"),
					Password: new(""),
				},
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,uid"),
				UserId:     types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Roles:      types.SetNull(types.StringType),
				Password:   types.StringValue(""),
				Host:       types.StringNull(),
				Port:       types.Int64Null(),
				Uri:        types.StringNull(),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			&postgresflex.CreateUserResponse{
				Item: &postgresflex.User{
					Id: new("uid"),
					Roles: []string{
						"role_1",
						"role_2",
						"",
					},
					Username: new("username"),
					Password: new("password"),
					Host:     new("host"),
					Port:     new(int64(1234)),
					Uri:      new("uri"),
				},
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
				Password: types.StringValue("password"),
				Host:     types.StringValue("host"),
				Port:     types.Int64Value(1234),
				Uri:      types.StringValue("uri"),
				Region:   types.StringValue(testRegion),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&postgresflex.CreateUserResponse{
				Item: &postgresflex.User{
					Id:       new("uid"),
					Roles:    []string{},
					Username: nil,
					Password: new(""),
					Host:     nil,
					Port:     new(int64(2123456789)),
					Uri:      nil,
				},
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,uid"),
				UserId:     types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Roles:      types.SetValueMust(types.StringType, []attr.Value{}),
				Password:   types.StringValue(""),
				Host:       types.StringNull(),
				Port:       types.Int64Value(2123456789),
				Uri:        types.StringNull(),
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
			&postgresflex.CreateUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&postgresflex.CreateUserResponse{
				Item: &postgresflex.User{},
			},
			testRegion,
			Model{},
			false,
		},
		{
			"no_password",
			&postgresflex.CreateUserResponse{
				Item: &postgresflex.User{
					Id: new("uid"),
				},
			},
			testRegion,
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
		input       *postgresflex.GetUserResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&postgresflex.GetUserResponse{
				Item: &postgresflex.UserResponse{},
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,uid"),
				UserId:     types.StringValue("uid"),
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
			&postgresflex.GetUserResponse{
				Item: &postgresflex.UserResponse{
					Roles: []string{
						"role_1",
						"role_2",
						"",
					},
					Username: new("username"),
					Host:     new("host"),
					Port:     new(int64(1234)),
				},
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
				Host:   types.StringValue("host"),
				Port:   types.Int64Value(1234),
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&postgresflex.GetUserResponse{
				Item: &postgresflex.UserResponse{
					Id:       new("uid"),
					Roles:    []string{},
					Username: nil,
					Host:     nil,
					Port:     new(int64(2123456789)),
				},
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,uid"),
				UserId:     types.StringValue("uid"),
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
			&postgresflex.GetUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&postgresflex.GetUserResponse{
				Item: &postgresflex.UserResponse{},
			},
			testRegion,
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
		expected    *postgresflex.CreateUserPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&postgresflex.CreateUserPayload{
				Roles:    []string{},
				Username: nil,
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
			&postgresflex.CreateUserPayload{
				Roles: []string{
					"role_1",
					"role_2",
				},
				Username: new("username"),
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
			&postgresflex.CreateUserPayload{
				Roles: []string{
					"",
				},
				Username: nil,
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

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		inputRoles  []string
		expected    *postgresflex.UpdateUserPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&postgresflex.UpdateUserPayload{
				Roles: []string{},
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
			&postgresflex.UpdateUserPayload{
				Roles: []string{
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
			[]string{
				"",
			},
			&postgresflex.UpdateUserPayload{
				Roles: []string{
					"",
				},
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
		})
	}
}
