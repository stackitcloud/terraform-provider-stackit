package volume

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &volumeDataSource{}
)

type DatasourceModel struct {
	// basically the same as the resource model, just without encryption parameters as they are only **sent** to the API, but **never returned**
	Id               types.String `tfsdk:"id"` // needed by TF
	ProjectId        types.String `tfsdk:"project_id"`
	Region           types.String `tfsdk:"region"`
	VolumeId         types.String `tfsdk:"volume_id"`
	Name             types.String `tfsdk:"name"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Labels           types.Map    `tfsdk:"labels"`
	Description      types.String `tfsdk:"description"`
	PerformanceClass types.String `tfsdk:"performance_class"`
	Size             types.Int64  `tfsdk:"size"`
	ServerId         types.String `tfsdk:"server_id"`
	Source           types.Object `tfsdk:"source"`
	Encrypted        types.Bool   `tfsdk:"encrypted"`
}

// NewVolumeDataSource is a helper function to simplify the provider implementation.
func NewVolumeDataSource() datasource.DataSource {
	return &volumeDataSource{}
}

// volumeDataSource is the data source implementation.
type volumeDataSource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *volumeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (d *volumeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (d *volumeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Volume resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`volume_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the volume is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				// the region cannot be found, so it has to be passed
				Optional: true,
			},
			"volume_id": schema.StringAttribute{
				Description: "The volume ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"server_id": schema.StringAttribute{
				Description: "The server ID of the server to which the volume is attached to.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the volume.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the volume.",
				Computed:    true,
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zone of the volume.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
			"performance_class": schema.StringAttribute{
				MarkdownDescription: "The performance class of the volume. Possible values are documented in [Service plans BlockStorage](https://docs.stackit.cloud/products/storage/block-storage/basics/service-plans/#currently-available-service-plans-performance-classes)",
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				Description: "The size of the volume in GB. It can only be updated to a larger value than the current size",
				Computed:    true,
			},
			"source": schema.SingleNestedAttribute{
				Description: "The source of the volume. It can be either a volume, an image, a snapshot or a backup",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "The type of the source. " + utils.FormatPossibleValues(SupportedSourceTypes...),
						Computed:    true,
					},
					"id": schema.StringAttribute{
						Description: "The ID of the source, e.g. image ID",
						Computed:    true,
					},
				},
			},
			"encrypted": schema.BoolAttribute{
				Description: "Indicates if the volume is encrypted.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *volumeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DatasourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	volumeId := model.VolumeId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	volumeResp, err := d.client.GetVolume(ctx, projectId, region, volumeId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading volume",
			fmt.Sprintf("Volume with ID %q does not exist in project %q.", volumeId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapDatasourceFields(ctx, volumeResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading volume", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "volume read")
}

func mapDatasourceFields(ctx context.Context, volumeResp *iaas.Volume, model *DatasourceModel, region string) error {
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, volumeId)
	model.Region = types.StringValue(region)

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
	model.Encrypted = types.BoolPointerValue(volumeResp.Encrypted)

	return nil
}
