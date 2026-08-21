package core

import (
	"fmt"
	"net/http"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
	iaasV2 "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	resourcemanager "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v0api"
	serviceenablementV2 "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
)

type ClientFactory interface {
	// methods are having the API versions in them here so we can still mix & match API versions just as we need

	// Service enablement
	newServiceEnablementV2Client() (serviceenablementV2.DefaultAPI, error)

	// IaaS
	newIaaSV2Client() (iaasV2.DefaultAPI, error)

	newResourceManagerClient() (resourcemanager.DefaultAPI, error)

	newModelExperimentsV1Client() (modelexperiments.DefaultAPI, error)
}

func initClientCollection(clientFactory ClientFactory) (*clientCollection, error) {
	iaasV2Client, err := clientFactory.newIaaSV2Client()
	if err != nil {
		return nil, err
	}

	resourcemanagerClient, err := clientFactory.newResourceManagerClient()
	if err != nil {
		return nil, err
	}

	return &clientCollection{
		IaaSv2Client:          iaasV2Client,
		ResourceManagerClient: resourcemanagerClient,
	}, nil
}

type DefaultClientFactory struct {
	RoundTripper    http.RoundTripper
	UserAgent       string
	CustomEndpoints CustomEndpointConfig
}

func (f *DefaultClientFactory) defaultConfigOptions(customEndpoint string) []config.ConfigurationOption {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(f.RoundTripper),
		config.WithUserAgent(f.UserAgent),
	}

	if customEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(customEndpoint))
	}

	return apiClientConfigOptions
}

func (f *DefaultClientFactory) newServiceEnablementV2Client() (serviceenablementV2.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ServiceEnablementCustomEndpoint)

	apiClient, err := serviceenablementV2.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newIaaSV2Client() (iaasV2.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.IaaSCustomEndpoint)

	apiClient, err := iaasV2.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newResourceManagerClient() (resourcemanager.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ResourceManagerCustomEndpoint)

	apiClient, err := resourcemanager.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}

func (f *DefaultClientFactory) newModelExperimentsV1Client() (modelexperiments.DefaultAPI, error) {
	apiClientConfigOptions := f.defaultConfigOptions(f.CustomEndpoints.ModelExperimentsCustomEndpoint)

	apiClient, err := modelexperiments.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		return nil, fmt.Errorf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err)
	}

	return apiClient.DefaultAPI, nil
}
