package routingtable

import (
	"context"
	"fmt"
	"net/http"

	"dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/iaasalpha"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/shared"
	iaasalphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// TODO: add alpha/beta/experimental check

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &routingTableDataSource{}
)

// NewRoutingTableDataSource is a helper function to simplify the provider implementation.
func NewRoutingTableDataSource() datasource.DataSource {
	return &routingTableDataSource{}
}

// routingTableDataSource is the data source implementation.
type routingTableDataSource struct {
	client *iaasalpha.APIClient
}

// Metadata returns the data source type name.
func (d *routingTableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_table"
}

func (d *routingTableDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *routingTableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Routing table datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes:          shared.GetDatasourceGetAttributes(),
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *routingTableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model shared.DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	region := model.Region.ValueString()
	routingTableId := model.RoutingTableId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	routingTableResp, err := d.client.GetRoutingTableOfArea(ctx, organizationId, networkAreaId, region, routingTableId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading routing table",
			fmt.Sprintf("Routing table with ID %q or network area with ID %q does not exist in organization %q.", routingTableId, networkAreaId, organizationId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Organization with ID %q not found or forbidden access", organizationId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = shared.MapDataSourceFields(ctx, routingTableResp, &model, region)
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
