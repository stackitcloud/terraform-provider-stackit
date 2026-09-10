package tfwriteid

import (
	"context"

	corewait "github.com/stackitcloud/stackit-sdk-go/core/wait"
	serviceenablementwait "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api/wait"
	"github.com/stackitcloud/terraform-provider-stackit/testtypes"
)

type resource struct{}

func (r *resource) Create(ctx context.Context, req testtypes.CreateRequest, resp *testtypes.CreateResponse) {
	_, _ = serviceenablementwait.EnableServiceWaitHandler(ctx, nil, "", "", "").WaitWithContext(ctx)

	waiter := corewait.New(func() (bool, *struct{}, error) { return true, nil, nil })
	_, _ = waiter.WaitWithContext(ctx) // want "tfwriteid: call to wait handler from github.com/stackitcloud/stackit-sdk-go must happen AFTER github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils.SetAndLogStateFields is called in Create github.com/stackitcloud/stackit-sdk-go/core/wait"
}
