package vpcroutingtable

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &vpcRoutingTableDatasource{}
	_ datasource.DataSourceWithConfigure = &vpcRoutingTableDatasource{}
)

// NewVpcRoutingTableDatasource is a helper function to simplify the provider implementation.
func NewVpcRoutingTableDatasource() datasource.DataSource {
	return &vpcRoutingTableDatasource{}
}

// vpcRoutingTableDatasource is the datasource implementation.
type vpcRoutingTableDatasource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the datasource type name.
func (r *vpcRoutingTableDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_routing_table"
}

// Configure adds the provider configured client to the datasource.
func (r *vpcRoutingTableDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &r.providerData, features.VpcExperiment, "stackit_vpc_routing_table", core.Datasource, &resp.Diagnostics)
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

func (r *vpcRoutingTableDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "VPC Regional routing table datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.VpcExperiment, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: schemaDescriptions["vpc_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"routing_table_id": schema.StringAttribute{
				Description: schemaDescriptions["routing_table_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: schemaDescriptions["name"],
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(127),
				},
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"labels": schema.MapAttribute{
				Description: schemaDescriptions["labels"],
				ElementType: types.StringType,
				Optional:    true,
			},
			"dynamic_routes": schema.BoolAttribute{
				Description: schemaDescriptions["dynamic_routes"],
				Optional:    true,
				Computed:    true,
			},
			"system_routes": schema.BoolAttribute{
				Description: schemaDescriptions["system_routes"],
				Optional:    true,
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *vpcRoutingTableDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)

	routingTableResp, err := r.client.DefaultAPI.GetVPCRoutingTable(ctx, projectId, vpcId, region, routingTableId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading vpc routing table",
			fmt.Sprintf("vpc routing table with ID %q does not exist in project %q.", routingTableId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, routingTableResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading routing table", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC routing table read")
}
