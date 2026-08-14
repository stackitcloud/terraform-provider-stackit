package link

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	telemetrylink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1api"
)

var testTime = time.Now()

func fixtureLink(mods ...func(link *telemetrylink.TelemetryLinkResponse)) *telemetrylink.TelemetryLinkResponse {
	link := &telemetrylink.TelemetryLinkResponse{
		Id:                "lid",
		DisplayName:       "name",
		TelemetryRouterId: "tlmrid",
		CreateTime:        testTime,
		Status:            "active",
	}
	for _, mod := range mods {
		mod(link)
	}
	return link
}

func fixtureModel(mods ...func(model *Model)) *Model {
	model := &Model{
		ID:                   types.StringValue("rtp,rid,reg"),
		Region:               types.StringValue("reg"),
		ResourceType:         types.StringValue("rtp"),
		ResourceID:           types.StringValue("rid"),
		DisplayName:          types.StringValue("name"),
		Description:          types.String{},
		TelemetryRouterID:    types.StringValue("tlmrid"),
		AccessToken:          types.String{},
		AccessTokenWo:        types.String{},
		AccessTokenWoVersion: types.Int64{},
		CreateTime:           types.StringValue(testTime.Format(time.RFC3339)),
		Status:               types.StringValue("active"),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *telemetrylink.TelemetryLinkResponse
		expected    *Model
		wantErr     bool
	}{
		{
			description: "min values",
			input:       fixtureLink(),
			expected:    fixtureModel(),
		},
		{
			description: "max values",
			input: fixtureLink(func(link *telemetrylink.TelemetryLinkResponse) {
				link.Description = new("description")
				link.DisplayName = "display-name"
				link.AccessToken = new("access-token")
				link.TelemetryRouterId = "tlmr-id"
			}),
			expected: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.TelemetryRouterID = types.StringValue("tlmr-id")
			}),
		},
		{
			description: "nil input",
			wantErr:     true,
			expected:    fixtureModel(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ResourceType: tt.expected.ResourceType,
				ResourceID:   tt.expected.ResourceID,
				Region:       tt.expected.Region,
			}
			err := mapFields(context.Background(), tt.input, state, tt.expected.Region.ValueString())
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

func TestToCreateOrUpdateOrganizationTelemetryLinkPayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		configModel    *Model
		expected       *telemetrylink.CreateOrUpdateOrganizationTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			configModel: fixtureModel(),
			expected: &telemetrylink.CreateOrUpdateOrganizationTelemetryLinkPayload{
				DisplayName:       "name",
				AccessToken:       "",
				TelemetryRouterId: "tlmrid",
			},
		},
		{
			description: "max values, legacy access_token",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.AccessToken = types.StringValue("access-token")
				model.TelemetryRouterID = types.StringValue("tlmr_id")
			}),
			configModel: fixtureModel(),
			expected: &telemetrylink.CreateOrUpdateOrganizationTelemetryLinkPayload{
				Description:       new("description"),
				DisplayName:       "display-name",
				AccessToken:       "access-token",
				TelemetryRouterId: "tlmr_id",
			},
		},
		{
			description: "write-only access_token_wo",
			model: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			configModel: fixtureModel(func(model *Model) {
				model.AccessTokenWo = types.StringValue("wo-access-token")
			}),
			expected: &telemetrylink.CreateOrUpdateOrganizationTelemetryLinkPayload{
				DisplayName:       "name",
				AccessToken:       "wo-access-token",
				TelemetryRouterId: "tlmrid",
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing plan model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreateOrUpdateOrganizationTelemetryLinkPayload(tt.model, tt.configModel)
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

func TestToCreateOrUpdateFolderTelemetryLinkPayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		configModel    *Model
		expected       *telemetrylink.CreateOrUpdateFolderTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			configModel: fixtureModel(),
			expected: &telemetrylink.CreateOrUpdateFolderTelemetryLinkPayload{
				DisplayName:       "name",
				AccessToken:       "",
				TelemetryRouterId: "tlmrid",
			},
		},
		{
			description: "max values, legacy access_token",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.AccessToken = types.StringValue("access-token")
				model.TelemetryRouterID = types.StringValue("tlmr_id")
			}),
			configModel: fixtureModel(),
			expected: &telemetrylink.CreateOrUpdateFolderTelemetryLinkPayload{
				Description:       new("description"),
				DisplayName:       "display-name",
				AccessToken:       "access-token",
				TelemetryRouterId: "tlmr_id",
			},
		},
		{
			description: "write-only access_token_wo",
			model: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			configModel: fixtureModel(func(model *Model) {
				model.AccessTokenWo = types.StringValue("wo-access-token")
			}),
			expected: &telemetrylink.CreateOrUpdateFolderTelemetryLinkPayload{
				DisplayName:       "name",
				AccessToken:       "wo-access-token",
				TelemetryRouterId: "tlmrid",
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing plan model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreateOrUpdateFolderTelemetryLinkPayload(tt.model, tt.configModel)
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

func TestToCreateOrUpdateProjectTelemetryLinkPayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		configModel    *Model
		expected       *telemetrylink.CreateOrUpdateProjectTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			configModel: fixtureModel(),
			expected: &telemetrylink.CreateOrUpdateProjectTelemetryLinkPayload{
				DisplayName:       "name",
				AccessToken:       "",
				TelemetryRouterId: "tlmrid",
			},
		},
		{
			description: "max values, legacy access_token",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.AccessToken = types.StringValue("access-token")
				model.TelemetryRouterID = types.StringValue("tlmr_id")
			}),
			configModel: fixtureModel(),
			expected: &telemetrylink.CreateOrUpdateProjectTelemetryLinkPayload{
				Description:       new("description"),
				DisplayName:       "display-name",
				AccessToken:       "access-token",
				TelemetryRouterId: "tlmr_id",
			},
		},
		{
			description: "write-only access_token_wo",
			model: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			configModel: fixtureModel(func(model *Model) {
				model.AccessTokenWo = types.StringValue("wo-access-token")
			}),
			expected: &telemetrylink.CreateOrUpdateProjectTelemetryLinkPayload{
				DisplayName:       "name",
				AccessToken:       "wo-access-token",
				TelemetryRouterId: "tlmrid",
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing plan model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreateOrUpdateProjectTelemetryLinkPayload(tt.model, tt.configModel)
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

func TestToPartialUpdateOrganizationTelemetryLinkPayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		stateModel     *Model
		configModel    *Model
		expected       *telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "legacy access_token is always resent",
			model: fixtureModel(func(model *Model) {
				model.AccessToken = types.StringValue("access-token")
			}),
			stateModel:  fixtureModel(),
			configModel: fixtureModel(),
			expected: &telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       new("access-token"),
			},
		},
		{
			description: "write-only access_token_wo, version unchanged - token untouched",
			model: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			stateModel: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			configModel: fixtureModel(func(model *Model) {
				model.AccessTokenWo = types.StringValue("wo-access-token")
			}),
			expected: &telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       nil,
			},
		},
		{
			description: "write-only access_token_wo, version bumped - token rotated",
			model: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(2)
			}),
			stateModel: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			configModel: fixtureModel(func(model *Model) {
				model.AccessTokenWo = types.StringValue("new-wo-access-token")
			}),
			expected: &telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       new("new-wo-access-token"),
			},
		},
		{
			// Regression test: removing `description` from the config plans it as null. The
			// PartialUpdate* API uses merge-patch semantics, where an omitted field means "leave
			// untouched" - so the payload must carry an explicit empty string here, or the API
			// would silently keep the old description and Terraform would fail with "Provider
			// produced inconsistent result after apply".
			description: "description removed from config - must be cleared, not left untouched",
			model:       fixtureModel(),
			stateModel: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("old description")
			}),
			configModel: fixtureModel(),
			expected: &telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       nil,
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing plan model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toPartialUpdateOrganizationTelemetryLinkPayload(tt.model, tt.stateModel, tt.configModel)
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

func TestToPartialUpdateFolderTelemetryLinkPayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		stateModel     *Model
		configModel    *Model
		expected       *telemetrylink.PartialUpdateFolderTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "legacy access_token is always resent",
			model: fixtureModel(func(model *Model) {
				model.AccessToken = types.StringValue("access-token")
			}),
			stateModel:  fixtureModel(),
			configModel: fixtureModel(),
			expected: &telemetrylink.PartialUpdateFolderTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       new("access-token"),
			},
		},
		{
			description: "write-only access_token_wo, version unchanged - token untouched",
			model: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			stateModel: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			configModel: fixtureModel(func(model *Model) {
				model.AccessTokenWo = types.StringValue("wo-access-token")
			}),
			expected: &telemetrylink.PartialUpdateFolderTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       nil,
			},
		},
		{
			description: "write-only access_token_wo, version bumped - token rotated",
			model: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(2)
			}),
			stateModel: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			configModel: fixtureModel(func(model *Model) {
				model.AccessTokenWo = types.StringValue("new-wo-access-token")
			}),
			expected: &telemetrylink.PartialUpdateFolderTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       new("new-wo-access-token"),
			},
		},
		{
			description: "description removed from config - must be cleared, not left untouched",
			model:       fixtureModel(),
			stateModel: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("old description")
			}),
			configModel: fixtureModel(),
			expected: &telemetrylink.PartialUpdateFolderTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       nil,
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing plan model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toPartialUpdateFolderTelemetryLinkPayload(tt.model, tt.stateModel, tt.configModel)
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

func TestToPartialUpdateProjectTelemetryLinkPayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		stateModel     *Model
		configModel    *Model
		expected       *telemetrylink.PartialUpdateProjectTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "legacy access_token is always resent",
			model: fixtureModel(func(model *Model) {
				model.AccessToken = types.StringValue("access-token")
			}),
			stateModel:  fixtureModel(),
			configModel: fixtureModel(),
			expected: &telemetrylink.PartialUpdateProjectTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       new("access-token"),
			},
		},
		{
			description: "write-only access_token_wo, version unchanged - token untouched",
			model: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			stateModel: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			configModel: fixtureModel(func(model *Model) {
				model.AccessTokenWo = types.StringValue("wo-access-token")
			}),
			expected: &telemetrylink.PartialUpdateProjectTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       nil,
			},
		},
		{
			description: "write-only access_token_wo, version bumped - token rotated",
			model: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(2)
			}),
			stateModel: fixtureModel(func(model *Model) {
				model.AccessTokenWoVersion = types.Int64Value(1)
			}),
			configModel: fixtureModel(func(model *Model) {
				model.AccessTokenWo = types.StringValue("new-wo-access-token")
			}),
			expected: &telemetrylink.PartialUpdateProjectTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       new("new-wo-access-token"),
			},
		},
		{
			description: "description removed from config - must be cleared, not left untouched",
			model:       fixtureModel(),
			stateModel: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("old description")
			}),
			configModel: fixtureModel(),
			expected: &telemetrylink.PartialUpdateProjectTelemetryLinkPayload{
				DisplayName:       new("name"),
				Description:       new(""),
				TelemetryRouterId: new("tlmrid"),
				AccessToken:       nil,
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing plan model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toPartialUpdateProjectTelemetryLinkPayload(tt.model, tt.stateModel, tt.configModel)
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
