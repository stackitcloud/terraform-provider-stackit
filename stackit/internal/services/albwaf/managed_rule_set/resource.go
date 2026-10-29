package managed_rule_set

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"

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
)

type Model struct {
	Id        types.String `tfsdk:"id"` // needed by TF
	ProjectId types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	Name      types.String `tfsdk:"name"`
	Groups    types.Map    `tfsdk:"groups"`
	Type      types.String `tfsdk:"type"`
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

type managedRuleSetResource struct {
	client       albWaf.DefaultAPI
	providerData core.ProviderData
}

func NewManagedRuleSetResource() resource.Resource {
	return &managedRuleSetResource{}
}

func (r *managedRuleSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.providerData = providerData
	r.client = clients.AlbWafV1CLient

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
	"groups":            "Inventory of all available Managed Rule Set groups and their current configuration.",
	"group_description": "A description of what this group covers.",
	"group_name":        "The name for the rule group.",
	"group_rules":       "Rules of the rule group.",
	"rule_description":  "A description of what this rule does.",
	"rule_mode":         "The current mode of the rule.",
	"rule_severity":     "Impact level.",
	"resource_groups": "Managed Rule Set groups, keyed by group ID (e.g. `911`). Without configuration, this is the inventory of all available groups and their current configuration. " +
		"Configure it to set the mode of individual rules: only the listed groups and rules are managed and kept in the state, and every other rule keeps the mode the API gives it. " +
		"A rule removed from the configuration is set back to `MODE_ENABLED`; removing `groups` entirely does that for all rules it managed.",
	"resource_group_rules": "Rules of the rule group, keyed by rule ID (e.g. `911100`). Required for every configured group.",
	"resource_rule_mode":   "The mode of the rule. Required for every configured rule. " + tfutils.FormatPossibleValues(ruleModeValues...),
}

// ruleModeValues are the modes a rule can be set to through groups.
var ruleModeValues = sdkUtils.EnumSliceToStringSlice(albWaf.AllowedModeEnumValues)

// groupsManagedKey is the private state key that records whether the
// configuration sets groups. If it does, the state holds only the configured
// groups and rules; otherwise it holds the full inventory.
const groupsManagedKey = "groups_managed"

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
			"groups": schema.MapNestedAttribute{
				Description: descriptions["resource_groups"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
				// rules and mode are Optional+Computed rather than Required: a
				// non-computed value in the state makes Terraform plan a null
				// groups when the configuration omits it. AlsoRequires makes
				// them required wherever a group or rule is configured.
				NestedObject: schema.NestedAttributeObject{
					Validators: []validator.Object{
						objectvalidator.AlsoRequires(path.MatchRelative().AtName("rules")),
					},
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
							Description: descriptions["resource_group_rules"],
							Optional:    true,
							Computed:    true,
							Validators: []validator.Map{
								mapvalidator.SizeAtLeast(1),
								mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "must be a numeric rule ID")),
							},
							NestedObject: schema.NestedAttributeObject{
								Validators: []validator.Object{
									objectvalidator.AlsoRequires(path.MatchRelative().AtName("mode")),
								},
								Attributes: map[string]schema.Attribute{
									"description": schema.StringAttribute{
										Description: descriptions["rule_description"],
										Computed:    true,
									},
									"mode": schema.StringAttribute{
										Description: descriptions["resource_rule_mode"],
										Optional:    true,
										Computed:    true,
										Validators: []validator.String{
											stringvalidator.OneOf(ruleModeValues...),
										},
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

	if !req.State.Raw.IsNull() {
		var stateModel Model
		resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
		if !resp.Diagnostics.HasError() {
			utils.WarnIfNameChanges(stateModel.Name, planModel.Name, "Managed Rule Set", &resp.Diagnostics)
			// Without groups in the configuration, Terraform keeps the groups of
			// the state. If those are rules the configuration managed before,
			// plan groups as unknown, so that Update sets them back to
			// MODE_ENABLED and returns the full inventory.
			if configModel.Groups.IsNull() {
				managed, diags := groupsManaged(ctx, req.Private)
				resp.Diagnostics.Append(diags...)
				if managed {
					planModel.Groups = types.MapUnknown(types.ObjectType{AttrTypes: ruleGroupType})
				}
			}
		}
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

	wanted, err := ruleModes(ctx, model.Groups)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Reading rule modes: %v", err))
		return
	}

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.CreateManagedRuleSet(ctx, projectId, region).CreateManagedRuleSetPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"name":       createResp.Name,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	ruleSet := createResp
	if wanted != nil {
		patchPayload, err := toPatchPayload(wanted, nil)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Creating rule modes payload: %v", err))
			return
		}
		patchResp, err := r.client.PatchManagedRuleSet(ctx, projectId, region, model.Name.ValueString()).PatchManagedRuleSetPayload(*patchPayload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Setting rule modes: %v", err))
			return
		}
		ruleSet, err = wantedRules(patchResp, wanted)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Setting rule modes: %v", err))
			return
		}
	}

	err = mapFields(ctx, ruleSet, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Managed Rule Set", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(setGroupsManaged(ctx, resp.Private, wanted != nil)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Managed Rule Set created")
}

// Update applies the rule modes in groups, the only attribute that changes without replacement.
func (r *managedRuleSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var stateModel Model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
	wasManaged, diags := groupsManaged(ctx, req.Private)
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

	wanted, err := ruleModes(ctx, model.Groups)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Managed Rule Set", fmt.Sprintf("Reading rule modes: %v", err))
		return
	}
	// Only rules the configuration managed before go back to MODE_ENABLED,
	// not the inventory a state without configured groups holds.
	var previous map[string]map[string]string
	if wasManaged {
		previous, err = ruleModes(ctx, stateModel.Groups)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Managed Rule Set", fmt.Sprintf("Reading previous rule modes: %v", err))
			return
		}
	}

	patchPayload, err := toPatchPayload(wanted, previous)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Managed Rule Set", fmt.Sprintf("Creating rule modes payload: %v", err))
		return
	}
	patchResp, err := r.client.PatchManagedRuleSet(ctx, projectId, region, name).PatchManagedRuleSetPayload(*patchPayload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Managed Rule Set", fmt.Sprintf("Setting rule modes: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ruleSet, err := wantedRules(patchResp, wanted)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Managed Rule Set", fmt.Sprintf("Setting rule modes: %v", err))
		return
	}

	err = mapFields(ctx, ruleSet, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Managed Rule Set", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(setGroupsManaged(ctx, resp.Private, wanted != nil)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Managed Rule Set updated")
}

func (r *managedRuleSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	managed, diags := groupsManaged(ctx, req.Private)
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

	managedRuleSetResp, err := r.client.GetManagedRuleSet(ctx, projectId, region, name).Execute()
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

	// With configured groups, refresh only the configured rules, so that a
	// mode changed outside Terraform shows as drift; a rule that no longer
	// exists is dropped.
	if managed && managedRuleSetResp != nil {
		keep, err := ruleModes(ctx, model.Groups)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF Managed Rule Set", fmt.Sprintf("Reading rule modes: %v", err))
			return
		}
		if keep != nil {
			managedRuleSetResp, _ = filterRuleSet(managedRuleSetResp, keep)
		}
	}

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

	_, err := r.client.DeleteManagedRuleSet(ctx, projectId, region, name).Execute()
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
		Name: model.Name.ValueString(),
		Type: albWaf.Type(model.Type.ValueString()),
	}

	return payload, nil
}

// privateStateGetter and privateStateSetter are the parts of the framework's
// private state that groupsManaged and setGroupsManaged use.
type privateStateGetter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

type privateStateSetter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// groupsManaged reports whether groups was configured at the last apply.
func groupsManaged(ctx context.Context, private privateStateGetter) (bool, diag.Diagnostics) {
	value, diags := private.GetKey(ctx, groupsManagedKey)
	return string(value) == "true", diags
}

// setGroupsManaged records whether groups is configured.
func setGroupsManaged(ctx context.Context, private privateStateSetter, managed bool) diag.Diagnostics {
	value := "false"
	if managed {
		value = "true"
	}
	return private.SetKey(ctx, groupsManagedKey, []byte(value))
}

// ruleModes returns the mode of every rule in groups, keyed by group ID and
// rule ID, or nil if groups is null or unknown.
func ruleModes(ctx context.Context, groups types.Map) (map[string]map[string]string, error) {
	if tfutils.IsUndefined(groups) {
		return nil, nil
	}
	var groupModels map[string]RuleGroupModel
	if diags := groups.ElementsAs(ctx, &groupModels, false); diags.HasError() {
		return nil, fmt.Errorf("reading groups: %w", core.DiagsToError(diags))
	}
	modes := map[string]map[string]string{}
	for groupKey, group := range groupModels {
		var rules map[string]RuleModel
		if diags := group.Rules.ElementsAs(ctx, &rules, false); diags.HasError() {
			return nil, fmt.Errorf("reading rules of group %q: %w", groupKey, core.DiagsToError(diags))
		}
		modes[groupKey] = map[string]string{}
		for ruleKey, rule := range rules {
			modes[groupKey][ruleKey] = rule.Mode.ValueString()
		}
	}
	return modes, nil
}

// wantedRules returns the rule set reduced to the wanted rules, or the full
// inventory if wanted is nil. A wanted rule that the rule set lacks is an error.
func wantedRules(ruleSet *albWaf.GetManagedRuleSetResponse, wanted map[string]map[string]string) (*albWaf.GetManagedRuleSetResponse, error) {
	if ruleSet == nil {
		return nil, fmt.Errorf("response input is nil")
	}
	if wanted == nil {
		return ruleSet, nil
	}
	filtered, missing := filterRuleSet(ruleSet, wanted)
	if len(missing) > 0 {
		return nil, fmt.Errorf("rules %s do not exist in managed rule set %q (version %s)", strings.Join(missing, ", "), ruleSet.Name, ruleSet.Version)
	}
	return filtered, nil
}

// toPatchPayload builds the PATCH that sets every wanted rule to its mode and
// every rule only in previous back to MODE_ENABLED.
func toPatchPayload(wanted, previous map[string]map[string]string) (*albWaf.PatchManagedRuleSetPayload, error) {
	groups := map[string]albWaf.PatchMRSRuleGroup{}
	setMode := func(groupKey, ruleKey, value string) error {
		mode, err := albWaf.NewModeFromValue(value)
		if err != nil {
			return fmt.Errorf("group %q, rule %q: %w", groupKey, ruleKey, err)
		}
		group, ok := groups[groupKey]
		if !ok {
			group = albWaf.PatchMRSRuleGroup{Rules: map[string]albWaf.PatchMRSRule{}}
			groups[groupKey] = group
		}
		group.Rules[ruleKey] = albWaf.PatchMRSRule{Mode: mode}
		return nil
	}

	for groupKey, rules := range previous {
		for ruleKey := range rules {
			if _, ok := wanted[groupKey][ruleKey]; ok {
				continue
			}
			if err := setMode(groupKey, ruleKey, string(albWaf.MODE_MODE_ENABLED)); err != nil {
				return nil, err
			}
		}
	}
	for groupKey, rules := range wanted {
		for ruleKey, mode := range rules {
			if err := setMode(groupKey, ruleKey, mode); err != nil {
				return nil, err
			}
		}
	}

	return albWaf.NewPatchManagedRuleSetPayload(groups), nil
}

// filterRuleSet returns a copy of the rule set that holds only the groups and
// rules in keep, and the rules in keep that the rule set lacks, as
// "group/rule".
func filterRuleSet(managedRuleSet *albWaf.GetManagedRuleSetResponse, keep map[string]map[string]string) (filtered *albWaf.GetManagedRuleSetResponse, missing []string) {
	ruleSetCopy := *managedRuleSet
	allGroups := managedRuleSet.GetGroups()
	groups := map[string]albWaf.MRSRuleGroup{}
	for groupKey, ruleKeys := range keep {
		group, groupFound := allGroups[groupKey]
		allRules := group.GetRules()
		rules := map[string]albWaf.MRSRule{}
		for ruleKey := range ruleKeys {
			rule, ok := allRules[ruleKey]
			if !ok {
				missing = append(missing, fmt.Sprintf("%s/%s", groupKey, ruleKey))
				continue
			}
			rules[ruleKey] = rule
		}
		if groupFound {
			group.Rules = &rules
			groups[groupKey] = group
		}
	}
	ruleSetCopy.Groups = &groups
	slices.Sort(missing)
	return &ruleSetCopy, missing
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

	model.Type = types.StringValue(string(managedRuleSet.Type))
	model.Version = types.StringValue(managedRuleSet.Version)

	groupsMap := map[string]attr.Value{}
	if groups, ok := managedRuleSet.GetGroupsOk(); ok {
		for groupKey, group := range *groups {
			groupTF := RuleGroupModel{
				Description: types.StringValue(group.Description),
				GroupName:   types.StringValue(group.GroupName),
			}

			ruleMap := map[string]attr.Value{}
			if rules, ok := group.GetRulesOk(); ok {
				for ruleKey, rule := range *rules {
					ruleTF := RuleModel{
						Description: types.StringValue(rule.Description),
						Mode:        types.StringValue(string(rule.Mode)),
						Severity:    types.StringValue(rule.Severity),
					}

					ruleMap[ruleKey], diags = types.ObjectValueFrom(ctx, ruleType, ruleTF)
					if diags.HasError() {
						return fmt.Errorf("mapping rule: %w", core.DiagsToError(diags))
					}
				}
			}
			groupTF.Rules, diags = types.MapValue(types.ObjectType{AttrTypes: ruleType}, ruleMap)
			if diags.HasError() {
				return fmt.Errorf("mapping rules: %w", core.DiagsToError(diags))
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

	return nil
}
