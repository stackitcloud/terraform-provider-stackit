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
	tests := []struct {
		description string
		state       Model
		input       *iaas.NIC
		expected    Model
		isValid     bool
	}{
		{
			"id_ok",
			Model{
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
			},
			&iaas.NIC{
				Id: utils.Ptr("nicid"),
			},
			Model{
				Id:                 types.StringValue("pid,nid,nicid"),
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
			},
			true,
		},
		{
			"values_ok",
			Model{
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
			},
			&iaas.NIC{
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
			Model{
				Id:                 types.StringValue("pid,nid,nicid"),
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
			},
			true,
		},
		{
			"allowed_addresses_changed_outside_tf",
			Model{
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
				AllowedAddresses: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("aa1"),
				}),
			},
			&iaas.NIC{
				Id: utils.Ptr("nicid"),
				AllowedAddresses: &[]iaas.AllowedAddressesInner{
					{
						String: utils.Ptr("aa2"),
					},
				},
			},
			Model{
				Id:                 types.StringValue("pid,nid,nicid"),
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
				Name:               types.StringNull(),
				SecurityGroupIds:   types.ListNull(types.StringType),
				AllowedAddresses: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("aa2"),
				}),
				Labels: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"empty_list_allowed_addresses",
			Model{
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
				AllowedAddresses:   types.ListValueMust(types.StringType, []attr.Value{}),
			},
			&iaas.NIC{
				Id:               utils.Ptr("nicid"),
				AllowedAddresses: nil,
			},
			Model{
				Id:                 types.StringValue("pid,nid,nicid"),
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				NetworkInterfaceId: types.StringValue("nicid"),
				Name:               types.StringNull(),
				SecurityGroupIds:   types.ListNull(types.StringType),
				AllowedAddresses:   types.ListValueMust(types.StringType, []attr.Value{}),
				Labels:             types.MapNull(types.StringType),
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
			&iaas.NIC{},
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
