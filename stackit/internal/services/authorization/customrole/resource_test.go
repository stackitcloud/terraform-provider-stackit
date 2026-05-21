package customrole

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	authorization "github.com/stackitcloud/stackit-sdk-go/services/authorization/v2api"
)

var (
	testRoleId     = uuid.New().String()
	testResourceId = uuid.New().String()
)

type testCase struct {
	description string
	input       *authorization.GetRoleResponse
	expected    *Model
	isValid     bool
}

func allResourceTypes(fn func(resourceType string) []testCase) []testCase {
	var tests []testCase

	for _, resourceType := range resourceTypes {
		tests = append(tests, fn(resourceType)...)
	}

	return tests
}

func TestMapFields(t *testing.T) {
	tests := allResourceTypes(func(resourceType string) []testCase {
		return []testCase{
			{
				description: fmt.Sprintf("full_input_%s", resourceType),
				input: &authorization.GetRoleResponse{
					ResourceId:   testResourceId,
					ResourceType: resourceType,
					Role: authorization.Role{
						Id:          &testRoleId,
						Name:        "role-name",
						Description: "Some description",
						Permissions: []authorization.Permission{
							{
								Name:        "iam.subject.get",
								Description: "Can read subjects.",
							},
						},
					},
				},
				expected: &Model{
					Id:          types.StringValue(fmt.Sprintf("%s,%s", testResourceId, testRoleId)),
					RoleId:      types.StringValue(testRoleId),
					ResourceId:  types.StringValue(testResourceId),
					Name:        types.StringValue("role-name"),
					Description: types.StringValue("Some description"),
					Permissions: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("iam.subject.get"),
					}),
				},
				isValid: true,
			},
			{
				description: fmt.Sprintf("partial_input_%s", resourceType),
				input: &authorization.GetRoleResponse{
					ResourceId:   testResourceId,
					ResourceType: resourceType,
					Role: authorization.Role{
						Id:          &testRoleId,
						Name:        "role-name",
						Description: "Some description",
						Permissions: []authorization.Permission{
							{
								Name: "iam.subject.get",
							},
						},
					},
				},
				expected: &Model{
					Id:          types.StringValue(fmt.Sprintf("%s,%s", testResourceId, testRoleId)),
					RoleId:      types.StringValue(testRoleId),
					ResourceId:  types.StringValue(testResourceId),
					Name:        types.StringValue("role-name"),
					Description: types.StringValue("Some description"),
					Permissions: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("iam.subject.get"),
					}),
				},
				isValid: true,
			},
			{
				description: fmt.Sprintf("missing_role_%s", resourceType),
				input: &authorization.GetRoleResponse{
					ResourceId:   testResourceId,
					ResourceType: resourceType,
				},
				expected: nil,
				isValid:  false,
			},
			{
				description: fmt.Sprintf("missing_permissions_%s", resourceType),
				input: &authorization.GetRoleResponse{
					ResourceId:   testResourceId,
					ResourceType: resourceType,
					Role: authorization.Role{
						Id: &testRoleId,
					},
				},
				expected: nil,
				isValid:  false,
			},
			{
				description: fmt.Sprintf("missing_role_id_%s", resourceType),
				input: &authorization.GetRoleResponse{
					ResourceId:   testResourceId,
					ResourceType: resourceType,
					Role: authorization.Role{
						Permissions: []authorization.Permission{},
					},
				},
				expected: nil,
				isValid:  false,
			},
			{
				description: fmt.Sprintf("missing_role_%s", resourceType),
				input: &authorization.GetRoleResponse{
					ResourceId:   testResourceId,
					ResourceType: resourceType,
				},
				expected: nil,
				isValid:  false,
			},
			{
				description: fmt.Sprintf("missing_permissions_%s", resourceType),
				input: &authorization.GetRoleResponse{
					ResourceId:   testResourceId,
					ResourceType: resourceType,
					Role: authorization.Role{
						Id: &testRoleId,
					},
				},
				expected: nil,
				isValid:  false,
			},
		}
	})

	tests = append(tests, []testCase{
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
	}...)

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
				Name:        "role-name",
				Description: "Some description",
				Permissions: []authorization.PermissionRequest{
					{
						Name: "iam.subject.get",
					},
				},
			},
		},
		{
			description: "empty values still valid",
			input:       &Model{},
			expected: authorization.AddRolePayload{
				Permissions: []authorization.PermissionRequest{},
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
