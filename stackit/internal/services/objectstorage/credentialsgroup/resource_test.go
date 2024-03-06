package objectstorage

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

type objectStorageClientMocked struct {
	returnError               bool
	listCredentialsGroupsResp *objectstorage.ListCredentialsGroupsResponse
}

func (c *objectStorageClientMocked) EnableServiceExecute(_ context.Context, projectId string) (*objectstorage.ProjectStatus, error) {
	if c.returnError {
		return nil, fmt.Errorf("create project failed")
	}

	return &objectstorage.ProjectStatus{
		Project: utils.Ptr(projectId),
	}, nil
}

func (c *objectStorageClientMocked) ListCredentialsGroupsExecute(_ context.Context, _ string) (*objectstorage.ListCredentialsGroupsResponse, error) {
	if c.returnError {
		return nil, fmt.Errorf("get credentials groups failed")
	}

	return c.listCredentialsGroupsResp, nil
}

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

func TestEnableProject(t *testing.T) {
	tests := []struct {
		description string
		enableFails bool
		isValid     bool
	}{
		{
			"default_values",
			false,
			true,
		},
		{
			"error_response",
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &objectStorageClientMocked{
				returnError: tt.enableFails,
			}
			err := enableProject(context.Background(), &Model{}, client)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
		})
	}
}

func TestReadCredentialsGroups(t *testing.T) {
	tests := []struct {
		description               string
		mockedResp                *objectstorage.ListCredentialsGroupsResponse
		expected                  Model
		getCredentialsGroupsFails bool
		isValid                   bool
	}{
		{
			"default_values",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: &[]objectstorage.CredentialsGroup{
					{
						CredentialsGroupId: utils.Ptr("cid"),
					},
					{
						CredentialsGroupId: utils.Ptr("foo-id"),
					},
				},
			},
			Model{
				Id:                 types.StringValue("pid,cid"),
				Name:               types.StringNull(),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringNull(),
			},
			false,
			true,
		},
		{
			"simple_values",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: &[]objectstorage.CredentialsGroup{
					{
						CredentialsGroupId: utils.Ptr("cid"),
						DisplayName:        utils.Ptr("name"),
						Urn:                utils.Ptr("urn"),
					},
					{
						CredentialsGroupId: utils.Ptr("foo-cid"),
						DisplayName:        utils.Ptr("foo-name"),
						Urn:                utils.Ptr("foo-urn"),
					},
				},
			},
			Model{
				Id:                 types.StringValue("pid,cid"),
				Name:               types.StringValue("name"),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringValue("urn"),
			},
			false,
			true,
		},
		{
			"empty_credentials_groups",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: &[]objectstorage.CredentialsGroup{},
			},
			Model{},
			false,
			false,
		},
		{
			"nil_credentials_groups",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: nil,
			},
			Model{},
			false,
			false,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
			false,
		},
		{
			"non_matching_credentials_group",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: &[]objectstorage.CredentialsGroup{
					{
						CredentialsGroupId: utils.Ptr("foo-other"),
						DisplayName:        utils.Ptr("foo-name"),
						Urn:                utils.Ptr("foo-urn"),
					},
				},
			},
			Model{},
			false,
			false,
		},
		{
			"error_response",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: &[]objectstorage.CredentialsGroup{
					{
						CredentialsGroupId: utils.Ptr("other_id"),
						DisplayName:        utils.Ptr("name"),
						Urn:                utils.Ptr("urn"),
					},
				},
			},
			Model{},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &objectStorageClientMocked{
				returnError:               tt.getCredentialsGroupsFails,
				listCredentialsGroupsResp: tt.mockedResp,
			}
			model := &Model{
				ProjectId:          tt.expected.ProjectId,
				CredentialsGroupId: tt.expected.CredentialsGroupId,
			}
			err := readCredentialsGroups(context.Background(), model, client)
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
