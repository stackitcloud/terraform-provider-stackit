package image

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &imageResource{}
	_ resource.ResourceWithConfigure   = &imageResource{}
	_ resource.ResourceWithImportState = &imageResource{}
)

type Model struct {
	Id            types.String `tfsdk:"id"` // needed by TF
	ProjectId     types.String `tfsdk:"project_id"`
	ImageId       types.String `tfsdk:"image_id"`
	Name          types.String `tfsdk:"name"`
	DiskFormat    types.String `tfsdk:"disk_format"`
	MinDiskSize   types.Int64  `tfsdk:"min_disk_size"`
	MinRAM        types.Int64  `tfsdk:"min_ram"`
	Protected     types.Bool   `tfsdk:"protected"`
	Scope         types.String `tfsdk:"scope"`
	Config        types.Object `tfsdk:"config"`
	Checksum      types.Object `tfsdk:"checksum"`
	Labels        types.Map    `tfsdk:"labels"`
	LocalFilePath types.String `tfsdk:"local_file_path"`
}

// Struct corresponding to Model.Config
type configModel struct {
	BootMenu               types.Bool   `tfsdk:"boot_menu"`
	CDROMBus               types.String `tfsdk:"cdrom_bus"`
	DiskBus                types.String `tfsdk:"disk_bus"`
	NICModel               types.String `tfsdk:"nic_model"`
	OperatingSystem        types.String `tfsdk:"operating_system"`
	OperatingSystemDistro  types.String `tfsdk:"operating_system_distro"`
	OperatingSystemVersion types.String `tfsdk:"operating_system_version"`
	RescueBus              types.String `tfsdk:"rescue_bus"`
	RescueDevice           types.String `tfsdk:"rescue_device"`
	SecureBoot             types.Bool   `tfsdk:"secure_boot"`
	UEFI                   types.Bool   `tfsdk:"uefi"`
	VideoModel             types.String `tfsdk:"video_model"`
	VirtioScsi             types.Bool   `tfsdk:"virtio_scsi"`
}

// Types corresponding to configModel
var configTypes = map[string]attr.Type{
	"boot_menu":                basetypes.BoolType{},
	"cdrom_bus":                basetypes.StringType{},
	"disk_bus":                 basetypes.StringType{},
	"nic_model":                basetypes.StringType{},
	"operating_system":         basetypes.StringType{},
	"operating_system_distro":  basetypes.StringType{},
	"operating_system_version": basetypes.StringType{},
	"rescue_bus":               basetypes.StringType{},
	"rescue_device":            basetypes.StringType{},
	"secure_boot":              basetypes.BoolType{},
	"uefi":                     basetypes.BoolType{},
	"video_model":              basetypes.StringType{},
	"virtio_scsi":              basetypes.BoolType{},
}

// Struct corresponding to Model.Checksum
type checksumModel struct {
	Algorithm types.String `tfsdk:"algorithm"`
	Digest    types.String `tfsdk:"digest"`
}

// Types corresponding to checksumModel
var checksumTypes = map[string]attr.Type{
	"algorithm": basetypes.StringType{},
	"digest":    basetypes.StringType{},
}

// NewImageResource is a helper function to simplify the provider implementation.
func NewImageResource() resource.Resource {
	return &imageResource{}
}

// imageResource is the resource implementation.
type imageResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *imageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

// Configure adds the provider configured client to the resource.
func (r *imageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_image", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	var apiClient *iaas.APIClient
	var err error
	if providerData.IaaSCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "iaas_custom_endpoint", providerData.IaaSCustomEndpoint)
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *imageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Image resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`image_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the image is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"image_id": schema.StringAttribute{
				Description: "The image ID.",
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
				Description: "The name of the image.",
				Required:    true,
			},
			"disk_format": schema.StringAttribute{
				Description: "The disk format of the image.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"local_file_path": schema.StringAttribute{
				Description: "The filepath of the raw image file to be uploaded.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					// Validating that the file exists in the plan is useful to avoid
					// creating an image resource where the local image upload will fail
					validate.FileExists(),
				},
			},
			"min_disk_size": schema.Int64Attribute{
				Description: "The minimum disk size of the image in GB.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_ram": schema.Int64Attribute{
				Description: "The minimum RAM of the image in MB.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"protected": schema.BoolAttribute{
				Description: "Whether the image is protected.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"scope": schema.StringAttribute{
				Description: "The scope of the image.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"config": schema.SingleNestedAttribute{
				Description: "Properties to set hardware and scheduling settings for an image.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"boot_menu": schema.BoolAttribute{
						Description: "Enables the BIOS bootmenu.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"cdrom_bus": schema.StringAttribute{
						Description: "Sets CDROM bus controller type.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"disk_bus": schema.StringAttribute{
						Description: "Sets Disk bus controller type.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"nic_model": schema.StringAttribute{
						Description: "Sets virtual network interface model.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"operating_system": schema.StringAttribute{
						Description: "Enables operating system specific optimizations.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"operating_system_distro": schema.StringAttribute{
						Description: "Operating system distribution.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"operating_system_version": schema.StringAttribute{
						Description: "Version of the operating system.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"rescue_bus": schema.StringAttribute{
						Description: "Sets the device bus when the image is used as a rescue image.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"rescue_device": schema.StringAttribute{
						Description: "Sets the device when the image is used as a rescue image.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"secure_boot": schema.BoolAttribute{
						Description: "Enables Secure Boot.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"uefi": schema.BoolAttribute{
						Description: "Enables UEFI boot.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"video_model": schema.StringAttribute{
						Description: "Sets Graphic device model.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"virtio_scsi": schema.BoolAttribute{
						Description: "Enables the use of VirtIO SCSI to provide block device access. By default instances use VirtIO Block.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"checksum": schema.SingleNestedAttribute{
				Description: "Representation of an image checksum.",
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"algorithm": schema.StringAttribute{
						Description: "Algorithm for the checksum of the image data.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"digest": schema.StringAttribute{
						Description: "Hexdigest of the checksum of the image data.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *imageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating image", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new image
	imageCreateResp, err := r.client.CreateImage(ctx, projectId).CreateImagePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating image", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "image_id", *imageCreateResp.Id)

	// Get the image object, as the create response does not contain all fields
	image, err := r.client.GetImage(ctx, projectId, *imageCreateResp.Id).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating image", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, image, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating image", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to partially populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Upload image
	err = uploadImage(ctx, &resp.Diagnostics, model.LocalFilePath.ValueString(), *imageCreateResp.UploadUrl)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating image", fmt.Sprintf("Uploading image: %v", err))
		return
	}

	// Wait for image to become available
	waitResp, err := wait.UploadImageWaitHandler(ctx, r.client, projectId, *imageCreateResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating image", fmt.Sprintf("Waiting for image to become available: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating image", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Image created")
}

// // Read refreshes the Terraform state with the latest data.
func (r *imageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	imageId := model.ImageId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "image_id", imageId)

	imageResp, err := r.client.GetImage(ctx, projectId, imageId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading image", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, imageResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading image", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Image read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *imageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	imageId := model.ImageId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "image_id", imageId)

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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating image", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing image
	updatedImage, err := r.client.UpdateImage(ctx, projectId, imageId).UpdateImagePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating image", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, updatedImage, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating image", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Image updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *imageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	imageId := model.ImageId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "image_id", imageId)

	// Delete existing image
	err := r.client.DeleteImage(ctx, projectId, imageId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting image", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteImageWaitHandler(ctx, r.client, projectId, imageId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting image", fmt.Sprintf("image deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Image deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,image_id
func (r *imageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing image",
			fmt.Sprintf("Expected import identifier with format: [project_id],[image_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	imageId := idParts[1]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "image_id", imageId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("image_id"), imageId)...)
	tflog.Info(ctx, "Image state imported")
}

func mapFields(ctx context.Context, imageResp *iaas.Image, model *Model) error {
	if imageResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var imageId string
	if model.ImageId.ValueString() != "" {
		imageId = model.ImageId.ValueString()
	} else if imageResp.Id != nil {
		imageId = *imageResp.Id
	} else {
		return fmt.Errorf("image id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		imageId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	// Map config
	var configModel = &configModel{}
	var configObject basetypes.ObjectValue
	diags := diag.Diagnostics{}
	if imageResp.Config != nil {
		configModel.BootMenu = types.BoolPointerValue(imageResp.Config.BootMenu)
		configModel.CDROMBus = types.StringPointerValue(imageResp.Config.GetCdromBus())
		configModel.DiskBus = types.StringPointerValue(imageResp.Config.GetDiskBus())
		configModel.NICModel = types.StringPointerValue(imageResp.Config.GetNicModel())
		configModel.OperatingSystem = types.StringPointerValue(imageResp.Config.OperatingSystem)
		configModel.OperatingSystemDistro = types.StringPointerValue(imageResp.Config.GetOperatingSystemDistro())
		configModel.OperatingSystemVersion = types.StringPointerValue(imageResp.Config.GetOperatingSystemVersion())
		configModel.RescueBus = types.StringPointerValue(imageResp.Config.GetRescueBus())
		configModel.RescueDevice = types.StringPointerValue(imageResp.Config.GetRescueDevice())
		configModel.SecureBoot = types.BoolPointerValue(imageResp.Config.SecureBoot)
		configModel.UEFI = types.BoolPointerValue(imageResp.Config.Uefi)
		configModel.VideoModel = types.StringPointerValue(imageResp.Config.GetVideoModel())
		configModel.VirtioScsi = types.BoolPointerValue(imageResp.Config.VirtioScsi)

		configObject, diags = types.ObjectValue(configTypes, map[string]attr.Value{
			"boot_menu":                configModel.BootMenu,
			"cdrom_bus":                configModel.CDROMBus,
			"disk_bus":                 configModel.DiskBus,
			"nic_model":                configModel.NICModel,
			"operating_system":         configModel.OperatingSystem,
			"operating_system_distro":  configModel.OperatingSystemDistro,
			"operating_system_version": configModel.OperatingSystemVersion,
			"rescue_bus":               configModel.RescueBus,
			"rescue_device":            configModel.RescueDevice,
			"secure_boot":              configModel.SecureBoot,
			"uefi":                     configModel.UEFI,
			"video_model":              configModel.VideoModel,
			"virtio_scsi":              configModel.VirtioScsi,
		})
	} else {
		configObject = types.ObjectNull(configTypes)
	}
	if diags.HasError() {
		return fmt.Errorf("creating config: %w", core.DiagsToError(diags))
	}

	// Map checksum
	var checksumModel = &checksumModel{}
	var checksumObject basetypes.ObjectValue
	if imageResp.Checksum != nil {
		checksumModel.Algorithm = types.StringPointerValue(imageResp.Checksum.Algorithm)
		checksumModel.Digest = types.StringPointerValue(imageResp.Checksum.Digest)
		checksumObject, diags = types.ObjectValue(checksumTypes, map[string]attr.Value{
			"algorithm": checksumModel.Algorithm,
			"digest":    checksumModel.Digest,
		})
	} else {
		checksumObject = types.ObjectNull(checksumTypes)
	}
	if diags.HasError() {
		return fmt.Errorf("creating checksum: %w", core.DiagsToError(diags))
	}

	// Map labels
	labels, diags := types.MapValueFrom(ctx, types.StringType, map[string]interface{}{})
	if diags.HasError() {
		return fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
	}
	if imageResp.Labels != nil && len(*imageResp.Labels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *imageResp.Labels)
		if diags.HasError() {
			return fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else if model.Labels.IsNull() {
		labels = types.MapNull(types.StringType)
	}

	model.ImageId = types.StringValue(imageId)
	model.Name = types.StringPointerValue(imageResp.Name)
	model.DiskFormat = types.StringPointerValue(imageResp.DiskFormat)
	model.MinDiskSize = types.Int64PointerValue(imageResp.MinDiskSize)
	model.MinRAM = types.Int64PointerValue(imageResp.MinRam)
	model.Protected = types.BoolPointerValue(imageResp.Protected)
	model.Scope = types.StringPointerValue(imageResp.Scope)
	model.Labels = labels
	model.Config = configObject
	model.Checksum = checksumObject
	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateImagePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var configModel = &configModel{}
	if !(model.Config.IsNull() || model.Config.IsUnknown()) {
		diags := model.Config.As(ctx, configModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("convert boot volume object to struct: %w", core.DiagsToError(diags))
		}
	}

	configPayload := &iaas.ImageConfig{
		BootMenu:               conversion.BoolValueToPointer(configModel.BootMenu),
		CdromBus:               iaas.NewNullableString(conversion.StringValueToPointer(configModel.CDROMBus)),
		DiskBus:                iaas.NewNullableString(conversion.StringValueToPointer(configModel.DiskBus)),
		NicModel:               iaas.NewNullableString(conversion.StringValueToPointer(configModel.NICModel)),
		OperatingSystem:        conversion.StringValueToPointer(configModel.OperatingSystem),
		OperatingSystemDistro:  iaas.NewNullableString(conversion.StringValueToPointer(configModel.OperatingSystemDistro)),
		OperatingSystemVersion: iaas.NewNullableString(conversion.StringValueToPointer(configModel.OperatingSystemVersion)),
		RescueBus:              iaas.NewNullableString(conversion.StringValueToPointer(configModel.RescueBus)),
		RescueDevice:           iaas.NewNullableString(conversion.StringValueToPointer(configModel.RescueDevice)),
		SecureBoot:             conversion.BoolValueToPointer(configModel.SecureBoot),
		Uefi:                   conversion.BoolValueToPointer(configModel.UEFI),
		VideoModel:             iaas.NewNullableString(conversion.StringValueToPointer(configModel.VideoModel)),
		VirtioScsi:             conversion.BoolValueToPointer(configModel.VirtioScsi),
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.CreateImagePayload{
		Name:        conversion.StringValueToPointer(model.Name),
		DiskFormat:  conversion.StringValueToPointer(model.DiskFormat),
		MinDiskSize: conversion.Int64ValueToPointer(model.MinDiskSize),
		MinRam:      conversion.Int64ValueToPointer(model.MinRAM),
		Protected:   conversion.BoolValueToPointer(model.Protected),
		Config:      configPayload,
		Labels:      &labels,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.UpdateImagePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var configModel = &configModel{}
	if !(model.Config.IsNull() || model.Config.IsUnknown()) {
		diags := model.Config.As(ctx, configModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("convert boot volume object to struct: %w", core.DiagsToError(diags))
		}
	}

	configPayload := &iaas.ImageConfig{
		BootMenu:               conversion.BoolValueToPointer(configModel.BootMenu),
		CdromBus:               iaas.NewNullableString(conversion.StringValueToPointer(configModel.CDROMBus)),
		DiskBus:                iaas.NewNullableString(conversion.StringValueToPointer(configModel.DiskBus)),
		NicModel:               iaas.NewNullableString(conversion.StringValueToPointer(configModel.NICModel)),
		OperatingSystem:        conversion.StringValueToPointer(configModel.OperatingSystem),
		OperatingSystemDistro:  iaas.NewNullableString(conversion.StringValueToPointer(configModel.OperatingSystemDistro)),
		OperatingSystemVersion: iaas.NewNullableString(conversion.StringValueToPointer(configModel.OperatingSystemVersion)),
		RescueBus:              iaas.NewNullableString(conversion.StringValueToPointer(configModel.RescueBus)),
		RescueDevice:           iaas.NewNullableString(conversion.StringValueToPointer(configModel.RescueDevice)),
		SecureBoot:             conversion.BoolValueToPointer(configModel.SecureBoot),
		Uefi:                   conversion.BoolValueToPointer(configModel.UEFI),
		VideoModel:             iaas.NewNullableString(conversion.StringValueToPointer(configModel.VideoModel)),
		VirtioScsi:             conversion.BoolValueToPointer(configModel.VirtioScsi),
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to go map: %w", err)
	}

	// DiskFormat is not sent in the update payload as does not have effect after image upload,
	// and the field has RequiresReplace set
	return &iaas.UpdateImagePayload{
		Name:        conversion.StringValueToPointer(model.Name),
		MinDiskSize: conversion.Int64ValueToPointer(model.MinDiskSize),
		MinRam:      conversion.Int64ValueToPointer(model.MinRAM),
		Protected:   conversion.BoolValueToPointer(model.Protected),
		Config:      configPayload,
		Labels:      &labels,
	}, nil
}

func uploadImage(ctx context.Context, diags *diag.Diagnostics, filePath, uploadURL string) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}
	if uploadURL == "" {
		return fmt.Errorf("upload URL is empty")
	}

	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(fileContents))
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload image: %w", err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			core.LogAndAddError(ctx, diags, "Error uploading image", fmt.Sprintf("Closing response body: %v", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload image: %s", resp.Status)
	}

	return nil
}
