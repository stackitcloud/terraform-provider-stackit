package token_test

import (
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

func TestCreate_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	validUntil := time.Now().Add(10 * time.Minute)

	createTokenResp := &modelexperiments.CreateTokenResponse{
		Token: modelexperiments.Token{
			Content:     "token",
			Description: &description,
			Id:          tokenId.String(),
			Name:        name,
			Region:      region,
			State:       "creating",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().CreateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().CreateInstanceTokenExecute(gomock.Any()).Return(createTokenResp, nil)

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
	}

	req := testutils.CreateInstanceTokenRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	tokenRes.Create(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Create should succeed, but got errors: %v", resp.Diagnostics.Errors())

	var createdState token.Model
	diags := resp.State.Get(tc.Ctx, &createdState)
	require.False(t, diags.HasError(), "Failed to get state")

	require.Equal(t, tokenId.String(), createdState.TokenId.ValueString(), "Should be equal")
	require.Equal(t, projectId.String(), createdState.ProjectId.ValueString(), "Should be equal")
	require.Equal(t, instanceId.String(), createdState.InstanceId.ValueString(), "Should be equal")
	require.Equal(t, name, createdState.Name.ValueString(), "Should be equal")
	require.Equal(t, "active", createdState.State.ValueString(), "Should be equal")
	require.Equal(t, description, createdState.Description.ValueString(), "Should be equal")
	require.Equal(t, "token", createdState.Token.ValueString(), "Should be equal")
	require.Equal(t, utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String()).ValueString(), createdState.Id.ValueString(), "Should be equal")
}

func TestCreate_TokenIdEmpty(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	validUntil := time.Now()

	createTokenResp := &modelexperiments.CreateTokenResponse{
		Token: modelexperiments.Token{
			Content:     "token",
			Description: &description,
			Id:          "",
			Name:        name,
			Region:      region,
			State:       "creating",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().CreateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().CreateInstanceTokenExecute(gomock.Any()).Return(createTokenResp, nil)

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
	}

	req := testutils.CreateInstanceTokenRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	tokenRes.Create(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Create should not succeed but got no errors")

	// state should not be created
	var createdState *token.Model
	diags := resp.State.Get(tc.Ctx, &createdState)
	require.False(t, diags.HasError(), "Failed to get state")
	require.Nil(t, createdState)
}

func TestCreate_CreateTokenFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: 400,
	}
	tc.MockInstanceCLient.EXPECT().CreateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().CreateInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

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
	}

	req := testutils.CreateInstanceTokenRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	tokenRes.Create(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Create should not succeed but got no errors")

	// state should not be created
	var createdState *token.Model
	diags := resp.State.Get(tc.Ctx, &createdState)
	require.False(t, diags.HasError(), "Failed to get state")
	require.Nil(t, createdState)
}

func TestCreate_GetTokenFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	validUntil := time.Now().Add(10 * time.Minute)

	createTokenResp := &modelexperiments.CreateTokenResponse{
		Token: modelexperiments.Token{
			Content:     "token",
			Description: &description,
			Id:          tokenId.String(),
			Name:        name,
			Region:      region,
			State:       "creating",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().CreateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().CreateInstanceTokenExecute(gomock.Any()).Return(createTokenResp, nil)

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: 404,
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
	}

	req := testutils.CreateInstanceTokenRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	tokenRes.Create(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Create should not succeed but got no errors")

	// state should be created
	var createdState token.Model
	diags := resp.State.Get(tc.Ctx, &createdState)
	require.False(t, diags.HasError(), "Failed to get state")

	require.Equal(t, tokenId.String(), createdState.TokenId.ValueString(), "Should be equal")
	require.Equal(t, projectId.String(), createdState.ProjectId.ValueString(), "Should be equal")
	require.Equal(t, instanceId.String(), createdState.InstanceId.ValueString(), "Should be equal")
	require.Equal(t, name, createdState.Name.ValueString(), "Should be equal")
	require.Equal(t, "unknown", createdState.State.ValueString(), "Should be equal")
	require.Equal(t, description, createdState.Description.ValueString(), "Should be equal")
	require.Equal(t, "token", createdState.Token.ValueString(), "Should be equal")
	require.Equal(t, utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String()).ValueString(), createdState.Id.ValueString(), "Should be equal")
}
