package images

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func TestMapFields(t *testing.T) {
	ctx := context.Background()
	response := &iaas.ImageListResponse{Items: []iaas.Image{
		{Id: new("image-b"), Name: "second"},
		{
			Id:          new("image-a"),
			Name:        "first",
			DiskFormat:  "qcow2",
			MinDiskSize: new(int64(10)),
			MinRam:      new(int64(20)),
			Protected:   new(true),
			Scope:       new("project"),
			Status:      new("active"),
			Labels:      map[string]any{"env": "test"},
			Config: &iaas.ImageConfig{
				BootMenu:        new(true),
				OperatingSystem: new("linux"),
			},
			Checksum: &iaas.ImageChecksum{Algorithm: "sha256", Digest: "digest"},
		},
	}}
	model := DataSourceModel{ProjectID: types.StringValue("project")}

	if err := mapFields(ctx, response, &model, "eu01"); err != nil {
		t.Fatalf("mapFields returned error: %v", err)
	}

	config := types.ObjectValueMust(configAttrTypes, map[string]attr.Value{
		"boot_menu":                types.BoolValue(true),
		"cdrom_bus":                types.StringNull(),
		"disk_bus":                 types.StringNull(),
		"nic_model":                types.StringNull(),
		"operating_system":         types.StringValue("linux"),
		"operating_system_distro":  types.StringNull(),
		"operating_system_version": types.StringNull(),
		"rescue_bus":               types.StringNull(),
		"rescue_device":            types.StringNull(),
		"secure_boot":              types.BoolNull(),
		"uefi":                     types.BoolNull(),
		"video_model":              types.StringNull(),
		"virtio_scsi":              types.BoolNull(),
	})
	checksum := types.ObjectValueMust(checksumAttrTypes, map[string]attr.Value{
		"algorithm": types.StringValue("sha256"),
		"digest":    types.StringValue("digest"),
	})
	wantResults := types.ListValueMust(types.ObjectType{AttrTypes: imageAttrTypes}, []attr.Value{
		types.ObjectValueMust(imageAttrTypes, map[string]attr.Value{
			"image_id":      types.StringValue("image-a"),
			"name":          types.StringValue("first"),
			"disk_format":   types.StringValue("qcow2"),
			"min_disk_size": types.Int64Value(10),
			"min_ram":       types.Int64Value(20),
			"protected":     types.BoolValue(true),
			"scope":         types.StringValue("project"),
			"status":        types.StringValue("active"),
			"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
				"env": types.StringValue("test"),
			}),
			"config":   config,
			"checksum": checksum,
		}),
		types.ObjectValueMust(imageAttrTypes, map[string]attr.Value{
			"image_id":      types.StringValue("image-b"),
			"name":          types.StringValue("second"),
			"disk_format":   types.StringValue(""),
			"min_disk_size": types.Int64Null(),
			"min_ram":       types.Int64Null(),
			"protected":     types.BoolNull(),
			"scope":         types.StringNull(),
			"status":        types.StringNull(),
			"labels":        types.MapNull(types.StringType),
			"config":        types.ObjectNull(configAttrTypes),
			"checksum":      types.ObjectNull(checksumAttrTypes),
		}),
	})
	want := DataSourceModel{
		ID:        types.StringValue("project,eu01"),
		ProjectID: types.StringValue("project"),
		Region:    types.StringValue("eu01"),
		Results:   wantResults,
	}
	if diff := cmp.Diff(want, model); diff != "" {
		t.Fatalf("mapped data does not match (-want +got):\n%s", diff)
	}
}

func TestMapFieldsErrors(t *testing.T) {
	tests := []struct {
		name     string
		response *iaas.ImageListResponse
		model    *DataSourceModel
	}{
		{name: "nil response", model: &DataSourceModel{}},
		{name: "nil model", response: &iaas.ImageListResponse{}},
		{name: "image missing ID", response: &iaas.ImageListResponse{Items: []iaas.Image{{}}}, model: &DataSourceModel{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mapFields(context.Background(), tt.response, tt.model, "eu01"); err == nil {
				t.Fatal("expected mapping error")
			}
		})
	}
}
