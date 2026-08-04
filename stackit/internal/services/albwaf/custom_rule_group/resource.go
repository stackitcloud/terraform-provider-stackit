package custom_rule_group

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"

	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/albwaf/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &customRuleGroupResource{}
	_ resource.ResourceWithConfigure   = &customRuleGroupResource{}
	_ resource.ResourceWithImportState = &customRuleGroupResource{}
	_ resource.ResourceWithModifyPlan  = &customRuleGroupResource{}

	variableTypeOptions   = sdkUtils.EnumSliceToStringSlice(albWaf.AllowedVariableEnumValues)
	transformationOptions = sdkUtils.EnumSliceToStringSlice(albWaf.AllowedTransformationEnumValues)
	operatorTypeOptions   = sdkUtils.EnumSliceToStringSlice(albWaf.AllowedOperatorEnumValues)
	actionOptions         = sdkUtils.EnumSliceToStringSlice(albWaf.AllowedActionEnumValues)
)

type Model struct {
	Id        types.String `tfsdk:"id"` // needed by TF
	ProjectId types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	Name      types.String `tfsdk:"name"`
	Rules     types.List   `tfsdk:"rules"`
}

type RuleModel struct {
	Behavior    types.Object `tfsdk:"behavior"`
	Conditions  types.List   `tfsdk:"conditions"`
	Description types.String `tfsdk:"description"`
	Id          types.Int32  `tfsdk:"id"`
}

var ruleType = map[string]attr.Type{
	"behavior": types.ObjectType{AttrTypes: behaviorType},
	"conditions": types.ListType{
		ElemType: types.ObjectType{AttrTypes: conditionType},
	},
	"description": types.StringType,
	"id":          types.Int32Type,
}

type BehaviorModel struct {
	Action   types.String `tfsdk:"action"`
	Log      types.Bool   `tfsdk:"log"`
	LogMsg   types.String `tfsdk:"log_msg"`
	Severity types.String `tfsdk:"severity"`
}

var behaviorType = map[string]attr.Type{
	"action":   types.StringType,
	"log":      types.BoolType,
	"log_msg":  types.StringType,
	"severity": types.StringType,
}

type ConditionModel struct {
	Operator        types.Object `tfsdk:"operator"`
	Transformations types.List   `tfsdk:"transformations"`
	Variable        types.Object `tfsdk:"variable"`
}

var conditionType = map[string]attr.Type{
	"operator":        types.ObjectType{AttrTypes: operatorType},
	"transformations": types.ListType{ElemType: types.StringType},
	"variable":        types.ObjectType{AttrTypes: variableType},
}

type OperatorModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

var operatorType = map[string]attr.Type{
	"type":  types.StringType,
	"value": types.StringType,
}

type VariableModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

var variableType = map[string]attr.Type{
	"type":  types.StringType,
	"value": types.StringType,
}

type customRuleGroupResource struct {
	client       *albWaf.APIClient
	providerData core.ProviderData
}

func NewCustomRuleGroupResource() resource.Resource {
	return &customRuleGroupResource{}
}

func (r *customRuleGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_alb_waf_custom_rule_group", core.Resource)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "ALB WAF client configured")
}

func (r *customRuleGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_waf_custom_rule_group"
}

// descriptions for the attributes in the Schema.
var descriptions = map[string]string{
	"id":                "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`name`\".",
	"project_id":        "STACKIT project ID associated with the ALB WAF Custom Rule Group.",
	"region":            "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"name":              "Custom rule group configuration name.",
	"rules":             "Enriched rules containing auto-generated IDs and computed severity values.",
	"rule_behavior":     "Behavior of the rule.",
	"rule_condition":    "Conditions for this rule (order matters, first condition match triggers execution).",
	"rule_description":  "A clear description explaining the threat vector or criteria addressed by this rule.",
	"rule_id":           "Backend auto-allocated unique rule ID within the valid 1-99999 threshold.",
	"behavior_action":   "The protective stance action. ACTION_DENY forces a 403 status response code.",
	"behavior_log":      "Determines whether an entry should be generated in the security ledger upon a rule hit.",
	"behavior_log_msg":  "Custom notification message string mapped to underlying logdata contexts. Required if log is true.",
	"behavior_severity": "Severity classification metric used by internal analytics graphs.",
	"operator":          "The comparison logic executed against the transformed variable.",
	"operator_type":     "The operational evaluation type definition macro.",
	"operator_value":    "The text or rule regex pattern arguments applied inside the operator execution loop.",
	"transformations":   "Ordered normalization steps applied before the operator runs.",
	"variable":          "The part of the HTTP transaction to inspect.",
	"variable_type":     "The targeted validation engine variable macro.",
	"variable_value":    "Optional key element context for map variables (e.g., matching a 'Host' header key).",
}

func (r *customRuleGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: features.AddBetaDescription(fmt.Sprintf("ALB WAF Custom Rule Group resource schema. %s", core.ResourceRegionFallbackDocstring), core.Resource),
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
			"rules": schema.ListNestedAttribute{
				Description: descriptions["rules"],
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"behavior": schema.SingleNestedAttribute{
							Description: descriptions["behavior"],
							Required:    true,
							Attributes: map[string]schema.Attribute{
								"action": schema.StringAttribute{
									Description: descriptions["behavior_action"],
									Required:    true,
									Validators: []validator.String{
										stringvalidator.OneOf(actionOptions...),
									},
								},
								"log": schema.BoolAttribute{
									Description: descriptions["behavior_log"],
									Optional:    true,
								},
								"log_msg": schema.StringAttribute{
									Description: descriptions["behavior_log_msg"],
									Optional:    true,
								},
								"severity": schema.StringAttribute{
									Description: descriptions["behavior_severity"],
									Computed:    true,
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
							},
						},
						"conditions": schema.ListNestedAttribute{
							Description: descriptions["rule_conditions"],
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"operator": schema.SingleNestedAttribute{
										Description: descriptions["operator"],
										Required:    true,
										Attributes: map[string]schema.Attribute{
											"type": schema.StringAttribute{
												Description: descriptions["operator_type"],
												Required:    true,
												Validators: []validator.String{
													stringvalidator.OneOf(operatorTypeOptions...),
												},
											},
											"value": schema.StringAttribute{
												Description: descriptions["operator_value"],
												Optional:    true,
											},
										},
									},
									"transformations": schema.ListAttribute{
										Description: descriptions["transformations"],
										Optional:    true,
										ElementType: types.StringType,
										Validators: []validator.List{
											listvalidator.ValueStringsAre(
												stringvalidator.OneOf(transformationOptions...),
											),
										},
									},
									"variable": schema.SingleNestedAttribute{
										Description: descriptions["variable"],
										Required:    true,
										Attributes: map[string]schema.Attribute{
											"type": schema.StringAttribute{
												Description: descriptions["variable_type"],
												Required:    true,
												Validators: []validator.String{
													stringvalidator.OneOf(variableTypeOptions...),
												},
											},
											"value": schema.StringAttribute{
												Description: descriptions["variable_value"],
												Optional:    true,
											},
										},
									},
								},
							},
						},
						"description": schema.StringAttribute{
							Description: descriptions["rule_description"],
							Optional:    true,
						},
						"id": schema.Int32Attribute{
							Description: descriptions["rule_id"],
							Computed:    true,
							PlanModifiers: []planmodifier.Int32{
								int32planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *customRuleGroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *customRuleGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing ALB WAF Custom Rule Group",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[name]  Got: %q", req.ID),
		)
		return
	}

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"name":       idParts[2],
	})
	tflog.Info(ctx, "ALB WAF Custom Rule Group state imported")
}

func (r *customRuleGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Custom Rule Group", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.DefaultAPI.CreateCustomRuleGroup(ctx, projectId, region).CreateCustomRuleGroupPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Custom Rule Group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp.Name == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Custom Rule Group", "Got empty Custom Rule Group name")
		return
	}
	customRuleGroupName := *createResp.Name

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"name":       customRuleGroupName,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	err = mapFields(ctx, createResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Custom Rule Group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Custom Rule Group created")
}

func (r *customRuleGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	customRuleGroupName := model.Name.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", customRuleGroupName)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Custom Rule Group", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateResp, err := r.client.DefaultAPI.UpdateCustomRuleGroup(ctx, projectId, region, customRuleGroupName).UpdateCustomRuleGroupPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Custom Rule Group", fmt.Sprintf("Calling API update endpoint: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, updateResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Custom Rule Group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "ALB WAF Custom Rule Group update")
}

func (r *customRuleGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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

	customRuleGroupResp, err := r.client.DefaultAPI.GetCustomRuleGroup(ctx, projectId, region, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
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

func (r *customRuleGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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

	_, err := r.client.DefaultAPI.DeleteCustomRuleGroup(ctx, projectId, region, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			tflog.Info(ctx, "ALB WAF Custom Rule Group was already deleted")
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting ALB WAF Custom Rule Group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "ALB WAF Custom Rule Group deleted")
}

func toCreatePayload(ctx context.Context, model *Model) (*albWaf.CreateCustomRuleGroupPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payloadRules, err := toRulesPayload(ctx, model.Rules)
	if err != nil {
		return nil, fmt.Errorf("generating rules payload: %w", err)
	} else if payloadRules == nil {
		return nil, fmt.Errorf("rules can not be empty")
	}

	payload := &albWaf.CreateCustomRuleGroupPayload{
		Name:  model.Name.ValueString(),
		Rules: *payloadRules,
	}

	return payload, nil
}

func toUpdatePayload(ctx context.Context, model *Model) (*albWaf.UpdateCustomRuleGroupPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payloadRules, err := toRulesPayload(ctx, model.Rules)
	if err != nil {
		return nil, fmt.Errorf("generating rules payload: %w", err)
	} else if payloadRules == nil {
		return nil, fmt.Errorf("rules can not be empty")
	}

	payload := &albWaf.UpdateCustomRuleGroupPayload{
		Name:  model.Name.ValueString(),
		Rules: *payloadRules,
	}

	return payload, nil
}

func toRulesPayload(ctx context.Context, modelRules basetypes.ListValue) (*[]albWaf.CreateCustomRule, error) {
	payloadRules := []albWaf.CreateCustomRule{}
	if !tfutils.IsUndefined(modelRules) {
		rules := []RuleModel{}
		diags := modelRules.ElementsAs(ctx, &rules, true)
		if diags.HasError() {
			return nil, fmt.Errorf("converting to rule map: %w", core.DiagsToError(diags))
		}

		for _, rule := range rules {
			behavior := BehaviorModel{}
			if !tfutils.IsUndefined(rule.Behavior) {
				diags := rule.Behavior.As(ctx, &behavior, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("converting to rule behavior: %w", core.DiagsToError(diags))
				}
			}

			conditions, err := toConditionsPayload(ctx, rule.Conditions)
			if err != nil {
				return nil, fmt.Errorf("converting conditions: %w", err)
			} else if conditions == nil {
				return nil, fmt.Errorf("conditions can not be empty")
			}

			payloadRules = append(payloadRules, albWaf.CreateCustomRule{
				Behaviour: albWaf.Behaviour{ // nolint:misspell // Generated from API spec
					Action: albWaf.Action(behavior.Action.ValueString()),
					Log:    behavior.Log.ValueBoolPointer(),
					LogMsg: behavior.LogMsg.ValueStringPointer(),
				},
				Conditions:  *conditions,
				Description: rule.Description.ValueStringPointer(),
			})
		}
	}

	return &payloadRules, nil
}

func toConditionsPayload(ctx context.Context, conditions basetypes.ListValue) (*[]albWaf.Condition, error) {
	result := []albWaf.Condition{}

	if !tfutils.IsUndefined(conditions) {
		conditionModels := []ConditionModel{}
		diags := conditions.ElementsAs(ctx, &conditionModels, true)
		if diags.HasError() {
			return nil, fmt.Errorf("converting to rule map: %w", core.DiagsToError(diags))
		}

		for _, condition := range conditionModels {
			transformations := []albWaf.Transformation{}
			if !tfutils.IsUndefined(condition.Transformations) {
				diags := condition.Transformations.ElementsAs(ctx, &transformations, true)
				if diags.HasError() {
					return nil, fmt.Errorf("converting transformations: %w", core.DiagsToError(diags))
				}
			}

			var operatorModel = OperatorModel{}
			diags = condition.Operator.As(ctx, &operatorModel, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, fmt.Errorf("converting operator: %w", core.DiagsToError(diags))
			}

			var variableModel = VariableModel{}
			diags = condition.Variable.As(ctx, &variableModel, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, fmt.Errorf("converting variable: %w", core.DiagsToError(diags))
			}

			result = append(result, albWaf.Condition{
				Operator: albWaf.ConditionOperator{
					Type:  albWaf.Operator(operatorModel.Type.ValueString()),
					Value: operatorModel.Value.ValueStringPointer(),
				},
				Transformations: transformations,
				Variable: albWaf.ConditionVariable{
					Type:  albWaf.Variable(variableModel.Type.ValueString()),
					Value: variableModel.Value.ValueStringPointer(),
				},
			})
		}
	}

	return &result, nil
}

func mapFields(ctx context.Context, customRuleGroup *albWaf.GetCustomRuleGroupResponse, model *Model, region string) error {
	if customRuleGroup == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = tfutils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, model.Name.ValueString())
	model.Name = types.StringValue(customRuleGroup.GetName())
	model.Region = types.StringValue(region)

	rules, err := mapRules(ctx, &customRuleGroup.Rules)
	if err != nil {
		return fmt.Errorf("map rules: %w", err)
	} else if rules == nil {
		return fmt.Errorf("rules can not be empty")
	}
	model.Rules = *rules

	return nil
}

func mapRules(ctx context.Context, rules *[]albWaf.GetCustomRule) (*basetypes.ListValue, error) {
	var diags diag.Diagnostics
	var result basetypes.ListValue

	if rules != nil {
		rulesList := []attr.Value{}
		for _, rule := range *rules {
			ruleTF := RuleModel{
				Id:          types.Int32PointerValue(rule.Id),
				Description: types.StringPointerValue(rule.Description),
			}

			behavior, err := mapBehavior(ctx, rule.Behaviour) // nolint:misspell // Generated from API spec
			if err != nil {
				return nil, fmt.Errorf("map behavior: %w", err)
			} else if behavior == nil {
				return nil, fmt.Errorf("behavior can not be empty")
			}
			ruleTF.Behavior = *behavior

			conditions, err := mapConditions(ctx, rule)
			if err != nil {
				return nil, fmt.Errorf("map conditions: %w", err)
			} else if conditions == nil {
				return nil, fmt.Errorf("conditions can not be empty")
			}
			ruleTF.Conditions = *conditions

			rule, diags := types.ObjectValueFrom(ctx, ruleType, ruleTF)
			if diags.HasError() {
				return nil, fmt.Errorf("mapping rule: %w", core.DiagsToError(diags))
			}
			rulesList = append(rulesList, rule)
		}
		result, diags = types.ListValue(types.ObjectType{AttrTypes: ruleType}, rulesList)
		if diags.HasError() {
			return nil, fmt.Errorf("creating rule object: %w", core.DiagsToError(diags))
		}
	} else {
		result = types.ListNull(types.ObjectType{AttrTypes: ruleType})
	}

	return &result, nil
}

func mapBehavior(ctx context.Context, behavior *albWaf.GetBehaviour) (*basetypes.ObjectValue, error) {
	var diags diag.Diagnostics
	var result basetypes.ObjectValue

	if behavior != nil {
		behaviorModel := BehaviorModel{
			Action:   types.StringPointerValue((*string)(behavior.Action)),
			Log:      types.BoolPointerValue(behavior.Log),
			LogMsg:   types.StringPointerValue(behavior.LogMsg),
			Severity: types.StringPointerValue((*string)(behavior.Severity)),
		}

		result, diags = types.ObjectValueFrom(ctx, behaviorType, behaviorModel)
		if diags.HasError() {
			return nil, fmt.Errorf("creating behavior object: %w", core.DiagsToError(diags))
		}
	} else {
		result = types.ObjectNull(behaviorType)
	}

	return &result, nil
}

func mapConditions(ctx context.Context, rule albWaf.GetCustomRule) (*basetypes.ListValue, error) {
	var diags diag.Diagnostics
	var result basetypes.ListValue

	if conditions, ok := rule.GetConditionsOk(); ok {
		conditionsList := []attr.Value{}
		for _, condition := range conditions {
			conditionTF := ConditionModel{}

			if operator, ok := condition.GetOperatorOk(); ok {
				operatorModel := OperatorModel{
					Type:  types.StringValue(string(operator.Type)),
					Value: types.StringPointerValue(operator.Value),
				}

				conditionTF.Operator, diags = types.ObjectValueFrom(ctx, operatorType, operatorModel)
				if diags.HasError() {
					return nil, fmt.Errorf("creating operator object: %w", core.DiagsToError(diags))
				}
			} else {
				conditionTF.Operator = types.ObjectNull(operatorType)
			}

			conditionTF.Transformations, diags = types.ListValueFrom(ctx, types.StringType, condition.Transformations)
			if diags.HasError() {
				return nil, fmt.Errorf("mapping transformations: %w", core.DiagsToError(diags))
			}

			if variable, ok := condition.GetVariableOk(); ok {
				variableModel := VariableModel{
					Type:  types.StringValue(string(variable.Type)),
					Value: types.StringPointerValue(variable.Value),
				}

				conditionTF.Variable, diags = types.ObjectValueFrom(ctx, variableType, variableModel)
				if diags.HasError() {
					return nil, fmt.Errorf("creating variable object: %w", core.DiagsToError(diags))
				}
			} else {
				conditionTF.Variable = types.ObjectNull(variableType)
			}

			condition, diags := types.ObjectValueFrom(ctx, conditionType, conditionTF)
			if diags.HasError() {
				return nil, fmt.Errorf("mapping condition: %w", core.DiagsToError(diags))
			}
			conditionsList = append(conditionsList, condition)
		}
		result, diags = types.ListValue(types.ObjectType{AttrTypes: conditionType}, conditionsList)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping conditions: %w", core.DiagsToError(diags))
		}
	} else {
		result = types.ListNull(types.ObjectType{AttrTypes: conditionType})
	}

	return &result, nil
}
