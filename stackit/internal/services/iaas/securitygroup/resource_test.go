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

//func fixtureRulesModel() basetypes.ListValue {
//	return types.ListValueMust(types.ObjectType{AttrTypes: ruleTypes}, []attr.Value{
//		types.ObjectValueMust(ruleTypes, map[string]attr.Value{
//			"description":              types.StringValue("desc"),
//			"direction":                types.StringValue("direction"),
//			"ether_type":               types.StringValue("ether"),
//			"id":                       types.StringValue("id"),
//			"ip_range":                 types.StringValue("range"),
//			"remote_security_group_id": types.StringValue("rsgid"),
//			"security_group_id":        types.StringValue("sgid"),
//			"icmp_parameters": types.ObjectValueMust(icmpParametersTypes, map[string]attr.Value{
//				"code": types.Int64Value(1),
//				"type": types.Int64Value(2),
//			}),
//			"port_range": types.ObjectValueMust(portRangeTypes, map[string]attr.Value{
//				"max": types.Int64Value(3),
//				"min": types.Int64Value(2),
//			}),
//			"protocol": types.ObjectValueMust(protocolTypes, map[string]attr.Value{
//				"name":     types.StringValue("name"),
//				"protocol": types.Int64Value(2),
//			}),
//		}),
//	})
//}
//
//func fixtureRulesResponse() iaasalpha.SecurityGroupRule {
//	return iaasalpha.SecurityGroupRule{
//		Description:           utils.Ptr("desc"),
//		Direction:             utils.Ptr("direction"),
//		Ethertype:             utils.Ptr("ether"),
//		Id:                    utils.Ptr("id"),
//		IpRange:               utils.Ptr("range"),
//		RemoteSecurityGroupId: utils.Ptr("rsgid"),
//		SecurityGroupId:       utils.Ptr("sgid"),
//		IcmpParameters: &iaasalpha.ICMPParameters{
//			Code: utils.Ptr(int64(1)),
//			Type: utils.Ptr(int64(2)),
//		},
//		PortRange: &iaasalpha.PortRange{
//			Max: utils.Ptr(int64(3)),
//			Min: utils.Ptr(int64(2)),
//		},
//		Protocol: &iaasalpha.V1SecurityGroupRuleProtocol{
//			Name:     utils.Ptr("name"),
//			Protocol: utils.Ptr(int64(2)),
//		},
//	}
//}

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
				Rules:           types.ListNull(types.ObjectType{AttrTypes: ruleTypes}),
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
				Rules:       types.ListNull(types.ObjectType{AttrTypes: ruleTypes}),
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
				Rules:           types.ListNull(types.ObjectType{AttrTypes: ruleTypes}),
			},
			true,
		},
		//{
		//	"with rules",
		//	Model{
		//		ProjectId:       types.StringValue("pid"),
		//		SecurityGroupId: types.StringValue("sgid"),
		//	},
		//	&iaasalpha.SecurityGroup{
		//		Id: utils.Ptr("sgid"),
		//		Rules: &[]iaasalpha.SecurityGroupRule{
		//			fixtureRulesResponse(),
		//		},
		//	},
		//	Model{
		//		Id:              types.StringValue("pid,sgid"),
		//		ProjectId:       types.StringValue("pid"),
		//		SecurityGroupId: types.StringValue("sgid"),
		//		Name:            types.StringNull(),
		//		Labels:          types.MapNull(types.StringType),
		//		Description:     types.StringNull(),
		//		Stateful:        types.BoolNull(),
		//		Rules: types.ListValueMust(types.ObjectType{AttrTypes: ruleTypes}, []attr.Value{
		//			fixtureRulesModel(),
		//		}),
		//	},
		//	true,
		//},
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
