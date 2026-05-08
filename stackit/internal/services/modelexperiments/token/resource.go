package token

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	modelexperiments "dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/modelexperiments/v1api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/wait"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	modelexperimentsutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource               = &tokenResource{}
	_ resource.ResourceWithConfigure  = &tokenResource{}
	_ resource.ResourceWithModifyPlan = &tokenResource{}
)

var markdownDescription string

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	Region      types.String `tfsdk:"region"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	InstanceId  types.String `tfsdk:"instance_id"`
	TokenId     types.String `tfsdk:"token_id"`
	Labels      types.Map    `tfsdk:"labels"`
	State       types.String `tfsdk:"state"`
	ValidUntil  types.String `tfsdk:"valid_until"`
	TTLDuration types.String `tfsdk:"ttl_duration"`
	Token       types.String `tfsdk:"token"`
}

// NewInstanceTokenResource is a helper function to simplify the provider implementation.
func NewInstanceTokenResource() resource.Resource {
	return &tokenResource{}
}

// tokenResource is the resource implementation.
type tokenResource struct {
	client                  *modelexperiments.APIClient
	providerData            core.ProviderData
	serviceEnablementClient *serviceenablement.APIClient
}

// Metadata returns the resource type name.
func (i *tokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_modelexperiments_token"
}

// Configure adds the provider configured client to the resource.
func (i *tokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	i.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := modelexperimentsutils.ConfigureClient(ctx, &i.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	serviceEnablementClient := modelexperimentsutils.ConfigureServiceEnablementClient(ctx, &i.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	i.client = apiClient
	i.serviceEnablementClient = serviceEnablementClient
	tflog.Info(ctx, "Model experiments client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (i *tokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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
		i.providerData.GetRegion(),
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
func (i *tokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: markdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`region`,`token_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the AI model experiments instance token is associated.",
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
				Description: "Region to which the AI model experiments instance token is associated. If not defined, the provider region is used",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the AI model experiments instance token.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "The AI model experiments instance ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"token_id": schema.StringAttribute{
				Description: "The AI model experiments instance token ID.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"labels": schema.MapAttribute{
				Description: "A map of arbitrary key/value pairs for the AI model experiments instance token.",
				Optional:    true,
				Required:    false,
				Computed:    true,
				ElementType: types.StringType,
			},
			"description": schema.StringAttribute{
				Description: "The description of the AI model experiments instance token.",
				Required:    false,
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 160),
				},
			},
			"state": schema.StringAttribute{
				Description: "State of the AI model experiments instance token.",
				Computed:    true,
			},
			"token": schema.StringAttribute{
				Description: "Content of the AI model experiments instance token.",
				Computed:    true,
				Sensitive:   true,
			},
			"valid_until": schema.StringAttribute{
				Description: "The time until the AI model experiments instance token is valid.",
				Computed:    true,
			},
			"ttl_duration": schema.StringAttribute{
				Description: "The TTL duration of the AI model experiments instance token. E.g. 5h30m40s,5h,5h30m,30m,30s",
				Required:    false,
				Optional:    true,
				Validators: []validator.String{
					validate.ValidDurationString(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (i *tokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := i.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model experiments instance token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createInstanceTokenResp, err := i.client.DefaultAPI.CreateInstanceToken(ctx, projectId, region, instanceId).CreateInstanceTokenPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating AI model experiments instance token",
			fmt.Sprintf("Calling API: %v", err),
		)
		return
	}
	ctx = core.LogResponse(ctx)

	if createInstanceTokenResp.Token.Id == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance token", "Got empty token id")
		return
	}

	tokenId := createInstanceTokenResp.Token.Id
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceId,
		"token_id":    tokenId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := CreateMExpTokenWaitHandler(ctx, i.client, region, projectId, instanceId, tokenId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model experiments instance token", fmt.Sprintf("Waiting for instance to be active: %v", err))

		err = mapCreateResponse(ctx, createInstanceTokenResp, waitResp, &model, region)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model experiments instance token", fmt.Sprintf("Processing API payload: %v", err))
		}
		diags = resp.State.Set(ctx, model)
		resp.Diagnostics.Append(diags...)

		return
	}

	// Map response body to schema
	err = mapCreateResponse(ctx, createInstanceTokenResp, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model experiments instance token", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model experiments instance token created")
}

// Read refreshes the Terraform state with the latest data.
func (i *tokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	tokenId := model.TokenId.ValueString()
	if tokenId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}

	region := i.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	getInstanceTokenResp, err := i.client.DefaultAPI.GetInstanceToken(ctx, projectId, region, tokenId, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				// Remove the resource from the state so Terraform will recreate it
				resp.State.RemoveResource(ctx)
				return
			}
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading AI model experiments instance token", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapToken(ctx, getInstanceTokenResp.Token, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI model experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model experiments instance token read")

}

// Update updates the resource and sets the updated Terraform state on success.
func (i *tokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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

	ctx = core.InitProviderContext(ctx)

	projectId := state.ProjectId.ValueString()
	instanceId := state.InstanceId.ValueString()
	tokenId := state.TokenId.ValueString()
	region := i.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI model experiments instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateInstanceTokenResp, err := i.client.DefaultAPI.PartialUpdateInstanceToken(ctx, projectId, region, tokenId, instanceId).PartialUpdateInstanceTokenPayload(*payload).Execute()
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
			"Error updating AI model experiments instance token",
			fmt.Sprintf(
				"Calling API: %v, tokenId: %s, instanceId: %s, region: %s, projectId: %s",
				err,
				tokenId,
				instanceId,
				region,
				projectId,
			),
		)
		return
	}

	ctx = core.LogResponse(ctx)

	if updateInstanceTokenResp != nil && updateInstanceTokenResp.Token.State == modelexperiments.TOKENSTATE_INACTIVE {
		resp.State.RemoveResource(ctx)
		core.LogAndAddWarning(ctx, &resp.Diagnostics, "Error updating AI model experiments instance token", "AI model experiments token has expired")
		return
	}

	model.Token = state.Token
	err = mapToken(ctx, updateInstanceTokenResp.Token, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI model experiments instance token", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model experiments instance token updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (i *tokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	tokenId := model.TokenId.ValueString()

	region := i.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	_, err := i.client.DefaultAPI.DeleteInstanceToken(ctx, projectId, region, tokenId, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
			if oapiErr.StatusCode != http.StatusConflict {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI model experiments instance token", fmt.Sprintf("Calling API: %v", err))
				return
			}
		} else {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI model experiments instance token", fmt.Sprintf("Calling API: %v", err))
			return
		}
	}

	ctx = core.LogResponse(ctx)

	_, err = DeleteMExpTokenWaitHandler(ctx, i.client, region, projectId, instanceId, tokenId).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI model experiments instance token", fmt.Sprintf("Waiting for instance to be deleted: %v", err))
		return
	}

	tflog.Info(ctx, "Model experiments instance token deleted")
}

// mapCreateResponse maps the instace creation response and GET instance response to the model
func mapCreateResponse(ctx context.Context, instanceTokenResp *modelexperiments.CreateTokenResponse, waitResp *modelexperiments.GetTokenResponse, model *Model, region string) error {
	if instanceTokenResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	token := instanceTokenResp.Token

	if token.Id == "" {
		return fmt.Errorf("token id not present")
	}

	if waitResp == nil {
		model.State = types.StringValue("unknown")
	} else {
		model.State = types.StringValue(string(waitResp.Token.State))
	}

	mapValue, diags := types.MapValueFrom(ctx, types.StringType, token.Labels)
	if diags.HasError() {
		return fmt.Errorf("failure in mapping labels")
	}

	validUntil := types.StringNull()
	if !token.ValidUntil.IsZero() {
		validUntil = types.StringValue(token.ValidUntil.Format(time.RFC3339))
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, token.Id)
	model.TokenId = types.StringValue(token.Id)
	model.Name = types.StringValue(token.Name)
	model.Description = types.StringPointerValue(token.Description)
	model.ValidUntil = validUntil
	model.Token = types.StringValue(token.Content)
	model.Labels = mapValue

	return nil
}

// mapToken maps instances to the resource model
func mapToken(ctx context.Context, token modelexperiments.TokenMetadata, model *Model) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if token.Id == "" {
		return fmt.Errorf("token id not present")
	}

	mapValue, diags := types.MapValueFrom(ctx, types.StringType, token.Labels)
	if diags.HasError() {
		return fmt.Errorf("failure in mapping labels")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.Region.ValueString(), model.TokenId.ValueString())
	model.TokenId = types.StringValue(token.Id)
	model.Name = types.StringValue(token.Name)
	model.State = types.StringValue(string(token.State))
	model.Description = types.StringPointerValue(token.Description)
	model.ValidUntil = types.StringValue(token.ValidUntil.Format(time.RFC3339))
	model.Labels = mapValue

	return nil
}

func toCreatePayload(model *Model) (*modelexperiments.CreateInstanceTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &modelexperiments.CreateInstanceTokenPayload{
		Name:        model.Name.ValueString(),
		Description: conversion.StringValueToPointer(model.Description),
		TtlDuration: conversion.StringValueToPointer(model.TTLDuration),
		Labels:      labels,
	}, nil
}

func toUpdatePayload(model *Model) (*modelexperiments.PartialUpdateInstanceTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}
	return &modelexperiments.PartialUpdateInstanceTokenPayload{
		Name:        model.Name.ValueStringPointer(),
		Description: model.Description.ValueStringPointer(),
		Labels:      labels,
	}, nil
}

func CreateMExpTokenWaitHandler(ctx context.Context, a *modelexperiments.APIClient, region, projectId, instanceId string, tokenId string) *wait.AsyncActionHandler[modelexperiments.GetTokenResponse] {
	handler := wait.New(func() (waitFinished bool, response *modelexperiments.GetTokenResponse, err error) {
		getTokenResp, err := a.DefaultAPI.GetInstanceToken(ctx, projectId, region, tokenId, instanceId).Execute()
		if err != nil {
			return false, nil, err
		}
		if getTokenResp.Token.State == modelexperimentsutils.TOKENSTATE_ACTIVE {
			return true, getTokenResp, nil
		}

		return false, nil, nil
	})

	handler.SetTimeout(10 * time.Minute)

	return handler
}

func DeleteMExpTokenWaitHandler(ctx context.Context, a *modelexperiments.APIClient, region, projectId, instanceId string, tokenId string) *wait.AsyncActionHandler[modelexperiments.GetInstanceResponse] {
	handler := wait.New(
		func() (waitFinished bool, response *modelexperiments.GetInstanceResponse, err error) {
			_, err = a.DefaultAPI.GetInstanceToken(ctx, projectId, region, tokenId, instanceId).Execute()
			if err != nil {
				var oapiErr *oapierror.GenericOpenAPIError
				if errors.As(err, &oapiErr) {
					if oapiErr.StatusCode == http.StatusNotFound {
						return true, nil, nil
					}
				}

				return false, nil, err
			}

			return false, nil, nil
		},
	)

	handler.SetTimeout(10 * time.Minute)

	return handler
}
