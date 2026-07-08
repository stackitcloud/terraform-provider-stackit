package objectstorage

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	defaultretention "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"
)

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	const testProjectId = "pid"
	const testBucketName = "bucket1"
	id := fmt.Sprintf("%s,%s,%s", testProjectId, testRegion, testBucketName)
	tests := []struct {
		description string
		input       *defaultretention.DefaultRetentionResponse
		expected    model
		isValid     bool
	}{
		{
			"simple_values",
			&defaultretention.DefaultRetentionResponse{
				Days:    2,
				Bucket:  testBucketName,
				Mode:    defaultretention.RETENTIONMODE_COMPLIANCE,
				Project: testProjectId,
			},
			model{
				Id:         types.StringValue(id),
				Days:       types.Int32Value(2),
				BucketName: types.StringValue(testBucketName),
				Mode:       types.StringValue(string(defaultretention.RETENTIONMODE_COMPLIANCE)),
				ProjectId:  types.StringValue(testProjectId),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			nil,
			model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &model{
				ProjectId: tt.expected.ProjectId,
			}
			err := mapFields(tt.input, model, "eu01")
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
