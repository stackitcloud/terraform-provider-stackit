package organizationmanager

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/scf"
)

func TestMapFieldsDataSource(t *testing.T) {
	createdTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", "2025-01-01 00:00:00 +0000 UTC")
	if err != nil {
		t.Fatalf("failed to parse test time: %v", err)
	}

	tests := []struct {
		description string
		input       *scf.OrgManager
		expected    *DataSourceModel
		isValid     bool
	}{
		{
			description: "minimal_input",
			input: &scf.OrgManager{
				Guid:      utils.Ptr(testUserId),
				OrgId:     utils.Ptr(testOrgId),
				ProjectId: utils.Ptr(testProjectId),
				Region:    utils.Ptr(testRegion),
				CreatedAt: &createdTime,
				UpdatedAt: &createdTime,
			},
			expected: &DataSourceModel{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", testProjectId, testRegion, testOrgId, testUserId)),
				UserId:     types.StringValue(testUserId),
				OrgId:      types.StringValue(testOrgId),
				ProjectId:  types.StringValue(testProjectId),
				Region:     types.StringValue(testRegion),
				UserName:   types.StringNull(),
				PlatformId: types.StringNull(),
				CreateAt:   types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
				UpdatedAt:  types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
			},
			isValid: true,
		},
		{
			description: "max_input",
			input: &scf.OrgManager{
				Guid:       utils.Ptr(testUserId),
				OrgId:      utils.Ptr(testOrgId),
				ProjectId:  utils.Ptr(testProjectId),
				PlatformId: utils.Ptr(testPlatformId),
				Region:     utils.Ptr(testRegion),
				CreatedAt:  &createdTime,
				UpdatedAt:  &createdTime,
				Username:   utils.Ptr("test-user"),
			},
			expected: &DataSourceModel{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", testProjectId, testRegion, testOrgId, testUserId)),
				UserId:     types.StringValue(testUserId),
				OrgId:      types.StringValue(testOrgId),
				ProjectId:  types.StringValue(testProjectId),
				PlatformId: types.StringValue(testPlatformId),
				Region:     types.StringValue(testRegion),
				UserName:   types.StringValue("test-user"),
				CreateAt:   types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
				UpdatedAt:  types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
			},
			isValid: true,
		},
		{
			description: "nil_org",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
		{
			description: "empty_org",
			input:       &scf.OrgManager{},
			expected:    nil,
			isValid:     false,
		},
		{
			description: "missing_id",
			input: &scf.OrgManager{
				Username: utils.Ptr("scf-missing-id"),
			},
			expected: nil,
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &DataSourceModel{}
			if tt.expected != nil {
				state.ProjectId = tt.expected.ProjectId
			}
			err := mapFieldsDataSource(tt.input, state)

			if tt.isValid && err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tt.isValid {
				if diff := cmp.Diff(tt.expected, state); diff != "" {
					t.Errorf("unexpected diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}
