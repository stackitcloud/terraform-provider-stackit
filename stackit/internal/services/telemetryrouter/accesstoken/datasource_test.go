package accesstoken

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
)

func fixtureDataSourceModel(mods ...func(model *DataSourceModel)) *DataSourceModel {
	model := &DataSourceModel{
		ID:             types.StringValue("pid,rid,iid,atid"),
		AccessTokenID:  types.StringValue("atid"),
		InstanceID:     types.StringValue("iid"),
		Region:         types.StringValue("rid"),
		ProjectID:      types.StringValue("pid"),
		CreatorID:      types.StringValue(""),
		Description:    types.String{},
		DisplayName:    types.StringValue(""),
		ExpirationTime: types.String{},
		Status:         types.StringValue("active"),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *telemetryrouter.GetAccessTokenResponse
		expected    *DataSourceModel
		wantErr     bool
	}{
		{
			description: "min values",
			input:       fixtureGetAccessToken(),
			expected:    fixtureDataSourceModel(),
		},
		{
			description: "max values",
			input: fixtureGetAccessToken(func(accessToken *telemetryrouter.GetAccessTokenResponse) {
				accessToken.Description = new("description")
				accessToken.DisplayName = "display-name"
				accessToken.CreatorId = "testUser"
				accessToken.ExpirationTime = *telemetryrouter.NewNullableTime(&testTime)
			}),
			expected: fixtureDataSourceModel(func(model *DataSourceModel) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CreatorID = types.StringValue("testUser")
				model.ExpirationTime = types.StringValue(testTime.Format(time.RFC3339))
			}),
		},
		{
			description: "nil input",
			wantErr:     true,
			expected:    fixtureDataSourceModel(),
		},
		{
			description: "nil access token id",
			input:       &telemetryrouter.GetAccessTokenResponse{},
			wantErr:     true,
			expected:    fixtureDataSourceModel(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &DataSourceModel{
				ProjectID:  tt.expected.ProjectID,
				Region:     tt.expected.Region,
				InstanceID: tt.expected.InstanceID,
			}
			err := mapDataSourceFields(context.Background(), tt.input, state)
			if tt.wantErr && err == nil {
				t.Fatalf("Should have failed")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if !tt.wantErr {
				diff := cmp.Diff(state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
