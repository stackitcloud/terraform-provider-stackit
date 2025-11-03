package machineimages

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	skeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Model types
type Model struct {
	Region        types.String `tfsdk:"region"`
	VersionState  types.String `tfsdk:"version_state"`
	MachineImages types.List   `tfsdk:"machine_images"`
}

var (
	versionStateOptions = []string{
		"UNSPECIFIED",
		"SUPPORTED",
	}

	machineImageVersionType = map[string]attr.Type{
		"version":         types.StringType,
		"state":           types.StringType,
		"expiration_date": types.StringType,
		"cri":             types.ListType{ElemType: types.StringType},
	}

	machineImageType = map[string]attr.Type{
		"name":     types.StringType,
		"versions": types.ListType{ElemType: types.ObjectType{AttrTypes: machineImageVersionType}},
	}
)

// Ensure implementation satisfies interface
var _ datasource.DataSource = &optionsDataSource{}

// NewKubernetesMachineImageVersionDataSource creates the data source instance
func NewKubernetesMachineImageVersionDataSource() datasource.DataSource {
	return &optionsDataSource{}
}

type optionsDataSource struct {
	client       *ske.APIClient
	providerData core.ProviderData
}

// Metadata sets the data source type name.
func (d *optionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_machine_image_versions"
}

func (d *optionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.client = skeUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if d.client == nil {
		return
	}

	tflog.Info(ctx, "SKE machine image versions client configured")
}

func (d *optionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Returns a list of supported Kubernetes machine image versions for the cluster nodes."

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Region override. If omitted, the provider’s region will be used.",
			},
			"version_state": schema.StringAttribute{
				Optional:    true,
				Description: "Filter returned machine image versions by their state. " + utils.FormatPossibleValues(versionStateOptions...),
				Validators: []validator.String{
					stringvalidator.OneOf(versionStateOptions...),
				},
			},
			"machine_images": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Supported machine image types and versions.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the OS image (e.g., `ubuntu` or `flatcar`).",
						},
						"versions": schema.ListNestedAttribute{
							Computed:    true,
							Description: "Supported versions of the image.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"version": schema.StringAttribute{
										Computed:    true,
										Description: "Machine image version string.",
									},
									"state": schema.StringAttribute{
										Computed:    true,
										Description: "State of the image version.",
									},
									"expiration_date": schema.StringAttribute{
										Computed:    true,
										Description: "Expiration date of the version in RFC3339 format.",
									},
									"cri": schema.ListAttribute{
										Computed:    true,
										ElementType: types.StringType,
										Description: "Container runtimes supported (e.g., `containerd`).",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *optionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	region := d.providerData.GetRegionWithOverride(model.Region)
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "region", region)

	listProviderOptionsReq := d.client.ListProviderOptions(ctx, region)

	if !utils.IsUndefined(model.VersionState) {
		listProviderOptionsReq = listProviderOptionsReq.VersionState(model.VersionState.ValueString())
	}

	optionsResp, err := listProviderOptionsReq.Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading SKE provider options failed",
			"Unable to read SKE provider options",
			map[int]string{
				http.StatusForbidden: "Forbidden access",
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	if err := mapFields(ctx, optionsResp, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading provider options", fmt.Sprintf("Mapping API payload: %v", err))
		return
	}

	// Set final state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Read SKE provider options successfully", map[string]interface{}{
		"region":       region,
		"versionState": model.VersionState.ValueString(),
	})
}

func mapFields(ctx context.Context, optionsResp *ske.ProviderOptions, model *Model) error {
	if optionsResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	// Machine Images
	miList := make([]attr.Value, 0)
	if optionsResp.MachineImages != nil {
		for _, img := range *optionsResp.MachineImages {
			versionsList := make([]attr.Value, 0)
			if img.Versions != nil {
				for _, ver := range *img.Versions {
					// CRI list
					criList := make([]types.String, 0)
					if ver.Cri != nil {
						for _, cri := range *ver.Cri {
							if cri.Name != nil {
								criList = append(criList, types.StringValue(string(*cri.Name.Ptr())))
							}
						}
					}
					criVal, diags := types.ListValueFrom(ctx, types.StringType, criList)
					if diags.HasError() {
						return core.DiagsToError(diags)
					}

					// Expiration date
					expDate := types.StringNull()
					if ver.ExpirationDate != nil {
						expDate = types.StringValue(ver.ExpirationDate.Format(time.RFC3339))
					}

					versionObj, diags := types.ObjectValue(machineImageVersionType, map[string]attr.Value{
						"version":         types.StringPointerValue(ver.Version),
						"state":           types.StringPointerValue(ver.State),
						"expiration_date": expDate,
						"cri":             criVal,
					})
					if diags.HasError() {
						return core.DiagsToError(diags)
					}
					versionsList = append(versionsList, versionObj)
				}
			}

			versions, diags := types.ListValue(types.ObjectType{AttrTypes: machineImageVersionType}, versionsList)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}

			imgObj, diags := types.ObjectValue(machineImageType, map[string]attr.Value{
				"name":     types.StringPointerValue(img.Name),
				"versions": versions,
			})
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			miList = append(miList, imgObj)
		}
	}

	mis, diags := types.ListValue(types.ObjectType{AttrTypes: machineImageType}, miList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.MachineImages = mis

	return nil
}
