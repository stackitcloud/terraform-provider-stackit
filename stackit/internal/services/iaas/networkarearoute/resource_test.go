package networkarearoute

import (
	"context"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapFields(t *testing.T) {
	type args struct {
		state  Model
		input  *iaas.Route
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
					OrganizationId:     types.StringValue("oid"),
					NetworkAreaId:      types.StringValue("naid"),
					NetworkAreaRouteId: types.StringValue("narid"),
				},
				input:  &iaas.Route{},
				region: "eu01",
			},
			expected: Model{
				Id:                 types.StringValue("oid,naid,eu01,narid"),
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				NetworkAreaRouteId: types.StringValue("narid"),
				Prefix:             types.StringNull(),
				NextHop:            types.StringNull(),
				Labels:             types.MapNull(types.StringType),
				Region:             types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "values_ok",
			args: args{
				state: Model{
					OrganizationId:     types.StringValue("oid"),
					NetworkAreaId:      types.StringValue("naid"),
					NetworkAreaRouteId: types.StringValue("narid"),
					Region:             types.StringValue("eu01"),
				},
				input: &iaas.Route{
					Destination: &iaas.RouteDestination{
						DestinationCIDRv4: &iaas.DestinationCIDRv4{
							Type:  utils.Ptr("cidrv4"),
							Value: utils.Ptr("prefix"),
						},
						DestinationCIDRv6: nil,
					},
					Nexthop: &iaas.RouteNexthop{
						NexthopIPv4: &iaas.NexthopIPv4{
							Type:  utils.Ptr("ipv4"),
							Value: utils.Ptr("hop"),
						},
					},
					Labels: &map[string]interface{}{
						"key": "value",
					},
				},
				region: "eu02",
			},
			expected: Model{
				Id:                 types.StringValue("oid,naid,eu02,narid"),
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				NetworkAreaRouteId: types.StringValue("narid"),
				Prefix:             types.StringValue("prefix"),
				NextHop:            types.StringValue("hop"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Region: types.StringValue("eu02"),
			},
			isValid: true,
		},
		{
			description: "response_fields_nil_fail",
			args: args{
				input: &iaas.Route{
					Destination: nil,
					Nexthop:     nil,
				},
			},
		},
		{
			description: "response_nil_fail",
		},
		{
			description: "no_resource_id",
			args: args{
				state: Model{
					OrganizationId: types.StringValue("oid"),
					NetworkAreaId:  types.StringValue("naid"),
				},
				input: &iaas.Route{},
			},
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
		expected    *iaas.CreateNetworkAreaRoutePayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &Model{
				Prefix:  types.StringValue("prefix"),
				NextHop: types.StringValue("hop"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			expected: &iaas.CreateNetworkAreaRoutePayload{
				Items: &[]iaas.Route{
					{
						Destination: &iaas.RouteDestination{
							DestinationCIDRv4: &iaas.DestinationCIDRv4{
								Type:  utils.Ptr("cidrv4"),
								Value: utils.Ptr("prefix"),
							},
							DestinationCIDRv6: nil,
						},
						Nexthop: &iaas.RouteNexthop{
							NexthopIPv4: &iaas.NexthopIPv4{
								Type:  utils.Ptr("ipv4"),
								Value: utils.Ptr("hop"),
							},
						},
						Labels: &map[string]interface{}{
							"key": "value",
						},
					},
				},
			},
			isValid: true,
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
		expected    *iaas.UpdateNetworkAreaRoutePayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				}),
			},
			&iaas.UpdateNetworkAreaRoutePayload{
				Labels: &map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
