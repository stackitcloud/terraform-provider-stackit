package image

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapDataSourceFields(t *testing.T) {
	type args struct {
		state  DataSourceModel
		input  *iaas.Image
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    DataSourceModel
		isValid     bool
	}{
		{
			description: "default_values",
			args: args{
				state: DataSourceModel{
					ProjectId: types.StringValue("pid"),
					ImageId:   types.StringValue("iid"),
				},
				input: &iaas.Image{
					Id: new("iid"),
				},
				region: "eu01",
			},
			expected: DataSourceModel{
				Id:        types.StringValue("pid,eu01,iid"),
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
				Labels:    types.MapNull(types.StringType),
				Region:    types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			args: args{
				state: DataSourceModel{
					ProjectId: types.StringValue("pid"),
					ImageId:   types.StringValue("iid"),
					Region:    types.StringValue("eu01"),
				},
				input: &iaas.Image{
					Id:          new("iid"),
					Name:        new("name"),
					DiskFormat:  new("format"),
					MinDiskSize: new(int64(1)),
					MinRam:      new(int64(1)),
					Protected:   new(true),
					Scope:       new("scope"),
					Config: &iaas.ImageConfig{
						BootMenu:               new(true),
						CdromBus:               iaas.NewNullableString(new("cdrom_bus")),
						DiskBus:                iaas.NewNullableString(new("disk_bus")),
						NicModel:               iaas.NewNullableString(new("model")),
						OperatingSystem:        new("os"),
						OperatingSystemDistro:  iaas.NewNullableString(new("os_distro")),
						OperatingSystemVersion: iaas.NewNullableString(new("os_version")),
						RescueBus:              iaas.NewNullableString(new("rescue_bus")),
						RescueDevice:           iaas.NewNullableString(new("rescue_device")),
						SecureBoot:             new(true),
						Uefi:                   new(true),
						VideoModel:             iaas.NewNullableString(new("model")),
						VirtioScsi:             new(true),
					},
					Checksum: &iaas.ImageChecksum{
						Algorithm: new("algorithm"),
						Digest:    new("digest"),
					},
					Labels: &map[string]any{
						"key": "value",
					},
				},
				region: "eu02",
			},
			expected: DataSourceModel{
				Id:          types.StringValue("pid,eu02,iid"),
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
				Region: types.StringValue("eu02"),
			},
			isValid: true,
		},
		{
			description: "empty_labels",
			args: args{
				state: DataSourceModel{
					ProjectId: types.StringValue("pid"),
					ImageId:   types.StringValue("iid"),
					Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
				},
				input: &iaas.Image{
					Id: new("iid"),
				},
				region: "eu01",
			},
			expected: DataSourceModel{
				Id:        types.StringValue("pid,eu01,iid"),
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
				Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
				Region:    types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "response_nil_fail",
		},
		{
			description: "no_resource_id",
			args: args{
				state: DataSourceModel{
					ProjectId: types.StringValue("pid"),
				},
				input: &iaas.Image{},
			},
			expected: DataSourceModel{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceFields(context.Background(), tt.args.input, &tt.args.state, tt.args.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.args.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
