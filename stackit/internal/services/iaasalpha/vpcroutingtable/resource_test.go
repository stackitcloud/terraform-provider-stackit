package vpcroutingtable

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
	id := fmt.Sprintf("%s,%s,%s,%s", "oid", "aid", testRegion, "rtid")
	tests := []struct {
		description string
		state       Model
		input       *iaas.VPCRoutingTable
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId: types.StringValue("oid"),
				VpcId:     types.StringValue("aid"),
			},
			&iaas.VPCRoutingTable{
				Id:   new("rtid"),
				Name: "default_values",
			},
			Model{
				Id:             types.StringValue(id),
				ProjectId:      types.StringValue("oid"),
				RoutingTableId: types.StringValue("rtid"),
				Name:           types.StringValue("default_values"),
				VpcId:          types.StringValue("aid"),
				Labels:         types.MapNull(types.StringType),
				Region:         types.StringValue(testRegion),
			},
			true,
		},
		{
			"values_ok",
			Model{
				ProjectId: types.StringValue("oid"),
				VpcId:     types.StringValue("aid"),
			},
			&iaas.VPCRoutingTable{
				Id:          new("rtid"),
				Name:        "values_ok",
				Description: new("Description"),
				Labels: map[string]any{
					"key": "value",
				},
			},
			Model{
				Id:             types.StringValue(id),
				ProjectId:      types.StringValue("oid"),
				RoutingTableId: types.StringValue("rtid"),
				Name:           types.StringValue("values_ok"),
				Description:    types.StringValue("Description"),
				VpcId:          types.StringValue("aid"),
				Region:         types.StringValue(testRegion),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			true,
		},
		{
			"response_fields_nil_fail",
			Model{},
			&iaas.VPCRoutingTable{
				Id: nil,
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
				ProjectId: types.StringValue("oid"),
				VpcId:     types.StringValue("naid"),
			},
			&iaas.VPCRoutingTable{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, testRegion)
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
		expected    *iaas.AddVPCRoutingTablePayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &Model{
				Description: types.StringValue("Description"),
				Name:        types.StringValue("default_ok"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				SystemRoutes:  types.BoolValue(true),
				DynamicRoutes: types.BoolValue(true),
			},
			expected: &iaas.AddVPCRoutingTablePayload{
				Description: new("Description"),
				Name:        "default_ok",
				Labels: map[string]any{
					"key": "value",
				},
				SystemRoutes:  new(true),
				DynamicRoutes: new(true),
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
		expected    *iaas.UpdateVPCRoutingTablePayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Description: types.StringValue("Description"),
				Name:        types.StringValue("default_ok"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				}),
				DynamicRoutes: types.BoolValue(false),
				SystemRoutes:  types.BoolValue(false),
			},
			&iaas.UpdateVPCRoutingTablePayload{
				Description: new("Description"),
				Name:        new("default_ok"),
				Labels: map[string]any{
					"key1": "value1",
					"key2": "value2",
				},
				DynamicRoutes: new(false),
				SystemRoutes:  new(false),
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
