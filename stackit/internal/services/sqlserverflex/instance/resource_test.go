package sqlserverflex

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"
)

type sqlserverflexClientMocked struct {
	returnError     bool
	listFlavorsResp *sqlserverflex.ListFlavorsResponse
	listFlavorsReq  sqlserverflex.ApiListFlavorsRequest
}

func (c *sqlserverflexClientMocked) ListFlavors(_ context.Context, _, _ string) sqlserverflex.ApiListFlavorsRequest {
	return c.listFlavorsReq
}

func (c *sqlserverflexClientMocked) ListFlavorsExecute(_ sqlserverflex.ApiListFlavorsRequest) (*sqlserverflex.ListFlavorsResponse, error) { // nolint:gocritic // function signature required by generated SDK
	if c.returnError {
		return nil, fmt.Errorf("get flavors failed")
	}

	return c.listFlavorsResp, nil
}

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		state       Model
		input       *sqlserverflex.GetInstanceResponse
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
			input:  &sqlserverflex.GetInstanceResponse{},
			flavor: &flavorModel{},
			region: testRegion,
			expected: Model{
				Id:             types.StringValue("pid,region,iid"),
				InstanceId:     types.StringValue("iid"),
				ProjectId:      types.StringValue("pid"),
				Name:           types.StringValue(""),
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringNull(),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Null(),
					"ram":         types.Int64Null(),
				}),
				FlavorId: types.StringValue(""),
				Replicas: types.Int32Value(0),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringNull(),
					"size":  types.Int64Null(),
				}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"edition":        types.StringValue(""),
					"retention_days": types.Int32Value(0),
				}),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListNull(types.StringType),
					"access_scope": types.StringNull(),
				}),
				RetentionDays: types.Int32Value(0),
				Edition:       types.StringValue(""),
				Version:       types.StringValue(""),
				Region:        types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			state: Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			input: &sqlserverflex.GetInstanceResponse{
				Network: sqlserverflex.InstanceNetwork{
					Acl: []string{
						"ip1",
						"ip2",
						"",
					},
				},
				BackupSchedule: "schedule",
				FlavorId:       "flavor_id",
				Id:             "iid",
				Name:           "name",
				Replicas:       56,
				State:          "status",
				Storage: sqlserverflex.Storage{
					Class: new("class"),
					Size:  new(int64(78)),
				},
				Edition:       "edition",
				RetentionDays: 1,
				Version:       "version",
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
				BackupSchedule: types.StringValue("schedule"),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Null(),
					"ram":         types.Int64Null(),
				}),
				FlavorId:      types.StringValue("flavor_id"),
				Edition:       types.StringValue("edition"),
				RetentionDays: types.Int32Value(1),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("ip1"), types.StringValue("ip2"), types.StringValue("")}),
					"access_scope": types.StringNull(),
				}),
				Replicas: types.Int32Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"edition":        types.StringValue("edition"),
					"retention_days": types.Int32Value(1),
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
			input: &sqlserverflex.GetInstanceResponse{
				BackupSchedule: "schedule",
				FlavorId:       "",
				Id:             "iid",
				Name:           "name",
				Replicas:       56,
				State:          "status",
				Storage: sqlserverflex.Storage{
					Class: new("class"),
					Size:  new(int64(78)),
				},
				Network: sqlserverflex.InstanceNetwork{
					Acl: []string{
						"ip1",
						"ip2",
						"",
					},
				},
				Edition:       "edition",
				RetentionDays: 1,
				Version:       "version",
			},
			flavor: &flavorModel{
				CPU: types.Int64Value(12),
				RAM: types.Int64Value(34),
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
				BackupSchedule: types.StringValue("schedule"),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Value(12),
					"ram":         types.Int64Value(34),
				}),
				FlavorId:      types.StringValue(""),
				Edition:       types.StringValue("edition"),
				RetentionDays: types.Int32Value(1),
				Replicas:      types.Int32Value(56),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("ip1"), types.StringValue("ip2"), types.StringValue("")}),
					"access_scope": types.StringNull(),
				}),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"edition":        types.StringValue("edition"),
					"retention_days": types.Int32Value(1),
				}),
				Version: types.StringValue("version"),
				Region:  types.StringValue(testRegion),
			},
			isValid: true,
		},
		{
			description: "acls_unordered",
			state: Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip2"),
					types.StringValue(""),
					types.StringValue("ip1"),
				}),
			},
			input: &sqlserverflex.GetInstanceResponse{
				Network: sqlserverflex.InstanceNetwork{
					Acl: []string{
						"",
						"ip1",
						"ip2",
					},
				},
				BackupSchedule: "schedule",
				FlavorId:       "",
				Id:             "iid",
				Name:           "name",
				Replicas:       56,
				State:          "status",
				Storage: sqlserverflex.Storage{
					Class: new("class"),
					Size:  new(int64(78)),
				},
				Edition:       "edition",
				RetentionDays: 1,
				Version:       "version",
			},
			flavor: &flavorModel{
				CPU: types.Int64Value(12),
				RAM: types.Int64Value(34),
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
				BackupSchedule: types.StringValue("schedule"),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Value(12),
					"ram":         types.Int64Value(34),
				}),
				FlavorId:      types.StringValue(""),
				Edition:       types.StringValue("edition"),
				RetentionDays: types.Int32Value(1),
				Replicas:      types.Int32Value(56),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("ip2"), types.StringValue(""), types.StringValue("ip1")}),
					"access_scope": types.StringNull(),
				}),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"edition":        types.StringValue("edition"),
					"retention_days": types.Int32Value(1),
				}),
				Version: types.StringValue("version"),
				Region:  types.StringValue(testRegion),
			},
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
		{
			description: "no_resource_id",
			state: Model{
				ProjectId: types.StringValue("pid"),
			},
			input:    &sqlserverflex.GetInstanceResponse{},
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
		inputEncryption *encryptionModel
		inputFlavor     *flavorModel
		inputStorage    *storageModel
		inputOptions    *optionsModel
		inputNetwork    *networkModel
		expected        *sqlserverflex.CreateInstancePayload
		isValid         bool
	}{
		{
			description: "default_values",
			input: &Model{
				FlavorId:      types.StringValue("fid"),
				RetentionDays: types.Int32Value(1),
			},
			inputAcl:        []string{},
			inputEncryption: &encryptionModel{},
			inputFlavor:     &flavorModel{},
			inputStorage:    &storageModel{},
			inputOptions:    &optionsModel{},
			inputNetwork:    &networkModel{},
			expected: &sqlserverflex.CreateInstancePayload{
				FlavorId:      "fid",
				RetentionDays: 1,
				Network: sqlserverflex.CreateInstancePayloadNetwork{
					Acl: []string{},
				},
				Storage: sqlserverflex.StorageCreate{
					Class: "",
					Size:  0,
				},
				Encryption: &sqlserverflex.InstanceEncryption{},
			},
			isValid: true,
		},
		{
			description: "simple_values",
			input: &Model{
				FlavorId:       types.StringValue("fid"),
				BackupSchedule: types.StringValue("schedule"),
				Name:           types.StringValue("name"),
				Replicas:       types.Int32Value(12),
				Version:        types.StringValue("version"),
			},
			inputAcl: []string{
				"ip_1",
				"ip_2",
			},
			inputFlavor: &flavorModel{},
			inputStorage: &storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(34),
			},
			inputOptions: &optionsModel{
				Edition:       types.StringValue("edition"),
				RetentionDays: types.Int32Value(1),
			},
			inputNetwork: &networkModel{},
			inputEncryption: &encryptionModel{
				KekKeyId:       types.StringValue("id"),
				KekKeyRingId:   types.StringValue("keyRingId"),
				KekKeyVersion:  types.StringValue("keyVersion"),
				ServiceAccount: types.StringValue("some_service_account"),
			},
			expected: &sqlserverflex.CreateInstancePayload{
				Network: sqlserverflex.CreateInstancePayloadNetwork{
					Acl: []string{"ip_1", "ip_2"},
				},
				BackupSchedule: "schedule",
				FlavorId:       "fid",
				Name:           "name",
				Storage: sqlserverflex.StorageCreate{
					Class: "class",
					Size:  34,
				},
				RetentionDays: 1,
				Version:       "version",
				Encryption: &sqlserverflex.InstanceEncryption{
					KekKeyId:       "id",
					KekKeyRingId:   "keyRingId",
					KekKeyVersion:  "keyVersion",
					ServiceAccount: "some_service_account",
				},
			},
			isValid: true,
		},
		{
			description: "null_fields_and_int_conversions",
			input: &Model{
				FlavorId:       types.StringValue("fid"),
				RetentionDays:  types.Int32Value(1),
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int32Value(2123456789),
				Version:        types.StringNull(),
			},
			inputAcl: []string{
				"",
			},
			inputFlavor: &flavorModel{},
			inputStorage: &storageModel{
				Class: types.StringNull(),
				Size:  types.Int64Null(),
			},
			inputOptions: &optionsModel{
				Edition:       types.StringNull(),
				RetentionDays: types.Int32Null(),
			},
			inputNetwork: &networkModel{},
			expected: &sqlserverflex.CreateInstancePayload{
				Network: sqlserverflex.CreateInstancePayloadNetwork{
					Acl: []string{""},
				},
				RetentionDays:  1,
				BackupSchedule: "",
				FlavorId:       "fid",
				Name:           "",
				Storage: sqlserverflex.StorageCreate{
					Class: "",
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
			inputOptions: &optionsModel{},
			inputNetwork: &networkModel{},
			expected:     nil,
			isValid:      false,
		},
		{
			description: "nil_acl",
			input: &Model{
				FlavorId:      types.StringValue("fid"),
				RetentionDays: types.Int32Value(0),
			},
			inputAcl:     nil,
			inputFlavor:  &flavorModel{},
			inputStorage: &storageModel{},
			inputOptions: &optionsModel{},
			inputNetwork: &networkModel{},
			expected: &sqlserverflex.CreateInstancePayload{
				FlavorId: "fid",
				Network: sqlserverflex.CreateInstancePayloadNetwork{
					Acl: []string{},
				},
			},
			isValid: true,
		},
		{
			description:  "nil_flavor",
			input:        &Model{},
			inputAcl:     []string{},
			inputFlavor:  nil,
			inputStorage: &storageModel{},
			inputOptions: &optionsModel{},
			inputNetwork: &networkModel{},
			expected:     nil,
			isValid:      false,
		},
		{
			description:  "nil_storage",
			input:        &Model{},
			inputAcl:     []string{},
			inputFlavor:  &flavorModel{},
			inputStorage: nil,
			inputOptions: &optionsModel{},
			inputNetwork: &networkModel{},
			expected:     &sqlserverflex.CreateInstancePayload{},
			isValid:      false,
		},
		{
			description:  "nil_options",
			input:        &Model{},
			inputAcl:     []string{},
			inputFlavor:  &flavorModel{},
			inputStorage: &storageModel{},
			inputOptions: nil,
			inputNetwork: &networkModel{},
			expected:     &sqlserverflex.CreateInstancePayload{},
			isValid:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputAcl, tt.inputEncryption, tt.inputFlavor, tt.inputStorage, tt.inputOptions, tt.inputNetwork)
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
		description  string
		input        *Model
		inputAcl     []string
		inputFlavor  *flavorModel
		inputStorage *storageModel
		inputOptions *optionsModel
		inputNetwork *networkModel
		expected     *sqlserverflex.PartialUpdateInstancePayload
		isValid      bool
	}{
		{
			description: "default_values",
			input: &Model{
				FlavorId: types.StringValue("fid"),
			},
			inputAcl:    []string{},
			inputFlavor: &flavorModel{},
			inputStorage: &storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(34),
			},
			inputOptions: &optionsModel{},
			inputNetwork: &networkModel{},
			expected: &sqlserverflex.PartialUpdateInstancePayload{
				FlavorId: new("fid"),
				Network: &sqlserverflex.PartialUpdateInstancePayloadNetwork{
					Acl: []string{},
				},
				Storage: &sqlserverflex.StorageUpdate{
					Size: new(int64(34)),
				},
			},
			isValid: true,
		},
		{
			description: "simple_values",
			input: &Model{
				BackupSchedule: types.StringValue("schedule"),
				Name:           types.StringValue("name"),
				Replicas:       types.Int32Value(12),
				Version:        types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
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
			inputOptions: &optionsModel{},
			inputNetwork: &networkModel{},
			expected: &sqlserverflex.PartialUpdateInstancePayload{
				Network: &sqlserverflex.PartialUpdateInstancePayloadNetwork{
					Acl: []string{
						"ip_1",
						"ip_2",
					},
				},
				Storage: &sqlserverflex.StorageUpdate{
					Size: new(int64(34)),
				},
				BackupSchedule: new("schedule"),
				FlavorId:       new("flavor_id"),
				Name:           new("name"),
				Version:        new(sqlserverflex.INSTANCEVERSIONOPT__2022),
			},
			isValid: true,
		},
		{
			description: "null_fields_and_int_conversions",
			input: &Model{
				FlavorId:       types.StringValue("fid"),
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int32Value(2123456789),
				Version:        types.StringNull(),
			},
			inputAcl: []string{
				"",
			},
			inputFlavor: &flavorModel{
				Id: types.StringNull(),
			},
			inputNetwork: &networkModel{},
			inputStorage: &storageModel{
				Size: types.Int64Value(0),
			},
			inputOptions: &optionsModel{},
			expected: &sqlserverflex.PartialUpdateInstancePayload{
				Network: &sqlserverflex.PartialUpdateInstancePayloadNetwork{
					Acl: []string{
						"",
					},
				},
				Storage: &sqlserverflex.StorageUpdate{
					Size: new(int64(0)),
				},
				BackupSchedule: nil,
				FlavorId:       new("fid"),
				Name:           nil,
				Version:        nil,
			},
			isValid: true,
		},
		{
			description:  "nil_model",
			input:        nil,
			inputAcl:     []string{},
			inputFlavor:  &flavorModel{},
			inputStorage: &storageModel{},
			inputOptions: &optionsModel{},
			inputNetwork: &networkModel{},
			expected:     nil,
			isValid:      false,
		},
		{
			description: "nil_acl",
			input: &Model{
				FlavorId: types.StringValue("fid"),
			},
			inputAcl:    nil,
			inputFlavor: &flavorModel{},
			inputStorage: &storageModel{
				Size: types.Int64Value(34),
			},
			inputOptions: &optionsModel{},
			inputNetwork: &networkModel{},
			expected: &sqlserverflex.PartialUpdateInstancePayload{
				FlavorId: new("fid"),
				Network: &sqlserverflex.PartialUpdateInstancePayloadNetwork{
					Acl: []string{},
				},
				Storage: &sqlserverflex.StorageUpdate{
					Size: new(int64(34)),
				},
			},
			isValid: true,
		},
		{
			description: "nil_flavor",
			input:       &Model{},
			inputAcl:    []string{},
			inputFlavor: nil,
			inputStorage: &storageModel{
				Size: types.Int64Value(34),
			},
			inputOptions: &optionsModel{},
			inputNetwork: &networkModel{},
			expected:     nil,
			isValid:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage, tt.inputOptions, tt.inputNetwork)
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
		mockedResp      *sqlserverflex.ListFlavorsResponse
		expected        *flavorModel
		getFlavorsFails bool
		isValid         bool
	}{
		{
			"ok_flavor",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&sqlserverflex.ListFlavorsResponse{
				Flavors: []sqlserverflex.ListFlavors{
					{
						Id:          "fid-1",
						Cpu:         2,
						Description: "description",
						Memory:      8,
					},
				},
			},
			&flavorModel{
				Id:          types.StringValue("fid-1"),
				Description: types.StringValue("description"),
				CPU:         types.Int64Value(2),
				RAM:         types.Int64Value(8),
			},
			false,
			true,
		},
		{
			"ok_flavor_2",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&sqlserverflex.ListFlavorsResponse{
				Flavors: []sqlserverflex.ListFlavors{
					{
						Id:          "fid-1",
						Cpu:         2,
						Description: "description",
						Memory:      8,
					},
					{
						Id:          "fid-2",
						Cpu:         1,
						Description: "description",
						Memory:      4,
					},
				},
			},
			&flavorModel{
				Id:          types.StringValue("fid-1"),
				Description: types.StringValue("description"),
				CPU:         types.Int64Value(2),
				RAM:         types.Int64Value(8),
			},
			false,
			true,
		},
		{
			"no_matching_flavor",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&sqlserverflex.ListFlavorsResponse{
				Flavors: []sqlserverflex.ListFlavors{
					{
						Id:          "fid-1",
						Cpu:         1,
						Description: "description",
						Memory:      8,
					},
					{
						Id:          "fid-2",
						Cpu:         1,
						Description: "description",
						Memory:      4,
					},
				},
			},
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			false,
			false,
		},
		{
			"nil_response",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&sqlserverflex.ListFlavorsResponse{},
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			false,
			false,
		},
		{
			"error_response",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&sqlserverflex.ListFlavorsResponse{},
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &sqlserverflexClientMocked{
				returnError:     tt.getFlavorsFails,
				listFlavorsResp: tt.mockedResp,
			}
			model := &Model{
				ProjectId: types.StringValue("pid"),
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
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("custom-class"),
					"size":  types.Int64Value(100),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(45),
			},
			planModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("custom-class"),
					"size":  types.Int64Value(100),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(45),
			},
			expectedPlan: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("custom-class"),
					"size":  types.Int64Value(100),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(45),
			},
			warnings: 0,
		},
		{
			name: "migration_triggered_for_all_null_fields",
			configModel: &Model{
				BackupSchedule: types.StringNull(),
				Storage:        types.ObjectNull(storageTypes),
				Version:        types.StringNull(),
				ACL:            types.ListNull(types.StringType),
				Network:        types.ObjectNull(networkTypes),
				RetentionDays:  types.Int32Null(),
				Options:        types.ObjectNull(optionsTypes),
			},
			planModel: &Model{
				BackupSchedule: types.StringNull(),
				Storage:        types.ObjectNull(storageTypes),
				Version:        types.StringNull(),
				ACL:            types.ListNull(types.StringType),
				Network:        types.ObjectNull(networkTypes),
				RetentionDays:  types.Int32Null(),
				Options:        types.ObjectNull(optionsTypes),
			},
			expectedPlan: &Model{
				BackupSchedule: types.StringValue("0 0 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Value(40),
				}),
				Version:       types.StringValue(string(sqlserverflex.INSTANCEVERSION__2022)),
				ACL:           types.ListNull(types.StringType),
				Network:       types.ObjectNull(networkTypes),
				RetentionDays: types.Int32Value(30),
				Options:       types.ObjectNull(optionsTypes),
			},
			warnings: 5,
		},
		{
			name: "no_storage_size_set",
			configModel: &Model{
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Null(),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
				ACL:     types.ListNull(types.StringType),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(1),
				Options:       types.ObjectNull(optionsTypes),
			},
			planModel: &Model{
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Null(),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
				ACL:     types.ListNull(types.StringType),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(1),
				Options:       types.ObjectNull(optionsTypes),
			},
			expectedPlan: &Model{
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Value(40),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSION__2022)),
				ACL:     types.ListNull(types.StringType),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(1),
				Options:       types.ObjectNull(optionsTypes),
			},
			warnings: 1,
		},
		{
			name: "no_storage_class_set",
			configModel: &Model{
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringNull(),
					"size":  types.Int64Value(20),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
				ACL:     types.ListNull(types.StringType),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(1),
				Options:       types.ObjectNull(optionsTypes),
			},
			planModel: &Model{
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringNull(),
					"size":  types.Int64Value(20),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
				ACL:     types.ListNull(types.StringType),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(1),
				Options:       types.ObjectNull(optionsTypes),
			},
			expectedPlan: &Model{
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Value(20),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSION__2022)),
				ACL:     types.ListNull(types.StringType),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(1),
				Options:       types.ObjectNull(optionsTypes),
			},
			warnings: 1,
		},
		{
			name: "no_version_set",
			configModel: &Model{
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Value(40),
				}),
				Version: types.StringNull(),
				ACL:     types.ListNull(types.StringType),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(1),
				Options:       types.ObjectNull(optionsTypes),
			},
			planModel: &Model{
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Value(40),
				}),
				Version: types.StringNull(),
				ACL:     types.ListNull(types.StringType),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(1),
				Options:       types.ObjectNull(optionsTypes),
			},
			expectedPlan: &Model{
				BackupSchedule: types.StringValue("1 2 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Value(40),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSION__2022)),
				ACL:     types.ListNull(types.StringType),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Value(1),
				Options:       types.ObjectNull(optionsTypes),
			},
			warnings: 1,
		},
		{
			name: "retention_days_provided_in_options_no_migration",
			configModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("0 0 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Value(40),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSION__2022)),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("193.148.160.0/19")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Null(),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"edition":        types.StringValue("edition"),
					"retention_days": types.Int32Value(15),
				}),
			},
			planModel: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("0 0 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Value(40),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSION__2022)),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("193.148.160.0/19")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Null(),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"edition":        types.StringValue("edition"),
					"retention_days": types.Int32Value(15),
				}),
			},
			expectedPlan: &Model{
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringValue("0 0 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("premium-perf12-stackit"),
					"size":  types.Int64Value(40),
				}),
				Version: types.StringValue(string(sqlserverflex.INSTANCEVERSION__2022)),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"acl":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("193.148.160.0/19")}),
					"access_scope": types.StringValue(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
				}),
				RetentionDays: types.Int32Null(),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"edition":        types.StringValue("edition"),
					"retention_days": types.Int32Value(15),
				}),
			},
			warnings: 0,
		},
		{
			name: "config ",
			configModel: &Model{
				BackupSchedule: types.StringNull(),
				Storage:        types.ObjectNull(storageTypes),
				Version:        types.StringNull(),
				ACL:            types.ListNull(types.StringType),
				Network:        types.ObjectNull(networkTypes),
				RetentionDays:  types.Int32Null(),
				Options:        types.ObjectNull(optionsTypes),
			},
			planModel: &Model{
				BackupSchedule: types.StringValue("1 1 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("custom-class"),
					"size":  types.Int64Value(80),
				}),
				Version:       types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
				ACL:           types.ListNull(types.StringType),
				Network:       types.ObjectNull(networkTypes),
				RetentionDays: types.Int32Value(60),
				Options:       types.ObjectNull(optionsTypes),
			},
			expectedPlan: &Model{
				BackupSchedule: types.StringValue("1 1 * * *"),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("custom-class"),
					"size":  types.Int64Value(80),
				}),
				Version:       types.StringValue(string(sqlserverflex.INSTANCEVERSIONOPT__2022)),
				ACL:           types.ListNull(types.StringType),
				Network:       types.ObjectNull(networkTypes),
				RetentionDays: types.Int32Value(60),
				Options:       types.ObjectNull(optionsTypes),
			},
			warnings: 5,
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
		mockedResp      *sqlserverflex.ListFlavorsResponse
		expected        *sqlserverflex.ListFlavors
		getFlavorsFails bool
		isValid         bool
	}{
		{
			description: "ok_flavor_found",
			flavorId:    "fid-1",
			mockedResp: &sqlserverflex.ListFlavorsResponse{
				Flavors: []sqlserverflex.ListFlavors{
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
			expected: &sqlserverflex.ListFlavors{
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
			mockedResp: &sqlserverflex.ListFlavorsResponse{
				Flavors: []sqlserverflex.ListFlavors{
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
			client := &sqlserverflexClientMocked{
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
