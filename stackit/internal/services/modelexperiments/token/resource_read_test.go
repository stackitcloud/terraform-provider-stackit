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
	"go.uber.org/mock/gomock"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/token"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
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
	if resp.Diagnostics.HasError() {
		t.Fatalf("Get should succeed but got errors")
	}
	// state should be removed
	var refreshedState *token.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}
	if refreshedState != nil {
		t.Fatalf("should be nil")
	}
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
	if resp.Diagnostics.HasError() {
		t.Fatalf("Get should succeed but got errors")
	}

	// state should be removed
	var refreshedState *token.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}
	if refreshedState != nil {
		t.Fatalf("should be nil")
	}
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
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Get should not succeed")
	}

	// state should not be edited
	var refreshedState token.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}

	if tokenId.String() != refreshedState.TokenId.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", tokenId.String(), refreshedState.TokenId.ValueString())
	}
	if projectId.String() != refreshedState.ProjectId.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", projectId.String(), refreshedState.ProjectId.ValueString())
	}
	if instanceId.String() != refreshedState.InstanceId.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", instanceId.String(), refreshedState.InstanceId.ValueString())
	}
	if name != refreshedState.Name.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", name, refreshedState.Name.ValueString())
	}
	if refreshedState.State.ValueString() != "active" {
		t.Fatalf("Should be equal - expected %v, got %v", "active", refreshedState.State.ValueString())
	}
	if description != refreshedState.Description.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", description, refreshedState.Description.ValueString())
	}
	if tokenContent != refreshedState.Token.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", tokenContent, refreshedState.Token.ValueString())
	}
	if id.ValueString() != refreshedState.Id.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", id.ValueString(), refreshedState.Id.ValueString())
	}
	if region != refreshedState.Region.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", region, refreshedState.Region.ValueString())
	}
	if refreshedState.ValidUntil.ValueString() != "2099-01-01T00:00:00Z" {
		t.Fatalf("Should be equal - expected %v, got %v", "2099-01-01T00:00:00Z", refreshedState.ValidUntil.ValueString())
	}
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
	if resp.Diagnostics.HasError() {
		t.Fatalf("Get should succeed but got errors")
	}

	// state should be removed
	var refreshedState *token.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}
	if refreshedState != nil {
		t.Fatalf("should be nil")
	}
}
