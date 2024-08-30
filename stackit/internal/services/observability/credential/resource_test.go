package observability

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *observability.Credentials
		expected    Model
		isValid     bool
	}{
		{
			"ok",
			&observability.Credentials{
				Username: utils.Ptr("username"),
				Password: utils.Ptr("password"),
			},
			Model{
				Id:         types.StringValue("pid,iid,username"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("iid"),
				Username:   types.StringValue("username"),
				Password:   types.StringValue("password"),
			},
			true,
		},
		{
			"response_nil_fail",
			nil,
			Model{},
			false,
		},
		{
			"response_fields_nil_fail",
			&observability.Credentials{
				Password: nil,
				Username: nil,
			},
			Model{},
			false,
		},
		{
			"no_resource_id",
			&observability.Credentials{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
