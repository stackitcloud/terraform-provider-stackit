package staticroute

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &staticRouteDatasource{}
	_ datasource.DataSourceWithConfigure = &staticRouteDatasource{}
)

type DataSourceModel struct {
	SharedModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewStaticRouteDatasource() datasource.DataSource {
	return &staticRouteDatasource{}
}

type staticRouteDatasource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

func (r *staticRouteDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_routing_table_static_route"
}

func (r *staticRouteDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &r.providerData, features.VpcExperiment, "stackit_vpc_routing_table_static_route", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := iaasAlphaUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "IaaS v2alpha client configured")
}

func (r *staticRouteDatasource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "VPC Routing table static route datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.VpcExperiment, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descId,
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descProjectId,
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: descVpcId,
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"routing_table_id": schema.StringAttribute{
				Description: descRoutingTableId,
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"route_id": schema.StringAttribute{
				Description: descRouteId,
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: descRegion,
				Optional:    true,
				Computed:    true,
			},
			"destination": schema.ObjectAttribute{
				Description:    descDestination,
				Computed:       true,
				AttributeTypes: destinationTypes,
			},
			"nexthop": schema.ObjectAttribute{
				Description:    descNexthop,
				Computed:       true,
				AttributeTypes: nexthopTypes,
			},
			"labels": schema.MapAttribute{
				Description: descLabels,
				ElementType: types.StringType,
				Computed:    true,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

func (r *staticRouteDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
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

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()
	routeId := model.RouteId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "route_id", routeId)

	route, err := r.client.DefaultAPI.GetVPCStaticRoute(ctx, projectId, vpcId, region, routingTableId, routeId).Execute()
	if err != nil {
		utils.LogError(ctx, &resp.Diagnostics, err, "Error reading vpc static route", fmt.Sprintf("Calling API: %v", err),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, route, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading static route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC static route read")
}
