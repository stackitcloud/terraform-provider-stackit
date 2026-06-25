package token_test

import (
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

func TestCreate_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	validUntil := time.Now().Add(10 * time.Minute)

	createTokenResp := &modelexperiments.CreateInstanceTokenResponse{
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

	getTokenResp := &modelexperiments.GetInstanceTokenResponse{
		Token: modelexperiments.TokenMetadata{
			Description: &description,
			Id:          tokenId.String(),
			Name:        name,
			Region:      region,
			State:       "active",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().GetInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceTokenRequest{
		ApiService: tc.MockInstanceCLient,
	})
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
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}

	var createdState token.Model
	diags := resp.State.Get(tc.Ctx, &createdState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}

	if tokenId.String() != createdState.TokenId.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", tokenId.String(), createdState.TokenId.ValueString())
	}
	if projectId.String() != createdState.ProjectId.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", projectId.String(), createdState.ProjectId.ValueString())
	}
	if instanceId.String() != createdState.InstanceId.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", instanceId.String(), createdState.InstanceId.ValueString())
	}
	if name != createdState.Name.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", name, createdState.Name.ValueString())
	}
	if createdState.State.ValueString() != "active" {
		t.Fatalf("Should be equal - expected %v, got %v", "active", createdState.State.ValueString())
	}
	if description != createdState.Description.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", description, createdState.Description.ValueString())
	}
	if createdState.Token.ValueString() != "token" {
		t.Fatalf("Should be equal - expected %v, got %v", "token", createdState.Token.ValueString())
	}
	if utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String()).ValueString() != createdState.Id.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String()).ValueString(), createdState.Id.ValueString())
	}
}

func TestCreate_TokenIdEmpty(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	validUntil := time.Now()

	createTokenResp := &modelexperiments.CreateInstanceTokenResponse{
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
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Create should not succeed but got no errors")
	}

	// state should not be created
	var createdState *token.Model
	diags := resp.State.Get(tc.Ctx, &createdState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}
	if createdState != nil {
		t.Fatalf("expected nil, got %v", createdState)
	}
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
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Create should not succeed but got no errors")
	}

	// state should not be created
	var createdState *token.Model
	diags := resp.State.Get(tc.Ctx, &createdState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}
	if createdState != nil {
		t.Fatalf("expected nil, got %v", createdState)
	}
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

	createTokenResp := &modelexperiments.CreateInstanceTokenResponse{
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
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Create should not succeed but got no errors")
	}

	// state should be created
	var createdState token.Model
	diags := resp.State.Get(tc.Ctx, &createdState)
	if diags.HasError() {
		t.Fatalf("failed to get state")
	}

	if tokenId.String() != createdState.TokenId.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", tokenId.String(), createdState.TokenId.ValueString())
	}
	if projectId.String() != createdState.ProjectId.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", projectId.String(), createdState.ProjectId.ValueString())
	}
	if instanceId.String() != createdState.InstanceId.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", instanceId.String(), createdState.InstanceId.ValueString())
	}
	if name != createdState.Name.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", name, createdState.Name.ValueString())
	}
	if createdState.State.ValueString() != "unknown" {
		t.Fatalf("Should be equal - expected %v, got %v", "unknown", createdState.State.ValueString())
	}
	if description != createdState.Description.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", description, createdState.Description.ValueString())
	}
	if createdState.Token.ValueString() != "token" {
		t.Fatalf("Should be equal - expected %v, got %v", "token", createdState.Token.ValueString())
	}
	if utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String()).ValueString() != createdState.Id.ValueString() {
		t.Fatalf("Should be equal - expected %v, got %v", utils.BuildInternalTerraformId(projectId.String(), region, tokenId.String()).ValueString(), createdState.Id.ValueString())
	}
}
