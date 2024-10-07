package securitygroup

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaasalpha.SecurityGroup
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
			},
			&iaasalpha.SecurityGroup{
				Id: utils.Ptr("sgid"),
			},
			Model{
				Id:              types.StringValue("pid,sgid"),
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
				Name:            types.StringNull(),
				Labels:          types.MapNull(types.StringType),
				Description:     types.StringNull(),
				Stateful:        types.BoolNull(),
			},
			true,
		},
		{
			"simple_values",
			Model{
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
			},
			// &sourceModel{},
			&iaasalpha.SecurityGroup{
				Id:       utils.Ptr("sgid"),
				Name:     utils.Ptr("name"),
				Stateful: utils.Ptr(true),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Description: utils.Ptr("desc"),
			},
			Model{
				Id:              types.StringValue("pid,sgid"),
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
				Name:            types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description: types.StringValue("desc"),
				Stateful:    types.BoolValue(true),
			},
			true,
		},
		{
			"empty_labels",
			Model{
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
			},
			&iaasalpha.SecurityGroup{
				Id:     utils.Ptr("sgid"),
				Labels: &map[string]interface{}{},
			},
			Model{
				Id:              types.StringValue("pid,sgid"),
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
				Name:            types.StringNull(),
				Labels:          types.MapNull(types.StringType),
				Description:     types.StringNull(),
				Stateful:        types.BoolNull(),
			},
			true,
		},
		{
			"response_nil_fail",
			Model{},
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				ProjectId: types.StringValue("pid"),
			},
			&iaasalpha.SecurityGroup{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.state, tt.expected)
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
		expected    *iaasalpha.CreateSecurityGroupPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name:     types.StringValue("name"),
				Stateful: types.BoolValue(true),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description: types.StringValue("desc"),
			},
			&iaasalpha.CreateSecurityGroupPayload{
				Name:     utils.Ptr("name"),
				Stateful: utils.Ptr(true),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Description: utils.Ptr("desc"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)
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
		expected    *iaasalpha.V1alpha1UpdateSecurityGroupPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description: types.StringValue("desc"),
			},
			&iaasalpha.V1alpha1UpdateSecurityGroupPayload{
				Name: utils.Ptr("name"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Description: utils.Ptr("desc"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, types.MapNull(types.StringType))
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
