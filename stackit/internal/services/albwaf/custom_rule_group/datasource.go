package custom_rule_group

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
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/albwaf/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &customRuleGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &customRuleGroupDataSource{}
)

type customRuleGroupDataSource struct {
	client       *albWaf.APIClient
	providerData core.ProviderData
}

func NewCustomRuleGroupDataSource() datasource.DataSource {
	return &customRuleGroupDataSource{}
}

func (r *customRuleGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "ALB WAF client configured")
}

func (r *customRuleGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_waf_custom_rule_group"
}

func (r *customRuleGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("ALB WAF Custom Rule Group resource schema. %s", core.ResourceRegionFallbackDocstring),
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
			"rules": schema.ListNestedAttribute{
				Description: descriptions["rules"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"behavior": schema.SingleNestedAttribute{
							Description: descriptions["behavior"],
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"action": schema.StringAttribute{
									Description: descriptions["behavior_action"],
									Computed:    true,
								},
								"log": schema.BoolAttribute{
									Description: descriptions["behavior_log"],
									Computed:    true,
								},
								"log_msg": schema.StringAttribute{
									Description: descriptions["behavior_log_msg"],
									Computed:    true,
								},
								"severity": schema.StringAttribute{
									Description: descriptions["behavior_severity"],
									Computed:    true,
								},
							},
						},
						"conditions": schema.ListNestedAttribute{
							Description: descriptions["rule_conditions"],
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"operator": schema.SingleNestedAttribute{
										Description: descriptions["operator"],
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"type": schema.StringAttribute{
												Description: descriptions["operator_type"],
												Computed:    true,
											},
											"value": schema.StringAttribute{
												Description: descriptions["operator_value"],
												Computed:    true,
											},
										},
									},
									"transformations": schema.ListAttribute{
										Description: descriptions["transformations"],
										Computed:    true,
										ElementType: types.StringType,
									},
									"variable": schema.SingleNestedAttribute{
										Description: descriptions["variable"],
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"type": schema.StringAttribute{
												Description: descriptions["variable_type"],
												Computed:    true,
											},
											"value": schema.StringAttribute{
												Description: descriptions["variable_value"],
												Computed:    true,
											},
										},
									},
								},
							},
						},
						"description": schema.StringAttribute{
							Description: descriptions["rule_description"],
							Computed:    true,
						},
						"id": schema.Int32Attribute{
							Description: descriptions["rule_id"],
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *customRuleGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	customRuleGroupResp, err := r.client.DefaultAPI.GetCustomRuleGroup(ctx, projectId, region, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("ALB WAF Custom Rule Group with name %q not found in project %q and region %q", name, projectId, region), err.Error())
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF Custom Rule Group", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, customRuleGroupResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF Custom Rule Group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Custom Rule Group read")
}
