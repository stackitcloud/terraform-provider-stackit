package schedule

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	serverbackup "github.com/stackitcloud/stackit-sdk-go/services/serverbackup/v2api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *serverbackup.BackupSchedule
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&serverbackup.BackupSchedule{
				Id: int32(5),
			},
			Model{
				ID:               types.StringValue("project_uid,eu01,server_uid,5"),
				ProjectId:        types.StringValue("project_uid"),
				ServerId:         types.StringValue("server_uid"),
				BackupScheduleId: types.Int32Value(5),
				Name:             types.StringValue(""),
				Rrule:            types.StringValue(""),
				Enabled:          types.BoolValue(false),
			},
			true,
		},
		{
			"simple_values",
			&serverbackup.BackupSchedule{
				Id:      int32(5),
				Enabled: true,
				Name:    "backup_schedule_name_1",
				Rrule:   "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1",
				BackupProperties: &serverbackup.BackupProperties{
					Name:            "backup_name_1",
					RetentionPeriod: int32(3),
					VolumeIds:       []string{"uuid1", "uuid2"},
				},
			},
			Model{
				ServerId:         types.StringValue("server_uid"),
				ProjectId:        types.StringValue("project_uid"),
				BackupScheduleId: types.Int32Value(5),
				ID:               types.StringValue("project_uid,eu01,server_uid,5"),
				Name:             types.StringValue("backup_schedule_name_1"),
				Rrule:            types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				Enabled:          types.BoolValue(true),
				BackupProperties: &scheduleBackupPropertiesModel{
					BackupName:      types.StringValue("backup_name_1"),
					RetentionPeriod: types.Int32Value(3),
					VolumeIds:       listValueFrom([]string{"uuid1", "uuid2"}),
				},
				Region: types.StringValue("eu01"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
				ServerId:  tt.expected.ServerId,
			}
			ctx := context.TODO()
			err := mapFields(ctx, tt.input, state, "eu01")
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
		description string
		input       *Model
		expected    *serverbackup.CreateBackupSchedulePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&serverbackup.CreateBackupSchedulePayload{
				BackupProperties: &serverbackup.BackupProperties{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:             types.StringValue("name"),
				Enabled:          types.BoolValue(true),
				Rrule:            types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				BackupProperties: nil,
			},
			&serverbackup.CreateBackupSchedulePayload{
				Name:             "name",
				Enabled:          true,
				Rrule:            "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1",
				BackupProperties: &serverbackup.BackupProperties{},
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name:  types.StringValue(""),
				Rrule: types.StringValue(""),
			},
			&serverbackup.CreateBackupSchedulePayload{
				BackupProperties: &serverbackup.BackupProperties{},
				Name:             "",
				Rrule:            "",
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
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
		description string
		input       *Model
		expected    *serverbackup.UpdateBackupSchedulePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&serverbackup.UpdateBackupSchedulePayload{
				BackupProperties: &serverbackup.BackupProperties{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:             types.StringValue("name"),
				Enabled:          types.BoolValue(true),
				Rrule:            types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				BackupProperties: nil,
			},
			&serverbackup.UpdateBackupSchedulePayload{
				Name:             "name",
				Enabled:          true,
				Rrule:            "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1",
				BackupProperties: &serverbackup.BackupProperties{},
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name:  types.StringValue(""),
				Rrule: types.StringValue(""),
			},
			&serverbackup.UpdateBackupSchedulePayload{
				BackupProperties: &serverbackup.BackupProperties{},
				Name:             "",
				Rrule:            "",
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input)
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
