package tfctxinit

import (
	"context"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/testtypes"
)

type resource struct{}

func (r *resource) Create(ctx context.Context, req testtypes.CreateRequest, resp *testtypes.CreateResponse) {
	// Creating an API client before calling InitProviderContext is fine.
	iaas.NewAPIClient()

	var service *iaas.DefaultAPIService
	service.AddNetworkToServerExecute(iaas.ApiAddNetworkToServerRequest{}) // want "tfctxinit: call to github.com/stackitcloud/stackit-sdk-go must happen AFTER github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core.InitProviderContext is called in Create"
	core.InitProviderContext(ctx)
}

func (r *resource) Read(ctx context.Context, req testtypes.ReadRequest, resp *testtypes.ReadResponse) {
	core.InitProviderContext(ctx)
	iaas.NewAPIClient()
}

func sdkCallOutsideLifecycleMethod() {
	iaas.NewAPIClient()
}
