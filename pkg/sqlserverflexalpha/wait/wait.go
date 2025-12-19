package wait

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/wait"
	sqlserverflex "github.com/stackitcloud/terraform-provider-stackit/pkg/sqlserverflexalpha"
)

const (
	InstanceStateEmpty      = ""
	InstanceStateProcessing = "Progressing"
	InstanceStateUnknown    = "Unknown"
	InstanceStateSuccess    = "Ready"
	InstanceStateFailed     = "Failed"
)

// Interface needed for tests
type APIClientInstanceInterface interface {
	GetInstanceRequestExecute(ctx context.Context, projectId, region, instanceId string) (*sqlserverflex.GetInstanceResponse, error)
}

// CreateInstanceWaitHandler will wait for instance creation
func CreateInstanceWaitHandler(ctx context.Context, a APIClientInstanceInterface, projectId, instanceId, region string) *wait.AsyncActionHandler[sqlserverflex.GetInstanceResponse] {
	handler := wait.New(func() (waitFinished bool, response *sqlserverflex.GetInstanceResponse, err error) {
		s, err := a.GetInstanceRequestExecute(ctx, projectId, region, instanceId)
		if err != nil {
			return false, nil, err
		}
		if s == nil || s.Id == nil || *s.Id != instanceId || s.Status == nil {
			return false, nil, nil
		}
		switch strings.ToLower(string(*s.Status)) {
		case strings.ToLower(InstanceStateSuccess):
			return true, s, nil
		case strings.ToLower(InstanceStateUnknown), strings.ToLower(InstanceStateFailed):
			return true, s, fmt.Errorf("create failed for instance with id %s", instanceId)
		default:
			return false, s, nil
		}
	})
	handler.SetTimeout(45 * time.Minute)
	handler.SetSleepBeforeWait(5 * time.Second)
	return handler
}

// UpdateInstanceWaitHandler will wait for instance update
func UpdateInstanceWaitHandler(ctx context.Context, a APIClientInstanceInterface, projectId, instanceId, region string) *wait.AsyncActionHandler[sqlserverflex.GetInstanceResponse] {
	handler := wait.New(func() (waitFinished bool, response *sqlserverflex.GetInstanceResponse, err error) {
		s, err := a.GetInstanceRequestExecute(ctx, projectId, region, instanceId)
		if err != nil {
			return false, nil, err
		}
		if s == nil || s.Id == nil || *s.Id != instanceId || s.Status == nil {
			return false, nil, nil
		}
		switch strings.ToLower(string(*s.Status)) {
		case strings.ToLower(InstanceStateSuccess):
			return true, s, nil
		case strings.ToLower(InstanceStateUnknown), strings.ToLower(InstanceStateFailed):
			return true, s, fmt.Errorf("update failed for instance with id %s", instanceId)
		default:
			return false, s, nil
		}
	})
	handler.SetSleepBeforeWait(2 * time.Second)
	handler.SetTimeout(45 * time.Minute)
	return handler
}

// PartialUpdateInstanceWaitHandler will wait for instance update
func PartialUpdateInstanceWaitHandler(ctx context.Context, a APIClientInstanceInterface, projectId, instanceId, region string) *wait.AsyncActionHandler[sqlserverflex.GetInstanceResponse] {
	return UpdateInstanceWaitHandler(ctx, a, projectId, instanceId, region)
}

// DeleteInstanceWaitHandler will wait for instance deletion
func DeleteInstanceWaitHandler(ctx context.Context, a APIClientInstanceInterface, projectId, instanceId, region string) *wait.AsyncActionHandler[struct{}] {
	handler := wait.New(func() (waitFinished bool, response *struct{}, err error) {
		_, err = a.GetInstanceRequestExecute(ctx, projectId, region, instanceId)
		if err == nil {
			return false, nil, nil
		}
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if !ok {
			return false, nil, fmt.Errorf("could not convert error to oapierror.GenericOpenAPIError")
		}
		if oapiErr.StatusCode != http.StatusNotFound {
			return false, nil, err
		}
		return true, nil, nil
	})
	handler.SetTimeout(15 * time.Minute)
	return handler
}
