package publicipassociate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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
					ProjectId:          types.StringValue("pid"),
					PublicIpId:         types.StringValue("pipid"),
					NetworkInterfaceId: types.StringValue("nicid"),
				},
				input: &iaas.PublicIp{
					Id:               utils.Ptr("pipid"),
					NetworkInterface: iaas.NewNullableString(utils.Ptr("nicid")),
				},
				region: "eu01",
			},
			expected: Model{
				Id:                 types.StringValue("pid,eu01,pipid,nicid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringNull(),
				NetworkInterfaceId: types.StringValue("nicid"),
				Region:             types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			args: args{
				state: Model{
					ProjectId:          types.StringValue("pid"),
					PublicIpId:         types.StringValue("pipid"),
					NetworkInterfaceId: types.StringValue("nicid"),
				},
				input: &iaas.PublicIp{
					Id:               utils.Ptr("pipid"),
					Ip:               utils.Ptr("ip"),
					NetworkInterface: iaas.NewNullableString(utils.Ptr("nicid")),
				},
				region: "eu02",
			},
			expected: Model{
				Id:                 types.StringValue("pid,eu02,pipid,nicid"),
				ProjectId:          types.StringValue("pid"),
				PublicIpId:         types.StringValue("pipid"),
				Ip:                 types.StringValue("ip"),
				NetworkInterfaceId: types.StringValue("nicid"),
				Region:             types.StringValue("eu02"),
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
			err := mapFields(tt.args.input, &tt.args.state, tt.args.region)
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
