package core

import (
	iaasV2 "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	resourcemanager "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v0api"
	serviceenablementV2 "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
)

var _ ClientFactory = &MockClientFactory{}

type MockClientFactory struct {
	defaultClientFactory DefaultClientFactory

	ServiceEnablementV2ClientMock serviceenablementV2.DefaultAPI
	IaaSV2ClientMock              iaasV2.DefaultAPI
	ModelExperimentsV1ClientMock  modelexperiments.DefaultAPI
	ResourceManagerClientMock     resourcemanager.DefaultAPI
}

func (m *MockClientFactory) newServiceEnablementV2Client() (serviceenablementV2.DefaultAPI, error) {
	if m.ServiceEnablementV2ClientMock != nil {
		return m.ServiceEnablementV2ClientMock, nil
	}

	return m.defaultClientFactory.newServiceEnablementV2Client()
}

func (m *MockClientFactory) newIaaSV2Client() (iaasV2.DefaultAPI, error) {
	if m.IaaSV2ClientMock != nil {
		return m.IaaSV2ClientMock, nil
	}

	return m.defaultClientFactory.newIaaSV2Client()
}

func (m *MockClientFactory) newResourceManagerClient() (resourcemanager.DefaultAPI, error) {
	if m.ResourceManagerClientMock != nil {
		return m.ResourceManagerClientMock, nil
	}

	return m.defaultClientFactory.newResourceManagerClient()
}

func (m *MockClientFactory) newModelExperimentsV1Client() (modelexperiments.DefaultAPI, error) {
	if m.ModelExperimentsV1ClientMock != nil {
		return m.ModelExperimentsV1ClientMock, nil
	}

	return m.defaultClientFactory.newModelExperimentsV1Client()
}
