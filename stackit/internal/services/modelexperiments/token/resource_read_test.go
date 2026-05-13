package token_test

import (
	"net/http"
	"testing"
	"time"

	modelexperiments "dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/modelexperiments/v1api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/token"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRead_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	validUntil := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	tokenContent := "token"
	state := "active"
	id := utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String())

	getTokenResp := &modelexperiments.GetTokenResponse{
		Token: modelexperiments.TokenMetadata{
			Description: &description,
			Id:          tokenId.String(),
			Name:        name,
			Region:      region,
			State:       "active",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().GetInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().GetInstanceTokenExecute(gomock.Any()).Return(getTokenResp, nil)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Name:        types.StringValue(name),
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		InstanceId:  types.StringValue(instanceId.String()),
		Labels:      types.MapNull(types.StringType),
		Token:       types.StringValue(tokenContent),
		TokenId:     types.StringValue(tokenId.String()),
		Id:          id,
		State:       types.StringValue(state),
		ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
	}

	req := testutils.ReadTokenRequest(tc.Ctx, schemaResp, model)
	resp := testutils.ReadTokenResponse(tc.Ctx, schemaResp, nil)

	tokenRes.Read(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Get should succeed, but got errors: %v", resp.Diagnostics.Errors())

	// state should be set
	var refreshedState token.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	require.False(t, diags.HasError(), "Failed to get state")

	require.Equal(t, tokenId.String(), refreshedState.TokenId.ValueString(), "Should be equal")
	require.Equal(t, projectId.String(), refreshedState.ProjectId.ValueString(), "Should be equal")
	require.Equal(t, instanceId.String(), refreshedState.InstanceId.ValueString(), "Should be equal")
	require.Equal(t, name, refreshedState.Name.ValueString(), "Should be equal")
	require.Equal(t, "active", refreshedState.State.ValueString(), "Should be equal")
	require.Equal(t, description, refreshedState.Description.ValueString(), "Should be equal")
	require.Equal(t, tokenContent, refreshedState.Token.ValueString(), "Should be equal")
	require.Equal(t, id.ValueString(), refreshedState.Id.ValueString(), "Should be equal")
	require.Equal(t, region, refreshedState.Region.ValueString(), "Should be equal")
	require.Equal(t, "2099-01-01T00:00:00Z", refreshedState.ValidUntil.ValueString(), "Should be equal")
}

func TestRead_TokenIdEmptyFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	tokenContent := "token"
	state := "active"
	id := utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String())

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Name:        types.StringValue(name),
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		InstanceId:  types.StringValue(instanceId.String()),
		Labels:      types.MapNull(types.StringType),
		Token:       types.StringValue(tokenContent),
		TokenId:     types.StringValue(""),
		Id:          id,
		State:       types.StringValue(state),
		ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
	}

	req := testutils.ReadTokenRequest(tc.Ctx, schemaResp, model)
	resp := testutils.ReadTokenResponse(tc.Ctx, schemaResp, &model)

	tokenRes.Read(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Get should succeed, but got errors: %v", resp.Diagnostics.Errors())

	// state should be removed
	var refreshedState *token.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	require.False(t, diags.HasError(), "Failed to get state")
	require.Nil(t, refreshedState)
}

func TestRead_TokenNotFound(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	tokenContent := "token"
	state := "active"
	id := utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String())

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusNotFound,
	}
	tc.MockInstanceCLient.EXPECT().GetInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().GetInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Name:        types.StringValue(name),
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		InstanceId:  types.StringValue(instanceId.String()),
		Labels:      types.MapNull(types.StringType),
		Token:       types.StringValue(tokenContent),
		TokenId:     types.StringValue(tokenId.String()),
		Id:          id,
		State:       types.StringValue(state),
		ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
	}

	req := testutils.ReadTokenRequest(tc.Ctx, schemaResp, model)
	resp := testutils.ReadTokenResponse(tc.Ctx, schemaResp, &model)

	tokenRes.Read(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Get should succeed, but got errors: %v", resp.Diagnostics.Errors())

	// state should be removed
	var refreshedState *token.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	require.False(t, diags.HasError(), "Failed to get state")
	require.Nil(t, refreshedState)

}

func TestRead_GetTokenRequestFailed(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	tokenContent := "token"
	state := "active"
	id := utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String())

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusInternalServerError,
	}
	tc.MockInstanceCLient.EXPECT().GetInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().GetInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Name:        types.StringValue(name),
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		InstanceId:  types.StringValue(instanceId.String()),
		Labels:      types.MapNull(types.StringType),
		Token:       types.StringValue(tokenContent),
		TokenId:     types.StringValue(tokenId.String()),
		Id:          id,
		State:       types.StringValue(state),
		ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
	}

	req := testutils.ReadTokenRequest(tc.Ctx, schemaResp, model)
	resp := testutils.ReadTokenResponse(tc.Ctx, schemaResp, &model)

	tokenRes.Read(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Get should not succeed")

	// state should not be edited
	var refreshedState token.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	require.False(t, diags.HasError(), "Failed to get state")

	require.Equal(t, tokenId.String(), refreshedState.TokenId.ValueString(), "Should be equal")
	require.Equal(t, projectId.String(), refreshedState.ProjectId.ValueString(), "Should be equal")
	require.Equal(t, instanceId.String(), refreshedState.InstanceId.ValueString(), "Should be equal")
	require.Equal(t, name, refreshedState.Name.ValueString(), "Should be equal")
	require.Equal(t, "active", refreshedState.State.ValueString(), "Should be equal")
	require.Equal(t, description, refreshedState.Description.ValueString(), "Should be equal")
	require.Equal(t, tokenContent, refreshedState.Token.ValueString(), "Should be equal")
	require.Equal(t, id.ValueString(), refreshedState.Id.ValueString(), "Should be equal")
	require.Equal(t, region, refreshedState.Region.ValueString(), "Should be equal")
	require.Equal(t, "2099-01-01T00:00:00Z", refreshedState.ValidUntil.ValueString(), "Should be equal")
}

func TestRead_TokenInvalidError(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	tokenContent := "token"
	state := "active"
	id := utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String())
	validUntil := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	getTokenResp := &modelexperiments.GetTokenResponse{
		Token: modelexperiments.TokenMetadata{
			Description: &description,
			Id:          tokenId.String(),
			Name:        name,
			Region:      region,
			State:       "inactive",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().GetInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().GetInstanceTokenExecute(gomock.Any()).Return(getTokenResp, nil)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Name:        types.StringValue(name),
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		InstanceId:  types.StringValue(instanceId.String()),
		Labels:      types.MapNull(types.StringType),
		Token:       types.StringValue(tokenContent),
		TokenId:     types.StringValue(tokenId.String()),
		Id:          id,
		State:       types.StringValue(state),
		ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
	}

	req := testutils.ReadTokenRequest(tc.Ctx, schemaResp, model)
	resp := testutils.ReadTokenResponse(tc.Ctx, schemaResp, &model)

	tokenRes.Read(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Get should succeed, but got errors: %v", resp.Diagnostics.Errors())

	// state should be removed
	var refreshedState *token.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	require.False(t, diags.HasError(), "Failed to get state")
	require.Nil(t, refreshedState)
}
