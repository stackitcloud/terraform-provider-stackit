package tflogresponse

import (
	"context"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/testtypes"
)

type resource struct{}

func (r *resource) Create(ctx context.Context, req testtypes.CreateRequest, resp *testtypes.CreateResponse) {
	c, _ := iaas.NewAPIClient()
	ctx = core.InitProviderContext(ctx)
	c.DefaultAPI.AddNetworkToServer(ctx, "", "", "", "").Execute()
	ctx = core.LogResponse(ctx)
}

func (r *resource) Read(ctx context.Context, req testtypes.ReadRequest, resp *testtypes.ReadResponse) {
	ctx = core.InitProviderContext(ctx)
	ctx = core.LogResponse(ctx) // want "tflogresponse: invalid sequence: LogResponse called without an intermediate call to github.com/stackitcloud/stackit-sdk-go after InitProviderContext"
}

func (r *resource) Update(ctx context.Context, req testtypes.UpdateRequest, resp *testtypes.UpdateResponse) {
	var service iaas.DefaultAPIService
	ctx = core.InitProviderContext(ctx) // want "tflogresponse: invalid sequence: InitProviderContext was called, but LogResponse was never called afterwards"
	service.AddNetworkToServerExecute(iaas.ApiAddNetworkToServerRequest{})
}

func (r *resource) Delete(ctx context.Context, req testtypes.DeleteRequest, resp *testtypes.DeleteResponse) {
	var service iaas.DefaultAPIService
	ctx = core.InitProviderContext(ctx)
	service.AddNetworkToServerExecute(iaas.ApiAddNetworkToServerRequest{})
	ctx = core.LogResponse(ctx)
}

func nonLifecycleMethod(ctx context.Context) {
	var service iaas.DefaultAPIService
	ctx = core.InitProviderContext(ctx)
	service.AddNetworkToServerExecute(iaas.ApiAddNetworkToServerRequest{})
}

// SDK service call through helper func

type resource2 struct{}

func (f *resource2) Read(ctx context.Context, req testtypes.ReadRequest, resp *testtypes.ReadResponse) {
	ctx = core.InitProviderContext(ctx)
	indirection(nil)
	ctx = core.LogResponse(ctx)
}

func indirection(service *iaas.DefaultAPIService) {
	service.AddNetworkToServerExecute(iaas.ApiAddNetworkToServerRequest{})
}

func wrapper[T any](fn func() (*T, error)) (*T, error) {
	return fn()
}

type resource3 struct{}

func (r *resource3) Delete(ctx context.Context, req testtypes.DeleteRequest, resp *testtypes.DeleteResponse) {
	c, _ := iaas.NewAPIClient()
	ctx = core.InitProviderContext(ctx)
	wrapper(c.DefaultAPI.CreateAffinityGroup(ctx, "", "").Execute)
	ctx = core.LogResponse(ctx)
}
