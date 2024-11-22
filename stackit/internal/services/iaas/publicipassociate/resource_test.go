package publicipassociate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				NetworkInterfaceId: types.StringValue("nicid"),
			},
			&iaas.PublicIp{
				Id:               utils.Ptr("pipid"),
				NetworkInterface: iaas.NewNullableString(utils.Ptr("nicid")),
			},
			Model{
				Id:                 types.StringValue("pid,pipid,nicid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringNull(),
				NetworkInterfaceId: types.StringValue("nicid"),
			},
			true,
		},
		{
			"simple_values",
			Model{
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				NetworkInterfaceId: types.StringValue("nicid"),
			},
			&iaas.PublicIp{
				Id:               utils.Ptr("pipid"),
				Ip:               utils.Ptr("ip"),
				NetworkInterface: iaas.NewNullableString(utils.Ptr("nicid")),
			},
			Model{
				Id:                 types.StringValue("pid,pipid,nicid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringValue("ip"),
				NetworkInterfaceId: types.StringValue("nicid"),
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
		expected    *iaas.UpdatePublicIPPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				NetworkInterfaceId: types.StringValue("interface"),
			},
			&iaas.UpdatePublicIPPayload{
				NetworkInterface: iaas.NewNullableString(utils.Ptr("interface")),
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
