package core

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	sdkconfig "github.com/stackitcloud/stackit-sdk-go/core/config"
)

type testDataSource struct {
	readContext context.Context
}

func (d *testDataSource) Metadata(context.Context, datasource.MetadataRequest, *datasource.MetadataResponse) {
}
func (d *testDataSource) Schema(context.Context, datasource.SchemaRequest, *datasource.SchemaResponse) {
}
func (d *testDataSource) Read(ctx context.Context, _ datasource.ReadRequest, _ *datasource.ReadResponse) { //nolint:gocritic // Framework signature requires values.
	d.readContext = ctx
}

type testResource struct {
	contexts map[string]context.Context
}

func (r *testResource) Metadata(context.Context, resource.MetadataRequest, *resource.MetadataResponse) {
}
func (r *testResource) Schema(context.Context, resource.SchemaRequest, *resource.SchemaResponse) {}
func (r *testResource) Create(ctx context.Context, _ resource.CreateRequest, _ *resource.CreateResponse) { //nolint:gocritic // Framework signature requires values.
	r.contexts["create"] = ctx
}
func (r *testResource) Read(ctx context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) { //nolint:gocritic // Framework signature requires values.
	r.contexts["read"] = ctx
}
func (r *testResource) Update(ctx context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) { //nolint:gocritic // Framework signature requires values.
	r.contexts["update"] = ctx
}
func (r *testResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) { //nolint:gocritic // Framework signature requires values.
	r.contexts["delete"] = ctx
}

func TestWrapDataSourceReadInitializesProviderContext(t *testing.T) {
	inner := &testDataSource{}
	wrapped := WrapDataSource(inner)

	wrapped.Read(context.Background(), datasource.ReadRequest{}, &datasource.ReadResponse{})

	if inner.readContext.Value(sdkconfig.ContextHTTPResponse) == nil {
		t.Error("Read context does not capture HTTP responses")
	}
}

func TestWrapResourceDoesNotAddIdentitySupport(t *testing.T) {
	if _, ok := WrapResource(&testResource{}).(resource.ResourceWithIdentity); ok {
		t.Error("wrapper must not add identity support when the resource does not implement it")
	}
}

func TestWrapResourceCRUDInitializesProviderContext(t *testing.T) {
	inner := &testResource{contexts: make(map[string]context.Context)}
	wrapped := WrapResource(inner)

	wrapped.Create(context.Background(), resource.CreateRequest{}, &resource.CreateResponse{})
	wrapped.Read(context.Background(), resource.ReadRequest{}, &resource.ReadResponse{})
	wrapped.Update(context.Background(), resource.UpdateRequest{}, &resource.UpdateResponse{})
	wrapped.Delete(context.Background(), resource.DeleteRequest{}, &resource.DeleteResponse{})

	for _, operation := range []string{"create", "read", "update", "delete"} {
		if inner.contexts[operation].Value(sdkconfig.ContextHTTPResponse) == nil {
			t.Errorf("%s context does not capture HTTP responses", operation)
		}
	}
}
