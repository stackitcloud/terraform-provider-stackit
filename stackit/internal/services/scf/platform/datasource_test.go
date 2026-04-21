package platform

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	scf "github.com/stackitcloud/stackit-sdk-go/services/scf/v1api"
)

var (
	testProjectId  = uuid.New().String()
	testPlatformId = uuid.New().String()
	testRegion     = "eu01"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *scf.Platforms
		expected    *Model
		isValid     bool
	}{
		{
			description: "minimal_input",
			input: &scf.Platforms{
				Guid:   testPlatformId,
				Region: testRegion,
			},
			expected: &Model{
				Id:          types.StringValue(fmt.Sprintf("%s,%s,%s", testProjectId, testRegion, testPlatformId)),
				PlatformId:  types.StringValue(testPlatformId),
				ProjectId:   types.StringValue(testProjectId),
				Region:      types.StringValue(testRegion),
				SystemId:    types.StringValue(""),
				DisplayName: types.StringValue(""),
				ApiUrl:      types.StringValue(""),
				ConsoleUrl:  types.StringNull(),
			},
			isValid: true,
		},
		{
			description: "max_input",
			input: &scf.Platforms{
				Guid:        testPlatformId,
				SystemId:    "eu01.01",
				DisplayName: "scf-full-org",
				Region:      testRegion,
				ApiUrl:      "https://example.scf.stackit.cloud",
				ConsoleUrl:  new("https://example.console.scf.stackit.cloud"),
			},
			expected: &Model{
				Id:          types.StringValue(fmt.Sprintf("%s,%s,%s", testProjectId, testRegion, testPlatformId)),
				ProjectId:   types.StringValue(testProjectId),
				PlatformId:  types.StringValue(testPlatformId),
				Region:      types.StringValue(testRegion),
				SystemId:    types.StringValue("eu01.01"),
				DisplayName: types.StringValue("scf-full-org"),
				ApiUrl:      types.StringValue("https://example.scf.stackit.cloud"),
				ConsoleUrl:  types.StringValue("https://example.console.scf.stackit.cloud"),
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
