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
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapFields(t *testing.T) {
	type args struct {
		state  Model
		input  *iaas.Image
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    Model
		isValid     bool
	}{
		{
			description: "default_values",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
					ImageId:   types.StringValue("iid"),
				},
				input: &iaas.Image{
					Id: new("iid"),
				},
				region: "eu01",
			},
			expected: Model{
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
				state: Model{
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
			expected: Model{
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
				state: Model{
					ProjectId: types.StringValue("pid"),
					ImageId:   types.StringValue("iid"),
					Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
				},
				input: &iaas.Image{
					Id: new("iid"),
				},
				region: "eu01",
			},
			expected: Model{
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
				state: Model{
					ProjectId: types.StringValue("pid"),
				},
				input: &iaas.Image{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.args.input, &tt.args.state, tt.args.region)
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
				Name:        new("name"),
				DiskFormat:  new("format"),
				MinDiskSize: new(int64(1)),
				MinRam:      new(int64(1)),
				Protected:   new(true),
				Config: &iaas.ImageConfig{
					BootMenu:               new(true),
					CdromBus:               iaas.NewNullableString(new("cdrom_bus")),
					DiskBus:                iaas.NewNullableString(new("disk_bus")),
					NicModel:               iaas.NewNullableString(new("nic_model")),
					OperatingSystem:        new("os"),
					OperatingSystemDistro:  iaas.NewNullableString(new("os_distro")),
					OperatingSystemVersion: iaas.NewNullableString(new("os_version")),
					RescueBus:              iaas.NewNullableString(new("rescue_bus")),
					RescueDevice:           iaas.NewNullableString(new("rescue_device")),
					SecureBoot:             new(true),
					Uefi:                   new(true),
					VideoModel:             iaas.NewNullableString(new("video_model")),
					VirtioScsi:             new(true),
				},
				Labels: &map[string]any{
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
				Name:        new("name"),
				MinDiskSize: new(int64(1)),
				MinRam:      new(int64(1)),
				Protected:   new(true),
				Config: &iaas.ImageConfig{
					BootMenu:               new(true),
					CdromBus:               iaas.NewNullableString(new("cdrom_bus")),
					DiskBus:                iaas.NewNullableString(new("disk_bus")),
					NicModel:               iaas.NewNullableString(new("nic_model")),
					OperatingSystem:        new("os"),
					OperatingSystemDistro:  iaas.NewNullableString(new("os_distro")),
					OperatingSystemVersion: iaas.NewNullableString(new("os_version")),
					RescueBus:              iaas.NewNullableString(new("rescue_bus")),
					RescueDevice:           iaas.NewNullableString(new("rescue_device")),
					SecureBoot:             new(true),
					Uefi:                   new(true),
					VideoModel:             iaas.NewNullableString(new("video_model")),
					VirtioScsi:             new(true),
				},
				Labels: &map[string]any{
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
