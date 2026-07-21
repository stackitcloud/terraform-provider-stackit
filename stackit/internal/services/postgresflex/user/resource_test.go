package postgresflex

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3beta1api"
)

func TestMapFieldsCreate(t *testing.T) {
	const (
		projectId  = "pid"
		instanceId = "iid"
		testRegion = "region"
		userId     = 123
		userIdStr  = "123"
	)
	var id = fmt.Sprintf("%s,%s,%s,%s", projectId, testRegion, instanceId, userIdStr)
	tests := []struct {
		description    string
		createUserResp *postgresflex.CreateUserResponse
		userResp       *postgresflex.GetUserResponse
		instanceResp   *postgresflex.GetInstanceResponse
		region         string
		expected       Model
		isValid        bool
	}{
		{
			description: "default_values",
			createUserResp: &postgresflex.CreateUserResponse{
				Id:       userId,
				Password: "",
			},
			userResp: &postgresflex.GetUserResponse{},
			instanceResp: &postgresflex.GetInstanceResponse{
				ConnectionInfo: postgresflex.InstanceConnectionInfo{
					Write: postgresflex.InstanceConnectionInfoWrite{
						Host: "localhost",
						Port: 5432,
					},
				},
			},
			region: testRegion,
			expected: Model{
				Id:                types.StringValue(id),
				UserId:            types.StringValue(userIdStr),
				InstanceId:        types.StringValue(instanceId),
				ProjectId:         types.StringValue(projectId),
				Username:          types.StringValue(""),
				Roles:             types.SetNull(types.StringType),
				Password:          types.StringValue(""),
				Host:              types.StringValue("localhost"),
				Port:              types.Int32Value(5432),
				Uri:               types.StringValue("postgresql://:@localhost:5432/stackit"),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			createUserResp: &postgresflex.CreateUserResponse{
				Id:       123,
				Name:     "username",
				Password: "password",
			},
			userResp: &postgresflex.GetUserResponse{
				Roles: []string{
					"role_1",
					"role_2",
					"",
				},
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
			expected: Model{
				Id:         types.StringValue("pid,region,iid,123"),
				UserId:     types.StringValue("123"),
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
				Uri:               types.StringValue("postgresql://username:password@host:1234/stackit"),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "null_fields_and_int_conversions",
			createUserResp: &postgresflex.CreateUserResponse{
				Id:       123,
				Name:     "",
				Password: "",
			},
			userResp: &postgresflex.GetUserResponse{
				Roles: []string{},
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
			expected: Model{
				Id:                types.StringValue("pid,region,iid,123"),
				UserId:            types.StringValue("123"),
				InstanceId:        types.StringValue("iid"),
				ProjectId:         types.StringValue("pid"),
				Username:          types.StringValue(""),
				Roles:             types.SetValueMust(types.StringType, []attr.Value{}),
				Password:          types.StringValue(""),
				Host:              types.StringValue(""),
				Port:              types.Int32Value(2123456789),
				Uri:               types.StringValue("postgresql://:@:2123456789/stackit"),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description:    "nil_response",
			createUserResp: nil,
			userResp:       &postgresflex.GetUserResponse{},
			instanceResp:   &postgresflex.GetInstanceResponse{},
			region:         testRegion,
			expected:       Model{},
			isValid:        false,
		},
		{
			description:    "nil_response_2",
			createUserResp: &postgresflex.CreateUserResponse{},
			userResp:       nil,
			instanceResp:   &postgresflex.GetInstanceResponse{},
			region:         testRegion,
			expected:       Model{},
			isValid:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:         tt.expected.ProjectId,
				InstanceId:        tt.expected.InstanceId,
				RotateWhenChanged: types.MapNull(types.StringType),
			}
			err := mapFieldsCreate(tt.createUserResp, tt.userResp, tt.instanceResp, state, tt.region)
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
		description  string
		userResp     *postgresflex.GetUserResponse
		instanceResp *postgresflex.GetInstanceResponse
		region       string
		expected     Model
		isValid      bool
	}{
		{
			description:  "default_values",
			userResp:     &postgresflex.GetUserResponse{},
			instanceResp: &postgresflex.GetInstanceResponse{},
			region:       testRegion,
			expected: Model{
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
			expected: Model{
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
			expected: Model{
				Id:                types.StringValue("pid,region,iid,uid"),
				UserId:            types.StringValue("uid"),
				InstanceId:        types.StringValue("iid"),
				ProjectId:         types.StringValue("pid"),
				Username:          types.StringValue(""),
				Roles:             types.SetValueMust(types.StringType, []attr.Value{}),
				Host:              types.StringValue(""),
				Port:              types.Int32Value(2123456789),
				Region:            types.StringValue(testRegion),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description:  "nil_response",
			userResp:     nil,
			instanceResp: &postgresflex.GetInstanceResponse{},
			region:       testRegion,
			expected:     Model{},
			isValid:      false,
		},
		{
			description:  "nil_response_2",
			userResp:     &postgresflex.GetUserResponse{},
			instanceResp: nil,
			region:       testRegion,
			expected:     Model{},
			isValid:      false,
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
			err := mapFields(tt.userResp, tt.instanceResp, state, tt.region)
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
				Roles: []string{},
				Name:  "",
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
				Name: "username",
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
				Name: "",
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
		expected    *postgresflex.PartialUpdateUserPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&postgresflex.PartialUpdateUserPayload{
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
			&postgresflex.PartialUpdateUserPayload{
				Name: new("username"),
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
			&postgresflex.PartialUpdateUserPayload{
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
