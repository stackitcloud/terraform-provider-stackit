package tfstateremove

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type testResource struct{}

func (*testResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // want "tfstateremove: resource Read must call resp.State.RemoveResource\\(ctx\\)"
}

type testResourceWithRemove struct{}

func (*testResourceWithRemove) Read(ctx context.Context, req resource.ReadRequest, response *resource.ReadResponse) {
	response.State.RemoveResource(ctx)
}

type OtherReadRequest struct{}
type OtherReadResponse struct{}

type otherReader struct{}

// A Read method using lookalike local types is not a Terraform resource Read method.
func (*otherReader) Read(ctx context.Context, req OtherReadRequest, resp *OtherReadResponse) {}
