package managed_rule_set

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
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/albwaf/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &managedRuleSetDataSource{}
	_ datasource.DataSourceWithConfigure = &managedRuleSetDataSource{}
)

type managedRuleSetDataSource struct {
	client       *albWaf.APIClient
	providerData core.ProviderData
}

func NewManagedRuleSetDataSource() datasource.DataSource {
	return &managedRuleSetDataSource{}
}

func (d *managedRuleSetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "ALB WAF client configured")
}

func (d *managedRuleSetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_waf_managed_rule_set"
}

func (d *managedRuleSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("ALB WAF Managed Rule Set resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`name`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID associated with the ALB WAF Managed Rule Set.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "STACKIT region name the resource is located in. If not defined, the provider region is used.",
				Computed:    true,
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "Managed Rule Set configuration name.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Set the Managed Rule Set type.",
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "Managed Rule Set version.",
				Computed:    true,
			},
			"usage": schema.SingleNestedAttribute{
				Description: "Managed Rule Set usage",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"count": schema.Int32Attribute{
						Description: "Number of WAFs using this Managed Rule Set.",
						Computed:    true,
					},
					"items": schema.ListAttribute{
						Description: "List of WAFs that use this Managed Rule Set.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
			"groups": schema.MapNestedAttribute{
				Description: "Inventory of all available Managed Rule Set groups and their current configuration.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Description: "A description of what this group covers.",
							Computed:    true,
						},
						"group_name": schema.StringAttribute{
							Description: "The name for the rule group.",
							Computed:    true,
						},
						"rules": schema.MapNestedAttribute{
							Description: "Rules of the rule group.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"description": schema.StringAttribute{
										Description: "A description of what this rule does.",
										Computed:    true,
									},
									"mode": schema.StringAttribute{
										Description: "The current mode of the rule.",
										Computed:    true,
									},
									"severity": schema.StringAttribute{
										Description: "Impact level.",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *managedRuleSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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
	ctx = tflog.SetField(ctx, "name", name)
	ctx = tflog.SetField(ctx, "region", region)

	managedRuleSetResp, err := d.client.DefaultAPI.GetManagedRuleSet(ctx, projectId, region, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF Managed Rule Set", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, managedRuleSetResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF Managed Rule Set", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Managed Rule Set read")
}
