package instance

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/git"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *git.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&git.Instance{
				Created: nil,
				Id:      utils.Ptr("id"),
				Name:    utils.Ptr("foo"),
				Url:     utils.Ptr("https://foo.com"),
				Version: utils.Ptr("v0.0.1"),
			},
			Model{
				Id:         types.StringValue("pid,id"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Name:       types.StringValue("foo"),
				Url:        types.StringValue("https://foo.com"),
				Version:    types.StringValue("v0.0.1"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"nil_response_2",
			&git.Instance{},
			Model{},
			false,
		},
		{
			"no_id",
			&git.Instance{
				Name: utils.Ptr("foo"),
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
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
