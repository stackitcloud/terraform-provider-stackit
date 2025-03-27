package schedule

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	sdk "github.com/stackitcloud/stackit-sdk-go/services/serverupdate"
)

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *sdk.UpdateSchedule
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&sdk.UpdateSchedule{
				Id: utils.Ptr(int64(5)),
			},
			testRegion,
			Model{
				ID:               types.StringValue("project_uid,region,server_uid,5"),
				ProjectId:        types.StringValue("project_uid"),
				ServerId:         types.StringValue("server_uid"),
				UpdateScheduleId: types.Int64Value(5),
				Region:           types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			&sdk.UpdateSchedule{
				Id:                utils.Ptr(int64(5)),
				Enabled:           utils.Ptr(true),
				Name:              utils.Ptr("update_schedule_name_1"),
				Rrule:             utils.Ptr("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				MaintenanceWindow: utils.Ptr(int64(1)),
			},
			testRegion,
			Model{
				ServerId:          types.StringValue("server_uid"),
				ProjectId:         types.StringValue("project_uid"),
				UpdateScheduleId:  types.Int64Value(5),
				ID:                types.StringValue("project_uid,region,server_uid,5"),
				Name:              types.StringValue("update_schedule_name_1"),
				Rrule:             types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				Enabled:           types.BoolValue(true),
				MaintenanceWindow: types.Int64Value(1),
				Region:            types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			nil,
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&sdk.UpdateSchedule{},
			testRegion,
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
			err := mapFields(tt.input, state, tt.region)
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
		expected    *sdk.CreateUpdateSchedulePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&sdk.CreateUpdateSchedulePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:              types.StringValue("name"),
				Enabled:           types.BoolValue(true),
				Rrule:             types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				MaintenanceWindow: types.Int64Value(1),
			},
			&sdk.CreateUpdateSchedulePayload{
				Name:              utils.Ptr("name"),
				Enabled:           utils.Ptr(true),
				Rrule:             utils.Ptr("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				MaintenanceWindow: utils.Ptr(int64(1)),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name:  types.StringValue(""),
				Rrule: types.StringValue(""),
			},
			&sdk.CreateUpdateSchedulePayload{
				Name:  utils.Ptr(""),
				Rrule: utils.Ptr(""),
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
		expected    *sdk.UpdateUpdateSchedulePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&sdk.UpdateUpdateSchedulePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:              types.StringValue("name"),
				Enabled:           types.BoolValue(true),
				Rrule:             types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				MaintenanceWindow: types.Int64Value(1),
			},
			&sdk.UpdateUpdateSchedulePayload{
				Name:              utils.Ptr("name"),
				Enabled:           utils.Ptr(true),
				Rrule:             utils.Ptr("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
				MaintenanceWindow: utils.Ptr(int64(1)),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name:  types.StringValue(""),
				Rrule: types.StringValue(""),
			},
			&sdk.UpdateUpdateSchedulePayload{
				Name:  utils.Ptr(""),
				Rrule: utils.Ptr(""),
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
