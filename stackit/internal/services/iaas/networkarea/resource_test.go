package networkarea

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaas.NetworkArea
		expected    Model
		isValid     bool
	}{
		{
			description: "id_ok",
			state: Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
			},
			input: &iaas.NetworkArea{
				Id: new("naid"),
			},
			expected: Model{
				Id:             types.StringValue("oid,naid"),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				Name:           types.StringValue(""),
				Labels:         types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "values_ok",
			state: Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
			},
			input: &iaas.NetworkArea{
				Id:   new("naid"),
				Name: "name",
				Labels: map[string]any{
					"key": "value",
				},
			},
			expected: Model{
				Id:             types.StringValue("oid,naid"),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				Name:           types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			isValid: true,
		},
		{
			description: "default_nameservers_changed_outside_tf",
			state: Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
			},
			input: &iaas.NetworkArea{
				Id: new("naid"),
			},
			expected: Model{
				Id:             types.StringValue("oid,naid"),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				Name:           types.StringValue(""),
				Labels:         types.MapNull(types.StringType),
			},
			isValid: true,
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
			},
			&iaas.NetworkArea{},
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
		expected    *iaas.CreateNetworkAreaPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaas.CreateNetworkAreaPayload{
				Name: "name",
				Labels: map[string]any{
					"key": "value",
				},
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
		expected    *iaas.PartialUpdateNetworkAreaPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaas.PartialUpdateNetworkAreaPayload{
				Name: new("name"),
				Labels: map[string]any{
					"key": "value",
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
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
