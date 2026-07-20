package managed_rule_set

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"

	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/albwaf/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &managedRuleSetResource{}
	_ resource.ResourceWithConfigure   = &managedRuleSetResource{}
	_ resource.ResourceWithImportState = &managedRuleSetResource{}
	_ resource.ResourceWithModifyPlan  = &managedRuleSetResource{}

	mrsTypeOptions = sdkUtils.EnumSliceToStringSlice(albWaf.AllowedMRSTypeEnumValues)
)

type Model struct {
	Id        types.String `tfsdk:"id"` // needed by TF
	ProjectId types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	Name      types.String `tfsdk:"name"`
	Groups    types.Map    `tfsdk:"groups"`
	Type      types.String `tfsdk:"type"`
	Usage     types.Object `tfsdk:"usage"`
	Version   types.String `tfsdk:"version"`
}

type RuleGroupModel struct {
	Description types.String `tfsdk:"description"`
	GroupName   types.String `tfsdk:"group_name"`
	Rules       types.Map    `tfsdk:"rules"`
}

var ruleGroupType = map[string]attr.Type{
	"description": types.StringType,
	"group_name":  types.StringType,
	"rules": types.MapType{
		ElemType: types.ObjectType{AttrTypes: ruleType},
	},
}

type RuleModel struct {
	Description types.String `tfsdk:"description"`
	Mode        types.String `tfsdk:"mode"`
	Severity    types.String `tfsdk:"severity"`
}

var ruleType = map[string]attr.Type{
	"description": types.StringType,
	"mode":        types.StringType,
	"severity":    types.StringType,
}

type UsageModel struct {
	Count types.Int32 `tfsdk:"count"`
	Items types.List  `tfsdk:"items"`
}

var usageType = map[string]attr.Type{
	"count": types.Int32Type,
	"items": types.ListType{ElemType: types.StringType},
}

type managedRuleSetResource struct {
	client       *albWaf.APIClient
	providerData core.ProviderData
}

func NewManagedRuleSetResource() resource.Resource {
	return &managedRuleSetResource{}
}

func (r *managedRuleSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *managedRuleSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_waf_managed_rule_set"
}

// descriptions for the attributes in the Schema.
var descriptions = map[string]string{
	"id":                "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`name`\".",
	"project_id":        "STACKIT project ID associated with the ALB WAF Managed Rule Set.",
	"region":            "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"name":              "Managed Rule Set configuration name.",
	"type":              "Type of the Managed Rule Set.",
	"version":           "Managed Rule Set version.",
	"usage":             "Managed Rule Set usage",
	"usage_count":       "Number of WAFs using this Managed Rule Set.",
	"usage_items":       "List of WAFs that use this Managed Rule Set.",
	"groups":            "Inventory of all available Managed Rule Set groups and their current configuration.",
	"group_description": "A description of what this group covers.",
	"group_name":        "The name for the rule group.",
	"group_rules":       "Rules of the rule group.",
	"rule_description":  "A description of what this rule does.",
	"rule_mode":         "The current mode of the rule.",
	"rule_severity":     "Impact level.",
}

func (r *managedRuleSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("ALB WAF Managed Rule Set resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
						"must start and end with an alphanumeric character, may contain hyphens, and be 1-63 characters long",
					),
				},
			},
			"type": schema.StringAttribute{
				Description: descriptions["type"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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

func (r *managedRuleSetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tfutils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *managedRuleSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing ALB WAF Managed Rule Set",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[name]  Got: %q", req.ID),
		)
		return
	}

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"name":       idParts[2],
	})
	tflog.Info(ctx, "ALB WAF Managed Rule Set state imported")
}

func (r *managedRuleSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", model.Name)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.DefaultAPI.CreateManagedRuleSet(ctx, projectId, region).CreateManagedRuleSetPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp.Name == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", "Got empty Managed Rule Set name")
		return
	}
	managedRuleSetName := *createResp.Name

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"name":       managedRuleSetName,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	err = mapFields(ctx, createResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Managed Rule Set created")
}

func (r *managedRuleSetResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Ressource not updatable", "alb Managed Rule Set is not updatable")
}

func (r *managedRuleSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
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

	managedRuleSetResp, err := r.client.DefaultAPI.GetManagedRuleSet(ctx, projectId, region, name).Execute()
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

func (r *managedRuleSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
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

	_, err := r.client.DefaultAPI.DeleteManagedRuleSet(ctx, projectId, region, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting ALB WAF Managed Rule Set", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "ALB WAF Managed Rule Set deleted")
}

func toCreatePayload(_ context.Context, model *Model) (*albWaf.CreateManagedRuleSetPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payload := &albWaf.CreateManagedRuleSetPayload{
		Name: model.Name.ValueStringPointer(),
		Type: new(albWaf.MRSType(model.Type.ValueString())),
	}

	return payload, nil
}

func mapFields(ctx context.Context, managedRuleSet *albWaf.GetManagedRuleSetResponse, model *Model, region string) error {
	if managedRuleSet == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var diags diag.Diagnostics

	model.Id = tfutils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, model.Name.ValueString())
	model.Name = types.StringValue(model.Name.ValueString())
	model.Region = types.StringValue(region)

	model.Type = types.StringPointerValue((*string)(managedRuleSet.Type))
	model.Version = types.StringPointerValue(managedRuleSet.Version)

	groupsMap := map[string]attr.Value{}
	if groups, ok := managedRuleSet.GetGroupsOk(); ok {
		for groupKey, group := range *groups {
			groupTF := RuleGroupModel{
				Description: types.StringPointerValue(group.Description),
				GroupName:   types.StringPointerValue(group.GroupName),
			}

			ruleMap := map[string]attr.Value{}
			if rules, ok := group.GetRulesOk(); ok {
				for ruleKey, rule := range *rules {
					ruleTF := RuleModel{
						Description: types.StringPointerValue(rule.Description),
						Mode:        types.StringPointerValue((*string)(rule.Mode)),
						Severity:    types.StringPointerValue(rule.Severity),
					}

					ruleMap[ruleKey], diags = types.ObjectValueFrom(ctx, ruleType, ruleTF)
					if diags.HasError() {
						return fmt.Errorf("mapping role: %w", core.DiagsToError(diags))
					}
				}
			}
			groupTF.Rules, diags = types.MapValue(types.ObjectType{AttrTypes: ruleType}, ruleMap)
			if diags.HasError() {
				return fmt.Errorf("mapping roles: %w", core.DiagsToError(diags))
			}

			groupsMap[groupKey], diags = types.ObjectValueFrom(ctx, ruleGroupType, groupTF)
			if diags.HasError() {
				return fmt.Errorf("mapping group: %w", core.DiagsToError(diags))
			}
		}
	}
	model.Groups, diags = types.MapValue(
		types.ObjectType{AttrTypes: ruleGroupType},
		groupsMap,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping groups: %w", core.DiagsToError(diags))
	}

	if usage, ok := managedRuleSet.GetUsageOk(); ok {
		usageModel := UsageModel{
			Count: types.Int32PointerValue(usage.Count),
		}

		usageModel.Items, diags = types.ListValueFrom(ctx, types.StringType, usage.GetItems())
		if diags.HasError() {
			return fmt.Errorf("creating usage object: %w", core.DiagsToError(diags))
		}

		model.Usage, diags = types.ObjectValueFrom(ctx, usageType, usageModel)
		if diags.HasError() {
			return fmt.Errorf("creating usage object: %w", core.DiagsToError(diags))
		}
	} else {
		model.Usage = types.ObjectNull(usageType)
	}

	return nil
}
