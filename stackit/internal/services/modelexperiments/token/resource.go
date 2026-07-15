package token

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	modelexperimentsutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &tokenResource{}
	_ resource.ResourceWithConfigure   = &tokenResource{}
	_ resource.ResourceWithImportState = &tokenResource{}
	_ resource.ResourceWithModifyPlan  = &tokenResource{}
)

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	Region      types.String `tfsdk:"region"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	InstanceId  types.String `tfsdk:"instance_id"`
	TokenId     types.String `tfsdk:"token_id"`
	Labels      types.Map    `tfsdk:"labels"`
	ValidUntil  types.String `tfsdk:"valid_until"`
	TTLDuration types.String `tfsdk:"ttl_duration"`
	Token       types.String `tfsdk:"token"`
	// RotateWhenChanged is a map of arbitrary key/value pairs that will force
	// recreation of the token when they change, enabling token rotation based on
	// external conditions such as a rotating timestamp. Changing this forces a new
	// resource to be created.
	RotateWhenChanged types.Map `tfsdk:"rotate_when_changed"`
}

// NewInstanceTokenResource is a helper function to simplify the provider implementation.
func NewInstanceTokenResourceEmpty() resource.Resource {
	return &tokenResource{}
}

func NewInstanceTokenResource(client modelexperiments.DefaultAPI, providerData core.ProviderData) resource.Resource { //nolint:gocritic
	return &tokenResource{
		client:       client,
		providerData: providerData,
	}
}

// tokenResource is the resource implementation.
type tokenResource struct {
	client       modelexperiments.DefaultAPI
	providerData core.ProviderData
}

var descriptions = map[string]string{ //nolint:gosec // no hardcoded credentials in here
	"main":            "Manages a STACKIT AI Model Experiments instance tokens.",
	"main_datasource": "Datasource scheme for a STACKIT AI Model Experiments instance tokens.",
	"id":              "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`\".",
	"project_id":      "STACKIT Project ID to which the resource is associated.",
	"instance_id":     "The AI Model Experiments instance ID.",
	"region":          "The STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"labels":          "A map of arbitrary key/value pairs that can be attached to the resource",
	"description":     "The description is a longer text chosen by the user to provide more context for the resource.",
	"name":            "The display name is a short name chosen by the user to identify the resource.",
	"token_id":        "The AI Model Experiments instance token ID.",
	"token":           "The content of the AI Model Experiments instance token.",
	"state":           "The state of the AI Model Experiments instance token.",
	"valid_until":     "The time until the AI Model Experiments instance token is valid.",
	"ttl_duration":    "The TTL duration of the AI Model Experiments instance token. E.g. 5h30m40s,5h,5h30m,30m,30s",
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
	i.client = apiClient.DefaultAPI
	tflog.Info(ctx, "Model Experiments client configured")
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
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token_id": schema.StringAttribute{
				Description: descriptions["token_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 160),
				},
			},
			"token": schema.StringAttribute{
				Description: descriptions["token"],
				Computed:    true,
				Sensitive:   true,
			},
			"valid_until": schema.StringAttribute{
				Description: descriptions["valid_until"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ttl_duration": schema.StringAttribute{
				Description: descriptions["ttl_duration"],
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
					"recreation of the resource when they change, enabling resource rotation " +
					"based on external conditions such as a rotating timestamp. Changing " +
					"this forces a new resource to be created.",
				Optional:    true,
				Required:    false,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI Model Experiments instance token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createInstanceTokenResp, err := i.client.CreateInstanceToken(ctx, projectId, region, instanceId).CreateInstanceTokenPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating AI Model Experiments instance token",
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
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceId,
		"token_id":    tokenId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	_, err = wait.CreateInstanceTokenWaitHandler(ctx, i.client, region, projectId, instanceId, tokenId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI Model Experiments instance token", fmt.Sprintf("Waiting for instance to be active: %v", err))
		return
	}

	// Map response body to schema
	err = mapCreateResponse(ctx, &createInstanceTokenResp.Token, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI Model Experiments instance token", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model Experiments instance token created")
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

	getInstanceTokenResp, err := i.client.GetInstanceToken(ctx, projectId, region, tokenId, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				// Remove the resource from the state so Terraform will recreate it
				resp.State.RemoveResource(ctx)
				return
			}
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading AI Model Experiments instance token", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapToken(ctx, &getInstanceTokenResp.Token, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI Model Experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model Experiments instance token read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (i *tokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var plan Model
	diags := req.Plan.Get(ctx, &plan)
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
	instanceId := plan.InstanceId.ValueString()
	tokenId := state.TokenId.ValueString()
	region := i.providerData.GetRegionWithOverride(plan.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(&plan)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI Model Experiments instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateInstanceTokenResp, err := i.client.PartialUpdateInstanceToken(ctx, projectId, region, tokenId, instanceId).PartialUpdateInstanceTokenPayload(*payload).Execute()
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
			"Error updating AI Model Experiments instance token",
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

	plan.Token = state.Token
	err = mapToken(ctx, &updateInstanceTokenResp.Token, &plan, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI Model Experiments instance token", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model Experiments instance token updated")
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

	_, err := i.client.DeleteInstanceToken(ctx, projectId, region, tokenId, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI Model Experiments instance token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteInstanceTokenWaitHandler(ctx, i.client, region, projectId, instanceId, tokenId).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI Model Experiments instance token", fmt.Sprintf("Waiting for instance to be deleted: %v", err))
		return
	}

	tflog.Info(ctx, "Model Experiments instance token deleted")
}

func (r *tokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing Model Experiments instance token",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[instance_id],[token_id], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"region":      idParts[1],
		"instance_id": idParts[2],
		"token_id":    idParts[3],
	})

	tflog.Info(ctx, "Model Experiments instance state imported")
}

// mapCreateResponse maps the instace creation response and GET instance response to the model
func mapCreateResponse(ctx context.Context, token *modelexperiments.Token, model *Model, region string) error {
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
func mapToken(ctx context.Context, token *modelexperiments.TokenMetadata, model *Model, region string) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if token.Id == "" {
		return fmt.Errorf("token id not present")
	}

	mapValue, err := utils.MapLabels(ctx, token.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, token.Id)
	model.TokenId = types.StringValue(token.Id)
	model.Name = types.StringValue(token.Name)
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
		Name:        conversion.StringValueToPointer(model.Name),
		Description: conversion.StringValueToPointer(model.Description),
		Labels:      labels,
	}, nil
}
