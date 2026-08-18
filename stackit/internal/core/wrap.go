package core

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// WrapDataSource wraps a data source so its Read method can capture the HTTP
// response used for provider logging.
func WrapDataSource(inner datasource.DataSource) datasource.DataSource {
	return &wrappedDataSource{inner: inner}
}

type wrappedDataSource struct {
	inner datasource.DataSource
}

var (
	_ datasource.DataSource                     = &wrappedDataSource{}
	_ datasource.DataSourceWithConfigure        = &wrappedDataSource{}
	_ datasource.DataSourceWithConfigValidators = &wrappedDataSource{}
	_ datasource.DataSourceWithValidateConfig   = &wrappedDataSource{}
)

func (w *wrappedDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	w.inner.Metadata(ctx, req, resp)
}

func (w *wrappedDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	w.inner.Schema(ctx, req, resp)
}

func (w *wrappedDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { //nolint:gocritic,tflogresponse // Framework signature requires values; wrapped method logs the response.
	ctx = InitProviderContext(ctx) //nolint:tflogresponse // The wrapped CRUD method logs the response.
	w.inner.Read(ctx, req, resp)
}

func (w *wrappedDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if inner, ok := w.inner.(datasource.DataSourceWithConfigure); ok {
		inner.Configure(ctx, req, resp)
	}
}

func (w *wrappedDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	if inner, ok := w.inner.(datasource.DataSourceWithConfigValidators); ok {
		return inner.ConfigValidators(ctx)
	}

	return nil
}

func (w *wrappedDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	if inner, ok := w.inner.(datasource.DataSourceWithValidateConfig); ok {
		inner.ValidateConfig(ctx, req, resp)
	}
}

// WrapResource wraps a resource so its CRUD methods can capture the HTTP
// response used for provider logging.
func WrapResource(inner resource.Resource) resource.Resource {
	wrapped := &wrappedResource{inner: inner}
	_, hasIdentity := inner.(resource.ResourceWithIdentity)
	_, hasUpgradeIdentity := inner.(resource.ResourceWithUpgradeIdentity)

	switch {
	case hasIdentity && hasUpgradeIdentity:
		return &wrappedResourceWithIdentityAndUpgrade{wrappedResourceWithIdentity: wrappedResourceWithIdentity{wrappedResource: wrapped}}
	case hasIdentity:
		return &wrappedResourceWithIdentity{wrappedResource: wrapped}
	case hasUpgradeIdentity:
		return &wrappedResourceWithUpgradeIdentity{wrappedResource: wrapped}
	default:
		return wrapped
	}
}

type wrappedResource struct {
	inner resource.Resource
}

var (
	_ resource.Resource                     = &wrappedResource{}
	_ resource.ResourceWithConfigure        = &wrappedResource{}
	_ resource.ResourceWithConfigValidators = &wrappedResource{}
	_ resource.ResourceWithImportState      = &wrappedResource{}
	_ resource.ResourceWithModifyPlan       = &wrappedResource{}
	_ resource.ResourceWithMoveState        = &wrappedResource{}
	_ resource.ResourceWithUpgradeState     = &wrappedResource{}
	_ resource.ResourceWithValidateConfig   = &wrappedResource{}
)

func (w *wrappedResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	w.inner.Metadata(ctx, req, resp)
}

func (w *wrappedResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	w.inner.Schema(ctx, req, resp)
}

func (w *wrappedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { //nolint:gocritic,tflogresponse // Framework signature requires values; wrapped method logs the response.
	ctx = InitProviderContext(ctx) //nolint:tflogresponse // The wrapped CRUD method logs the response.
	w.inner.Create(ctx, req, resp)
}

func (w *wrappedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { //nolint:gocritic,tflogresponse // Framework signature requires values; wrapped method logs the response.
	ctx = InitProviderContext(ctx) //nolint:tflogresponse // The wrapped CRUD method logs the response.
	w.inner.Read(ctx, req, resp)
}

func (w *wrappedResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { //nolint:gocritic,tflogresponse // Framework signature requires values; wrapped method logs the response.
	ctx = InitProviderContext(ctx) //nolint:tflogresponse // The wrapped CRUD method logs the response.
	w.inner.Update(ctx, req, resp)
}

func (w *wrappedResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { //nolint:gocritic,tflogresponse // Framework signature requires values; wrapped method logs the response.
	ctx = InitProviderContext(ctx) //nolint:tflogresponse // The wrapped CRUD method logs the response.
	w.inner.Delete(ctx, req, resp)
}

func (w *wrappedResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if inner, ok := w.inner.(resource.ResourceWithConfigure); ok {
		inner.Configure(ctx, req, resp)
	}
}

func (w *wrappedResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	if inner, ok := w.inner.(resource.ResourceWithConfigValidators); ok {
		return inner.ConfigValidators(ctx)
	}

	return nil
}

func (w *wrappedResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if inner, ok := w.inner.(resource.ResourceWithImportState); ok {
		inner.ImportState(ctx, req, resp)
		return
	}

	resp.Diagnostics.AddError(
		"Resource Import Not Implemented",
		"This resource does not support import. Please contact the provider developer for additional information.",
	)
}

func (w *wrappedResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { //nolint:gocritic // Framework signature requires values.
	if inner, ok := w.inner.(resource.ResourceWithModifyPlan); ok {
		inner.ModifyPlan(ctx, req, resp)
	}
}

func (w *wrappedResource) MoveState(ctx context.Context) []resource.StateMover {
	if inner, ok := w.inner.(resource.ResourceWithMoveState); ok {
		return inner.MoveState(ctx)
	}

	return nil
}

func (w *wrappedResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	if inner, ok := w.inner.(resource.ResourceWithUpgradeState); ok {
		return inner.UpgradeState(ctx)
	}

	return nil
}

func (w *wrappedResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if inner, ok := w.inner.(resource.ResourceWithValidateConfig); ok {
		inner.ValidateConfig(ctx, req, resp)
	}
}

type wrappedResourceWithIdentity struct {
	*wrappedResource
}

var _ resource.ResourceWithIdentity = &wrappedResourceWithIdentity{}

func (w *wrappedResourceWithIdentity) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	if inner, ok := w.inner.(resource.ResourceWithIdentity); ok {
		inner.IdentitySchema(ctx, req, resp)
	}
}

type wrappedResourceWithUpgradeIdentity struct {
	*wrappedResource
}

var _ resource.ResourceWithUpgradeIdentity = &wrappedResourceWithUpgradeIdentity{}

func (w *wrappedResourceWithUpgradeIdentity) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	if inner, ok := w.inner.(resource.ResourceWithUpgradeIdentity); ok {
		return inner.UpgradeIdentity(ctx)
	}

	return nil
}

type wrappedResourceWithIdentityAndUpgrade struct {
	wrappedResourceWithIdentity
}

var _ resource.ResourceWithUpgradeIdentity = &wrappedResourceWithIdentityAndUpgrade{}

func (w *wrappedResourceWithIdentityAndUpgrade) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	if inner, ok := w.inner.(resource.ResourceWithUpgradeIdentity); ok {
		return inner.UpgradeIdentity(ctx)
	}

	return nil
}
