package accesstoken

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	logs "github.com/stackitcloud/stackit-sdk-go/services/logs/v1api"
)

var testTime = time.Now()

func fixtureAccessToken(mods ...func(accessToken *logs.AccessToken)) *logs.AccessToken {
	accessToken := &logs.AccessToken{
		Id:     "atid",
		Status: "active",
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
		AccessToken:   types.String{},
		Description:   types.String{},
		DisplayName:   types.String{},
		Expires:       types.Bool{},
		ValidUntil:    types.String{},
		Lifetime:      types.Int32{},
		Permissions:   types.ListNull(types.StringType),
		Status:        types.StringValue("active"),
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
			input: fixtureAccessToken(func(accessToken *logs.AccessToken) {
				accessToken.DisplayName = "display-name"
			}),
			expected: fixtureModel(func(model *Model) {
				model.DisplayName = types.StringValue("display-name")
				model.Creator = types.StringValue("")
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
			expected: fixtureModel(func(model *Model) {
				model.Permissions = types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("write"),
				})
				model.AccessToken = types.StringValue("")
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
				model.Lifetime = types.Int32Value(7)
			}),
			expected: &logs.CreateAccessTokenPayload{
				Permissions: []string{"read", "write"},
				Description: new("description"),
				DisplayName: "display-name",
				Lifetime:    new(int32(7)),
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
				Description: new("description"),
				DisplayName: new("display-name"),
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
