package redis

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/redis"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *redis.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&redis.Instance{},
			Model{
				Id:                 types.StringValue("pid,iid"),
				InstanceId:         types.StringValue("iid"),
				ProjectId:          types.StringValue("pid"),
				PlanId:             types.StringNull(),
				Name:               types.StringNull(),
				CfGuid:             types.StringNull(),
				CfSpaceGuid:        types.StringNull(),
				DashboardUrl:       types.StringNull(),
				ImageUrl:           types.StringNull(),
				CfOrganizationGuid: types.StringNull(),
				Parameters:         types.ObjectNull(parametersTypes),
			},
			true,
		},
		{
			"simple_values",
			&redis.Instance{
				PlanId:             utils.Ptr("plan"),
				CfGuid:             utils.Ptr("cf"),
				CfSpaceGuid:        utils.Ptr("space"),
				DashboardUrl:       utils.Ptr("dashboard"),
				ImageUrl:           utils.Ptr("image"),
				InstanceId:         utils.Ptr("iid"),
				Name:               utils.Ptr("name"),
				CfOrganizationGuid: utils.Ptr("org"),
				Parameters: &map[string]interface{}{
					"sgw_acl": "acl",
				},
			},
			Model{
				Id:                 types.StringValue("pid,iid"),
				InstanceId:         types.StringValue("iid"),
				ProjectId:          types.StringValue("pid"),
				PlanId:             types.StringValue("plan"),
				Name:               types.StringValue("name"),
				CfGuid:             types.StringValue("cf"),
				CfSpaceGuid:        types.StringValue("space"),
				DashboardUrl:       types.StringValue("dashboard"),
				ImageUrl:           types.StringValue("image"),
				CfOrganizationGuid: types.StringValue("org"),
				Parameters: types.ObjectValueMust(parametersTypes, map[string]attr.Value{
					"sgw_acl": types.StringValue("acl"),
				}),
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
			"no_resource_id",
			&redis.Instance{},
			Model{},
			false,
		},
		{
			"wrong_param_types_1",
			&redis.Instance{
				Parameters: &map[string]interface{}{
					"sgw_acl": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&redis.Instance{
				Parameters: &map[string]interface{}{
					"sgw_acl": 1,
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
		description     string
		input           *Model
		inputParameters *parametersModel
		expected        *redis.CreateInstancePayload
		isValid         bool
	}{
		{
			"default_values",
			&Model{},
			&parametersModel{},
			&redis.CreateInstancePayload{
				Parameters: &redis.InstanceParameters{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:   types.StringValue("name"),
				PlanId: types.StringValue("plan"),
			},
			&parametersModel{
				SgwAcl: types.StringValue("sgw"),
			},
			&redis.CreateInstancePayload{
				InstanceName: utils.Ptr("name"),
				Parameters: &redis.InstanceParameters{
					SgwAcl: utils.Ptr("sgw"),
				},
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name:   types.StringValue(""),
				PlanId: types.StringValue(""),
			},
			&parametersModel{
				SgwAcl: types.StringNull(),
			},
			&redis.CreateInstancePayload{
				InstanceName: utils.Ptr(""),
				Parameters: &redis.InstanceParameters{
					SgwAcl: nil,
				},
				PlanId: utils.Ptr(""),
			},
			true,
		},
		{
			"nil_model",
			nil,
			&parametersModel{},
			nil,
			false,
		},
		{
			"nil_parameters",
			&Model{
				Name:   types.StringValue("name"),
				PlanId: types.StringValue("plan"),
			},
			nil,
			&redis.CreateInstancePayload{
				InstanceName: utils.Ptr("name"),
				PlanId:       utils.Ptr("plan"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputParameters)
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
		description     string
		input           *Model
		inputParameters *parametersModel
		expected        *redis.UpdateInstancePayload
		isValid         bool
	}{
		{
			"default_values",
			&Model{},
			&parametersModel{},
			&redis.UpdateInstancePayload{
				Parameters: &redis.InstanceParameters{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				PlanId: types.StringValue("plan"),
			},
			&parametersModel{
				SgwAcl: types.StringValue("sgw"),
			},
			&redis.UpdateInstancePayload{
				Parameters: &redis.InstanceParameters{
					SgwAcl: utils.Ptr("sgw"),
				},
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				PlanId: types.StringValue(""),
			},
			&parametersModel{
				SgwAcl: types.StringNull(),
			},
			&redis.UpdateInstancePayload{
				Parameters: &redis.InstanceParameters{
					SgwAcl: nil,
				},
				PlanId: utils.Ptr(""),
			},
			true,
		},
		{
			"nil_model",
			nil,
			&parametersModel{},
			nil,
			false,
		},
		{
			"nil_parameters",
			&Model{
				PlanId: types.StringValue("plan"),
			},
			nil,
			&redis.UpdateInstancePayload{
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input, tt.inputParameters)
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
