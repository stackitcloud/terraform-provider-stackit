package customrole

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
)

var (
	testRoleId    = uuid.New().String()
	testProjectId = uuid.New().String()
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *authorization.GetRoleResponse
		expected    *Model
		isValid     bool
	}{
		{
			description: "full_input",
			input: &authorization.GetRoleResponse{
				ResourceId:   &testProjectId,
				ResourceType: utils.Ptr("project"),
				Role: utils.Ptr(authorization.Role{
					Id:          &testRoleId,
					Name:        utils.Ptr("role-name"),
					Description: utils.Ptr("Some description"),
					Permissions: utils.Ptr([]authorization.Permission{
						{
							Name:        utils.Ptr("iam.subject.get"),
							Description: utils.Ptr("Can read subjects."),
						},
					}),
				}),
			},
			expected: &Model{
				Id:          types.StringValue(fmt.Sprintf("%s,%s", testProjectId, testRoleId)),
				RoleId:      types.StringValue(testRoleId),
				ResourceId:  types.StringValue(testProjectId),
				Name:        types.StringValue("role-name"),
				Description: types.StringValue("Some description"),
				Permissions: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("iam.subject.get"),
				}),
			},
			isValid: true,
		},
		{
			description: "partial_input",
			input: &authorization.GetRoleResponse{
				ResourceId:   &testProjectId,
				ResourceType: utils.Ptr("project"),
				Role: utils.Ptr(authorization.Role{
					Id: &testRoleId,
					Permissions: utils.Ptr([]authorization.Permission{
						{
							Name: utils.Ptr("iam.subject.get"),
						},
					}),
				}),
			},
			expected: &Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s", testProjectId, testRoleId)),
				RoleId:     types.StringValue(testRoleId),
				ResourceId: types.StringValue(testProjectId),
				Permissions: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("iam.subject.get"),
				}),
			},
			isValid: true,
		},
		{
			description: "partial_input_without_permissions",
			input: &authorization.GetRoleResponse{
				ResourceId:   &testProjectId,
				ResourceType: utils.Ptr("project"),
				Role: utils.Ptr(authorization.Role{
					Id:          &testRoleId,
					Permissions: utils.Ptr([]authorization.Permission{}),
				}),
			},
			expected: &Model{
				Id:          types.StringValue(fmt.Sprintf("%s,%s", testProjectId, testRoleId)),
				RoleId:      types.StringValue(testRoleId),
				ResourceId:  types.StringValue(testProjectId),
				Permissions: types.ListNull(types.StringType),
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
			input:       &authorization.GetRoleResponse{},
			expected:    nil,
			isValid:     false,
		},
		{
			description: "missing_role",
			input: &authorization.GetRoleResponse{
				ResourceId:   &testProjectId,
				ResourceType: utils.Ptr("project"),
			},
			expected: nil,
			isValid:  false,
		},
		{
			description: "missing_permissions",
			input: &authorization.GetRoleResponse{
				ResourceId:   &testProjectId,
				ResourceType: utils.Ptr("project"),
				Role: utils.Ptr(authorization.Role{
					Id: &testRoleId,
				}),
			},
			expected: nil,
			isValid:  false,
		},
		{
			description: "missing_role_id",
			input: &authorization.GetRoleResponse{
				ResourceId:   &testProjectId,
				ResourceType: utils.Ptr("project"),
				Role: utils.Ptr(authorization.Role{
					Permissions: utils.Ptr([]authorization.Permission{}),
				}),
			},
			expected: nil,
			isValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{}
			err := mapGetCustomRoleResponse(context.Background(), tt.input, state)

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
		expected    authorization.AddRolePayload
		expectError bool
	}{
		{
			description: "all values",
			input: &Model{
				Name:        types.StringValue("role-name"),
				Description: types.StringValue("Some description"),
				Permissions: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("iam.subject.get"),
				}),
			},
			expected: authorization.AddRolePayload{
				Name:        utils.Ptr("role-name"),
				Description: utils.Ptr("Some description"),
				Permissions: utils.Ptr([]authorization.PermissionRequest{
					{
						Name: utils.Ptr("iam.subject.get"),
					},
				}),
			},
		},
		{
			description: "empty values still valid",
			input:       &Model{},
			expected: authorization.AddRolePayload{
				Permissions: utils.Ptr([]authorization.PermissionRequest{}),
			},
			expectError: false,
		},
		{
			description: "nil input model",
			input:       nil,
			expected:    authorization.AddRolePayload{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)

			if tt.expectError && err == nil {
				t.Fatalf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectError && output == nil {
				// skip diff when error was expected
				return
			}

			if diff := cmp.Diff(&tt.expected, output); diff != "" {
				t.Fatalf("unexpected payload (-want +got):\n%s", diff)
			}
		})
	}
}
