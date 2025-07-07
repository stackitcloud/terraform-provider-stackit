package instance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/git"
)

var (
	testInstanceId = uuid.New().String()
	testProjectId  = uuid.New().String()
)

func TestMapFields(t *testing.T) {
	createdTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", "2025-01-01 00:00:00 +0000 UTC")
	if err != nil {
		t.Fatalf("failed to parse test time: %v", err)
	}

	tests := []struct {
		description string
		input       *git.Instance
		expected    *Model
		isValid     bool
	}{
		{
			description: "minimal_input_name_only",
			input: &git.Instance{
				Id:   utils.Ptr(testInstanceId),
				Name: utils.Ptr("git-min-instance"),
			},
			expected: &Model{
				Id:                    types.StringValue(fmt.Sprintf("%s,%s", testProjectId, testInstanceId)),
				ProjectId:             types.StringValue(testProjectId),
				InstanceId:            types.StringValue(testInstanceId),
				Name:                  types.StringValue("git-min-instance"),
				ACL:                   types.ListNull(types.StringType),
				Flavor:                types.StringNull(),
				Url:                   types.StringNull(),
				Version:               types.StringNull(),
				Created:               types.StringNull(),
				ConsumedDisk:          types.StringNull(),
				ConsumedObjectStorage: types.StringNull(),
			},
			isValid: true,
		},
		{
			description: "full_input_with_acl_and_flavor",
			input: &git.Instance{
				Acl:                   &[]string{"192.168.0.0/24"},
				ConsumedDisk:          utils.Ptr("1.00 GB"),
				ConsumedObjectStorage: utils.Ptr("2.00 GB"),
				Created:               &createdTime,
				Flavor:                utils.Ptr("git-100"),
				Id:                    utils.Ptr(testInstanceId),
				Name:                  utils.Ptr("git-full-instance"),
				Url:                   utils.Ptr("https://git-full-instance.git.onstackit.cloud"),
				Version:               utils.Ptr("v1.9.1"),
			},
			expected: &Model{
				Id:                    types.StringValue(fmt.Sprintf("%s,%s", testProjectId, testInstanceId)),
				ProjectId:             types.StringValue(testProjectId),
				InstanceId:            types.StringValue(testInstanceId),
				Name:                  types.StringValue("git-full-instance"),
				ACL:                   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("192.168.0.0/24")}),
				Flavor:                types.StringValue("git-100"),
				Url:                   types.StringValue("https://git-full-instance.git.onstackit.cloud"),
				Version:               types.StringValue("v1.9.1"),
				Created:               types.StringValue("2025-01-01 00:00:00 +0000 UTC"),
				ConsumedDisk:          types.StringValue("1.00 GB"),
				ConsumedObjectStorage: types.StringValue("2.00 GB"),
			},
			isValid: true,
		},
		{
			description: "empty_acls",
			input: &git.Instance{
				Id:   utils.Ptr(testInstanceId),
				Name: utils.Ptr("git-empty-acl"),
				Acl:  &[]string{},
			},
			expected: &Model{
				Id:                    types.StringValue(fmt.Sprintf("%s,%s", testProjectId, testInstanceId)),
				ProjectId:             types.StringValue(testProjectId),
				InstanceId:            types.StringValue(testInstanceId),
				Name:                  types.StringValue("git-empty-acl"),
				ACL:                   types.ListNull(types.StringType),
				Flavor:                types.StringNull(),
				Url:                   types.StringNull(),
				Version:               types.StringNull(),
				Created:               types.StringNull(),
				ConsumedDisk:          types.StringNull(),
				ConsumedObjectStorage: types.StringNull(),
			},
			isValid: true,
		},
		{
			description: "nil_instance",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
		{
			description: "empty_instance",
			input:       &git.Instance{},
			expected:    nil,
			isValid:     false,
		},
		{
			description: "missing_id",
			input: &git.Instance{
				Name: utils.Ptr("git-missing-id"),
			},
			expected: nil,
			isValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{}
			if tt.expected != nil {
				state.ProjectId = tt.expected.ProjectId
			}
			err := mapFields(context.Background(), tt.input, state)

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
			output, diags := toCreatePayload(context.Background(), tt.input)

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
