package image

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
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
				Id:         types.StringValue("pid,eu01,iid"),
				ProjectId:  types.StringValue("pid"),
				ImageId:    types.StringValue("iid"),
				Labels:     types.MapNull(types.StringType),
				Region:     types.StringValue("eu01"),
				Name:       types.StringValue(""),
				DiskFormat: types.StringValue(""),
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
					Name:        "name",
					DiskFormat:  "format",
					MinDiskSize: new(int64(1)),
					MinRam:      new(int64(1)),
					Protected:   new(true),
					Scope:       new("scope"),
					Config: &iaas.ImageConfig{
						BootMenu:               new(true),
						CdromBus:               *iaas.NewNullableString(new("cdrom_bus")),
						DiskBus:                *iaas.NewNullableString(new("disk_bus")),
						NicModel:               *iaas.NewNullableString(new("model")),
						OperatingSystem:        new("os"),
						OperatingSystemDistro:  *iaas.NewNullableString(new("os_distro")),
						OperatingSystemVersion: *iaas.NewNullableString(new("os_version")),
						RescueBus:              *iaas.NewNullableString(new("rescue_bus")),
						RescueDevice:           *iaas.NewNullableString(new("rescue_device")),
						SecureBoot:             new(true),
						Uefi:                   new(true),
						VideoModel:             *iaas.NewNullableString(new("model")),
						VirtioScsi:             new(true),
					},
					Checksum: &iaas.ImageChecksum{
						Algorithm: "algorithm",
						Digest:    "digest",
					},
					Labels: map[string]any{
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
				Id:         types.StringValue("pid,eu01,iid"),
				ProjectId:  types.StringValue("pid"),
				ImageId:    types.StringValue("iid"),
				Labels:     types.MapValueMust(types.StringType, map[string]attr.Value{}),
				Region:     types.StringValue("eu01"),
				Name:       types.StringValue(""),
				DiskFormat: types.StringValue(""),
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
				Name:        "name",
				DiskFormat:  "format",
				MinDiskSize: new(int64(1)),
				MinRam:      new(int64(1)),
				Protected:   new(true),
				Config: &iaas.ImageConfig{
					BootMenu:               new(true),
					CdromBus:               *iaas.NewNullableString(new("cdrom_bus")),
					DiskBus:                *iaas.NewNullableString(new("disk_bus")),
					NicModel:               *iaas.NewNullableString(new("nic_model")),
					OperatingSystem:        new("os"),
					OperatingSystemDistro:  *iaas.NewNullableString(new("os_distro")),
					OperatingSystemVersion: *iaas.NewNullableString(new("os_version")),
					RescueBus:              *iaas.NewNullableString(new("rescue_bus")),
					RescueDevice:           *iaas.NewNullableString(new("rescue_device")),
					SecureBoot:             new(true),
					Uefi:                   new(true),
					VideoModel:             *iaas.NewNullableString(new("video_model")),
					VirtioScsi:             new(true),
				},
				Labels: map[string]any{
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
					CdromBus:               *iaas.NewNullableString(new("cdrom_bus")),
					DiskBus:                *iaas.NewNullableString(new("disk_bus")),
					NicModel:               *iaas.NewNullableString(new("nic_model")),
					OperatingSystem:        new("os"),
					OperatingSystemDistro:  *iaas.NewNullableString(new("os_distro")),
					OperatingSystemVersion: *iaas.NewNullableString(new("os_version")),
					RescueBus:              *iaas.NewNullableString(new("rescue_bus")),
					RescueDevice:           *iaas.NewNullableString(new("rescue_device")),
					SecureBoot:             new(true),
					Uefi:                   new(true),
					VideoModel:             *iaas.NewNullableString(new("video_model")),
					VirtioScsi:             new(true),
				},
				Labels: map[string]any{
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
			err = uploadImage(context.Background(), tt.filePath, uploadURL.String())
			if (err != nil) != tt.wantErr {
				t.Errorf("uploadImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_DownloadImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/404":
			w.WriteHeader(http.StatusNotFound)
		case "/empty":
			w.WriteHeader(http.StatusOK)
		case "/drop-conn":
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Logf("Failed to hijack connection: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := conn.Close(); err != nil {
				t.Logf("Failed to close hijacked connection: %v", err)
			}

		default:
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("dummy content")); err != nil {
				t.Logf("Failed to write mock response payload: %v", err)
			}
		}
	}))
	t.Cleanup(server.Close)

	tests := []struct {
		name        string
		downloadURL string
		wantBytes   []byte
		wantErr     bool
	}{{
		name:        "ok",
		downloadURL: server.URL,
		wantErr:     false,
	},
		{
			name:        "invalid_url_format",
			downloadURL: "http://127.0.0.1:0/invalid",
			wantErr:     true,
		},
		{
			name:        "status_404_not_found",
			downloadURL: server.URL + "/404",
			wantErr:     true,
		},
		{
			name:        "empty_body_200_ok",
			downloadURL: server.URL + "/empty",
			wantErr:     false,
		},
		{
			name:        "connection_dropped_mid_stream",
			downloadURL: server.URL + "/drop-conn",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath, err := downloadImage(context.Background(), tt.downloadURL)

			if filePath != "" {
				t.Cleanup(func() {
					if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
						t.Logf("failed to cleanup temporary file %s: %v", filePath, err)
					}
				})
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("downloadImage() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				_, err := os.Stat(filePath)
				if os.IsNotExist(err) {
					t.Fatalf("downloadImage() returned path %q, but file does not exist on disk", filePath)
				} else if err != nil {
					t.Fatalf("os.Stat(%q) error = %v", filePath, err)
				}
				if !strings.HasSuffix(filePath, ".img") {
					t.Errorf("filePath %q does not end with %q", filePath, ".img")
				}
			}
		})
	}
}

var localTypes = map[string]attr.Type{
	"file_path":               basetypes.StringType{},
	"disable_plan_validation": basetypes.BoolType{},
}

var downloadTypes = map[string]attr.Type{
	"url": basetypes.StringType{},
}

var imageFileTypes = map[string]attr.Type{
	"local":    basetypes.ObjectType{AttrTypes: localTypes},
	"download": basetypes.ObjectType{AttrTypes: downloadTypes},
}

func Test_ResolveFilePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	tests := []struct {
		description          string
		input                *Model
		expectedString       string
		expectedBool         bool
		expectedErrSubstring string
		isValid              bool
	}{
		{
			"legacy ok",
			&Model{
				ProjectId:     types.StringValue("pid"),
				Name:          types.StringValue("name"),
				DiskFormat:    types.StringValue("format"),
				LocalFilePath: types.StringValue("./mock-file"),
				ImageFile: types.ObjectValueMust(
					imageFileTypes,
					map[string]attr.Value{
						"download": types.ObjectNull(downloadTypes),
						"local":    types.ObjectNull(localTypes),
					},
				),
			},
			"./mock-file",
			false,
			"",
			true,
		},

		{
			"parsing image file block error",
			&Model{
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				DiskFormat: types.StringValue("format"),
				ImageFile: types.ObjectValueMust(
					map[string]attr.Type{
						"download": types.Int64Type,
						"local":    types.StringType,
					},
					map[string]attr.Value{
						"download": types.Int64Value(123),
						"local":    types.StringValue("invalid"),
					},
				),
			},
			"",
			false,
			"parsing image_file block",
			false,
		},
		{
			"parsing download block error",
			&Model{
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				DiskFormat: types.StringValue("format"),
				ImageFile: types.ObjectValueMust(
					map[string]attr.Type{
						"local": basetypes.ObjectType{AttrTypes: localTypes},
						"download": basetypes.ObjectType{AttrTypes: map[string]attr.Type{
							"url": basetypes.Int64Type{},
						}},
					},
					map[string]attr.Value{
						"download": types.ObjectValueMust(
							map[string]attr.Type{
								"url": types.Int64Type,
							},
							map[string]attr.Value{
								"url": types.Int64Value(404),
							},
						),
						"local": types.ObjectNull(localTypes),
					},
				),
			},
			"",
			false,
			"parsing download block",
			false,
		},
		{
			"parsing local block error",
			&Model{
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				DiskFormat: types.StringValue("format"),
				ImageFile: types.ObjectValueMust(
					map[string]attr.Type{
						"local": basetypes.ObjectType{AttrTypes: map[string]attr.Type{
							"file_path":               basetypes.Int64Type{},
							"disable_plan_validation": basetypes.BoolType{},
						}},
						"download": basetypes.ObjectType{AttrTypes: downloadTypes},
					},
					map[string]attr.Value{
						"download": types.ObjectNull(downloadTypes),
						"local": types.ObjectValueMust(
							map[string]attr.Type{
								"file_path":               types.Int64Type,
								"disable_plan_validation": types.BoolType,
							},
							map[string]attr.Value{
								"file_path":               types.Int64Value(123),
								"disable_plan_validation": types.BoolValue(false),
							},
						),
					},
				),
			},
			"",
			false,
			"parsing local block",
			false,
		},
		{
			"local ok",
			&Model{
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				DiskFormat: types.StringValue("format"),
				ImageFile: types.ObjectValueMust(
					imageFileTypes,
					map[string]attr.Value{
						"download": types.ObjectNull(downloadTypes),
						"local": types.ObjectValueMust(
							localTypes,
							map[string]attr.Value{
								"file_path":               types.StringValue("./mock-file"),
								"disable_plan_validation": types.BoolValue(false),
							},
						),
					},
				),
			},
			"./mock-file",
			false,
			"",
			true,
		},
		{
			"download ok",
			&Model{
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				DiskFormat: types.StringValue("format"),
				ImageFile: types.ObjectValueMust(
					imageFileTypes,
					map[string]attr.Value{
						"download": types.ObjectValueMust(downloadTypes,
							map[string]attr.Value{
								"url": types.StringValue(server.URL + "/mock-file"),
							}),
						"local": types.ObjectNull(localTypes),
					},
				),
			},
			".img",
			true,
			"",
			true,
		},
		{
			"download error",
			&Model{
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				DiskFormat: types.StringValue("format"),
				ImageFile: types.ObjectValueMust(
					imageFileTypes,
					map[string]attr.Value{
						"download": types.ObjectValueMust(downloadTypes,
							map[string]attr.Value{
								"url": types.StringValue(""),
							}),
						"local": types.ObjectNull(localTypes),
					},
				),
			},
			"",
			false,
			"downloading image",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			outputString, outputBool, err := resolveFilePath(context.Background(), tt.input)

			if !tt.isValid {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedErrSubstring)
				}
				if tt.expectedErrSubstring != "" && !strings.Contains(err.Error(), tt.expectedErrSubstring) {
					t.Fatalf("unexpected error message:\n got: %v\n want substring: %q", err, tt.expectedErrSubstring)
				}
				return // Skip success
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.HasSuffix(outputString, tt.expectedString) {
				t.Fatalf("output mismatch:\n got: %s and %t\n want suffix: %s and %t", outputString, outputBool, tt.expectedString, tt.expectedBool)
			}
		})
	}
}
