package organizationmanager

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	scf "github.com/stackitcloud/stackit-sdk-go/services/scf/v1api"
)

var (
	testOrgId      = uuid.New().String()
	testProjectId  = uuid.New().String()
	testPlatformId = uuid.New().String()
	testUserId     = uuid.New().String()
	testRegion     = "eu01"
)

func TestMapFields(t *testing.T) {
	createdTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", "2025-01-01 00:00:00 +0000 UTC")
	if err != nil {
		t.Fatalf("failed to parse test time: %v", err)
	}

	tests := []struct {
		description string
		input       *scf.OrgManager
		expected    *Model
		isValid     bool
	}{
		{
			description: "minimal_input",
			input: &scf.OrgManager{
				Guid:      testUserId,
				OrgId:     testOrgId,
				ProjectId: testProjectId,
				Region:    testRegion,
				CreatedAt: createdTime,
				UpdatedAt: createdTime,
			},
			expected: &Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", testProjectId, testRegion, testOrgId, testUserId)),
				UserId:     types.StringValue(testUserId),
				OrgId:      types.StringValue(testOrgId),
				ProjectId:  types.StringValue(testProjectId),
				Region:     types.StringValue(testRegion),
				UserName:   types.StringValue(""),
				PlatformId: types.StringValue(""),
				CreateAt:   types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
				UpdatedAt:  types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
			},
			isValid: true,
		},
		{
			description: "max_input",
			input: &scf.OrgManager{
				Guid:       testUserId,
				OrgId:      testOrgId,
				ProjectId:  testProjectId,
				PlatformId: testPlatformId,
				Region:     testRegion,
				CreatedAt:  createdTime,
				UpdatedAt:  createdTime,
				Username:   "test-user",
			},
			expected: &Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", testProjectId, testRegion, testOrgId, testUserId)),
				UserId:     types.StringValue(testUserId),
				OrgId:      types.StringValue(testOrgId),
				ProjectId:  types.StringValue(testProjectId),
				PlatformId: types.StringValue(testPlatformId),
				Region:     types.StringValue(testRegion),
				Password:   types.StringNull(),
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
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{}
			if tt.expected != nil {
				state.ProjectId = tt.expected.ProjectId
			}
			err := mapFieldsRead(tt.input, state)

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

func TestMapFieldsCreate(t *testing.T) {
	createdTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", "2025-01-01 00:00:00 +0000 UTC")
	if err != nil {
		t.Fatalf("failed to parse test time: %v", err)
	}

	tests := []struct {
		description string
		input       *scf.OrgManagerResponse
		expected    *Model
		isValid     bool
	}{
		{
			description: "minimal_input",
			input: &scf.OrgManagerResponse{
				Guid:      testUserId,
				OrgId:     testOrgId,
				ProjectId: testProjectId,
				Region:    testRegion,
				CreatedAt: createdTime,
				UpdatedAt: createdTime,
			},
			expected: &Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", testProjectId, testRegion, testOrgId, testUserId)),
				UserId:     types.StringValue(testUserId),
				OrgId:      types.StringValue(testOrgId),
				ProjectId:  types.StringValue(testProjectId),
				Region:     types.StringValue(testRegion),
				UserName:   types.StringValue(""),
				PlatformId: types.StringValue(""),
				Password:   types.StringValue(""),
				CreateAt:   types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
				UpdatedAt:  types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
			},
			isValid: true,
		},
		{
			description: "max_input",
			input: &scf.OrgManagerResponse{
				Guid:       testUserId,
				OrgId:      testOrgId,
				ProjectId:  testProjectId,
				PlatformId: testPlatformId,
				Region:     testRegion,
				CreatedAt:  createdTime,
				UpdatedAt:  createdTime,
				Username:   "test-user",
				Password:   "test-password",
			},
			expected: &Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s,%s", testProjectId, testRegion, testOrgId, testUserId)),
				UserId:     types.StringValue(testUserId),
				OrgId:      types.StringValue(testOrgId),
				ProjectId:  types.StringValue(testProjectId),
				PlatformId: types.StringValue(testPlatformId),
				Region:     types.StringValue(testRegion),
				UserName:   types.StringValue("test-user"),
				Password:   types.StringValue("test-password"),
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
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{}
			if tt.expected != nil {
				state.ProjectId = tt.expected.ProjectId
			}
			err := mapFieldsCreate(tt.input, state)

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
