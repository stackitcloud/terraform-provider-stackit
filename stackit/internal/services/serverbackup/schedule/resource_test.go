package schedule

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	sdk "github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *sdk.BackupSchedule
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&sdk.BackupSchedule{
				Id: utils.Ptr(int64(5)),
			},
			Model{
				ID:               types.StringValue("project_uid,eu01,server_uid,5"),
				ProjectId:        types.StringValue("project_uid"),
				ServerId:         types.StringValue("server_uid"),
				BackupScheduleId: types.Int64Value(5),
			},
			true,
		},
		{
			"simple_values",
			&sdk.BackupSchedule{
				Id:      utils.Ptr(int64(5)),
				Enabled: utils.Ptr(true),
				Name:    utils.Ptr("backup_schedule_name_1"),
				Rrule:   utils.Ptr("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				BackupProperties: &sdk.BackupProperties{
					Name:            utils.Ptr("backup_name_1"),
					RetentionPeriod: utils.Ptr(int64(3)),
					VolumeIds:       &[]string{"uuid1", "uuid2"},
				},
			},
			Model{
				ServerId:         types.StringValue("server_uid"),
				ProjectId:        types.StringValue("project_uid"),
				BackupScheduleId: types.Int64Value(5),
				ID:               types.StringValue("project_uid,eu01,server_uid,5"),
				Name:             types.StringValue("backup_schedule_name_1"),
				Rrule:            types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				Enabled:          types.BoolValue(true),
				BackupProperties: &scheduleBackupPropertiesModel{
					BackupName:      types.StringValue("backup_name_1"),
					RetentionPeriod: types.Int64Value(3),
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
		{
			"no_resource_id",
			&sdk.BackupSchedule{},
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
		expected    *sdk.CreateBackupSchedulePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&sdk.CreateBackupSchedulePayload{
				BackupProperties: &sdk.BackupProperties{},
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
			&sdk.CreateBackupSchedulePayload{
				Name:             utils.Ptr("name"),
				Enabled:          utils.Ptr(true),
				Rrule:            utils.Ptr("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				BackupProperties: &sdk.BackupProperties{},
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name:  types.StringValue(""),
				Rrule: types.StringValue(""),
			},
			&sdk.CreateBackupSchedulePayload{
				BackupProperties: &sdk.BackupProperties{},
				Name:             utils.Ptr(""),
				Rrule:            utils.Ptr(""),
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
		expected    *sdk.UpdateBackupSchedulePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&sdk.UpdateBackupSchedulePayload{
				BackupProperties: &sdk.BackupProperties{},
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
			&sdk.UpdateBackupSchedulePayload{
				Name:             utils.Ptr("name"),
				Enabled:          utils.Ptr(true),
				Rrule:            utils.Ptr("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				BackupProperties: &sdk.BackupProperties{},
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name:  types.StringValue(""),
				Rrule: types.StringValue(""),
			},
			&sdk.UpdateBackupSchedulePayload{
				BackupProperties: &sdk.BackupProperties{},
				Name:             utils.Ptr(""),
				Rrule:            utils.Ptr(""),
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
