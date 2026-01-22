// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: Apache-2.0

package sqlserverFlexAlphaFlavor

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"

	sqlserverflexalphaGen "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/sqlserverflexalpha/flavor/datasources_gen"
	sqlserverflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/sqlserverflexalpha/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &flavorDataSource{}
	_ datasource.DataSourceWithConfigure = &flavorDataSource{}
)

type FlavorModel struct {
	ProjectId      types.String `tfsdk:"project_id"`
	Region         types.String `tfsdk:"region"`
	StorageClass   types.String `tfsdk:"storage_class"`
	Cpu            types.Int64  `tfsdk:"cpu"`
	Description    types.String `tfsdk:"description"`
	Id             types.String `tfsdk:"id"`
	FlavorId       types.String `tfsdk:"flavor_id"`
	MaxGb          types.Int64  `tfsdk:"max_gb"`
	Memory         types.Int64  `tfsdk:"ram"`
	MinGb          types.Int64  `tfsdk:"min_gb"`
	NodeType       types.String `tfsdk:"node_type"`
	StorageClasses types.List   `tfsdk:"storage_classes"`
}

// NewFlavorDataSource is a helper function to simplify the provider implementation.
func NewFlavorDataSource() datasource.DataSource {
	return &flavorDataSource{}
}

// flavorDataSource is the data source implementation.
type flavorDataSource struct {
	client       *sqlserverflexalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *flavorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflexalpha_flavor"
}

// Configure adds the provider configured client to the data source.
func (r *flavorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := sqlserverflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Postgres Flex instance client configured")
}

func (r *flavorDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required:            true,
				Description:         "The cpu count of the instance.",
				MarkdownDescription: "The cpu count of the instance.",
			},
			"region": schema.StringAttribute{
				Required:            true,
				Description:         "The flavor description.",
				MarkdownDescription: "The flavor description.",
			},
			"cpu": schema.Int64Attribute{
				Required:            true,
				Description:         "The cpu count of the instance.",
				MarkdownDescription: "The cpu count of the instance.",
			},
			"ram": schema.Int64Attribute{
				Required:            true,
				Description:         "The memory of the instance in Gibibyte.",
				MarkdownDescription: "The memory of the instance in Gibibyte.",
			},
			"storage_class": schema.StringAttribute{
				Required:            true,
				Description:         "The memory of the instance in Gibibyte.",
				MarkdownDescription: "The memory of the instance in Gibibyte.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				Description:         "The flavor description.",
				MarkdownDescription: "The flavor description.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "The terraform id of the instance flavor.",
				MarkdownDescription: "The terraform id of the instance flavor.",
			},
			"flavor_id": schema.StringAttribute{
				Computed:            true,
				Description:         "The flavor id of the instance flavor.",
				MarkdownDescription: "The flavor id of the instance flavor.",
			},
			"max_gb": schema.Int64Attribute{
				Computed:            true,
				Description:         "maximum storage which can be ordered for the flavor in Gigabyte.",
				MarkdownDescription: "maximum storage which can be ordered for the flavor in Gigabyte.",
			},
			"min_gb": schema.Int64Attribute{
				Computed:            true,
				Description:         "minimum storage which is required to order in Gigabyte.",
				MarkdownDescription: "minimum storage which is required to order in Gigabyte.",
			},
			"node_type": schema.StringAttribute{
				Required:            true,
				Description:         "defines the nodeType it can be either single or replica",
				MarkdownDescription: "defines the nodeType it can be either single or replica",
			},
			"storage_classes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"class": schema.StringAttribute{
							Computed: true,
						},
						"max_io_per_sec": schema.Int64Attribute{
							Computed: true,
						},
						"max_through_in_mb": schema.Int64Attribute{
							Computed: true,
						},
					},
					CustomType: sqlserverflexalphaGen.StorageClassesType{
						ObjectType: types.ObjectType{
							AttrTypes: sqlserverflexalphaGen.StorageClassesValue{}.AttributeTypes(ctx),
						},
					},
				},
			},
		},
	}
}

func (r *flavorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model FlavorModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	flavors, err := getAllFlavors(ctx, r.client, projectId, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading flavors", fmt.Sprintf("getAllFlavors: %v", err))
		return
	}

	var foundFlavors []sqlserverflexalpha.ListFlavors
	for _, flavor := range flavors {
		if model.Cpu.ValueInt64() != *flavor.Cpu {
			continue
		}
		if model.Memory.ValueInt64() != *flavor.Memory {
			continue
		}
		if model.NodeType.ValueString() != *flavor.NodeType {
			continue
		}
		for _, sc := range *flavor.StorageClasses {
			if model.StorageClass.ValueString() != *sc.Class {
				continue
			}
			foundFlavors = append(foundFlavors, flavor)
		}
	}
	if len(foundFlavors) == 0 {
		resp.Diagnostics.AddError("get flavor", "could not find requested flavor")
		return
	}
	if len(foundFlavors) > 1 {
		resp.Diagnostics.AddError("get flavor", "found too many matching flavors")
		return
	}

	f := foundFlavors[0]
	model.Description = types.StringValue(*f.Description)
	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, *f.Id)
	model.FlavorId = types.StringValue(*f.Id)
	model.MaxGb = types.Int64Value(*f.MaxGB)
	model.MinGb = types.Int64Value(*f.MinGB)

	if f.StorageClasses == nil {
		model.StorageClasses = types.ListNull(sqlserverflexalphaGen.StorageClassesType{
			ObjectType: basetypes.ObjectType{
				AttrTypes: sqlserverflexalphaGen.StorageClassesValue{}.AttributeTypes(ctx),
			},
		})
	} else {
		var scList []attr.Value
		for _, sc := range *f.StorageClasses {
			scList = append(
				scList,
				sqlserverflexalphaGen.NewStorageClassesValueMust(
					sqlserverflexalphaGen.StorageClassesValue{}.AttributeTypes(ctx),
					map[string]attr.Value{
						"class":             types.StringValue(*sc.Class),
						"max_io_per_sec":    types.Int64Value(*sc.MaxIoPerSec),
						"max_through_in_mb": types.Int64Value(*sc.MaxThroughInMb),
					},
				),
			)
		}
		storageClassesList := types.ListValueMust(
			sqlserverflexalphaGen.StorageClassesType{
				ObjectType: basetypes.ObjectType{
					AttrTypes: sqlserverflexalphaGen.StorageClassesValue{}.AttributeTypes(ctx),
				},
			},
			scList,
		)
		model.StorageClasses = storageClassesList
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex flavors read")
}
