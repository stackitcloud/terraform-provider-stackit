package networkarearoute

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
		input       *iaas.Route
		expected    Model
		isValid     bool
	}{
		{
			"id_ok",
			Model{
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				NetworkAreaRouteId: types.StringValue("narid"),
			},
			&iaas.Route{},
			Model{
				Id:                 types.StringValue("oid,naid,narid"),
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				NetworkAreaRouteId: types.StringValue("narid"),
				Prefix:             types.StringNull(),
				NextHop:            types.StringNull(),
			},
			true,
		},
		{
			"values_ok",
			Model{
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				NetworkAreaRouteId: types.StringValue("narid"),
			},
			&iaas.Route{
				Prefix:  utils.Ptr("prefix"),
				Nexthop: utils.Ptr("hop"),
			},
			Model{
				Id:                 types.StringValue("oid,naid,narid"),
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				NetworkAreaRouteId: types.StringValue("narid"),
				Prefix:             types.StringValue("prefix"),
				NextHop:            types.StringValue("hop"),
			},
			true,
		},
		{
			"response_fields_nil_fail",
			Model{},
			&iaas.Route{
				Prefix:  nil,
				Nexthop: nil,
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
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
			},
			&iaas.Route{},
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
		expected    *iaas.CreateNetworkAreaRoutePayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &Model{
				Prefix:  types.StringValue("prefix"),
				NextHop: types.StringValue("hop"),
			},
			expected: &iaas.CreateNetworkAreaRoutePayload{
				Ipv4: &[]iaas.Route{
					{
						Prefix:  utils.Ptr("prefix"),
						Nexthop: utils.Ptr("hop"),
					},
				},
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
