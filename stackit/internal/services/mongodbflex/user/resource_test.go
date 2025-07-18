package mongodbflex

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
)

const (
	testRegion = "eu02"
)

var (
	projectId  = uuid.NewString()
	instanceId = uuid.NewString()
	userId     = uuid.NewString()
)

func TestMapFieldsCreate(t *testing.T) {
	tests := []struct {
		description string
		input       *mongodbflex.CreateUserResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{
					Id:       utils.Ptr(userId),
					Password: utils.Ptr(""),
				},
			},
			testRegion,
			Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, testRegion, instanceId, userId)),
				UserId:     types.StringValue(userId),
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
				Username:   types.StringNull(),
				Database:   types.StringNull(),
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
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{
					Id: utils.Ptr(userId),
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
			testRegion,
			Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, testRegion, instanceId, userId)),
				UserId:     types.StringValue(userId),
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
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
				Region:   types.StringValue(testRegion),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{
					Id:       utils.Ptr(userId),
					Roles:    &[]string{},
					Username: nil,
					Database: nil,
					Password: utils.Ptr(""),
					Host:     nil,
					Port:     utils.Ptr(int64(2123456789)),
					Uri:      nil,
				},
			},
			testRegion,
			Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, testRegion, instanceId, userId)),
				UserId:     types.StringValue(userId),
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
				Username:   types.StringNull(),
				Database:   types.StringNull(),
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
			&mongodbflex.CreateUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{},
			},
			testRegion,
			Model{},
			false,
		},
		{
			"no_password",
			&mongodbflex.CreateUserResponse{
				Item: &mongodbflex.User{
					Id: utils.Ptr(userId),
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
	tests := []struct {
		description string
		input       *mongodbflex.GetUserResponse
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&mongodbflex.GetUserResponse{
				Item: &mongodbflex.InstanceResponseUser{},
			},
			testRegion,
			Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, testRegion, instanceId, userId)),
				UserId:     types.StringValue(userId),
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
				Username:   types.StringNull(),
				Database:   types.StringNull(),
				Roles:      types.SetNull(types.StringType),
				Host:       types.StringNull(),
				Port:       types.Int64Null(),
				Region:     types.StringValue(testRegion),
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
			testRegion,
			Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, testRegion, instanceId, userId)),
				UserId:     types.StringValue(userId),
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
				Username:   types.StringValue("username"),
				Database:   types.StringValue("database"),
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
			&mongodbflex.GetUserResponse{
				Item: &mongodbflex.InstanceResponseUser{
					Id:       utils.Ptr(userId),
					Roles:    &[]string{},
					Username: nil,
					Database: nil,
					Host:     nil,
					Port:     utils.Ptr(int64(2123456789)),
				},
			},
			testRegion,
			Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, testRegion, instanceId, userId)),
				UserId:     types.StringValue(userId),
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
				Username:   types.StringNull(),
				Database:   types.StringNull(),
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
			&mongodbflex.GetUserResponse{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&mongodbflex.GetUserResponse{
				Item: &mongodbflex.InstanceResponseUser{},
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

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		inputRoles  []string
		expected    *mongodbflex.UpdateUserPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&mongodbflex.UpdateUserPayload{
				Roles:    &[]string{},
				Database: nil,
			},
			true,
		},
		{
			"simple values",
			&Model{
				Username: types.StringValue("username"),
				Database: types.StringValue("database"),
			},
			[]string{
				"role_1",
				"role_2",
			},
			&mongodbflex.UpdateUserPayload{
				Roles: &[]string{
					"role_1",
					"role_2",
				},
				Database: utils.Ptr("database"),
			},
			true,
		},
		{
			"null_fields",
			&Model{
				Username: types.StringNull(),
				Database: types.StringNull(),
			},
			[]string{
				"",
			},
			&mongodbflex.UpdateUserPayload{
				Roles: &[]string{
					"",
				},
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
