package accounts

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"
)

func TestMapDataSourceFields(t *testing.T) {
	projectId := "test-project-id"
	emailA := "sa-a-1234567@sa.stackit.cloud"
	emailB := "sa-b-1234567@sa.stackit.cloud"
	emailC := "sa-c-1234567@ske.sa.stackit.cloud"

	idA := "550e8400-e29b-41d4-a716-446655440001"
	idB := "550e8400-e29b-41d4-a716-446655440002"
	idC := "550e8400-e29b-41d4-a716-446655440003"

	nameA := "sa-a"
	nameB := "sa-b"
	nameC := "sa-c"

	tests := []struct {
		description   string
		apiItems      []serviceaccount.ServiceAccount
		initialModel  ServiceAccountsModel
		regexStr      string
		expectedModel ServiceAccountsModel
		isValid       bool
	}{
		{
			description: "default_sort_descending",
			apiItems: []serviceaccount.ServiceAccount{
				{Email: emailA, Id: idA},
				{Email: emailC, Id: idC},
				{Email: emailB, Id: idB},
			},
			initialModel: ServiceAccountsModel{
				ProjectId:     types.StringValue(projectId),
				SortAscending: types.BoolNull(), // Default should trigger descending sort
			},
			expectedModel: ServiceAccountsModel{
				Id:            types.StringValue(projectId),
				ProjectId:     types.StringValue(projectId),
				SortAscending: types.BoolNull(),
				Items: []ServiceAccountItem{
					{Email: types.StringValue(emailC), Name: types.StringValue(nameC), ServiceAccountId: types.StringValue(idC)},
					{Email: types.StringValue(emailB), Name: types.StringValue(nameB), ServiceAccountId: types.StringValue(idB)},
					{Email: types.StringValue(emailA), Name: types.StringValue(nameA), ServiceAccountId: types.StringValue(idA)},
				},
			},
			isValid: true,
		},
		{
			description: "sort_ascending",
			apiItems: []serviceaccount.ServiceAccount{
				{Email: emailC, Id: idC},
				{Email: emailA, Id: idA},
				{Email: emailB, Id: idB},
			},
			initialModel: ServiceAccountsModel{
				ProjectId:     types.StringValue(projectId),
				SortAscending: types.BoolValue(true),
			},
			expectedModel: ServiceAccountsModel{
				Id:            types.StringValue(projectId),
				ProjectId:     types.StringValue(projectId),
				SortAscending: types.BoolValue(true),
				Items: []ServiceAccountItem{
					{Email: types.StringValue(emailA), Name: types.StringValue(nameA), ServiceAccountId: types.StringValue(idA)},
					{Email: types.StringValue(emailB), Name: types.StringValue(nameB), ServiceAccountId: types.StringValue(idB)},
					{Email: types.StringValue(emailC), Name: types.StringValue(nameC), ServiceAccountId: types.StringValue(idC)},
				},
			},
			isValid: true,
		},
		{
			description: "regex_filter_match",
			apiItems: []serviceaccount.ServiceAccount{
				{Email: emailA, Id: idA},
				{Email: emailB, Id: idB},
				{Email: emailC, Id: idC},
			},
			initialModel: ServiceAccountsModel{
				ProjectId:     types.StringValue(projectId),
				EmailRegex:    types.StringValue(`.*-b-.*`),
				SortAscending: types.BoolValue(true),
			},
			regexStr: `.*-b-.*`,
			expectedModel: ServiceAccountsModel{
				Id:            types.StringValue(projectId),
				ProjectId:     types.StringValue(projectId),
				EmailRegex:    types.StringValue(`.*-b-.*`),
				SortAscending: types.BoolValue(true),
				Items: []ServiceAccountItem{
					{Email: types.StringValue(emailB), Name: types.StringValue(nameB), ServiceAccountId: types.StringValue(idB)},
				},
			},
			isValid: true,
		},
		{
			description: "suffix_filter_match",
			apiItems: []serviceaccount.ServiceAccount{
				{Email: emailA, Id: idA},
				{Email: emailB, Id: idB},
				{Email: emailC, Id: idC},
			},
			initialModel: ServiceAccountsModel{
				ProjectId:   types.StringValue(projectId),
				EmailSuffix: types.StringValue(`@ske.sa.stackit.cloud`),
			},
			expectedModel: ServiceAccountsModel{
				Id:          types.StringValue(projectId),
				ProjectId:   types.StringValue(projectId),
				EmailSuffix: types.StringValue(`@ske.sa.stackit.cloud`),
				Items: []ServiceAccountItem{
					{Email: types.StringValue(emailC), Name: types.StringValue(nameC), ServiceAccountId: types.StringValue(idC)},
				},
			},
			isValid: true,
		},
		{
			description:  "nil_model",
			apiItems:     []serviceaccount.ServiceAccount{},
			initialModel: ServiceAccountsModel{},
			isValid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var compiledRegex *regexp.Regexp
			if tt.regexStr != "" {
				compiledRegex = regexp.MustCompile(tt.regexStr)
			}

			// Handle nil model scenario
			var modelPtr *ServiceAccountsModel
			if tt.description != "nil_model" {
				modelCopy := tt.initialModel
				modelPtr = &modelCopy
			}

			err := mapDataSourceFields(tt.apiItems, modelPtr, compiledRegex)

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
