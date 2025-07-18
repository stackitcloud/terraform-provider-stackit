package machineType

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var _ datasource.DataSource = &machineTypeDataSource{}

type DataSourceModel struct {
	Id            types.String `tfsdk:"id"` // required by Terraform to identify state
	ProjectId     types.String `tfsdk:"project_id"`
	SortAscending types.Bool   `tfsdk:"sort_ascending"`
	Filter        types.String `tfsdk:"filter"`
	Description   types.String `tfsdk:"description"`
	Disk          types.Int64  `tfsdk:"disk"`
	ExtraSpecs    types.Map    `tfsdk:"extra_specs"`
	Name          types.String `tfsdk:"name"`
	Ram           types.Int64  `tfsdk:"ram"`
	Vcpus         types.Int64  `tfsdk:"vcpus"`
}

// NewMachineTypeDataSource instantiates the data source
func NewMachineTypeDataSource() datasource.DataSource {
	return &machineTypeDataSource{}
}

type machineTypeDataSource struct {
	client *iaas.APIClient
}

func (m *machineTypeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine_type"
}

func (m *machineTypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	client := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	m.client = client

	tflog.Info(ctx, "IAAS client configured")
}

func (m *machineTypeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Machine type data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`image_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT Project ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"sort_ascending": schema.BoolAttribute{
				Description: "Sort machine types by name ascending (`true`) or descending (`false`). Defaults to `false`",
				Optional:    true,
			},
			"filter": schema.StringAttribute{
				Description: `Expr-lang filter for filtering machine types.

Examples:
- vcpus == 2
- ram >= 2048
- extraSpecs.cpu == "intel-icelake-generic"
- extraSpecs.cpu == "intel-icelake-generic" && vcpus == 2

See https://expr-lang.org/docs/language-definition for syntax.`,
				Required: true,
			},
			"description": schema.StringAttribute{
				Description: "Machine type description.",
				Computed:    true,
			},
			"disk": schema.Int64Attribute{
				Description: "Disk size in GB.",
				Computed:    true,
			},
			"extra_specs": schema.MapAttribute{
				Description: "Extra specs (e.g., CPU type, overcommit ratio).",
				ElementType: types.StringType,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the machine type (e.g. 's1.2').",
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
		},
	}
}

func (m *machineTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	sortAscending := model.SortAscending.ValueBool()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "filter_is_null", model.Filter.IsNull())
	ctx = tflog.SetField(ctx, "filter_is_unknown", model.Filter.IsUnknown())

	listMachineTypeReq := m.client.ListMachineTypes(ctx, projectId)

	if !model.Filter.IsNull() && !model.Filter.IsUnknown() {
		if filter := strings.TrimSpace(model.Filter.ValueString()); filter != "" {
			listMachineTypeReq = listMachineTypeReq.Filter(filter)
		}
	}

	apiResp, err := listMachineTypeReq.Execute()
	if err != nil {
		utils.LogError(ctx, &resp.Diagnostics, err, "Failed to read machine types",
			fmt.Sprintf("Unable to retrieve machine types for project %q %s.", projectId, err),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Access denied to project %q.", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	items := apiResp.Items
	if items == nil || len(*items) == 0 {
		core.LogAndAddWarning(ctx, &resp.Diagnostics, "No machine types found", "No matching machine types.")
		return
	}

	// Convert items to []*iaas.MachineType
	machineTypes := make([]*iaas.MachineType, len(*items))
	for i := range *items {
		machineTypes[i] = &(*items)[i]
	}

	sorted, err := sortMachineTypeByName(machineTypes, sortAscending)
	if err != nil {
		core.LogAndAddWarning(ctx, &resp.Diagnostics, "Unable to sort", err.Error())
		return
	}

	first := sorted[0]
	if err := mapDataSourceFields(ctx, first, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Mapping error", fmt.Sprintf("Failed to translate API response: %v", err))
		return
	}

	if err := mapDataSourceFields(ctx, first, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Mapping error", fmt.Sprintf("Failed to translate API response: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Info(ctx, "Successfully read machine type")
}

func mapDataSourceFields(ctx context.Context, machineType *iaas.MachineType, model *DataSourceModel) error {
	if machineType == nil || model == nil {
		return fmt.Errorf("nil input provided")
	}

	if machineType.Name == nil || *machineType.Name == "" {
		return fmt.Errorf("machine type name is missing")
	}
	name := *machineType.Name

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), name)
	model.Name = types.StringPointerValue(machineType.Name)
	model.Description = types.StringPointerValue(machineType.Description)
	model.Disk = types.Int64PointerValue(machineType.Disk)
	model.Ram = types.Int64PointerValue(machineType.Ram)
	model.Vcpus = types.Int64PointerValue(machineType.Vcpus)

	extra := types.MapNull(types.StringType)
	if machineType.ExtraSpecs != nil && len(*machineType.ExtraSpecs) > 0 {
		var diags diag.Diagnostics
		extra, diags = types.MapValueFrom(ctx, types.StringType, *machineType.ExtraSpecs)
		if diags.HasError() {
			return fmt.Errorf("converting extraspecs: %w", core.DiagsToError(diags))
		}
	}
	model.ExtraSpecs = extra
	return nil
}

func sortMachineTypeByName(input []*iaas.MachineType, ascending bool) ([]*iaas.MachineType, error) {
	if input == nil {
		return nil, fmt.Errorf("input slice is nil")
	}

	// Filter out nil or missing name
	filtered := make([]*iaas.MachineType, 0)
	for _, m := range input {
		if m != nil && m.Name != nil {
			filtered = append(filtered, m)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if ascending {
			return *filtered[i].Name < *filtered[j].Name
		}
		return *filtered[i].Name > *filtered[j].Name
	})

	return filtered, nil
}
