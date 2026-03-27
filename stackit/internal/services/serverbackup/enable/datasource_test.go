package enable

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
)

func TestDataMapFields(t *testing.T) {
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s,%s", "pid", "sid", testRegion)
	tests := []struct {
		description string
		input       *serverbackup.GetBackupServiceResponse
		expected    DataModel
		isValid     bool
	}{
		{
			"default_values",
			&serverbackup.GetBackupServiceResponse{},
			DataModel{
				Id:        types.StringValue(id),
				ProjectId: types.StringValue("pid"),
				ServerId:  types.StringValue("sid"),
				Region:    types.StringValue("eu01"),
			},
			true,
		},
		{
			"simple_values",
			&serverbackup.GetBackupServiceResponse{
				Enabled: utils.Ptr(true),
			},
			DataModel{
				Id:        types.StringValue(id),
				ProjectId: types.StringValue("pid"),
				ServerId:  types.StringValue("sid"),
				Region:    types.StringValue("eu01"),
				Enabled:   types.BoolValue(true),
			},
			true,
		},
		{
			"nil_response",
			nil,
			DataModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &DataModel{
				ProjectId: tt.expected.ProjectId,
				ServerId:  tt.expected.ServerId,
			}
			err := mapDataFields(tt.input, model, "eu01")
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
