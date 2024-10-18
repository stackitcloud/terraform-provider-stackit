package securitygrouprule

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

var fixtureModelIcmpParameters = types.ObjectValueMust(icmpParametersTypes, map[string]attr.Value{
	"code": types.Int64Value(1),
	"type": types.Int64Value(2),
})

var fixtureIcmpParameters = iaas.ICMPParameters{
	Code: utils.Ptr(int64(1)),
	Type: utils.Ptr(int64(2)),
}

var fixtureModelPortRange = types.ObjectValueMust(portRangeTypes, map[string]attr.Value{
	"max": types.Int64Value(2),
	"min": types.Int64Value(1),
})

var fixturePortRange = iaas.PortRange{
	Max: utils.Ptr(int64(2)),
	Min: utils.Ptr(int64(1)),
}

var fixtureModelProtocol = types.ObjectValueMust(protocolTypes, map[string]attr.Value{
	"name":   types.StringValue("name"),
	"number": types.Int64Value(1),
})

var fixtureProtocol = iaas.Protocol{
	Name:   utils.Ptr("name"),
	Number: utils.Ptr(int64(1)),
}

var fixtureModelCreateProtocol = types.ObjectValueMust(protocolTypes, map[string]attr.Value{
	"name":   types.StringValue("name"),
	"number": types.Int64Null(),
})

var fixtureCreateProtocol = iaas.CreateProtocol{
	String: utils.Ptr("name"),
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaas.SecurityGroupRule
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId:           types.StringValue("pid"),
				SecurityGroupId:     types.StringValue("sgid"),
				SecurityGroupRuleId: types.StringValue("sgrid"),
			},
			&iaas.SecurityGroupRule{
				Id: utils.Ptr("sgrid"),
			},
			Model{
				Id:                    types.StringValue("pid,sgid,sgrid"),
				ProjectId:             types.StringValue("pid"),
				SecurityGroupId:       types.StringValue("sgid"),
				SecurityGroupRuleId:   types.StringValue("sgrid"),
				Direction:             types.StringNull(),
				Description:           types.StringNull(),
				EtherType:             types.StringNull(),
				IpRange:               types.StringNull(),
				RemoteSecurityGroupId: types.StringNull(),
				IcmpParameters:        types.ObjectNull(icmpParametersTypes),
				PortRange:             types.ObjectNull(portRangeTypes),
				Protocol:              types.ObjectNull(protocolTypes),
			},
			true,
		},
		{
			"simple_values",
			Model{
				ProjectId:           types.StringValue("pid"),
				SecurityGroupId:     types.StringValue("sgid"),
				SecurityGroupRuleId: types.StringValue("sgrid"),
			},
			&iaas.SecurityGroupRule{
				Id:                    utils.Ptr("sgrid"),
				Description:           utils.Ptr("desc"),
				Direction:             utils.Ptr("ingress"),
				Ethertype:             utils.Ptr("ether"),
				IpRange:               utils.Ptr("iprange"),
				RemoteSecurityGroupId: utils.Ptr("remote"),
				IcmpParameters:        &fixtureIcmpParameters,
				PortRange:             &fixturePortRange,
				Protocol:              &fixtureProtocol,
			},
			Model{
				Id:                    types.StringValue("pid,sgid,sgrid"),
				ProjectId:             types.StringValue("pid"),
				SecurityGroupId:       types.StringValue("sgid"),
				SecurityGroupRuleId:   types.StringValue("sgrid"),
				Direction:             types.StringValue("ingress"),
				Description:           types.StringValue("desc"),
				EtherType:             types.StringValue("ether"),
				IpRange:               types.StringValue("iprange"),
				RemoteSecurityGroupId: types.StringValue("remote"),
				IcmpParameters:        fixtureModelIcmpParameters,
				PortRange:             fixtureModelPortRange,
				Protocol:              fixtureModelProtocol,
			},
			true,
		},
		{
			"protocol_only_with_name",
			Model{
				ProjectId:           types.StringValue("pid"),
				SecurityGroupId:     types.StringValue("sgid"),
				SecurityGroupRuleId: types.StringValue("sgrid"),
				Protocol: types.ObjectValueMust(protocolTypes, map[string]attr.Value{
					"name":   types.StringValue("name"),
					"number": types.Int64Null(),
				}),
			},
			&iaas.SecurityGroupRule{
				Id:       utils.Ptr("sgrid"),
				Protocol: &fixtureProtocol,
			},
			Model{
				Id:                    types.StringValue("pid,sgid,sgrid"),
				ProjectId:             types.StringValue("pid"),
				SecurityGroupId:       types.StringValue("sgid"),
				SecurityGroupRuleId:   types.StringValue("sgrid"),
				Direction:             types.StringNull(),
				Description:           types.StringNull(),
				EtherType:             types.StringNull(),
				IpRange:               types.StringNull(),
				RemoteSecurityGroupId: types.StringNull(),
				IcmpParameters:        types.ObjectNull(icmpParametersTypes),
				PortRange:             types.ObjectNull(portRangeTypes),
				Protocol:              fixtureModelProtocol,
			},
			true,
		},
		{
			"protocol_only_with_number",
			Model{
				ProjectId:           types.StringValue("pid"),
				SecurityGroupId:     types.StringValue("sgid"),
				SecurityGroupRuleId: types.StringValue("sgrid"),
				Protocol: types.ObjectValueMust(protocolTypes, map[string]attr.Value{
					"name":   types.StringNull(),
					"number": types.Int64Value(1),
				}),
			},
			&iaas.SecurityGroupRule{
				Id:       utils.Ptr("sgrid"),
				Protocol: &fixtureProtocol,
			},
			Model{
				Id:                    types.StringValue("pid,sgid,sgrid"),
				ProjectId:             types.StringValue("pid"),
				SecurityGroupId:       types.StringValue("sgid"),
				SecurityGroupRuleId:   types.StringValue("sgrid"),
				Direction:             types.StringNull(),
				Description:           types.StringNull(),
				EtherType:             types.StringNull(),
				IpRange:               types.StringNull(),
				RemoteSecurityGroupId: types.StringNull(),
				IcmpParameters:        types.ObjectNull(icmpParametersTypes),
				PortRange:             types.ObjectNull(portRangeTypes),
				Protocol:              fixtureModelProtocol,
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
				ProjectId:       types.StringValue("pid"),
				SecurityGroupId: types.StringValue("sgid"),
			},
			&iaas.SecurityGroupRule{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(tt.input, &tt.state)
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
		expected    *iaas.CreateSecurityGroupRulePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&iaas.CreateSecurityGroupRulePayload{},
			true,
		},
		{
			"default_ok",
			&Model{
				Description:    types.StringValue("desc"),
				Direction:      types.StringValue("ingress"),
				IcmpParameters: fixtureModelIcmpParameters,
				PortRange:      fixtureModelPortRange,
				Protocol:       fixtureModelCreateProtocol,
			},
			&iaas.CreateSecurityGroupRulePayload{
				Description:    utils.Ptr("desc"),
				Direction:      utils.Ptr("ingress"),
				IcmpParameters: &fixtureIcmpParameters,
				PortRange:      &fixturePortRange,
				Protocol:       &fixtureCreateProtocol,
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
			var icmpParameters *icmpParametersModel
			var portRange *portRangeModel
			var protocol *protocolModel
			if tt.input != nil {
				if !(tt.input.IcmpParameters.IsNull() || tt.input.IcmpParameters.IsUnknown()) {
					icmpParameters = &icmpParametersModel{}
					diags := tt.input.IcmpParameters.As(context.Background(), icmpParameters, basetypes.ObjectAsOptions{})
					if diags.HasError() {
						t.Fatalf("Error converting icmp parameters: %v", diags.Errors())
					}
				}

				if !(tt.input.PortRange.IsNull() || tt.input.PortRange.IsUnknown()) {
					portRange = &portRangeModel{}
					diags := tt.input.PortRange.As(context.Background(), portRange, basetypes.ObjectAsOptions{})
					if diags.HasError() {
						t.Fatalf("Error converting port range: %v", diags.Errors())
					}
				}

				if !(tt.input.Protocol.IsNull() || tt.input.Protocol.IsUnknown()) {
					protocol = &protocolModel{}
					diags := tt.input.Protocol.As(context.Background(), protocol, basetypes.ObjectAsOptions{})
					if diags.HasError() {
						t.Fatalf("Error converting protocol: %v", diags.Errors())
					}
				}
			}

			output, err := toCreatePayload(tt.input, icmpParameters, portRange, protocol)
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
