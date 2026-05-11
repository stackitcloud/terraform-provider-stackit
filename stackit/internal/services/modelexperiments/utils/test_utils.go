package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	mock_instance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance/mock"
	mock_serviceenablement "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/utils/mock"
	"go.uber.org/mock/gomock"
)

type TestContext struct {
	T                           *testing.T
	MockCtrl                    *gomock.Controller
	MockInstanceCLient          *mock_instance.MockDefaultAPI
	MockServiceEnablementClient *mock_serviceenablement.MockDefaultAPI
	Resource                    *resource.Resource
	Ctx                         context.Context
	CancelFunc                  context.CancelFunc
}
