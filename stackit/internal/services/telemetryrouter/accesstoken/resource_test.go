package accesstoken

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
)

var testTime = time.Now()

func fixtureGetAccessToken(mods ...func(accessToken *telemetryrouter.GetAccessTokenResponse)) *telemetryrouter.GetAccessTokenResponse {
	accessToken := &telemetryrouter.GetAccessTokenResponse{
		Id:     "atid",
		Status: AccessTokenStatusActive,
	}
	for _, mod := range mods {
		mod(accessToken)
	}
	return accessToken
}

func fixtureCreateAccessToken(mods ...func(accessToken *telemetryrouter.CreateAccessTokenResponse)) *telemetryrouter.CreateAccessTokenResponse {
	accessToken := &telemetryrouter.CreateAccessTokenResponse{
		Id:     "atid",
		Status: AccessTokenStatusActive,
	}
	for _, mod := range mods {
		mod(accessToken)
	}
	return accessToken
}

func fixtureUpdateAccessToken(mods ...func(accessToken *telemetryrouter.UpdateAccessTokenResponse)) *telemetryrouter.UpdateAccessTokenResponse {
	accessToken := &telemetryrouter.UpdateAccessTokenResponse{
		Id:     "atid",
		Status: AccessTokenStatusActive,
	}
	for _, mod := range mods {
		mod(accessToken)
	}
	return accessToken
}

func fixtureModel(mods ...func(model *Model)) *Model {
	model := &Model{
		ID:             types.StringValue("pid,rid,iid,atid"),
		AccessTokenID:  types.StringValue("atid"),
		InstanceID:     types.StringValue("iid"),
		Region:         types.StringValue("rid"),
		ProjectID:      types.StringValue("pid"),
		CreatorID:      types.StringValue(""),
		Description:    types.String{},
		DisplayName:    types.StringValue(""),
		ExpirationTime: types.String{},
		Ttl:            types.Int32{},
		Status:         types.StringValue(AccessTokenStatusActive),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapGetFields(t *testing.T) {
	tests := []struct {
		description string
		input       any
		expected    *Model
		wantErr     bool
	}{
		{
			description: "nil input",
			wantErr:     true,
			expected:    fixtureModel(),
		},
		{
			description: "wrong type",
			wantErr:     true,
			input:       "invalid",
			expected:    fixtureModel(),
		},
		{
			description: "min values GetAccessTokenResponse",
			input:       fixtureGetAccessToken(),
			expected:    fixtureModel(),
		},
		{
			description: "max values GetAccessTokenResponse",
			input: fixtureGetAccessToken(func(accessToken *telemetryrouter.GetAccessTokenResponse) {
				accessToken.Description = new("description")
				accessToken.DisplayName = "display-name"
				accessToken.CreatorId = "testUser"
				accessToken.ExpirationTime = *telemetryrouter.NewNullableTime(&testTime)
			}),
			expected: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CreatorID = types.StringValue("testUser")
				model.ExpirationTime = types.StringValue(testTime.Format(time.RFC3339))
			}),
		},
		{
			description: "nil access token id GetAccessTokenResponse",
			input:       &telemetryrouter.GetAccessTokenResponse{},
			wantErr:     true,
			expected:    fixtureModel(),
		},
		{
			description: "min values CreateAccessTokenResponse",
			input:       fixtureCreateAccessToken(),
			expected:    fixtureModel(),
		},
		{
			description: "max values CreateAccessTokenResponse",
			input: fixtureCreateAccessToken(func(accessToken *telemetryrouter.CreateAccessTokenResponse) {
				accessToken.Description = new("description")
				accessToken.DisplayName = "display-name"
				accessToken.CreatorId = "testUser"
				accessToken.ExpirationTime = *telemetryrouter.NewNullableTime(&testTime)
				accessToken.AccessToken = "acc"
			}),
			expected: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CreatorID = types.StringValue("testUser")
				model.ExpirationTime = types.StringValue(testTime.Format(time.RFC3339))
				model.AccessToken = types.StringValue("acc")
			}),
		},
		{
			description: "nil access token id CreateAccessTokenResponse",
			input:       &telemetryrouter.CreateAccessTokenResponse{},
			wantErr:     true,
			expected:    fixtureModel(),
		},
		{
			description: "min values UpdateAccessTokenResponse",
			input:       fixtureUpdateAccessToken(),
			expected:    fixtureModel(),
		},
		{
			description: "max values UpdateAccessTokenResponse",
			input: fixtureUpdateAccessToken(func(accessToken *telemetryrouter.UpdateAccessTokenResponse) {
				accessToken.Description = new("description")
				accessToken.DisplayName = "display-name"
				accessToken.CreatorId = "testUser"
				accessToken.ExpirationTime = *telemetryrouter.NewNullableTime(&testTime)
			}),
			expected: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CreatorID = types.StringValue("testUser")
				model.ExpirationTime = types.StringValue(testTime.Format(time.RFC3339))
			}),
		},
		{
			description: "nil access token id UpdateAccessTokenResponse",
			input:       &telemetryrouter.UpdateAccessTokenResponse{},
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
		expected       *telemetryrouter.CreateAccessTokenPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetryrouter.CreateAccessTokenPayload{
				Ttl: *telemetryrouter.NewNullableInt32(nil),
			},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.Ttl = types.Int32Value(7)
			}),
			expected: &telemetryrouter.CreateAccessTokenPayload{
				Description: new("description"),
				DisplayName: "display-name",
				Ttl:         *telemetryrouter.NewNullableInt32(new(int32(7))),
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
			diff := cmp.Diff(got, tt.expected, cmp.Comparer(compareNullableString), cmp.Comparer(compareNullableInt32))
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
		expected       *telemetryrouter.UpdateAccessTokenPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetryrouter.UpdateAccessTokenPayload{
				DisplayName: *telemetryrouter.NewNullableString(new("")),
				Description: *telemetryrouter.NewNullableString(nil),
			},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
			}),
			expected: &telemetryrouter.UpdateAccessTokenPayload{
				Description: *telemetryrouter.NewNullableString(new("description")),
				DisplayName: *telemetryrouter.NewNullableString(new("display-name")),
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

			diff := cmp.Diff(got, tt.expected, cmp.Comparer(compareNullableString), cmp.Comparer(compareNullableInt32))
			if diff != "" {
				t.Fatalf("Payload does not match: %s", diff)
			}
		})
	}
}

func compareNullableString(x, y telemetryrouter.NullableString) bool {
	if x.IsSet() != y.IsSet() {
		return false
	}

	if !x.IsSet() && !y.IsSet() {
		return true
	}

	valX := x.Get()
	valY := y.Get()

	if valX == nil || valY == nil {
		return valX == valY
	}

	return *valX == *valY
}

func compareNullableInt32(x, y telemetryrouter.NullableInt32) bool {
	if x.IsSet() != y.IsSet() {
		return false
	}

	if !x.IsSet() && !y.IsSet() {
		return true
	}

	valX := x.Get()
	valY := y.Get()

	if valX == nil || valY == nil {
		return valX == valY
	}

	return *valX == *valY
}
