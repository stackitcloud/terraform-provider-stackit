package volume

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &volumeResource{}
	_ resource.ResourceWithConfigure   = &volumeResource{}
	_ resource.ResourceWithImportState = &volumeResource{}

	SupportedSourceTypes = []string{"volume", "image", "snapshot", "backup"}
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
	Source           types.Object `tfsdk:"source"`
}

// Struct corresponding to Model.Source
type sourceModel struct {
	Type types.String `tfsdk:"type"`
	Id   types.String `tfsdk:"id"`
}

// Types corresponding to sourceModel
var sourceTypes = map[string]attr.Type{
	"type": basetypes.StringType{},
	"id":   basetypes.StringType{},
}

// NewVolumeResource is a helper function to simplify the provider implementation.
func NewVolumeResource() resource.Resource {
	return &volumeResource{}
}

// volumeResource is the resource implementation.
type volumeResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *volumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

// ConfigValidators validates the resource configuration
func (r *volumeResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("source"),
			path.MatchRoot("size"),
		),
	}
}

// Configure adds the provider configured client to the resource.
func (r *volumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *volumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Volume resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
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
				MarkdownDescription: "The performance class of the volume. Possible values are documented in [Service plans BlockStorage](https://docs.stackit.cloud/stackit/en/service-plans-blockstorage-75137974.html#ServiceplansBlockStorage-CurrentlyavailableServicePlans%28performanceclasses%29)",
				Optional:            true,
				Computed:            true,
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
				Description: "The size of the volume in GB. It can only be updated to a larger value than the current size. Either `size` or `source` must be provided",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					volumeResizeModifier{},
				},
			},
			"source": schema.SingleNestedAttribute{
				Description: "The source of the volume. It can be either a volume, an image, a snapshot or a backup. Either `size` or `source` must be provided",
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "The type of the source. " + utils.FormatPossibleValues(SupportedSourceTypes...),
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"id": schema.StringAttribute{
						Description: "The ID of the source, e.g. image ID",
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
		},
	}
}

var _ planmodifier.Int64 = volumeResizeModifier{}

type volumeResizeModifier struct {
}

// Description implements planmodifier.String.
func (v volumeResizeModifier) Description(context.Context) string {
	return "validates volume resize"
}

// MarkdownDescription implements planmodifier.String.
func (v volumeResizeModifier) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// PlanModifyInt64 implements planmodifier.Int64.
func (v volumeResizeModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) { // nolint:gocritic // function signature required by Terraform
	var planSize types.Int64
	var currentSize types.Int64

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("size"), &planSize)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("size"), &currentSize)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planSize.ValueInt64() < currentSize.ValueInt64() {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error changing volume size", "A volume cannot be made smaller in order to prevent data loss.")
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

	var source = &sourceModel{}
	if !(model.Source.IsNull() || model.Source.IsUnknown()) {
		diags = model.Source.As(ctx, source, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model, source)
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
			payload := iaas.ResizeVolumePayload{
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

func mapFields(ctx context.Context, volumeResp *iaas.Volume, model *Model) error {
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), volumeId)

	labels, err := iaasUtils.MapLabels(ctx, volumeResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	var sourceValues map[string]attr.Value
	var sourceObject basetypes.ObjectValue
	if volumeResp.Source == nil {
		sourceObject = types.ObjectNull(sourceTypes)
	} else {
		sourceValues = map[string]attr.Value{
			"type": types.StringPointerValue(volumeResp.Source.Type),
			"id":   types.StringPointerValue(volumeResp.Source.Id),
		}
		var diags diag.Diagnostics
		sourceObject, diags = types.ObjectValue(sourceTypes, sourceValues)
		if diags.HasError() {
			return fmt.Errorf("creating source: %w", core.DiagsToError(diags))
		}
	}

	model.VolumeId = types.StringValue(volumeId)
	model.AvailabilityZone = types.StringPointerValue(volumeResp.AvailabilityZone)
	model.Description = types.StringPointerValue(volumeResp.Description)
	model.Name = types.StringPointerValue(volumeResp.Name)
	// Workaround for volumes with no names which return an empty string instead of nil
	if name := volumeResp.Name; name != nil && *name == "" {
		model.Name = types.StringNull()
	}
	model.Labels = labels
	model.PerformanceClass = types.StringPointerValue(volumeResp.PerformanceClass)
	model.ServerId = types.StringPointerValue(volumeResp.ServerId)
	model.Size = types.Int64PointerValue(volumeResp.Size)
	model.Source = sourceObject
	return nil
}

func toCreatePayload(ctx context.Context, model *Model, source *sourceModel) (*iaas.CreateVolumePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	var sourcePayload *iaas.VolumeSource

	if !source.Id.IsNull() && !source.Type.IsNull() {
		sourcePayload = &iaas.VolumeSource{
			Id:   conversion.StringValueToPointer(source.Id),
			Type: conversion.StringValueToPointer(source.Type),
		}
	}

	return &iaas.CreateVolumePayload{
		AvailabilityZone: conversion.StringValueToPointer(model.AvailabilityZone),
		Description:      conversion.StringValueToPointer(model.Description),
		Labels:           &labels,
		Name:             conversion.StringValueToPointer(model.Name),
		PerformanceClass: conversion.StringValueToPointer(model.PerformanceClass),
		Size:             conversion.Int64ValueToPointer(model.Size),
		Source:           sourcePayload,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.UpdateVolumePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.UpdateVolumePayload{
		Description: conversion.StringValueToPointer(model.Description),
		Name:        conversion.StringValueToPointer(model.Name),
		Labels:      &labels,
	}, nil
}
