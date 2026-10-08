package machinetypes

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
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
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &machineTypesDataSource{}
	_ datasource.DataSourceWithConfigure = &machineTypesDataSource{}
)

type filterModel struct {
	Name       types.String `tfsdk:"name"`
	Disk       types.Int64  `tfsdk:"disk"`
	RAM        types.Int64  `tfsdk:"ram"`
	VCPU       types.Int64  `tfsdk:"vcpu"`
	ExtraSpecs types.Map    `tfsdk:"extra_specs"`
}

type machineTypeModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Disk        types.Int64  `tfsdk:"disk"`
	RAM         types.Int64  `tfsdk:"ram"`
	VCPUs       types.Int64  `tfsdk:"vcpus"`
	ExtraSpecs  types.Map    `tfsdk:"extra_specs"`
}

type DataSourceModel struct {
	ID        types.String   `tfsdk:"id"`
	ProjectID types.String   `tfsdk:"project_id"`
	Region    types.String   `tfsdk:"region"`
	Filter    types.Object   `tfsdk:"filter"`
	Results   types.List     `tfsdk:"results"`
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
}

type machineTypesDataSource struct {
	client       iaas.DefaultAPI
	providerData core.ProviderData
}

var machineTypeAttrTypes = map[string]attr.Type{
	"name":        types.StringType,
	"description": types.StringType,
	"disk":        types.Int64Type,
	"ram":         types.Int64Type,
	"vcpus":       types.Int64Type,
	"extra_specs": types.MapType{ElemType: types.StringType},
}

func NewMachineTypesDataSource() datasource.DataSource {
	return &machineTypesDataSource{}
}

func (d *machineTypesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine_types"
}

func (d *machineTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.providerData = providerData
	d.client = clients.IaaSv2Client

	features.CheckBetaResourcesEnabled(ctx, &d.providerData, &resp.Diagnostics, "stackit_machine_types", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "IaaS client configured")
}

func (d *machineTypesDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := features.AddBetaDescription(
		"Lists all available machine types in a project and region. Results are sorted by name.", core.Datasource,
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
				Description: "Experimental API-side equality filters, which may be subject to breaking changes. All configured attributes and extra spec entries are combined with AND. Omit this object or use an empty object to list all machine types.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "Exact machine type name to match.",
						Optional:    true,
					},
					"disk": schema.Int64Attribute{
						Description: "Exact disk size in GB to match.",
						Optional:    true,
					},
					"ram": schema.Int64Attribute{
						Description: "Exact RAM size in MB to match.",
						Optional:    true,
					},
					"vcpu": schema.Int64Attribute{
						Description: "Exact number of vCPUs to match.",
						Optional:    true,
					},
					"extra_specs": schema.MapAttribute{
						Description: "Extra specs to match by key and exact value (e.g., cpu or overcommit).",
						ElementType: types.StringType,
						Optional:    true,
					},
				},
			},
			"results": schema.ListNestedAttribute{
				Description: "All matching machine types, sorted by name ascending. Empty if no machine types match.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Machine type name.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Machine type description.",
							Computed:    true,
						},
						"disk": schema.Int64Attribute{
							Description: "Disk size in GB.",
							Computed:    true,
						},
						"ram": schema.Int64Attribute{
							Description: "RAM size in MB.",
							Computed:    true,
						},
						"vcpus": schema.Int64Attribute{
							Description: "Number of vCPUs.",
							Computed:    true,
						},
						"extra_specs": schema.MapAttribute{
							Description: "Extra specs (e.g., CPU type or overcommit ratio).",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

// nolint:gocritic // framework signature required
func (d *machineTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading machine types", fmt.Sprintf("Creating API request: %v", err))
		return
	}

	apiResponse, err := request.Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading machine types", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if err = mapFields(ctx, apiResponse, &model, region); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading machine types", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Successfully read machine types")
}

// buildFilter translates typed equality filters to an expr-lang expression.
// Quoting both map keys and string values prevents them from becoming expressions.
func buildFilter(ctx context.Context, value types.Object) (string, error) {
	if value.IsNull() || value.IsUnknown() {
		return "", nil
	}

	var filter filterModel
	if diags := value.As(ctx, &filter, basetypes.ObjectAsOptions{}); diags.HasError() {
		return "", fmt.Errorf("converting filter: %w", core.DiagsToError(diags))
	}

	var conditions []string
	if !filter.Name.IsNull() && !filter.Name.IsUnknown() {
		conditions = append(conditions, "name == "+strconv.Quote(filter.Name.ValueString()))
	}

	for _, field := range []struct {
		name  string
		value types.Int64
	}{
		{"disk", filter.Disk},
		{"ram", filter.RAM},
		{"vcpus", filter.VCPU},
	} {
		if !field.value.IsNull() && !field.value.IsUnknown() {
			conditions = append(conditions, fmt.Sprintf("%s == %d", field.name, field.value.ValueInt64()))
		}
	}

	if !filter.ExtraSpecs.IsNull() && !filter.ExtraSpecs.IsUnknown() {
		entries := filter.ExtraSpecs.Elements()
		for _, key := range slices.Sorted(maps.Keys(entries)) {
			value, ok := entries[key].(types.String)
			if !ok || value.IsNull() || value.IsUnknown() {
				return "", fmt.Errorf("extra spec %q must have a known, non-null string value", key)
			}

			conditions = append(conditions, fmt.Sprintf("extraSpecs[%s] == %s", strconv.Quote(key), strconv.Quote(value.ValueString())))
		}
	}

	return strings.Join(conditions, " && "), nil
}

func toRequest(ctx context.Context, client iaas.DefaultAPI, model *DataSourceModel, providerData *core.ProviderData) (iaas.ApiListMachineTypesRequest, error) {
	if model == nil {
		return iaas.ApiListMachineTypesRequest{}, fmt.Errorf("data source model is nil")
	}

	filter, err := buildFilter(ctx, model.Filter)
	if err != nil {
		return iaas.ApiListMachineTypesRequest{}, err
	}

	request := client.ListMachineTypes(ctx, model.ProjectID.ValueString(), providerData.GetRegionWithOverride(model.Region))
	if filter != "" {
		request = request.Filter(filter)
	}

	return request, nil
}

func mapFields(ctx context.Context, response *iaas.MachineTypeListResponse, model *DataSourceModel, region string) error {
	if response == nil {
		return fmt.Errorf("machine type list response is nil")
	}
	if model == nil {
		return fmt.Errorf("data source model is nil")
	}

	items := response.Items
	for i := range items {
		if items[i].Name == "" {
			return fmt.Errorf("machine type at index %d has no name", i)
		}
	}

	values := make([]machineTypeModel, 0, len(items))
	for i := range items {
		item := items[i]

		extra := types.MapNull(types.StringType)
		if len(item.ExtraSpecs) > 0 {
			mapped, ds := types.MapValueFrom(ctx, types.StringType, item.ExtraSpecs)
			if ds.HasError() {
				return fmt.Errorf("mapping extra specs for machine type %q: %w", item.Name, core.DiagsToError(ds))
			}
			extra = mapped
		}

		values = append(values, machineTypeModel{
			Name:        types.StringValue(item.Name),
			Description: types.StringPointerValue(item.Description),
			Disk:        types.Int64Value(item.Disk),
			RAM:         types.Int64Value(item.Ram),
			VCPUs:       types.Int64Value(item.Vcpus),
			ExtraSpecs:  extra,
		})
	}

	slices.SortFunc(values, func(a, b machineTypeModel) int { return strings.Compare(a.Name.ValueString(), b.Name.ValueString()) })

	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: machineTypeAttrTypes}, values)
	if diags.HasError() {
		return fmt.Errorf("converting machine type results: %w", core.DiagsToError(diags))
	}

	model.Results = list
	model.Region = types.StringValue(region)
	model.ID = utils.BuildInternalTerraformId(model.ProjectID.ValueString(), region)

	return nil
}
