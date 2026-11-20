package group

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	identity "github.com/stackitcloud/stackit-sdk-go/services/identity/v1api"
)

func TestToCreatePayload(t *testing.T) {
	desc := "example description"
	tests := []struct {
		description string
		input       *Model
		expected    *identity.CreateGroupPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{
				DisplayName: types.StringValue("example-group"),
				Description: types.StringNull(),
			},
			&identity.CreateGroupPayload{
				DisplayName: "example-group",
			},
			true,
		},
		{
			"with_description",
			&Model{
				DisplayName: types.StringValue("example-group"),
				Description: types.StringValue(desc),
			},
			&identity.CreateGroupPayload{
				DisplayName: "example-group",
				Description: &desc,
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapFields(t *testing.T) {
	groupId := "550e8400-e29b-41d4-a716-446655440000"
	orgId := "org-id-123"
	displayName := "example-group"
	description := "example description"

	tests := []struct {
		description string
		input       *identity.GetGroupResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&identity.GetGroupResponse{
				Id:          &groupId,
				DisplayName: &displayName,
			},
			Model{
				Id:             types.StringValue("org-id-123,550e8400-e29b-41d4-a716-446655440000"),
				OrganizationId: types.StringValue(orgId),
				GroupId:        types.StringValue(groupId),
				DisplayName:    types.StringValue(displayName),
				Description:    types.StringNull(),
			},
			true,
		},
		{
			"with_description",
			&identity.GetGroupResponse{
				Id:          &groupId,
				DisplayName: &displayName,
				Description: &description,
			},
			Model{
				Id:             types.StringValue("org-id-123,550e8400-e29b-41d4-a716-446655440000"),
				OrganizationId: types.StringValue(orgId),
				GroupId:        types.StringValue(groupId),
				DisplayName:    types.StringValue(displayName),
				Description:    types.StringValue(description),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				OrganizationId: tt.expected.OrganizationId,
			}
			err := mapFields(tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(*state, tt.expected, cmp.AllowUnexported(types.String{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
