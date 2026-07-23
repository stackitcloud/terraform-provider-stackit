package managed_rule_set

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
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

	features.CheckBetaResourcesEnabled(ctx, &d.providerData, &resp.Diagnostics, "stackit_alb_waf_managed_rule_set", core.Datasource)
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

func (d *managedRuleSetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_waf_managed_rule_set"
}

func (d *managedRuleSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: features.AddBetaDescription(fmt.Sprintf("ALB WAF Managed Rule Set DataSource schema. %s", core.DatasourceRegionFallbackDocstring), core.Datasource),
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
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
						"must start and end with an alphanumeric character, may contain hyphens, and be 1-63 characters long",
					),
				},
			},
			"type": schema.StringAttribute{
				Description: descriptions["type"],
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Computed:    true,
			},
			"usage": schema.SingleNestedAttribute{
				Description: descriptions["usage"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"count": schema.Int32Attribute{
						Description: descriptions["usage_count"],
						Computed:    true,
					},
					"items": schema.ListAttribute{
						Description: descriptions["usage_items"],
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
			"groups": schema.MapNestedAttribute{
				Description: descriptions["groups"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Description: descriptions["group_description"],
							Computed:    true,
						},
						"group_name": schema.StringAttribute{
							Description: descriptions["group_name"],
							Computed:    true,
						},
						"rules": schema.MapNestedAttribute{
							Description: descriptions["group_rules"],
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"description": schema.StringAttribute{
										Description: descriptions["rule_description"],
										Computed:    true,
									},
									"mode": schema.StringAttribute{
										Description: descriptions["rule_mode"],
										Computed:    true,
									},
									"severity": schema.StringAttribute{
										Description: descriptions["rule_severity"],
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
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	managedRuleSetResp, err := d.client.DefaultAPI.GetManagedRuleSet(ctx, projectId, region, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("ALB WAF Managed Rule Set with name %q not found in project %q and region %q", name, projectId, region), err.Error())
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
