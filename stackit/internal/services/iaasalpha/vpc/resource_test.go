package vpc

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"
)

func TestMapFields(t *testing.T) {
	const (
		projectId = "project-id"
		vpcId     = "vpc-id"
		id        = projectId + "," + vpcId

		name        = "name"
		description = "description"
	)

	tests := []struct {
		description string
		state       Model
		input       *iaas.VPC
		expected    Model
		isValid     bool
	}{
		{
			description: "id_ok",
			state: Model{
				ProjectId:   types.StringValue(projectId),
				Name:        types.StringValue(""),
				Description: types.StringValue(""),
				Labels:      types.MapNull(types.StringType),
			},
			input: &iaas.VPC{
				Id: vpcId,
			},
			expected: Model{
				Id:          types.StringValue(id),
				ProjectId:   types.StringValue(projectId),
				VpcId:       types.StringValue(vpcId),
				Name:        types.StringValue(""),
				Description: types.StringValue(""),
				Labels:      types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "values_ok",
			state: Model{
				ProjectId:   types.StringValue(projectId),
				Name:        types.StringValue(name),
				Description: types.StringValue(description),
				Labels:      types.MapNull(types.StringType),
			},
			input: &iaas.VPC{
				Id:          vpcId,
				Name:        name,
				Description: description,
			},
			expected: Model{
				Id:          types.StringValue(id),
				ProjectId:   types.StringValue(projectId),
				VpcId:       types.StringValue(vpcId),
				Name:        types.StringValue(name),
				Description: types.StringValue(description),
				Labels:      types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "response nil fails",
			state: Model{
				ProjectId:   types.StringValue(projectId),
				Name:        types.StringValue(name),
				Description: types.StringValue(description),
				Labels:      types.MapNull(types.StringType),
			},
			input:    nil,
			expected: Model{},
			isValid:  false,
		},
		{
			description: "no response id",
			state: Model{
				ProjectId:   types.StringValue(projectId),
				Name:        types.StringValue(name),
				Description: types.StringValue(description),
				Labels:      types.MapNull(types.StringType),
			},
			input:    &iaas.VPC{},
			expected: Model{},
			isValid:  false,
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
		expected    *iaas.CreateVPCPayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			expected: &iaas.CreateVPCPayload{
				Name:        "name",
				Description: new("description"),
				Labels: map[string]interface{}{
					"key": "value",
				},
			},
			isValid: true,
		},
		{
			description: "no labels",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				Labels:      types.MapNull(types.StringType),
			},
			expected: &iaas.CreateVPCPayload{
				Name:        "name",
				Description: new("description"),
				Labels:      map[string]interface{}{},
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
		description   string
		input         *Model
		currentLabels types.Map
		expected      *iaas.PartialUpdateVPCPayload
		isValid       bool
	}{
		{
			description: "default_ok",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{}),
			expected: &iaas.PartialUpdateVPCPayload{
				Name:        new("name"),
				Description: new("description"),
				Labels: map[string]interface{}{
					"key": "value",
				},
			},
			isValid: true,
		},
		{
			description: "no labels",
			input: &Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				Labels:      types.MapNull(types.StringType),
			},
			expected: &iaas.PartialUpdateVPCPayload{
				Name:        new("name"),
				Description: new("description"),
				Labels:      map[string]interface{}{},
			},
			isValid: true,
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
