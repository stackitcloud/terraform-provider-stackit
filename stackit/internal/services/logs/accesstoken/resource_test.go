package accesstoken

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/logs"
)

var testTime = time.Now()

func fixtureAccessToken(mods ...func(accessToken *logs.AccessToken)) *logs.AccessToken {
	accessToken := &logs.AccessToken{
		Id:     utils.Ptr("atid"),
		Status: utils.Ptr(logs.ACCESSTOKENSTATUS_ACTIVE),
	}
	for _, mod := range mods {
		mod(accessToken)
	}
	return accessToken
}

func fixtureModel(mods ...func(model *Model)) *Model {
	model := &Model{
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
		Lifetime:      types.Int64{},
		Permissions:   types.ListNull(types.StringType),
		Status:        types.StringValue(string(logs.ACCESSTOKENSTATUS_ACTIVE)),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *logs.AccessToken
		expected    *Model
		wantErr     bool
	}{
		{
			description: "min values",
			input:       fixtureAccessToken(),
			expected:    fixtureModel(),
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
			expected: fixtureModel(func(model *Model) {
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
			expected:    fixtureModel(),
		},
		{
			description: "nil access token id",
			input:       &logs.AccessToken{},
			wantErr:     true,
			expected:    fixtureModel(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectID:  tt.expected.ProjectID,
				Region:     tt.expected.Region,
				InstanceID: tt.expected.InstanceID,
			}
			err := mapFields(context.Background(), tt.input, state)
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

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		expected       *logs.CreateAccessTokenPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected:    &logs.CreateAccessTokenPayload{},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Permissions = types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("read"),
					types.StringValue("write"),
				})
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.Lifetime = types.Int64Value(7)
			}),
			expected: &logs.CreateAccessTokenPayload{
				Permissions: &[]string{"read", "write"},
				Description: utils.Ptr("description"),
				DisplayName: utils.Ptr("display-name"),
				Lifetime:    utils.Ptr(int64(7)),
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreatePayload(t.Context(), diag.Diagnostics{}, tt.model)
			if tt.wantErrMessage != "" && (err == nil || err.Error() != tt.wantErrMessage) {
				t.Fatalf("Expected error: %v, got: %v", tt.wantErrMessage, err)
			}
			if tt.wantErrMessage == "" && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			diff := cmp.Diff(got, tt.expected)
			if diff != "" {
				t.Fatalf("Payload does not match: %s", diff)
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		expected       *logs.UpdateAccessTokenPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected:    &logs.UpdateAccessTokenPayload{},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
			}),
			expected: &logs.UpdateAccessTokenPayload{
				Description: utils.Ptr("description"),
				DisplayName: utils.Ptr("display-name"),
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toUpdatePayload(tt.model)
			if tt.wantErrMessage != "" && (err == nil || err.Error() != tt.wantErrMessage) {
				t.Fatalf("Expected error: %v, got: %v", tt.wantErrMessage, err)
			}
			if tt.wantErrMessage == "" && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			diff := cmp.Diff(got, tt.expected)
			if diff != "" {
				t.Fatalf("Payload does not match: %s", diff)
			}
		})
	}
}
