package vpcregion

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"
)

func TestMapFields(t *testing.T) {
	const (
		projectId = "project-id"
		vpcId     = "vpc-id"
		region    = "region"
		id        = projectId + "," + vpcId + "," + region
	)

	tests := []struct {
		description string
		state       *SharedModel
		input       *iaas.RegionalVPC
		expected    SharedModel
		isValid     bool
	}{
		{
			description: "default_ok",
			state: &SharedModel{
				ProjectId: types.StringValue(projectId),
				VPCId:     types.StringValue(vpcId),
				Region:    types.StringValue(region),
			},
			input: &iaas.RegionalVPC{},
			expected: SharedModel{
				Id:        types.StringValue(id),
				ProjectId: types.StringValue(projectId),
				VPCId:     types.StringValue(vpcId),
				Region:    types.StringValue(region),
			},
			isValid: true,
		},
		{
			description: "no_ipv4",
			state: &SharedModel{
				ProjectId: types.StringValue(projectId),
				VPCId:     types.StringValue(vpcId),
				Region:    types.StringValue(region),
			},
			input: &iaas.RegionalVPC{},
			expected: SharedModel{
				Id:        types.StringValue(id),
				ProjectId: types.StringValue(projectId),
				VPCId:     types.StringValue(vpcId),
				Region:    types.StringValue(region),
			},
			isValid: true,
		},
		{
			description: "empty_nameservers",
			state: &SharedModel{
				ProjectId: types.StringValue(projectId),
				VPCId:     types.StringValue(vpcId),
				Region:    types.StringValue(region),
			},
			input: &iaas.RegionalVPC{
				Ipv4: &iaas.RegionalVPCIPv4{
					DefaultNameservers: []string{},
				},
			},
			expected: SharedModel{
				Id:        types.StringValue(id),
				ProjectId: types.StringValue(projectId),
				VPCId:     types.StringValue(vpcId),
				Region:    types.StringValue(region),
			},
			isValid: true,
		},
		{
			description: "response nil fails",
			state: &SharedModel{
				ProjectId: types.StringValue(projectId),
				VPCId:     types.StringValue(vpcId),
				Region:    types.StringValue(region),
			},
			input:    nil,
			expected: SharedModel{},
			isValid:  false,
		},
		{
			description: "model nil fails",
			state:       nil,
			input: &iaas.RegionalVPC{
				Ipv4: &iaas.RegionalVPCIPv4{
					DefaultNameservers: []string{"1.1.1.1", "8.8.8.8"},
				},
			},
			expected: SharedModel{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(*tt.state, tt.expected, cmpopts.IgnoreUnexported(Model{}, ipv4Model{}))
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
		input       *SharedModel
		expected    *iaas.CreateVPCRegionPayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input:       &SharedModel{},
			expected:    &iaas.CreateVPCRegionPayload{},
			isValid:     true,
		},
		{
			description: "no_ipv4",
			input:       &SharedModel{},
			expected: &iaas.CreateVPCRegionPayload{
				Ipv4: nil,
			},
			isValid: true,
		},
		{
			description: "nil_model_fails",
			input:       nil,
			expected:    nil,
			isValid:     false,
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
		input       *SharedModel
		expected    iaas.UpdateVPCRegionPayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input:       &SharedModel{},
			expected:    iaas.UpdateVPCRegionPayload{},
			isValid:     true,
		},
		{
			description: "nil_model_fails",
			input:       nil,
			expected:    iaas.UpdateVPCRegionPayload{},
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input)
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
