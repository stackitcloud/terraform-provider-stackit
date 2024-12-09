package schedule

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	sdk "github.com/stackitcloud/stackit-sdk-go/services/serverupdate"
)

func TestMapSchedulesDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *sdk.GetUpdateSchedulesResponse
		expected    schedulesDataSourceModel
		isValid     bool
	}{
		{
			"empty response",
			&sdk.GetUpdateSchedulesResponse{
				Items: &[]sdk.UpdateSchedule{},
			},
			schedulesDataSourceModel{
				ID:        types.StringValue("project_uid,server_uid"),
				ProjectId: types.StringValue("project_uid"),
				ServerId:  types.StringValue("server_uid"),
				Items:     nil,
			},
			true,
		},
		{
			"simple_values",
			&sdk.GetUpdateSchedulesResponse{
				Items: &[]sdk.UpdateSchedule{
					{
						Id:                utils.Ptr(int64(5)),
						Enabled:           utils.Ptr(true),
						Name:              utils.Ptr("update_schedule_name_1"),
						Rrule:             utils.Ptr("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
						MaintenanceWindow: utils.Ptr(int64(1)),
					},
				},
			},
			schedulesDataSourceModel{
				ID:        types.StringValue("project_uid,server_uid"),
				ServerId:  types.StringValue("server_uid"),
				ProjectId: types.StringValue("project_uid"),
				Items: []schedulesDatasourceItemModel{
					{
						UpdateScheduleId:  types.Int64Value(5),
						Name:              types.StringValue("update_schedule_name_1"),
						Rrule:             types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
						Enabled:           types.BoolValue(true),
						MaintenanceWindow: types.Int64Value(1),
					},
				},
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
			err := mapSchedulesDatasourceFields(ctx, tt.input, state)
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
