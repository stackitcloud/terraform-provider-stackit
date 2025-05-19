package mongodbflex

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
					"type":                              types.StringNull(),
					"snapshot_retention_days":           types.Int64Null(),
					"daily_snapshot_retention_days":     types.Int64Null(),
					"weekly_snapshot_retention_weeks":   types.Int64Null(),
					"monthly_snapshot_retention_months": types.Int64Null(),
					"point_in_time_window_hours":        types.Int64Null(),
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
					Status:   mongodbflex.INSTANCESTATUS_READY.Ptr(),
					Storage: &mongodbflex.Storage{
						Class: utils.Ptr("class"),
						Size:  utils.Ptr(int64(78)),
					},
					Options: &map[string]string{
						"type":                           "type",
						"snapshotRetentionDays":          "5",
						"dailySnapshotRetentionDays":     "6",
						"weeklySnapshotRetentionWeeks":   "7",
						"monthlySnapshotRetentionMonths": "8",
						"pointInTimeWindowHours":         "9",
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
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int64Value(5),
					"daily_snapshot_retention_days":     types.Int64Value(6),
					"weekly_snapshot_retention_weeks":   types.Int64Value(7),
					"monthly_snapshot_retention_months": types.Int64Value(8),
					"point_in_time_window_hours":        types.Int64Value(9),
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
					Status:         mongodbflex.INSTANCESTATUS_READY.Ptr(),
					Storage:        nil,
					Options: &map[string]string{
						"type":                           "type",
						"snapshotRetentionDays":          "5",
						"dailySnapshotRetentionDays":     "6",
						"weeklySnapshotRetentionWeeks":   "7",
						"monthlySnapshotRetentionMonths": "8",
						"pointInTimeWindowHours":         "9",
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
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int64Value(5),
					"daily_snapshot_retention_days":     types.Int64Value(6),
					"weekly_snapshot_retention_weeks":   types.Int64Value(7),
					"monthly_snapshot_retention_months": types.Int64Value(8),
					"point_in_time_window_hours":        types.Int64Value(9),
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
					Status:         mongodbflex.INSTANCESTATUS_READY.Ptr(),
					Storage:        nil,
					Options: &map[string]string{
						"type":                           "type",
						"snapshotRetentionDays":          "5",
						"dailySnapshotRetentionDays":     "6",
						"weeklySnapshotRetentionWeeks":   "7",
						"monthlySnapshotRetentionMonths": "8",
						"pointInTimeWindowHours":         "9",
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
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int64Value(5),
					"daily_snapshot_retention_days":     types.Int64Value(6),
					"weekly_snapshot_retention_weeks":   types.Int64Value(7),
					"monthly_snapshot_retention_months": types.Int64Value(8),
					"point_in_time_window_hours":        types.Int64Value(9),
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

func TestMapOptions(t *testing.T) {
	tests := []struct {
		description string
		model       *Model
		options     *optionsModel
		backup      *mongodbflex.BackupSchedule
		expected    *Model
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&optionsModel{},
			nil,
			&Model{
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringNull(),
					"snapshot_retention_days":           types.Int64Null(),
					"daily_snapshot_retention_days":     types.Int64Null(),
					"weekly_snapshot_retention_weeks":   types.Int64Null(),
					"monthly_snapshot_retention_months": types.Int64Null(),
					"point_in_time_window_hours":        types.Int64Null(),
				}),
			},
			true,
		},
		{
			"simple_values",
			&Model{},
			&optionsModel{
				Type: types.StringValue("type"),
			},
			&mongodbflex.BackupSchedule{
				SnapshotRetentionDays:          utils.Ptr(int64(1)),
				DailySnapshotRetentionDays:     utils.Ptr(int64(2)),
				WeeklySnapshotRetentionWeeks:   utils.Ptr(int64(3)),
				MonthlySnapshotRetentionMonths: utils.Ptr(int64(4)),
				PointInTimeWindowHours:         utils.Ptr(int64(5)),
			},
			&Model{
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int64Value(1),
					"daily_snapshot_retention_days":     types.Int64Value(2),
					"weekly_snapshot_retention_weeks":   types.Int64Value(3),
					"monthly_snapshot_retention_months": types.Int64Value(4),
					"point_in_time_window_hours":        types.Int64Value(5),
				}),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapOptions(tt.model, tt.options, tt.backup)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.model, tt.expected, cmpopts.IgnoreFields(Model{}, "ACL", "Flavor", "Replicas", "Storage", "Version"))
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

func TestToUpdateBackupScheduleOptionsPayload(t *testing.T) {
	tests := []struct {
		description       string
		model             *Model
		configuredOptions *optionsModel
		expected          *mongodbflex.UpdateBackupSchedulePayload
		isValid           bool
	}{
		{
			"default_values",
			&Model{},
			&optionsModel{},
			&mongodbflex.UpdateBackupSchedulePayload{
				BackupSchedule:                 nil,
				SnapshotRetentionDays:          nil,
				DailySnapshotRetentionDays:     nil,
				WeeklySnapshotRetentionWeeks:   nil,
				MonthlySnapshotRetentionMonths: nil,
				PointInTimeWindowHours:         nil,
			},
			true,
		},
		{
			"config values override current values in model",
			&Model{
				BackupSchedule: types.StringValue("schedule"),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int64Value(1),
					"daily_snapshot_retention_days":     types.Int64Value(2),
					"weekly_snapshot_retention_weeks":   types.Int64Value(3),
					"monthly_snapshot_retention_months": types.Int64Value(4),
					"point_in_time_window_hours":        types.Int64Value(5),
				}),
			},
			&optionsModel{
				SnapshotRetentionDays:          types.Int64Value(6),
				DailySnapshotRetentionDays:     types.Int64Value(7),
				WeeklySnapshotRetentionWeeks:   types.Int64Value(8),
				MonthlySnapshotRetentionMonths: types.Int64Value(9),
				PointInTimeWindowHours:         types.Int64Value(10),
			},
			&mongodbflex.UpdateBackupSchedulePayload{
				BackupSchedule:                 utils.Ptr("schedule"),
				SnapshotRetentionDays:          utils.Ptr(int64(6)),
				DailySnapshotRetentionDays:     utils.Ptr(int64(7)),
				WeeklySnapshotRetentionWeeks:   utils.Ptr(int64(8)),
				MonthlySnapshotRetentionMonths: utils.Ptr(int64(9)),
				PointInTimeWindowHours:         utils.Ptr(int64(10)),
			},
			true,
		},
		{
			"current values in model fill in missing values in config",
			&Model{
				BackupSchedule: types.StringValue("schedule"),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int64Value(1),
					"daily_snapshot_retention_days":     types.Int64Value(2),
					"weekly_snapshot_retention_weeks":   types.Int64Value(3),
					"monthly_snapshot_retention_months": types.Int64Value(4),
					"point_in_time_window_hours":        types.Int64Value(5),
				}),
			},
			&optionsModel{
				SnapshotRetentionDays:          types.Int64Value(6),
				DailySnapshotRetentionDays:     types.Int64Value(7),
				WeeklySnapshotRetentionWeeks:   types.Int64Null(),
				MonthlySnapshotRetentionMonths: types.Int64Null(),
				PointInTimeWindowHours:         types.Int64Null(),
			},
			&mongodbflex.UpdateBackupSchedulePayload{
				BackupSchedule:                 utils.Ptr("schedule"),
				SnapshotRetentionDays:          utils.Ptr(int64(6)),
				DailySnapshotRetentionDays:     utils.Ptr(int64(7)),
				WeeklySnapshotRetentionWeeks:   utils.Ptr(int64(3)),
				MonthlySnapshotRetentionMonths: utils.Ptr(int64(4)),
				PointInTimeWindowHours:         utils.Ptr(int64(5)),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				BackupSchedule: types.StringNull(),
			},
			&optionsModel{
				SnapshotRetentionDays:          types.Int64Null(),
				DailySnapshotRetentionDays:     types.Int64Null(),
				WeeklySnapshotRetentionWeeks:   types.Int64Null(),
				MonthlySnapshotRetentionMonths: types.Int64Null(),
				PointInTimeWindowHours:         types.Int64Null(),
			},
			&mongodbflex.UpdateBackupSchedulePayload{
				BackupSchedule:                 nil,
				SnapshotRetentionDays:          nil,
				DailySnapshotRetentionDays:     nil,
				WeeklySnapshotRetentionWeeks:   nil,
				MonthlySnapshotRetentionMonths: nil,
				PointInTimeWindowHours:         nil,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdateBackupScheduleOptionsPayload(context.Background(), tt.model, tt.configuredOptions)
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
