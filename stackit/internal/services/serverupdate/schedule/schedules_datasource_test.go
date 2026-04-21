package schedule

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	serverupdate "github.com/stackitcloud/stackit-sdk-go/services/serverupdate/v2api"
)

func TestMapSchedulesDataSourceFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *serverupdate.GetUpdateSchedulesResponse
		region      string
		expected    schedulesDataSourceModel
		isValid     bool
	}{
		{
			"empty response",
			&serverupdate.GetUpdateSchedulesResponse{
				Items: []serverupdate.UpdateSchedule{},
			},
			testRegion,
			schedulesDataSourceModel{
				ID:        types.StringValue("project_uid,region,server_uid"),
				ProjectId: types.StringValue("project_uid"),
				ServerId:  types.StringValue("server_uid"),
				Items:     nil,
				Region:    types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			&serverupdate.GetUpdateSchedulesResponse{
				Items: []serverupdate.UpdateSchedule{
					{
						Id:                5,
						Enabled:           true,
						Name:              "update_schedule_name_1",
						Rrule:             "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1",
						MaintenanceWindow: 1,
					},
				},
			},
			testRegion,
			schedulesDataSourceModel{
				ID:        types.StringValue("project_uid,region,server_uid"),
				ServerId:  types.StringValue("server_uid"),
				ProjectId: types.StringValue("project_uid"),
				Items: []schedulesDatasourceItemModel{
					{
						UpdateScheduleId:  types.Int32Value(5),
						Name:              types.StringValue("update_schedule_name_1"),
						Rrule:             types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
						Enabled:           types.BoolValue(true),
						MaintenanceWindow: types.Int32Value(1),
					},
				},
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			nil,
			testRegion,
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
			err := mapSchedulesDatasourceFields(ctx, tt.input, state, tt.region)
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
