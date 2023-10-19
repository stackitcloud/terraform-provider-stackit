package secretsmanager

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/secretsmanager"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description   string
		input         *secretsmanager.User
		modelPassword *string
		expected      Model
		isValid       bool
	}{
		{
			"default_values",
			&secretsmanager.User{
				Id: utils.Ptr("uid"),
			},
			nil,
			Model{
				Id:           types.StringValue("pid,iid,uid"),
				UserId:       types.StringValue("uid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Description:  types.StringNull(),
				WriteEnabled: types.BoolNull(),
				Username:     types.StringNull(),
				Password:     types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&secretsmanager.User{
				Id:          utils.Ptr("uid"),
				Description: utils.Ptr("description"),
				Write:       utils.Ptr(false),
				Username:    utils.Ptr("username"),
				Password:    utils.Ptr("password"),
			},
			nil,
			Model{
				Id:           types.StringValue("pid,iid,uid"),
				UserId:       types.StringValue("uid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Description:  types.StringValue("description"),
				WriteEnabled: types.BoolValue(false),
				Username:     types.StringValue("username"),
				Password:     types.StringValue("password"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&secretsmanager.User{},
			nil,
			Model{},
			false,
		},
		{
			"no_password_in_response_1",
			&secretsmanager.User{
				Id:          utils.Ptr("uid"),
				Description: utils.Ptr("description"),
				Write:       utils.Ptr(false),
				Username:    utils.Ptr("username"),
			},
			utils.Ptr("password"),
			Model{
				Id:           types.StringValue("pid,iid,uid"),
				UserId:       types.StringValue("uid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Description:  types.StringValue("description"),
				WriteEnabled: types.BoolValue(false),
				Username:     types.StringValue("username"),
				Password:     types.StringValue("password"),
			},
			true,
		},
		{
			"no_password_in_response_2",
			&secretsmanager.User{
				Id:          utils.Ptr("uid"),
				Description: utils.Ptr("description"),
				Write:       utils.Ptr(false),
				Username:    utils.Ptr("username"),
				Password:    utils.Ptr(""),
			},
			utils.Ptr("password"),
			Model{
				Id:           types.StringValue("pid,iid,uid"),
				UserId:       types.StringValue("uid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Description:  types.StringValue("description"),
				WriteEnabled: types.BoolValue(false),
				Username:     types.StringValue("username"),
				Password:     types.StringValue("password"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			if tt.modelPassword != nil {
				state.Password = types.StringPointerValue(tt.modelPassword)
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
		expected    *secretsmanager.CreateUserPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&secretsmanager.CreateUserPayload{
				Description: nil,
				Write:       nil,
			},
			true,
		},
		{
			"simple_values",
			&Model{
				Description:  types.StringValue("description"),
				WriteEnabled: types.BoolValue(false),
			},
			&secretsmanager.CreateUserPayload{
				Description: utils.Ptr("description"),
				Write:       utils.Ptr(false),
			},
			true,
		},
		{
			"null_fields",
			&Model{
				Description:  types.StringNull(),
				WriteEnabled: types.BoolNull(),
			},
			&secretsmanager.CreateUserPayload{
				Description: nil,
				Write:       nil,
			},
			true,
		},
		{
			"empty_fields",
			&Model{
				Description:  types.StringValue(""),
				WriteEnabled: types.BoolNull(),
			},
			&secretsmanager.CreateUserPayload{
				Description: utils.Ptr(""),
				Write:       nil,
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
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
		expected    *secretsmanager.UpdateUserPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&secretsmanager.UpdateUserPayload{
				Write: nil,
			},
			true,
		},
		{
			"simple_values",
			&Model{
				WriteEnabled: types.BoolValue(false),
			},
			&secretsmanager.UpdateUserPayload{
				Write: utils.Ptr(false),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input)
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
