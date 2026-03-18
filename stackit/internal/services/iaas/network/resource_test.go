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
	const testRegion = "region"
	tests := []struct {
		description string
		state       Model
		input       *iaas.Network
		region      string
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
				Id: utils.Ptr("nid"),
				Ipv4: &iaas.NetworkIPv4{
					Gateway: iaas.NewNullableString(nil),
				},
			},
			testRegion,
			Model{
				Id:               types.StringValue("pid,region,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				IPv4Nameservers:  types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Null(),
				IPv4Gateway:      types.StringNull(),
				IPv4Prefix:       types.StringNull(),
				IPv4Prefixes:     types.ListNull(types.StringType),
				IPv6Nameservers:  types.ListNull(types.StringType),
				IPv6PrefixLength: types.Int64Null(),
				IPv6Gateway:      types.StringNull(),
				IPv6Prefix:       types.StringNull(),
				IPv6Prefixes:     types.ListNull(types.StringType),
				PublicIP:         types.StringNull(),
				Labels:           types.MapNull(types.StringType),
				Routed:           types.BoolNull(),
				Region:           types.StringValue(testRegion),
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
				Id:   utils.Ptr("nid"),
				Name: utils.Ptr("name"),
				Ipv4: &iaas.NetworkIPv4{
					Nameservers: utils.Ptr([]string{"ns1", "ns2"}),
					Prefixes: utils.Ptr(
						[]string{
							"192.168.42.0/24",
							"10.100.10.0/16",
						},
					),
					PublicIp: utils.Ptr("publicIp"),
					Gateway:  iaas.NewNullableString(utils.Ptr("gateway")),
				},
				Ipv6: &iaas.NetworkIPv6{
					Nameservers: utils.Ptr([]string{"ns1", "ns2"}),
					Prefixes: utils.Ptr([]string{
						"fd12:3456:789a:1::/64",
						"fd12:3456:789b:1::/64",
					}),
					Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Routed: utils.Ptr(true),
				Dhcp:   utils.Ptr(true),
			},
			testRegion,
			Model{
				Id:        types.StringValue("pid,region,nid"),
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Name:      types.StringValue("name"),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv4PrefixLength: types.Int64Value(24),
				IPv4Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("192.168.42.0/24"),
					types.StringValue("10.100.10.0/16"),
				}),
				IPv4Prefix: types.StringValue("192.168.42.0/24"),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv6PrefixLength: types.Int64Value(64),
				IPv6Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("fd12:3456:789a:1::/64"),
					types.StringValue("fd12:3456:789b:1::/64"),
				}),
				IPv6Prefix: types.StringValue("fd12:3456:789a:1::/64"),
				PublicIP:   types.StringValue("publicIp"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(true),
				IPv4Gateway: types.StringValue("gateway"),
				IPv6Gateway: types.StringValue("gateway"),
				Region:      types.StringValue(testRegion),
				DHCP:        types.BoolValue(true),
			},
			true,
		},
		{
			"ipv4_nameservers_changed_outside_tf",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
			},
			&iaas.Network{
				Id: utils.Ptr("nid"),
				Ipv4: &iaas.NetworkIPv4{
					Nameservers: utils.Ptr([]string{
						"ns2",
						"ns3",
					}),
				},
			},
			testRegion,
			Model{
				Id:              types.StringValue("pid,region,nid"),
				ProjectId:       types.StringValue("pid"),
				NetworkId:       types.StringValue("nid"),
				Name:            types.StringNull(),
				IPv6Prefixes:    types.ListNull(types.StringType),
				IPv6Nameservers: types.ListNull(types.StringType),
				IPv4Prefixes:    types.ListNull(types.StringType),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns2"),
					types.StringValue("ns3"),
				}),
				Labels: types.MapNull(types.StringType),
				Region: types.StringValue(testRegion),
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
				Id: utils.Ptr("nid"),
				Ipv6: &iaas.NetworkIPv6{
					Nameservers: utils.Ptr([]string{
						"ns2",
						"ns3",
					}),
				},
			},
			testRegion,
			Model{
				Id:              types.StringValue("pid,region,nid"),
				ProjectId:       types.StringValue("pid"),
				NetworkId:       types.StringValue("nid"),
				Name:            types.StringNull(),
				IPv6Prefixes:    types.ListNull(types.StringType),
				IPv4Nameservers: types.ListNull(types.StringType),
				IPv4Prefixes:    types.ListNull(types.StringType),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns2"),
					types.StringValue("ns3"),
				}),
				Labels: types.MapNull(types.StringType),
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"ipv4_prefixes_changed_outside_tf",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
			},
			&iaas.Network{
				Id: utils.Ptr("nid"),
				Ipv4: &iaas.NetworkIPv4{
					Prefixes: utils.Ptr(
						[]string{
							"192.168.54.0/24",
							"192.168.55.0/24",
						},
					),
				},
			},
			testRegion,
			Model{
				Id:               types.StringValue("pid,region,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				IPv6Nameservers:  types.ListNull(types.StringType),
				IPv6PrefixLength: types.Int64Null(),
				IPv6Prefixes:     types.ListNull(types.StringType),
				Labels:           types.MapNull(types.StringType),
				IPv4Nameservers:  types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Value(24),
				IPv4Prefix:       types.StringValue("192.168.54.0/24"),
				IPv4Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("192.168.54.0/24"),
					types.StringValue("192.168.55.0/24"),
				}),
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"ipv6_prefixes_changed_outside_tf",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				IPv6Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("fd12:3456:789a:1::/64"),
					types.StringValue("fd12:3456:789a:2::/64"),
				}),
			},
			&iaas.Network{
				Id: utils.Ptr("nid"),
				Ipv6: &iaas.NetworkIPv6{
					Prefixes: utils.Ptr(
						[]string{
							"fd12:3456:789a:1::/64",
							"fd12:3456:789a:2::/64",
						},
					),
				},
			},
			testRegion,
			Model{
				Id:               types.StringValue("pid,region,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				IPv4Nameservers:  types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Null(),
				IPv4Prefixes:     types.ListNull(types.StringType),
				Labels:           types.MapNull(types.StringType),
				IPv6Nameservers:  types.ListNull(types.StringType),
				IPv6PrefixLength: types.Int64Value(64),
				IPv6Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("fd12:3456:789a:1::/64"),
					types.StringValue("fd12:3456:789a:2::/64"),
				}),
				IPv6Prefix: types.StringValue("fd12:3456:789a:1::/64"),
				Region:     types.StringValue(testRegion),
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
				Id: utils.Ptr("nid"),
			},
			testRegion,
			Model{
				Id:               types.StringValue("pid,region,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				IPv4Nameservers:  types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Null(),
				IPv4Gateway:      types.StringNull(),
				IPv4Prefixes:     types.ListNull(types.StringType),
				IPv6Nameservers:  types.ListNull(types.StringType),
				IPv6PrefixLength: types.Int64Null(),
				IPv6Gateway:      types.StringNull(),
				IPv6Prefixes:     types.ListNull(types.StringType),
				PublicIP:         types.StringNull(),
				Labels:           types.MapNull(types.StringType),
				Routed:           types.BoolNull(),
				Region:           types.StringValue(testRegion),
			},
			true,
		},
		{
			"response_nil_fail",
			Model{},
			nil,
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				ProjectId: types.StringValue("pid"),
			},
			&iaas.Network{},
			testRegion,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, tt.region)
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
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv4Gateway: types.StringValue("gateway"),
				IPv4Prefix:  types.StringValue("prefix"),
				DHCP:        types.BoolValue(true),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaas.CreateNetworkIPv4{
					CreateNetworkIPv4WithPrefix: &iaas.CreateNetworkIPv4WithPrefix{
						Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
						Nameservers: utils.Ptr([]string{
							"ns1",
							"ns2",
						}),
						Prefix: utils.Ptr("prefix"),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Routed: utils.Ptr(false),
				Dhcp:   utils.Ptr(true),
			},
			true,
		},
		{
			"ipv4_nameservers_okay",
			&Model{
				Name: types.StringValue("name"),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv4Gateway: types.StringValue("gateway"),
				IPv4Prefix:  types.StringValue("prefix"),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaas.CreateNetworkIPv4{
					CreateNetworkIPv4WithPrefix: &iaas.CreateNetworkIPv4WithPrefix{
						Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
						Nameservers: utils.Ptr([]string{
							"ns1",
							"ns2",
						}),
						Prefix: utils.Ptr("prefix"),
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
			"ipv4_nameservers_null",
			&Model{
				Name:            types.StringValue("name"),
				IPv4Nameservers: types.ListNull(types.StringType),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv4Gateway: types.StringValue("gateway"),
				IPv4Prefix:  types.StringValue("prefix"),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaas.CreateNetworkIPv4{
					CreateNetworkIPv4WithPrefix: &iaas.CreateNetworkIPv4WithPrefix{
						Gateway:     iaas.NewNullableString(utils.Ptr("gateway")),
						Nameservers: nil,
						Prefix:      utils.Ptr("prefix"),
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
			"ipv4_nameservers_empty_slice",
			&Model{
				Name:            types.StringValue("name"),
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv4Gateway: types.StringValue("gateway"),
				IPv4Prefix:  types.StringValue("prefix"),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaas.CreateNetworkIPv4{
					CreateNetworkIPv4WithPrefix: &iaas.CreateNetworkIPv4WithPrefix{
						Gateway:     iaas.NewNullableString(utils.Ptr("gateway")),
						Nameservers: utils.Ptr([]string{}),
						Prefix:      utils.Ptr("prefix"),
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
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv6Gateway: types.StringValue("gateway"),
				IPv6Prefix:  types.StringValue("prefix"),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv6: &iaas.CreateNetworkIPv6{
					CreateNetworkIPv6WithPrefix: &iaas.CreateNetworkIPv6WithPrefix{
						Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
						Nameservers: utils.Ptr([]string{
							"ns1",
							"ns2",
						}),
						Prefix: utils.Ptr("prefix"),
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
			"ipv6_nameserver_null",
			&Model{
				Name:            types.StringValue("name"),
				IPv6Nameservers: types.ListNull(types.StringType),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv6Gateway: types.StringValue("gateway"),
				IPv6Prefix:  types.StringValue("prefix"),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv6: &iaas.CreateNetworkIPv6{
					CreateNetworkIPv6WithPrefix: &iaas.CreateNetworkIPv6WithPrefix{
						Nameservers: nil,
						Gateway:     iaas.NewNullableString(utils.Ptr("gateway")),
						Prefix:      utils.Ptr("prefix"),
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
			"ipv6_nameserver_empty_list",
			&Model{
				Name:            types.StringValue("name"),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Routed:      types.BoolValue(false),
				IPv6Gateway: types.StringValue("gateway"),
				IPv6Prefix:  types.StringValue("prefix"),
			},
			&iaas.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv6: &iaas.CreateNetworkIPv6{
					CreateNetworkIPv6WithPrefix: &iaas.CreateNetworkIPv6WithPrefix{
						Nameservers: utils.Ptr([]string{}),
						Gateway:     iaas.NewNullableString(utils.Ptr("gateway")),
						Prefix:      utils.Ptr("prefix"),
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
				DHCP:        types.BoolValue(true),
			},
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaas.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaas.UpdateNetworkIPv4Body{
					Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
					Nameservers: utils.Ptr([]string{
						"ns1",
						"ns2",
					}),
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Dhcp: utils.Ptr(true),
			},
			true,
		},
		{
			"ipv4_nameservers_okay",
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
				Ipv4: &iaas.UpdateNetworkIPv4Body{
					Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
					Nameservers: utils.Ptr([]string{
						"ns1",
						"ns2",
					}),
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
				Ipv4: &iaas.UpdateNetworkIPv4Body{
					Nameservers: utils.Ptr([]string{
						"ns1",
						"ns2",
					}),
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
		{
			"ipv4_nameservers_null",
			&Model{
				Name:            types.StringValue("name"),
				IPv4Nameservers: types.ListNull(types.StringType),
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
				Ipv4: nil,
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
		{
			"ipv4_nameservers_null_and_gateway_set",
			&Model{
				Name:            types.StringValue("name"),
				IPv4Nameservers: types.ListNull(types.StringType),
				IPv4Gateway:     types.StringValue("gateway"),
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
				Ipv4: &iaas.UpdateNetworkIPv4Body{
					Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
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
				Ipv6: &iaas.UpdateNetworkIPv6Body{
					Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
					Nameservers: utils.Ptr([]string{
						"ns1",
						"ns2",
					}),
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
				Ipv6: &iaas.UpdateNetworkIPv6Body{
					Nameservers: utils.Ptr([]string{
						"ns1",
						"ns2",
					}),
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
		{
			"ipv6_nameserver_null",
			&Model{
				Name:            types.StringValue("name"),
				IPv6Nameservers: types.ListNull(types.StringType),
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
				Ipv6: &iaas.UpdateNetworkIPv6Body{
					Nameservers: nil,
					Gateway:     iaas.NewNullableString(utils.Ptr("gateway")),
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
		{
			"ipv6_nameserver_empty_list",
			&Model{
				Name:            types.StringValue("name"),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{}),
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
				Ipv6: &iaas.UpdateNetworkIPv6Body{
					Nameservers: utils.Ptr([]string{}),
					Gateway:     iaas.NewNullableString(utils.Ptr("gateway")),
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

func TestModelIsIPv4ConfigSet(t *testing.T) {
	tests := []struct {
		name  string
		model Model
		want  bool
	}{
		{
			name: "no ipv4 field is set",
			model: Model{
				IPv4Nameservers: types.List{},
				IPv4Gateway:     types.String{},
				NoIPv4Gateway:   types.Bool{},
			},
			want: false,
		},
		{
			name: "ipv4 fields are set to null",
			model: Model{
				IPv4Nameservers: types.ListNull(types.StringType),
				IPv4Gateway:     types.StringNull(),
				NoIPv4Gateway:   types.BoolNull(),
			},
			want: false,
		},
		{
			name: "no ipv4 gateway is true",
			model: Model{
				IPv4Nameservers: types.ListNull(types.StringType),
				IPv4Gateway:     types.StringNull(),
				NoIPv4Gateway:   types.BoolValue(true),
			},
			want: true,
		},
		{
			name: "ipv4 nameserver is set",
			model: Model{
				IPv4Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv4Gateway:   types.StringNull(),
				NoIPv4Gateway: types.BoolNull(),
			},
			want: true,
		},
		{
			name: "ipv4 gateway is set",
			model: Model{
				IPv4Nameservers: types.ListNull(types.StringType),
				IPv4Gateway:     types.StringValue("gateway"),
				NoIPv4Gateway:   types.BoolNull(),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.isIPv4UpdateConfigSet(); got != tt.want {
				t.Errorf("isIPv4ConfigSet() = %v, want %v", got, tt.want)
			}
		})
	}
}
