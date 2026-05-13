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

func TestUpdate_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	nameUpdated := "token update"
	region := "eu01"
	description := "token description"
	descriptionUpdated := "description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	validUntil := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	tokenContent := "token"
	state := "active"
	id := utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String())

	updateTokenResp := &modelexperiments.PartialUpdateTokenResponse{
		Token: modelexperiments.TokenMetadata{
			Description: &descriptionUpdated,
			Id:          tokenId.String(),
			Name:        nameUpdated,
			Region:      region,
			State:       "active",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiPartialUpdateInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceTokenExecute(gomock.Any()).Return(updateTokenResp, nil)

	providerData := core.ProviderData{
		DefaultRegion: region,
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	currentState := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		InstanceId:  types.StringValue(instanceId.String()),
		Name:        types.StringValue(name),
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		Labels:      types.MapNull(types.StringType),
		Token:       types.StringValue(tokenContent),
		TokenId:     types.StringValue(tokenId.String()),
		Id:          id,
		State:       types.StringValue(state),
		ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
	}

	plannedState := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Name:        types.StringValue(nameUpdated),
		Region:      types.StringValue(region),
		Description: types.StringValue(descriptionUpdated),
		InstanceId:  types.StringValue(instanceId.String()),
		Labels:      types.MapNull(types.StringType),
	}

	req := testutils.UpdateTokenRequest(tc.Ctx, schemaResp, currentState, plannedState)
	resp := testutils.UpdateTokenResponse(tc.Ctx, schemaResp, &currentState)

	// Execute Update
	tokenRes.Update(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Update should succeed, but got errors: %v", resp.Diagnostics.Errors())

	// state should be updated
	var updatedState token.Model
	diags := resp.State.Get(tc.Ctx, &updatedState)
	require.False(t, diags.HasError(), "Failed to get state")

	require.Equal(t, tokenId.String(), updatedState.TokenId.ValueString(), "Should be equal")
	require.Equal(t, projectId.String(), updatedState.ProjectId.ValueString(), "Should be equal")
	require.Equal(t, instanceId.String(), updatedState.InstanceId.ValueString(), "Should be equal")
	require.Equal(t, nameUpdated, updatedState.Name.ValueString(), "Should be equal")
	require.Equal(t, "active", updatedState.State.ValueString(), "Should be equal")
	require.Equal(t, descriptionUpdated, updatedState.Description.ValueString(), "Should be equal")
	require.Equal(t, tokenContent, updatedState.Token.ValueString(), "Should be equal")
	require.Equal(t, id.ValueString(), updatedState.Id.ValueString(), "Should be equal")
	require.Equal(t, region, updatedState.Region.ValueString(), "Should be equal")
	require.Equal(t, "2099-01-01T00:00:00Z", updatedState.ValidUntil.ValueString(), "Should be equal")
}

func TestUpdate_TokenNotFound(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	nameUpdated := "token update"
	region := "eu01"
	description := "token description"
	descriptionUpdated := "description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	tokenContent := "token"
	state := "active"
	id := utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String())

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusNotFound,
	}
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiPartialUpdateInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: region,
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	currentState := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		InstanceId:  types.StringValue(instanceId.String()),
		Name:        types.StringValue(name),
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		Labels:      types.MapNull(types.StringType),
		Token:       types.StringValue(tokenContent),
		TokenId:     types.StringValue(tokenId.String()),
		Id:          id,
		State:       types.StringValue(state),
		ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
	}

	plannedState := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Name:        types.StringValue(nameUpdated),
		Region:      types.StringValue(region),
		Description: types.StringValue(descriptionUpdated),
		InstanceId:  types.StringValue(instanceId.String()),
		Labels:      types.MapNull(types.StringType),
	}

	req := testutils.UpdateTokenRequest(tc.Ctx, schemaResp, currentState, plannedState)
	resp := testutils.UpdateTokenResponse(tc.Ctx, schemaResp, &currentState)

	// Execute Update
	tokenRes.Update(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Update should succeed, but got errors: %v", resp.Diagnostics.Errors())

	// state should be removed
	var updatedState *token.Model
	diags := resp.State.Get(tc.Ctx, &updatedState)
	require.False(t, diags.HasError(), "Failed to get state")
	require.Nil(t, updatedState)
}

func TestUpdate_TokenUpdateError(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	nameUpdated := "token update"
	region := "eu01"
	description := "token description"
	descriptionUpdated := "description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	tokenContent := "token"
	state := "active"
	id := utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String())

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusInternalServerError,
	}
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiPartialUpdateInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: region,
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	currentState := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		InstanceId:  types.StringValue(instanceId.String()),
		Name:        types.StringValue(name),
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		Labels:      types.MapNull(types.StringType),
		Token:       types.StringValue(tokenContent),
		TokenId:     types.StringValue(tokenId.String()),
		Id:          id,
		State:       types.StringValue(state),
		ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
	}

	plannedState := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Name:        types.StringValue(nameUpdated),
		Region:      types.StringValue(region),
		Description: types.StringValue(descriptionUpdated),
		InstanceId:  types.StringValue(instanceId.String()),
		Labels:      types.MapNull(types.StringType),
	}

	req := testutils.UpdateTokenRequest(tc.Ctx, schemaResp, currentState, plannedState)
	resp := testutils.UpdateTokenResponse(tc.Ctx, schemaResp, &currentState)

	// Execute Update
	tokenRes.Update(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Update should not succeed")

	// state should not be changed
	var updatedState token.Model
	diags := resp.State.Get(tc.Ctx, &updatedState)
	require.False(t, diags.HasError(), "Failed to get state")

	require.Equal(t, tokenId.String(), updatedState.TokenId.ValueString(), "Should be equal")
	require.Equal(t, projectId.String(), updatedState.ProjectId.ValueString(), "Should be equal")
	require.Equal(t, instanceId.String(), updatedState.InstanceId.ValueString(), "Should be equal")
	require.Equal(t, name, updatedState.Name.ValueString(), "Should be equal")
	require.Equal(t, "active", updatedState.State.ValueString(), "Should be equal")
	require.Equal(t, description, updatedState.Description.ValueString(), "Should be equal")
	require.Equal(t, tokenContent, updatedState.Token.ValueString(), "Should be equal")
	require.Equal(t, id.ValueString(), updatedState.Id.ValueString(), "Should be equal")
	require.Equal(t, region, updatedState.Region.ValueString(), "Should be equal")
	require.Equal(t, "2099-01-01T00:00:00Z", updatedState.ValidUntil.ValueString(), "Should be equal")
}

func TestUpdate_TokenInvalidStateError(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	nameUpdated := "token update"
	region := "eu01"
	description := "token description"
	descriptionUpdated := "description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	tokenContent := "token"
	state := "active"
	id := utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String())
	validUntil := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	updateTokenResp := &modelexperiments.PartialUpdateTokenResponse{
		Token: modelexperiments.TokenMetadata{
			Description: &descriptionUpdated,
			Id:          tokenId.String(),
			Name:        nameUpdated,
			Region:      region,
			State:       "inactive",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiPartialUpdateInstanceTokenRequest{})
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceTokenExecute(gomock.Any()).Return(updateTokenResp, nil)

	providerData := core.ProviderData{
		DefaultRegion: region,
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	currentState := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		InstanceId:  types.StringValue(instanceId.String()),
		Name:        types.StringValue(name),
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		Labels:      types.MapNull(types.StringType),
		Token:       types.StringValue(tokenContent),
		TokenId:     types.StringValue(tokenId.String()),
		Id:          id,
		State:       types.StringValue(state),
		ValidUntil:  types.StringValue("2099-01-01T00:00:00Z"),
	}

	plannedState := token.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Name:        types.StringValue(nameUpdated),
		Region:      types.StringValue(region),
		Description: types.StringValue(descriptionUpdated),
		InstanceId:  types.StringValue(instanceId.String()),
		Labels:      types.MapNull(types.StringType),
	}

	req := testutils.UpdateTokenRequest(tc.Ctx, schemaResp, currentState, plannedState)
	resp := testutils.UpdateTokenResponse(tc.Ctx, schemaResp, &currentState)

	// Execute Update
	tokenRes.Update(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Update should succeed")

	// state should not be removed
	var updatedState *token.Model
	diags := resp.State.Get(tc.Ctx, &updatedState)
	require.False(t, diags.HasError(), "Failed to get state")
	require.Nil(t, updatedState)
}
