package postgresflex

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex"
)

type postgresFlexClientMocked struct {
	returnError    bool
	getFlavorsResp *postgresflex.FlavorsResponse
}

func (c *postgresFlexClientMocked) GetFlavorsExecute(_ context.Context, _ string) (*postgresflex.FlavorsResponse, error) {
	if c.returnError {
		return nil, fmt.Errorf("get flavors failed")
	}

	return c.getFlavorsResp, nil
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *postgresflex.InstanceResponse
		flavor      *flavorModel
		storage     *storageModel
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&postgresflex.InstanceResponse{
				Item: &postgresflex.InstanceSingleInstance{},
			},
			&flavorModel{},
			&storageModel{},
			Model{
				Id:             types.StringValue("pid,iid"),
				InstanceId:     types.StringValue("iid"),
				ProjectId:      types.StringValue("pid"),
				Name:           types.StringNull(),
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringNull(),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Null(),
					"ram":         types.Int64Null(),
				}),
				Replicas: types.Int64Null(),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringNull(),
					"size":  types.Int64Null(),
				}),
				Version: types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&postgresflex.InstanceResponse{
				Item: &postgresflex.InstanceSingleInstance{
					Acl: &postgresflex.InstanceAcl{
						Items: &[]string{
							"ip1",
							"ip2",
							"",
						},
					},
					BackupSchedule: utils.Ptr("schedule"),
					Flavor: &postgresflex.InstanceFlavor{
						Cpu:         utils.Ptr(int32(12)),
						Description: utils.Ptr("description"),
						Id:          utils.Ptr("flavor_id"),
						Memory:      utils.Ptr(int32(34)),
					},
					Id:       utils.Ptr("iid"),
					Name:     utils.Ptr("name"),
					Replicas: utils.Ptr(int32(56)),
					Status:   utils.Ptr("status"),
					Storage: &postgresflex.InstanceStorage{
						Class: utils.Ptr("class"),
						Size:  utils.Ptr(int32(78)),
					},
					Version: utils.Ptr("version"),
				},
			},
			&flavorModel{},
			&storageModel{},
			Model{
				Id:         types.StringValue("pid,iid"),
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
					"id":          types.StringValue("flavor_id"),
					"description": types.StringValue("description"),
					"cpu":         types.Int64Value(12),
					"ram":         types.Int64Value(34),
				}),
				Replicas: types.Int64Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Version: types.StringValue("version"),
			},
			true,
		},
		{
			"simple_values_no_flavor_and_storage",
			&postgresflex.InstanceResponse{
				Item: &postgresflex.InstanceSingleInstance{
					Acl: &postgresflex.InstanceAcl{
						Items: &[]string{
							"ip1",
							"ip2",
							"",
						},
					},
					BackupSchedule: utils.Ptr("schedule"),
					Flavor:         nil,
					Id:             utils.Ptr("iid"),
					Name:           utils.Ptr("name"),
					Replicas:       utils.Ptr(int32(56)),
					Status:         utils.Ptr("status"),
					Storage:        nil,
					Version:        utils.Ptr("version"),
				},
			},
			&flavorModel{
				CPU: types.Int64Value(12),
				RAM: types.Int64Value(34),
			},
			&storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(78),
			},
			Model{
				Id:         types.StringValue("pid,iid"),
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
				Replicas: types.Int64Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Version: types.StringValue("version"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			&flavorModel{},
			&storageModel{},
			Model{},
			false,
		},
		{
			"no_resource_id",
			&postgresflex.InstanceResponse{},
			&flavorModel{},
			&storageModel{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(tt.input, state, tt.flavor, tt.storage)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description  string
		input        *Model
		inputAcl     []string
		inputFlavor  *flavorModel
		inputStorage *storageModel
		expected     *postgresflex.CreateInstancePayload
		isValid      bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&flavorModel{},
			&storageModel{},
			&postgresflex.CreateInstancePayload{
				Acl: &postgresflex.InstanceAcl{
					Items: &[]string{},
				},
				Storage: &postgresflex.InstanceStorage{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				BackupSchedule: types.StringValue("schedule"),
				Name:           types.StringValue("name"),
				Replicas:       types.Int64Value(12),
				Version:        types.StringValue("version"),
			},
			[]string{
				"ip_1",
				"ip_2",
			},
			&flavorModel{
				Id: types.StringValue("flavor_id"),
			},
			&storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(34),
			},
			&postgresflex.CreateInstancePayload{
				Acl: &postgresflex.InstanceAcl{
					Items: &[]string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: utils.Ptr("schedule"),
				FlavorId:       utils.Ptr("flavor_id"),
				Name:           utils.Ptr("name"),
				Replicas:       utils.Ptr(int32(12)),
				Storage: &postgresflex.InstanceStorage{
					Class: utils.Ptr("class"),
					Size:  utils.Ptr(int32(34)),
				},
				Version: utils.Ptr("version"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int64Value(2123456789),
				Version:        types.StringNull(),
			},
			[]string{
				"",
			},
			&flavorModel{
				Id: types.StringNull(),
			},
			&storageModel{
				Class: types.StringNull(),
				Size:  types.Int64Null(),
			},
			&postgresflex.CreateInstancePayload{
				Acl: &postgresflex.InstanceAcl{
					Items: &[]string{
						"",
					},
				},
				BackupSchedule: nil,
				FlavorId:       nil,
				Name:           nil,
				Replicas:       utils.Ptr(int32(2123456789)),
				Storage: &postgresflex.InstanceStorage{
					Class: nil,
					Size:  nil,
				},
				Version: nil,
			},
			true,
		},
		{
			"nil_model",
			nil,
			[]string{},
			&flavorModel{},
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_acl",
			&Model{},
			nil,
			&flavorModel{},
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_flavor",
			&Model{},
			[]string{},
			nil,
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_storage",
			&Model{},
			[]string{},
			&flavorModel{},
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage)
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
		expected     *postgresflex.UpdateInstancePayload
		isValid      bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&flavorModel{},
			&storageModel{},
			&postgresflex.UpdateInstancePayload{
				Acl: &postgresflex.InstanceAcl{
					Items: &[]string{},
				},
				Storage: &postgresflex.InstanceStorage{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				BackupSchedule: types.StringValue("schedule"),
				Name:           types.StringValue("name"),
				Replicas:       types.Int64Value(12),
				Version:        types.StringValue("version"),
			},
			[]string{
				"ip_1",
				"ip_2",
			},
			&flavorModel{
				Id: types.StringValue("flavor_id"),
			},
			&storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(34),
			},
			&postgresflex.UpdateInstancePayload{
				Acl: &postgresflex.InstanceAcl{
					Items: &[]string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: utils.Ptr("schedule"),
				FlavorId:       utils.Ptr("flavor_id"),
				Name:           utils.Ptr("name"),
				Replicas:       utils.Ptr(int32(12)),
				Storage: &postgresflex.InstanceStorage{
					Class: utils.Ptr("class"),
					Size:  utils.Ptr(int32(34)),
				},
				Version: utils.Ptr("version"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int64Value(2123456789),
				Version:        types.StringNull(),
			},
			[]string{
				"",
			},
			&flavorModel{
				Id: types.StringNull(),
			},
			&storageModel{
				Class: types.StringNull(),
				Size:  types.Int64Null(),
			},
			&postgresflex.UpdateInstancePayload{
				Acl: &postgresflex.InstanceAcl{
					Items: &[]string{
						"",
					},
				},
				BackupSchedule: nil,
				FlavorId:       nil,
				Name:           nil,
				Replicas:       utils.Ptr(int32(2123456789)),
				Storage: &postgresflex.InstanceStorage{
					Class: nil,
					Size:  nil,
				},
				Version: nil,
			},
			true,
		},
		{
			"nil_model",
			nil,
			[]string{},
			&flavorModel{},
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_acl",
			&Model{},
			nil,
			&flavorModel{},
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_flavor",
			&Model{},
			[]string{},
			nil,
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_storage",
			&Model{},
			[]string{},
			&flavorModel{},
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage)
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
		mockedResp      *postgresflex.FlavorsResponse
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
			&postgresflex.FlavorsResponse{
				Flavors: &[]postgresflex.InstanceFlavor{
					{
						Id:          utils.Ptr("fid-1"),
						Cpu:         utils.Ptr(int32(2)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int32(8)),
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
			&postgresflex.FlavorsResponse{
				Flavors: &[]postgresflex.InstanceFlavor{
					{
						Id:          utils.Ptr("fid-1"),
						Cpu:         utils.Ptr(int32(2)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int32(8)),
					},
					{
						Id:          utils.Ptr("fid-2"),
						Cpu:         utils.Ptr(int32(1)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int32(4)),
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
			&postgresflex.FlavorsResponse{
				Flavors: &[]postgresflex.InstanceFlavor{
					{
						Id:          utils.Ptr("fid-1"),
						Cpu:         utils.Ptr(int32(1)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int32(8)),
					},
					{
						Id:          utils.Ptr("fid-2"),
						Cpu:         utils.Ptr(int32(1)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int32(4)),
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
			&postgresflex.FlavorsResponse{},
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
			&postgresflex.FlavorsResponse{},
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
			client := &postgresFlexClientMocked{
				returnError:    tt.getFlavorsFails,
				getFlavorsResp: tt.mockedResp,
			}
			model := Model{
				ProjectId: types.StringValue("pid"),
			}
			flavorModel := &flavorModel{
				CPU: tt.inputFlavor.CPU,
				RAM: tt.inputFlavor.RAM,
			}
			err := loadFlavorId(context.Background(), client, &model, flavorModel)
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
