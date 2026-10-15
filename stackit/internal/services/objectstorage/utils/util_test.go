package utils

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"
)

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
			synctest.Test(t, func(t *testing.T) {
				client := &objectstorage.DefaultAPIServiceMock{
					EnableServiceExecuteMock: new(func(_ objectstorage.ApiEnableServiceRequest) (*objectstorage.ProjectStatus, error) {
						if tt.enableFails {
							return nil, fmt.Errorf("create project failed")
						}

						return &objectstorage.ProjectStatus{}, nil
					}),
				}

				err := EnableProject(context.Background(), "pid", "eu01", client)
				if !tt.isValid && err == nil {
					t.Fatalf("Should have failed")
				}
				if tt.isValid && err != nil {
					t.Fatalf("Should not have failed: %v", err)
				}
			})
		})
	}
}

// A 409 from a concurrent enable call must be retried instead of failing the apply.
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

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
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

				err := EnableProject(ctx, "pid", "eu01", client)
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

	if err := EnableProject(context.Background(), "pid", "eu01", client); err == nil {
		t.Fatal("Should have failed")
	}
	if attempts != 1 {
		t.Fatalf("Expected a single attempt, got %d", attempts)
	}
}
