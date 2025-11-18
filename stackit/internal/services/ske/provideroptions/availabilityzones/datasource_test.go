package availabilityzones

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
)

// TODO: fix tests
func TestMapFields(t *testing.T) {
	timestamp := time.Date(2025, 2, 5, 10, 20, 30, 0, time.UTC)
	tests := []struct {
		name     string
		input    *ske.ProviderOptions
		expected *Model
		isValid  bool
	}{
		{
			name: "normal_case",
			input: &ske.ProviderOptions{
				AvailabilityZones: &[]ske.AvailabilityZone{
					{Name: utils.Ptr("eu01-01")},
					{Name: utils.Ptr("eu01-02")},
				},
				VolumeTypes: &[]ske.VolumeType{
					{Name: utils.Ptr("storage_premium_perf1")},
					{Name: utils.Ptr("storage_premium_perf2")},
				},
				KubernetesVersions: &[]ske.KubernetesVersion{
					{
						Version:        utils.Ptr("1.33.5"),
						State:          utils.Ptr("supported"),
						ExpirationDate: &timestamp,
					},
				},
				MachineTypes: &[]ske.MachineType{
					{
						Name:         utils.Ptr("n2.56d.g4"),
						Architecture: utils.Ptr("amd64"),
						Cpu:          utils.Ptr(int64(4)),
						Gpu:          utils.Ptr(int64(1)),
						Memory:       utils.Ptr(int64(16)),
					},
				},
				MachineImages: &[]ske.MachineImage{
					{
						Name: utils.Ptr("ubuntu"),
						Versions: &[]ske.MachineImageVersion{
							{
								Version:        utils.Ptr("2204.20250620.0"),
								State:          utils.Ptr("supported"),
								ExpirationDate: &timestamp,
								Cri: &[]ske.CRI{
									{Name: utils.Ptr(ske.CRINAME_CONTAINERD)},
								},
							},
						},
					},
				},
			},
			expected: &Model{
				AvailabilityZones: types.ListValueMust(
					types.StringType,
					[]attr.Value{
						types.StringValue("eu01-01"),
						types.StringValue("eu01-02"),
					},
				),
			},
			isValid: true,
		},
		{
			name: "partial_fields",
			input: &ske.ProviderOptions{
				AvailabilityZones: &[]ske.AvailabilityZone{
					{Name: utils.Ptr("eu01-01")},
				},
				MachineTypes: &[]ske.MachineType{
					{
						Name: utils.Ptr("g1a.16d"),
						Cpu:  utils.Ptr(int64(2)),
					},
				},
			},
			expected: &Model{
				AvailabilityZones: types.ListValueMust(types.StringType,
					[]attr.Value{types.StringValue("eu01-01")},
				),
			},
			isValid: true,
		},
		{
			name: "az_with_nil_name",
			input: &ske.ProviderOptions{
				AvailabilityZones: &[]ske.AvailabilityZone{
					{Name: nil},
					{Name: utils.Ptr("eu01-01")},
				},
			},
			expected: &Model{
				AvailabilityZones: types.ListValueMust(
					types.StringType,
					[]attr.Value{types.StringValue("eu01-01")},
				),
			},
			isValid: true,
		},
		{
			name: "machine_image_with_nil_versions",
			input: &ske.ProviderOptions{
				MachineImages: &[]ske.MachineImage{
					{
						Name:     utils.Ptr("ubuntu"),
						Versions: nil,
					},
				},
			},
			expected: &Model{
				AvailabilityZones: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			isValid: true,
		},
		{
			name: "image_version_with_nil_cri",
			input: &ske.ProviderOptions{
				MachineImages: &[]ske.MachineImage{
					{
						Name: utils.Ptr("ubuntu"),
						Versions: &[]ske.MachineImageVersion{
							{
								Version:        utils.Ptr("1.1"),
								State:          utils.Ptr("deprecated"),
								ExpirationDate: &timestamp,
								Cri:            nil,
							},
						},
					},
				},
			},
			expected: &Model{
				AvailabilityZones: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			isValid: true,
		},
		{
			name: "machine_type_null_fields",
			input: &ske.ProviderOptions{
				MachineTypes: &[]ske.MachineType{
					{}, // all pointer fields are nil
				},
			},
			expected: &Model{
				AvailabilityZones: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			isValid: true,
		},
		{
			name: "all_nil_fields",
			input: &ske.ProviderOptions{
				AvailabilityZones:  nil,
				VolumeTypes:        nil,
				KubernetesVersions: nil,
				MachineImages:      nil,
				MachineTypes:       nil,
			},
			expected: &Model{
				AvailabilityZones: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			isValid: true,
		},
		{
			name: "all_empty_fields",
			input: &ske.ProviderOptions{
				AvailabilityZones:  &[]ske.AvailabilityZone{},
				VolumeTypes:        &[]ske.VolumeType{},
				KubernetesVersions: &[]ske.KubernetesVersion{},
				MachineImages:      &[]ske.MachineImage{},
				MachineTypes:       &[]ske.MachineType{},
			},
			expected: &Model{
				AvailabilityZones: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &Model{}
			err := mapFields(context.Background(), tt.input, model)

			if tt.isValid && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tt.isValid && err == nil {
				t.Fatal("expected error but got none")
			}

			if tt.isValid {
				if diff := cmp.Diff(tt.expected, model); diff != "" {
					t.Fatalf("Mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
