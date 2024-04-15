package mongodbflex

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
)

type mongoDBFlexClientMocked struct {
	returnError     bool
	listFlavorsResp *mongodbflex.ListFlavorsResponse
}

func (c *mongoDBFlexClientMocked) ListFlavorsExecute(_ context.Context, _ string) (*mongodbflex.ListFlavorsResponse, error) {
	if c.returnError {
		return nil, fmt.Errorf("get flavors failed")
	}

	return c.listFlavorsResp, nil
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *mongodbflex.GetInstanceResponse
		flavor      *flavorModel
		storage     *storageModel
		options     *optionsModel
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&mongodbflex.GetInstanceResponse{
				Item: &mongodbflex.Instance{},
			},
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
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
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type": types.StringNull(),
				}),
				Version: types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&mongodbflex.GetInstanceResponse{
				Item: &mongodbflex.Instance{
					Acl: &mongodbflex.ACL{
						Items: &[]string{
							"ip1",
							"ip2",
							"",
						},
					},
					BackupSchedule: utils.Ptr("schedule"),
					Flavor: &mongodbflex.Flavor{
						Cpu:         utils.Ptr(int64(12)),
						Description: utils.Ptr("description"),
						Id:          utils.Ptr("flavor_id"),
						Memory:      utils.Ptr(int64(34)),
					},
					Id:       utils.Ptr("iid"),
					Name:     utils.Ptr("name"),
					Replicas: utils.Ptr(int64(56)),
					Status:   utils.Ptr("status"),
					Storage: &mongodbflex.Storage{
						Class: utils.Ptr("class"),
						Size:  utils.Ptr(int64(78)),
					},
					Options: &map[string]string{
						"type": "type",
					},
					Version: utils.Ptr("version"),
				},
			},
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
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
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type": types.StringValue("type"),
				}),
				Version: types.StringValue("version"),
			},
			true,
		},
		{
			"simple_values_no_flavor_and_storage",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&mongodbflex.GetInstanceResponse{
				Item: &mongodbflex.Instance{
					Acl: &mongodbflex.ACL{
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
					Replicas:       utils.Ptr(int64(56)),
					Status:         utils.Ptr("status"),
					Storage:        nil,
					Options: &map[string]string{
						"type": "type",
					},
					Version: utils.Ptr("version"),
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
			&optionsModel{
				Type: types.StringValue("type"),
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
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type": types.StringValue("type"),
				}),
				Version: types.StringValue("version"),
			},
			true,
		},
		{
			"acls_unordered",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip2"),
					types.StringValue(""),
					types.StringValue("ip1"),
				}),
			},
			&mongodbflex.GetInstanceResponse{
				Item: &mongodbflex.Instance{
					Acl: &mongodbflex.ACL{
						Items: &[]string{
							"",
							"ip1",
							"ip2",
						},
					},
					BackupSchedule: utils.Ptr("schedule"),
					Flavor:         nil,
					Id:             utils.Ptr("iid"),
					Name:           utils.Ptr("name"),
					Replicas:       utils.Ptr(int64(56)),
					Status:         utils.Ptr("status"),
					Storage:        nil,
					Options: &map[string]string{
						"type": "type",
					},
					Version: utils.Ptr("version"),
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
			&optionsModel{
				Type: types.StringValue("type"),
			},
			Model{
				Id:         types.StringValue("pid,iid"),
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
				Replicas: types.Int64Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type": types.StringValue("type"),
				}),
				Version: types.StringValue("version"),
			},
			true,
		},
		{
			"nil_response",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			nil,
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&mongodbflex.GetInstanceResponse{},
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, tt.flavor, tt.storage, tt.options)
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
		description  string
		input        *Model
		inputAcl     []string
		inputFlavor  *flavorModel
		inputStorage *storageModel
		inputOptions *optionsModel
		expected     *mongodbflex.CreateInstancePayload
		isValid      bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			&mongodbflex.CreateInstancePayload{
				Acl: &mongodbflex.ACL{
					Items: &[]string{},
				},
				Storage: &mongodbflex.Storage{},
				Options: &map[string]string{},
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
			&optionsModel{
				Type: types.StringValue("type"),
			},
			&mongodbflex.CreateInstancePayload{
				Acl: &mongodbflex.ACL{
					Items: &[]string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: utils.Ptr("schedule"),
				FlavorId:       utils.Ptr("flavor_id"),
				Name:           utils.Ptr("name"),
				Replicas:       utils.Ptr(int64(12)),
				Storage: &mongodbflex.Storage{
					Class: utils.Ptr("class"),
					Size:  utils.Ptr(int64(34)),
				},
				Options: &map[string]string{"type": "type"},
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
			&optionsModel{
				Type: types.StringNull(),
			},
			&mongodbflex.CreateInstancePayload{
				Acl: &mongodbflex.ACL{
					Items: &[]string{
						"",
					},
				},
				BackupSchedule: nil,
				FlavorId:       nil,
				Name:           nil,
				Replicas:       utils.Ptr(int64(2123456789)),
				Storage: &mongodbflex.Storage{
					Class: nil,
					Size:  nil,
				},
				Options: &map[string]string{},
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
			&optionsModel{},
			nil,
			false,
		},
		{
			"nil_acl",
			&Model{},
			nil,
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			nil,
			false,
		},
		{
			"nil_flavor",
			&Model{},
			[]string{},
			nil,
			&storageModel{},
			&optionsModel{},
			nil,
			false,
		},
		{
			"nil_storage",
			&Model{},
			[]string{},
			&flavorModel{},
			nil,
			&optionsModel{},
			nil,
			false,
		},
		{
			"nil_options",
			&Model{},
			[]string{},
			&flavorModel{},
			&storageModel{},
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage, tt.inputOptions)
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
		expected     *mongodbflex.PartialUpdateInstancePayload
		isValid      bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			&mongodbflex.PartialUpdateInstancePayload{
				Acl: &mongodbflex.ACL{
					Items: &[]string{},
				},
				Storage: &mongodbflex.Storage{},
				Options: &map[string]string{},
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
			&optionsModel{
				Type: types.StringValue("type"),
			},
			&mongodbflex.PartialUpdateInstancePayload{
				Acl: &mongodbflex.ACL{
					Items: &[]string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: utils.Ptr("schedule"),
				FlavorId:       utils.Ptr("flavor_id"),
				Name:           utils.Ptr("name"),
				Replicas:       utils.Ptr(int64(12)),
				Storage: &mongodbflex.Storage{
					Class: utils.Ptr("class"),
					Size:  utils.Ptr(int64(34)),
				},
				Options: &map[string]string{"type": "type"},
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
			&optionsModel{
				Type: types.StringNull(),
			},
			&mongodbflex.PartialUpdateInstancePayload{
				Acl: &mongodbflex.ACL{
					Items: &[]string{
						"",
					},
				},
				BackupSchedule: nil,
				FlavorId:       nil,
				Name:           nil,
				Replicas:       utils.Ptr(int64(2123456789)),
				Storage: &mongodbflex.Storage{
					Class: nil,
					Size:  nil,
				},
				Options: &map[string]string{},
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
			&optionsModel{},
			nil,
			false,
		},
		{
			"nil_acl",
			&Model{},
			nil,
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			nil,
			false,
		},
		{
			"nil_flavor",
			&Model{},
			[]string{},
			nil,
			&storageModel{},
			&optionsModel{},
			nil,
			false,
		},
		{
			"nil_storage",
			&Model{},
			[]string{},
			&flavorModel{},
			nil,
			&optionsModel{},
			nil,
			false,
		},
		{
			"nil_options",
			&Model{},
			[]string{},
			&flavorModel{},
			&storageModel{},
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage, tt.inputOptions)
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
		mockedResp      *mongodbflex.ListFlavorsResponse
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
			&mongodbflex.ListFlavorsResponse{
				Flavors: &[]mongodbflex.HandlersInfraFlavor{
					{
						Id:          utils.Ptr("fid-1"),
						Cpu:         utils.Ptr(int64(2)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(8)),
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
			&mongodbflex.ListFlavorsResponse{
				Flavors: &[]mongodbflex.HandlersInfraFlavor{
					{
						Id:          utils.Ptr("fid-1"),
						Cpu:         utils.Ptr(int64(2)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(8)),
					},
					{
						Id:          utils.Ptr("fid-2"),
						Cpu:         utils.Ptr(int64(1)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(4)),
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
			&mongodbflex.ListFlavorsResponse{
				Flavors: &[]mongodbflex.HandlersInfraFlavor{
					{
						Id:          utils.Ptr("fid-1"),
						Cpu:         utils.Ptr(int64(1)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(8)),
					},
					{
						Id:          utils.Ptr("fid-2"),
						Cpu:         utils.Ptr(int64(1)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(4)),
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
			&mongodbflex.ListFlavorsResponse{},
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
			&mongodbflex.ListFlavorsResponse{},
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
			client := &mongoDBFlexClientMocked{
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

func TestSimplifyBackupSchedule(t *testing.T) {
	tests := []struct {
		description string
		input       string
		expected    string
	}{
		{
			"simple schedule",
			"0 0 * * *",
			"0 0 * * *",
		},
		{
			"schedule with leading zeros",
			"00 00 * * *",
			"0 0 * * *",
		},
		{
			"schedule with leading zeros 2",
			"00 001 * * *",
			"0 1 * * *",
		},
		{
			"schedule with leading zeros 3",
			"00 0010 * * *",
			"0 10 * * *",
		},
		{
			"simple schedule with slash",
			"0 0/6 * * *",
			"0 0/6 * * *",
		},
		{
			"schedule with leading zeros and slash",
			"00 00/6 * * *",
			"0 0/6 * * *",
		},
		{
			"schedule with leading zeros and slash 2",
			"00 001/06 * * *",
			"0 1/6 * * *",
		},
		{
			"simple schedule with comma",
			"0 10,15 * * *",
			"0 10,15 * * *",
		},
		{
			"schedule with leading zeros and comma",
			"0 010,0015 * * *",
			"0 10,15 * * *",
		},
		{
			"simple schedule with comma and slash",
			"0 0-11/10 * * *",
			"0 0-11/10 * * *",
		},
		{
			"schedule with leading zeros, comma, and slash",
			"00 000-011/010 * * *",
			"0 0-11/10 * * *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := simplifyBackupSchedule(tt.input)
			if output != tt.expected {
				t.Fatalf("Data does not match: %s", output)
			}
		})
	}
}
