package publicipranges

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	coreUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func TestMapPublicIpRanges(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		input    *[]iaas.PublicNetwork
		expected Model
		isValid  bool
	}{
		{
			name:    "nil input should return error",
			input:   nil,
			isValid: false,
		},
		{
			name:  "empty input should return nulls",
			input: &[]iaas.PublicNetwork{},
			expected: Model{
				PublicIpRanges: types.ListNull(types.ObjectType{AttrTypes: publicIpRangesTypes}),
				CidrList:       types.ListNull(types.StringType),
			},
			isValid: true,
		},
		{
			name: "valid cidr entries",
			input: &[]iaas.PublicNetwork{
				{Cidr: coreUtils.Ptr("192.168.0.0/24")},
				{Cidr: coreUtils.Ptr("192.168.1.0/24")},
			},
			expected: func() Model {
				cidrs := []string{"192.168.0.0/24", "192.168.1.0/24"}
				ipRangesList := make([]attr.Value, 0, len(cidrs))
				for _, cidr := range cidrs {
					ipRange, _ := types.ObjectValue(publicIpRangesTypes, map[string]attr.Value{
						"cidr": types.StringValue(cidr),
					})
					ipRangesList = append(ipRangesList, ipRange)
				}
				ipRangesVal, _ := types.ListValue(types.ObjectType{AttrTypes: publicIpRangesTypes}, ipRangesList)
				cidrListVal, _ := types.ListValueFrom(ctx, types.StringType, cidrs)

				return Model{
					PublicIpRanges: ipRangesVal,
					CidrList:       cidrListVal,
					Id:             utils.BuildInternalTerraformId(cidrs...),
				}
			}(),
			isValid: true,
		},
		{
			name: "filter out empty CIDRs",
			input: &[]iaas.PublicNetwork{
				{Cidr: coreUtils.Ptr("")},
				{Cidr: nil},
				{Cidr: coreUtils.Ptr("10.0.0.0/8")},
			},
			expected: func() Model {
				cidrs := []string{"10.0.0.0/8"}
				ipRange, _ := types.ObjectValue(publicIpRangesTypes, map[string]attr.Value{
					"cidr": types.StringValue("10.0.0.0/8"),
				})
				ipRangesVal, _ := types.ListValue(types.ObjectType{AttrTypes: publicIpRangesTypes}, []attr.Value{ipRange})
				cidrListVal, _ := types.ListValueFrom(ctx, types.StringType, cidrs)
				return Model{
					PublicIpRanges: ipRangesVal,
					CidrList:       cidrListVal,
					Id:             utils.BuildInternalTerraformId(cidrs...),
				}
			}(),
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var model Model
			err := mapPublicIpRanges(ctx, tt.input, &model)

			if !tt.isValid {
				if err == nil {
					t.Fatalf("Expected error but got nil")
				}
				return
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.expected.Id, model.Id); diff != "" {
				t.Errorf("ID does not match:\n%s", diff)
			}

			if diff := cmp.Diff(tt.expected.CidrList, model.CidrList); diff != "" {
				t.Errorf("cidr_list does not match:\n%s", diff)
			}

			if diff := cmp.Diff(tt.expected.PublicIpRanges, model.PublicIpRanges); diff != "" {
				t.Errorf("public_ip_ranges does not match:\n%s", diff)
			}
		})
	}
}
