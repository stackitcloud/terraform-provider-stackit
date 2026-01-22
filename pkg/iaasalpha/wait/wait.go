// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: Apache-2.0

package wait

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/wait"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

const (
	CreateSuccess = "CREATED"
)

// Interfaces needed for tests
type APIClientInterface interface {
	GetNetworkExecute(ctx context.Context, projectId, region, networkId string) (*iaasalpha.Network, error)
}

// CreateNetworkWaitHandler will wait for network creation using network id
func CreateNetworkWaitHandler(ctx context.Context, a APIClientInterface, projectId, region, networkId string) *wait.AsyncActionHandler[iaasalpha.Network] {
	handler := wait.New(func() (waitFinished bool, response *iaasalpha.Network, err error) {
		network, err := a.GetNetworkExecute(ctx, projectId, region, networkId)
		if err != nil {
			return false, network, err
		}
		if network.Id == nil || network.Status == nil {
			return false, network, fmt.Errorf("create failed for network with id %s, the response is not valid: the id or the state are missing", networkId)
		}
		// The state returns to "CREATED" after a successful creation is completed
		if *network.Id == networkId && *network.Status == CreateSuccess {
			return true, network, nil
		}
		return false, network, nil
	})
	handler.SetSleepBeforeWait(2 * time.Second)
	handler.SetTimeout(15 * time.Minute)
	return handler
}

// UpdateNetworkWaitHandler will wait for network update
func UpdateNetworkWaitHandler(ctx context.Context, a APIClientInterface, projectId, region, networkId string) *wait.AsyncActionHandler[iaasalpha.Network] {
	handler := wait.New(func() (waitFinished bool, response *iaasalpha.Network, err error) {
		network, err := a.GetNetworkExecute(ctx, projectId, region, networkId)
		if err != nil {
			return false, network, err
		}
		if network.Id == nil || network.Status == nil {
			return false, network, fmt.Errorf("update failed for network with id %s, the response is not valid: the id or the state are missing", networkId)
		}
		// The state returns to "CREATED" after a successful update is completed
		if *network.Id == networkId && *network.Status == CreateSuccess {
			return true, network, nil
		}
		return false, network, nil
	})
	handler.SetSleepBeforeWait(2 * time.Second)
	handler.SetTimeout(15 * time.Minute)
	return handler
}

// DeleteNetworkWaitHandler will wait for network deletion
func DeleteNetworkWaitHandler(ctx context.Context, a APIClientInterface, projectId, region, networkId string) *wait.AsyncActionHandler[iaasalpha.Network] {
	handler := wait.New(func() (waitFinished bool, response *iaasalpha.Network, err error) {
		network, err := a.GetNetworkExecute(ctx, projectId, region, networkId)
		if err == nil {
			return false, nil, nil
		}
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if !ok {
			return false, network, fmt.Errorf("could not convert error to oapierror.GenericOpenAPIError: %w", err)
		}
		if oapiErr.StatusCode != http.StatusNotFound {
			return false, network, err
		}
		return true, nil, nil
	})
	handler.SetTimeout(15 * time.Minute)
	return handler
}
