package objectstorage

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *objectstorage.CreateCredentialsGroupResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.CreateCredentialsGroupResponse{
				CredentialsGroup: &objectstorage.CredentialsGroup{},
			},
			Model{
				Id:                 types.StringValue("pid,cid"),
				Name:               types.StringNull(),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&objectstorage.CreateCredentialsGroupResponse{
				CredentialsGroup: &objectstorage.CredentialsGroup{
					DisplayName: utils.Ptr("name"),
					Urn:         utils.Ptr("urn"),
				},
			},
			Model{
				Id:                 types.StringValue("pid,cid"),
				Name:               types.StringValue("name"),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringValue("urn"),
			},
			true,
		},
		{
			"empty_strings",
			&objectstorage.CreateCredentialsGroupResponse{
				CredentialsGroup: &objectstorage.CredentialsGroup{
					DisplayName: utils.Ptr(""),
					Urn:         utils.Ptr(""),
				},
			},
			Model{
				Id:                 types.StringValue("pid,cid"),
				Name:               types.StringValue(""),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringValue(""),
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
			"no_bucket",
			&objectstorage.CreateCredentialsGroupResponse{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId:          tt.expected.ProjectId,
				CredentialsGroupId: tt.expected.CredentialsGroupId,
			}
			err := mapFields(tt.input, model)
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
