package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// ConfigureClient configures an API-Client to communicate with the authorization API
func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *authorization.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.AuthorizationCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.AuthorizationCustomEndpoint))
	}
	apiClient, err := authorization.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

// TypeConverter converts objects with equal JSON tags
func TypeConverter[R any](data any) (*R, error) {
	var result R
	b, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

// Global map to hold locks for specific assignment IDs
// This ensures that creating the same assignment in parallel waits for the first one to finish
var (
	assignmentLocksMu sync.Mutex
	assignmentLocks   = make(map[string]*sync.Mutex)
)

// LockAssignment acquires a lock for a specific assignment identifier.
// It returns an unlock function that must be deferred.
func LockAssignment(id string) func() {
	assignmentLocksMu.Lock()
	mu, ok := assignmentLocks[id]
	if !ok {
		mu = &sync.Mutex{}
		assignmentLocks[id] = mu
	}
	assignmentLocksMu.Unlock()

	mu.Lock()

	// Return the cleanup function
	return func() {
		mu.Unlock()
	}
}
