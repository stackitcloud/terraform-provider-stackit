package networkinterface

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
		input  *iaas.NIC
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    Model
		isValid     bool
	}{
		{
			description: "id_ok",
			args: args{
				state: Model{
					ProjectId:          types.StringValue("pid"),
					NetworkId:          types.StringValue("nid"),
					NetworkInterfaceId: types.StringValue("nicid"),
				},
				input: &iaas.NIC{
					Id: utils.Ptr("nicid"),
				},
				region: "eu01",
			},
			expected: Model{
				Id:                 types.StringValue("pid,eu01,nid,nicid"),
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
				Name:               types.StringNull(),
				AllowedAddresses:   types.ListNull(types.StringType),
				SecurityGroupIds:   types.ListNull(types.StringType),
				IPv4:               types.StringNull(),
				Security:           types.BoolNull(),
				Device:             types.StringNull(),
				Mac:                types.StringNull(),
				Type:               types.StringNull(),
				Labels:             types.MapNull(types.StringType),
				Region:             types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "values_ok",
			args: args{
				state: Model{
					ProjectId:          types.StringValue("pid"),
					NetworkId:          types.StringValue("nid"),
					NetworkInterfaceId: types.StringValue("nicid"),
					Region:             types.StringValue("eu01"),
				},
				input: &iaas.NIC{
					Id:   utils.Ptr("nicid"),
					Name: utils.Ptr("name"),
					AllowedAddresses: &[]iaas.AllowedAddressesInner{
						{
							String: utils.Ptr("aa1"),
						},
					},
					SecurityGroups: &[]string{
						"prefix1",
						"prefix2",
					},
					Ipv4:        utils.Ptr("ipv4"),
					Ipv6:        utils.Ptr("ipv6"),
					NicSecurity: utils.Ptr(true),
					Device:      utils.Ptr("device"),
					Mac:         utils.Ptr("mac"),
					Status:      utils.Ptr("status"),
					Type:        utils.Ptr("type"),
					Labels: &map[string]interface{}{
						"label1": "ref1",
					},
				},
				region: "eu02",
			},
			expected: Model{
				Id:                 types.StringValue("pid,eu02,nid,nicid"),
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
				Name:               types.StringValue("name"),
				AllowedAddresses: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("aa1"),
				}),
				SecurityGroupIds: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix1"),
					types.StringValue("prefix2"),
				}),
				IPv4:     types.StringValue("ipv4"),
				Security: types.BoolValue(true),
				Device:   types.StringValue("device"),
				Mac:      types.StringValue("mac"),
				Type:     types.StringValue("type"),
				Labels:   types.MapValueMust(types.StringType, map[string]attr.Value{"label1": types.StringValue("ref1")}),
				Region:   types.StringValue("eu02"),
			},
			isValid: true,
		},
		{
			description: "allowed_addresses_changed_outside_tf",
			args: args{
				state: Model{
					ProjectId:          types.StringValue("pid"),
					NetworkId:          types.StringValue("nid"),
					NetworkInterfaceId: types.StringValue("nicid"),
					AllowedAddresses: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("aa1"),
					}),
				},
				input: &iaas.NIC{
					Id: utils.Ptr("nicid"),
					AllowedAddresses: &[]iaas.AllowedAddressesInner{
						{
							String: utils.Ptr("aa2"),
						},
					},
				},
				region: "eu01",
			},
			expected: Model{
				Id:                 types.StringValue("pid,eu01,nid,nicid"),
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
				Name:               types.StringNull(),
				SecurityGroupIds:   types.ListNull(types.StringType),
				AllowedAddresses: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("aa2"),
				}),
				Labels: types.MapNull(types.StringType),
				Region: types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "empty_list_allowed_addresses",
			args: args{
				state: Model{
					ProjectId:          types.StringValue("pid"),
					NetworkId:          types.StringValue("nid"),
					NetworkInterfaceId: types.StringValue("nicid"),
					AllowedAddresses:   types.ListValueMust(types.StringType, []attr.Value{}),
				},
				input: &iaas.NIC{
					Id:               utils.Ptr("nicid"),
					AllowedAddresses: nil,
				},
				region: "eu01",
			},
			expected: Model{
				Id:                 types.StringValue("pid,eu01,nid,nicid"),
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
				Name:               types.StringNull(),
				SecurityGroupIds:   types.ListNull(types.StringType),
				AllowedAddresses:   types.ListValueMust(types.StringType, []attr.Value{}),
				Labels:             types.MapNull(types.StringType),
				Region:             types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "response_nil_fail",
			args: args{
				state: Model{},
				input: nil,
			},
			expected: Model{},
			isValid:  false,
		},
		{
			description: "no_resource_id",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
				},
				input: &iaas.NIC{},
			},
			expected: Model{},
			isValid:  false,
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
		expected    *iaas.CreateNicPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				SecurityGroupIds: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("sg1"),
					types.StringValue("sg2"),
				}),
				AllowedAddresses: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("aa1"),
				}),
				Security: types.BoolValue(true),
			},
			&iaas.CreateNicPayload{
				Name: utils.Ptr("name"),
				SecurityGroups: &[]string{
					"sg1",
					"sg2",
				},
				AllowedAddresses: &[]iaas.AllowedAddressesInner{
					{
						String: utils.Ptr("aa1"),
					},
				},
				NicSecurity: utils.Ptr(true),
			},
			true,
		},
		{
			"empty_allowed_addresses",
			&Model{
				Name: types.StringValue("name"),
				SecurityGroupIds: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("sg1"),
					types.StringValue("sg2"),
				}),

				AllowedAddresses: types.ListNull(types.StringType),
			},
			&iaas.CreateNicPayload{
				Name: utils.Ptr("name"),
				SecurityGroups: &[]string{
					"sg1",
					"sg2",
				},
				AllowedAddresses: nil,
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
		expected    *iaas.UpdateNicPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				SecurityGroupIds: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("sg1"),
					types.StringValue("sg2"),
				}),
				AllowedAddresses: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("aa1"),
				}),
				Security: types.BoolValue(true),
			},
			&iaas.UpdateNicPayload{
				Name: utils.Ptr("name"),
				SecurityGroups: &[]string{
					"sg1",
					"sg2",
				},
				AllowedAddresses: &[]iaas.AllowedAddressesInner{
					{
						String: utils.Ptr("aa1"),
					},
				},
				NicSecurity: utils.Ptr(true),
			},
			true,
		},
		{
			"empty_allowed_addresses",
			&Model{
				Name: types.StringValue("name"),
				SecurityGroupIds: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("sg1"),
					types.StringValue("sg2"),
				}),

				AllowedAddresses: types.ListNull(types.StringType),
			},
			&iaas.UpdateNicPayload{
				Name: utils.Ptr("name"),
				SecurityGroups: &[]string{
					"sg1",
					"sg2",
				},
				AllowedAddresses: utils.Ptr([]iaas.AllowedAddressesInner{}),
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
