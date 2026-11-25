package members

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	identity "github.com/stackitcloud/stackit-sdk-go/services/identity/v1api"
)

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       []MemberItem
		expected    identity.UpdateGroupMembersPayload
	}{
		{
			"default_values",
			[]MemberItem{
				{
					Id:   types.StringValue("user-id-123"),
					Type: types.StringValue("user"),
				},
				{
					Id:   types.StringValue("sa-id-456"),
					Type: types.StringValue("service_account"),
				},
			},
			identity.UpdateGroupMembersPayload{
				Members: []identity.UpdateMemberRequest{
					{
						Id:   "user-id-123",
						Type: identity.UpdateMemberRequestType("user"),
					},
					{
						Id:   "sa-id-456",
						Type: identity.UpdateMemberRequestType("service_account"),
					},
				},
			},
		},
		{
			"empty_members",
			[]MemberItem{},
			identity.UpdateGroupMembersPayload{
				Members: []identity.UpdateMemberRequest{},
			},
		},
		{
			"nil_input",
			nil,
			identity.UpdateGroupMembersPayload{
				Members: []identity.UpdateMemberRequest{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := toUpdatePayload(tt.input)
			diff := cmp.Diff(output, tt.expected)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func TestMapMembersResponse(t *testing.T) {
	userId := "user-id-123"
	userType := identity.MemberType("user")

	saId := "sa-id-456"
	saType := identity.MemberType("service_account")

	tests := []struct {
		description string
		input       *identity.MembersResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&identity.MembersResponse{
				Members: []identity.MemberResponse{
					{
						Id:   &userId,
						Type: &userType,
					},
					{
						Id:   &saId,
						Type: &saType,
					},
				},
			},
			Model{
				Members: []MemberItem{
					{
						Id:   types.StringValue(userId),
						Type: types.StringValue(string(userType)),
					},
					{
						Id:   types.StringValue(saId),
						Type: types.StringValue(string(saType)),
					},
				},
			},
			true,
		},
		{
			"empty_members",
			&identity.MembersResponse{
				Members: []identity.MemberResponse{},
			},
			Model{
				Members: []MemberItem{},
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
			if !tt.isValid && tt.input == nil {
				return
			}

			state := &Model{}
			mapMembersResponse(tt.input, state)

			diff := cmp.Diff(*state, tt.expected, cmp.AllowUnexported(types.String{}))
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}
