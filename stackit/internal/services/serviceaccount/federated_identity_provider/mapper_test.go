package federated_identity_provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"
)

func assertionsObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"item":     types.StringType,
			"operator": types.StringType,
			"value":    types.StringType,
		},
	}
}

func assertionsListFromModels(t *testing.T, assertions []AssertionModel) types.List {
	t.Helper()

	listValue, diags := types.ListValueFrom(t.Context(), assertionsObjectType(), assertions)
	if diags.HasError() {
		t.Fatalf("failed to build assertions list: %v", diags.Errors())
	}
	return listValue
}

func ptrString(s string) *string { return &s }

func TestMapFields(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		description          string
		input                *serviceaccount.FederatedIdentityProvider
		projectID            string
		serviceAccountEmail  string
		expectError          bool
		expectAssertionsNull bool
		expectedAssertions   []AssertionModel
	}{
		{
			description:         "default_values",
			projectID:           "pid",
			serviceAccountEmail: "service-account@sa.stackit.cloud",
			input: &serviceaccount.FederatedIdentityProvider{
				Id:     ptrString("fed-uuid-123"),
				Name:   "provider-name",
				Issuer: "https://issuer.example.com",
				Assertions: []serviceaccount.FederatedIdentityProviderAssertionsInner{
					{Item: "iss", Operator: "equals", Value: "https://issuer.example.com"},
					{Item: "sub", Operator: "equals", Value: "user@example.com"},
				},
			},
			expectedAssertions: []AssertionModel{
				{Item: types.StringValue("iss"), Operator: types.StringValue("equals"), Value: types.StringValue("https://issuer.example.com")},
				{Item: types.StringValue("sub"), Operator: types.StringValue("equals"), Value: types.StringValue("user@example.com")},
			},
		},
		{
			description:          "empty_optional_values",
			projectID:            "pid",
			serviceAccountEmail:  "service-account@sa.stackit.cloud",
			input:                &serviceaccount.FederatedIdentityProvider{},
			expectAssertionsNull: true,
		},
		{
			description: "nil_response",
			input:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{}

			err := mapFields(ctx, tt.input, model, tt.projectID, tt.serviceAccountEmail)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if model.ProjectId.ValueString() != tt.projectID {
				t.Fatalf("project_id mismatch: got %q, expected %q", model.ProjectId.ValueString(), tt.projectID)
			}
			if model.ServiceAccountEmail.ValueString() != tt.serviceAccountEmail {
				t.Fatalf("service_account_email mismatch: got %q, expected %q", model.ServiceAccountEmail.ValueString(), tt.serviceAccountEmail)
			}

			if tt.description == "default_values" {
				if model.Name.ValueString() != "provider-name" {
					t.Fatalf("name mismatch: got %q", model.Name.ValueString())
				}
				if model.Id.ValueString() != "pid,service-account@sa.stackit.cloud,provider-name" {
					t.Fatalf("id mismatch: got %q", model.Id.ValueString())
				}
				if model.FederationId.ValueString() != "fed-uuid-123" {
					t.Fatalf("federation_id mismatch: got %q", model.FederationId.ValueString())
				}
				if model.Issuer.ValueString() != "https://issuer.example.com" {
					t.Fatalf("issuer mismatch: got %q", model.Issuer.ValueString())
				}
			}

			if tt.expectAssertionsNull {
				if !model.Assertions.IsNull() {
					t.Fatalf("expected assertions to be null")
				}
				if !model.Issuer.IsNull() {
					t.Fatalf("expected issuer to be null")
				}
				return
			}

			var mappedAssertions []AssertionModel
			diags := model.Assertions.ElementsAs(ctx, &mappedAssertions, false)
			if diags.HasError() {
				t.Fatalf("failed to decode assertions: %v", diags.Errors())
			}
			if len(mappedAssertions) != len(tt.expectedAssertions) {
				t.Fatalf("assertions length mismatch: got %d, expected %d", len(mappedAssertions), len(tt.expectedAssertions))
			}
			for i := range mappedAssertions {
				if mappedAssertions[i].Item.ValueString() != tt.expectedAssertions[i].Item.ValueString() {
					t.Fatalf("assertions[%d].item mismatch: got %q, expected %q", i, mappedAssertions[i].Item.ValueString(), tt.expectedAssertions[i].Item.ValueString())
				}
				if mappedAssertions[i].Operator.ValueString() != tt.expectedAssertions[i].Operator.ValueString() {
					t.Fatalf("assertions[%d].operator mismatch: got %q, expected %q", i, mappedAssertions[i].Operator.ValueString(), tt.expectedAssertions[i].Operator.ValueString())
				}
				if mappedAssertions[i].Value.ValueString() != tt.expectedAssertions[i].Value.ValueString() {
					t.Fatalf("assertions[%d].value mismatch: got %q, expected %q", i, mappedAssertions[i].Value.ValueString(), tt.expectedAssertions[i].Value.ValueString())
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	validAssertions := []AssertionModel{
		{Item: types.StringValue("iss"), Operator: types.StringValue("equals"), Value: types.StringValue("https://issuer.example.com")},
		{Item: types.StringValue("sub"), Operator: types.StringValue("equals"), Value: types.StringValue("user@example.com")},
	}

	tests := []struct {
		description string
		model       *Model
		expectError bool
	}{
		{
			description: "default_values",
			model: &Model{
				Name:       types.StringValue("provider-name"),
				Issuer:     types.StringValue("https://issuer.example.com"),
				Assertions: assertionsListFromModels(t, validAssertions),
			},
		},
		{
			description: "without_assertions",
			model: &Model{
				Name:   types.StringValue("provider-name"),
				Issuer: types.StringValue("https://issuer.example.com"),
				Assertions: types.ListNull(types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"item":     types.StringType,
						"operator": types.StringType,
						"value":    types.StringType,
					},
				}),
			},
		},
		{
			description: "invalid_assertions_type",
			model: &Model{
				Name:       types.StringValue("provider-name"),
				Issuer:     types.StringValue("https://issuer.example.com"),
				Assertions: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("not-an-object")}),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toCreatePayload(t.Context(), tt.model)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if payload != nil {
					t.Fatalf("expected nil payload on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if payload.Name != "provider-name" {
				t.Fatalf("name mismatch: got %q", payload.Name)
			}
			if payload.Issuer != "https://issuer.example.com" {
				t.Fatalf("issuer mismatch: got %q", payload.Issuer)
			}

			switch tt.description {
			case "default_values":
				if len(payload.Assertions) != 2 {
					t.Fatalf("assertions length mismatch: got %d", len(payload.Assertions))
				}
				if payload.Assertions[0].Item == nil || *payload.Assertions[0].Item != "iss" {
					t.Fatalf("assertions[0].item mismatch")
				}
				if payload.Assertions[0].Operator == nil || *payload.Assertions[0].Operator != "equals" {
					t.Fatalf("assertions[0].operator mismatch")
				}
				if payload.Assertions[0].Value == nil || *payload.Assertions[0].Value != "https://issuer.example.com" {
					t.Fatalf("assertions[0].value mismatch")
				}
				if payload.Assertions[1].Item == nil || *payload.Assertions[1].Item != "sub" {
					t.Fatalf("assertions[1].item mismatch")
				}
				if payload.Assertions[1].Operator == nil || *payload.Assertions[1].Operator != "equals" {
					t.Fatalf("assertions[1].operator mismatch")
				}
				if payload.Assertions[1].Value == nil || *payload.Assertions[1].Value != "user@example.com" {
					t.Fatalf("assertions[1].value mismatch")
				}
			case "without_assertions":
				if len(payload.Assertions) != 0 {
					t.Fatalf("expected no assertions, got %d", len(payload.Assertions))
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	validAssertions := []AssertionModel{
		{Item: types.StringValue("aud"), Operator: types.StringValue("equals"), Value: types.StringValue("https://example.com")},
		{Item: types.StringValue("sub"), Operator: types.StringValue("equals"), Value: types.StringValue("user@example.com")},
	}

	tests := []struct {
		description string
		model       *Model
		expectError bool
	}{
		{
			description: "all_fields_set",
			model: &Model{
				Name:       types.StringValue("provider-name"),
				Issuer:     types.StringValue("https://issuer.example.com"),
				Assertions: assertionsListFromModels(t, validAssertions),
			},
		},
		{
			description: "null_assertions_replaces_external",
			model: &Model{
				Name:   types.StringValue("provider-name"),
				Issuer: types.StringValue("https://issuer.example.com"),
				Assertions: types.ListNull(types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"item":     types.StringType,
						"operator": types.StringType,
						"value":    types.StringType,
					},
				}),
			},
		},
		{
			description: "null_issuer_and_name",
			model: &Model{
				Name:       types.StringNull(),
				Issuer:     types.StringNull(),
				Assertions: assertionsListFromModels(t, validAssertions[:1]),
			},
		},
		{
			description: "invalid_assertions_type",
			model: &Model{
				Name:       types.StringValue("provider-name"),
				Issuer:     types.StringValue("https://issuer.example.com"),
				Assertions: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("not-an-object")}),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toUpdatePayload(t.Context(), tt.model)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if payload != nil {
					t.Fatalf("expected nil payload on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			switch tt.description {
			case "all_fields_set":
				if payload.Name != "provider-name" {
					t.Fatalf("name mismatch: got %q", payload.Name)
				}
				if payload.Issuer != "https://issuer.example.com" {
					t.Fatalf("issuer mismatch: got %q", payload.Issuer)
				}
				if len(payload.Assertions) != 2 {
					t.Fatalf("assertions length mismatch: got %d, expected 2", len(payload.Assertions))
				}
				if payload.Assertions[0].Item == nil || *payload.Assertions[0].Item != "aud" {
					t.Fatalf("assertions[0].item mismatch")
				}
				if payload.Assertions[0].Operator == nil || *payload.Assertions[0].Operator != "equals" {
					t.Fatalf("assertions[0].operator mismatch")
				}
				if payload.Assertions[0].Value == nil || *payload.Assertions[0].Value != "https://example.com" {
					t.Fatalf("assertions[0].value mismatch")
				}
				if payload.Assertions[1].Item == nil || *payload.Assertions[1].Item != "sub" {
					t.Fatalf("assertions[1].item mismatch")
				}
			case "null_assertions_replaces_external":
				if len(payload.Assertions) != 0 {
					t.Fatalf("expected assertions to be empty when null, got %d", len(payload.Assertions))
				}
			case "null_issuer_and_name":
				if payload.Issuer != "" {
					t.Fatalf("expected empty issuer for null, got %q", payload.Issuer)
				}
				if payload.Name != "" {
					t.Fatalf("expected empty name for null, got %q", payload.Name)
				}
				if len(payload.Assertions) != 1 {
					t.Fatalf("assertions length mismatch: got %d, expected 1", len(payload.Assertions))
				}
			}
		})
	}
}
