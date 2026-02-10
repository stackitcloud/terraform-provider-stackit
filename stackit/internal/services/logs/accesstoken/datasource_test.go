package accesstoken

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/logs"
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
		Status:        types.StringValue(string(logs.ACCESSTOKENSTATUS_ACTIVE)),
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
			input:       fixtureAccessToken(),
			expected:    fixtureDataSourceModel(),
		},
		{
			description: "max values",
			input: fixtureAccessToken(func(accessToken *logs.AccessToken) {
				accessToken.Permissions = &[]string{"write"}
				accessToken.AccessToken = utils.Ptr("")
				accessToken.Description = utils.Ptr("description")
				accessToken.DisplayName = utils.Ptr("display-name")
				accessToken.Creator = utils.Ptr("testUser")
				accessToken.Expires = utils.Ptr(false)
				accessToken.ValidUntil = utils.Ptr(testTime)
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
				diff := cmp.Diff(state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
