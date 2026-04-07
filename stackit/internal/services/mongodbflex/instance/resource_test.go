package mongodbflex

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mongodbflex "github.com/stackitcloud/stackit-sdk-go/services/mongodbflex/v2api"
)

const (
	testRegion = "eu02"
)

var (
	projectId  = uuid.NewString()
	instanceId = uuid.NewString()
)

type mongoDBFlexClientMocked struct {
	returnError     bool
	listFlavorsResp *mongodbflex.ListFlavorsResponse
	listFlavorsReq  mongodbflex.ApiListFlavorsRequest
}

func (c *mongoDBFlexClientMocked) ListFlavors(_ context.Context, _, _ string) mongodbflex.ApiListFlavorsRequest {
	return c.listFlavorsReq
}

func (c *mongoDBFlexClientMocked) ListFlavorsExecute(_ mongodbflex.ApiListFlavorsRequest) (*mongodbflex.ListFlavorsResponse, error) {
	if c.returnError {
		return nil, fmt.Errorf("get flavors failed")
	}

	return c.listFlavorsResp, nil
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *mongodbflex.InstanceResponse
		flavor      *flavorModel
		storage     *storageModel
		options     *optionsModel
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
			},
			&mongodbflex.InstanceResponse{
				Item: &mongodbflex.Instance{},
			},
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			testRegion,
			Model{
				Id:             types.StringValue(fmt.Sprintf("%s,%s,%s", projectId, testRegion, instanceId)),
				InstanceId:     types.StringValue(instanceId),
				ProjectId:      types.StringValue(projectId),
				Name:           types.StringNull(),
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringNull(),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int32Null(),
					"ram":         types.Int32Null(),
				}),
				Replicas: types.Int32Null(),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringNull(),
					"size":  types.Int64Null(),
				}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringNull(),
					"snapshot_retention_days":           types.Int32Null(),
					"daily_snapshot_retention_days":     types.Int32Null(),
					"weekly_snapshot_retention_weeks":   types.Int32Null(),
					"monthly_snapshot_retention_months": types.Int32Null(),
					"point_in_time_window_hours":        types.Int32Null(),
				}),
				Version: types.StringNull(),
				Region:  types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			Model{
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
			},
			&mongodbflex.InstanceResponse{
				Item: &mongodbflex.Instance{
					Acl: &mongodbflex.ACL{
						Items: []string{
							"ip1",
							"ip2",
							"",
						},
					},
					BackupSchedule: new("schedule"),
					Flavor: &mongodbflex.Flavor{
						Cpu:         new(int32(12)),
						Description: new("description"),
						Id:          new("flavor_id"),
						Memory:      new(int32(34)),
					},
					Id:       new(instanceId),
					Name:     new("name"),
					Replicas: new(int32(56)),
					Status:   new("READY"),
					Storage: &mongodbflex.Storage{
						Class: new("class"),
						Size:  new(int64(78)),
					},
					Options: &map[string]string{
						"type":                           "type",
						"snapshotRetentionDays":          "5",
						"dailySnapshotRetentionDays":     "6",
						"weeklySnapshotRetentionWeeks":   "7",
						"monthlySnapshotRetentionMonths": "8",
						"pointInTimeWindowHours":         "9",
					},
					Version: new("version"),
				},
			},
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			testRegion,
			Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s", projectId, testRegion, instanceId)),
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
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
					"cpu":         types.Int32Value(12),
					"ram":         types.Int32Value(34),
				}),
				Replicas: types.Int32Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int32Value(5),
					"daily_snapshot_retention_days":     types.Int32Value(6),
					"weekly_snapshot_retention_weeks":   types.Int32Value(7),
					"monthly_snapshot_retention_months": types.Int32Value(8),
					"point_in_time_window_hours":        types.Int32Value(9),
				}),
				Region:  types.StringValue(testRegion),
				Version: types.StringValue("version"),
			},
			true,
		},
		{
			"simple_values_no_flavor_and_storage",
			Model{
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
			},
			&mongodbflex.InstanceResponse{
				Item: &mongodbflex.Instance{
					Acl: &mongodbflex.ACL{
						Items: []string{
							"ip1",
							"ip2",
							"",
						},
					},
					BackupSchedule: new("schedule"),
					Flavor:         nil,
					Id:             new(instanceId),
					Name:           new("name"),
					Replicas:       new(int32(56)),
					Status:         new("READY"),
					Storage:        nil,
					Options: &map[string]string{
						"type":                           "type",
						"snapshotRetentionDays":          "5",
						"dailySnapshotRetentionDays":     "6",
						"weeklySnapshotRetentionWeeks":   "7",
						"monthlySnapshotRetentionMonths": "8",
						"pointInTimeWindowHours":         "9",
					},
					Version: new("version"),
				},
			},
			&flavorModel{
				CPU: types.Int32Value(12),
				RAM: types.Int32Value(34),
			},
			&storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(78),
			},
			&optionsModel{
				Type: types.StringValue("type"),
			},
			testRegion,
			Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s", projectId, testRegion, instanceId)),
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
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
					"cpu":         types.Int32Value(12),
					"ram":         types.Int32Value(34),
				}),
				Replicas: types.Int32Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int32Value(5),
					"daily_snapshot_retention_days":     types.Int32Value(6),
					"weekly_snapshot_retention_weeks":   types.Int32Value(7),
					"monthly_snapshot_retention_months": types.Int32Value(8),
					"point_in_time_window_hours":        types.Int32Value(9),
				}),
				Region:  types.StringValue(testRegion),
				Version: types.StringValue("version"),
			},
			true,
		},
		{
			"acls_unordered",
			Model{
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip2"),
					types.StringValue(""),
					types.StringValue("ip1"),
				}),
			},
			&mongodbflex.InstanceResponse{
				Item: &mongodbflex.Instance{
					Acl: &mongodbflex.ACL{
						Items: []string{
							"",
							"ip1",
							"ip2",
						},
					},
					BackupSchedule: new("schedule"),
					Flavor:         nil,
					Id:             new(instanceId),
					Name:           new("name"),
					Replicas:       new(int32(56)),
					Status:         new("READY"),
					Storage:        nil,
					Options: &map[string]string{
						"type":                           "type",
						"snapshotRetentionDays":          "5",
						"dailySnapshotRetentionDays":     "6",
						"weeklySnapshotRetentionWeeks":   "7",
						"monthlySnapshotRetentionMonths": "8",
						"pointInTimeWindowHours":         "9",
					},
					Version: new("version"),
				},
			},
			&flavorModel{
				CPU: types.Int32Value(12),
				RAM: types.Int32Value(34),
			},
			&storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(78),
			},
			&optionsModel{
				Type: types.StringValue("type"),
			},
			testRegion,
			Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s", projectId, testRegion, instanceId)),
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
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
					"cpu":         types.Int32Value(12),
					"ram":         types.Int32Value(34),
				}),
				Replicas: types.Int32Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int32Value(5),
					"daily_snapshot_retention_days":     types.Int32Value(6),
					"weekly_snapshot_retention_weeks":   types.Int32Value(7),
					"monthly_snapshot_retention_months": types.Int32Value(8),
					"point_in_time_window_hours":        types.Int32Value(9),
				}),
				Region:  types.StringValue(testRegion),
				Version: types.StringValue("version"),
			},
			true,
		},
		{
			"nil_response",
			Model{
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
			},
			nil,
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				InstanceId: types.StringValue(instanceId),
				ProjectId:  types.StringValue(projectId),
			},
			&mongodbflex.InstanceResponse{},
			&flavorModel{},
			&storageModel{},
			&optionsModel{},
			testRegion,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, tt.flavor, tt.storage, tt.options, tt.region)
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
					"snapshot_retention_days":           types.Int32Null(),
					"daily_snapshot_retention_days":     types.Int32Null(),
					"weekly_snapshot_retention_weeks":   types.Int32Null(),
					"monthly_snapshot_retention_months": types.Int32Null(),
					"point_in_time_window_hours":        types.Int32Null(),
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
				SnapshotRetentionDays:          new(int32(1)),
				DailySnapshotRetentionDays:     new(int32(2)),
				WeeklySnapshotRetentionWeeks:   new(int32(3)),
				MonthlySnapshotRetentionMonths: new(int32(4)),
				PointInTimeWindowHours:         new(int32(5)),
			},
			&Model{
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int32Value(1),
					"daily_snapshot_retention_days":     types.Int32Value(2),
					"weekly_snapshot_retention_weeks":   types.Int32Value(3),
					"monthly_snapshot_retention_months": types.Int32Value(4),
					"point_in_time_window_hours":        types.Int32Value(5),
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
				Acl: mongodbflex.ACL{
					Items: []string{},
				},
				Storage: mongodbflex.Storage{},
				Options: map[string]string{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				BackupSchedule: types.StringValue("schedule"),
				Name:           types.StringValue("name"),
				Replicas:       types.Int32Value(12),
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
				Acl: mongodbflex.ACL{
					Items: []string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: "schedule",
				FlavorId:       "flavor_id",
				Name:           "name",
				Replicas:       int32(12),
				Storage: mongodbflex.Storage{
					Class: new("class"),
					Size:  new(int64(34)),
				},
				Options: map[string]string{"type": "type"},
				Version: "version",
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int32Value(2123456789),
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
				Acl: mongodbflex.ACL{
					Items: []string{
						"",
					},
				},
				BackupSchedule: "",
				FlavorId:       "",
				Name:           "",
				Replicas:       int32(2123456789),
				Storage: mongodbflex.Storage{
					Class: nil,
					Size:  nil,
				},
				Options: map[string]string{},
				Version: "",
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
					Items: []string{},
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
				Replicas:       types.Int32Value(12),
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
					Items: []string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: new("schedule"),
				FlavorId:       new("flavor_id"),
				Name:           new("name"),
				Replicas:       new(int32(12)),
				Storage: &mongodbflex.Storage{
					Class: new("class"),
					Size:  new(int64(34)),
				},
				Options: &map[string]string{"type": "type"},
				Version: new("version"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int32Value(2123456789),
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
					Items: []string{
						"",
					},
				},
				BackupSchedule: nil,
				FlavorId:       nil,
				Name:           nil,
				Replicas:       new(int32(2123456789)),
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
					"snapshot_retention_days":           types.Int32Value(1),
					"daily_snapshot_retention_days":     types.Int32Value(2),
					"weekly_snapshot_retention_weeks":   types.Int32Value(3),
					"monthly_snapshot_retention_months": types.Int32Value(4),
					"point_in_time_window_hours":        types.Int32Value(5),
				}),
			},
			&optionsModel{
				SnapshotRetentionDays:          types.Int32Value(6),
				DailySnapshotRetentionDays:     types.Int32Value(7),
				WeeklySnapshotRetentionWeeks:   types.Int32Value(8),
				MonthlySnapshotRetentionMonths: types.Int32Value(9),
				PointInTimeWindowHours:         types.Int32Value(10),
			},
			&mongodbflex.UpdateBackupSchedulePayload{
				BackupSchedule:                 new("schedule"),
				SnapshotRetentionDays:          new(int32(6)),
				DailySnapshotRetentionDays:     new(int32(7)),
				WeeklySnapshotRetentionWeeks:   new(int32(8)),
				MonthlySnapshotRetentionMonths: new(int32(9)),
				PointInTimeWindowHours:         new(int32(10)),
			},
			true,
		},
		{
			"current values in model fill in missing values in config",
			&Model{
				BackupSchedule: types.StringValue("schedule"),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"type":                              types.StringValue("type"),
					"snapshot_retention_days":           types.Int32Value(1),
					"daily_snapshot_retention_days":     types.Int32Value(2),
					"weekly_snapshot_retention_weeks":   types.Int32Value(3),
					"monthly_snapshot_retention_months": types.Int32Value(4),
					"point_in_time_window_hours":        types.Int32Value(5),
				}),
			},
			&optionsModel{
				SnapshotRetentionDays:          types.Int32Value(6),
				DailySnapshotRetentionDays:     types.Int32Value(7),
				WeeklySnapshotRetentionWeeks:   types.Int32Null(),
				MonthlySnapshotRetentionMonths: types.Int32Null(),
				PointInTimeWindowHours:         types.Int32Null(),
			},
			&mongodbflex.UpdateBackupSchedulePayload{
				BackupSchedule:                 new("schedule"),
				SnapshotRetentionDays:          new(int32(6)),
				DailySnapshotRetentionDays:     new(int32(7)),
				WeeklySnapshotRetentionWeeks:   new(int32(3)),
				MonthlySnapshotRetentionMonths: new(int32(4)),
				PointInTimeWindowHours:         new(int32(5)),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				BackupSchedule: types.StringNull(),
			},
			&optionsModel{
				SnapshotRetentionDays:          types.Int32Null(),
				DailySnapshotRetentionDays:     types.Int32Null(),
				WeeklySnapshotRetentionWeeks:   types.Int32Null(),
				MonthlySnapshotRetentionMonths: types.Int32Null(),
				PointInTimeWindowHours:         types.Int32Null(),
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
				CPU: types.Int32Value(2),
				RAM: types.Int32Value(8),
			},
			&mongodbflex.ListFlavorsResponse{
				Flavors: []mongodbflex.InstanceFlavor{
					{
						Id:          new("fid-1"),
						Cpu:         new(int32(2)),
						Description: new("description"),
						Memory:      new(int32(8)),
					},
				},
			},
			&flavorModel{
				Id:          types.StringValue("fid-1"),
				Description: types.StringValue("description"),
				CPU:         types.Int32Value(2),
				RAM:         types.Int32Value(8),
			},
			false,
			true,
		},
		{
			"ok_flavor_2",
			&flavorModel{
				CPU: types.Int32Value(2),
				RAM: types.Int32Value(8),
			},
			&mongodbflex.ListFlavorsResponse{
				Flavors: []mongodbflex.InstanceFlavor{
					{
						Id:          new("fid-1"),
						Cpu:         new(int32(2)),
						Description: new("description"),
						Memory:      new(int32(8)),
					},
					{
						Id:          new("fid-2"),
						Cpu:         new(int32(1)),
						Description: new("description"),
						Memory:      new(int32(4)),
					},
				},
			},
			&flavorModel{
				Id:          types.StringValue("fid-1"),
				Description: types.StringValue("description"),
				CPU:         types.Int32Value(2),
				RAM:         types.Int32Value(8),
			},
			false,
			true,
		},
		{
			"no_matching_flavor",
			&flavorModel{
				CPU: types.Int32Value(2),
				RAM: types.Int32Value(8),
			},
			&mongodbflex.ListFlavorsResponse{
				Flavors: []mongodbflex.InstanceFlavor{
					{
						Id:          new("fid-1"),
						Cpu:         new(int32(1)),
						Description: new("description"),
						Memory:      new(int32(8)),
					},
					{
						Id:          new("fid-2"),
						Cpu:         new(int32(1)),
						Description: new("description"),
						Memory:      new(int32(4)),
					},
				},
			},
			&flavorModel{
				CPU: types.Int32Value(2),
				RAM: types.Int32Value(8),
			},
			false,
			false,
		},
		{
			"nil_response",
			&flavorModel{
				CPU: types.Int32Value(2),
				RAM: types.Int32Value(8),
			},
			&mongodbflex.ListFlavorsResponse{},
			&flavorModel{
				CPU: types.Int32Value(2),
				RAM: types.Int32Value(8),
			},
			false,
			false,
		},
		{
			"error_response",
			&flavorModel{
				CPU: types.Int32Value(2),
				RAM: types.Int32Value(8),
			},
			&mongodbflex.ListFlavorsResponse{},
			&flavorModel{
				CPU: types.Int32Value(2),
				RAM: types.Int32Value(8),
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
				ProjectId: types.StringValue(projectId),
			}
			flavorModel := &flavorModel{
				CPU: tt.inputFlavor.CPU,
				RAM: tt.inputFlavor.RAM,
			}
			err := loadFlavorId(context.Background(), client, model, flavorModel, testRegion)
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
