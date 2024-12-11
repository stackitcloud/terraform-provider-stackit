package image

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		state       DataSourceModel
		input       *iaasalpha.Image
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
			},
			&iaasalpha.Image{
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
			&iaasalpha.Image{
				Id:          utils.Ptr("iid"),
				Name:        utils.Ptr("name"),
				DiskFormat:  utils.Ptr("format"),
				MinDiskSize: utils.Ptr(int64(1)),
				MinRam:      utils.Ptr(int64(1)),
				Protected:   utils.Ptr(true),
				Scope:       utils.Ptr("scope"),
				Config: &iaasalpha.ImageConfig{
					BootMenu:               utils.Ptr(true),
					CdromBus:               iaasalpha.NewNullableString(utils.Ptr("bus")),
					DiskBus:                iaasalpha.NewNullableString(utils.Ptr("bus")),
					NicModel:               iaasalpha.NewNullableString(utils.Ptr("model")),
					OperatingSystem:        utils.Ptr("os"),
					OperatingSystemDistro:  iaasalpha.NewNullableString(utils.Ptr("distro")),
					OperatingSystemVersion: iaasalpha.NewNullableString(utils.Ptr("version")),
					RescueBus:              iaasalpha.NewNullableString(utils.Ptr("bus")),
					RescueDevice:           iaasalpha.NewNullableString(utils.Ptr("device")),
					SecureBoot:             utils.Ptr(true),
					Uefi:                   utils.Ptr(true),
					VideoModel:             iaasalpha.NewNullableString(utils.Ptr("model")),
					VirtioScsi:             utils.Ptr(true),
				},
				Checksum: &iaasalpha.ImageChecksum{
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
					"cdrom_bus":                types.StringNull(), // TODO: Should take value after SDK util fix
					"disk_bus":                 types.StringNull(), // TODO: Should take value after SDK util fix
					"nic_model":                types.StringNull(), // TODO: Should take value after SDK util fix
					"operating_system":         types.StringValue("os"),
					"operating_system_distro":  types.StringNull(), // TODO: Should take value after SDK util fix
					"operating_system_version": types.StringNull(), // TODO: Should take value after SDK util fix
					"rescue_bus":               types.StringNull(), // TODO: Should take value after SDK util fix
					"rescue_device":            types.StringNull(), // TODO: Should take value after SDK util fix
					"secure_boot":              types.BoolValue(true),
					"uefi":                     types.BoolValue(true),
					"video_model":              types.StringNull(), // TODO: Should take value after SDK util fix
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
			&iaasalpha.Image{
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
			&iaasalpha.Image{},
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
