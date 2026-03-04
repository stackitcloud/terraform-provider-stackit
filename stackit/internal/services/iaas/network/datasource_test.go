package network

import (
	"context"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/services/iaas"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

const (
	testRegion = "region"
)

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		state       DataSourceModel
		input       *iaas.Network
		region      string
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"id_ok",
			DataSourceModel{
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
			DataSourceModel{
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
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
			},
			&iaas.Network{
				Id:   utils.Ptr("nid"),
				Name: utils.Ptr("name"),
				Ipv4: &iaas.NetworkIPv4{
					Nameservers: &[]string{
						"ns1",
						"ns2",
					},
					Prefixes: &[]string{
						"192.168.42.0/24",
						"10.100.10.0/16",
					},
					PublicIp: utils.Ptr("publicIp"),
					Gateway:  iaas.NewNullableString(utils.Ptr("gateway")),
				},
				Ipv6: &iaas.NetworkIPv6{
					Nameservers: &[]string{
						"ns1",
						"ns2",
					},
					Prefixes: &[]string{
						"fd12:3456:789a:1::/64",
						"fd12:3456:789a:2::/64",
					},
					Gateway: iaas.NewNullableString(utils.Ptr("gateway")),
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Routed: utils.Ptr(true),
				Dhcp:   utils.Ptr(true),
			},
			testRegion,
			DataSourceModel{
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
				IPv4Prefix: types.StringValue("192.168.42.0/24"),
				IPv4Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("192.168.42.0/24"),
					types.StringValue("10.100.10.0/16"),
				}),
				IPv6Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv6PrefixLength: types.Int64Value(64),
				IPv6Prefix:       types.StringValue("fd12:3456:789a:1::/64"),
				IPv6Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("fd12:3456:789a:1::/64"),
					types.StringValue("fd12:3456:789a:2::/64"),
				}),
				PublicIP: types.StringValue("publicIp"),
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
			DataSourceModel{
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
				Id: utils.Ptr("nid"),
				Ipv4: &iaas.NetworkIPv4{
					Nameservers: &[]string{
						"ns2",
						"ns3",
					},
				},
			},
			testRegion,
			DataSourceModel{
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
			DataSourceModel{
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
					Nameservers: &[]string{
						"ns2",
						"ns3",
					},
				},
			},
			testRegion,
			DataSourceModel{
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
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("192.168.42.0/24"),
					types.StringValue("10.100.10.0/16"),
				}),
			},
			&iaas.Network{
				Id: utils.Ptr("nid"),
				Ipv4: &iaas.NetworkIPv4{
					Prefixes: &[]string{
						"10.100.20.0/16",
						"10.100.10.0/16",
					},
				},
			},
			testRegion,
			DataSourceModel{
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
				IPv4PrefixLength: types.Int64Value(16),
				IPv4Prefix:       types.StringValue("10.100.20.0/16"),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("10.100.20.0/16"),
					types.StringValue("10.100.10.0/16"),
				}),
				IPv4Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("10.100.20.0/16"),
					types.StringValue("10.100.10.0/16"),
				}),
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"ipv6_prefixes_changed_outside_tf",
			DataSourceModel{
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
					Prefixes: &[]string{
						"fd12:3456:789a:3::/64",
						"fd12:3456:789a:4::/64",
					},
				},
			},
			testRegion,
			DataSourceModel{
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
				IPv6Prefix:       types.StringValue("fd12:3456:789a:3::/64"),
				IPv6Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("fd12:3456:789a:3::/64"),
					types.StringValue("fd12:3456:789a:4::/64"),
				}),
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"ipv4_ipv6_gateway_nil",
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
			},
			&iaas.Network{
				Id: utils.Ptr("nid"),
			},
			testRegion,
			DataSourceModel{
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
			DataSourceModel{},
			nil,
			testRegion,
			DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
			},
			&iaas.Network{},
			testRegion,
			DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceFields(context.Background(), tt.input, &tt.state, tt.region)
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
