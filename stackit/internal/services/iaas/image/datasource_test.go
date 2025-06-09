package image

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		state       DataSourceModel
		input       *iaas.Image
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
			},
			&iaas.Image{
				Id: utils.Ptr("iid"),
			},
			DataSourceModel{
				Id:        types.StringValue("pid,iid"),
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
				Labels:    types.MapNull(types.StringType),
			},
			true,
		},
		{
			"simple_values",
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
			},
			&iaas.Image{
				Id:          utils.Ptr("iid"),
				Name:        utils.Ptr("name"),
				DiskFormat:  utils.Ptr("format"),
				MinDiskSize: utils.Ptr(int64(1)),
				MinRam:      utils.Ptr(int64(1)),
				Protected:   utils.Ptr(true),
				Scope:       utils.Ptr("scope"),
				Config: &iaas.ImageConfig{
					BootMenu:               utils.Ptr(true),
					CdromBus:               iaas.NewNullableString(utils.Ptr("cdrom_bus")),
					DiskBus:                iaas.NewNullableString(utils.Ptr("disk_bus")),
					NicModel:               iaas.NewNullableString(utils.Ptr("model")),
					OperatingSystem:        utils.Ptr("os"),
					OperatingSystemDistro:  iaas.NewNullableString(utils.Ptr("os_distro")),
					OperatingSystemVersion: iaas.NewNullableString(utils.Ptr("os_version")),
					RescueBus:              iaas.NewNullableString(utils.Ptr("rescue_bus")),
					RescueDevice:           iaas.NewNullableString(utils.Ptr("rescue_device")),
					SecureBoot:             utils.Ptr(true),
					Uefi:                   utils.Ptr(true),
					VideoModel:             iaas.NewNullableString(utils.Ptr("model")),
					VirtioScsi:             utils.Ptr(true),
				},
				Checksum: &iaas.ImageChecksum{
					Algorithm: utils.Ptr("algorithm"),
					Digest:    utils.Ptr("digest"),
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			DataSourceModel{
				Id:          types.StringValue("pid,iid"),
				ProjectId:   types.StringValue("pid"),
				ImageId:     types.StringValue("iid"),
				Name:        types.StringValue("name"),
				DiskFormat:  types.StringValue("format"),
				MinDiskSize: types.Int64Value(1),
				MinRAM:      types.Int64Value(1),
				Protected:   types.BoolValue(true),
				Scope:       types.StringValue("scope"),
				Config: types.ObjectValueMust(configTypes, map[string]attr.Value{
					"boot_menu":                types.BoolValue(true),
					"cdrom_bus":                types.StringValue("cdrom_bus"),
					"disk_bus":                 types.StringValue("disk_bus"),
					"nic_model":                types.StringValue("model"),
					"operating_system":         types.StringValue("os"),
					"operating_system_distro":  types.StringValue("os_distro"),
					"operating_system_version": types.StringValue("os_version"),
					"rescue_bus":               types.StringValue("rescue_bus"),
					"rescue_device":            types.StringValue("rescue_device"),
					"secure_boot":              types.BoolValue(true),
					"uefi":                     types.BoolValue(true),
					"video_model":              types.StringValue("model"),
					"virtio_scsi":              types.BoolValue(true),
				}),
				Checksum: types.ObjectValueMust(checksumTypes, map[string]attr.Value{
					"algorithm": types.StringValue("algorithm"),
					"digest":    types.StringValue("digest"),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			true,
		},
		{
			"empty_labels",
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
				Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			&iaas.Image{
				Id: utils.Ptr("iid"),
			},
			DataSourceModel{
				Id:        types.StringValue("pid,iid"),
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
				Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			true,
		},
		{
			"response_nil_fail",
			DataSourceModel{},
			nil,
			DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
			},
			&iaas.Image{},
			DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceFields(context.Background(), tt.input, &tt.state)
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

func TestImageMatchesFilter(t *testing.T) {
	testCases := []struct {
		name     string
		img      *iaas.Image
		filter   *Filter
		expected bool
	}{
		{
			name:     "nil filter - always match",
			img:      &iaas.Image{Config: &iaas.ImageConfig{}},
			filter:   nil,
			expected: true,
		},
		{
			name:     "nil config - always false",
			img:      &iaas.Image{Config: nil},
			filter:   &Filter{},
			expected: false,
		},
		{
			name: "all fields match",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					OperatingSystem:        utils.Ptr("linux"),
					OperatingSystemDistro:  iaas.NewNullableString(utils.Ptr("ubuntu")),
					OperatingSystemVersion: iaas.NewNullableString(utils.Ptr("22.04")),
					Uefi:                   utils.Ptr(true),
					SecureBoot:             utils.Ptr(true),
				},
			},
			filter: &Filter{
				OS:         types.StringValue("linux"),
				Distro:     types.StringValue("ubuntu"),
				Version:    types.StringValue("22.04"),
				UEFI:       types.BoolValue(true),
				SecureBoot: types.BoolValue(true),
			},
			expected: true,
		},
		{
			name: "OS mismatch",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					OperatingSystem: utils.Ptr("windows"),
				},
			},
			filter: &Filter{
				OS: types.StringValue("linux"),
			},
			expected: false,
		},
		{
			name: "Distro mismatch",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					OperatingSystemDistro: iaas.NewNullableString(utils.Ptr("debian")),
				},
			},
			filter: &Filter{
				Distro: types.StringValue("ubuntu"),
			},
			expected: false,
		},
		{
			name: "Version mismatch",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					OperatingSystemVersion: iaas.NewNullableString(utils.Ptr("20.04")),
				},
			},
			filter: &Filter{
				Version: types.StringValue("22.04"),
			},
			expected: false,
		},
		{
			name: "UEFI mismatch",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					Uefi: utils.Ptr(false),
				},
			},
			filter: &Filter{
				UEFI: types.BoolValue(true),
			},
			expected: false,
		},
		{
			name: "SecureBoot mismatch",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					SecureBoot: utils.Ptr(false),
				},
			},
			filter: &Filter{
				SecureBoot: types.BoolValue(true),
			},
			expected: false,
		},
		{
			name: "SecureBoot match - true",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					SecureBoot: utils.Ptr(true),
				},
			},
			filter: &Filter{
				SecureBoot: types.BoolValue(true),
			},
			expected: true,
		},
		{
			name: "SecureBoot match - false",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					SecureBoot: utils.Ptr(false),
				},
			},
			filter: &Filter{
				SecureBoot: types.BoolValue(false),
			},
			expected: true,
		},
		{
			name: "SecureBoot field missing in image but required in filter",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					SecureBoot: nil,
				},
			},
			filter: &Filter{
				SecureBoot: types.BoolValue(true),
			},
			expected: false,
		},
		{
			name: "partial filter match - only distro set and match",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					OperatingSystemDistro: iaas.NewNullableString(utils.Ptr("ubuntu")),
				},
			},
			filter: &Filter{
				Distro: types.StringValue("ubuntu"),
			},
			expected: true,
		},
		{
			name: "partial filter match - distro mismatch",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					OperatingSystemDistro: iaas.NewNullableString(utils.Ptr("centos")),
				},
			},
			filter: &Filter{
				Distro: types.StringValue("ubuntu"),
			},
			expected: false,
		},
		{
			name: "filter provided but attribute is null in image",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					OperatingSystemDistro: nil,
				},
			},
			filter: &Filter{
				Distro: types.StringValue("ubuntu"),
			},
			expected: false,
		},
		{
			name: "image has valid config, but filter has null values",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					OperatingSystem:        utils.Ptr("linux"),
					OperatingSystemDistro:  iaas.NewNullableString(utils.Ptr("ubuntu")),
					OperatingSystemVersion: iaas.NewNullableString(utils.Ptr("22.04")),
					Uefi:                   utils.Ptr(false),
					SecureBoot:             utils.Ptr(false),
				},
			},
			filter: &Filter{
				OS:         types.StringNull(),
				Distro:     types.StringNull(),
				Version:    types.StringNull(),
				UEFI:       types.BoolNull(),
				SecureBoot: types.BoolNull(),
			},
			expected: true,
		},
		{
			name: "image has nil fields in config, filter expects values",
			img: &iaas.Image{
				Config: &iaas.ImageConfig{
					OperatingSystem:        nil,
					OperatingSystemDistro:  nil,
					OperatingSystemVersion: nil,
					Uefi:                   nil,
					SecureBoot:             nil,
				},
			},
			filter: &Filter{
				OS:         types.StringValue("linux"),
				Distro:     types.StringValue("ubuntu"),
				Version:    types.StringValue("22.04"),
				UEFI:       types.BoolValue(true),
				SecureBoot: types.BoolValue(true),
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := imageMatchesFilter(tc.img, tc.filter)
			if result != tc.expected {
				t.Errorf("Expected match = %v, got %v", tc.expected, result)
			}
		})
	}
}
