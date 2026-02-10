package publicip

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
		input  *iaas.PublicIp
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    Model
		isValid     bool
	}{
		{
			description: "default_values",
			args: args{
				state: Model{
					ProjectId:  types.StringValue("pid"),
					PublicIpId: types.StringValue("pipid"),
				},
				input: &iaas.PublicIp{
					Id:               utils.Ptr("pipid"),
					NetworkInterface: iaas.NewNullableString(nil),
				},
				region: "eu01",
			},
			expected: Model{
				Id:                 types.StringValue("pid,eu01,pipid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringNull(),
				Labels:             types.MapNull(types.StringType),
				NetworkInterfaceId: types.StringNull(),
				Region:             types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			args: args{
				state: Model{
					ProjectId:  types.StringValue("pid"),
					PublicIpId: types.StringValue("pipid"),
					Region:     types.StringValue("eu01"),
				},
				input: &iaas.PublicIp{
					Id: utils.Ptr("pipid"),
					Ip: utils.Ptr("ip"),
					Labels: &map[string]interface{}{
						"key": "value",
					},
					NetworkInterface: iaas.NewNullableString(utils.Ptr("interface")),
				},
				region: "eu02",
			},
			expected: Model{
				Id:         types.StringValue("pid,eu02,pipid"),
				ProjectId:  types.StringValue("pid"),
				PublicIpId: types.StringValue("pipid"),
				Ip:         types.StringValue("ip"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				NetworkInterfaceId: types.StringValue("interface"),
				Region:             types.StringValue("eu02"),
			},
			isValid: true,
		},
		{
			description: "empty_labels",
			args: args{
				state: Model{
					ProjectId:  types.StringValue("pid"),
					PublicIpId: types.StringValue("pipid"),
					Labels:     types.MapValueMust(types.StringType, map[string]attr.Value{}),
				},
				input: &iaas.PublicIp{
					Id:               utils.Ptr("pipid"),
					NetworkInterface: iaas.NewNullableString(utils.Ptr("interface")),
				},
				region: "eu01",
			},
			expected: Model{
				Id:                 types.StringValue("pid,eu01,pipid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringNull(),
				Labels:             types.MapValueMust(types.StringType, map[string]attr.Value{}),
				NetworkInterfaceId: types.StringValue("interface"),
				Region:             types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "network_interface_id_nil",
			args: args{
				state: Model{
					ProjectId:  types.StringValue("pid"),
					PublicIpId: types.StringValue("pipid"),
				},
				input: &iaas.PublicIp{
					Id: utils.Ptr("pipid"),
				},
				region: "eu01",
			},
			expected: Model{
				Id:                 types.StringValue("pid,eu01,pipid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringNull(),
				Labels:             types.MapNull(types.StringType),
				NetworkInterfaceId: types.StringNull(),
				Region:             types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "response_nil_fail",
		},
		{
			description: "no_resource_id",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
				},
				input: &iaas.PublicIp{},
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
		expected    *iaas.CreatePublicIPPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Ip: types.StringValue("ip"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				NetworkInterfaceId: types.StringValue("interface"),
			},
			&iaas.CreatePublicIPPayload{
				Ip: utils.Ptr("ip"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				NetworkInterface: iaas.NewNullableString(utils.Ptr("interface")),
			},
			true,
		},
		{
			"network_interface_nil",
			&Model{
				Ip: types.StringValue("ip"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaas.CreatePublicIPPayload{
				Ip: utils.Ptr("ip"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				NetworkInterface: iaas.NewNullableString(nil),
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
		expected    *iaas.UpdatePublicIPPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Ip: types.StringValue("ip"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				NetworkInterfaceId: types.StringValue("interface"),
			},
			&iaas.UpdatePublicIPPayload{
				Labels: &map[string]interface{}{
					"key": "value",
				},
				NetworkInterface: iaas.NewNullableString(utils.Ptr("interface")),
			},
			true,
		},
		{
			"network_interface_nil",
			&Model{
				Ip: types.StringValue("ip"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaas.UpdatePublicIPPayload{
				Labels: &map[string]interface{}{
					"key": "value",
				},
				NetworkInterface: iaas.NewNullableString(nil),
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
