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
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaasalpha.Image
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
			},
			&iaasalpha.Image{
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
			Model{
				ProjectId: types.StringValue("pid"),
				ImageId:   types.StringValue("iid"),
				Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			&iaasalpha.Image{
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
			&iaasalpha.Image{},
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
		expected    *iaasalpha.CreateImagePayload
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
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaasalpha.CreateImagePayload{
				Name:        utils.Ptr("name"),
				DiskFormat:  utils.Ptr("format"),
				MinDiskSize: utils.Ptr(int64(1)),
				MinRam:      utils.Ptr(int64(1)),
				Protected:   utils.Ptr(true),
				Config: &iaasalpha.ImageConfig{
					BootMenu:               utils.Ptr(true),
					CdromBus:               iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					DiskBus:                iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					NicModel:               iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					OperatingSystem:        utils.Ptr("os"),
					OperatingSystemDistro:  iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					OperatingSystemVersion: iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					RescueBus:              iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					RescueDevice:           iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					SecureBoot:             utils.Ptr(true),
					Uefi:                   utils.Ptr(true),
					VideoModel:             iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaasalpha.NullableString{}))
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
		expected    *iaasalpha.UpdateImagePayload
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
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaasalpha.UpdateImagePayload{
				Name:        utils.Ptr("name"),
				MinDiskSize: utils.Ptr(int64(1)),
				MinRam:      utils.Ptr(int64(1)),
				Protected:   utils.Ptr(true),
				Config: &iaasalpha.ImageConfig{
					BootMenu:               utils.Ptr(true),
					CdromBus:               iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					DiskBus:                iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					NicModel:               iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					OperatingSystem:        utils.Ptr("os"),
					OperatingSystemDistro:  iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					OperatingSystemVersion: iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					RescueBus:              iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					RescueDevice:           iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
					SecureBoot:             utils.Ptr(true),
					Uefi:                   utils.Ptr(true),
					VideoModel:             iaasalpha.NewNullableString(nil), // TODO: Should take value after SDK util fix
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
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaasalpha.NullableString{}))
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
