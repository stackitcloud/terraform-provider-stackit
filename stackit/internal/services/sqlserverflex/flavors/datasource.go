package flavors

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	sqlserverflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = new(flavors)
	_ datasource.DataSourceWithConfigure = new(flavors)
)

type model struct {
	ID        types.String   `tfsdk:"id"`
	ProjectId types.String   `tfsdk:"project_id"`
	Region    types.String   `tfsdk:"region"`
	Flavors   []flavor       `tfsdk:"flavors"`
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
}

type flavor struct {
	Id             types.String   `tfsdk:"id"`
	Description    types.String   `tfsdk:"description"`
	CPU            types.Int64    `tfsdk:"cpu"`
	Memory         types.Int64    `tfsdk:"memory"`
	MinGB          types.Int32    `tfsdk:"min_gb"`
	MaxGB          types.Int32    `tfsdk:"max_gb"`
	NodeType       types.String   `tfsdk:"node_type"`
	StorageClasses []storageClass `tfsdk:"storage_classes"`
}

type storageClass struct {
	Class          types.String `tfsdk:"class"`
	MaxIOPerSec    types.Int32  `tfsdk:"max_io_per_sec"`
	MaxThroughInMB types.Int32  `tfsdk:"max_through_in_mb"`
}

type flavors struct {
	client       *sqlserverflex.APIClient
	providerData core.ProviderData
}

func NewFlavorsDataSource() datasource.DataSource {
	return new(flavors)
}

func (f *flavors) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflex_flavors"
}

func (f *flavors) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	f.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := sqlserverflexUtils.ConfigureClient(ctx, &f.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	f.client = apiClient
	tflog.Info(ctx, "SqlserverFlex client configured")
}

func (f *flavors) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "SqlserverFlex flavors data source schema",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source ID, structured as \"`project_id`,`region`\"",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "SqlserverFlex flavors data source region. If undefined the providers region is used.",
				Optional:    true,
				Computed:    true,
			},
			"timeouts": timeouts.Attributes(ctx),
			"flavors": schema.ListNestedAttribute{
				Description: "A list of available flavors.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "ID of the flavor.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the flavor.",
							Computed:    true,
						},
						"cpu": schema.Int64Attribute{
							Description: "The CPU count of the instance.",
							Computed:    true,
						},
						"memory": schema.Int64Attribute{
							Description: "The memory (Gibibyte) of the instance.",
							Computed:    true,
						},
						"min_gb": schema.Int32Attribute{
							Description: "Minimum storage, which is required to order in Gigabyte.",
							Computed:    true,
						},
						"max_gb": schema.Int32Attribute{
							Description: "Maximum storage, which can be ordered for the flavor in Gigabyte.",
							Computed:    true,
						},
						"node_type": schema.StringAttribute{
							Description: "Defines the node type (either single or HA).",
							Computed:    true,
						},
						"storage_classes": schema.ListNestedAttribute{
							Description: "Storage classes available for the flavor.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"class": schema.StringAttribute{
										Description: "Class of the instance.",
										Computed:    true,
									},
									"max_io_per_sec": schema.Int32Attribute{
										Description: "Maximum I/O per second.",
										Computed:    true,
									},
									"max_through_in_mb": schema.Int32Attribute{
										Description: "Maximum throughput in Megabyte.",
										Computed:    true,
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

func (f *flavors) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
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

	projectId := model.ProjectId.ValueString()
	region := f.providerData.GetRegionWithOverride(model.Region)
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
	})

	ctx = core.InitProviderContext(ctx)

	const pageSize = 100
	flavorsResp, err := f.client.DefaultAPI.ListFlavors(ctx, projectId, region).Size(pageSize).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Reading flavors", fmt.Sprintf("Error calling ListFlavors: %v", err))
		return
	}
	if flavorsResp.Pagination.TotalRows > pageSize {
		core.LogAndAddWarning(ctx, &resp.Diagnostics,
			"Truncated results",
			fmt.Sprintf("Due to API limitations we currently do not support more than %d flavors, but %d exist. The result is truncated", pageSize, flavorsResp.Pagination.TotalRows),
		)
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(flavorsResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Reading Flavors", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SqlserverFlex flavor read")
}

func mapFields(resp *sqlserverflex.ListFlavorsResponse, m *model) error {
	if resp == nil {
		return fmt.Errorf("nil response")
	}
	if m == nil {
		return fmt.Errorf("nil model")
	}

	m.ID = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), m.Region.ValueString())

	slices.SortFunc(resp.Flavors, func(a, b sqlserverflex.ListFlavors) int {
		return strings.Compare(a.Id, b.Id)
	})

	for _, respFlavor := range resp.Flavors {
		modelFlavor := flavor{
			Id:          types.StringValue(respFlavor.Id),
			Description: types.StringValue(respFlavor.Description),
			CPU:         types.Int64Value(respFlavor.Cpu),
			Memory:      types.Int64Value(respFlavor.Memory),
			MinGB:       types.Int32Value(respFlavor.MinGB),
			MaxGB:       types.Int32Value(respFlavor.MaxGB),
			NodeType:    types.StringValue(respFlavor.NodeType),
		}

		slices.SortFunc(respFlavor.StorageClasses, func(a, b sqlserverflex.FlavorStorageClassesStorageClass) int {
			return strings.Compare(a.Class, b.Class)
		})

		for _, respSC := range respFlavor.StorageClasses {
			modelSC := storageClass{
				Class:          types.StringValue(respSC.Class),
				MaxIOPerSec:    types.Int32Value(respSC.MaxIoPerSec),
				MaxThroughInMB: types.Int32Value(respSC.MaxThroughInMb),
			}
			modelFlavor.StorageClasses = append(modelFlavor.StorageClasses, modelSC)
		}
		m.Flavors = append(m.Flavors, modelFlavor)
	}
	return nil
}
