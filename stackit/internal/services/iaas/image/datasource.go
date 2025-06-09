package image

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &imageDataSource{}
)

type DataSourceModel struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	ImageId        types.String `tfsdk:"image_id"`
	Name           types.String `tfsdk:"name"`
	NameRegex      types.String `tfsdk:"name_regex"`
	SortDescending types.Bool   `tfsdk:"sort_descending"`
	Filter         types.Object `tfsdk:"filter"`

	DiskFormat  types.String `tfsdk:"disk_format"`
	MinDiskSize types.Int64  `tfsdk:"min_disk_size"`
	MinRAM      types.Int64  `tfsdk:"min_ram"`
	Protected   types.Bool   `tfsdk:"protected"`
	Scope       types.String `tfsdk:"scope"`
	Config      types.Object `tfsdk:"config"`
	Checksum    types.Object `tfsdk:"checksum"`
	Labels      types.Map    `tfsdk:"labels"`
}

type Filter struct {
	OS         types.String `tfsdk:"os"`
	Distro     types.String `tfsdk:"distro"`
	Version    types.String `tfsdk:"version"`
	UEFI       types.Bool   `tfsdk:"uefi"`
	SecureBoot types.Bool   `tfsdk:"secure_boot"`
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

func (d *imageDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.Conflicting(
			path.MatchRoot("name"),
			path.MatchRoot("name_regex"),
			path.MatchRoot("image_id"),
		),
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("name"),
			path.MatchRoot("name_regex"),
			path.MatchRoot("image_id"),
			path.MatchRoot("filter"),
		),
	}
}

// Schema defines the schema for the datasource.
func (d *imageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
				Description: "Image ID to fetch directly",
				Optional:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Exact image name to match. Optionally applies a `filter` block to further refine results in case multiple images share the same name. The first match is returned, optionally sorted by name in descending order. Cannot be used together with `name_regex`.",
				Optional:    true,
			},
			"name_regex": schema.StringAttribute{
				Description: "Regular expression to match against image names. Optionally applies a `filter` block to narrow down results when multiple image names match the regex. The first match is returned, optionally sorted by name in descending order. Cannot be used together with `name`.",
				Optional:    true,
			},
			"sort_descending": schema.BoolAttribute{
				Description: "If set to `true`, images are sorted in descending lexicographical order by image name before selecting the first match. Defaults to `false` (ascending).",
				Optional:    true,
			},
			"filter": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Additional filtering options based on image properties. Can be used independently or in conjunction with `name` or `name_regex`.",
				Attributes: map[string]schema.Attribute{
					"os": schema.StringAttribute{
						Optional:    true,
						Description: "Filter images by operating system type, such as `linux` or `windows`.",
					},
					"distro": schema.StringAttribute{
						Optional:    true,
						Description: "Filter images by operating system distribution. For example: `ubuntu`, `ubuntu-arm64`, `debian`, `rhel`, etc.",
					},
					"version": schema.StringAttribute{
						Optional:    true,
						Description: "Filter images by OS distribution version, such as `22.04`, `11`, or `9.1`.",
					},
					"uefi": schema.BoolAttribute{
						Optional:    true,
						Description: "Filter images based on UEFI support. Set to `true` to match images that support UEFI.",
					},
					"secure_boot": schema.BoolAttribute{
						Optional:    true,
						Description: "Filter images with Secure Boot support. Set to `true` to match images that support Secure Boot.",
					},
				},
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

// Read refreshes the Terraform state with the latest data.
func (d *imageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := model.ProjectId.ValueString()
	imageID := model.ImageId.ValueString()
	name := model.Name.ValueString()
	nameRegex := model.NameRegex.ValueString()
	sortDescending := model.SortDescending.ValueBool()

	var filter Filter
	if !model.Filter.IsNull() && !model.Filter.IsUnknown() {
		if diagnostics := model.Filter.As(ctx, &filter, basetypes.ObjectAsOptions{}); diagnostics.HasError() {
			resp.Diagnostics.Append(diagnostics...)
			return
		}
	}

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "image_id", imageID)
	ctx = tflog.SetField(ctx, "name", name)
	ctx = tflog.SetField(ctx, "name_regex", nameRegex)
	ctx = tflog.SetField(ctx, "sort_descending", sortDescending)

	var imageResp *iaas.Image
	var err error

	// Case 1: Direct lookup by image ID
	if imageID != "" {
		imageResp, err = d.client.GetImage(ctx, projectID, imageID).Execute()
		if err != nil {
			utils.LogError(ctx, &resp.Diagnostics, err, "Reading image",
				fmt.Sprintf("Image with ID %q does not exist in project %q.", imageID, projectID),
				map[int]string{
					http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectID),
				})
			resp.State.RemoveResource(ctx)
			return
		}
	} else {
		// Case 2: Lookup by name or name_regex

		// Compile regex
		var compiledRegex *regexp.Regexp
		if nameRegex != "" {
			compiledRegex, err = regexp.Compile(nameRegex)
			if err != nil {
				core.LogAndAddWarning(ctx, &resp.Diagnostics, "Invalid name_regex", err.Error())
				return
			}
		}

		// Fetch all available images
		imageList, err := d.client.ListImages(ctx, projectID).Execute()
		if err != nil {
			utils.LogError(ctx, &resp.Diagnostics, err, "List images", "Unable to fetch images", nil)
			return
		}

		// Step 1: Match images by name or regular expression (name or name_regex, if provided)
		var matchedImages []*iaas.Image
		for _, img := range *imageList.Items {
			if name != "" && img.Name != nil && *img.Name == name {
				matchedImages = append(matchedImages, &img)
			}
			if compiledRegex != nil && img.Name != nil && compiledRegex.MatchString(*img.Name) {
				matchedImages = append(matchedImages, &img)
			}
			// If neither name nor name_regex is specified, include all images for filter evaluation later
			if name == "" && nameRegex == "" {
				matchedImages = append(matchedImages, &img)
			}
		}

		// Step 2: Sort matched images by name (optional, based on sortDescending flag)
		if len(matchedImages) > 1 {
			sort.SliceStable(matchedImages, func(i, j int) bool {
				a, b := matchedImages[i].Name, matchedImages[j].Name
				if a == nil {
					return false
				}
				if b == nil {
					return true
				}
				if sortDescending {
					return *a > *b
				}
				return *a < *b
			})
		}

		// Step 3: Apply additional filtering based on OS, distro, version, UEFI, secure boot, etc.
		filteredImages := make([]*iaas.Image, 0, len(matchedImages))
		for _, img := range matchedImages {
			if imageMatchesFilter(img, &filter) {
				filteredImages = append(filteredImages, img)
			}
		}

		// Check if any images passed all filters; warn if no matching image was found
		if len(filteredImages) == 0 {
			core.LogAndAddWarning(ctx, &resp.Diagnostics, "No match",
				"No matching image found using name, name_regex, and filter criteria.")
			return
		}

		// Step 4: Use the first image from the filtered and sorted result list
		imageResp = filteredImages[0]
	}

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

// imageMatchesFilter checks whether a given image matches all specified filter conditions.
// It returns true only if all non-null fields in the filter match corresponding fields in the image's config.
func imageMatchesFilter(img *iaas.Image, filter *Filter) bool {
	if filter == nil {
		return true
	}

	if img.Config == nil {
		return false
	}

	cfg := img.Config

	if !filter.OS.IsNull() {
		if cfg.OperatingSystem == nil || filter.OS.ValueString() != *cfg.OperatingSystem {
			return false
		}
	}

	if !filter.Distro.IsNull() {
		if cfg.OperatingSystemDistro == nil || cfg.OperatingSystemDistro.Get() == nil ||
			filter.Distro.ValueString() != *cfg.OperatingSystemDistro.Get() {
			return false
		}
	}

	if !filter.Version.IsNull() {
		if cfg.OperatingSystemVersion == nil || cfg.OperatingSystemVersion.Get() == nil ||
			filter.Version.ValueString() != *cfg.OperatingSystemVersion.Get() {
			return false
		}
	}

	if !filter.UEFI.IsNull() {
		if cfg.Uefi == nil || filter.UEFI.ValueBool() != *cfg.Uefi {
			return false
		}
	}

	if !filter.SecureBoot.IsNull() {
		if cfg.SecureBoot == nil || filter.SecureBoot.ValueBool() != *cfg.SecureBoot {
			return false
		}
	}

	return true
}
