package volume

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapFields(t *testing.T) {
	type args struct {
		state  Model
		input  *iaas.Volume
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
					VolumeId:  types.StringValue("nid"),
				},
				input: &iaas.Volume{
					Id:                   utils.Ptr("nid"),
					EncryptionParameters: nil,
				},
				region: "eu01",
			},
			expected: Model{
				Id:                   types.StringValue("pid,eu01,nid"),
				ProjectId:            types.StringValue("pid"),
				VolumeId:             types.StringValue("nid"),
				Name:                 types.StringNull(),
				AvailabilityZone:     types.StringNull(),
				Labels:               types.MapNull(types.StringType),
				Description:          types.StringNull(),
				PerformanceClass:     types.StringNull(),
				ServerId:             types.StringNull(),
				Size:                 types.Int64Null(),
				Source:               types.ObjectNull(sourceTypes),
				Region:               types.StringValue("eu01"),
				EncryptionParameters: nil,
			},
			isValid: true,
		},
		{
			description: "simple_values",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
					VolumeId:  types.StringValue("nid"),
					Region:    types.StringValue("eu01"),
					EncryptionParameters: &encryptionParametersModel{
						KekKeyId:         types.StringValue("kek-key-id"),
						KekKeyVersion:    types.Int64Value(int64(1)),
						KekKeyringId:     types.StringValue("kek-keyring-id"),
						KeyPayloadBase64: types.StringValue("cm91dGVkb3VidGV2ZXJvdmVyY2xhc3Nkcml2aW5ndGhpbmdmbGFtZWNyb3dkcXVpY2s="),
						ServiceAccount:   types.StringValue("test-sa@sa.stackit.cloud"),
					},
				},
				input: &iaas.Volume{
					Id:               utils.Ptr("nid"),
					Name:             utils.Ptr("name"),
					AvailabilityZone: utils.Ptr("zone"),
					Labels: &map[string]interface{}{
						"key": "value",
					},
					Description:      utils.Ptr("desc"),
					PerformanceClass: utils.Ptr("class"),
					ServerId:         utils.Ptr("sid"),
					Size:             utils.Ptr(int64(1)),
					Source:           &iaas.VolumeSource{},
					Encrypted:        utils.Ptr(true),
					EncryptionParameters: &iaas.VolumeEncryptionParameter{
						KekKeyId:       utils.Ptr("kek-key-id"),
						KekKeyVersion:  utils.Ptr(int64(1)),
						KekKeyringId:   utils.Ptr("kek-keyring-id"),
						KekProjectId:   utils.Ptr("kek-project-id"),
						KeyPayload:     nil,
						ServiceAccount: utils.Ptr("test-sa@sa.stackit.cloud"),
					},
				},
				region: "eu02",
			},
			expected: Model{
				Id:               types.StringValue("pid,eu02,nid"),
				ProjectId:        types.StringValue("pid"),
				VolumeId:         types.StringValue("nid"),
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description:      types.StringValue("desc"),
				PerformanceClass: types.StringValue("class"),
				ServerId:         types.StringValue("sid"),
				Size:             types.Int64Value(1),
				Source: types.ObjectValueMust(sourceTypes, map[string]attr.Value{
					"type": types.StringNull(),
					"id":   types.StringNull(),
				}),
				Region:    types.StringValue("eu02"),
				Encrypted: types.BoolValue(true),
				EncryptionParameters: &encryptionParametersModel{
					KekKeyId:         types.StringValue("kek-key-id"),
					KekKeyVersion:    types.Int64Value(int64(1)),
					KekKeyringId:     types.StringValue("kek-keyring-id"),
					KeyPayloadBase64: types.StringValue("cm91dGVkb3VidGV2ZXJvdmVyY2xhc3Nkcml2aW5ndGhpbmdmbGFtZWNyb3dkcXVpY2s="),
					ServiceAccount:   types.StringValue("test-sa@sa.stackit.cloud"),
				},
			},
			isValid: true,
		},
		{
			description: "empty labels",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
					VolumeId:  types.StringValue("nid"),
					Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
				},
				input: &iaas.Volume{
					Id: utils.Ptr("nid"),
				},
				region: "eu01",
			},
			expected: Model{
				Id:               types.StringValue("pid,eu01,nid"),
				ProjectId:        types.StringValue("pid"),
				VolumeId:         types.StringValue("nid"),
				Name:             types.StringNull(),
				AvailabilityZone: types.StringNull(),
				Labels:           types.MapValueMust(types.StringType, map[string]attr.Value{}),
				Description:      types.StringNull(),
				PerformanceClass: types.StringNull(),
				ServerId:         types.StringNull(),
				Size:             types.Int64Null(),
				Source:           types.ObjectNull(sourceTypes),
				Region:           types.StringValue("eu01"),
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
				input: &iaas.Volume{},
			},
			expected: Model{},
			isValid:  false,
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
		source      *sourceModel
		expected    *iaas.CreateVolumePayload
		isValid     bool
	}{
		{
			description: "no volume encryption",
			input: &Model{
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description:      types.StringValue("desc"),
				PerformanceClass: types.StringValue("class"),
				Size:             types.Int64Value(1),
				Source: types.ObjectValueMust(sourceTypes, map[string]attr.Value{
					"type": types.StringNull(),
					"id":   types.StringNull(),
				}),
			},
			source: &sourceModel{
				Type: types.StringValue("volume"),
				Id:   types.StringValue("id"),
			},
			expected: &iaas.CreateVolumePayload{
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Description:      utils.Ptr("desc"),
				PerformanceClass: utils.Ptr("class"),
				Size:             utils.Ptr(int64(1)),
				Source: &iaas.VolumeSource{
					Type: utils.Ptr("volume"),
					Id:   utils.Ptr("id"),
				},
			},
			isValid: true,
		},
		{
			description: "with volume encryption without key payload",
			input: &Model{
				Labels: types.MapNull(types.StringType),
				EncryptionParameters: &encryptionParametersModel{
					KekKeyId:         types.StringValue("kek-key-id"),
					KekKeyVersion:    types.Int64Value(int64(1)),
					KekKeyringId:     types.StringValue("kek-keyring-id"),
					KeyPayloadBase64: types.StringNull(),
					ServiceAccount:   types.StringValue("test-sa@sa.stackit.cloud"),
				},
			},
			source: &sourceModel{
				Type: types.StringValue("volume"),
				Id:   types.StringValue("id"),
			},
			expected: &iaas.CreateVolumePayload{
				Source: &iaas.VolumeSource{
					Type: utils.Ptr("volume"),
					Id:   utils.Ptr("id"),
				},
				Labels: &map[string]interface{}{},
				EncryptionParameters: &iaas.VolumeEncryptionParameter{
					KekKeyId:       utils.Ptr("kek-key-id"),
					KekKeyVersion:  utils.Ptr(int64(1)),
					KekKeyringId:   utils.Ptr("kek-keyring-id"),
					KekProjectId:   nil,
					KeyPayload:     nil,
					ServiceAccount: utils.Ptr("test-sa@sa.stackit.cloud"),
				},
			},
			isValid: true,
		},
		{
			description: "with volume encryption including key payload",
			input: &Model{
				Labels: types.MapNull(types.StringType),
				EncryptionParameters: &encryptionParametersModel{
					KekKeyId:         types.StringValue("kek-key-id"),
					KekKeyVersion:    types.Int64Value(int64(1)),
					KekKeyringId:     types.StringValue("kek-keyring-id"),
					KeyPayloadBase64: types.StringValue("VGhlIHF1aWNrIGJyb3duIGZveCBqdW1wcyBvdmVyIDEzIGxhenkgZG9ncy4="), // The quick brown fox jumps over 13 lazy dogs.
					ServiceAccount:   types.StringValue("test-sa@sa.stackit.cloud"),
				},
			},
			source: &sourceModel{
				Type: types.StringValue("volume"),
				Id:   types.StringValue("id"),
			},
			expected: &iaas.CreateVolumePayload{
				Source: &iaas.VolumeSource{
					Type: utils.Ptr("volume"),
					Id:   utils.Ptr("id"),
				},
				Labels: &map[string]interface{}{},
				EncryptionParameters: &iaas.VolumeEncryptionParameter{
					KekKeyId:      utils.Ptr("kek-key-id"),
					KekKeyVersion: utils.Ptr(int64(1)),
					KekKeyringId:  utils.Ptr("kek-keyring-id"),
					KekProjectId:  nil,
					KeyPayload: func() *[]byte {
						keyPayload := []byte{
							0x56, 0x47, 0x68, 0x6c, 0x49, 0x48, 0x46, 0x31, 0x61, 0x57, 0x4e, 0x72, 0x49, 0x47, 0x4a,
							0x79, 0x62, 0x33, 0x64, 0x75, 0x49, 0x47, 0x5a, 0x76, 0x65, 0x43, 0x42, 0x71, 0x64, 0x57,
							0x31, 0x77, 0x63, 0x79, 0x42, 0x76, 0x64, 0x6d, 0x56, 0x79, 0x49, 0x44, 0x45, 0x7a, 0x49,
							0x47, 0x78, 0x68, 0x65, 0x6e, 0x6b, 0x67, 0x5a, 0x47, 0x39, 0x6e, 0x63, 0x79, 0x34, 0x3d,
						}
						return &keyPayload
					}(),
					ServiceAccount: utils.Ptr("test-sa@sa.stackit.cloud"),
				},
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input, tt.source)
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
		description string
		input       *Model
		expected    *iaas.UpdateVolumePayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description: types.StringValue("desc"),
			},
			&iaas.UpdateVolumePayload{
				Name: utils.Ptr("name"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Description: utils.Ptr("desc"),
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
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
