package accesstoken

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	logs "github.com/stackitcloud/stackit-sdk-go/services/logs/v1api"
)

func fixtureDataSourceModel(mods ...func(model *DataSourceModel)) *DataSourceModel {
	model := &DataSourceModel{
		ID:            types.StringValue("pid,rid,iid,atid"),
		AccessTokenID: types.StringValue("atid"),
		InstanceID:    types.StringValue("iid"),
		Region:        types.StringValue("rid"),
		ProjectID:     types.StringValue("pid"),
		Creator:       types.String{},
		Description:   types.String{},
		DisplayName:   types.String{},
		Expires:       types.Bool{},
		ValidUntil:    types.String{},
		Permissions:   types.ListNull(types.StringType),
		Status:        types.StringValue("active"),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *logs.AccessToken
		expected    *DataSourceModel
		wantErr     bool
	}{
		{
			description: "min values",
			input: fixtureAccessToken(func(accessToken *logs.AccessToken) {
				accessToken.DisplayName = "display-name"
			}),
			expected: fixtureDataSourceModel(func(model *DataSourceModel) {
				model.Creator = types.StringValue("")
				model.DisplayName = types.StringValue("display-name")
				model.Expires = types.BoolValue(false)
			}),
		},
		{
			description: "max values",
			input: fixtureAccessToken(func(accessToken *logs.AccessToken) {
				accessToken.Permissions = []string{"write"}
				accessToken.AccessToken = new("")
				accessToken.Description = new("description")
				accessToken.DisplayName = "display-name"
				accessToken.Creator = "testUser"
				accessToken.Expires = false
				accessToken.ValidUntil = new(testTime)
			}),
			expected: fixtureDataSourceModel(func(model *DataSourceModel) {
				model.Permissions = types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("write"),
				})
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.Creator = types.StringValue("testUser")
				model.Expires = types.BoolValue(false)
				model.ValidUntil = types.StringValue(testTime.Format(time.RFC3339))
			}),
		},
		{
			description: "nil input",
			wantErr:     true,
			expected:    fixtureDataSourceModel(),
		},
		{
			description: "nil access token id",
			input:       &logs.AccessToken{},
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
				diff := cmp.Diff(tt.expected, state)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
