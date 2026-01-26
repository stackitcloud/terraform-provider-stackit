package roleassignments

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	tfUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func TestToCreatePayload(t *testing.T) {
	apiName := "test-resource"

	tests := []struct {
		name        string
		input       *Model
		apiName     *string
		expectError bool
		expected    *authorization.AddMembersPayload
	}{
		{
			name: "valid model",
			input: &Model{
				Role:    types.StringValue("editor"),
				Subject: types.StringValue("foo.bar@stackit.cloud"),
			},
			apiName:     &apiName,
			expectError: false,
			expected: &authorization.AddMembersPayload{
				ResourceType: &apiName,
				Members: &[]authorization.Member{
					{
						Role:    utils.Ptr("editor"),
						Subject: utils.Ptr("foo.bar@stackit.cloud"),
					},
				},
			},
		},
		{
			name:        "nil model",
			input:       nil,
			apiName:     &apiName,
			expectError: true,
		},
		{
			name: "unknown role",
			input: &Model{
				Role:    types.StringUnknown(),
				Subject: types.StringValue("foo.bar@stackit.cloud"),
			},
			apiName:     &apiName,
			expectError: true,
		},
		{
			name: "empty role value",
			input: &Model{
				Role:    types.StringValue(""),
				Subject: types.StringValue("foo.bar@stackit.cloud"),
			},
			apiName:     &apiName,
			expectError: true,
		},
		{
			name: "unknown subject",
			input: &Model{
				Role:    types.StringValue("editor"),
				Subject: types.StringUnknown(),
			},
			apiName:     &apiName,
			expectError: true,
		},
		{
			name: "empty subject value",
			input: &Model{
				Role:    types.StringValue("editor"),
				Subject: types.StringValue(""),
			},
			apiName:     &apiName,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toCreatePayload(tt.input, tt.apiName)

			if tt.expectError && err == nil {
				t.Fatalf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if diff := cmp.Diff(tt.expected, got); diff != "" {
					t.Errorf("Payload mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMapListMembersResponse(t *testing.T) {
	role := "editor"
	subject := "foo.bar@stackit.cloud"
	resourceID := "project"

	tests := []struct {
		name        string
		resp        *authorization.ListMembersResponse
		inputModel  *Model
		expectError bool
		expected    *Model
	}{
		{
			name: "successfully maps values",
			resp: &authorization.ListMembersResponse{
				ResourceId: &resourceID,
				Members: &[]authorization.Member{
					{
						Role:    &role,
						Subject: &subject,
					},
				},
			},
			inputModel: &Model{
				Role:    types.StringValue(role),
				Subject: types.StringValue(subject),
			},
			expectError: false,
			expected: &Model{
				ResourceId: types.StringPointerValue(&resourceID),
				Role:       types.StringPointerValue(&role),
				Subject:    types.StringPointerValue(&subject),
				Id:         tfUtils.BuildInternalTerraformId(resourceID, role, subject),
			},
		},
		{
			name: "nil response input",
			resp: nil,
			inputModel: &Model{
				Role:    types.StringValue(role),
				Subject: types.StringValue(subject),
			},
			expectError: true,
		},
		{
			name: "nil members input",
			resp: &authorization.ListMembersResponse{
				ResourceId: &resourceID,
				Members:    nil,
			},
			inputModel: &Model{
				Role:    types.StringValue(role),
				Subject: types.StringValue(subject),
			},
			expectError: true,
		},
		{
			name: "nil resource_id input",
			resp: &authorization.ListMembersResponse{
				ResourceId: nil,
				Members: &[]authorization.Member{
					{
						Role:    &role,
						Subject: &subject,
					},
				},
			},
			inputModel: &Model{
				Role:    types.StringValue(role),
				Subject: types.StringValue(subject),
			},
			expectError: true,
		},
		{
			name: "nil model input",
			resp: &authorization.ListMembersResponse{
				ResourceId: &resourceID,
				Members: &[]authorization.Member{
					{
						Role:    &role,
						Subject: &subject,
					},
				},
			},
			inputModel:  nil,
			expectError: true,
		},
		{
			name: "no matching role/subject pair",
			resp: &authorization.ListMembersResponse{
				ResourceId: &resourceID,
				Members: &[]authorization.Member{
					{
						Role:    utils.Ptr("reader"),
						Subject: utils.Ptr("foo.bar@stackit.cloud"),
					},
				},
			},
			inputModel: &Model{
				Role:    types.StringValue(role),
				Subject: types.StringValue(subject),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapListMembersResponse(tt.resp, tt.inputModel)

			if tt.expectError && err == nil {
				t.Fatalf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			if !tt.expectError {
				if diff := cmp.Diff(tt.expected, tt.inputModel); diff != "" {
					t.Errorf("Mapped model mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCheckDuplicate(t *testing.T) {
	role := "editor"
	subject := "foo.bar@stackit.cloud"
	resourceID := "project"

	tests := []struct {
		name        string
		resp        *authorization.ListMembersResponse
		inputModel  Model
		expectError bool
		expectedErr error
	}{
		{
			name: "no members => no duplicate",
			resp: &authorization.ListMembersResponse{
				ResourceId: &resourceID,
				Members:    &[]authorization.Member{},
			},
			inputModel: Model{
				ResourceId: types.StringValue(resourceID),
				Role:       types.StringValue(role),
				Subject:    types.StringValue(subject),
			},
			expectError: false,
		},
		{
			name: "members but no matching role/subject => no duplicate",
			resp: &authorization.ListMembersResponse{
				ResourceId: &resourceID,
				Members: &[]authorization.Member{
					{
						Role:    utils.Ptr("reader"),
						Subject: utils.Ptr("foo.bar@stackit.cloud"),
					},
					{
						Role:    utils.Ptr("editor"),
						Subject: utils.Ptr("someoneelse@stackit.cloud"),
					},
				},
			},
			inputModel: Model{
				ResourceId: types.StringValue(resourceID),
				Role:       types.StringValue(role),
				Subject:    types.StringValue(subject),
			},
			expectError: false,
		},
		{
			name: "matching role/subject exists => duplicate error",
			resp: &authorization.ListMembersResponse{
				ResourceId: &resourceID,
				Members: &[]authorization.Member{
					{
						Role:    &role,
						Subject: &subject,
					},
				},
			},
			inputModel: Model{
				ResourceId: types.StringValue(resourceID),
				Role:       types.StringValue(role),
				Subject:    types.StringValue(subject),
			},
			expectError: true,
			expectedErr: errRoleAssignmentDuplicateFound,
		},
		{
			name: "nil response input => propagated error",
			resp: nil,
			inputModel: Model{
				ResourceId: types.StringValue(resourceID),
				Role:       types.StringValue(role),
				Subject:    types.StringValue(subject),
			},
			expectError: true,
			// we don't compare by value here, just expect some error
		},
		{
			name: "nil members input => propagated error",
			resp: &authorization.ListMembersResponse{
				ResourceId: &resourceID,
				Members:    nil,
			},
			inputModel: Model{
				ResourceId: types.StringValue(resourceID),
				Role:       types.StringValue(role),
				Subject:    types.StringValue(subject),
			},
			expectError: true,
		},
		{
			name: "nil resource_id input => propagated error",
			resp: &authorization.ListMembersResponse{
				ResourceId: nil,
				Members: &[]authorization.Member{
					{
						Role:    &role,
						Subject: &subject,
					},
				},
			},
			inputModel: Model{
				ResourceId: types.StringValue(resourceID),
				Role:       types.StringValue(role),
				Subject:    types.StringValue(subject),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkDuplicate(tt.inputModel, tt.resp)

			if tt.expectError && err == nil {
				t.Fatalf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			if tt.expectError && tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Fatalf("Expected error %v, but got: %v", tt.expectedErr, err)
			}
		})
	}
}
