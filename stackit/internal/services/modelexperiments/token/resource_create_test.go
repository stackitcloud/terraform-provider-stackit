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

// NOTE: These tests will be refactored.
// Please DO NOT use this file as a pattern or reference for writing new tests.
func TestCreate_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	name := "token"
	region := "eu01"
	description := "token description"
	instanceId := uuid.New()
	tokenId := uuid.New()
	validUntil := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	content := "token"
	tfId := utils.BuildInternalTerraformId(projectId.String(), region, instanceId.String(), tokenId.String())

	createTokenResp := &modelexperiments.CreateInstanceTokenResponse{
		Token: modelexperiments.Token{
			Content:     content,
			Description: &description,
			Id:          tokenId.String(),
			Name:        name,
			Region:      region,
			State:       "creating",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().CreateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceTokenRequest{
		ApiService: tc.MockInstanceCLient,
	})
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
		ProjectId:         types.StringValue(projectId.String()),
		Name:              types.StringValue(name),
		Region:            types.StringValue(region),
		Description:       types.StringValue(description),
		InstanceId:        types.StringValue(instanceId.String()),
		Labels:            types.MapNull(types.StringType),
		RotateWhenChanged: types.MapNull(types.StringType),
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

	if createdState.ProjectId.ValueString() != projectId.String() {
		t.Fatalf("ProjectId mismatch: got %v, want %v", createdState.ProjectId.ValueString(), projectId.String())
	}
	if createdState.Region.ValueString() != region {
		t.Fatalf("Region mismatch: got %v, want %v", createdState.Region.ValueString(), region)
	}
	if createdState.Name.ValueString() != name {
		t.Fatalf("Name mismatch: got %v, want %v", createdState.Name.ValueString(), name)
	}
	if createdState.Description.ValueString() != description {
		t.Fatalf("Description mismatch: got %v, want %v", createdState.Description.ValueString(), description)
	}
	if createdState.InstanceId.ValueString() != instanceId.String() {
		t.Fatalf("InstanceId mismatch: got %v, want %v", createdState.InstanceId.ValueString(), instanceId.String())
	}
	if createdState.TokenId.ValueString() != tokenId.String() {
		t.Fatalf("TokenId mismatch: got %v, want %v", createdState.TokenId.ValueString(), tokenId.String())
	}
	if createdState.Id != tfId {
		t.Fatalf("Id mismatch: got %v, want %v", createdState.Id.ValueString(), tfId)
	}
	if createdState.ValidUntil.ValueString() != "2099-01-01T00:00:00Z" {
		t.Fatalf("ValidUntil mismatch: got %v, want 2099-01-01T00:00:00Z", createdState.ValidUntil.ValueString())
	}
	if !createdState.Labels.IsNull() {
		t.Fatalf("Labels should be null")
	}
	if createdState.Token.ValueString() != content {
		t.Fatalf("Token mismatch: got %v, want %v", createdState.Token.ValueString(), content)
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
	content := "token"

	createTokenResp := &modelexperiments.CreateInstanceTokenResponse{
		Token: modelexperiments.Token{
			Content:     content,
			Description: &description,
			Id:          "",
			Name:        name,
			Region:      region,
			State:       "creating",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().CreateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceTokenRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().CreateInstanceTokenExecute(gomock.Any()).Return(createTokenResp, nil)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := token.Model{
		ProjectId:         types.StringValue(projectId.String()),
		Name:              types.StringValue(name),
		Region:            types.StringValue(region),
		Description:       types.StringValue(description),
		InstanceId:        types.StringValue(instanceId.String()),
		Labels:            types.MapNull(types.StringType),
		RotateWhenChanged: types.MapNull(types.StringType),
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
	tc.MockInstanceCLient.EXPECT().CreateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceTokenRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().CreateInstanceTokenExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	tokenRes := token.NewInstanceTokenResource(tc.MockInstanceCLient, providerData)

	schemaResp := resource.SchemaResponse{}
	tokenRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := token.Model{
		ProjectId:         types.StringValue(projectId.String()),
		Name:              types.StringValue(name),
		Region:            types.StringValue(region),
		Description:       types.StringValue(description),
		InstanceId:        types.StringValue(instanceId.String()),
		Labels:            types.MapNull(types.StringType),
		RotateWhenChanged: types.MapNull(types.StringType),
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
	validUntil := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	content := "token"

	createTokenResp := &modelexperiments.CreateInstanceTokenResponse{
		Token: modelexperiments.Token{
			Content:     content,
			Description: &description,
			Id:          tokenId.String(),
			Name:        name,
			Region:      region,
			State:       "creating",
			ValidUntil:  validUntil,
		},
	}
	tc.MockInstanceCLient.EXPECT().CreateInstanceToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceTokenRequest{
		ApiService: tc.MockInstanceCLient,
	})
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
		ProjectId:         types.StringValue(projectId.String()),
		Name:              types.StringValue(name),
		Region:            types.StringValue(region),
		Description:       types.StringValue(description),
		InstanceId:        types.StringValue(instanceId.String()),
		Labels:            types.MapNull(types.StringType),
		RotateWhenChanged: types.MapNull(types.StringType),
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

	if createdState.ProjectId.ValueString() != projectId.String() {
		t.Fatalf("ProjectId mismatch: got %v, want %v", createdState.ProjectId.ValueString(), projectId.String())
	}
	if createdState.Region.ValueString() != region {
		t.Fatalf("Region mismatch: got %v, want %v", createdState.Region.ValueString(), region)
	}
	if createdState.Name.ValueString() != "" {
		t.Fatalf("Name mismatch: got %v, want %v", createdState.Name.ValueString(), "")
	}
	if createdState.Description.ValueString() != "" {
		t.Fatalf("Description mismatch: got %v, want %v", createdState.Description.ValueString(), "")
	}
	if createdState.InstanceId.ValueString() != instanceId.String() {
		t.Fatalf("InstanceId mismatch: got %v, want %v", createdState.InstanceId.ValueString(), instanceId.String())
	}
	if createdState.TokenId.ValueString() != tokenId.String() {
		t.Fatalf("TokenId mismatch: got %v, want %v", createdState.TokenId.ValueString(), tokenId.String())
	}
	if createdState.Id.ValueString() != "" {
		t.Fatalf("Id mismatch: got %v, want %v", createdState.Id.ValueString(), "")
	}
	if createdState.ValidUntil.ValueString() != "" {
		t.Fatalf("ValidUntil mismatch: got %v, want %v", createdState.ValidUntil.ValueString(), "")
	}
	if !createdState.Labels.IsNull() {
		t.Fatalf("Labels should be null")
	}
	if createdState.Token.ValueString() != "" {
		t.Fatalf("Token mismatch: got %v, want %v", createdState.Token.ValueString(), "")
	}
}
