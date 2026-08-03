package objectstorage

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"
)

type mockSettings struct {
	returnError               bool
	listCredentialsGroupsResp *objectstorage.ListCredentialsGroupsResponse
}

func newAPIMock(settings *mockSettings) objectstorage.DefaultAPI {
	return &objectstorage.DefaultAPIServiceMock{
		EnableServiceExecuteMock: new(func(_ objectstorage.ApiEnableServiceRequest) (*objectstorage.ProjectStatus, error) {
			if settings.returnError {
				return nil, fmt.Errorf("create project failed")
			}

			return &objectstorage.ProjectStatus{}, nil
		}),
		ListCredentialsGroupsExecuteMock: new(func(_ objectstorage.ApiListCredentialsGroupsRequest) (*objectstorage.ListCredentialsGroupsResponse, error) {
			if settings.returnError {
				return nil, fmt.Errorf("get credentials groups failed")
			}

			return settings.listCredentialsGroupsResp, nil
		}),
	}
}

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s,%s", "pid", testRegion, "cid")
	tests := []struct {
		description string
		input       *objectstorage.CreateCredentialsGroupResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.CreateCredentialsGroupResponse{
				CredentialsGroup: objectstorage.CredentialsGroup{},
			},
			Model{
				Id:                 types.StringValue(id),
				Name:               types.StringValue(""),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringValue(""),
				Region:             types.StringValue("eu01"),
			},
			true,
		},
		{
			"simple_values",
			&objectstorage.CreateCredentialsGroupResponse{
				CredentialsGroup: objectstorage.CredentialsGroup{
					DisplayName: "name",
					Urn:         "urn",
				},
			},
			Model{
				Id:                 types.StringValue(id),
				Name:               types.StringValue("name"),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringValue("urn"),
				Region:             types.StringValue("eu01"),
			},
			true,
		},
		{
			"empty_strings",
			&objectstorage.CreateCredentialsGroupResponse{
				CredentialsGroup: objectstorage.CredentialsGroup{
					DisplayName: "",
					Urn:         "",
				},
			},
			Model{
				Id:                 types.StringValue(id),
				Name:               types.StringValue(""),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringValue(""),
				Region:             types.StringValue("eu01"),
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
			err := mapFields(tt.input, model, "eu01")
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
	// enableProject retries, and the mock returns a plain error rather than an
	// *oapierror.GenericOpenAPIError - RetryRequest only filters by status code
	// when it can type-assert the error, so the failing case uses up every
	// attempt. Without shrinking the delay this test would sleep for seconds.
	oldDelay := enableProjectRetryDelay
	enableProjectRetryDelay = time.Millisecond
	defer func() { enableProjectRetryDelay = oldDelay }()

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
			client := newAPIMock(&mockSettings{
				returnError: tt.enableFails,
			})

			err := enableProject(context.Background(), &Model{}, "eu01", client)
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
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s,%s", "pid", testRegion, "cid")
	tests := []struct {
		description               string
		mockedResp                *objectstorage.ListCredentialsGroupsResponse
		expectedModel             Model
		expectedFound             bool
		getCredentialsGroupsFails bool
		isValid                   bool
	}{
		{
			"default_values",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: []objectstorage.CredentialsGroup{
					{
						CredentialsGroupId: "cid",
					},
					{
						CredentialsGroupId: "foo-id",
					},
				},
			},
			Model{
				Id:                 types.StringValue(id),
				Name:               types.StringValue(""),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringValue(""),
			},
			true,
			false,
			true,
		},
		{
			"simple_values",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: []objectstorage.CredentialsGroup{
					{
						CredentialsGroupId: "cid",
						DisplayName:        "name",
						Urn:                "urn",
					},
					{
						CredentialsGroupId: "foo-cid",
						DisplayName:        "foo-name",
						Urn:                "foo-urn",
					},
				},
			},
			Model{
				Id:                 types.StringValue(id),
				Name:               types.StringValue("name"),
				ProjectId:          types.StringValue("pid"),
				CredentialsGroupId: types.StringValue("cid"),
				URN:                types.StringValue("urn"),
			},
			true,
			false,
			true,
		},
		{
			"empty_credentials_groups",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: []objectstorage.CredentialsGroup{},
			},
			Model{},
			false,
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
			false,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
			false,
			false,
		},
		{
			"non_matching_credentials_group",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: []objectstorage.CredentialsGroup{
					{
						CredentialsGroupId: "foo-other",
						DisplayName:        "foo-name",
						Urn:                "foo-urn",
					},
				},
			},
			Model{},
			false,
			false,
			false,
		},
		{
			"error_response",
			&objectstorage.ListCredentialsGroupsResponse{
				CredentialsGroups: []objectstorage.CredentialsGroup{
					{
						CredentialsGroupId: "other_id",
						DisplayName:        "name",
						Urn:                "urn",
					},
				},
			},
			Model{},
			false,
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := newAPIMock(&mockSettings{
				returnError:               tt.getCredentialsGroupsFails,
				listCredentialsGroupsResp: tt.mockedResp,
			})

			model := &Model{
				ProjectId:          tt.expectedModel.ProjectId,
				CredentialsGroupId: tt.expectedModel.CredentialsGroupId,
			}
			found, err := readCredentialsGroups(context.Background(), model, "eu01", client)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(model, &tt.expectedModel)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}

				if found != tt.expectedFound {
					t.Fatalf("Found does not match")
				}
			}
		})
	}
}

// Two object storage resources created in the same apply enable the project concurrently.
// The API answers the losing call with 409 project.create_conflict; enableProject must retry
// instead of failing the apply.
func TestEnableProjectRetriesOnConflict(t *testing.T) {
	tests := []struct {
		description  string
		conflicts    int
		isValid      bool
		wantAttempts int
	}{
		{"succeeds immediately", 0, true, 1},
		{"one conflict, then success", 1, true, 2},
		{"conflicts until the attempts are used up", enableProjectAttempts, false, enableProjectAttempts},
	}

	old := enableProjectRetryDelay
	enableProjectRetryDelay = time.Millisecond
	defer func() { enableProjectRetryDelay = old }()

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			attempts := 0
			client := &objectstorage.DefaultAPIServiceMock{
				EnableServiceExecuteMock: new(func(_ objectstorage.ApiEnableServiceRequest) (*objectstorage.ProjectStatus, error) {
					attempts++
					if attempts <= tt.conflicts {
						return nil, &oapierror.GenericOpenAPIError{StatusCode: http.StatusConflict}
					}
					return &objectstorage.ProjectStatus{}, nil
				}),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := enableProject(ctx, &Model{}, "eu01", client)
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if !tt.isValid && err == nil {
				t.Fatal("Should have failed")
			}
			if attempts != tt.wantAttempts {
				t.Fatalf("Expected %d attempts, got %d", tt.wantAttempts, attempts)
			}
		})
	}
}

// A non-conflict error must not be retried.
func TestEnableProjectDoesNotRetryOtherErrors(t *testing.T) {
	attempts := 0
	client := &objectstorage.DefaultAPIServiceMock{
		EnableServiceExecuteMock: new(func(_ objectstorage.ApiEnableServiceRequest) (*objectstorage.ProjectStatus, error) {
			attempts++
			return nil, &oapierror.GenericOpenAPIError{StatusCode: http.StatusForbidden}
		}),
	}

	if err := enableProject(context.Background(), &Model{}, "eu01", client); err == nil {
		t.Fatal("Should have failed")
	}
	if attempts != 1 {
		t.Fatalf("Expected a single attempt, got %d", attempts)
	}
}
