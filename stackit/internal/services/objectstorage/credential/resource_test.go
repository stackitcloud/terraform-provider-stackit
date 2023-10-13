package objectstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

type objectStorageClientMocked struct {
	returnError bool
}

func (c *objectStorageClientMocked) CreateProjectExecute(_ context.Context, projectId string) (*objectstorage.GetProjectResponse, error) {
	if c.returnError {
		return nil, fmt.Errorf("create project failed")
	}

	return &objectstorage.GetProjectResponse{
		Project: utils.Ptr(projectId),
	}, nil
}

func TestMapFields(t *testing.T) {
	now := time.Now()

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
				ExpirationTimestamp: types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&objectstorage.CreateAccessKeyResponse{
				AccessKey:       utils.Ptr("key"),
				DisplayName:     utils.Ptr("name"),
				Expires:         utils.Ptr(now.Format(time.RFC3339)),
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
				ExpirationTimestamp: types.StringValue(now.Format(time.RFC3339)),
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
				ExpirationTimestamp: types.StringNull(),
			},
			true,
		},
		{
			"expiration_timestamp_with_fractional_seconds",
			&objectstorage.CreateAccessKeyResponse{
				Expires: utils.Ptr(now.Format(time.RFC3339Nano)),
			},
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				AccessKey:           types.StringNull(),
				ExpirationTimestamp: types.StringValue(now.Format(time.RFC3339)),
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
		expected    Model
		enableFails bool
		isValid     bool
	}{
		{
			"default_values",
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				AccessKey:           types.StringNull(),
				SecretAccessKey:     types.StringNull(),
				ExpirationTimestamp: types.StringNull(),
			},
			false,
			true,
		},
		{
			"error_response",
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				AccessKey:           types.StringNull(),
				SecretAccessKey:     types.StringNull(),
				ExpirationTimestamp: types.StringNull(),
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &objectStorageClientMocked{
				returnError: tt.enableFails,
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

func TestReadCredentials(t *testing.T) {
	timeNow := time.Now()

	tests := []struct {
		description         string
		mockedResp          *objectstorage.GetAccessKeysResponse
		expected            Model
		getCredentialsFails bool
		isValid             bool
	}{
		{
			"default_values",
			&objectstorage.GetAccessKeysResponse{
				AccessKeys: &[]objectstorage.AccessKey{
					{
						KeyId: utils.Ptr("foo-cid"),
					},
					{
						KeyId: utils.Ptr("bar-cid"),
					},
					{
						KeyId: utils.Ptr("cid"),
					},
				},
			},
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				AccessKey:           types.StringNull(),
				SecretAccessKey:     types.StringNull(),
				ExpirationTimestamp: types.StringNull(),
			},
			false,
			true,
		},
		{
			"simple_values",
			&objectstorage.GetAccessKeysResponse{
				AccessKeys: &[]objectstorage.AccessKey{
					{
						KeyId:       utils.Ptr("foo-cid"),
						DisplayName: utils.Ptr("foo-name"),
						Expires:     utils.Ptr(timeNow.Add(time.Hour).Format(time.RFC3339)),
					},
					{
						KeyId:       utils.Ptr("bar-cid"),
						DisplayName: utils.Ptr("bar-name"),
						Expires:     utils.Ptr(timeNow.Add(time.Minute).Format(time.RFC3339)),
					},
					{
						KeyId:       utils.Ptr("cid"),
						DisplayName: utils.Ptr("name"),
						Expires:     utils.Ptr(timeNow.Format(time.RFC3339)),
					},
				},
			},
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringValue("name"),
				AccessKey:           types.StringNull(),
				SecretAccessKey:     types.StringNull(),
				ExpirationTimestamp: types.StringValue(timeNow.Format(time.RFC3339)),
			},
			false,
			true,
		},
		{
			"expiration_timestamp_with_fractional_seconds",
			&objectstorage.GetAccessKeysResponse{
				AccessKeys: &[]objectstorage.AccessKey{
					{
						KeyId:       utils.Ptr("foo-cid"),
						DisplayName: utils.Ptr("foo-name"),
						Expires:     utils.Ptr(timeNow.Add(time.Hour).Format(time.RFC3339Nano)),
					},
					{
						KeyId:       utils.Ptr("bar-cid"),
						DisplayName: utils.Ptr("bar-name"),
						Expires:     utils.Ptr(timeNow.Add(time.Minute).Format(time.RFC3339Nano)),
					},
					{
						KeyId:       utils.Ptr("cid"),
						DisplayName: utils.Ptr("name"),
						Expires:     utils.Ptr(timeNow.Format(time.RFC3339Nano)),
					},
				},
			},
			Model{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringValue("name"),
				AccessKey:           types.StringNull(),
				SecretAccessKey:     types.StringNull(),
				ExpirationTimestamp: types.StringValue(timeNow.Format(time.RFC3339)),
			},
			false,
			true,
		},
		{
			"empty_credentials",
			&objectstorage.GetAccessKeysResponse{
				AccessKeys: &[]objectstorage.AccessKey{},
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
			"non_matching_credential",
			&objectstorage.GetAccessKeysResponse{
				AccessKeys: &[]objectstorage.AccessKey{
					{
						KeyId:       utils.Ptr("foo-cid"),
						DisplayName: utils.Ptr("foo-name"),
						Expires:     utils.Ptr(timeNow.Add(time.Hour).Format(time.RFC3339)),
					},
					{
						KeyId:       utils.Ptr("bar-cid"),
						DisplayName: utils.Ptr("bar-name"),
						Expires:     utils.Ptr(timeNow.Add(time.Minute).Format(time.RFC3339)),
					},
				},
			},
			Model{},
			false,
			false,
		},
		{
			"error_response",
			&objectstorage.GetAccessKeysResponse{
				AccessKeys: &[]objectstorage.AccessKey{
					{
						KeyId:       utils.Ptr("cid"),
						DisplayName: utils.Ptr("name"),
						Expires:     utils.Ptr(timeNow.Format(time.RFC3339)),
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
			mockedRespBytes, err := json.Marshal(tt.mockedResp)
			if err != nil {
				t.Fatalf("Failed to marshal mocked response: %v", err)
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tt.getCredentialsFails {
					w.WriteHeader(http.StatusBadGateway)
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte("{\"message\": \"Something bad happened\""))
					if err != nil {
						t.Errorf("Failed to write bad response: %v", err)
					}
					return
				}

				_, err := w.Write(mockedRespBytes)
				if err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			})
			mockedServer := httptest.NewServer(handler)
			defer mockedServer.Close()
			client, err := objectstorage.NewAPIClient(
				config.WithEndpoint(mockedServer.URL),
				config.WithoutAuthentication(),
				config.WithRetryTimeout(time.Millisecond),
			)
			if err != nil {
				t.Fatalf("Failed to initialize client: %v", err)
			}

			model := &Model{
				ProjectId:          tt.expected.ProjectId,
				CredentialsGroupId: tt.expected.CredentialsGroupId,
				CredentialId:       tt.expected.CredentialId,
			}
			err = readCredentials(context.Background(), model, client)
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
