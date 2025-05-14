package token

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	modelservingUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelserving/utils"
	serviceenablementUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceenablement/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/modelserving"
	"github.com/stackitcloud/stackit-sdk-go/services/modelserving/wait"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceenablement"
	serviceEnablementWait "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource               = &tokenResource{}
	_ resource.ResourceWithConfigure  = &tokenResource{}
	_ resource.ResourceWithModifyPlan = &tokenResource{}
)

const (
	inactiveState = "inactive"
)

//go:embed description.md
var markdownDescription string

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	Region      types.String `tfsdk:"region"`
	TokenId     types.String `tfsdk:"token_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
	ValidUntil  types.String `tfsdk:"valid_until"`
	TTLDuration types.String `tfsdk:"ttl_duration"`
	Token       types.String `tfsdk:"token"`
	// RotateWhenChanged is a map of arbitrary key/value pairs that will force
	// recreation of the token when they change, enabling token rotation based on
	// external conditions such as a rotating timestamp. Changing this forces a new
	// resource to be created.
	RotateWhenChanged types.Map `tfsdk:"rotate_when_changed"`
}

// NewTokenResource is a helper function to simplify the provider implementation.
func NewTokenResource() resource.Resource {
	return &tokenResource{}
}

// tokenResource is the resource implementation.
type tokenResource struct {
	client                  *modelserving.APIClient
	providerData            core.ProviderData
	serviceEnablementClient *serviceenablement.APIClient
}

// Metadata returns the resource type name.
func (r *tokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_modelserving_token"
}

// Configure adds the provider configured client to the resource.
func (r *tokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := modelservingUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	serviceEnablementClient := serviceenablementUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	r.serviceEnablementClient = serviceEnablementClient
	tflog.Info(ctx, "Model-Serving auth token client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *tokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model

	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.AdaptRegion(
		ctx,
		configModel.Region,
		&planModel.Region,
		r.providerData.GetRegion(),
		resp,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Schema defines the schema for the resource.
func (r *tokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: markdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`region`,`token_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the AI model serving auth token is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: "Region to which the AI model serving auth token is associated. If not defined, the provider region is used",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token_id": schema.StringAttribute{
				Description: "The AI model serving auth token ID.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"ttl_duration": schema.StringAttribute{
				Description: "The TTL duration of the AI model serving auth token. E.g. 5h30m40s,5h,5h30m,30m,30s",
				Required:    false,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.ValidDurationString(),
				},
			},
			"rotate_when_changed": schema.MapAttribute{
				Description: "A map of arbitrary key/value pairs that will force " +
					"recreation of the token when they change, enabling token rotation " +
					"based on external conditions such as a rotating timestamp. Changing " +
					"this forces a new resource to be created.",
				Optional:    true,
				Required:    false,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the AI model serving auth token.",
				Required:    false,
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 2000),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the AI model serving auth token.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"state": schema.StringAttribute{
				Description: "State of the AI model serving auth token.",
				Computed:    true,
			},
			"token": schema.StringAttribute{
				Description: "Content of the AI model serving auth token.",
				Computed:    true,
				Sensitive:   true,
			},
			"valid_until": schema.StringAttribute{
				Description: "The time until the AI model serving auth token is valid.",
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

	var region string
	if utils.IsUndefined(model.Region) {
		region = r.providerData.GetRegion()
	} else {
		region = model.Region.ValueString()
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// If AI model serving is not enabled, enable it
	err := r.serviceEnablementClient.EnableServiceRegional(ctx, region, projectId, utils.ModelServingServiceId).
		Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error enabling AI model serving",
					fmt.Sprintf("Service not available in region %s \n%v", region, err),
				)
				return
			}
		}
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error enabling AI model serving",
			fmt.Sprintf("Error enabling AI model serving: %v", err),
		)
		return
	}

	_, err = serviceEnablementWait.EnableServiceWaitHandler(ctx, r.serviceEnablementClient, region, projectId, utils.ModelServingServiceId).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error enabling AI model serving",
			fmt.Sprintf("Error enabling AI model serving: %v", err),
		)
		return
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model serving auth token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new AI model serving auth token
	createTokenResp, err := r.client.CreateToken(ctx, region, projectId).
		CreateTokenPayload(*payload).
		Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating AI model serving auth token",
			fmt.Sprintf("Calling API: %v", err),
		)
		return
	}

	waitResp, err := wait.CreateModelServingWaitHandler(ctx, r.client, region, projectId, *createTokenResp.Token.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model serving auth token", fmt.Sprintf("Waiting for token to be active: %v", err))
		return
	}

	// Map response body to schema
	err = mapCreateResponse(createTokenResp, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model serving auth token", fmt.Sprintf("Processing API payload: %v", err))
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

	var region string
	if utils.IsUndefined(model.Region) {
		region = r.providerData.GetRegion()
	} else {
		region = model.Region.ValueString()
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	getTokenResp, err := r.client.GetToken(ctx, region, projectId, tokenId).
		Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				// Remove the resource from the state so Terraform will recreate it
				resp.State.RemoveResource(ctx)
				return
			}
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading AI model serving auth token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if getTokenResp != nil && getTokenResp.Token.State != nil &&
		*getTokenResp.Token.State == inactiveState {
		resp.State.RemoveResource(ctx)
		core.LogAndAddWarning(ctx, &resp.Diagnostics, "Error reading AI model serving auth token", "AI model serving auth token has expired")
		return
	}

	// Map response body to schema
	err = mapGetResponse(getTokenResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading AI model serving auth token", fmt.Sprintf("Processing API payload: %v", err))
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

	// Get current state
	var state Model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := state.ProjectId.ValueString()
	tokenId := state.TokenId.ValueString()

	var region string
	if utils.IsUndefined(model.Region) {
		region = r.providerData.GetRegion()
	} else {
		region = model.Region.ValueString()
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI model serving auth token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Update AI model serving auth token
	updateTokenResp, err := r.client.PartialUpdateToken(ctx, region, projectId, tokenId).PartialUpdateTokenPayload(*payload).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				// Remove the resource from the state so Terraform will recreate it
				resp.State.RemoveResource(ctx)
				return
			}
		}

		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error updating AI model serving auth token",
			fmt.Sprintf(
				"Calling API: %v, tokenId: %s, region: %s, projectId: %s",
				err,
				tokenId,
				region,
				projectId,
			),
		)
		return
	}

	if updateTokenResp != nil && updateTokenResp.Token.State != nil &&
		*updateTokenResp.Token.State == inactiveState {
		resp.State.RemoveResource(ctx)
		core.LogAndAddWarning(ctx, &resp.Diagnostics, "Error updating AI model serving auth token", "AI model serving auth token has expired")
		return
	}

	waitResp, err := wait.UpdateModelServingWaitHandler(ctx, r.client, region, projectId, tokenId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI model serving auth token", fmt.Sprintf("Waiting for token to be updated: %v", err))
		return
	}

	// Since STACKIT is not saving the content of the token. We have to use it from the state.
	model.Token = state.Token
	err = mapGetResponse(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI model serving auth token", fmt.Sprintf("Processing API payload: %v", err))
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

	var region string
	if utils.IsUndefined(model.Region) {
		region = r.providerData.GetRegion()
	} else {
		region = model.Region.ValueString()
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing AI model serving auth token. We will ignore the state 'deleting' for now.
	_, err := r.client.DeleteToken(ctx, region, projectId, tokenId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI model serving auth token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.DeleteModelServingWaitHandler(ctx, r.client, region, projectId, tokenId).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI model serving auth token", fmt.Sprintf("Waiting for token to be deleted: %v", err))
		return
	}

	tflog.Info(ctx, "Model-Serving auth token deleted")
}

func mapCreateResponse(tokenCreateResp *modelserving.CreateTokenResponse, waitResp *modelserving.GetTokenResponse, model *Model, region string) error {
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

	validUntil := types.StringNull()
	if token.ValidUntil != nil {
		validUntil = types.StringValue(token.ValidUntil.Format(time.RFC3339))
	}

	if waitResp == nil || waitResp.Token == nil || waitResp.Token.State == nil {
		return fmt.Errorf("response input is nil")
	}

	idParts := []string{model.ProjectId.ValueString(), region, *tokenCreateResp.Token.Id}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.TokenId = types.StringPointerValue(token.Id)
	model.Name = types.StringPointerValue(token.Name)
	model.State = types.StringPointerValue(waitResp.Token.State)
	model.ValidUntil = validUntil
	model.Token = types.StringPointerValue(token.Content)
	model.Description = types.StringPointerValue(token.Description)

	return nil
}

func mapGetResponse(tokenGetResp *modelserving.GetTokenResponse, model *Model) error {
	if tokenGetResp == nil {
		return fmt.Errorf("response input is nil")
	}

	if tokenGetResp.Token == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	// theoretically, should never happen, but still catch null pointers
	validUntil := types.StringNull()
	if tokenGetResp.Token.ValidUntil != nil {
		validUntil = types.StringValue(tokenGetResp.Token.ValidUntil.Format(time.RFC3339))
	}

	idParts := []string{model.ProjectId.ValueString(), model.Region.ValueString(), model.TokenId.ValueString()}
	model.Id = types.StringValue(strings.Join(idParts, core.Separator))
	model.TokenId = types.StringPointerValue(tokenGetResp.Token.Id)
	model.Name = types.StringPointerValue(tokenGetResp.Token.Name)
	model.State = types.StringPointerValue(tokenGetResp.Token.State)
	model.ValidUntil = validUntil
	model.Description = types.StringPointerValue(tokenGetResp.Token.Description)

	return nil
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

func toUpdatePayload(model *Model) (*modelserving.PartialUpdateTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &modelserving.PartialUpdateTokenPayload{
		Name:        conversion.StringValueToPointer(model.Name),
		Description: conversion.StringValueToPointer(model.Description),
	}, nil
}
