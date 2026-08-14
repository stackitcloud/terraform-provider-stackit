package vpcnetworkrange

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

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &vpcNetworkRangeDatasource{}
	_ datasource.DataSourceWithConfigure = &vpcNetworkRangeDatasource{}
)

// NewVpcNetworkRangeDatasource is a helper function to simplify the provider implementation.
func NewVpcNetworkRangeDatasource() datasource.DataSource {
	return &vpcNetworkRangeDatasource{}
}

type DatasourceModel struct {
	SharedModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// vpcNetworkRangeDatasource is the datasource implementation.
type vpcNetworkRangeDatasource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the datasource type name.
func (r *vpcNetworkRangeDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_network_range"
}

// Configure adds the provider configured client to the datasource.
func (r *vpcNetworkRangeDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &r.providerData, features.VpcExperiment, "stackit_vpc_network_range", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.client = iaasAlphaUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "IaaS v2alpha client configured")
}

// Schema defines the schema for the datasource.
func (r *vpcNetworkRangeDatasource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         datasourceDescription,
		MarkdownDescription: descriptions["datasource.markdown"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: descriptions["vpc_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_range_id": schema.StringAttribute{
				Description: descriptions["network_range_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Computed:    true,
			},
			"ip_version": schema.StringAttribute{
				Description: descriptions["ip_version"],
				Computed:    true,
			},
			"default_prefix_length": schema.Int64Attribute{
				Description: descriptions["default_prefix_length"],
				Computed:    true,
			},
			"max_prefix_length": schema.Int64Attribute{
				Description: descriptions["max_prefix_length"],
				Computed:    true,
			},
			"min_prefix_length": schema.Int64Attribute{
				Description: descriptions["min_prefix_length"],
				Computed:    true,
			},
			"nameservers": schema.ListAttribute{
				Description: descriptions["nameservers"],
				ElementType: types.StringType,
				Computed:    true,
			},
			"prefix": schema.StringAttribute{
				Description: descriptions["prefix"],
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				ElementType: types.StringType,
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *vpcNetworkRangeDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DatasourceModel
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

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	networkRangeId := model.NetworkRangeId.ValueString()
	if networkRangeId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_range_id", networkRangeId)

	networkRangeResp, err := r.client.DefaultAPI.GetVPCNetworkRange(ctx, projectId, vpcId, region, networkRangeId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading vpc network range",
			fmt.Sprintf("VPC network range with ID %q does not exist in vpc %q.", networkRangeId, vpcId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q or VPC with ID %q not found or forbidden access", projectId, vpcId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, networkRangeResp, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network range", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC Network range read")
}
