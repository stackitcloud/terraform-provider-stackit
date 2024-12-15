package virtualip

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaasalpha.VirtualIp
		expected    Model
		isValid     bool
	}{
		{
			"id_ok",
			Model{
				ProjectId:   types.StringValue("pid"),
				NetworkId:   types.StringValue("nid"),
				VirtualIpId: types.StringValue("vipid"),
			},
			&iaasalpha.VirtualIp{
				Id: utils.Ptr("pid,nid,vipid"),
			},
			Model{
				Id:          types.StringValue("pid,nid,vipid"),
				ProjectId:   types.StringValue("pid"),
				NetworkId:   types.StringValue("nid"),
				VirtualIpId: types.StringValue("vipid"),
				Name:        types.StringNull(),
				IP:          types.StringNull(),
				Labels:      types.MapNull(types.StringType),
			},
			true,
		},
		{
			"values_ok",
			Model{
				ProjectId:   types.StringValue("pid"),
				NetworkId:   types.StringValue("nid"),
				VirtualIpId: types.StringValue("vipid"),
			},
			&iaasalpha.VirtualIp{
				Id:   utils.Ptr("pid,nid,vipid"),
				Name: utils.Ptr("vip-name"),
				Ip:   utils.Ptr("10.0.0.1"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			Model{
				Id:          types.StringValue("pid,nid,vipid"),
				ProjectId:   types.StringValue("pid"),
				NetworkId:   types.StringValue("nid"),
				VirtualIpId: types.StringValue("vipid"),
				Name:        types.StringValue("vip-name"),
				IP:          types.StringValue("10.0.0.1"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			true,
		},
		{
			"response_fields_nil_fail",
			Model{},
			&iaasalpha.VirtualIp{
				Name: nil,
				Ip:   nil,
			},
			Model{},
			false,
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
				NetworkId: types.StringValue("nid"),
			},
			&iaasalpha.VirtualIp{},
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
		expected    *iaasalpha.CreateVirtualIPPayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &Model{
				Name: types.StringValue("vip-name"),
				IP:   types.StringValue("10.0.0.1"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			expected: &iaasalpha.CreateVirtualIPPayload{
				Name: utils.Ptr("vip-name"),
				Ip:   utils.Ptr("10.0.0.1"),
				Labels: &map[string]interface{}{
					"key": "value",
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
		expected    *iaasalpha.UpdateVirtualIPPayload
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
			&iaasalpha.UpdateVirtualIPPayload{
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaasalpha.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
