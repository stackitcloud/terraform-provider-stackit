package image

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &imageDataSource{}
)

type DataSourceModel struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	ImageId     types.String `tfsdk:"image_id"`
	Name        types.String `tfsdk:"name"`
	DiskFormat  types.String `tfsdk:"disk_format"`
	MinDiskSize types.Int64  `tfsdk:"min_disk_size"`
	MinRAM      types.Int64  `tfsdk:"min_ram"`
	Protected   types.Bool   `tfsdk:"protected"`
	Scope       types.String `tfsdk:"scope"`
	Config      types.Object `tfsdk:"config"`
	Checksum    types.Object `tfsdk:"checksum"`
	Labels      types.Map    `tfsdk:"labels"`
}

// NewImageDataSource is a helper function to simplify the provider implementation.
func NewImageDataSource() datasource.DataSource {
	return &imageDataSource{}
}

// imageDataSource is the data source implementation.
type imageDataSource struct {
	client *iaas.APIClient
}

// Metadata returns the data source type name.
func (d *imageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (d *imageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the datasource.
func (r *imageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Image datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`image_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the image is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"image_id": schema.StringAttribute{
				Description: "The image ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the image.",
				Computed:    true,
			},
			"disk_format": schema.StringAttribute{
				Description: "The disk format of the image.",
				Computed:    true,
			},
			"min_disk_size": schema.Int64Attribute{
				Description: "The minimum disk size of the image in GB.",
				Computed:    true,
			},
			"min_ram": schema.Int64Attribute{
				Description: "The minimum RAM of the image in MB.",
				Computed:    true,
			},
			"protected": schema.BoolAttribute{
				Description: "Whether the image is protected.",
				Computed:    true,
			},
			"scope": schema.StringAttribute{
				Description: "The scope of the image.",
				Computed:    true,
			},
			"config": schema.SingleNestedAttribute{
				Description: "Properties to set hardware and scheduling settings for an image.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"boot_menu": schema.BoolAttribute{
						Description: "Enables the BIOS bootmenu.",
						Computed:    true,
					},
					"cdrom_bus": schema.StringAttribute{
						Description: "Sets CDROM bus controller type.",
						Computed:    true,
					},
					"disk_bus": schema.StringAttribute{
						Description: "Sets Disk bus controller type.",
						Computed:    true,
					},
					"nic_model": schema.StringAttribute{
						Description: "Sets virtual network interface model.",
						Computed:    true,
					},
					"operating_system": schema.StringAttribute{
						Description: "Enables operating system specific optimizations.",
						Computed:    true,
					},
					"operating_system_distro": schema.StringAttribute{
						Description: "Operating system distribution.",
						Computed:    true,
					},
					"operating_system_version": schema.StringAttribute{
						Description: "Version of the operating system.",
						Computed:    true,
					},
					"rescue_bus": schema.StringAttribute{
						Description: "Sets the device bus when the image is used as a rescue image.",
						Computed:    true,
					},
					"rescue_device": schema.StringAttribute{
						Description: "Sets the device when the image is used as a rescue image.",
						Computed:    true,
					},
					"secure_boot": schema.BoolAttribute{
						Description: "Enables Secure Boot.",
						Computed:    true,
					},
					"uefi": schema.BoolAttribute{
						Description: "Enables UEFI boot.",
						Computed:    true,
					},
					"video_model": schema.StringAttribute{
						Description: "Sets Graphic device model.",
						Computed:    true,
					},
					"virtio_scsi": schema.BoolAttribute{
						Description: "Enables the use of VirtIO SCSI to provide block device access. By default instances use VirtIO Block.",
						Computed:    true,
					},
				},
			},
			"checksum": schema.SingleNestedAttribute{
				Description: "Representation of an image checksum.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"algorithm": schema.StringAttribute{
						Description: "Algorithm for the checksum of the image data.",
						Computed:    true,
					},
					"digest": schema.StringAttribute{
						Description: "Hexdigest of the checksum of the image data.",
						Computed:    true,
					},
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

// // Read refreshes the Terraform state with the latest data.
func (r *imageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
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
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading image",
			fmt.Sprintf("Image with ID %q does not exist in project %q.", imageId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapDataSourceFields(ctx, imageResp, &model)
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
	tflog.Info(ctx, "image read")
}

func mapDataSourceFields(ctx context.Context, imageResp *iaas.Image, model *DataSourceModel) error {
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), imageId)

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
		configModel.VirtioScsi = types.BoolPointerValue(iaas.PtrBool(imageResp.Config.GetVirtioScsi()))

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
	labels, err := iaasUtils.MapLabels(ctx, imageResp.Labels, model.Labels)
	if err != nil {
		return err
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
