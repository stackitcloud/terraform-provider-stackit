package machineimages

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	skeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Model types for nested structures
type Model struct {
	Region        types.String `tfsdk:"region"`
	MachineImages types.List   `tfsdk:"machine_images"`
}

var (
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

// Metadata sets the data source type name
func (d *optionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_machine_image_versions"
}

func (d *optionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = skeUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	tflog.Info(ctx, "SKE options client configured")
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
			"machine_images": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Supported machine image types and software versions.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the OS image (e.g., `ubuntu`).",
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
										Description: "State of the image version (e.g., `supported`, `preview`, `deprecated`).",
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
	ctx = tflog.SetField(ctx, "region", region)

	optionsResp, err := d.client.ListProviderOptions(ctx, region).Execute()
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

	err = mapFields(ctx, optionsResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &diags, "Error reading provider options", fmt.Sprintf("Mapping API Payload: %v", err))
		return
	}

	// Set final state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Read SKE provider options successfully")
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
