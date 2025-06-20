package instance

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
				Acl:                   nil,
				ConsumedDisk:          utils.Ptr("foo"),
				ConsumedObjectStorage: utils.Ptr("foo"),
				Created:               nil,
				Flavor:                utils.Ptr("foo"),
				Id:                    utils.Ptr("id"),
				Name:                  utils.Ptr("foo"),
				Url:                   utils.Ptr("https://foo.com"),
				Version:               utils.Ptr("v0.0.1"),
			},
			Model{
				ACL:                   types.ListNull(types.StringType),
				ConsumedDisk:          types.StringValue("foo"),
				ConsumedObjectStorage: types.StringValue("foo"),
				Created:               types.StringNull(),
				Flavor:                types.StringValue("foo"),
				Id:                    types.StringValue("pid,id"),
				InstanceId:            types.StringValue("id"),
				Name:                  types.StringValue("foo"),
				ProjectId:             types.StringValue("pid"),
				Url:                   types.StringValue("https://foo.com"),
				Version:               types.StringValue("v0.0.1"),
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
			err := mapFields(context.Background(), tt.input, state)
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

func TestCreatePayloadFromModel(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    git.CreateInstancePayload
		expectError bool
	}{
		{
			description: "default values",
			input: &Model{
				Name:   types.StringValue("example-instance"),
				Flavor: types.StringNull(),
				ACL:    types.ListNull(types.StringType),
			},
			expected: git.CreateInstancePayload{
				Name: utils.Ptr("example-instance"),
			},
			expectError: false,
		},
		{
			description: "simple values with ACL and Flavor",
			input: &Model{
				Name:   types.StringValue("my-instance"),
				Flavor: types.StringValue("git-100"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("10.0.0.1"),
					types.StringValue("10.0.0.2"),
				}),
			},
			expected: git.CreateInstancePayload{
				Name:   utils.Ptr("my-instance"),
				Flavor: utils.Ptr("git-100"),
				Acl:    &[]string{"10.0.0.1", "10.0.0.2"},
			},
			expectError: false,
		},
		{
			description: "empty ACL still valid",
			input: &Model{
				Name:   types.StringValue("my-instance"),
				Flavor: types.StringValue("git-101"),
				ACL:    types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: git.CreateInstancePayload{
				Name:   utils.Ptr("my-instance"),
				Flavor: utils.Ptr("git-101"),
				Acl:    &[]string{},
			},
			expectError: false,
		},
		{
			description: "nil input model",
			input:       nil,
			expected:    git.CreateInstancePayload{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, diags := createPayloadFromModel(context.Background(), tt.input)

			if tt.expectError && !diags.HasError() {
				t.Fatalf("expected diagnostics error but got none")
			}

			if !tt.expectError && diags.HasError() {
				t.Fatalf("unexpected diagnostics error: %v", diags)
			}

			if diff := cmp.Diff(tt.expected, output); diff != "" {
				t.Fatalf("unexpected payload (-want +got):\n%s", diff)
			}
		})
	}
}
