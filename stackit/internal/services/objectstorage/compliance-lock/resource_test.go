package compliancelock

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"
)

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s", "pid", testRegion)
	retentionDays := int32(30)
	tests := []struct {
		description string
		input       *objectstorage.ComplianceLockResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.ComplianceLockResponse{},
			Model{
				Id:               types.StringValue(id),
				ProjectId:        types.StringValue("pid"),
				Region:           types.StringValue("eu01"),
				MaxRetentionDays: types.Int32Value(0),
			},
			true,
		},
		{
			"simple_values",
			&objectstorage.ComplianceLockResponse{
				MaxRetentionDays: retentionDays,
			},
			Model{
				Id:               types.StringValue(id),
				ProjectId:        types.StringValue("pid"),
				Region:           types.StringValue("eu01"),
				MaxRetentionDays: types.Int32Value(retentionDays),
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
			model := &Model{
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
