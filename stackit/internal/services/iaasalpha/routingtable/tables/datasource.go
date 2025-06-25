package tables

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/shared"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasalphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &routingTablesDataSource{}
)

type DataSourceModelTables struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	OrganizationId types.String `tfsdk:"organization_id"`
	NetworkAreaId  types.String `tfsdk:"network_area_id"`
	Region         types.String `tfsdk:"region"`
	Items          types.List   `tfsdk:"items"`
}

// NewRoutingTablesDataSource is a helper function to simplify the provider implementation.
func NewRoutingTablesDataSource() datasource.DataSource {
	return &routingTablesDataSource{}
}

// routingTableDataSource is the data source implementation.
type routingTablesDataSource struct {
	client       *iaasalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *routingTablesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_tables"
}

func (d *routingTablesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &d.providerData, &resp.Diagnostics, "stackit_routing_tables", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := iaasalphaUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the data source.
func (d *routingTablesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Routing table datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddBetaDescription(description),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal datasource ID. It is structured as \"`organization_id`,`region`,`network_area_id`\".",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT organization ID to which the routing table is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_area_id": schema.StringAttribute{
				Description: "The network area ID to which the routing table is associated.",
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
			"items": schema.ListNestedAttribute{
				Description: "List of routing tables.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: shared.RoutingTableResponseAttributes(),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *routingTablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModelTables
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	networkAreaId := model.NetworkAreaId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	routingTablesResp, err := d.client.ListRoutingTablesOfArea(ctx, organizationId, networkAreaId, region).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading routing tables",
			fmt.Sprintf("Routing tables with network area with ID %q does not exist in organization %q.", networkAreaId, organizationId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Organization with ID %q not found or forbidden access", organizationId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapDataSourceRoutingTables(ctx, routingTablesResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading routing table", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table read")
}

func mapDataSourceRoutingTables(ctx context.Context, routingTables *iaasalpha.RoutingTableListResponse, model *DataSourceModelTables, region string) error {
	if routingTables == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	if routingTables.Items == nil {
		return fmt.Errorf("items input is nil")
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()

	model.Id = utils.BuildInternalTerraformId(organizationId, region, networkAreaId)

	itemsList := []attr.Value{}
	for i, routingTable := range *routingTables.Items {
		var routingTableModel shared.RoutingTableReadModel
		err := shared.MapRoutingTableReadModel(ctx, &routingTable, &routingTableModel)
		if err != nil {
			return fmt.Errorf("mapping routes: %w", err)
		}

		routingTableMap := map[string]attr.Value{
			"routing_table_id": routingTableModel.RoutingTableId,
			"name":             routingTableModel.Name,
			"description":      routingTableModel.Description,
			"labels":           routingTableModel.Labels,
			"created_at":       routingTableModel.CreatedAt,
			"updated_at":       routingTableModel.UpdatedAt,
			"default":          routingTableModel.Default,
			"system_routes":    routingTableModel.SystemRoutes,
		}

		routingTableTF, diags := types.ObjectValue(shared.RoutingTableReadModelTypes(), routingTableMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}
		itemsList = append(itemsList, routingTableTF)
	}

	itemsListTF, diags := types.ListValue(types.ObjectType{AttrTypes: shared.RoutingTableReadModelTypes()}, itemsList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Items = itemsListTF

	return nil
}
