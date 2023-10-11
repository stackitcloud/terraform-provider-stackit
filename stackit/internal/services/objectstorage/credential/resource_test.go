package objectstorage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
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
		return nil, fmt.Errorf("get credentials groups failed")
	}

	return c.createProjectExecuteResp, nil
}

func TestMapFields(t *testing.T) {
	timeValue := time.Now()

	tests := []struct {
		description string
		input       *objectstorage.CreateAccessKeyResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.CreateAccessKeyResponse{},
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				AccessKey:           types.StringNull(),
				SecretAccessKey:     types.StringNull(),
				ExpirationTimestamp: timetypes.NewRFC3339Null(),
			},
			true,
		},
		{
			"simple_values",
			&objectstorage.CreateAccessKeyResponse{
				AccessKey:       utils.Ptr("key"),
				DisplayName:     utils.Ptr("name"),
				Expires:         utils.Ptr(timeValue.Format(time.RFC3339)),
				SecretAccessKey: utils.Ptr("secret-key"),
			},
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringValue("name"),
				AccessKey:           types.StringValue("key"),
				SecretAccessKey:     types.StringValue("secret-key"),
				ExpirationTimestamp: timetypes.NewRFC3339TimeValue(timeValue),
			},
			true,
		},
		{
			"empty_strings",
			&objectstorage.CreateAccessKeyResponse{
				AccessKey:       utils.Ptr(""),
				DisplayName:     utils.Ptr(""),
				SecretAccessKey: utils.Ptr(""),
			},
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringValue(""),
				AccessKey:           types.StringValue(""),
				SecretAccessKey:     types.StringValue(""),
				ExpirationTimestamp: timetypes.NewRFC3339Null(),
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
			"bad_time",
			&objectstorage.CreateAccessKeyResponse{
				Expires: utils.Ptr("foo-bar"),
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId:          tt.expected.ProjectId,
				CredentialsGroupId: tt.expected.CredentialsGroupId,
				CredentialId:       tt.expected.CredentialId,
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
		expected    Model
		enableFails bool
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.GetProjectResponse{},
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				AccessKey:           types.StringNull(),
				SecretAccessKey:     types.StringNull(),
				ExpirationTimestamp: timetypes.NewRFC3339Null(),
			},
			false,
			true,
		},
		{
			"nil_response",
			nil,
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				AccessKey:           types.StringNull(),
				SecretAccessKey:     types.StringNull(),
				ExpirationTimestamp: timetypes.NewRFC3339Null(),
			},
			false,
			true,
		},
		{
			"error_response",
			&objectstorage.GetProjectResponse{},
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				AccessKey:           types.StringNull(),
				SecretAccessKey:     types.StringNull(),
				ExpirationTimestamp: timetypes.NewRFC3339Null(),
			},
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
			model := &Model{
				ProjectId:          tt.expected.ProjectId,
				CredentialsGroupId: tt.expected.CredentialsGroupId,
				CredentialId:       tt.expected.CredentialId,
			}
			err := enableProject(context.Background(), model, client)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
		})
	}
}
