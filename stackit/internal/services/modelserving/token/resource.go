package token

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/services/modelserving"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/modelserving/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &tokenResource{}
	_ resource.ResourceWithConfigure   = &tokenResource{}
	_ resource.ResourceWithImportState = &tokenResource{}
)

const (
	inactiveState = "inactive"
	activeState   = "active"
)

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	Region      types.String `tfsdk:"region"`
	TokenId     types.String `tfsdk:"token_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
	ValidUntil  types.String `tfsdk:"validUntil"`
	TTLDuration types.String `tfsdk:"ttlDuration"`
	Content     types.String `tfsdk:"content"`
}

// NewTokenResource is a helper function to simplify the provider implementation.
func NewTokenResource() resource.Resource {
	return &tokenResource{}
}

// tokenResource is the resource implementation.
type tokenResource struct {
	client *modelserving.APIClient
}

// Metadata returns the resource type name.
func (r *tokenResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_modelserving_token"
}

// Configure adds the provider configured client to the resource.
func (r *tokenResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error configuring API client",
			fmt.Sprintf(
				"Expected configure type stackit.ProviderData, got %T",
				req.ProviderData,
			),
		)
		return
	}

	var apiClient *modelserving.APIClient
	var err error
	if providerData.ModelServingCustomEndpoint != "" {
		ctx = tflog.SetField(
			ctx,
			"modelserving_custom_endpoint",
			providerData.ModelServingCustomEndpoint,
		)
		apiClient, err = modelserving.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ModelServingCustomEndpoint),
			config.WithRegion(providerData.GetRegion()),
		)
	} else {
		apiClient, err = modelserving.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error configuring API client",
			fmt.Sprintf(
				"Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration",
				err,
			),
		)
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Model-Serving auth token client configured")
}

// Schema defines the schema for the resource.
func (r *tokenResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Model Serving Auth Token Resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`token_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the model serving auth token is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "STACKIT region to which the model serving auth token is associated.",
				Required:    false,
				Optional:    true,
			},
			"token_id": schema.StringAttribute{
				Description: "The model serving auth token ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the model serving auth token.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the model serving auth token.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "State of the model serving auth token.",
				Computed:    true,
			},
			"content": schema.StringAttribute{
				Description: "Content of the model serving auth token.",
				Computed:    true,
			},
			"valid_until": schema.StringAttribute{
				Description: "The time until the model serving auth token is valid.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *tokenResource) Create(
	ctx context.Context,
	req resource.CreateRequest, //nolint:gocritic // function signature required by Terraform
	resp *resource.CreateResponse,
) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	if region == "" {
		region = r.client.GetConfig().Region
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating model serving auth token",
			fmt.Sprintf("Creating API payload: %v", err),
		)
		return
	}

	// Create new model serving auth token
	createTokenResp, err := r.client.CreateToken(ctx, region, projectId).
		CreateTokenPayload(*payload).
		Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating model serving auth token",
			fmt.Sprintf("Calling API: %v", err),
		)
		return
	}

	waitResp, err := wait.CreateModelServingWaitHandler(ctx, r.client, region, projectId, *createTokenResp.Token.Id).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating model serving auth token",
			fmt.Sprintf("Waiting for token to be active: %v", err),
		)
		return
	}

	// Map response body to schema
	err = mapCreateResponse(createTokenResp, waitResp, &model)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating model serving auth token",
			fmt.Sprintf("Processing API payload: %v", err),
		)
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model-Serving auth token created")
}

// Read refreshes the Terraform state with the latest data.
func (r *tokenResource) Read(
	ctx context.Context,
	req resource.ReadRequest, //nolint:gocritic // function signature required by Terraform
	resp *resource.ReadResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	tokenId := model.TokenId.ValueString()
	region := model.Region.ValueString()
	if region == "" {
		region = r.client.GetConfig().Region
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	getTokenResp, err := r.client.GetToken(ctx, region, projectId, tokenId).
		Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading model serving auth token",
			fmt.Sprintf("Calling API: %v", err),
		)
		return
	}

	if getTokenResp != nil && getTokenResp.Token.State != nil &&
		*getTokenResp.Token.State == inactiveState {
		resp.State.RemoveResource(ctx)
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading model serving auth token",
			"Model serving auth token has expired",
		)
		return
	}

	// Map response body to schema
	err = mapGetResponse(getTokenResp, &model)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading model serving auth token",
			fmt.Sprintf("Processing API payload: %v", err),
		)
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model-Serving auth token read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *tokenResource) Update(
	ctx context.Context,
	req resource.UpdateRequest, //nolint:gocritic // function signature required by Terraform
	resp *resource.UpdateResponse,
) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	tokenId := model.TokenId.ValueString()
	region := model.Region.ValueString()
	if region == "" {
		region = r.client.GetConfig().Region
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error updating model serving auth token",
			fmt.Sprintf("Creating API payload: %v", err),
		)
		return
	}

	// Update model serving auth token
	updateTokenResp, err := r.client.PartialUpdateToken(ctx, region, projectId, tokenId).
		PartialUpdateTokenPayload(*payload).
		Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error updating model serving auth token",
			fmt.Sprintf("Calling API: %v", err),
		)
		return
	}

	if updateTokenResp != nil && updateTokenResp.Token.State != nil &&
		*updateTokenResp.Token.State == inactiveState {
		resp.State.RemoveResource(ctx)
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error updating model serving auth token",
			"Model serving auth token has expired",
		)
		return
	}

	waitResp, err := wait.UpdateModelServingWaitHandler(ctx, r.client, region, projectId, tokenId).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error updating model serving auth token",
			fmt.Sprintf("Waiting for token to be updated: %v", err),
		)
		return
	}

	err = mapGetResponse(waitResp, &model)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error updating model serving auth token",
			fmt.Sprintf("Processing API payload: %v", err),
		)
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model-Serving auth token updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *tokenResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest, //nolint:gocritic // function signature required by Terraform
	resp *resource.DeleteResponse,
) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	tokenId := model.TokenId.ValueString()
	region := model.Region.ValueString()
	if region == "" {
		region = r.client.GetConfig().Region
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing model serving auth token. We will ignore the state 'deleting' for now.
	_, err := r.client.DeleteToken(ctx, region, projectId, tokenId).Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error deleting model serving auth token",
			fmt.Sprintf("Calling API: %v", err),
		)
		return
	}

	_, err = wait.DeleteModelServingWaitHandler(ctx, r.client, region, projectId, tokenId).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error deleting model serving auth token",
			fmt.Sprintf("Waiting for token to be deleted: %v", err),
		)
		return
	}

	tflog.Info(ctx, "Model-Serving auth token deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,zone_id,record_set_id
func (r *tokenResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error importing model serving auth token",
			fmt.Sprintf(
				"Expected import identifier with format [project_id],[token_id], got %q",
				req.ID,
			),
		)
		return
	}

	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("token_id"), idParts[1])...)

	tflog.Info(ctx, "Model-Serving auth token state imported")
}

func mapCreateResponse(
	tokenCreateResp *modelserving.CreateTokenResponse,
	waitResp *modelserving.GetTokenResponse,
	model *Model,
) error {
	if tokenCreateResp == nil || tokenCreateResp.Token == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	token := tokenCreateResp.Token

	if token.Id == nil {
		return fmt.Errorf("token id not present")
	}

	validUntil := time.Now().Format(time.RFC3339)
	if token.ValidUntil != nil {
		validUntil = token.ValidUntil.Format(time.RFC3339)
	}

	if waitResp == nil || waitResp.Token == nil || waitResp.Token.State == nil {
		return fmt.Errorf("response input is nil")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		*tokenCreateResp.Token.Id,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.TokenId = types.StringPointerValue(token.Id)
	model.Name = types.StringPointerValue(token.Name)
	model.Region = types.StringPointerValue(token.Region)
	model.State = types.StringPointerValue(waitResp.Token.State)
	model.ValidUntil = types.StringValue(validUntil)
	model.Content = types.StringPointerValue(token.Content)
	model.Description = types.StringPointerValue(token.Description)

	return nil
}

func mapToken(token *modelserving.Token, model *Model) error {
	if token == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	// theoretically, should never happen, but still catch null pointers
	validUntil := time.Now().Format(time.RFC3339)
	if token.ValidUntil != nil {
		validUntil = token.ValidUntil.Format(time.RFC3339)
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		model.TokenId.ValueString(),
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.TokenId = types.StringPointerValue(token.Id)
	model.Name = types.StringPointerValue(token.Name)
	model.Region = types.StringPointerValue(token.Region)
	model.State = types.StringPointerValue(token.State)
	model.ValidUntil = types.StringValue(validUntil)
	model.Description = types.StringPointerValue(token.Description)

	return nil
}

func mapGetResponse(
	tokenGetResp *modelserving.GetTokenResponse,
	model *Model,
) error {
	if tokenGetResp == nil {
		return fmt.Errorf("response input is nil")
	}

	return mapToken(tokenGetResp.Token, model)
}

func toCreatePayload(model *Model) (*modelserving.CreateTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &modelserving.CreateTokenPayload{
		Name:        conversion.StringValueToPointer(model.Name),
		Description: conversion.StringValueToPointer(model.Description),
		TtlDuration: conversion.StringValueToPointer(model.TTLDuration),
	}, nil
}

func toUpdatePayload(
	model *Model,
) (*modelserving.PartialUpdateTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &modelserving.PartialUpdateTokenPayload{
		Name:        conversion.StringValueToPointer(model.Name),
		Description: conversion.StringValueToPointer(model.Description),
	}, nil
}
