package postgresflex

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api"
)

type postgresFlexClientMocked struct {
	returnError     bool
	listFlavorsResp *postgresflex.ListFlavorsResponse
	listFlavorsReq  postgresflex.ApiListFlavorsRequest
}

func (c *postgresFlexClientMocked) ListFlavors(_ context.Context, _, _ string) postgresflex.ApiListFlavorsRequest {
	return c.listFlavorsReq
}

func (c *postgresFlexClientMocked) ListFlavorsExecute(_ postgresflex.ApiListFlavorsRequest) (*postgresflex.ListFlavorsResponse, error) { // nolint:gocritic // function signature required by the Go SDK
	if c.returnError {
		return nil, fmt.Errorf("get flavors failed")
	}

	return c.listFlavorsResp, nil
}

func TestMapFields(t *testing.T) {
	const testRegion = "region"

	fixtureModel := func(mods ...func(*Model)) Model {
		m := Model{
			Id:             types.StringValue("pid,region,iid"),
			InstanceId:     types.StringValue("iid"),
			ProjectId:      types.StringValue("pid"),
			Name:           types.StringValue(""),
			ACL:            types.ListNull(types.StringType),
			BackupSchedule: types.StringNull(),
			ConnectionInfo: types.ObjectValueMust(connectionInfoTypes, map[string]attr.Value{
				"write": types.ObjectValueMust(connectionInfoWriteTypes, map[string]attr.Value{
					"host": types.StringValue(""),
					"port": types.Int32Value(0),
				}),
			}),
			FlavorId: types.StringValue(""),
			Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
				"id":          types.StringNull(),
				"description": types.StringNull(),
				"cpu":         types.Int64Null(),
				"ram":         types.Int64Null(),
				"node_type":   types.StringNull(),
			}),
			Replicas: types.Int32Null(),
			Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
				"class": types.StringNull(),
				"size":  types.Int64Null(),
			}),
			Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
				"acl":              types.ListNull(types.StringType),
				"access_scope":     types.StringNull(),
				"instance_address": types.StringNull(),
				"router_address":   types.StringNull(),
			}),
			Version: types.StringValue(""),
			Region:  types.StringValue(testRegion),
		}

		for _, mod := range mods {
			mod(&m)
		}

		return m
	}

	tests := []struct {
		description string
		state       Model
		input       *postgresflex.GetInstanceResponse
		flavor      *flavorModel
		region      string
		expected    Model
		isValid     bool
	}{
		{
			description: "default_values",
			state: Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			input:    &postgresflex.GetInstanceResponse{},
			flavor:   &flavorModel{},
			region:   testRegion,
			expected: fixtureModel(),
			isValid:  true,
		},
		{
			description: "simple_values",
			state: Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			input: &postgresflex.GetInstanceResponse{
				Network: postgresflex.InstanceNetwork{
					Acl: []string{
						"ip1",
						"ip2",
						"",
					},
				},
				BackupSchedule: "schedule",
				FlavorId:       "4.8",
				Id:             "iid",
				Name:           "name",
				State:          postgresflex.STATE_READY,
				Storage: postgresflex.Storage{
					Class: new("class"),
					Size:  new(int64(78)),
				},
				Version: "version",
			},
			flavor: &flavorModel{},
			region: testRegion,
			expected: Model{
				Id:         types.StringValue("pid,region,iid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip1"),
					types.StringValue("ip2"),
					types.StringValue(""),
				}),
				ConnectionInfo: types.ObjectValueMust(connectionInfoTypes, map[string]attr.Value{
					"write": types.ObjectValueMust(connectionInfoWriteTypes, map[string]attr.Value{
						"host": types.StringValue(""),
						"port": types.Int32Value(0),
					}),
				}),
				BackupSchedule: types.StringValue("schedule"),
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
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Null(),
					"ram":         types.Int64Null(),
					"node_type":   types.StringNull(),
				}),
				FlavorId: types.StringValue("4.8"),
				Replicas: types.Int32Null(),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Version: types.StringValue("version"),
				Region:  types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "simple_values_no_flavor_and_storage",
			state: Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			input: &postgresflex.GetInstanceResponse{
				Network: postgresflex.InstanceNetwork{
					Acl: []string{
						"ip1",
						"ip2",
						"",
					},
				},
				BackupSchedule: "schedule",
				FlavorId:       "",
				Id:             "iid",
				Name:           "name",
				State:          postgresflex.STATE_READY,
				Storage: postgresflex.Storage{
					Class: new("class"),
					Size:  new(int64(78)),
				},
				Version: "version",
			},
			flavor: &flavorModel{
				Id:       types.StringValue("12.34"),
				CPU:      types.Int64Value(12),
				RAM:      types.Int64Value(34),
				NodeType: types.StringValue(NODE_TYPE_SINGLE),
			},
			region: testRegion,
			expected: Model{
				Id:         types.StringValue("pid,region,iid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip1"),
					types.StringValue("ip2"),
					types.StringValue(""),
				}),
				ConnectionInfo: types.ObjectValueMust(connectionInfoTypes, map[string]attr.Value{
					"write": types.ObjectValueMust(connectionInfoWriteTypes, map[string]attr.Value{
						"host": types.StringValue(""),
						"port": types.Int32Value(0),
					}),
				}),
				FlavorId: types.StringValue(""),
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
				BackupSchedule: types.StringValue("schedule"),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringValue("12.34"),
					"description": types.StringNull(),
					"cpu":         types.Int64Value(12),
					"ram":         types.Int64Value(34),
					"node_type":   types.StringValue(NODE_TYPE_SINGLE),
				}),
				Replicas: types.Int32Value(1),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Version: types.StringValue("version"),
				Region:  types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "acl_unordered",
			state: Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip2"),
					types.StringValue(""),
					types.StringValue("ip1"),
				}),
			},
			input: &postgresflex.GetInstanceResponse{
				Network: postgresflex.InstanceNetwork{
					Acl: []string{
						"",
						"ip1",
						"ip2",
					},
				},
				BackupSchedule: "schedule",
				Id:             "iid",
				Name:           "name",
				Storage: postgresflex.Storage{
					Class: new("class"),
					Size:  new(int64(78)),
				},
				Version: "version",
			},
			flavor: &flavorModel{
				CPU:      types.Int64Value(12),
				RAM:      types.Int64Value(34),
				NodeType: types.StringValue(NODE_TYPE_REPLICA),
			},
			region: testRegion,
			expected: Model{
				Id:         types.StringValue("pid,region,iid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip2"),
					types.StringValue(""),
					types.StringValue("ip1"),
				}),
				ConnectionInfo: types.ObjectValueMust(connectionInfoTypes, map[string]attr.Value{
					"write": types.ObjectValueMust(connectionInfoWriteTypes, map[string]attr.Value{
						"host": types.StringValue(""),
						"port": types.Int32Value(0),
					}),
				}),
				BackupSchedule: types.StringValue("schedule"),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Value(12),
					"ram":         types.Int64Value(34),
					"node_type":   types.StringValue(NODE_TYPE_REPLICA),
				}),
				FlavorId: types.StringValue(""),
				Replicas: types.Int32Value(NODE_TYPE_REPLICA_VALUE),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("ip2"),
						types.StringValue(""),
						types.StringValue("ip1"),
					}),
					"access_scope":     types.StringNull(),
					"instance_address": types.StringNull(),
					"router_address":   types.StringNull(),
				}),
				Version: types.StringValue("version"),
				Region:  types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "backup schedule - keep state value when API strips leading zeros",
			state: fixtureModel(func(m *Model) {
				m.BackupSchedule = types.StringValue("00 00 * * *")
			}),
			input: &postgresflex.GetInstanceResponse{
				BackupSchedule: "0 0 * * *",
			},
			flavor: &flavorModel{},
			region: testRegion,
			expected: fixtureModel(func(m *Model) {
				m.BackupSchedule = types.StringValue("00 00 * * *")
			}),
			isValid: true,
		},
		{
			description: "backup schedule - use updated value from API if cron actually changed",
			state: fixtureModel(func(m *Model) {
				m.BackupSchedule = types.StringValue("00 01 * * *")
			}),
			input: &postgresflex.GetInstanceResponse{
				BackupSchedule: "0 2 * * *",
			},
			flavor: &flavorModel{},
			region: testRegion,
			expected: fixtureModel(func(m *Model) {
				m.BackupSchedule = types.StringValue("0 2 * * *")
			}),
			isValid: true,
		},
		{
			description: "nil_response",
			state: Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			input:    nil,
			flavor:   &flavorModel{},
			region:   testRegion,
			expected: Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, tt.flavor, tt.region)
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
		description     string
		input           *Model
		inputAcl        []string
		inputFlavor     *flavorModel
		inputStorage    *storageModel
		inputNetwork    *networkModel
		inputEncryption *encryptionModel
		expected        *postgresflex.CreateInstancePayload
		isValid         bool
	}{
		{
			description: "default_values",
			input: &Model{
				FlavorId: types.StringValue("1.2"),
			},
			inputAcl:        []string{},
			inputFlavor:     &flavorModel{},
			inputStorage:    &storageModel{},
			inputNetwork:    &networkModel{},
			inputEncryption: nil,
			expected: &postgresflex.CreateInstancePayload{
				FlavorId: "1.2",
				Network: postgresflex.InstanceNetworkCreate{
					Acl: []string{},
				},
				Storage: postgresflex.StorageCreate{},
			},
			isValid: true,
		},
		{
			description: "simple_values",
			input: &Model{
				BackupSchedule: types.StringValue("schedule"),
				Name:           types.StringValue("name"),
				Replicas:       types.Int32Value(12),
				Version:        types.StringValue("version"),
			},
			inputAcl: []string{
				"ip_1",
				"ip_2",
			},
			inputFlavor: &flavorModel{
				Id: types.StringValue("flavor_id"),
			},
			inputStorage: &storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(34),
			},
			inputNetwork: &networkModel{
				Acl: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("test"),
				}),
			},
			inputEncryption: nil,
			expected: &postgresflex.CreateInstancePayload{
				Network: postgresflex.InstanceNetworkCreate{
					Acl: []string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: "schedule",
				FlavorId:       "flavor_id",
				Name:           "name",
				Storage: postgresflex.StorageCreate{
					Class: new("class"),
					Size:  34,
				},
				Version: "version",
			},
			isValid: true,
		},
		{
			description: "null_fields_and_int_conversions",
			input: &Model{
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int32Value(2123456789),
				Version:        types.StringNull(),
				FlavorId:       types.StringValue("flavor_id"),
			},
			inputAcl: []string{
				"",
			},
			inputFlavor: &flavorModel{
				Id: types.StringNull(),
			},
			inputStorage: &storageModel{
				Class: types.StringNull(),
				Size:  types.Int64Null(),
			},
			inputNetwork: &networkModel{
				Acl: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("test"),
				}),
			},
			inputEncryption: nil,
			expected: &postgresflex.CreateInstancePayload{
				Network: postgresflex.InstanceNetworkCreate{
					Acl: []string{
						"",
					},
				},
				BackupSchedule: "",
				FlavorId:       "flavor_id",
				Name:           "",
				Storage: postgresflex.StorageCreate{
					Class: nil,
					Size:  0,
				},
				Version: "",
			},
			isValid: true,
		},
		{
			description:  "nil_model",
			input:        nil,
			inputAcl:     []string{},
			inputFlavor:  &flavorModel{},
			inputStorage: &storageModel{},
			inputNetwork: &networkModel{
				Acl: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("test"),
				}),
			},
			inputEncryption: nil,
			expected:        nil,
			isValid:         false,
		},
		{
			description:  "nil_acl",
			input:        &Model{},
			inputAcl:     nil,
			inputFlavor:  &flavorModel{},
			inputStorage: &storageModel{},
			inputNetwork: &networkModel{
				Acl: types.ListNull(types.StringType),
			},
			inputEncryption: nil,
			expected:        nil,
			isValid:         false,
		},
		{
			description:     "nil_flavor",
			input:           &Model{},
			inputAcl:        []string{},
			inputFlavor:     nil,
			inputStorage:    &storageModel{},
			inputNetwork:    &networkModel{},
			inputEncryption: nil,
			expected:        nil,
			isValid:         false,
		},
		{
			description:     "nil_storage",
			input:           &Model{},
			inputAcl:        []string{},
			inputFlavor:     &flavorModel{},
			inputStorage:    nil,
			inputNetwork:    &networkModel{},
			inputEncryption: nil,
			expected:        nil,
			isValid:         false,
		},
		{
			description:     "nil_network",
			input:           &Model{},
			inputAcl:        []string{},
			inputFlavor:     &flavorModel{},
			inputStorage:    &storageModel{},
			inputNetwork:    nil,
			inputEncryption: nil,
			expected:        nil,
			isValid:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage, tt.inputNetwork, tt.inputEncryption)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(postgresflex.NullableInt32{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description  string
		input        *Model
		inputAcl     []string
		inputFlavor  *flavorModel
		inputStorage *storageModel
		inputNetwork *networkModel
		expected     *postgresflex.PartialUpdateInstancePayload
		isValid      bool
	}{
		{
			description: "default_values",
			input: &Model{
				FlavorId: types.StringValue("flavor_id"),
			},
			inputAcl:     []string{},
			inputFlavor:  &flavorModel{},
			inputStorage: &storageModel{},
			expected: &postgresflex.PartialUpdateInstancePayload{
				FlavorId: new("flavor_id"),
				Network: &postgresflex.InstanceNetworkOpt{
					Acl: []string{},
				},
				Storage: &postgresflex.StorageUpdate{},
			},
			isValid: true,
		},
		{
			description: "simple_values",
			input: &Model{
				BackupSchedule: types.StringValue("schedule"),
				Name:           types.StringValue("name"),
				Replicas:       types.Int32Value(12),
				Version:        types.StringValue("version"),
			},
			inputAcl: []string{
				"ip_1",
				"ip_2",
			},
			inputFlavor: &flavorModel{
				Id: types.StringValue("flavor_id"),
			},
			inputStorage: &storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(34),
			},
			expected: &postgresflex.PartialUpdateInstancePayload{
				Network: &postgresflex.InstanceNetworkOpt{
					Acl: []string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: new("schedule"),
				FlavorId:       new("flavor_id"),
				Name:           new("name"),
				Version:        new("version"),
				Storage: &postgresflex.StorageUpdate{
					Size: new(int64(34)),
				},
			},
			isValid: true,
		},
		{
			description: "null_fields_and_int_conversions",
			input: &Model{
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int32Value(2123456789),
				Version:        types.StringNull(),
				FlavorId:       types.StringValue("flavor_id"),
			},
			inputAcl: []string{
				"",
			},
			inputFlavor: &flavorModel{
				Id: types.StringNull(),
			},
			inputStorage: &storageModel{
				Class: types.StringNull(),
				Size:  types.Int64Null(),
			},
			expected: &postgresflex.PartialUpdateInstancePayload{
				Network: &postgresflex.InstanceNetworkOpt{
					Acl: []string{
						"",
					},
				},
				BackupSchedule: nil,
				FlavorId:       new("flavor_id"),
				Name:           nil,
				Version:        nil,
				Storage:        &postgresflex.StorageUpdate{},
			},
			isValid: true,
		},
		{
			description:  "nil_model",
			input:        nil,
			inputAcl:     []string{},
			inputFlavor:  &flavorModel{},
			inputStorage: &storageModel{},
			expected:     nil,
			isValid:      false,
		},
		{
			description:  "nil_acl",
			input:        &Model{},
			inputAcl:     nil,
			inputFlavor:  &flavorModel{},
			inputStorage: &storageModel{},
			expected:     nil,
			isValid:      false,
		},
		{
			description:  "nil_flavor",
			input:        &Model{},
			inputAcl:     []string{},
			inputFlavor:  nil,
			inputStorage: &storageModel{},
			expected:     nil,
			isValid:      false,
		},
		{
			description:  "nil_storage",
			input:        &Model{},
			inputAcl:     []string{},
			inputFlavor:  &flavorModel{},
			inputStorage: nil,
			expected:     nil,
			isValid:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage, tt.inputNetwork)
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

func TestLoadFlavorId(t *testing.T) {
	tests := []struct {
		description     string
		inputFlavor     *flavorModel
		inputReplicas   int32
		mockedResp      *postgresflex.ListFlavorsResponse
		expected        *flavorModel
		getFlavorsFails bool
		isValid         bool
	}{
		{
			description: "ok_flavor",
			inputFlavor: &flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			inputReplicas: NODE_TYPE_SINGLE_VALUE,
			mockedResp: &postgresflex.ListFlavorsResponse{
				Flavors: []postgresflex.ListFlavors{
					{
						Id:          "fid-1",
						Cpu:         2,
						Description: "description",
						Memory:      8,
						NodeType:    NODE_TYPE_SINGLE,
					},
				},
			},
			expected: &flavorModel{
				Id:          types.StringValue("fid-1"),
				Description: types.StringValue("description"),
				CPU:         types.Int64Value(2),
				RAM:         types.Int64Value(8),
				NodeType:    types.StringValue(NODE_TYPE_SINGLE),
			},
			getFlavorsFails: false,
			isValid:         true,
		},
		{
			description: "ok_flavor_2",
			inputFlavor: &flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			inputReplicas: NODE_TYPE_REPLICA_VALUE,
			mockedResp: &postgresflex.ListFlavorsResponse{
				Flavors: []postgresflex.ListFlavors{
					{
						Id:          "fid-1",
						Cpu:         2,
						Description: "description",
						Memory:      8,
						NodeType:    NODE_TYPE_REPLICA,
					},
					{
						Id:          "fid-2",
						Cpu:         1,
						Description: "description",
						Memory:      4,
						NodeType:    NODE_TYPE_SINGLE,
					},
				},
			},
			expected: &flavorModel{
				Id:          types.StringValue("fid-1"),
				Description: types.StringValue("description"),
				CPU:         types.Int64Value(2),
				RAM:         types.Int64Value(8),
				NodeType:    types.StringValue(NODE_TYPE_REPLICA),
			},
			getFlavorsFails: false,
			isValid:         true,
		},
		{
			description: "no_matching_flavor",
			inputFlavor: &flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			inputReplicas: NODE_TYPE_SINGLE_VALUE,
			mockedResp: &postgresflex.ListFlavorsResponse{
				Flavors: []postgresflex.ListFlavors{
					{
						Id:          "fid-1",
						Cpu:         1,
						Description: "description",
						Memory:      8,
						NodeType:    NODE_TYPE_REPLICA,
					},
					{
						Id:          "fid-2",
						Cpu:         1,
						Description: "description",
						Memory:      4,
						NodeType:    NODE_TYPE_REPLICA,
					},
					{
						Id:          "fid-3",
						Cpu:         2,
						Description: "description",
						Memory:      8,
						NodeType:    NODE_TYPE_REPLICA,
					},
				},
			},
			expected: &flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			getFlavorsFails: false,
			isValid:         false,
		},
		{
			description: "nil_response",
			inputFlavor: &flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			mockedResp: &postgresflex.ListFlavorsResponse{},
			expected: &flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			getFlavorsFails: false,
			isValid:         false,
		},
		{
			description: "error_response",
			inputFlavor: &flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			mockedResp: &postgresflex.ListFlavorsResponse{},
			expected: &flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			getFlavorsFails: true,
			isValid:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &postgresFlexClientMocked{
				returnError:     tt.getFlavorsFails,
				listFlavorsResp: tt.mockedResp,
			}
			model := &Model{
				ProjectId: types.StringValue("pid"),
				Replicas:  types.Int32Value(tt.inputReplicas),
			}
			flavorModel := &flavorModel{
				CPU: tt.inputFlavor.CPU,
				RAM: tt.inputFlavor.RAM,
			}
			err := loadFlavorId(context.Background(), client, model, flavorModel)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(flavorModel, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestHandleV3Migration(t *testing.T) {
	tests := []struct {
		name         string
		configModel  *Model
		planModel    *Model
		expectedPlan *Model
		warnings     int
	}{
		{
			name: "all_values_provided_no_migration",
			configModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("0 16 * * *"),
				RetentionDays:  types.Int32Value(40),
			},
			planModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("0 16 * * *"),
				RetentionDays:  types.Int32Value(40),
			},
			expectedPlan: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("0 16 * * *"),
				RetentionDays:  types.Int32Value(40),
			},
			warnings: 0,
		},
		{
			name: "migration_triggered_for_all_null_fields",
			configModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringNull(),
				RetentionDays:  types.Int32Null(),
			},
			planModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringNull(),
				RetentionDays:  types.Int32Null(),
			},
			expectedPlan: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringNull(),
				RetentionDays:  types.Int32Value(32),
			},
			warnings: 1, // retention_days fallback
		},
		{
			name: "migration_triggered_for_unknown_retention_days",
			configModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("0 16 * * *"),
				RetentionDays:  types.Int32Unknown(),
			},
			planModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("0 16 * * *"),
				RetentionDays:  types.Int32Unknown(),
			},
			expectedPlan: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("0 16 * * *"),
				RetentionDays:  types.Int32Value(32),
			},
			warnings: 1,
		},
		{
			name: "backup_schedule_unsimplified_triggers_warning",
			configModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("00 16 * * *"),
				RetentionDays:  types.Int32Value(40),
			},
			planModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("00 16 * * *"),
				RetentionDays:  types.Int32Value(40),
			},
			expectedPlan: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("00 16 * * *"),
				RetentionDays:  types.Int32Value(40),
			},
			warnings: 1, // backup_schedule simplification warning
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &resource.ModifyPlanResponse{}
			handleV3Migration(context.Background(), tt.planModel, tt.configModel, resp)

			if len(resp.Diagnostics.Warnings()) != tt.warnings {
				t.Errorf("expected %d warnings, got %d", tt.warnings, len(resp.Diagnostics.Warnings()))
			}

			diff := cmp.Diff(tt.planModel, tt.expectedPlan)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func TestGetFlavor(t *testing.T) {
	tests := []struct {
		description     string
		flavorId        string
		mockedResp      *postgresflex.ListFlavorsResponse
		expected        *postgresflex.ListFlavors
		getFlavorsFails bool
		isValid         bool
	}{
		{
			description: "ok_flavor_found",
			flavorId:    "fid-1",
			mockedResp: &postgresflex.ListFlavorsResponse{
				Flavors: []postgresflex.ListFlavors{
					{
						Id:          "fid-1",
						Cpu:         2,
						Description: "description-1",
						Memory:      8,
					},
					{
						Id:          "fid-2",
						Cpu:         4,
						Description: "description-2",
						Memory:      16,
					},
				},
			},
			expected: &postgresflex.ListFlavors{
				Id:          "fid-1",
				Cpu:         2,
				Description: "description-1",
				Memory:      8,
			},
			getFlavorsFails: false,
			isValid:         true,
		},
		{
			description: "flavor_not_found",
			flavorId:    "fid-3",
			mockedResp: &postgresflex.ListFlavorsResponse{
				Flavors: []postgresflex.ListFlavors{
					{
						Id:          "fid-1",
						Cpu:         2,
						Description: "description-1",
						Memory:      8,
					},
				},
			},
			expected:        nil,
			getFlavorsFails: false,
			isValid:         false,
		},
		{
			description:     "error_response",
			flavorId:        "fid-1",
			mockedResp:      nil,
			expected:        nil,
			getFlavorsFails: true,
			isValid:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &postgresFlexClientMocked{
				returnError:     tt.getFlavorsFails,
				listFlavorsResp: tt.mockedResp,
			}
			got, err := getFlavor(context.Background(), client, "pid", "region", tt.flavorId)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(got, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
