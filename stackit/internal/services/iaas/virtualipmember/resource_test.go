package virtualipmember

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				VirtualIpId:        types.StringValue("vipid"),
				NetworkInterfaceId: types.StringValue("nicid"),
			},
			&iaasalpha.VirtualIp{
				Id: utils.Ptr("pid,nid,vipid"),
			},
			Model{
				Id:                 types.StringValue("pid,nid,vipid,nicid"),
				ProjectId:          types.StringValue("pid"),
				NetworkId:          types.StringValue("nid"),
				VirtualIpId:        types.StringValue("vipid"),
				NetworkInterfaceId: types.StringValue("nicid"),
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
			err := mapFields(tt.input, &tt.state)
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
		expected    *iaasalpha.AddMemberToVirtualIPPayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &Model{
				NetworkInterfaceId: types.StringValue("nic-id"),
			},
			expected: &iaasalpha.AddMemberToVirtualIPPayload{
				Member: utils.Ptr("nic-id"),
			},
			isValid: true,
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

func TestToDeletePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *iaasalpha.RemoveMemberFromVirtualIPPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				NetworkInterfaceId: types.StringValue("nic-id"),
			},
			&iaasalpha.RemoveMemberFromVirtualIPPayload{
				Member: utils.Ptr("nic-id"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toDeletePayload(tt.input)
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
