// Copyright (c) STACKIT

package sqlserverflex

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	sqlserverflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

// type sqlserverflexClientMocked struct {
//	returnError     bool
//	listFlavorsResp *sqlserverflex.GetFlavorsResponse
// }
//
// func (c *sqlserverflexClientMocked) GetFlavorsExecute(_ context.Context, _, _ string) (*sqlserverflex.GetFlavorsResponse, error) {
//	if c.returnError {
//		return nil, fmt.Errorf("get flavors failed")
//	}
//
//	return c.listFlavorsResp, nil
// }

func TestMapFields(t *testing.T) {
	t.Skip("Skipping - needs refactoring")
	const testRegion = "region"
	tests := []struct {
		description string
		state       Model
		input       *sqlserverflex.GetInstanceResponse
		flavor      *flavorModel
		storage     *storageModel
		encryption  *encryptionModel
		network     *networkModel
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				InstanceId:    types.StringValue("iid"),
				ProjectId:     types.StringValue("pid"),
				Replicas:      types.Int64Value(1),
				RetentionDays: types.Int64Value(1),
				Version:       types.StringValue("v1"),
				Edition:       types.StringValue("edition 1"),
				Status:        types.StringValue("status"),
				IsDeletable:   types.BoolValue(true),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringValue("flavor_id"),
					"description": types.StringNull(),
					"cpu":         types.Int64Null(),
					"ram":         types.Int64Null(),
					"node_type":   types.StringNull(),
				}),
			},
			&sqlserverflex.GetInstanceResponse{
				FlavorId:      utils.Ptr("flavor_id"),
				Replicas:      sqlserverflex.GetInstanceResponseGetReplicasAttributeType(utils.Ptr(int32(1))),
				RetentionDays: utils.Ptr(int64(1)),
				Version:       sqlserverflex.GetInstanceResponseGetVersionAttributeType(utils.Ptr("v1")),
				Edition:       sqlserverflex.GetInstanceResponseGetEditionAttributeType(utils.Ptr("edition 1")),
				Status:        sqlserverflex.GetInstanceResponseGetStatusAttributeType(utils.Ptr("status")),
				IsDeletable:   utils.Ptr(true),
			},
			&flavorModel{
				Id: types.StringValue("flavor_id"),
			},
			&storageModel{},
			&encryptionModel{},
			&networkModel{
				ACL: types.ListNull(basetypes.StringType{}),
			},
			testRegion,
			Model{
				Id:             types.StringValue("pid,region,iid"),
				InstanceId:     types.StringValue("iid"),
				ProjectId:      types.StringValue("pid"),
				Name:           types.StringNull(),
				BackupSchedule: types.StringNull(),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringValue("flavor_id"),
					"description": types.StringNull(),
					"cpu":         types.Int64Null(),
					"ram":         types.Int64Null(),
					"node_type":   types.StringNull(),
				}),
				Replicas: types.Int64Value(1),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringNull(),
					"size":  types.Int64Null(),
				}),
				Encryption: types.ObjectValueMust(encryptionTypes, map[string]attr.Value{
					"keyring_id":      types.StringNull(),
					"key_id":          types.StringNull(),
					"key_version":     types.StringNull(),
					"service_account": types.StringNull(),
				}),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":              types.ListNull(types.StringType),
					"access_scope":     types.StringNull(),
					"instance_address": types.StringNull(),
					"router_address":   types.StringNull(),
				}),
				IsDeletable:   types.BoolValue(true),
				Edition:       types.StringValue("edition 1"),
				Status:        types.StringValue("status"),
				RetentionDays: types.Int64Value(1),
				Version:       types.StringValue("v1"),
				Region:        types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&sqlserverflex.GetInstanceResponse{
				BackupSchedule: utils.Ptr("schedule"),
				FlavorId:       utils.Ptr("flavor_id"),
				Id:             utils.Ptr("iid"),
				Name:           utils.Ptr("name"),
				Replicas:       sqlserverflex.GetInstanceResponseGetReplicasAttributeType(utils.Ptr(int32(56))),
				Status:         sqlserverflex.GetInstanceResponseGetStatusAttributeType(utils.Ptr("status")),
				Storage: &sqlserverflex.Storage{
					Class: utils.Ptr("class"),
					Size:  utils.Ptr(int64(78)),
				},
				Edition:       sqlserverflex.GetInstanceResponseGetEditionAttributeType(utils.Ptr("edition")),
				RetentionDays: utils.Ptr(int64(1)),
				Version:       sqlserverflex.GetInstanceResponseGetVersionAttributeType(utils.Ptr("version")),
				IsDeletable:   utils.Ptr(true),
				Encryption:    nil,
				Network: &sqlserverflex.InstanceNetwork{
					AccessScope: nil,
					Acl: &[]string{
						"ip1",
						"ip2",
						"",
					},
					InstanceAddress: nil,
					RouterAddress:   nil,
				},
			},
			&flavorModel{
				Id:          basetypes.NewStringValue("flavor_id"),
				Description: basetypes.NewStringValue("description"),
				CPU:         basetypes.NewInt64Value(12),
				RAM:         basetypes.NewInt64Value(34),
				NodeType:    basetypes.NewStringValue("node_type"),
			},
			&storageModel{},
			&encryptionModel{},
			&networkModel{
				ACL: types.ListValueMust(basetypes.StringType{}, []attr.Value{
					types.StringValue("ip1"),
					types.StringValue("ip2"),
					types.StringValue(""),
				}),
			},
			testRegion,
			Model{
				Id:             types.StringValue("pid,region,iid"),
				InstanceId:     types.StringValue("iid"),
				ProjectId:      types.StringValue("pid"),
				Name:           types.StringValue("name"),
				BackupSchedule: types.StringValue("schedule"),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringValue("flavor_id"),
					"description": types.StringValue("description"),
					"cpu":         types.Int64Value(12),
					"ram":         types.Int64Value(34),
					"node_type":   types.StringValue("node_type"),
				}),
				Replicas: types.Int64Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("ip1"),
						types.StringValue("ip2"),
						types.StringValue(""),
					}),
					"access_scope":     types.StringNull(),
					"instance_address": types.StringNull(),
					"router_address":   types.StringNull(),
				}),
				Edition:       types.StringValue("edition"),
				RetentionDays: types.Int64Value(1),
				Version:       types.StringValue("version"),
				Region:        types.StringValue(testRegion),
				IsDeletable:   types.BoolValue(true),
				Encryption: types.ObjectValueMust(encryptionTypes, map[string]attr.Value{
					"keyring_id":      types.StringNull(),
					"key_id":          types.StringNull(),
					"key_version":     types.StringNull(),
					"service_account": types.StringNull(),
				}),
				Status: types.StringValue("status"),
			},
			true,
		},
		// {
		//	"simple_values_no_flavor_and_storage",
		//	Model{
		//		InstanceId: types.StringValue("iid"),
		//		ProjectId:  types.StringValue("pid"),
		//	},
		//	&sqlserverflex.GetInstanceResponse{
		//		Acl: &[]string{
		//			"ip1",
		//			"ip2",
		//			"",
		//		},
		//		BackupSchedule: utils.Ptr("schedule"),
		//		FlavorId:       nil,
		//		Id:             utils.Ptr("iid"),
		//		Name:           utils.Ptr("name"),
		//		Replicas:       sqlserverflex.GetInstanceResponseGetReplicasAttributeType(utils.Ptr(int32(56))),
		//		Status:         sqlserverflex.GetInstanceResponseGetStatusAttributeType(utils.Ptr("status")),
		//		Storage:        nil,
		//		Edition:        sqlserverflex.GetInstanceResponseGetEditionAttributeType(utils.Ptr("edition")),
		//		RetentionDays:  utils.Ptr(int64(1)),
		//		Version:        sqlserverflex.GetInstanceResponseGetVersionAttributeType(utils.Ptr("version")),
		//	},
		//	&flavorModel{
		//		CPU: types.Int64Value(12),
		//		RAM: types.Int64Value(34),
		//	},
		//	&storageModel{
		//		Class: types.StringValue("class"),
		//		Size:  types.Int64Value(78),
		//	},
		//	&optionsModel{
		//		Edition:       types.StringValue("edition"),
		//		RetentionDays: types.Int64Value(1),
		//	},
		//	testRegion,
		//	Model{
		//		Id:         types.StringValue("pid,region,iid"),
		//		InstanceId: types.StringValue("iid"),
		//		ProjectId:  types.StringValue("pid"),
		//		Name:       types.StringValue("name"),
		//		ACL: types.ListValueMust(types.StringType, []attr.Value{
		//			types.StringValue("ip1"),
		//			types.StringValue("ip2"),
		//			types.StringValue(""),
		//		}),
		//		BackupSchedule: types.StringValue("schedule"),
		//		Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
		//			"id":          types.StringNull(),
		//			"description": types.StringNull(),
		//			"cpu":         types.Int64Value(12),
		//			"ram":         types.Int64Value(34),
		//		}),
		//		Replicas: types.Int64Value(56),
		//		Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
		//			"class": types.StringValue("class"),
		//			"size":  types.Int64Value(78),
		//		}),
		//		Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
		//			"edition":        types.StringValue("edition"),
		//			"retention_days": types.Int64Value(1),
		//		}),
		//		Version: types.StringValue("version"),
		//		Region:  types.StringValue(testRegion),
		//	},
		//	true,
		// },
		// {
		//	"acls_unordered",
		//	Model{
		//		InstanceId: types.StringValue("iid"),
		//		ProjectId:  types.StringValue("pid"),
		//		ACL: types.ListValueMust(types.StringType, []attr.Value{
		//			types.StringValue("ip2"),
		//			types.StringValue(""),
		//			types.StringValue("ip1"),
		//		}),
		//	},
		//	&sqlserverflex.GetInstanceResponse{
		//		Acl: &[]string{
		//			"",
		//			"ip1",
		//			"ip2",
		//		},
		//		BackupSchedule: utils.Ptr("schedule"),
		//		FlavorId:       nil,
		//		Id:             utils.Ptr("iid"),
		//		Name:           utils.Ptr("name"),
		//		Replicas:       sqlserverflex.GetInstanceResponseGetReplicasAttributeType(utils.Ptr(int32(56))),
		//		Status:         sqlserverflex.GetInstanceResponseGetStatusAttributeType(utils.Ptr("status")),
		//		Storage:        nil,
		//		//Options: &map[string]string{
		//		//	"edition":       "edition",
		//		//	"retentionDays": "1",
		//		//},
		//		Version: sqlserverflex.GetInstanceResponseGetVersionAttributeType(utils.Ptr("version")),
		//	},
		//	&flavorModel{
		//		CPU: types.Int64Value(12),
		//		RAM: types.Int64Value(34),
		//	},
		//	&storageModel{
		//		Class: types.StringValue("class"),
		//		Size:  types.Int64Value(78),
		//	},
		//	&optionsModel{},
		//	testRegion,
		//	Model{
		//		Id:         types.StringValue("pid,region,iid"),
		//		InstanceId: types.StringValue("iid"),
		//		ProjectId:  types.StringValue("pid"),
		//		Name:       types.StringValue("name"),
		//		ACL: types.ListValueMust(types.StringType, []attr.Value{
		//			types.StringValue("ip2"),
		//			types.StringValue(""),
		//			types.StringValue("ip1"),
		//		}),
		//		BackupSchedule: types.StringValue("schedule"),
		//		Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
		//			"id":          types.StringNull(),
		//			"description": types.StringNull(),
		//			"cpu":         types.Int64Value(12),
		//			"ram":         types.Int64Value(34),
		//		}),
		//		Replicas: types.Int64Value(56),
		//		Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
		//			"class": types.StringValue("class"),
		//			"size":  types.Int64Value(78),
		//		}),
		//		Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
		//			"edition":        types.StringValue("edition"),
		//			"retention_days": types.Int64Value(1),
		//		}),
		//		Version: types.StringValue("version"),
		//		Region:  types.StringValue(testRegion),
		//	},
		//	true,
		// },
		// {
		//	"nil_response",
		//	Model{
		//		InstanceId: types.StringValue("iid"),
		//		ProjectId:  types.StringValue("pid"),
		//	},
		//	nil,
		//	&flavorModel{},
		//	&storageModel{},
		//	&optionsModel{},
		//	testRegion,
		//	Model{},
		//	false,
		// },
		// {
		//	"no_resource_id",
		//	Model{
		//		InstanceId: types.StringValue("iid"),
		//		ProjectId:  types.StringValue("pid"),
		//	},
		//	&sqlserverflex.GetInstanceResponse{},
		//	&flavorModel{},
		//	&storageModel{},
		//	&optionsModel{},
		//	testRegion,
		//	Model{},
		//	false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, tt.flavor, tt.storage, tt.encryption, tt.network, tt.region)
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

// func TestToCreatePayload(t *testing.T) {
//	tests := []struct {
//		description  string
//		input        *Model
//		inputAcl     []string
//		inputFlavor  *flavorModel
//		inputStorage *storageModel
//		inputOptions *optionsModel
//		expected     *sqlserverflex.CreateInstanceRequestPayload
//		isValid      bool
//	}{
//		{
//			"default_values",
//			&Model{},
//			[]string{},
//			&flavorModel{},
//			&storageModel{},
//			&optionsModel{},
//			&sqlserverflex.CreateInstanceRequestPayload{
//				Acl:     &sqlserverflex.CreateInstanceRequestPayloadGetAclArgType{},
//				Storage: &sqlserverflex.CreateInstanceRequestPayloadGetStorageArgType{},
//			},
//			true,
//		},
//		{
//			"simple_values",
//			&Model{
//				BackupSchedule: types.StringValue("schedule"),
//				Name:           types.StringValue("name"),
//				Replicas:       types.Int64Value(12),
//				Version:        types.StringValue("version"),
//			},
//			[]string{
//				"ip_1",
//				"ip_2",
//			},
//			&flavorModel{
//				Id: types.StringValue("flavor_id"),
//			},
//			&storageModel{
//				Class: types.StringValue("class"),
//				Size:  types.Int64Value(34),
//			},
//			&optionsModel{
//				Edition:       types.StringValue("edition"),
//				RetentionDays: types.Int64Value(1),
//			},
//			&sqlserverflex.CreateInstancePayload{
//				Acl: &sqlserverflex.CreateInstancePayloadAcl{
//					Items: &[]string{
//						"ip_1",
//						"ip_2",
//					},
//				},
//				BackupSchedule: utils.Ptr("schedule"),
//				FlavorId:       utils.Ptr("flavor_id"),
//				Name:           utils.Ptr("name"),
//				Storage: &sqlserverflex.CreateInstancePayloadStorage{
//					Class: utils.Ptr("class"),
//					Size:  utils.Ptr(int64(34)),
//				},
//				Options: &sqlserverflex.CreateInstancePayloadOptions{
//					Edition:       utils.Ptr("edition"),
//					RetentionDays: utils.Ptr("1"),
//				},
//				Version: utils.Ptr("version"),
//			},
//			true,
//		},
//		{
//			"null_fields_and_int_conversions",
//			&Model{
//				BackupSchedule: types.StringNull(),
//				Name:           types.StringNull(),
//				Replicas:       types.Int64Value(2123456789),
//				Version:        types.StringNull(),
//			},
//			[]string{
//				"",
//			},
//			&flavorModel{
//				Id: types.StringNull(),
//			},
//			&storageModel{
//				Class: types.StringNull(),
//				Size:  types.Int64Null(),
//			},
//			&optionsModel{
//				Edition:       types.StringNull(),
//				RetentionDays: types.Int64Null(),
//			},
//			&sqlserverflex.CreateInstancePayload{
//				Acl: &sqlserverflex.CreateInstancePayloadAcl{
//					Items: &[]string{
//						"",
//					},
//				},
//				BackupSchedule: nil,
//				FlavorId:       nil,
//				Name:           nil,
//				Storage: &sqlserverflex.CreateInstancePayloadStorage{
//					Class: nil,
//					Size:  nil,
//				},
//				Options: &sqlserverflex.CreateInstancePayloadOptions{},
//				Version: nil,
//			},
//			true,
//		},
//		{
//			"nil_model",
//			nil,
//			[]string{},
//			&flavorModel{},
//			&storageModel{},
//			&optionsModel{},
//			nil,
//			false,
//		},
//		{
//			"nil_acl",
//			&Model{},
//			nil,
//			&flavorModel{},
//			&storageModel{},
//			&optionsModel{},
//			&sqlserverflex.CreateInstancePayload{
//				Acl:     &sqlserverflex.CreateInstancePayloadAcl{},
//				Storage: &sqlserverflex.CreateInstancePayloadStorage{},
//				Options: &sqlserverflex.CreateInstancePayloadOptions{},
//			},
//			true,
//		},
//		{
//			"nil_flavor",
//			&Model{},
//			[]string{},
//			nil,
//			&storageModel{},
//			&optionsModel{},
//			nil,
//			false,
//		},
//		{
//			"nil_storage",
//			&Model{},
//			[]string{},
//			&flavorModel{},
//			nil,
//			&optionsModel{},
//			&sqlserverflex.CreateInstancePayload{
//				Acl: &sqlserverflex.CreateInstancePayloadAcl{
//					Items: &[]string{},
//				},
//				Storage: &sqlserverflex.CreateInstancePayloadStorage{},
//				Options: &sqlserverflex.CreateInstancePayloadOptions{},
//			},
//			true,
//		},
//		{
//			"nil_options",
//			&Model{},
//			[]string{},
//			&flavorModel{},
//			&storageModel{},
//			nil,
//			&sqlserverflex.CreateInstancePayload{
//				Acl: &sqlserverflex.CreateInstancePayloadAcl{
//					Items: &[]string{},
//				},
//				Storage: &sqlserverflex.CreateInstancePayloadStorage{},
//				Options: &sqlserverflex.CreateInstancePayloadOptions{},
//			},
//			true,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.description, func(t *testing.T) {
//			output, err := toCreatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage, tt.inputOptions)
//			if !tt.isValid && err == nil {
//				t.Fatalf("Should have failed")
//			}
//			if tt.isValid && err != nil {
//				t.Fatalf("Should not have failed: %v", err)
//			}
//			if tt.isValid {
//				diff := cmp.Diff(output, tt.expected)
//				if diff != "" {
//					t.Fatalf("Data does not match: %s", diff)
//				}
//			}
//		})
//	}
// }
//
// func TestToUpdatePayload(t *testing.T) {
//	tests := []struct {
//		description string
//		input       *Model
//		inputAcl    []string
//		inputFlavor *flavorModel
//		expected    *sqlserverflex.PartialUpdateInstancePayload
//		isValid     bool
//	}{
//		{
//			"default_values",
//			&Model{},
//			[]string{},
//			&flavorModel{},
//			&sqlserverflex.PartialUpdateInstancePayload{
//				Acl: &sqlserverflex.CreateInstancePayloadAcl{
//					Items: &[]string{},
//				},
//			},
//			true,
//		},
//		{
//			"simple_values",
//			&Model{
//				BackupSchedule: types.StringValue("schedule"),
//				Name:           types.StringValue("name"),
//				Replicas:       types.Int64Value(12),
//				Version:        types.StringValue("version"),
//			},
//			[]string{
//				"ip_1",
//				"ip_2",
//			},
//			&flavorModel{
//				Id: types.StringValue("flavor_id"),
//			},
//			&sqlserverflex.PartialUpdateInstancePayload{
//				Acl: &sqlserverflex.CreateInstancePayloadAcl{
//					Items: &[]string{
//						"ip_1",
//						"ip_2",
//					},
//				},
//				BackupSchedule: utils.Ptr("schedule"),
//				FlavorId:       utils.Ptr("flavor_id"),
//				Name:           utils.Ptr("name"),
//				Version:        utils.Ptr("version"),
//			},
//			true,
//		},
//		{
//			"null_fields_and_int_conversions",
//			&Model{
//				BackupSchedule: types.StringNull(),
//				Name:           types.StringNull(),
//				Replicas:       types.Int64Value(2123456789),
//				Version:        types.StringNull(),
//			},
//			[]string{
//				"",
//			},
//			&flavorModel{
//				Id: types.StringNull(),
//			},
//			&sqlserverflex.PartialUpdateInstancePayload{
//				Acl: &sqlserverflex.CreateInstancePayloadAcl{
//					Items: &[]string{
//						"",
//					},
//				},
//				BackupSchedule: nil,
//				FlavorId:       nil,
//				Name:           nil,
//				Version:        nil,
//			},
//			true,
//		},
//		{
//			"nil_model",
//			nil,
//			[]string{},
//			&flavorModel{},
//			nil,
//			false,
//		},
//		{
//			"nil_acl",
//			&Model{},
//			nil,
//			&flavorModel{},
//			&sqlserverflex.PartialUpdateInstancePayload{
//				Acl: &sqlserverflex.CreateInstancePayloadAcl{},
//			},
//			true,
//		},
//		{
//			"nil_flavor",
//			&Model{},
//			[]string{},
//			nil,
//			nil,
//			false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.description, func(t *testing.T) {
//			output, err := toUpdatePayload(tt.input, tt.inputAcl, tt.inputFlavor)
//			if !tt.isValid && err == nil {
//				t.Fatalf("Should have failed")
//			}
//			if tt.isValid && err != nil {
//				t.Fatalf("Should not have failed: %v", err)
//			}
//			if tt.isValid {
//				diff := cmp.Diff(output, tt.expected)
//				if diff != "" {
//					t.Fatalf("Data does not match: %s", diff)
//				}
//			}
//		})
//	}
// }
//
// func TestLoadFlavorId(t *testing.T) {
//	tests := []struct {
//		description     string
//		inputFlavor     *flavorModel
//		mockedResp      *sqlserverflex.ListFlavorsResponse
//		expected        *flavorModel
//		getFlavorsFails bool
//		isValid         bool
//	}{
//		{
//			"ok_flavor",
//			&flavorModel{
//				CPU: types.Int64Value(2),
//				RAM: types.Int64Value(8),
//			},
//			&sqlserverflex.ListFlavorsResponse{
//				Flavors: &[]sqlserverflex.InstanceFlavorEntry{
//					{
//						Id:          utils.Ptr("fid-1"),
//						Cpu:         utils.Ptr(int64(2)),
//						Description: utils.Ptr("description"),
//						Ram:      utils.Ptr(int64(8)),
//					},
//				},
//			},
//			&flavorModel{
//				Id:          types.StringValue("fid-1"),
//				Description: types.StringValue("description"),
//				CPU:         types.Int64Value(2),
//				RAM:         types.Int64Value(8),
//			},
//			false,
//			true,
//		},
//		{
//			"ok_flavor_2",
//			&flavorModel{
//				CPU: types.Int64Value(2),
//				RAM: types.Int64Value(8),
//			},
//			&sqlserverflex.ListFlavorsResponse{
//				Flavors: &[]sqlserverflex.InstanceFlavorEntry{
//					{
//						Id:          utils.Ptr("fid-1"),
//						Cpu:         utils.Ptr(int64(2)),
//						Description: utils.Ptr("description"),
//						Ram:      utils.Ptr(int64(8)),
//					},
//					{
//						Id:          utils.Ptr("fid-2"),
//						Cpu:         utils.Ptr(int64(1)),
//						Description: utils.Ptr("description"),
//						Ram:      utils.Ptr(int64(4)),
//					},
//				},
//			},
//			&flavorModel{
//				Id:          types.StringValue("fid-1"),
//				Description: types.StringValue("description"),
//				CPU:         types.Int64Value(2),
//				RAM:         types.Int64Value(8),
//			},
//			false,
//			true,
//		},
//		{
//			"no_matching_flavor",
//			&flavorModel{
//				CPU: types.Int64Value(2),
//				RAM: types.Int64Value(8),
//			},
//			&sqlserverflex.ListFlavorsResponse{
//				Flavors: &[]sqlserverflex.InstanceFlavorEntry{
//					{
//						Id:          utils.Ptr("fid-1"),
//						Cpu:         utils.Ptr(int64(1)),
//						Description: utils.Ptr("description"),
//						Ram:      utils.Ptr(int64(8)),
//					},
//					{
//						Id:          utils.Ptr("fid-2"),
//						Cpu:         utils.Ptr(int64(1)),
//						Description: utils.Ptr("description"),
//						Ram:      utils.Ptr(int64(4)),
//					},
//				},
//			},
//			&flavorModel{
//				CPU: types.Int64Value(2),
//				RAM: types.Int64Value(8),
//			},
//			false,
//			false,
//		},
//		{
//			"nil_response",
//			&flavorModel{
//				CPU: types.Int64Value(2),
//				RAM: types.Int64Value(8),
//			},
//			&sqlserverflex.ListFlavorsResponse{},
//			&flavorModel{
//				CPU: types.Int64Value(2),
//				RAM: types.Int64Value(8),
//			},
//			false,
//			false,
//		},
//		{
//			"error_response",
//			&flavorModel{
//				CPU: types.Int64Value(2),
//				RAM: types.Int64Value(8),
//			},
//			&sqlserverflex.ListFlavorsResponse{},
//			&flavorModel{
//				CPU: types.Int64Value(2),
//				RAM: types.Int64Value(8),
//			},
//			true,
//			false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.description, func(t *testing.T) {
//			client := &sqlserverflexClientMocked{
//				returnError:     tt.getFlavorsFails,
//				listFlavorsResp: tt.mockedResp,
//			}
//			model := &Model{
//				ProjectId: types.StringValue("pid"),
//			}
//			flavorModel := &flavorModel{
//				CPU: tt.inputFlavor.CPU,
//				RAM: tt.inputFlavor.RAM,
//			}
//			err := loadFlavorId(context.Background(), client, model, flavorModel)
//			if !tt.isValid && err == nil {
//				t.Fatalf("Should have failed")
//			}
//			if tt.isValid && err != nil {
//				t.Fatalf("Should not have failed: %v", err)
//			}
//			if tt.isValid {
//				diff := cmp.Diff(flavorModel, tt.expected)
//				if diff != "" {
//					t.Fatalf("Data does not match: %s", diff)
//				}
//			}
//		})
//	}
// }
