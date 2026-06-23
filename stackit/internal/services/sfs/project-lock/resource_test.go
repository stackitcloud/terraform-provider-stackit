package projectlock

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"
)

func Test_mapFields(t *testing.T) {
	const projectId = "eu01"
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s", projectId, testRegion)
	tests := []struct {
		name     string
		response sfsLockResponse
		expected Model
		isValid  bool
	}{
		{
			name: "default_values",
			response: &sfs.GetLockResponse{
				LockId:               new("lock-id"),
				AdditionalProperties: nil,
			},
			expected: Model{
				Id:        types.StringValue(id),
				ProjectId: types.StringValue(projectId),
				Region:    types.StringValue(testRegion),
				LockId:    types.StringValue("lock-id"),
			},
			isValid: true,
		},
		{
			name: "lock id nil",
			response: &sfs.GetLockResponse{
				LockId:               nil,
				AdditionalProperties: nil,
			},
			expected: Model{
				Id:        types.StringValue(id),
				ProjectId: types.StringValue(projectId),
				Region:    types.StringValue(testRegion),
				LockId:    types.StringNull(),
			},
			isValid: true,
		},
		{
			name:     "nil_response",
			response: nil,
			expected: Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &Model{
				ProjectId: tt.expected.ProjectId,
			}
			err := mapFields(tt.response, model, "eu01")
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(model, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
