package providerOptions

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
	Region             types.String `tfsdk:"region"`
	AvailabilityZones  types.List   `tfsdk:"availability_zones"`
	KubernetesVersions types.List   `tfsdk:"kubernetes_versions"`
	MachineTypes       types.List   `tfsdk:"machine_types"`
	MachineImages      types.List   `tfsdk:"machine_images"`
	VolumeTypes        types.List   `tfsdk:"volume_types"`
}

var (
	kubernetesVersionType = map[string]attr.Type{
		"version":         types.StringType,
		"expiration_date": types.StringType,
		"state":           types.StringType,
	}

	machineTypeAttributeType = map[string]attr.Type{
		"name":         types.StringType,
		"architecture": types.StringType,
		"cpu":          types.Int64Type,
		"gpu":          types.Int64Type,
		"memory":       types.Int64Type,
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

// NewOptionsDataSource creates the data source instance
func NewOptionsDataSource() datasource.DataSource {
	return &optionsDataSource{}
}

type optionsDataSource struct {
	client       *ske.APIClient
	providerData core.ProviderData
}

// Metadata sets the data source type name
func (d *optionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_provider_options"
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
	description := "Returns a list of supported Kubernetes versions and a list of supported machine types for the cluster nodes."

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Region override. If omitted, the provider’s region will be used.",
			},
			"availability_zones": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of availability zones in the selected region.",
			},
			"kubernetes_versions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Supported Kubernetes versions.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "Kubernetes version string (e.g., `1.33`).",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "Expiration date of the version in RFC3339 format.",
						},
						"state": schema.StringAttribute{
							Computed:    true,
							Description: "Version state, such as `supported`, `preview`, or `deprecated`.",
						},
					},
				},
			},
			"machine_types": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of machine types (node sizes) available in the region.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Machine type name (e.g., `c2i.2`).",
						},
						"architecture": schema.StringAttribute{
							Computed:    true,
							Description: "CPU architecture (e.g., `x86_64`, `arm64`).",
						},
						"cpu": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of virtual CPUs.",
						},
						"gpu": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of GPUs included.",
						},
						"memory": schema.Int64Attribute{
							Computed:    true,
							Description: "Memory size in GB.",
						},
					},
				},
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
			"volume_types": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Supported root volume types (e.g., `storage_premium_perf1`).",
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

	// Availability Zones
	azList := make([]types.String, 0)
	if optionsResp.AvailabilityZones != nil {
		for _, az := range *optionsResp.AvailabilityZones {
			if az.Name != nil {
				azList = append(azList, types.StringValue(*az.Name))
			}
		}
	}
	avZones, diags := types.ListValueFrom(ctx, types.StringType, azList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.AvailabilityZones = avZones

	// Volume Types
	volList := make([]types.String, 0)
	if optionsResp.VolumeTypes != nil {
		for _, vt := range *optionsResp.VolumeTypes {
			if vt.Name != nil {
				volList = append(volList, types.StringValue(*vt.Name))
			}
		}
	}
	volTypes, diags := types.ListValueFrom(ctx, types.StringType, volList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.VolumeTypes = volTypes

	// Kubernetes Versions
	kvList := make([]attr.Value, 0)
	if optionsResp.KubernetesVersions != nil {
		for _, kv := range *optionsResp.KubernetesVersions {
			expDate := types.StringNull()
			if kv.ExpirationDate != nil {
				expDate = types.StringValue(kv.ExpirationDate.Format(time.RFC3339))
			}

			obj, diags := types.ObjectValue(kubernetesVersionType, map[string]attr.Value{
				"version":         types.StringPointerValue(kv.Version),
				"state":           types.StringPointerValue(kv.State),
				"expiration_date": expDate,
			})
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			kvList = append(kvList, obj)
		}
	}
	kvs, diags := types.ListValue(types.ObjectType{AttrTypes: kubernetesVersionType}, kvList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.KubernetesVersions = kvs

	// Machine Types
	mtList := make([]attr.Value, 0)
	if optionsResp.MachineTypes != nil {
		for _, mt := range *optionsResp.MachineTypes {
			vals := map[string]attr.Value{
				"name":         types.StringPointerValue(mt.Name),
				"architecture": types.StringPointerValue(mt.Architecture),
				"cpu":          types.Int64PointerValue(mt.Cpu),
				"gpu":          types.Int64PointerValue(mt.Gpu),
				"memory":       types.Int64PointerValue(mt.Memory),
			}
			obj, diags := types.ObjectValue(machineTypeAttributeType, vals)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			mtList = append(mtList, obj)
		}
	}
	mts, diags := types.ListValue(types.ObjectType{AttrTypes: machineTypeAttributeType}, mtList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.MachineTypes = mts

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
