package token_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	"go.uber.org/mock/gomock"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/token"
)

func TestDelete_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	instanceId := uuid.New()
	tokenId := uuid.New()

	tc.MockInstanceCLient.EXPECT().DeleteInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceTokenExecute(gomock.Any()).Return(nil, nil)

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusNotFound,
	}
	tc.MockInstanceCLient.EXPECT().GetInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceTokenRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().GetInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	state := token.Model{
		ProjectId:  types.StringValue(projectId.String()),
		InstanceId: types.StringValue(instanceId.String()),
		Region:     types.StringValue(region),
		Name:       types.StringValue(name),
		TokenId:    types.StringValue(tokenId.String()),
		Labels:     types.MapNull(types.StringType),
	}

	req := testutils.DeleteTokenRequest(tc.Ctx, schemaResp, state)
	resp := testutils.DeleteTokenResponse(tc.Ctx, schemaResp, nil)

	tokenRes.Delete(tc.Ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}
}

func TestDelete_DeleteTokenFailed(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	instanceId := uuid.New()
	tokenId := uuid.New()

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusInternalServerError,
	}
	tc.MockInstanceCLient.EXPECT().DeleteInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	state := token.Model{
		ProjectId:  types.StringValue(projectId.String()),
		InstanceId: types.StringValue(instanceId.String()),
		Region:     types.StringValue(region),
		Name:       types.StringValue(name),
		TokenId:    types.StringValue(tokenId.String()),
		Labels:     types.MapNull(types.StringType),
	}

	req := testutils.DeleteTokenRequest(tc.Ctx, schemaResp, state)
	resp := testutils.DeleteTokenResponse(tc.Ctx, schemaResp, &state)

	tokenRes.Delete(tc.Ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Delete should not succeed")
	}

	// state should not be removed
	var deletedState token.Model
	diags := resp.State.Get(tc.Ctx, &deletedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}

	if instanceId.String() != deletedState.InstanceId.ValueString() {
		t.Fatalf("expected %v, got %v", instanceId.String(), deletedState.InstanceId.ValueString())
	}
}

func TestDelete_TokenNotFound(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	instanceId := uuid.New()
	tokenId := uuid.New()

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusNotFound,
	}
	tc.MockInstanceCLient.EXPECT().DeleteInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	state := token.Model{
		ProjectId:  types.StringValue(projectId.String()),
		InstanceId: types.StringValue(instanceId.String()),
		Region:     types.StringValue(region),
		Name:       types.StringValue(name),
		TokenId:    types.StringValue(tokenId.String()),
		Labels:     types.MapNull(types.StringType),
	}

	req := testutils.DeleteTokenRequest(tc.Ctx, schemaResp, state)
	resp := testutils.DeleteTokenResponse(tc.Ctx, schemaResp, &state)

	tokenRes.Delete(tc.Ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}

	// state should be removed
	var deletedState *token.Model
	diags := resp.State.Get(tc.Ctx, &deletedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}
	if deletedState != nil {
		t.Fatalf("should be nil")
	}
}

func TestDelete_GetTokenFailed(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	instanceId := uuid.New()
	tokenId := uuid.New()

	tc.MockInstanceCLient.EXPECT().DeleteInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceTokenExecute(gomock.Any()).Return(nil, nil)

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusInternalServerError,
	}
	tc.MockInstanceCLient.EXPECT().GetInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceTokenRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().GetInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	state := token.Model{
		ProjectId:  types.StringValue(projectId.String()),
		InstanceId: types.StringValue(instanceId.String()),
		Region:     types.StringValue(region),
		Name:       types.StringValue(name),
		TokenId:    types.StringValue(tokenId.String()),
		Labels:     types.MapNull(types.StringType),
	}

	req := testutils.DeleteTokenRequest(tc.Ctx, schemaResp, state)
	resp := testutils.DeleteTokenResponse(tc.Ctx, schemaResp, &state)

	tokenRes.Delete(tc.Ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Delete should not succeed")
	}

	// state should not be removed
	var deletedState token.Model
	diags := resp.State.Get(tc.Ctx, &deletedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}

	if instanceId.String() != deletedState.InstanceId.ValueString() {
		t.Fatalf("expected %v, got %v", instanceId.String(), deletedState.InstanceId.ValueString())
	}
}
