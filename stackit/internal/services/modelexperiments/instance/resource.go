package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api/wait"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	serviceEnablementWait "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	modelexperimentsutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/utils"
	serviceEnablementUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceenablement/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
	_ resource.ResourceWithModifyPlan  = &instanceResource{}
)

type Model struct {
	Id                         types.String `tfsdk:"id"` // needed by TF
	ProjectId                  types.String `tfsdk:"project_id"`
	Region                     types.String `tfsdk:"region"`
	Name                       types.String `tfsdk:"name"`
	Description                types.String `tfsdk:"description"`
	DeletedExperimentRetention types.String `tfsdk:"deleted_experiment_retention"`
	Labels                     types.Map    `tfsdk:"labels"`
	BucketName                 types.String `tfsdk:"bucket_name"`
	InstanceId                 types.String `tfsdk:"instance_id"`
	Url                        types.String `tfsdk:"url"`
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResourceEmpty() resource.Resource {
	return &instanceResource{}
}

func NewInstanceResource(client modelexperiments.DefaultAPI, serviceClient serviceenablement.DefaultAPI, providerData core.ProviderData) resource.Resource { //nolint:gocritic
	return &instanceResource{
		client:                  client,
		providerData:            providerData,
		serviceEnablementClient: serviceClient,
	}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client                  modelexperiments.DefaultAPI
	providerData            core.ProviderData
	serviceEnablementClient serviceenablement.DefaultAPI
}

var descriptions = map[string]string{ //nolint:gosec // no hardcoded credentials in here
	"main":                         "Manages a STACKIT AI Model Experiments instance.",
	"main_datasource":              "Datasource scheme for a STACKIT AI Model Experiments instance.",
	"id":                           "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`\".",
	"project_id":                   "STACKIT Project ID to which the resource is associated.",
	"instance_id":                  "The AI Model Experiments instance ID.",
	"region":                       "The STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"labels":                       "A map of arbitrary key/value pairs that can be attached to the resource",
	"description":                  "The description is a longer text chosen by the user to provide more context for the resource.",
	"name":                         "The display name is a short name chosen by the user to identify the resource.",
	"url":                          "The Dashboard URL of the AI Model Experiments instance.",
	"deleted_experiment_retention": "The deleted experiment retention time of the AI Model Experiments instance.",
	"bucket_name":                  "The object storage bucket name of the AI Model Experiments instance.",
}

// Metadata returns the resource type name.
func (i *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_modelexperiments_instance"
}

// Configure adds the provider configured client to the resource.
func (i *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	i.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := modelexperimentsutils.ConfigureClient(ctx, &i.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	serviceEnablementClient := serviceEnablementUtils.ConfigureClient(ctx, &i.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	i.client = apiClient.DefaultAPI
	i.serviceEnablementClient = serviceEnablementClient.DefaultAPI
	tflog.Info(ctx, "Model Experiments client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (i *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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
func (i *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
			"url": schema.StringAttribute{
				Description: descriptions["url"],
				Computed:    true,
			},
			"deleted_experiment_retention": schema.StringAttribute{
				Description: descriptions["deleted_experiment_retention"],
				Optional:    true,
				Computed:    true,
			},
			"bucket_name": schema.StringAttribute{
				Description: descriptions["bucket_name"],
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (i *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := i.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	err := i.serviceEnablementClient.EnableServiceRegional(ctx, region, projectId, utils.ModelExperimentsServiceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error enabling AI Model Experiments",
					fmt.Sprintf("Service not available in region %s \n%v", region, err),
				)
				return
			}
		}

		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error enabling AI Model Experiments",
			fmt.Sprintf("Error enabling AI Model Experiments: %v", err),
		)
		return
	}

	_, err = serviceEnablementWait.EnableServiceWaitHandler(ctx, i.serviceEnablementClient, region, projectId, utils.ModelExperimentsServiceId).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error enabling AI Model Experiments",
			fmt.Sprintf("Error enabling AI Model Experiments: %v", err),
		)
		return
	}

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI Model Experiments instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createInstanceResp, err := i.client.CreateInstance(ctx, projectId, region).CreateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating AI Model Experiments instance",
			fmt.Sprintf("Calling API: %v", err),
		)
		return
	}
	ctx = core.LogResponse(ctx)

	if createInstanceResp.Instance.Id == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", "Got empty instance id")
		return
	}

	instanceId := createInstanceResp.Instance.Id
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	_, err = wait.CreateInstanceWaitHandler(ctx, i.client, region, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI Model Experiments instance", fmt.Sprintf("Waiting for instance to be active: %v", err))
		return
	}

	// Map response body to schema
	err = mapInstance(ctx, &createInstanceResp.Instance, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI Model Experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model Experiments instance created")
}

// Read refreshes the Terraform state with the latest data.
func (i *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	if instanceId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}
	region := i.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	getInstanceResp, err := i.client.GetInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading AI Model Experiments instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapInstance(ctx, &getInstanceResp.Instance, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI Model Experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model Experiments instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (i *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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
	instanceId := state.InstanceId.ValueString()

	region := i.providerData.GetRegionWithOverride(plan.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(&plan)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI Model Experiments instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateInstanceResp, err := i.client.PartialUpdateInstance(ctx, projectId, region, instanceId).PartialUpdateInstancePayload(*payload).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI Model Experiments instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapInstance(ctx, &updateInstanceResp.Instance, &plan, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI Model Experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model Experiments instance updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (i *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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

	region := i.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	_, err := i.client.DeleteInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI Model Experiments instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteInstanceWaitHandler(ctx, i.client, region, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI Model Experiments instance", fmt.Sprintf("Waiting for instance to be deleted: %v", err))
		return
	}

	tflog.Info(ctx, "Model Experiments instance deleted")
}

func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing Model Experiments instance",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[instance_id], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"region":      idParts[1],
		"instance_id": idParts[2],
	})

	tflog.Info(ctx, "Model Experiments instance state imported")
}

// mapInstance maps instances to the resource model
func mapInstance(ctx context.Context, instance *modelexperiments.Instance, model *Model, region string) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if instance.Id == "" {
		return fmt.Errorf("instance id not present")
	}

	mapValue, err := utils.MapLabels(ctx, instance.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instance.Id)
	model.InstanceId = types.StringValue(instance.Id)
	model.Name = types.StringValue(instance.Name)
	model.Description = types.StringPointerValue(instance.Description)
	model.DeletedExperimentRetention = types.StringPointerValue(instance.DeletedExperimentRetention)
	model.BucketName = types.StringPointerValue(instance.BucketName)
	model.Labels = mapValue
	model.Url = types.StringValue(instance.Url)

	return nil
}

func toCreatePayload(model *Model) (*modelexperiments.CreateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &modelexperiments.CreateInstancePayload{
		Name:                       model.Name.ValueString(),
		Description:                conversion.StringValueToPointer(model.Description),
		DeletedExperimentRetention: conversion.StringValueToPointer(model.DeletedExperimentRetention),
		Labels:                     labels,
	}, nil
}

func toUpdatePayload(model *Model) (*modelexperiments.PartialUpdateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}
	return &modelexperiments.PartialUpdateInstancePayload{
		Name:                       conversion.StringValueToPointer(model.Name),
		Description:                conversion.StringValueToPointer(model.Description),
		Labels:                     labels,
		DeletedExperimentRetention: conversion.StringValueToPointer(model.DeletedExperimentRetention),
	}, nil
}
