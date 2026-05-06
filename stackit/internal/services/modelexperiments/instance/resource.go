package instance

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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/wait"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	serviceEnablementWait "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	modelexperimentsutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource              = &instanceResource{}
	_ resource.ResourceWithConfigure = &instanceResource{}
	//_ resource.ResourceWithModifyPlan = &tokenResource{}
)

//go:embed description.md
var markdownDescription string

type Model struct {
	Id                         types.String `tfsdk:"id"` // needed by TF
	ProjectId                  types.String `tfsdk:"project_id"`
	Region                     types.String `tfsdk:"region"`
	Name                       types.String `tfsdk:"name"`
	Description                types.String `tfsdk:"description"`
	DeletedExperimentRetention types.String `tfsdk:"deletedExperimentRetention"`
	Labels                     types.Map    `tfsdk:"labels"`
	State                      types.String `tfsdk:"state"`
	BucketName                 types.String `tfsdk:"bucket_name"`
	ErrorMessage               types.String `tfsdk:"error_message"`
	InstanceId                 types.String `tfsdk:"instance_id"`
	Url                        types.String `tfsdk:"url"`
}

func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

type instanceResource struct {
	client                  *modelexperiments.APIClient
	providerData            core.ProviderData
	serviceEnablementClient *serviceenablement.APIClient
}

func (i *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_order"
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
	serviceEnablementClient := modelexperimentsutils.ConfigureServiceEnablementClient(ctx, &i.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	i.client = apiClient
	i.serviceEnablementClient = serviceEnablementClient
	tflog.Info(ctx, "Model experiments client configured")
}

func (i *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: markdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the AI model experiments instance is associated.",
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
				Description: "Region to which the AI model experiments instance is associated. If not defined, the provider region is used",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "The AI model experiments instance ID.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"labels": schema.MapAttribute{
				Description: "A map of arbitrary key/value pairs for the AI model experiments instance",
				Optional:    true,
				Required:    false,
				Computed:    true,
				ElementType: types.StringType,
			},
			"description": schema.StringAttribute{
				Description: "The description of the AI model experiments instance.",
				Required:    false,
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 160),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the AI model experiments instance.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"state": schema.StringAttribute{
				Description: "State of the AI model experiments instance.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "URL of the AI model experiments instance.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1000),
				},
			},
			"deletedExperimentRetention": schema.StringAttribute{
				Description: "The deleted experiment retention of the AI model experiments instance.",
				Optional:    true,
				Required:    false,
				Computed:    true,
			},
			"bucket_name": schema.StringAttribute{
				Description: "The object storage bucket name of the AI model experiments instance.",
				Computed:    true,
			},
			"error_message": schema.StringAttribute{
				Description: "Error messages of the AI model experiments instance.",
				Optional:    true,
				Required:    false,
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

	err := i.serviceEnablementClient.DefaultAPI.EnableServiceRegional(ctx, region, projectId, utils.ModelExperimentsServiceId).
		Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error enabling AI model experiments",
					fmt.Sprintf("Service not available in region %s \n%v", region, err),
				)
				return
			}
		}
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error enabling AI model experiments",
			fmt.Sprintf("Error enabling AI model experiments: %v", err),
		)
		return
	}

	_, err = serviceEnablementWait.EnableServiceWaitHandler(ctx, i.serviceEnablementClient.DefaultAPI, region, projectId, utils.ModelServingServiceId).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error enabling AI model experiments",
			fmt.Sprintf("Error enabling AI model serving: %v", err),
		)
		return
	}

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model experiments instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createInstanceResp, err := i.client.DefaultAPI.CreateInstance(ctx, projectId, region).CreateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating AI model experiments instance",
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
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	var mapValue basetypes.MapValue
	if createInstanceResp.Instance.Labels != nil {
		mapValue, diags = types.MapValueFrom(ctx, types.StringType, createInstanceResp.Instance.Labels)
		if diags.HasError() {
			return
		}
	}

	//If model experiments instance is impaired, write state avoid dangling resources and return
	waitResp, err := CreateMExpInstanceWaitHandler(ctx, i.client, region, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		mapCreateResponse(createInstanceResp, waitResp, &model, region, mapValue)
		diags = resp.State.Set(ctx, model)
		resp.Diagnostics.Append(diags...)

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model experiments instance", fmt.Sprintf("Waiting for instance to be active: %v", err))
		return
	}

	// Map response body to schema
	err = mapCreateResponse(createInstanceResp, waitResp, &model, region, mapValue)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating AI model experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model experiments instance created")
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

	getInstanceResp, err := i.client.DefaultAPI.GetInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				// Remove the resource from the state so Terraform will recreate it
				resp.State.RemoveResource(ctx)
				return
			}
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading AI model experiments instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapInstance(ctx, getInstanceResp.Instance, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI model experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model experiments instance read")

}

// Update updates the resource and sets the updated Terraform state on success.
func (i *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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

	region := i.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI model experiments instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateInstanceResp, err := i.client.DefaultAPI.PartialUpdateInstance(ctx, projectId, region, instanceId).PartialUpdateInstancePayload(*payload).Execute()
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
			"Error updating AI model experiments instance",
			fmt.Sprintf(
				"Calling API: %v, instanceId: %s, region: %s, projectId: %s",
				err,
				instanceId,
				region,
				projectId,
			),
		)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapInstance(ctx, updateInstanceResp.Instance, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI model experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model experiments instance updated")
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

	_, err := i.client.DefaultAPI.DeleteInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI model experiments instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = DeleteMExpInstanceWaitHandler(ctx, i.client, region, projectId, instanceId).
		WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting AI model experiments instance", fmt.Sprintf("Waiting for instance to be deleted: %v", err))
		return
	}

	tflog.Info(ctx, "Model experiments instance deleted")
}

func mapCreateResponse(instanceCreateResp *modelexperiments.CreateInstanceResponse, waitResp *modelexperiments.GetInstanceResponse, model *Model, region string, labels basetypes.MapValue) error {
	if instanceCreateResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	instance := instanceCreateResp.Instance

	if instance.Id == "" {
		return fmt.Errorf("instance id not present")
	}

	if waitResp == nil {
		return fmt.Errorf("response input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instanceCreateResp.Instance.Id)
	model.InstanceId = types.StringValue(instance.Id)
	model.Name = types.StringValue(instance.Name)
	model.State = types.StringValue(waitResp.Instance.State)
	model.Description = types.StringPointerValue(instance.Description)
	model.DeletedExperimentRetention = types.StringPointerValue(instance.DeletedExperimentRetention)
	model.BucketName = types.StringPointerValue(instance.BucketName)
	model.ErrorMessage = types.StringPointerValue(instance.ErrorMessage)
	model.Labels = labels

	return nil
}

func mapInstance(ctx context.Context, instance modelexperiments.Instance, model *Model) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	mapValue, diags := types.MapValueFrom(ctx, types.StringType, instance.Labels)
	if diags.HasError() {
		return fmt.Errorf("failure in mapping labels")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.Region.ValueString(), model.InstanceId.ValueString())
	model.InstanceId = types.StringValue(instance.Id)
	model.Name = types.StringValue(instance.Name)
	model.State = types.StringValue(instance.State)
	model.Description = types.StringPointerValue(instance.Description)
	model.DeletedExperimentRetention = types.StringPointerValue(instance.DeletedExperimentRetention)
	model.BucketName = types.StringPointerValue(instance.BucketName)
	model.ErrorMessage = types.StringPointerValue(instance.ErrorMessage)
	model.Labels = mapValue

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
		Name:                       model.Name.ValueStringPointer(),
		Description:                model.Description.ValueStringPointer(),
		DeletedExperimentRetention: model.DeletedExperimentRetention.ValueStringPointer(),
		Labels:                     labels,
	}, nil
}

func CreateMExpInstanceWaitHandler(ctx context.Context, a *modelexperiments.APIClient, region, projectId, instanceId string) *wait.AsyncActionHandler[modelexperiments.GetInstanceResponse] {
	handler := wait.New(func() (waitFinished bool, response *modelexperiments.GetInstanceResponse, err error) {
		getInstanceResp, err := a.DefaultAPI.GetInstance(ctx, region, projectId, instanceId).Execute()
		if err != nil {
			return false, nil, err
		}
		if getInstanceResp.Instance.State == modelexperimentsutils.INSTANCESTATE_ACTIVE {
			return true, getInstanceResp, nil
		}
		if getInstanceResp.Instance.State == modelexperimentsutils.INSTANCESTATE_IMPAIRED {
			return true, getInstanceResp, fmt.Errorf("AI model experiments instance is impaired")
		}

		return false, nil, nil
	})

	handler.SetTimeout(10 * time.Minute)

	return handler
}

func DeleteMExpInstanceWaitHandler(ctx context.Context, a *modelexperiments.APIClient, region, projectId, instanceId string) *wait.AsyncActionHandler[modelexperiments.GetInstanceResponse] {
	handler := wait.New(
		func() (waitFinished bool, response *modelexperiments.GetInstanceResponse, err error) {
			_, err = a.DefaultAPI.GetInstance(ctx, region, projectId, instanceId).Execute()
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
