package organization

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
	testQuotaId    = uuid.New().String()
	testRegion     = "eu01"
)

func TestMapFields(t *testing.T) {
	createdTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", "2025-01-01 00:00:00 +0000 UTC")
	if err != nil {
		t.Fatalf("failed to parse test time: %v", err)
	}

	tests := []struct {
		description string
		input       *scf.Organization
		expected    *Model
		isValid     bool
	}{
		{
			description: "minimal_input",
			input: &scf.Organization{
				Guid:      testOrgId,
				Name:      "scf-org-min-instance",
				Region:    testRegion,
				CreatedAt: createdTime,
				UpdatedAt: createdTime,
				ProjectId: testProjectId,
			},
			expected: &Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s", testProjectId, testRegion, testOrgId)),
				ProjectId:  types.StringValue(testProjectId),
				Region:     types.StringValue(testRegion),
				Name:       types.StringValue("scf-org-min-instance"),
				PlatformId: types.StringValue(""),
				OrgId:      types.StringValue(testOrgId),
				QuotaId:    types.StringValue(""),
				Status:     types.StringValue(""),
				Suspended:  types.BoolValue(false),
				CreateAt:   types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
				UpdatedAt:  types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
			},
			isValid: true,
		},
		{
			description: "max_input",
			input: &scf.Organization{
				CreatedAt:  createdTime,
				Guid:       testOrgId,
				Name:       "scf-full-org",
				PlatformId: testPlatformId,
				ProjectId:  testProjectId,
				QuotaId:    testQuotaId,
				Region:     testRegion,
				Status:     "",
				Suspended:  true,
				UpdatedAt:  createdTime,
			},
			expected: &Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,%s", testProjectId, testRegion, testOrgId)),
				ProjectId:  types.StringValue(testProjectId),
				OrgId:      types.StringValue(testOrgId),
				Name:       types.StringValue("scf-full-org"),
				Region:     types.StringValue(testRegion),
				PlatformId: types.StringValue(testPlatformId),
				CreateAt:   types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
				UpdatedAt:  types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
				QuotaId:    types.StringValue(testQuotaId),
				Status:     types.StringValue(""),
				Suspended:  types.BoolValue(true),
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
			err := mapFields(tt.input, state)

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

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    scf.CreateOrganizationPayload
		expectError bool
	}{
		{
			description: "default values",
			input: &Model{
				Name:       types.StringValue("example-org"),
				PlatformId: types.StringValue(testPlatformId),
			},
			expected: scf.CreateOrganizationPayload{
				Name:       "example-org",
				PlatformId: &testPlatformId,
			},
			expectError: false,
		},
		{
			description: "nil input model",
			input:       nil,
			expected:    scf.CreateOrganizationPayload{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)

			if tt.expectError && err == nil {
				t.Fatalf("expected diagnostics error but got none")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("unexpected diagnostics error: %v", err)
			}

			if diff := cmp.Diff(tt.expected, output); diff != "" {
				t.Fatalf("unexpected payload (-want +got):\n%s", diff)
			}
		})
	}
}
