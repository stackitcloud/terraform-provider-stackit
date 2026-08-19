package clientutils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	iaasV2 "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	serviceenablementV2 "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

var _ ClientFactory = &MockClientFactory{}

type MockClientFactory struct {
	defaultClientFactory DefaultClientFactory

	ServiceEnablementV2ClientMock serviceenablementV2.DefaultAPI
	IaaSV2ClientMock              iaasV2.DefaultAPI
}

func (m *MockClientFactory) NewServiceEnablementV2Client(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) serviceenablementV2.DefaultAPI {
	if m.ServiceEnablementV2ClientMock != nil {
		return m.ServiceEnablementV2ClientMock
	}

	return m.defaultClientFactory.NewServiceEnablementV2Client(ctx, providerData, diags)
}

func (m *MockClientFactory) NewIaaSV2Client(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) iaasV2.DefaultAPI {
	if m.IaaSV2ClientMock != nil {
		return m.IaaSV2ClientMock
	}

	return m.defaultClientFactory.NewIaaSV2Client(ctx, providerData, diags)
}
