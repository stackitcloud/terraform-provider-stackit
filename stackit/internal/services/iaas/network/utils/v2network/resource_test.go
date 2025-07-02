package v2network

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/model"
)

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		state       model.Model
		input       *iaasalpha.Network
		region      string
		expected    model.Model
		isValid     bool
	}{
		{
			"id_ok",
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
			},
			&iaasalpha.Network{
				Id: utils.Ptr("nid"),
				Ipv4: &iaasalpha.NetworkIPv4{
					Gateway: iaasalpha.NewNullableString(nil),
				},
			},
			testRegion,
			model.Model{
				Id:               types.StringValue("pid,region,nid"),
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
				Region:           types.StringValue(testRegion),
			},
			true,
		},
		{
			"values_ok",
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
			},
			&iaasalpha.Network{
				Id:   utils.Ptr("nid"),
				Name: utils.Ptr("name"),
				Ipv4: &iaasalpha.NetworkIPv4{
					Nameservers: utils.Ptr([]string{"ns1", "ns2"}),
					Prefixes: utils.Ptr(
						[]string{
							"192.168.42.0/24",
							"10.100.10.0/16",
						},
					),
					PublicIp: utils.Ptr("publicIp"),
					Gateway:  iaasalpha.NewNullableString(utils.Ptr("gateway")),
				},
				Ipv6: &iaasalpha.NetworkIPv6{
					Nameservers: utils.Ptr([]string{"ns1", "ns2"}),
					Prefixes: utils.Ptr([]string{
						"fd12:3456:789a:1::/64",
						"fd12:3456:789b:1::/64",
					}),
					Gateway: iaasalpha.NewNullableString(utils.Ptr("gateway")),
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Routed: utils.Ptr(true),
			},
			testRegion,
			model.Model{
				Id:        types.StringValue("pid,region,nid"),
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
				IPv4PrefixLength: types.Int64Value(24),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("192.168.42.0/24"),
					types.StringValue("10.100.10.0/16"),
				}),
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
			},
			true,
		},
		{
			"ipv4_nameservers_changed_outside_tf",
			model.Model{
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
			&iaasalpha.Network{
				Id: utils.Ptr("nid"),
				Ipv4: &iaasalpha.NetworkIPv4{
					Nameservers: utils.Ptr([]string{
						"ns2",
						"ns3",
					}),
				},
			},
			testRegion,
			model.Model{
				Id:              types.StringValue("pid,region,nid"),
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
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"ipv6_nameservers_changed_outside_tf",
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
			},
			&iaasalpha.Network{
				Id: utils.Ptr("nid"),
				Ipv6: &iaasalpha.NetworkIPv6{
					Nameservers: utils.Ptr([]string{
						"ns2",
						"ns3",
					}),
				},
			},
			testRegion,
			model.Model{
				Id:              types.StringValue("pid,region,nid"),
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
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"ipv4_prefixes_changed_outside_tf",
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("192.168.42.0/24"),
					types.StringValue("10.100.10.0/24"),
				}),
			},
			&iaasalpha.Network{
				Id: utils.Ptr("nid"),
				Ipv4: &iaasalpha.NetworkIPv4{
					Prefixes: utils.Ptr(
						[]string{
							"192.168.54.0/24",
							"192.168.55.0/24",
						},
					),
				},
			},
			testRegion,
			model.Model{
				Id:               types.StringValue("pid,region,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				IPv6Nameservers:  types.ListNull(types.StringType),
				IPv6PrefixLength: types.Int64Null(),
				IPv6Prefixes:     types.ListNull(types.StringType),
				Labels:           types.MapNull(types.StringType),
				Nameservers:      types.ListNull(types.StringType),
				IPv4Nameservers:  types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Value(24),
				IPv4Prefix:       types.StringValue("192.168.54.0/24"),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("192.168.54.0/24"),
					types.StringValue("192.168.55.0/24"),
				}),
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
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				IPv6Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("fd12:3456:789a:1::/64"),
					types.StringValue("fd12:3456:789a:2::/64"),
				}),
			},
			&iaasalpha.Network{
				Id: utils.Ptr("nid"),
				Ipv6: &iaasalpha.NetworkIPv6{
					Prefixes: utils.Ptr(
						[]string{
							"fd12:3456:789a:1::/64",
							"fd12:3456:789a:2::/64",
						},
					),
				},
			},
			testRegion,
			model.Model{
				Id:               types.StringValue("pid,region,nid"),
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
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
			},
			&iaasalpha.Network{
				Id: utils.Ptr("nid"),
			},
			testRegion,
			model.Model{
				Id:               types.StringValue("pid,region,nid"),
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
				Region:           types.StringValue(testRegion),
			},
			true,
		},
		{
			"response_nil_fail",
			model.Model{},
			nil,
			testRegion,
			model.Model{},
			false,
		},
		{
			"no_resource_id",
			model.Model{
				ProjectId: types.StringValue("pid"),
			},
			&iaasalpha.Network{},
			testRegion,
			model.Model{},
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
		input       *model.Model
		expected    *iaasalpha.CreateNetworkPayload
		isValid     bool
	}{
		{
			"default_ok",
			&model.Model{
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
			&iaasalpha.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaasalpha.CreateNetworkIPv4{
					CreateNetworkIPv4WithPrefix: &iaasalpha.CreateNetworkIPv4WithPrefix{
						Gateway: iaasalpha.NewNullableString(utils.Ptr("gateway")),
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
			"ipv4_nameservers_okay",
			&model.Model{
				Name: types.StringValue("name"),
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
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
			&iaasalpha.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaasalpha.CreateNetworkIPv4{
					CreateNetworkIPv4WithPrefix: &iaasalpha.CreateNetworkIPv4WithPrefix{
						Gateway: iaasalpha.NewNullableString(utils.Ptr("gateway")),
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
			"ipv6_default_ok",
			&model.Model{
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
			&iaasalpha.CreateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv6: &iaasalpha.CreateNetworkIPv6{
					CreateNetworkIPv6WithPrefix: &iaasalpha.CreateNetworkIPv6WithPrefix{
						Gateway: iaasalpha.NewNullableString(utils.Ptr("gateway")),
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaasalpha.NullableString{}))
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
		input       *model.Model
		state       model.Model
		expected    *iaasalpha.PartialUpdateNetworkPayload
		isValid     bool
	}{
		{
			"default_ok",
			&model.Model{
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
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaasalpha.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaasalpha.UpdateNetworkIPv4Body{
					Gateway: iaasalpha.NewNullableString(utils.Ptr("gateway")),
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
			"ipv4_nameservers_okay",
			&model.Model{
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
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaasalpha.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaasalpha.UpdateNetworkIPv4Body{
					Gateway: iaasalpha.NewNullableString(utils.Ptr("gateway")),
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
			&model.Model{
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
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaasalpha.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv4: &iaasalpha.UpdateNetworkIPv4Body{
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
			"ipv6_default_ok",
			&model.Model{
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
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaasalpha.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv6: &iaasalpha.UpdateNetworkIPv6Body{
					Gateway: iaasalpha.NewNullableString(utils.Ptr("gateway")),
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
			&model.Model{
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
			model.Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Labels:    types.MapNull(types.StringType),
			},
			&iaasalpha.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				Ipv6: &iaasalpha.UpdateNetworkIPv6Body{
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaasalpha.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
