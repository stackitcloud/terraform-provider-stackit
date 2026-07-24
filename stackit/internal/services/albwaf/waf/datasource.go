package waf

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	waf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/albwaf/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &wafDatasource{}
	_ datasource.DataSourceWithConfigure = &wafDatasource{}
)

type wafDatasource struct {
	client       *waf.APIClient
	providerData core.ProviderData
}

func NewWafDatasource() datasource.DataSource {
	return &wafDatasource{}
}

// Configure implements [datasource.DataSourceWithConfigure].
func (d *wafDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) { // nolint:gocritic // function signature required by Terraform
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &d.providerData, &resp.Diagnostics, "stackit_alb_waf", core.Datasource)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "ALB WAF client configured")
}

// Schema implements [datasource.DataSource].
func (d *wafDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: descriptions["main"],
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
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				Optional:    true,
				ElementType: types.StringType,
			},
			"managed_rule_set_name": schema.StringAttribute{
				Description: descriptions["managed_rule_set_name"],
				Optional:    true,
			},
			"custom_rule_group_name": schema.StringAttribute{
				Description: descriptions["custom_rule_group_name"],
				Optional:    true,
			},
			"usage": schema.SingleNestedAttribute{
				Description: descriptions["usage"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"count": schema.Int32Attribute{
						Description: descriptions["count"],
						Computed:    true,
					},
					"items": schema.ListNestedAttribute{
						Description: descriptions["items"],
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"listener_names": schema.ListAttribute{
									Description: descriptions["listener_names"],
									Computed:    true,
									ElementType: types.StringType,
								},
								"load_balancer_name": schema.StringAttribute{
									Description: descriptions["load_balancer_name"],
									Computed:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

// Metadata implements [datasource.DataSource].
func (d *wafDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_waf"
}

// Read implements [datasource.DataSource].
func (d *wafDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)
	if name == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF", "Name must be defined when updating ALB WAF")
		return
	}

	foundWAF, err := d.client.DefaultAPI.GetWAF(ctx, projectId, region, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("ALB WAF with name %q not found in project %q and region %q", name, projectId, region), err.Error())
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, foundWAF, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Managed Rule Set read")
}
