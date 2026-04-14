package schedule

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	serverbackup "github.com/stackitcloud/stackit-sdk-go/services/serverbackup/v2api"
)

func listValueFrom(items []string) basetypes.ListValue {
	val, _ := types.ListValueFrom(context.TODO(), types.StringType, items)
	return val
}

func TestMapSchedulesDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *serverbackup.GetBackupSchedulesResponse
		expected    schedulesDataSourceModel
		isValid     bool
	}{
		{
			"empty response",
			&serverbackup.GetBackupSchedulesResponse{
				Items: []serverbackup.BackupSchedule{},
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
			&serverbackup.GetBackupSchedulesResponse{
				Items: []serverbackup.BackupSchedule{
					{
						Id:      int32(5),
						Enabled: true,
						Name:    "backup_schedule_name_1",
						Rrule:   "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1",
						BackupProperties: &serverbackup.BackupProperties{
							Name:            "backup_name_1",
							RetentionPeriod: int32(14),
							VolumeIds:       []string{"uuid1", "uuid2"},
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
						BackupScheduleId: types.Int32Value(5),
						Name:             types.StringValue("backup_schedule_name_1"),
						Rrule:            types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
						Enabled:          types.BoolValue(true),
						BackupProperties: &scheduleBackupPropertiesModel{
							BackupName:      types.StringValue("backup_name_1"),
							RetentionPeriod: types.Int32Value(14),
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
