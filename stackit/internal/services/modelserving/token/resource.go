package token

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
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
	client *dns.APIClient
}

// Metadata returns the resource type name.
func (r *tokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_serving_token"
}

// Configure adds the provider configured client to the resource.
func (r *tokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	// TODO: Add correct client
	var apiClient *dns.APIClient
	var err error
	if providerData.DnsCustomEndpoint != "" {
		apiClient, err = dns.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.DnsCustomEndpoint),
		)
	} else {
		apiClient, err = dns.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "DNS record set client configured")
}

// Schema defines the schema for the resource.
func (r *tokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Required:    true,
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
			"validUntil": schema.StringAttribute{
				Description: "The time until the model serving auth token is valid.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *tokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating model serving auth token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new model serving auth token
	// TODO: Add correct client
	println(payload)
	// recordSetResp, err := r.client.CreateRecordSet(ctx, projectId, zoneId).CreateRecordSetPayload(*payload).Execute()
	// if err != nil || recordSetResp.Rrset == nil || recordSetResp.Rrset.Id == nil {
	// 	core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating record set", fmt.Sprintf("Calling API: %v", err))
	// 	return
	// }
	// ctx = tflog.SetField(ctx, "record_set_id", *recordSetResp.Rrset.Id)
	//
	// waitResp, err := wait.CreateRecordSetWaitHandler(ctx, r.client, projectId, zoneId, *recordSetResp.Rrset.Id).WaitWithContext(ctx)
	// if err != nil {
	// 	core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating record set", fmt.Sprintf("Instance creation waiting: %v", err))
	// 	return
	// }

	// Map response body to schema
	waitResp := &CreateTokenResponse{}
	err = mapCreateResponse(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating model serving auth token", fmt.Sprintf("Processing API payload: %v", err))
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
func (r *tokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	tokenId := model.TokenId.ValueString()
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	// TODO: Add correct client
	// recordSetResp, err := r.client.GetRecordSet(ctx, projectId, zoneId, recordSetId).Execute()
	// if err != nil {
	// 	core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading record set", fmt.Sprintf("Calling API: %v", err))
	// 	return
	// }
	// if recordSetResp != nil && recordSetResp.Rrset.State != nil && *recordSetResp.Rrset.State == wait.DeleteSuccess {
	// 	resp.State.RemoveResource(ctx)
	// 	return
	// }

	// Map response body to schema
	getTokenResp := &GetTokenResponse{}
	err := mapGetResponse(getTokenResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading model serving auth token", fmt.Sprintf("Processing API payload: %v", err))
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
func (r *tokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating model serving auth token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Update model serving auth token
	// TODO: Add correct client
	println(payload)
	// _, err = r.client.PartialUpdateRecordSet(ctx, projectId, zoneId, recordSetId).PartialUpdateRecordSetPayload(*payload).Execute()
	// if err != nil {
	// 	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating record set", err.Error())
	// 	return
	// }
	// waitResp, err := wait.PartialUpdateRecordSetWaitHandler(ctx, r.client, projectId, zoneId, recordSetId).WaitWithContext(ctx)
	// if err != nil {
	// 	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating record set", fmt.Sprintf("Instance update waiting: %v", err))
	// 	return
	// }

	waitResp := &UpdateTokenResponse{}
	err = mapUpdateResponse(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating model serving auth token", fmt.Sprintf("Processing API payload: %v", err))
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
func (r *tokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing model serving auth token
	// TODO: Add correct client
	// _, err := r.client.DeleteRecordSet(ctx, projectId, zoneId, recordSetId).Execute()
	// if err != nil {
	// 	core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting record set", fmt.Sprintf("Calling API: %v", err))
	// }
	// _, err = wait.DeleteRecordSetWaitHandler(ctx, r.client, projectId, zoneId, recordSetId).WaitWithContext(ctx)
	// if err != nil {
	// 	core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting record set", fmt.Sprintf("Instance deletion waiting: %v", err))
	// 	return
	// }

	tflog.Info(ctx, "Model-Serving auth token deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,zone_id,record_set_id
func (r *tokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing model serving auth token",
			fmt.Sprintf("Expected import identifier with format [project_id],[token_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("token_id"), idParts[1])...)

	tflog.Info(ctx, "Model-Serving auth token state imported")
}

type CreateTokenResponse struct {
	Token *TokenCreated `json:"token"`
}

type TokenCreated struct {
	ID          *string `json:"id"`
	Content     *string `json:"content"`
	State       *string `json:"state"`
	ValidUntil  *string `json:"validUntil"`
	Name        *string `json:"name"`
	Region      *string `json:"region"`
	Description *string `json:"description"`
}

func mapCreateResponse(tokenCreateResp *CreateTokenResponse, model *Model) error {
	if tokenCreateResp == nil || tokenCreateResp.Token == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	token := tokenCreateResp.Token

	if token.ID == nil {
		return fmt.Errorf("token id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		*tokenCreateResp.Token.ID,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.TokenId = types.StringPointerValue(token.ID)
	model.Name = types.StringPointerValue(token.Name)
	model.Region = types.StringPointerValue(token.Region)
	model.State = types.StringPointerValue(token.State)
	model.ValidUntil = types.StringPointerValue(token.ValidUntil)
	model.Content = types.StringPointerValue(token.Content)
	model.Description = types.StringPointerValue(token.Description)

	return nil
}

type UpdateTokenResponse struct {
	Token *Token
}

type Token struct {
	ID          *string `json:"id"`
	ValidUntil  *string `json:"validUntil"`
	State       *string `json:"state"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Region      *string `json:"region"`
}

func mapUpdateResponse(tokenUpdateResp *UpdateTokenResponse, model *Model) error {
	if tokenUpdateResp == nil {
		return fmt.Errorf("response input is nil")
	}

	return mapToken(tokenUpdateResp.Token, model)
}

type GetTokenResponse struct {
	Token *Token
}

func mapToken(token *Token, model *Model) error {
	if token == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		model.TokenId.ValueString(),
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.TokenId = types.StringPointerValue(token.ID)
	model.Name = types.StringPointerValue(token.Name)
	model.Region = types.StringPointerValue(token.Region)
	model.State = types.StringPointerValue(token.State)
	model.ValidUntil = types.StringPointerValue(token.ValidUntil)
	model.Description = types.StringPointerValue(token.Description)

	return nil
}

func mapGetResponse(tokenGetResp *GetTokenResponse, model *Model) error {
	if tokenGetResp == nil {
		return fmt.Errorf("response input is nil")
	}

	return mapToken(tokenGetResp.Token, model)
}

type CreateTokenPayload struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	TTLDuration *string `json:"ttl_duration"`
}

func toCreatePayload(model *Model) (*CreateTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &CreateTokenPayload{
		Name:        conversion.StringValueToPointer(model.Name),
		Description: conversion.StringValueToPointer(model.Description),
		TTLDuration: conversion.StringValueToPointer(model.TTLDuration),
	}, nil
}

type UpdateTokenPayload struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func toUpdatePayload(model *Model) (*UpdateTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &UpdateTokenPayload{
		Name:        conversion.StringValueToPointer(model.Name),
		Description: conversion.StringValueToPointer(model.Description),
	}, nil
}
