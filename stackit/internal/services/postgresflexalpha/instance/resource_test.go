package postgresflexalpha

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

// type postgresFlexClientMocked struct {
//	returnError    bool
//	getFlavorsResp *postgresflex.GetFlavorsResponse
// }
//
// func (c *postgresFlexClientMocked) ListFlavorsExecute(_ context.Context, _, _ string) (*postgresflex.GetFlavorsResponse, error) {
//	if c.returnError {
//		return nil, fmt.Errorf("get flavors failed")
//	}
//
//	return c.getFlavorsResp, nil
// }

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		state       Model
		input       *postgresflex.GetInstanceResponse
		flavor      *flavorModel
		storage     *storageModel
		encryption  *encryptionModel
		network     *networkModel
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values does exactly mean what?",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Replicas:   types.Int64Value(1),
			},
			&postgresflex.GetInstanceResponse{
				FlavorId: utils.Ptr("flavor_id"),
				Replicas: postgresflex.GetInstanceResponseGetReplicasAttributeType(utils.Ptr(int32(1))),
			},
			&flavorModel{
				NodeType: types.StringValue("Single"),
			},
			&storageModel{},
			&encryptionModel{},
			&networkModel{
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("0.0.0.0/0"),
				}),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringNull(),
				//ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringNull(),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Null(),
					"ram":         types.Int64Null(),
					"node_type":   types.StringValue("Single"),
				}),
				Replicas: types.Int64Value(1),
				Encryption: types.ObjectValueMust(encryptionTypes, map[string]attr.Value{
					"keyring_id":      types.StringNull(),
					"key_id":          types.StringNull(),
					"key_version":     types.StringNull(),
					"service_account": types.StringNull(),
				}),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringNull(),
					"size":  types.Int64Null(),
				}),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("0.0.0.0/0"),
					}),
					"access_scope":     types.StringNull(),
					"instance_address": types.StringNull(),
					"router_address":   types.StringNull(),
				}),
				Version: types.StringNull(),
				Region:  types.StringValue(testRegion),
			},
			true,
		},
		// {
		//	"acl_unordered",
		//	Model{
		//		InstanceId: types.StringValue("iid"),
		//		ProjectId:  types.StringValue("pid"),
		//		// ACL: types.ListValueMust(types.StringType, []attr.Value{
		//		//	types.StringValue("ip2"),
		//		//	types.StringValue(""),
		//		//	types.StringValue("ip1"),
		//		// }),
		//	},
		//	&postgresflex.GetInstanceResponse{
		//		// Acl: &[]string{
		//		//	"",
		//		//	"ip1",
		//		//	"ip2",
		//		// },
		//		BackupSchedule: utils.Ptr("schedule"),
		//		FlavorId:       nil,
		//		Id:             utils.Ptr("iid"),
		//		Name:           utils.Ptr("name"),
		//		Replicas:       postgresflex.GetInstanceResponseGetReplicasAttributeType(utils.Ptr(int32(56))),
		//		Status:         postgresflex.GetInstanceResponseGetStatusAttributeType(utils.Ptr("status")),
		//		Storage:        nil,
		//		Version:        utils.Ptr("version"),
		//	},
		//	&flavorModel{
		//		CPU: types.Int64Value(12),
		//		RAM: types.Int64Value(34),
		//	},
		//	&storageModel{
		//		Class: types.StringValue("class"),
		//		Size:  types.Int64Value(78),
		//	},
		//	&encryptionModel{},
		//	&networkModel{},
		//	testRegion,
		//	Model{
		//		Id:         types.StringValue("pid,region,iid"),
		//		InstanceId: types.StringValue("iid"),
		//		ProjectId:  types.StringValue("pid"),
		//		Name:       types.StringValue("name"),
		//		// ACL: types.ListValueMust(types.StringType, []attr.Value{
		//		//	types.StringValue("ip2"),
		//		//	types.StringValue(""),
		//		//	types.StringValue("ip1"),
		//		// }),
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
		//		Version: types.StringValue("version"),
		//		Region:  types.StringValue(testRegion),
		//	},
		//	true,
		// },
		{
			"nil_response",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			nil,
			&flavorModel{},
			&storageModel{},
			&encryptionModel{},
			&networkModel{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&postgresflex.GetInstanceResponse{},
			&flavorModel{},
			&storageModel{},
			&encryptionModel{},
			&networkModel{},
			testRegion,
			Model{},
			false,
		},
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
				diff := cmp.Diff(tt.expected, tt.state)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description     string
		input           *Model
		inputAcl        []string
		inputFlavor     *flavorModel
		inputStorage    *storageModel
		inputEncryption *encryptionModel
		inputNetwork    *networkModel
		expected        *postgresflex.CreateInstanceRequestPayload
		isValid         bool
	}{
		{
			"default_values",
			&Model{
				Replicas: types.Int64Value(1),
			},
			[]string{},
			&flavorModel{},
			&storageModel{},
			&encryptionModel{},
			&networkModel{
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("0.0.0.0/0"),
				}),
			},
			&postgresflex.CreateInstanceRequestPayload{
				Storage:    postgresflex.CreateInstanceRequestPayloadGetStorageAttributeType(&postgresflex.Storage{}),
				Encryption: &postgresflex.InstanceEncryption{},
				Network: &postgresflex.InstanceNetwork{
					Acl: &[]string{"0.0.0.0/0"},
				},
				Replicas: postgresflex.CreateInstanceRequestPayloadGetReplicasAttributeType(utils.Ptr(int32(1))),
			},
			true,
		},
		{
			"use flavor node_type instead of replicas",
			&Model{},
			[]string{},
			&flavorModel{
				NodeType: types.StringValue("Single"),
			},
			&storageModel{},
			&encryptionModel{},
			&networkModel{
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("0.0.0.0/0"),
				}),
			},
			&postgresflex.CreateInstanceRequestPayload{
				//Acl:     &[]string{},
				Storage:    postgresflex.CreateInstanceRequestPayloadGetStorageAttributeType(&postgresflex.Storage{}),
				Encryption: &postgresflex.InstanceEncryption{},
				Network: &postgresflex.InstanceNetwork{
					Acl: &[]string{"0.0.0.0/0"},
				},
				Replicas: postgresflex.CreateInstanceRequestPayloadGetReplicasAttributeType(utils.Ptr(int32(1))),
			},
			true,
		},
		{
			"nil_model",
			nil,
			[]string{},
			&flavorModel{},
			&storageModel{},
			&encryptionModel{},
			&networkModel{},
			nil,
			false,
		},
		{
			"nil_acl",
			&Model{},
			nil,
			&flavorModel{},
			&storageModel{},
			&encryptionModel{},
			&networkModel{},
			nil,
			false,
		},
		{
			"nil_flavor",
			&Model{},
			[]string{},
			nil,
			&storageModel{},
			&encryptionModel{},
			&networkModel{},
			nil,
			false,
		},
		{
			"nil_storage",
			&Model{},
			[]string{},
			&flavorModel{},
			nil,
			&encryptionModel{},
			&networkModel{},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputFlavor, tt.inputStorage, tt.inputEncryption, tt.inputNetwork)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.expected, output)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

// func TestToUpdatePayload(t *testing.T) {
//	tests := []struct {
//		description  string
//		input        *Model
//		inputAcl     []string
//		inputFlavor  *flavorModel
//		inputStorage *storageModel
//		expected     *postgresflex.PartialUpdateInstancePayload
//		isValid      bool
//	}{
//		{
//			"default_values",
//			&Model{},
//			[]string{},
//			&flavorModel{},
//			&storageModel{},
//			&postgresflex.PartialUpdateInstancePayload{
//				Acl: &postgresflex.ACL{
//					Items: &[]string{},
//				},
//				Storage: &postgresflex.Storage{},
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
//			&postgresflex.PartialUpdateInstancePayload{
//				Acl: &postgresflex.ACL{
//					Items: &[]string{
//						"ip_1",
//						"ip_2",
//					},
//				},
//				BackupSchedule: utils.Ptr("schedule"),
//				FlavorId:       utils.Ptr("flavor_id"),
//				Name:           utils.Ptr("name"),
//				Replicas:       utils.Ptr(int64(12)),
//				Storage: &postgresflex.Storage{
//					Class: utils.Ptr("class"),
//					Size:  utils.Ptr(int64(34)),
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
//			&postgresflex.PartialUpdateInstancePayload{
//				Acl: &postgresflex.ACL{
//					Items: &[]string{
//						"",
//					},
//				},
//				BackupSchedule: nil,
//				FlavorId:       nil,
//				Name:           nil,
//				Replicas:       utils.Ptr(int64(2123456789)),
//				Storage: &postgresflex.Storage{
//					Class: nil,
//					Size:  nil,
//				},
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
//			nil,
//			false,
//		},
//		{
//			"nil_acl",
//			&Model{},
//			nil,
//			&flavorModel{},
//			&storageModel{},
//			nil,
//			false,
//		},
//		{
//			"nil_flavor",
//			&Model{},
//			[]string{},
//			nil,
//			&storageModel{},
//			nil,
//			false,
//		},
//		{
//			"nil_storage",
//			&Model{},
//			[]string{},
//			&flavorModel{},
//			nil,
//			nil,
//			false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.description, func(t *testing.T) {
//			output, err := toUpdatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage)
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
//		mockedResp      *postgresflex.ListFlavorsResponse
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
//			&postgresflex.ListFlavorsResponse{
//				Flavors: &[]postgresflex.Flavor{
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
//			&postgresflex.ListFlavorsResponse{
//				Flavors: &[]postgresflex.Flavor{
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
//			&postgresflex.ListFlavorsResponse{
//				Flavors: &[]postgresflex.Flavor{
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
//			&postgresflex.ListFlavorsResponse{},
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
//			&postgresflex.ListFlavorsResponse{},
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
//			client := &postgresFlexClientMocked{
//				returnError:    tt.getFlavorsFails,
//				getFlavorsResp: tt.mockedResp,
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

func TestNewInstanceResource(t *testing.T) {
	tests := []struct {
		name string
		want resource.Resource
	}{
		{
			name: "create empty instance resource",
			want: &instanceResource{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInstanceResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInstanceResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
