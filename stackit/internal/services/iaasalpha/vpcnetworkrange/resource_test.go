package vpcnetworkrange

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
		projectId      = "project-id"
		vpcId          = "vpc-id"
		region         = "region"
		networkRangeId = "network-range-id"
		id             = projectId + "," + vpcId + "," + region + "," + networkRangeId

		name                      = "name"
		description               = "description"
		prefix                    = "192.168.0.0/24"
		defaultPrefixLength int64 = 24
		minPrefixLength     int64 = 23
		maxPrefixLength     int64 = 30
	)

	tests := []struct {
		description string
		state       SharedModel
		input       *iaas.VPCNetworkRange
		expected    SharedModel
		isValid     bool
	}{
		{
			description: "id_ok",
			state: SharedModel{
				ProjectId:           types.StringValue(projectId),
				VpcId:               types.StringValue(vpcId),
				Region:              types.StringValue(region),
				Id:                  types.StringNull(),
				NetworkRangeId:      types.StringNull(),
				Description:         types.StringNull(),
				IpVersion:           types.StringNull(),
				DefaultPrefixLength: types.Int64Null(),
				MaxPrefixLength:     types.Int64Null(),
				MinPrefixLength:     types.Int64Null(),
				Labels:              types.MapNull(types.StringType),
				Nameservers:         types.ListNull(types.StringType),
				Prefix:              types.StringNull(),
			},
			input: &iaas.VPCNetworkRange{
				VPCNetworkRangeIPv4: &iaas.VPCNetworkRangeIPv4{
					Id: new(networkRangeId),
				},
			},
			expected: SharedModel{
				Id:                  types.StringValue(id),
				ProjectId:           types.StringValue(projectId),
				VpcId:               types.StringValue(vpcId),
				NetworkRangeId:      types.StringValue(networkRangeId),
				Region:              types.StringValue(region),
				IpVersion:           types.StringValue(""),
				Prefix:              types.StringValue(""),
				Description:         types.StringNull(),
				DefaultPrefixLength: types.Int64Null(),
				MaxPrefixLength:     types.Int64Null(),
				MinPrefixLength:     types.Int64Null(),
				Labels:              types.MapNull(types.StringType),
				Nameservers:         types.ListNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "values_ok",
			state: SharedModel{
				Id:                  types.StringValue(id),
				ProjectId:           types.StringValue(projectId),
				VpcId:               types.StringValue(vpcId),
				NetworkRangeId:      types.StringValue(networkRangeId),
				Description:         types.StringValue(description),
				IpVersion:           types.StringValue(string(iaas.VPCNETWORKRANGEIPV4ALLOFIPVERSION_IPV4)),
				DefaultPrefixLength: types.Int64Value(defaultPrefixLength),
				MaxPrefixLength:     types.Int64Value(maxPrefixLength),
				MinPrefixLength:     types.Int64Value(minPrefixLength),
				Prefix:              types.StringValue(prefix),
				Region:              types.StringValue(region),
				Labels:              types.MapNull(types.StringType),
				Nameservers:         types.ListNull(types.StringType),
			},
			input: &iaas.VPCNetworkRange{
				VPCNetworkRangeIPv4: &iaas.VPCNetworkRangeIPv4{
					Id:               new(networkRangeId),
					DefaultPrefixLen: new(defaultPrefixLength),
					Description:      new(description),
					IpVersion:        iaas.VPCNETWORKRANGEIPV4ALLOFIPVERSION_IPV4,
					MaxPrefixLen:     new(maxPrefixLength),
					MinPrefixLen:     new(minPrefixLength),
					Prefix:           prefix,
				},
			},
			expected: SharedModel{
				Id:                  types.StringValue(id),
				ProjectId:           types.StringValue(projectId),
				VpcId:               types.StringValue(vpcId),
				NetworkRangeId:      types.StringValue(networkRangeId),
				Description:         types.StringValue(description),
				IpVersion:           types.StringValue(string(iaas.VPCNETWORKRANGEIPV4ALLOFIPVERSION_IPV4)),
				DefaultPrefixLength: types.Int64Value(defaultPrefixLength),
				MaxPrefixLength:     types.Int64Value(maxPrefixLength),
				MinPrefixLength:     types.Int64Value(minPrefixLength),
				Prefix:              types.StringValue(prefix),
				Region:              types.StringValue(region),
				Labels:              types.MapNull(types.StringType),
				Nameservers:         types.ListNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "response nil fails",
			state: SharedModel{
				ProjectId:           types.StringValue(projectId),
				VpcId:               types.StringValue(vpcId),
				Region:              types.StringValue(region),
				Id:                  types.StringNull(),
				NetworkRangeId:      types.StringNull(),
				Description:         types.StringNull(),
				IpVersion:           types.StringNull(),
				DefaultPrefixLength: types.Int64Null(),
				MaxPrefixLength:     types.Int64Null(),
				MinPrefixLength:     types.Int64Null(),
				Labels:              types.MapNull(types.StringType),
				Nameservers:         types.ListNull(types.StringType),
				Prefix:              types.StringNull(),
			},
			input:    nil,
			expected: SharedModel{},
			isValid:  false,
		},
		{
			description: "no response id",
			state: SharedModel{
				ProjectId:           types.StringValue(projectId),
				VpcId:               types.StringValue(vpcId),
				Region:              types.StringValue(region),
				Id:                  types.StringNull(),
				NetworkRangeId:      types.StringNull(),
				Description:         types.StringNull(),
				IpVersion:           types.StringNull(),
				DefaultPrefixLength: types.Int64Null(),
				MaxPrefixLength:     types.Int64Null(),
				MinPrefixLength:     types.Int64Null(),
				Labels:              types.MapNull(types.StringType),
				Nameservers:         types.ListNull(types.StringType),
				Prefix:              types.StringNull(),
			},
			input:    &iaas.VPCNetworkRange{},
			expected: SharedModel{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, region)
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
		input       *SharedModel
		expected    *iaas.CreateVPCNetworkRangePayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &SharedModel{
				Description: types.StringValue("description"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			expected: &iaas.CreateVPCNetworkRangePayload{
				Description: new("description"),
				Labels: map[string]interface{}{
					"key": "value",
				},
			},
			isValid: true,
		},
		{
			description: "no labels",
			input: &SharedModel{
				Description: types.StringValue("description"),
				Labels:      types.MapNull(types.StringType),
			},
			expected: &iaas.CreateVPCNetworkRangePayload{
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
		input         *SharedModel
		currentLabels types.Map
		expected      *iaas.UpdateVPCNetworkRangePayload
		isValid       bool
	}{
		{
			description: "default_ok",
			input: &SharedModel{
				Description: types.StringValue("description"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{}),
			expected: &iaas.UpdateVPCNetworkRangePayload{
				Description: new("description"),
				Labels: map[string]interface{}{
					"key": "value",
				},
			},
			isValid: true,
		},
		{
			description: "no labels",
			input: &SharedModel{
				Description: types.StringValue("description"),
				Labels:      types.MapNull(types.StringType),
			},
			expected: &iaas.UpdateVPCNetworkRangePayload{
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
