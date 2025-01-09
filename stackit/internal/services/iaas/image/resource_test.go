package image

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaas.Image
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
			},
			&iaas.Image{
				Id: utils.Ptr("iid"),
			},
			Model{
				Id:        types.StringValue("pid,iid"),
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
				Labels:    types.MapNull(types.StringType),
			},
			true,
		},
		{
			"simple_values",
			Model{
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
			Model{
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
			Model{
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
				Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			&iaas.Image{
				Id: utils.Ptr("iid"),
			},
			Model{
				Id:        types.StringValue("pid,iid"),
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
				Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			true,
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
				ProjectId: types.StringValue("pid"),
			},
			&iaas.Image{},
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
		expected    *iaas.CreateImagePayload
		isValid     bool
	}{
		{
			"ok",
			&Model{
				Id:          types.StringValue("pid,iid"),
				ProjectId:   types.StringValue("pid"),
				ImageId:     types.StringValue("iid"),
				Name:        types.StringValue("name"),
				DiskFormat:  types.StringValue("format"),
				MinDiskSize: types.Int64Value(1),
				MinRAM:      types.Int64Value(1),
				Protected:   types.BoolValue(true),
				Config: types.ObjectValueMust(configTypes, map[string]attr.Value{
					"boot_menu":                types.BoolValue(true),
					"cdrom_bus":                types.StringValue("cdrom_bus"),
					"disk_bus":                 types.StringValue("disk_bus"),
					"nic_model":                types.StringValue("nic_model"),
					"operating_system":         types.StringValue("os"),
					"operating_system_distro":  types.StringValue("os_distro"),
					"operating_system_version": types.StringValue("os_version"),
					"rescue_bus":               types.StringValue("rescue_bus"),
					"rescue_device":            types.StringValue("rescue_device"),
					"secure_boot":              types.BoolValue(true),
					"uefi":                     types.BoolValue(true),
					"video_model":              types.StringValue("video_model"),
					"virtio_scsi":              types.BoolValue(true),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaas.CreateImagePayload{
				Name:        utils.Ptr("name"),
				DiskFormat:  utils.Ptr("format"),
				MinDiskSize: utils.Ptr(int64(1)),
				MinRam:      utils.Ptr(int64(1)),
				Protected:   utils.Ptr(true),
				Config: &iaas.ImageConfig{
					BootMenu:               utils.Ptr(true),
					CdromBus:               iaas.NewNullableString(utils.Ptr("cdrom_bus")),
					DiskBus:                iaas.NewNullableString(utils.Ptr("disk_bus")),
					NicModel:               iaas.NewNullableString(utils.Ptr("nic_model")),
					OperatingSystem:        utils.Ptr("os"),
					OperatingSystemDistro:  iaas.NewNullableString(utils.Ptr("os_distro")),
					OperatingSystemVersion: iaas.NewNullableString(utils.Ptr("os_version")),
					RescueBus:              iaas.NewNullableString(utils.Ptr("rescue_bus")),
					RescueDevice:           iaas.NewNullableString(utils.Ptr("rescue_device")),
					SecureBoot:             utils.Ptr(true),
					Uefi:                   utils.Ptr(true),
					VideoModel:             iaas.NewNullableString(utils.Ptr("video_model")),
					VirtioScsi:             utils.Ptr(true),
				},
				Labels: &map[string]interface{}{
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
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
		expected    *iaas.UpdateImagePayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Id:          types.StringValue("pid,iid"),
				ProjectId:   types.StringValue("pid"),
				ImageId:     types.StringValue("iid"),
				Name:        types.StringValue("name"),
				DiskFormat:  types.StringValue("format"),
				MinDiskSize: types.Int64Value(1),
				MinRAM:      types.Int64Value(1),
				Protected:   types.BoolValue(true),
				Config: types.ObjectValueMust(configTypes, map[string]attr.Value{
					"boot_menu":                types.BoolValue(true),
					"cdrom_bus":                types.StringValue("cdrom_bus"),
					"disk_bus":                 types.StringValue("disk_bus"),
					"nic_model":                types.StringValue("nic_model"),
					"operating_system":         types.StringValue("os"),
					"operating_system_distro":  types.StringValue("os_distro"),
					"operating_system_version": types.StringValue("os_version"),
					"rescue_bus":               types.StringValue("rescue_bus"),
					"rescue_device":            types.StringValue("rescue_device"),
					"secure_boot":              types.BoolValue(true),
					"uefi":                     types.BoolValue(true),
					"video_model":              types.StringValue("video_model"),
					"virtio_scsi":              types.BoolValue(true),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaas.UpdateImagePayload{
				Name:        utils.Ptr("name"),
				MinDiskSize: utils.Ptr(int64(1)),
				MinRam:      utils.Ptr(int64(1)),
				Protected:   utils.Ptr(true),
				Config: &iaas.ImageConfig{
					BootMenu:               utils.Ptr(true),
					CdromBus:               iaas.NewNullableString(utils.Ptr("cdrom_bus")),
					DiskBus:                iaas.NewNullableString(utils.Ptr("disk_bus")),
					NicModel:               iaas.NewNullableString(utils.Ptr("nic_model")),
					OperatingSystem:        utils.Ptr("os"),
					OperatingSystemDistro:  iaas.NewNullableString(utils.Ptr("os_distro")),
					OperatingSystemVersion: iaas.NewNullableString(utils.Ptr("os_version")),
					RescueBus:              iaas.NewNullableString(utils.Ptr("rescue_bus")),
					RescueDevice:           iaas.NewNullableString(utils.Ptr("rescue_device")),
					SecureBoot:             utils.Ptr(true),
					Uefi:                   utils.Ptr(true),
					VideoModel:             iaas.NewNullableString(utils.Ptr("video_model")),
					VirtioScsi:             utils.Ptr(true),
				},
				Labels: &map[string]interface{}{
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func Test_UploadImage(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		uploadFails bool
		wantErr     bool
	}{
		{
			name:        "ok",
			filePath:    "testdata/mock-image.txt",
			uploadFails: false,
			wantErr:     false,
		},
		{
			name:        "upload_fails",
			filePath:    "testdata/mock-image.txt",
			uploadFails: true,
			wantErr:     true,
		},
		{
			name:        "file_not_found",
			filePath:    "testdata/non-existing-file.txt",
			uploadFails: false,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup a test server
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.uploadFails {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = fmt.Fprintln(w, `{"status":"some error occurred"}`)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintln(w, `{"status":"ok"}`)
			})
			server := httptest.NewServer(handler)
			defer server.Close()
			uploadURL, err := url.Parse(server.URL)
			if err != nil {
				t.Error(err)
				return
			}

			// Call the function
			err = uploadImage(context.Background(), &diag.Diagnostics{}, tt.filePath, uploadURL.String())
			if (err != nil) != tt.wantErr {
				t.Errorf("uploadImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
