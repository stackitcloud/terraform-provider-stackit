package securitygroup

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapFields(t *testing.T) {
	type args struct {
		state  Model
		input  *iaas.SecurityGroup
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    Model
		isValid     bool
	}{
		{
			description: "default_values",
			args: args{
				state: Model{
					ProjectId:       types.StringValue("pid"),
					SecurityGroupId: types.StringValue("sgid"),
				},
				input: &iaas.SecurityGroup{
					Id: utils.Ptr("sgid"),
				},
				region: "eu01",
			},
			expected: Model{
				Id:              types.StringValue("pid,eu01,sgid"),
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
				Name:            types.StringNull(),
				Labels:          types.MapNull(types.StringType),
				Description:     types.StringNull(),
				Stateful:        types.BoolNull(),
				Region:          types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			args: args{
				state: Model{
					ProjectId:       types.StringValue("pid"),
					SecurityGroupId: types.StringValue("sgid"),
					Region:          types.StringValue("eu01"),
				},
				input: &iaas.SecurityGroup{
					Id:       utils.Ptr("sgid"),
					Name:     utils.Ptr("name"),
					Stateful: utils.Ptr(true),
					Labels: &map[string]interface{}{
						"key": "value",
					},
					Description: utils.Ptr("desc"),
				},
				region: "eu02",
			},
			expected: Model{
				Id:              types.StringValue("pid,eu02,sgid"),
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
				Name:            types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description: types.StringValue("desc"),
				Stateful:    types.BoolValue(true),
				Region:      types.StringValue("eu02"),
			},
			isValid: true,
		},
		{
			description: "empty_labels",
			args: args{
				state: Model{
					ProjectId:       types.StringValue("pid"),
					SecurityGroupId: types.StringValue("sgid"),
				},
				input: &iaas.SecurityGroup{
					Id:     utils.Ptr("sgid"),
					Labels: &map[string]interface{}{},
				},
				region: "eu01",
			},
			expected: Model{
				Id:              types.StringValue("pid,eu01,sgid"),
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
				Name:            types.StringNull(),
				Labels:          types.MapNull(types.StringType),
				Description:     types.StringNull(),
				Stateful:        types.BoolNull(),
				Region:          types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "response_nil_fail",
		},
		{
			description: "no_resource_id",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
				},
				input: &iaas.SecurityGroup{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.args.input, &tt.args.state, tt.args.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.args.state, tt.expected)
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
		expected    *iaas.CreateSecurityGroupPayload
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
			&iaas.CreateSecurityGroupPayload{
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
		expected    *iaas.UpdateSecurityGroupPayload
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
			&iaas.UpdateSecurityGroupPayload{
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
