package token_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	"go.uber.org/mock/gomock"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/token"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
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

	updateTokenResp := &modelexperiments.PartialUpdateInstanceTokenResponse{
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
	if resp.Diagnostics.HasError() {
		t.Fatalf("update should succeed")
	}

	// state should be updated
	var updatedState token.Model
	diags := resp.State.Get(tc.Ctx, &updatedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}
	if updatedState.Name.ValueString() != nameUpdated {
		t.Fatalf("should be equal")
	}
	if updatedState.Description.ValueString() != descriptionUpdated {
		t.Fatalf("should be equal")
	}
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
	if resp.Diagnostics.HasError() {
		t.Fatalf("update should succeed")
	}

	// state should be removed
	var updatedState *token.Model
	diags := resp.State.Get(tc.Ctx, &updatedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}
	if updatedState != nil {
		t.Fatalf("state should be nil")
	}
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
	if !resp.Diagnostics.HasError() {
		t.Fatalf("update should not succeed")
	}

	// state should not be changed
	var updatedState token.Model
	diags := resp.State.Get(tc.Ctx, &updatedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}

	if updatedState.Name.ValueString() != name {
		t.Fatalf("should be equal")
	}
	if updatedState.Description.ValueString() != description {
		t.Fatalf("should be equal")
	}
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

	updateTokenResp := &modelexperiments.PartialUpdateInstanceTokenResponse{
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("update should succeed")
	}

	// state should not be removed
	var updatedState *token.Model
	diags := resp.State.Get(tc.Ctx, &updatedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}
	if updatedState != nil {
		t.Fatalf("state should be nil")
	}
}
