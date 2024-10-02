package volume

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &volumeResource{}
	_ resource.ResourceWithConfigure   = &volumeResource{}
	_ resource.ResourceWithImportState = &volumeResource{}
)

type Model struct {
	Id               types.String `tfsdk:"id"` // needed by TF
	ProjectId        types.String `tfsdk:"project_id"`
	VolumeId         types.String `tfsdk:"volume_id"`
	Name             types.String `tfsdk:"name"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Labels           types.Map    `tfsdk:"labels"`
	Description      types.String `tfsdk:"description"`
	PerformanceClass types.String `tfsdk:"performance_class"`
	Size             types.Int64  `tfsdk:"size"`
	ServerId         types.String `tfsdk:"server_id"`
}

// NewVolumeResource is a helper function to simplify the provider implementation.
func NewVolumeResource() resource.Resource {
	return &volumeResource{}
}

// volumeResource is the resource implementation.
type volumeResource struct {
	client *iaasalpha.APIClient
}

// Metadata returns the resource type name.
func (r *volumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

// ConfigValidators validates the resource configuration
func (r *volumeResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("size"),
		),
	}
}

// Configure adds the provider configured client to the resource.
func (r *volumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !resourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_volume", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	var apiClient *iaasalpha.APIClient
	var err error
	if providerData.IaaSCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "iaas_custom_endpoint", providerData.IaaSCustomEndpoint)
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "iaasalpha client configured")
}

// Schema defines the schema for the resource.
func (r *volumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Volume resource schema. Must have a `region` specified in the provider configuration."),
		Description:         "Volume resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`volume_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the volume is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"volume_id": schema.StringAttribute{
				Description: "The volume ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"server_id": schema.StringAttribute{
				Description: "The server ID of the server to which the volume is attached to.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the volume.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the volume.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(127),
				},
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zone of the volume.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
			"performance_class": schema.StringAttribute{
				Description: "The performance class of the volume.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
			},
			"size": schema.Int64Attribute{
				Description: "The size of the volume in GB. It can only be updated to a larger value than the current size",
				Optional:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating volume", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new volume

	volume, err := r.client.CreateVolume(ctx, projectId).CreateVolumePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating volume", fmt.Sprintf("Calling API: %v", err))
		return
	}

	volumeId := *volume.Id
	volume, err = wait.CreateVolumeWaitHandler(ctx, r.client, projectId, volumeId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating volume", fmt.Sprintf("volume creation waiting: %v", err))
		return
	}

	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	// Map response body to schema
	err = mapFields(ctx, volume, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating volume", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Volume created")
}

// Read refreshes the Terraform state with the latest data.
func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	volumeId := model.VolumeId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	volumeResp, err := r.client.GetVolume(ctx, projectId, volumeId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading volume", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, volumeResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading volume", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "volume read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	volumeId := model.VolumeId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	// Retrieve values from state
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing volume
	updatedVolume, err := r.client.UpdateVolume(ctx, projectId, volumeId).UpdateVolumePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Resize existing volume
	modelSize := conversion.Int64ValueToPointer(model.Size)
	if modelSize != nil && updatedVolume.Size != nil {
		// A volume can only be resized to larger values, otherwise an error occurs
		if *modelSize < *updatedVolume.Size {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume", fmt.Sprintf("The new volume size must be larger than the current size (%d GB)", *updatedVolume.Size))
		} else if *modelSize > *updatedVolume.Size {
			payload := iaasalpha.ResizeVolumePayload{
				Size: modelSize,
			}
			err := r.client.ResizeVolume(ctx, projectId, volumeId).ResizeVolumePayload(payload).Execute()
			if err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume", fmt.Sprintf("Resizing the volume, calling API: %v", err))
			}
			// Update volume model because the API doesn't return a volume object as response
			updatedVolume.Size = modelSize
		}
	}
	err = mapFields(ctx, updatedVolume, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "volume updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	volumeId := model.VolumeId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	// Delete existing volume
	err := r.client.DeleteVolume(ctx, projectId, volumeId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting volume", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteVolumeWaitHandler(ctx, r.client, projectId, volumeId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting volume", fmt.Sprintf("volume deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "volume deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,volume_id
func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing volume",
			fmt.Sprintf("Expected import identifier with format: [project_id],[volume_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	volumeId := idParts[1]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("volume_id"), volumeId)...)
	tflog.Info(ctx, "volume state imported")
}

func mapFields(ctx context.Context, volumeResp *iaasalpha.Volume, model *Model) error {
	if volumeResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var volumeId string
	if model.VolumeId.ValueString() != "" {
		volumeId = model.VolumeId.ValueString()
	} else if volumeResp.Id != nil {
		volumeId = *volumeResp.Id
	} else {
		return fmt.Errorf("Volume id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		volumeId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	var labels basetypes.MapValue
	if volumeResp.Labels != nil && len(*volumeResp.Labels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *volumeResp.Labels)
		if diags.HasError() {
			return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else {
		labels = types.MapNull(types.StringType)
	}

	model.VolumeId = types.StringValue(volumeId)
	model.AvailabilityZone = types.StringPointerValue(volumeResp.AvailabilityZone)
	model.Description = types.StringPointerValue(volumeResp.Description)
	model.Name = types.StringPointerValue(volumeResp.Name)
	model.Labels = labels
	model.PerformanceClass = types.StringPointerValue(volumeResp.PerformanceClass)
	model.ServerId = types.StringPointerValue(volumeResp.ServerId)
	model.Size = types.Int64PointerValue(volumeResp.Size)
	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaasalpha.CreateVolumePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaasalpha.CreateVolumePayload{
		AvailabilityZone: conversion.StringValueToPointer(model.AvailabilityZone),
		Description:      conversion.StringValueToPointer(model.Description),
		Labels:           &labels,
		Name:             conversion.StringValueToPointer(model.Name),
		PerformanceClass: conversion.StringValueToPointer(model.PerformanceClass),
		Size:             conversion.Int64ValueToPointer(model.Size),
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaasalpha.UpdateVolumePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := utils.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaasalpha.UpdateVolumePayload{
		Description: conversion.StringValueToPointer(model.Description),
		Name:        conversion.StringValueToPointer(model.Name),
		Labels:      &labels,
	}, nil
}
