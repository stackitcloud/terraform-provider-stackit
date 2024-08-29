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
			},
			Model{
				Id:               types.StringValue("pid,nid"),
				ProjectId:        types.StringValue("pid"),
				NetworkId:        types.StringValue("nid"),
				Name:             types.StringNull(),
				Nameservers:      types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Null(),
				Prefixes:         types.ListNull(types.StringType),
				PublicIP:         types.StringNull(),
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
				PublicIp: utils.Ptr("publicIp"),
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
				IPv4PrefixLength: types.Int64Null(),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix1"),
					types.StringValue("prefix2"),
				}),
				PublicIP: types.StringValue("publicIp"),
			},
			true,
		},
		{
			"nameservers_changed_outside_tf",
			Model{
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
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
				Id:        types.StringValue("pid,nid"),
				ProjectId: types.StringValue("pid"),
				NetworkId: types.StringValue("nid"),
				Name:      types.StringNull(),
				Prefixes:  types.ListNull(types.StringType),
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns2"),
					types.StringValue("ns3"),
				}),
			},
			true,
		},
		{
			"prefixes_changed_outisde_tf",
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
				Nameservers:      types.ListNull(types.StringType),
				IPv4PrefixLength: types.Int64Null(),
				Prefixes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("prefix2"),
					types.StringValue("prefix3"),
				}),
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
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				IPv4PrefixLength: types.Int64Value(24),
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
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
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
		expected    *iaas.PartialUpdateNetworkPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				Nameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
			},
			&iaas.PartialUpdateNetworkPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.UpdateNetworkAddressFamily{
					Ipv4: &iaas.UpdateNetworkIPv4Body{
						Nameservers: &[]string{
							"ns1",
							"ns2",
						},
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input)
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
