package routingtables

import (
	"context"
	"fmt"
	"net/http"

	"dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/iaasalpha"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/shared"
	iaasalphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// TODO: add alpha/beta/experimental check

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

var dataSourceModelTablesTypes = map[string]attr.Type{
	"items": types.ObjectType{AttrTypes: shared.DataSourceTypes},
}

// NewRoutingTablesDataSource is a helper function to simplify the provider implementation.
func NewRoutingTablesDataSource() datasource.DataSource {
	return &routingTablesDataSource{}
}

// routingTablesDataSource is the data source implementation.
type routingTablesDataSource struct {
	client       *iaasalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *routingTablesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_tables"
}

func (d *routingTablesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasalphaUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
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
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
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
					Attributes: shared.RoutingTableResponseAttributes,
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *routingTablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model DataSourceModelTables
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	var region string
	if utils.IsUndefined(model.Region) {
		region = d.providerData.GetRegion()
	} else {
		region = model.Region.ValueString()
	}
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

// TODO: extend when routes are implemented
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

	itemsListAttr := []attr.Value{}
	for i, item := range *routingTables.Items {
		dataSourceModel := &shared.DataSourceModel{}
		if err := shared.MapDataSourceFields(ctx, &item, dataSourceModel, region); err != nil {
			return fmt.Errorf("mapping of routing table failed")
		}

		itemsListAttrMap := map[string]attr.Value{
			"routing_table_id":   types.StringValue(dataSourceModel.RoutingTableId.ValueString()),
			"name":               types.StringValue(dataSourceModel.Name.ValueString()),
			"description":        types.StringValue(dataSourceModel.Description.ValueString()),
			"region":             types.StringValue(dataSourceModel.Region.ValueString()),
			"main_routing_table": types.BoolValue(dataSourceModel.MainRoutingTable.ValueBool()),
			"system_routes":      types.BoolValue(dataSourceModel.SystemRoutes.ValueBool()),
			"created_at":         types.StringValue(dataSourceModel.CreatedAt.ValueString()),
			"updated_at":         types.StringValue(dataSourceModel.UpdatedAt.ValueString()),
			"labels":             dataSourceModel.Labels,
			// TODO: extend when routes are implemented
			"routes": types.ListNull(types.StringType),
		}

		itemsListAttrTF, diags := types.ObjectValue(shared.DataSourceTypes, itemsListAttrMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		itemsListAttr = append(itemsListAttr, itemsListAttrTF)
	}

	itemsListTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: shared.DataSourceTypes},
		itemsListAttr,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Items = itemsListTF

	return nil
}
