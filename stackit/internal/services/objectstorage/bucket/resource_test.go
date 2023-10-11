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
	returnError              bool
	createProjectExecuteResp *objectstorage.GetProjectResponse
}

func (c *objectStorageClientMocked) CreateProjectExecute(_ context.Context, _ string) (*objectstorage.GetProjectResponse, error) {
	if c.returnError {
		return nil, fmt.Errorf("create project failed")
	}

	return c.createProjectExecuteResp, nil
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *objectstorage.GetBucketResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.GetBucketResponse{
				Bucket: &objectstorage.Bucket{},
			},
			Model{
				Id:                    types.StringValue("pid,bname"),
				BucketName:            types.StringValue("bname"),
				ProjectId:             types.StringValue("pid"),
				URLPathStyle:          types.StringNull(),
				URLVirtualHostedStyle: types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&objectstorage.GetBucketResponse{
				Bucket: &objectstorage.Bucket{
					UrlPathStyle:          utils.Ptr("url/path/style"),
					UrlVirtualHostedStyle: utils.Ptr("url/virtual/hosted/style"),
				},
			},
			Model{
				Id:                    types.StringValue("pid,bname"),
				BucketName:            types.StringValue("bname"),
				ProjectId:             types.StringValue("pid"),
				URLPathStyle:          types.StringValue("url/path/style"),
				URLVirtualHostedStyle: types.StringValue("url/virtual/hosted/style"),
			},
			true,
		},
		{
			"empty_strings",
			&objectstorage.GetBucketResponse{
				Bucket: &objectstorage.Bucket{
					UrlPathStyle:          utils.Ptr(""),
					UrlVirtualHostedStyle: utils.Ptr(""),
				},
			},
			Model{
				Id:                    types.StringValue("pid,bname"),
				BucketName:            types.StringValue("bname"),
				ProjectId:             types.StringValue("pid"),
				URLPathStyle:          types.StringValue(""),
				URLVirtualHostedStyle: types.StringValue(""),
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
			&objectstorage.GetBucketResponse{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId:  tt.expected.ProjectId,
				BucketName: tt.expected.BucketName,
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
		mockedResp  *objectstorage.GetProjectResponse
		enableFails bool
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.GetProjectResponse{},
			false,
			true,
		},
		{
			"nil_response",
			nil,
			false,
			true,
		},
		{
			"error_response",
			&objectstorage.GetProjectResponse{},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &objectStorageClientMocked{
				returnError:              tt.enableFails,
				createProjectExecuteResp: tt.mockedResp,
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
