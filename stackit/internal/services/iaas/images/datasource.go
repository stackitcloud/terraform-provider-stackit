package images

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &imagesDataSource{}
	_ datasource.DataSourceWithConfigure = &imagesDataSource{}
)

type filterModel struct {
	LabelSelector types.String `tfsdk:"label_selector"`
}

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

type checksumModel struct {
	Algorithm types.String `tfsdk:"algorithm"`
	Digest    types.String `tfsdk:"digest"`
}

type imageModel struct {
	ImageID     types.String `tfsdk:"image_id"`
	Name        types.String `tfsdk:"name"`
	DiskFormat  types.String `tfsdk:"disk_format"`
	MinDiskSize types.Int64  `tfsdk:"min_disk_size"`
	MinRAM      types.Int64  `tfsdk:"min_ram"`
	Protected   types.Bool   `tfsdk:"protected"`
	Scope       types.String `tfsdk:"scope"`
	Status      types.String `tfsdk:"status"`
	Labels      types.Map    `tfsdk:"labels"`
	Config      types.Object `tfsdk:"config"`
	Checksum    types.Object `tfsdk:"checksum"`
}

type DataSourceModel struct {
	ID        types.String   `tfsdk:"id"`
	ProjectID types.String   `tfsdk:"project_id"`
	Region    types.String   `tfsdk:"region"`
	Filter    types.Object   `tfsdk:"filter"`
	Results   types.List     `tfsdk:"results"`
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
}

type imagesDataSource struct {
	client       iaas.DefaultAPI
	providerData core.ProviderData
}

var configAttrTypes = map[string]attr.Type{
	"boot_menu":                types.BoolType,
	"cdrom_bus":                types.StringType,
	"disk_bus":                 types.StringType,
	"nic_model":                types.StringType,
	"operating_system":         types.StringType,
	"operating_system_distro":  types.StringType,
	"operating_system_version": types.StringType,
	"rescue_bus":               types.StringType,
	"rescue_device":            types.StringType,
	"secure_boot":              types.BoolType,
	"uefi":                     types.BoolType,
	"video_model":              types.StringType,
	"virtio_scsi":              types.BoolType,
}

var checksumAttrTypes = map[string]attr.Type{
	"algorithm": types.StringType,
	"digest":    types.StringType,
}

var imageAttrTypes = map[string]attr.Type{
	"image_id":      types.StringType,
	"name":          types.StringType,
	"disk_format":   types.StringType,
	"min_disk_size": types.Int64Type,
	"min_ram":       types.Int64Type,
	"protected":     types.BoolType,
	"scope":         types.StringType,
	"status":        types.StringType,
	"labels":        types.MapType{ElemType: types.StringType},
	"config":        types.ObjectType{AttrTypes: configAttrTypes},
	"checksum":      types.ObjectType{AttrTypes: checksumAttrTypes},
}

func NewImagesDataSource() datasource.DataSource {
	return &imagesDataSource{}
}

func (d *imagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images"
}

func (d *imagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.providerData = providerData
	d.client = clients.IaaSv2Client

	features.CheckBetaResourcesEnabled(ctx, &d.providerData, &resp.Diagnostics, "stackit_images", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "iaas client configured")
}

func (d *imagesDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := features.AddBetaDescription(
		"Lists all IaaS images in a project and region. Results are sorted by image ID.",
		core.Datasource,
	)

	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Stable Terraform ID, structured as \"`project_id`,`region`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID.",
				Required:    true,
				Validators:  []validator.String{validate.UUID(), validate.NoSeparator()},
			},
			"region": schema.StringAttribute{
				Description: "Region override. Uses the provider region when omitted.",
				Optional:    true,
			},
			"filter": schema.SingleNestedAttribute{
				Description: "API-side image filtering options.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"label_selector": schema.StringAttribute{
						Description: "Experimental: API label selector passed directly to the image-list endpoint.",
						Optional:    true,
					},
				},
			},
			"results": schema.ListNestedAttribute{
				Description: "All matching images, sorted by image_id ascending.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"image_id": schema.StringAttribute{
							Description: "Image ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Image name.",
							Computed:    true,
						},
						"disk_format": schema.StringAttribute{
							Description: "Image disk format.",
							Computed:    true,
						},
						"min_disk_size": schema.Int64Attribute{
							Description: "Minimum disk size in GB.",
							Computed:    true,
						},
						"min_ram": schema.Int64Attribute{
							Description: "Minimum RAM in MB.",
							Computed:    true,
						},
						"protected": schema.BoolAttribute{
							Description: "Whether the image is protected.",
							Computed:    true,
						},
						"scope": schema.StringAttribute{
							Description: "Image scope.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Image status.",
							Computed:    true,
						},
						"labels": schema.MapAttribute{
							Description: "Image labels.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"config": schema.SingleNestedAttribute{
							Description: "Image hardware and operating-system configuration.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"boot_menu":                schema.BoolAttribute{Computed: true},
								"cdrom_bus":                schema.StringAttribute{Computed: true},
								"disk_bus":                 schema.StringAttribute{Computed: true},
								"nic_model":                schema.StringAttribute{Computed: true},
								"operating_system":         schema.StringAttribute{Computed: true},
								"operating_system_distro":  schema.StringAttribute{Computed: true},
								"operating_system_version": schema.StringAttribute{Computed: true},
								"rescue_bus":               schema.StringAttribute{Computed: true},
								"rescue_device":            schema.StringAttribute{Computed: true},
								"secure_boot":              schema.BoolAttribute{Computed: true},
								"uefi":                     schema.BoolAttribute{Computed: true},
								"video_model":              schema.StringAttribute{Computed: true},
								"virtio_scsi":              schema.BoolAttribute{Computed: true},
							},
						},
						"checksum": schema.SingleNestedAttribute{
							Description: "Image checksum.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"algorithm": schema.StringAttribute{Computed: true},
								"digest":    schema.StringAttribute{Computed: true},
							},
						},
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

func toRequest(ctx context.Context, client iaas.DefaultAPI, model *DataSourceModel, providerData *core.ProviderData) (iaas.ApiListImagesRequest, error) {
	if model == nil {
		return iaas.ApiListImagesRequest{}, fmt.Errorf("data source model is nil")
	}

	request := client.ListImages(ctx, model.ProjectID.ValueString(), providerData.GetRegionWithOverride(model.Region)).All(true)
	if !model.Filter.IsNull() && !model.Filter.IsUnknown() {
		var filter filterModel
		diags := model.Filter.As(ctx, &filter, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return iaas.ApiListImagesRequest{}, fmt.Errorf("converting filter: %w", core.DiagsToError(diags))
		}

		if !filter.LabelSelector.IsNull() && !filter.LabelSelector.IsUnknown() && filter.LabelSelector.ValueString() != "" {
			request = request.LabelSelector(filter.LabelSelector.ValueString())
		}
	}

	return request, nil
}

// nolint:gocritic // framework signature required
func (d *imagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model DataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)
	region := d.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", model.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "region", region)

	request, err := toRequest(ctx, d.client, &model, &d.providerData)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading images", fmt.Sprintf("Creating API request: %v", err))
		return
	}

	apiResponse, err := request.Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading images", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	if err = mapFields(ctx, apiResponse, &model, region); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading images", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "images read")
}

func mapFields(ctx context.Context, response *iaas.ImageListResponse, model *DataSourceModel, region string) error {
	if response == nil {
		return fmt.Errorf("image list response is nil")
	}
	if model == nil {
		return fmt.Errorf("data source model is nil")
	}

	items := response.Items
	for i := range items {
		if items[i].Id == nil || *items[i].Id == "" {
			return fmt.Errorf("image at index %d has no ID", i)
		}
	}
	slices.SortFunc(items, func(a, b iaas.Image) int { return strings.Compare(*a.Id, *b.Id) })

	values := make([]imageModel, 0, len(items))
	for i := range items {
		item, err := mapImage(ctx, &items[i])
		if err != nil {
			return fmt.Errorf("mapping image at index %d (ID %q): %w", i, *items[i].Id, err)
		}
		values = append(values, item)
	}

	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: imageAttrTypes}, values)
	if diags.HasError() {
		return fmt.Errorf("converting image results: %w", core.DiagsToError(diags))
	}

	model.Results = list
	model.Region = types.StringValue(region)
	model.ID = utils.BuildInternalTerraformId(model.ProjectID.ValueString(), region)

	return nil
}

func mapImage(ctx context.Context, image *iaas.Image) (imageModel, error) {
	if image == nil {
		return imageModel{}, fmt.Errorf("image is nil")
	}
	if image.Id == nil || *image.Id == "" {
		return imageModel{}, fmt.Errorf("image ID is missing")
	}

	m := imageModel{
		ImageID:     types.StringValue(*image.Id),
		Name:        types.StringValue(image.Name),
		DiskFormat:  types.StringValue(image.DiskFormat),
		MinDiskSize: types.Int64PointerValue(image.MinDiskSize),
		MinRAM:      types.Int64PointerValue(image.MinRam),
		Protected:   types.BoolPointerValue(image.Protected),
		Scope:       types.StringPointerValue(image.Scope),
		Status:      types.StringPointerValue(image.Status),
	}

	labels, err := iaasUtils.MapLabels(ctx, image.Labels, types.MapNull(types.StringType))
	if err != nil {
		return m, fmt.Errorf("mapping labels: %w", err)
	}
	m.Labels = labels

	if image.Config == nil {
		m.Config = types.ObjectNull(configAttrTypes)
	} else {
		c := image.Config
		config := configModel{
			BootMenu:               types.BoolPointerValue(c.BootMenu),
			CDROMBus:               types.StringPointerValue(c.CdromBus.Get()),
			DiskBus:                types.StringPointerValue(c.DiskBus.Get()),
			NICModel:               types.StringPointerValue(c.NicModel.Get()),
			OperatingSystem:        types.StringPointerValue(c.OperatingSystem),
			OperatingSystemDistro:  types.StringPointerValue(c.OperatingSystemDistro.Get()),
			OperatingSystemVersion: types.StringPointerValue(c.OperatingSystemVersion.Get()),
			RescueBus:              types.StringPointerValue(c.RescueBus.Get()),
			RescueDevice:           types.StringPointerValue(c.RescueDevice.Get()),
			SecureBoot:             types.BoolPointerValue(c.SecureBoot),
			UEFI:                   types.BoolPointerValue(c.Uefi),
			VideoModel:             types.StringPointerValue(c.VideoModel.Get()),
			VirtioScsi:             types.BoolPointerValue(c.VirtioScsi),
		}
		obj, ds := types.ObjectValueFrom(ctx, configAttrTypes, config)
		if ds.HasError() {
			return m, fmt.Errorf("mapping config: %w", core.DiagsToError(ds))
		}
		m.Config = obj
	}

	if image.Checksum == nil {
		m.Checksum = types.ObjectNull(checksumAttrTypes)
	} else {
		cs := checksumModel{
			Algorithm: types.StringValue(image.Checksum.Algorithm),
			Digest:    types.StringValue(image.Checksum.Digest),
		}
		obj, ds := types.ObjectValueFrom(ctx, checksumAttrTypes, cs)
		if ds.HasError() {
			return m, fmt.Errorf("mapping checksum: %w", core.DiagsToError(ds))
		}
		m.Checksum = obj
	}

	return m, nil
}
