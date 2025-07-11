package table

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/shared"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasalphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

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
	client       *iaasalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *routingTableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_table"
}

func (d *routingTableDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &d.providerData, features.RoutingTablesExperiment, "stackit_routing_table", core.Datasource, &resp.Diagnostics)
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
func (d *routingTableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Routing table datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.RoutingTablesExperiment, core.Datasource),
		Attributes:          shared.GetDatasourceGetAttributes(),
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *routingTableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model shared.RoutingTableDataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
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

	err = mapDatasourceFields(ctx, routingTableResp, &model, region)
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

func mapDatasourceFields(ctx context.Context, routingTable *iaasalpha.RoutingTable, model *shared.RoutingTableDataSourceModel, region string) error {
	if routingTable == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var routingTableId string
	if model.RoutingTableId.ValueString() != "" {
		routingTableId = model.RoutingTableId.ValueString()
	} else if routingTable.Id != nil {
		routingTableId = *routingTable.Id
	} else {
		return fmt.Errorf("routing table id not present")
	}

	idParts := []string{
		model.OrganizationId.ValueString(),
		region,
		model.NetworkAreaId.ValueString(),
		routingTableId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	err := shared.MapRoutingTableReadModel(ctx, routingTable, &model.RoutingTableReadModel)
	if err != nil {
		return err
	}

	model.Region = types.StringValue(region)
	return nil
}
