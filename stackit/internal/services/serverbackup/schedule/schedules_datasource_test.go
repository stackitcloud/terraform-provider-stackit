package schedule

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	sdk "github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
)

func listValueFrom(items []string) basetypes.ListValue {
	val, _ := types.ListValueFrom(context.TODO(), types.StringType, items)
	return val
}

func TestMapSchedulesDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *sdk.GetBackupSchedulesResponse
		expected    schedulesDataSourceModel
		isValid     bool
	}{
		{
			"empty response",
			&sdk.GetBackupSchedulesResponse{
				Items: &[]sdk.BackupSchedule{},
			},
			schedulesDataSourceModel{
				ID:        types.StringValue("project_uid,eu01,server_uid"),
				ProjectId: types.StringValue("project_uid"),
				ServerId:  types.StringValue("server_uid"),
				Items:     nil,
				Region:    types.StringValue("eu01"),
			},
			true,
		},
		{
			"simple_values",
			&sdk.GetBackupSchedulesResponse{
				Items: &[]sdk.BackupSchedule{
					{
						Id:      utils.Ptr(int64(5)),
						Enabled: utils.Ptr(true),
						Name:    utils.Ptr("backup_schedule_name_1"),
						Rrule:   utils.Ptr("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
						BackupProperties: &sdk.BackupProperties{
							Name:            utils.Ptr("backup_name_1"),
							RetentionPeriod: utils.Ptr(int64(14)),
							VolumeIds:       &[]string{"uuid1", "uuid2"},
						},
					},
				},
			},
			schedulesDataSourceModel{
				ID:        types.StringValue("project_uid,eu01,server_uid"),
				ServerId:  types.StringValue("server_uid"),
				ProjectId: types.StringValue("project_uid"),
				Items: []schedulesDatasourceItemModel{
					{
						BackupScheduleId: types.Int64Value(5),
						Name:             types.StringValue("backup_schedule_name_1"),
						Rrule:            types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
						Enabled:          types.BoolValue(true),
						BackupProperties: &scheduleBackupPropertiesModel{
							BackupName:      types.StringValue("backup_name_1"),
							RetentionPeriod: types.Int64Value(14),
							VolumeIds:       listValueFrom([]string{"uuid1", "uuid2"}),
						},
					},
				},
				Region: types.StringValue("eu01"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			schedulesDataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &schedulesDataSourceModel{
				ProjectId: tt.expected.ProjectId,
				ServerId:  tt.expected.ServerId,
			}
			ctx := context.TODO()
			err := mapSchedulesDatasourceFields(ctx, tt.input, state, "eu01")
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
