package wait

import (
	"context"
	"fmt"
	"time"

	postgresflex "github.com/stackitcloud/terraform-provider-stackit/pkg/postgresflexalpha"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/wait"
)

const (
	InstanceStateEmpty       = ""
	InstanceStateProgressing = "Progressing"
	InstanceStateSuccess     = "Ready"
	InstanceStateFailed      = "Failure"
	InstanceStateDeleted     = "Deleted"
)

// Interface needed for tests
type APIClientInstanceInterface interface {
	GetInstanceRequestExecute(ctx context.Context, projectId, region, instanceId string) (*postgresflex.GetInstanceResponse, error)
	ListUsersRequestExecute(ctx context.Context, projectId, region, instanceId string) (*postgresflex.ListUserResponse, error)
}

// Interface needed for tests
type APIClientUserInterface interface {
	GetUserRequestExecute(ctx context.Context, projectId, region, instanceId string, userId int64) (*postgresflex.GetUserResponse, error)
}

// CreateInstanceWaitHandler will wait for instance creation
func CreateInstanceWaitHandler(ctx context.Context, a APIClientInstanceInterface, projectId, region, instanceId string) *wait.AsyncActionHandler[postgresflex.GetInstanceResponse] {
	instanceCreated := false
	var instanceGetResponse *postgresflex.GetInstanceResponse

	handler := wait.New(func() (waitFinished bool, response *postgresflex.GetInstanceResponse, err error) {
		if !instanceCreated {
			s, err := a.GetInstanceRequestExecute(ctx, projectId, region, instanceId)
			if err != nil {
				return false, nil, err
			}
			if s == nil || s.Id == nil || *s.Id != instanceId || s.Status == nil {
				return false, nil, nil
			}
			switch *s.Status {
			default:
				return true, s, fmt.Errorf("instance with id %s has unexpected status %s", instanceId, *s.Status)
			case InstanceStateEmpty:
				return false, nil, nil
			case InstanceStateProgressing:
				return false, nil, nil
			case InstanceStateSuccess:
				instanceCreated = true
				instanceGetResponse = s
			case InstanceStateFailed:
				return true, s, fmt.Errorf("create failed for instance with id %s", instanceId)
			}
		}

		// User operations aren't available right after an instance is deemed successful
		// To check if they are, perform a users request
		_, err = a.ListUsersRequestExecute(ctx, projectId, region, instanceId)
		if err == nil {
			return true, instanceGetResponse, nil
		}
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if !ok {
			return false, nil, err
		}
		if oapiErr.StatusCode < 500 {
			return true, instanceGetResponse, fmt.Errorf("users request after instance creation returned %d status code", oapiErr.StatusCode)
		}
		return false, nil, nil
	})
	// Sleep before wait is set because sometimes API returns 404 right after creation request
	handler.SetTimeout(45 * time.Minute).SetSleepBeforeWait(15 * time.Second)
	return handler
}

// PartialUpdateInstanceWaitHandler will wait for instance update
func PartialUpdateInstanceWaitHandler(ctx context.Context, a APIClientInstanceInterface, projectId, region, instanceId string) *wait.AsyncActionHandler[postgresflex.GetInstanceResponse] {
	handler := wait.New(func() (waitFinished bool, response *postgresflex.GetInstanceResponse, err error) {
		s, err := a.GetInstanceRequestExecute(ctx, projectId, region, instanceId)
		if err != nil {
			return false, nil, err
		}
		if s == nil || s.Id == nil || *s.Id != instanceId || s.Status == nil {
			return false, nil, nil
		}
		switch *s.Status {
		default:
			return true, s, fmt.Errorf("instance with id %s has unexpected status %s", instanceId, *s.Status)
		case InstanceStateEmpty:
			return false, nil, nil
		case InstanceStateProgressing:
			return false, nil, nil
		case InstanceStateSuccess:
			return true, s, nil
		case InstanceStateFailed:
			return true, s, fmt.Errorf("update failed for instance with id %s", instanceId)
		}
	})
	handler.SetTimeout(45 * time.Minute)
	return handler
}

// DeleteInstanceWaitHandler will wait for instance deletion
func DeleteInstanceWaitHandler(ctx context.Context, a APIClientInstanceInterface, projectId, region, instanceId string) *wait.AsyncActionHandler[struct{}] {
	handler := wait.New(func() (waitFinished bool, response *struct{}, err error) {
		s, err := a.GetInstanceRequestExecute(ctx, projectId, region, instanceId)
		if err != nil {
			return false, nil, err
		}
		if s == nil || s.Id == nil || *s.Id != instanceId || s.Status == nil {
			return false, nil, nil
		}
		switch *s.Status {
		default:
			return true, nil, fmt.Errorf("instance with id %s has unexpected status %s", instanceId, *s.Status)
		case InstanceStateSuccess:
			return false, nil, nil
		case InstanceStateDeleted:
			return true, nil, nil
		}
	})
	handler.SetTimeout(5 * time.Minute)
	return handler
}

// ForceDeleteInstanceWaitHandler will wait for instance deletion
func ForceDeleteInstanceWaitHandler(ctx context.Context, a APIClientInstanceInterface, projectId, region, instanceId string) *wait.AsyncActionHandler[struct{}] {
	handler := wait.New(func() (waitFinished bool, response *struct{}, err error) {
		_, err = a.GetInstanceRequestExecute(ctx, projectId, region, instanceId)
		if err == nil {
			return false, nil, nil
		}
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if !ok {
			return false, nil, err
		}
		if oapiErr.StatusCode != 404 {
			return false, nil, err
		}
		return true, nil, nil
	})
	handler.SetTimeout(15 * time.Minute)
	return handler
}

// DeleteUserWaitHandler will wait for delete
func DeleteUserWaitHandler(ctx context.Context, a APIClientUserInterface, projectId, region, instanceId string, userId int64) *wait.AsyncActionHandler[struct{}] {
	handler := wait.New(func() (waitFinished bool, response *struct{}, err error) {
		_, err = a.GetUserRequestExecute(ctx, projectId, region, instanceId, userId)
		if err == nil {
			return false, nil, nil
		}
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if !ok {
			return false, nil, err
		}
		if oapiErr.StatusCode != 404 {
			return false, nil, err
		}
		return true, nil, nil
	})
	handler.SetTimeout(1 * time.Minute)
	return handler
}
