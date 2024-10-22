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
	tests := []struct {
		description string
		state       Model
		input       *iaas.PublicIp
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId:  types.StringValue("pid"),
				PublicIpId: types.StringValue("pipid"),
			},
			&iaas.PublicIp{
				Id:               utils.Ptr("pipid"),
				NetworkInterface: iaas.NewNullableString(nil),
			},
			Model{
				Id:                 types.StringValue("pid,pipid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringNull(),
				Labels:             types.MapNull(types.StringType),
				NetworkInterfaceId: types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			Model{
				ProjectId:  types.StringValue("pid"),
				PublicIpId: types.StringValue("pipid"),
			},
			&iaas.PublicIp{
				Id: utils.Ptr("pipid"),
				Ip: utils.Ptr("ip"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				NetworkInterface: iaas.NewNullableString(utils.Ptr("interface")),
			},
			Model{
				Id:         types.StringValue("pid,pipid"),
				ProjectId:  types.StringValue("pid"),
				PublicIpId: types.StringValue("pipid"),
				Ip:         types.StringValue("ip"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				NetworkInterfaceId: types.StringValue("interface"),
			},
			true,
		},
		{
			"empty_labels",
			Model{
				ProjectId:  types.StringValue("pid"),
				PublicIpId: types.StringValue("pipid"),
				Labels:     types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			&iaas.PublicIp{
				Id:               utils.Ptr("pipid"),
				NetworkInterface: iaas.NewNullableString(utils.Ptr("interface")),
			},
			Model{
				Id:                 types.StringValue("pid,pipid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringNull(),
				Labels:             types.MapValueMust(types.StringType, map[string]attr.Value{}),
				NetworkInterfaceId: types.StringValue("interface"),
			},
			true,
		},
		{
			"network_interface_id_nil",
			Model{
				ProjectId:  types.StringValue("pid"),
				PublicIpId: types.StringValue("pipid"),
			},
			&iaas.PublicIp{
				Id: utils.Ptr("pipid"),
			},
			Model{
				Id:                 types.StringValue("pid,pipid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringNull(),
				Labels:             types.MapNull(types.StringType),
				NetworkInterfaceId: types.StringNull(),
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
			&iaas.PublicIp{},
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
