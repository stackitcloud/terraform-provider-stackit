package tflogresponse

import (
	"context"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/testtypes"
)

type resource struct{}

func (r *resource) Create(ctx context.Context, req testtypes.CreateRequest, resp *testtypes.CreateResponse) {
	ctx = core.InitProviderContext(ctx)
	iaas.NewAPIClient()
	ctx = core.LogResponse(ctx)
}

func (r *resource) Read(ctx context.Context, req testtypes.ReadRequest, resp *testtypes.ReadResponse) {
	ctx = core.InitProviderContext(ctx)
	ctx = core.LogResponse(ctx) // want "tflogresponse: invalid sequence: LogResponse called without an intermediate call to github.com/stackitcloud/stackit-sdk-go after InitProviderContext"
}

func (r *resource) Update(ctx context.Context, req testtypes.UpdateRequest, resp *testtypes.UpdateResponse) {
	ctx = core.InitProviderContext(ctx) // want "tflogresponse: invalid sequence: InitProviderContext was called, but LogResponse was never called afterwards"
	iaas.NewAPIClient()
}

func (r *resource) Delete(ctx context.Context, req testtypes.DeleteRequest, resp *testtypes.DeleteResponse) {
	ctx = core.InitProviderContext(ctx)
	iaas.NewAPIClient()
	ctx = core.LogResponse(ctx)
}

func nonLifecycleMethod(ctx context.Context) {
	ctx = core.InitProviderContext(ctx)
	iaas.NewAPIClient()
}

// fals positive, SDK call through helper func

type falsePositive struct{}

func (f *falsePositive) Read(ctx context.Context, req testtypes.ReadRequest, resp *testtypes.ReadResponse) {
	ctx = core.InitProviderContext(ctx)
	indirection()
	ctx = core.LogResponse(ctx) // want "tflogresponse: invalid sequence: LogResponse called without an intermediate call to github.com/stackitcloud/stackit-sdk-go after InitProviderContext"
}

func indirection() {
	iaas.NewAPIClient()
}
