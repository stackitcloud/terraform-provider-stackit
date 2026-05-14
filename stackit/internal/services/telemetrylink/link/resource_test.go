package link

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	telemetrylink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1betaapi"
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
		ID:                types.StringValue("rtp,rid,reg"),
		LinkID:            types.StringValue("lid"),
		Region:            types.StringValue("reg"),
		ResourceType:      types.StringValue("rtp"),
		ResourceID:        types.StringValue("rid"),
		DisplayName:       types.StringValue("name"),
		Description:       types.String{},
		TelemetryRouterID: types.StringValue("tlmrid"),
		AccessToken:       types.String{},
		CreateTime:        types.StringValue(testTime.String()),
		Status:            types.StringValue("active"),
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

func TestToCreateOrUpdateOrganizationTelemetryLinkPayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		expected       *telemetrylink.CreateOrUpdateOrganizationTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetrylink.CreateOrUpdateOrganizationTelemetryLinkPayload{
				DisplayName:       "name",
				AccessToken:       "",
				TelemetryRouterId: "tlmrid",
			},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.AccessToken = types.StringValue("access-token")
				model.TelemetryRouterID = types.StringValue("tlmr_id")
			}),
			expected: &telemetrylink.CreateOrUpdateOrganizationTelemetryLinkPayload{
				Description:       new("description"),
				DisplayName:       "display-name",
				AccessToken:       "access-token",
				TelemetryRouterId: "tlmr_id",
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreateOrUpdateOrganizationTelemetryLinkPayload(t.Context(), diag.Diagnostics{}, tt.model)
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
		expected       *telemetrylink.CreateOrUpdateFolderTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetrylink.CreateOrUpdateFolderTelemetryLinkPayload{
				DisplayName:       "name",
				AccessToken:       "",
				TelemetryRouterId: "tlmrid",
			},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.AccessToken = types.StringValue("access-token")
				model.TelemetryRouterID = types.StringValue("tlmr_id")
			}),
			expected: &telemetrylink.CreateOrUpdateFolderTelemetryLinkPayload{
				Description:       new("description"),
				DisplayName:       "display-name",
				AccessToken:       "access-token",
				TelemetryRouterId: "tlmr_id",
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreateOrUpdateFolderTelemetryLinkPayload(t.Context(), diag.Diagnostics{}, tt.model)
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
		expected       *telemetrylink.CreateOrUpdateProjectTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetrylink.CreateOrUpdateProjectTelemetryLinkPayload{
				DisplayName:       "name",
				AccessToken:       "",
				TelemetryRouterId: "tlmrid",
			},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.AccessToken = types.StringValue("access-token")
				model.TelemetryRouterID = types.StringValue("tlmr_id")
			}),
			expected: &telemetrylink.CreateOrUpdateProjectTelemetryLinkPayload{
				Description:       new("description"),
				DisplayName:       "display-name",
				AccessToken:       "access-token",
				TelemetryRouterId: "tlmr_id",
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreateOrUpdateProjectTelemetryLinkPayload(t.Context(), diag.Diagnostics{}, tt.model)
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
		expected       *telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload{
				DisplayName:       new("name"),
				AccessToken:       new(""),
				TelemetryRouterId: new("tlmrid"),
			},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.AccessToken = types.StringValue("access-token")
				model.TelemetryRouterID = types.StringValue("tlmr_id")
			}),
			expected: &telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload{
				Description:       new("description"),
				DisplayName:       new("display-name"),
				AccessToken:       new("access-token"),
				TelemetryRouterId: new("tlmr_id"),
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreateOrUpdateOrganizationTelemetryLinkPayload(t.Context(), diag.Diagnostics{}, tt.model)
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
		expected       *telemetrylink.PartialUpdateFolderTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetrylink.PartialUpdateFolderTelemetryLinkPayload{
				DisplayName:       new("name"),
				AccessToken:       new(""),
				TelemetryRouterId: new("tlmrid"),
			},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.AccessToken = types.StringValue("access-token")
				model.TelemetryRouterID = types.StringValue("tlmr_id")
			}),
			expected: &telemetrylink.PartialUpdateFolderTelemetryLinkPayload{
				Description:       new("description"),
				DisplayName:       new("display-name"),
				AccessToken:       new("access-token"),
				TelemetryRouterId: new("tlmr_id"),
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreateOrUpdateFolderTelemetryLinkPayload(t.Context(), diag.Diagnostics{}, tt.model)
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
		expected       *telemetrylink.PartialUpdateProjectTelemetryLinkPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetrylink.PartialUpdateProjectTelemetryLinkPayload{
				DisplayName:       new("name"),
				AccessToken:       new(""),
				TelemetryRouterId: new("tlmrid"),
			},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.AccessToken = types.StringValue("access-token")
				model.TelemetryRouterID = types.StringValue("tlmr_id")
			}),
			expected: &telemetrylink.PartialUpdateProjectTelemetryLinkPayload{
				Description:       new("description"),
				DisplayName:       new("display-name"),
				AccessToken:       new("access-token"),
				TelemetryRouterId: new("tlmr_id"),
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreateOrUpdateProjectTelemetryLinkPayload(t.Context(), diag.Diagnostics{}, tt.model)
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
