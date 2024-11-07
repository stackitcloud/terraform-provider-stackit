package network

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
		input       *iaas.Network
		expected    Model
		isValid     bool
	}{
		{
			"id_ok",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
			},
			&iaas.Network{
				NetworkId: utils.Ptr("nid"),
				Gateway:   iaas.NewNullableString(nil),
			},
			Model{
				Id:               types.StringValue("pid,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				Nameservers:      types.ListNull(types.StringType),
				IPv4Nameservers:  types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Null(),
				IPv4Gateway:      types.StringNull(),
				IPv4Prefix:       types.StringNull(),
				Prefixes:         types.ListNull(types.StringType),
				IPv4Prefixes:     types.ListNull(types.StringType),
				IPv6Nameservers:  types.ListNull(types.StringType),
				IPv6PrefixLength: types.Int64Null(),
				IPv6Gateway:      types.StringNull(),
				IPv6Prefix:       types.StringNull(),
				IPv6Prefixes:     types.ListNull(types.StringType),
				PublicIP:         types.StringNull(),
				Labels:           types.MapNull(types.StringType),
				Routed:           types.BoolNull(),
			},
			true,
		},
		{
			"values_ok",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
			},
			&iaas.Network{
				NetworkId: utils.Ptr("nid"),
				Name:      utils.Ptr("name"),
				Nameservers: &[]string{
					"ns1",
					"ns2",
				},
				Prefixes: &[]string{
					"prefix1",
					"prefix2",
				},
				NameserversV6: &[]string{
					"ns1",
					"ns2",
				},
				PrefixesV6: &[]string{
					"prefix1",
					"prefix2",
				},
				PublicIp: utils.Ptr("publicIp"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Routed:    utils.Ptr(true),
				Gateway:   iaas.NewNullableString(utils.Ptr("gateway")),
				Gatewayv6: iaas.NewNullableString(utils.Ptr("gateway")),
			},
			Model{
				Id:        types.StringValue("pid,nid"),
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Name:      types.StringValue("name"),
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv4PrefixLength: types.Int64Null(),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix1"),
					types.StringValue("prefix2"),
				}),
				IPv4Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix1"),
					types.StringValue("prefix2"),
				}),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv6PrefixLength: types.Int64Null(),
				IPv6Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix1"),
					types.StringValue("prefix2"),
				}),
				PublicIP: types.StringValue("publicIp"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(true),
				IPv4Gateway: types.StringValue("gateway"),
				IPv6Gateway: types.StringValue("gateway"),
			},
			true,
		},
		{
			"ipv4_nameservers_changed_outside_tf",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
			},
			&iaas.Network{
				NetworkId: utils.Ptr("nid"),
				Nameservers: &[]string{
					"ns2",
					"ns3",
				},
			},
			Model{
				Id:              types.StringValue("pid,nid"),
				ProjectId:       types.StringValue("pid"),
				NetworkId:       types.StringValue("nid"),
				Name:            types.StringNull(),
				IPv6Prefixes:    types.ListNull(types.StringType),
				IPv6Nameservers: types.ListNull(types.StringType),
				Prefixes:        types.ListNull(types.StringType),
				IPv4Prefixes:    types.ListNull(types.StringType),
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns2"),
					types.StringValue("ns3"),
				}),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns2"),
					types.StringValue("ns3"),
				}),
				Labels: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"ipv6_nameservers_changed_outside_tf",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
			},
			&iaas.Network{
				NetworkId: utils.Ptr("nid"),
				NameserversV6: &[]string{
					"ns2",
					"ns3",
				},
			},
			Model{
				Id:              types.StringValue("pid,nid"),
				ProjectId:       types.StringValue("pid"),
				NetworkId:       types.StringValue("nid"),
				Name:            types.StringNull(),
				IPv6Prefixes:    types.ListNull(types.StringType),
				IPv4Nameservers: types.ListNull(types.StringType),
				Prefixes:        types.ListNull(types.StringType),
				IPv4Prefixes:    types.ListNull(types.StringType),
				Nameservers:     types.ListNull(types.StringType),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns2"),
					types.StringValue("ns3"),
				}),
				Labels: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"ipv4_prefixes_changed_outside_tf",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix1"),
					types.StringValue("prefix2"),
				}),
			},
			&iaas.Network{
				NetworkId: utils.Ptr("nid"),
				Prefixes: &[]string{
					"prefix2",
					"prefix3",
				},
			},
			Model{
				Id:               types.StringValue("pid,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				IPv6Nameservers:  types.ListNull(types.StringType),
				IPv6PrefixLength: types.Int64Null(),
				IPv6Prefixes:     types.ListNull(types.StringType),
				Labels:           types.MapNull(types.StringType),
				Nameservers:      types.ListNull(types.StringType),
				IPv4Nameservers:  types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Null(),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix2"),
					types.StringValue("prefix3"),
				}),
				IPv4Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix2"),
					types.StringValue("prefix3"),
				}),
			},
			true,
		},
		{
			"ipv6_prefixes_changed_outside_tf",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				IPv6Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix1"),
					types.StringValue("prefix2"),
				}),
			},
			&iaas.Network{
				NetworkId: utils.Ptr("nid"),
				PrefixesV6: &[]string{
					"prefix2",
					"prefix3",
				},
			},
			Model{
				Id:               types.StringValue("pid,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				IPv4Nameservers:  types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Null(),
				Prefixes:         types.ListNull(types.StringType),
				IPv4Prefixes:     types.ListNull(types.StringType),
				Labels:           types.MapNull(types.StringType),
				Nameservers:      types.ListNull(types.StringType),
				IPv6Nameservers:  types.ListNull(types.StringType),
				IPv6PrefixLength: types.Int64Null(),
				IPv6Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix2"),
					types.StringValue("prefix3"),
				}),
			},
			true,
		},
		{
			"ipv4_ipv6_gateway_nil",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
			},
			&iaas.Network{
				NetworkId: utils.Ptr("nid"),
			},
			Model{
				Id:               types.StringValue("pid,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				Nameservers:      types.ListNull(types.StringType),
				IPv4Nameservers:  types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Null(),
				IPv4Gateway:      types.StringNull(),
				Prefixes:         types.ListNull(types.StringType),
				IPv4Prefixes:     types.ListNull(types.StringType),
				IPv6Nameservers:  types.ListNull(types.StringType),
				IPv6PrefixLength: types.Int64Null(),
				IPv6Gateway:      types.StringNull(),
				IPv6Prefixes:     types.ListNull(types.StringType),
				PublicIP:         types.StringNull(),
				Labels:           types.MapNull(types.StringType),
				Routed:           types.BoolNull(),
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
			&iaas.Network{},
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
		expected    *iaas.CreateNetworkPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv4PrefixLength: types.Int64Value(24),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv4Gateway: types.StringValue("gateway"),
				IPv4Prefix:  types.StringValue("prefix"),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.CreateNetworkAddressFamily{
					Ipv4: &iaas.CreateNetworkIPv4Body{
						Nameservers: &[]string{
							"ns1",
							"ns2",
						},
						PrefixLength: utils.Ptr(int64(24)),
						Gateway:      iaas.NewNullableString(utils.Ptr("gateway")),
						Prefix:       utils.Ptr("prefix"),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Routed: utils.Ptr(false),
			},
			true,
		},
		{
			"ipv4_nameservers_okay",
			&Model{
				Name: types.StringValue("name"),
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv4PrefixLength: types.Int64Value(24),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv4Gateway: types.StringValue("gateway"),
				IPv4Prefix:  types.StringValue("prefix"),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.CreateNetworkAddressFamily{
					Ipv4: &iaas.CreateNetworkIPv4Body{
						Nameservers: &[]string{
							"ns1",
							"ns2",
						},
						PrefixLength: utils.Ptr(int64(24)),
						Gateway:      iaas.NewNullableString(utils.Ptr("gateway")),
						Prefix:       utils.Ptr("prefix"),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Routed: utils.Ptr(false),
			},
			true,
		},
		{
			"ipv6_default_ok",
			&Model{
				Name: types.StringValue("name"),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv6PrefixLength: types.Int64Value(24),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv6Gateway: types.StringValue("gateway"),
				IPv6Prefix:  types.StringValue("prefix"),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.CreateNetworkAddressFamily{
					Ipv6: &iaas.CreateNetworkIPv6Body{
						Nameservers: &[]string{
							"ns1",
							"ns2",
						},
						PrefixLength: utils.Ptr(int64(24)),
						Gateway:      iaas.NewNullableString(utils.Ptr("gateway")),
						Prefix:       utils.Ptr("prefix"),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Routed: utils.Ptr(false),
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
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
		state       Model
		expected    *iaas.PartialUpdateNetworkPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(true),
				IPv4Gateway: types.StringValue("gateway"),
			},
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaas.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.UpdateNetworkAddressFamily{
					Ipv4: &iaas.UpdateNetworkIPv4Body{
						Nameservers: &[]string{
							"ns1",
							"ns2",
						},
						Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
		{
			"ipv4_nameservers_okay",
			&Model{
				Name: types.StringValue("name"),
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(true),
				IPv4Gateway: types.StringValue("gateway"),
			},
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaas.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.UpdateNetworkAddressFamily{
					Ipv4: &iaas.UpdateNetworkIPv4Body{
						Nameservers: &[]string{
							"ns1",
							"ns2",
						},
						Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
		{
			"ipv4_gateway_nil",
			&Model{
				Name: types.StringValue("name"),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed: types.BoolValue(true),
			},
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaas.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.UpdateNetworkAddressFamily{
					Ipv4: &iaas.UpdateNetworkIPv4Body{
						Nameservers: &[]string{
							"ns1",
							"ns2",
						},
						Gateway: iaas.NewNullableString(nil),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
		{
			"ipv6_default_ok",
			&Model{
				Name: types.StringValue("name"),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(true),
				IPv6Gateway: types.StringValue("gateway"),
			},
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaas.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.UpdateNetworkAddressFamily{
					Ipv6: &iaas.UpdateNetworkIPv6Body{
						Nameservers: &[]string{
							"ns1",
							"ns2",
						},
						Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
		{
			"ipv6_gateway_nil",
			&Model{
				Name: types.StringValue("name"),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed: types.BoolValue(true),
			},
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaas.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.UpdateNetworkAddressFamily{
					Ipv6: &iaas.UpdateNetworkIPv6Body{
						Nameservers: &[]string{
							"ns1",
							"ns2",
						},
						Gateway: iaas.NewNullableString(nil),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, &tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
