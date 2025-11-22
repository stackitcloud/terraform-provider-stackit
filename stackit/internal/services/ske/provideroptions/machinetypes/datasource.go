package machinetypes

import (
	"context"
	"fmt"
	"net/http"

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
	Region       types.String `tfsdk:"region"`
	MachineTypes types.List   `tfsdk:"machine_types"`
}

var (
	machineTypeAttributeType = map[string]attr.Type{
		"name":         types.StringType,
		"architecture": types.StringType,
		"cpu":          types.Int64Type,
		"gpu":          types.Int64Type,
		"memory":       types.Int64Type,
	}
)

// Ensure implementation satisfies interface
var _ datasource.DataSource = &optionsDataSource{}

// NewKubernetesMachineTypeDataSource creates the data source instance
func NewKubernetesMachineTypeDataSource() datasource.DataSource {
	return &optionsDataSource{}
}

type optionsDataSource struct {
	client       *ske.APIClient
	providerData core.ProviderData
}

// Metadata sets the data source type name
func (d *optionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_machine_types"
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
	description := "Returns a list of supported machine types for the cluster nodes."

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Region override. If omitted, the provider’s region will be used.",
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

func mapFields(_ context.Context, optionsResp *ske.ProviderOptions, model *Model) error {
	if optionsResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

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

	return nil
}
