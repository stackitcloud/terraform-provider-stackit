package groups

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	identity "github.com/stackitcloud/stackit-sdk-go/services/identity/v1api"
)

func TestMapGroupsDataSourceFields(t *testing.T) {
	orgId := "test-org-id"

	nameA := "group-a"
	nameB := "group-b"
	nameC := "special-group-c"

	idA := "uuid-1"
	idB := "uuid-2"
	idC := "uuid-3"

	tests := []struct {
		description   string
		apiItems      []identity.GetGroupResponse
		initialModel  Model
		regexStr      string
		expectedModel Model
		isValid       bool
	}{
		{
			description: "default_sort_descending",
			apiItems: []identity.GetGroupResponse{
				{DisplayName: &nameA, Id: &idA},
				{DisplayName: &nameC, Id: &idC},
				{DisplayName: &nameB, Id: &idB},
			},
			initialModel: Model{
				OrganizationId: types.StringValue(orgId),
				SortAscending:  types.BoolNull(),
			},
			expectedModel: Model{
				Id:             types.StringValue(orgId),
				OrganizationId: types.StringValue(orgId),
				SortAscending:  types.BoolNull(),
				Items: []GroupItem{
					{DisplayName: types.StringValue(nameC), GroupId: types.StringValue(idC), Description: types.StringNull()},
					{DisplayName: types.StringValue(nameB), GroupId: types.StringValue(idB), Description: types.StringNull()},
					{DisplayName: types.StringValue(nameA), GroupId: types.StringValue(idA), Description: types.StringNull()},
				},
			},
			isValid: true,
		},
		{
			description: "sort_ascending",
			apiItems: []identity.GetGroupResponse{
				{DisplayName: &nameC, Id: &idC},
				{DisplayName: &nameA, Id: &idA},
				{DisplayName: &nameB, Id: &idB},
			},
			initialModel: Model{
				OrganizationId: types.StringValue(orgId),
				SortAscending:  types.BoolValue(true),
			},
			expectedModel: Model{
				Id:             types.StringValue(orgId),
				OrganizationId: types.StringValue(orgId),
				SortAscending:  types.BoolValue(true),
				Items: []GroupItem{
					{DisplayName: types.StringValue(nameA), GroupId: types.StringValue(idA), Description: types.StringNull()},
					{DisplayName: types.StringValue(nameB), GroupId: types.StringValue(idB), Description: types.StringNull()},
					{DisplayName: types.StringValue(nameC), GroupId: types.StringValue(idC), Description: types.StringNull()},
				},
			},
			isValid: true,
		},
		{
			description: "regex_filter_match",
			apiItems: []identity.GetGroupResponse{
				{DisplayName: &nameA, Id: &idA},
				{DisplayName: &nameB, Id: &idB},
				{DisplayName: &nameC, Id: &idC},
			},
			initialModel: Model{
				OrganizationId:   types.StringValue(orgId),
				DisplayNameRegex: types.StringValue(`.*special-.*`),
				SortAscending:    types.BoolValue(true),
			},
			regexStr: `.*special-.*`,
			expectedModel: Model{
				Id:               types.StringValue(orgId),
				OrganizationId:   types.StringValue(orgId),
				DisplayNameRegex: types.StringValue(`.*special-.*`),
				SortAscending:    types.BoolValue(true),
				Items: []GroupItem{
					{DisplayName: types.StringValue(nameC), GroupId: types.StringValue(idC), Description: types.StringNull()},
				},
			},
			isValid: true,
		},
		{
			description:  "nil_model",
			apiItems:     []identity.GetGroupResponse{},
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

			err := mapGroupsDataSourceFields(tt.apiItems, modelPtr, compiledRegex)

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
