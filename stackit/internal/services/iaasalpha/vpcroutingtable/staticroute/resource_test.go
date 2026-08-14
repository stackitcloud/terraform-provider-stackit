package staticroute

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"
)

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s,%s,%s,%s", "pid", "vid", testRegion, "rtid", "rid")
	tests := []struct {
		description string
		state       SharedModel
		input       *iaas.Route
		expected    SharedModel
		wantErr     bool
	}{
		{
			description: "default_values",
			state: SharedModel{
				ProjectId:      types.StringValue("pid"),
				VpcId:          types.StringValue("vid"),
				RoutingTableId: types.StringValue("rtid"),
			},
			input: &iaas.Route{
				Id: new("rid"),
				Destination: iaas.DestinationCIDRv4AsRouteDestination(&iaas.DestinationCIDRv4{
					Type:  "cidrv4",
					Value: "10.0.0.0/24",
				}),
				Nexthop: iaas.NexthopIPv4AsRouteNexthop(&iaas.NexthopIPv4{
					Type:  "ipv4",
					Value: "192.168.1.1",
				}),
			},
			expected: SharedModel{
				Id:             types.StringValue(id),
				ProjectId:      types.StringValue("pid"),
				VpcId:          types.StringValue("vid"),
				RoutingTableId: types.StringValue("rtid"),
				RouteId:        types.StringValue("rid"),
				Region:         types.StringValue(testRegion),
				Labels:         types.MapNull(types.StringType),
				Destination: types.ObjectValueMust(destinationTypes, map[string]attr.Value{
					"type":  types.StringValue("cidrv4"),
					"value": types.StringValue("10.0.0.0/24"),
				}),
				Nexthop: types.ObjectValueMust(nexthopTypes, map[string]attr.Value{
					"type":  types.StringValue("ipv4"),
					"value": types.StringValue("192.168.1.1"),
				}),
			},
		},
		{
			description: "blackhole_nexthop",
			state: SharedModel{
				ProjectId:      types.StringValue("pid"),
				VpcId:          types.StringValue("vid"),
				RoutingTableId: types.StringValue("rtid"),
			},
			input: &iaas.Route{
				Id: new("rid"),
				Destination: iaas.DestinationCIDRv6AsRouteDestination(&iaas.DestinationCIDRv6{
					Type:  "cidrv6",
					Value: "2001:db8::/32",
				}),
				Nexthop: iaas.NexthopBlackholeAsRouteNexthop(&iaas.NexthopBlackhole{
					Type: "blackhole",
				}),
			},
			expected: SharedModel{
				Id:             types.StringValue(id),
				ProjectId:      types.StringValue("pid"),
				VpcId:          types.StringValue("vid"),
				RoutingTableId: types.StringValue("rtid"),
				RouteId:        types.StringValue("rid"),
				Region:         types.StringValue(testRegion),
				Labels:         types.MapNull(types.StringType),
				Destination: types.ObjectValueMust(destinationTypes, map[string]attr.Value{
					"type":  types.StringValue("cidrv6"),
					"value": types.StringValue("2001:db8::/32"),
				}),
				Nexthop: types.ObjectValueMust(nexthopTypes, map[string]attr.Value{
					"type":  types.StringValue("blackhole"),
					"value": types.StringNull(),
				}),
			},
		},
		{
			description: "response_fields_nil_fail",
			state:       SharedModel{},
			input: &iaas.Route{
				Id: nil,
			},
			expected: SharedModel{},
			wantErr:  true,
		},
		{
			description: "response_nil_fail",
			state:       SharedModel{},
			input:       nil,
			expected:    SharedModel{},
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, testRegion)
			if tt.wantErr && err == nil {
				t.Fatalf("Should have failed")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if !tt.wantErr {
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
		input       *SharedModel
		expected    *iaas.AddVPCStaticRoutePayload
		wantErr     bool
	}{
		{
			description: "default_ok",
			input: &SharedModel{
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Destination: types.ObjectValueMust(destinationTypes, map[string]attr.Value{
					"type":  types.StringValue("cidrv4"),
					"value": types.StringValue("10.0.0.0/24"),
				}),
				Nexthop: types.ObjectValueMust(nexthopTypes, map[string]attr.Value{
					"type":  types.StringValue("ipv4"),
					"value": types.StringValue("192.168.1.1"),
				}),
			},
			expected: &iaas.AddVPCStaticRoutePayload{
				Labels: map[string]interface{}{
					"key": "value",
				},
				Destination: iaas.AddVPCStaticRoutePayloadDestination{
					DestinationCIDRv4: &iaas.DestinationCIDRv4{
						Type:  "cidrv4",
						Value: "10.0.0.0/24",
					},
				},
				Nexthop: iaas.AddVPCStaticRoutePayloadNexthop{
					NexthopIPv4: &iaas.NexthopIPv4{
						Type:  "ipv4",
						Value: "192.168.1.1",
					},
				},
			},
		},
		{
			description: "blackhole",
			input: &SharedModel{
				Labels: types.MapNull(types.StringType),
				Destination: types.ObjectValueMust(destinationTypes, map[string]attr.Value{
					"type":  types.StringValue("cidrv6"),
					"value": types.StringValue("2001:db8::/32"),
				}),
				Nexthop: types.ObjectValueMust(nexthopTypes, map[string]attr.Value{
					"type":  types.StringValue("blackhole"),
					"value": types.StringNull(),
				}),
			},
			expected: &iaas.AddVPCStaticRoutePayload{
				Labels: map[string]interface{}{},
				Destination: iaas.AddVPCStaticRoutePayloadDestination{
					DestinationCIDRv6: &iaas.DestinationCIDRv6{
						Type:  "cidrv6",
						Value: "2001:db8::/32",
					},
				},
				Nexthop: iaas.AddVPCStaticRoutePayloadNexthop{
					NexthopBlackhole: &iaas.NexthopBlackhole{
						Type: "blackhole",
					},
				},
			},
		},
		{
			description: "internet",
			input: &SharedModel{
				Labels: types.MapNull(types.StringType),
				Destination: types.ObjectValueMust(destinationTypes, map[string]attr.Value{
					"type":  types.StringValue("cidrv4"),
					"value": types.StringValue("10.0.0.0/24"),
				}),
				Nexthop: types.ObjectValueMust(nexthopTypes, map[string]attr.Value{
					"type":  types.StringValue("internet"),
					"value": types.StringNull(),
				}),
			},
			expected: &iaas.AddVPCStaticRoutePayload{
				Labels: map[string]interface{}{},
				Destination: iaas.AddVPCStaticRoutePayloadDestination{
					DestinationCIDRv4: &iaas.DestinationCIDRv4{
						Type:  "cidrv4",
						Value: "10.0.0.0/24",
					},
				},
				Nexthop: iaas.AddVPCStaticRoutePayloadNexthop{
					NexthopInternet: &iaas.NexthopInternet{
						Type: "internet",
					},
				},
			},
		},
		{
			description: "ipv6",
			input: &SharedModel{
				Labels: types.MapNull(types.StringType),
				Destination: types.ObjectValueMust(destinationTypes, map[string]attr.Value{
					"type":  types.StringValue("cidrv6"),
					"value": types.StringValue("2001:db8::/32"),
				}),
				Nexthop: types.ObjectValueMust(nexthopTypes, map[string]attr.Value{
					"type":  types.StringValue("ipv6"),
					"value": types.StringValue("2001:db8::1"),
				}),
			},
			expected: &iaas.AddVPCStaticRoutePayload{
				Labels: map[string]interface{}{},
				Destination: iaas.AddVPCStaticRoutePayloadDestination{
					DestinationCIDRv6: &iaas.DestinationCIDRv6{
						Type:  "cidrv6",
						Value: "2001:db8::/32",
					},
				},
				Nexthop: iaas.AddVPCStaticRoutePayloadNexthop{
					NexthopIPv6: &iaas.NexthopIPv6{
						Type:  "ipv6",
						Value: "2001:db8::1",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("Should have failed")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if !tt.wantErr {
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
		description   string
		input         *SharedModel
		expected      *iaas.UpdateVPCStaticRoutePayload
		currentLabels *types.Map
		wantErr       bool
	}{
		{
			description: "default_ok",
			input: &SharedModel{
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				}),
			},
			expected: &iaas.UpdateVPCStaticRoutePayload{
				Labels: map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
		},
		{
			description: "no_labels",
			input: &SharedModel{
				Labels: types.MapNull(types.StringType),
			},
			expected: &iaas.UpdateVPCStaticRoutePayload{
				Labels: map[string]interface{}{},
			},
		},
		{
			description: "existing labels",
			input: &SharedModel{
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				}),
			},
			currentLabels: new(types.MapValueMust(types.StringType, map[string]attr.Value{
				"key1": types.StringValue("value0"),
				"key3": types.StringValue("value3"),
			})),
			expected: &iaas.UpdateVPCStaticRoutePayload{
				Labels: map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
					"key3": nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var current types.Map
			if tt.currentLabels != nil {
				current = *tt.currentLabels
			} else {
				current = types.MapNull(types.StringType)
			}
			output, err := toUpdatePayload(context.Background(), tt.input, current)
			if tt.wantErr && err == nil {
				t.Fatalf("Should have failed")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if !tt.wantErr {
				diff := cmp.Diff(&output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
