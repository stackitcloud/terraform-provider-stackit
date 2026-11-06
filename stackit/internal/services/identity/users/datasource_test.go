package users

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	identity "github.com/stackitcloud/stackit-sdk-go/services/identity/v1api"
)

func TestMapUsersDataSourceFields(t *testing.T) {
	orgId := "test-org-id"

	emailA := "alice@example.com"
	emailB := "bob@example.com"
	emailC := "charlie@stackit.cloud"

	nameA := "Alice"
	nameB := "Bob"
	nameC := "Charlie"

	idA := "uuid-1"
	idB := "uuid-2"
	idC := "uuid-3"

	tests := []struct {
		description   string
		apiItems      []identity.GetUserResponse
		initialModel  Model
		regexStr      string
		expectedModel Model
		isValid       bool
	}{
		{
			description: "default_sort_descending",
			apiItems: []identity.GetUserResponse{
				{Email: &emailA, Id: &idA, DisplayName: &nameA},
				{Email: &emailC, Id: &idC, DisplayName: &nameC},
				{Email: &emailB, Id: &idB, DisplayName: &nameB},
			},
			initialModel: Model{
				OrganizationId: types.StringValue(orgId),
				SortAscending:  types.BoolNull(),
			},
			expectedModel: Model{
				Id:             types.StringValue(orgId),
				OrganizationId: types.StringValue(orgId),
				SortAscending:  types.BoolNull(),
				Items: []Item{
					{Email: types.StringValue(emailC), UserId: types.StringValue(idC), DisplayName: types.StringValue(nameC)},
					{Email: types.StringValue(emailB), UserId: types.StringValue(idB), DisplayName: types.StringValue(nameB)},
					{Email: types.StringValue(emailA), UserId: types.StringValue(idA), DisplayName: types.StringValue(nameA)},
				},
			},
			isValid: true,
		},
		{
			description: "sort_ascending",
			apiItems: []identity.GetUserResponse{
				{Email: &emailC, Id: &idC, DisplayName: &nameC},
				{Email: &emailA, Id: &idA, DisplayName: &nameA},
				{Email: &emailB, Id: &idB, DisplayName: &nameB},
			},
			initialModel: Model{
				OrganizationId: types.StringValue(orgId),
				SortAscending:  types.BoolValue(true),
			},
			expectedModel: Model{
				Id:             types.StringValue(orgId),
				OrganizationId: types.StringValue(orgId),
				SortAscending:  types.BoolValue(true),
				Items: []Item{
					{Email: types.StringValue(emailA), UserId: types.StringValue(idA), DisplayName: types.StringValue(nameA)},
					{Email: types.StringValue(emailB), UserId: types.StringValue(idB), DisplayName: types.StringValue(nameB)},
					{Email: types.StringValue(emailC), UserId: types.StringValue(idC), DisplayName: types.StringValue(nameC)},
				},
			},
			isValid: true,
		},
		{
			description: "regex_filter_match",
			apiItems: []identity.GetUserResponse{
				{Email: &emailA, Id: &idA, DisplayName: &nameA},
				{Email: &emailB, Id: &idB, DisplayName: &nameB},
				{Email: &emailC, Id: &idC, DisplayName: &nameC},
			},
			initialModel: Model{
				OrganizationId: types.StringValue(orgId),
				EmailRegex:     types.StringValue(`.*@stackit\.cloud`),
				SortAscending:  types.BoolValue(true),
			},
			regexStr: `.*@stackit\.cloud`,
			expectedModel: Model{
				Id:             types.StringValue(orgId),
				OrganizationId: types.StringValue(orgId),
				EmailRegex:     types.StringValue(`.*@stackit\.cloud`),
				SortAscending:  types.BoolValue(true),
				Items: []Item{
					{Email: types.StringValue(emailC), UserId: types.StringValue(idC), DisplayName: types.StringValue(nameC)},
				},
			},
			isValid: true,
		},
		{
			description:  "nil_model",
			apiItems:     []identity.GetUserResponse{},
			initialModel: Model{},
			isValid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var compiledRegex *regexp.Regexp
			if tt.regexStr != "" {
				compiledRegex = regexp.MustCompile(tt.regexStr)
			}

			var modelPtr *Model
			if tt.description != "nil_model" {
				modelCopy := tt.initialModel
				modelPtr = &modelCopy
			}

			err := mapUsersDataSourceFields(tt.apiItems, modelPtr, compiledRegex)

			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}

			if tt.isValid {
				diff := cmp.Diff(*modelPtr, tt.expectedModel, cmp.AllowUnexported(types.String{}, types.Bool{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
