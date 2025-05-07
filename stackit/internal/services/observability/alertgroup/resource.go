package alertgroup

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &alertGroupResource{}
	_ resource.ResourceWithConfigure   = &alertGroupResource{}
	_ resource.ResourceWithImportState = &alertGroupResource{}
)

type Model struct {
	Id         types.String `tfsdk:"id"`
	ProjectId  types.String `tfsdk:"project_id"`
	InstanceId types.String `tfsdk:"instance_id"`
	Name       types.String `tfsdk:"name"`
	Interval   types.String `tfsdk:"interval"`
	Rules      types.List   `tfsdk:"rules"`
}

type rule struct {
	Alert       types.String `tfsdk:"alert"`
	Annotations types.Map    `tfsdk:"annotations"`
	Labels      types.Map    `tfsdk:"labels"`
	Expression  types.String `tfsdk:"expression"`
	For         types.String `tfsdk:"for"`
}

var ruleTypes = map[string]attr.Type{
	"alert":       basetypes.StringType{},
	"annotations": basetypes.MapType{ElemType: types.StringType},
	"labels":      basetypes.MapType{ElemType: types.StringType},
	"expression":  basetypes.StringType{},
	"for":         basetypes.StringType{},
}

// Descriptions for the resource and data source schemas are centralized here.
var descriptions = map[string]string{
	"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`,`name`\".",
	"project_id":  "STACKIT project ID to which the alert group is associated.",
	"instance_id": "Observability instance ID to which the alert group is associated.",
	"name":        "The name of the alert group. Is the identifier and must be unique in the group.",
	"interval":    "Specifies the frequency at which rules within the group are evaluated. The interval must be at least 60 seconds and defaults to 60 seconds if not set. Supported formats include hours, minutes, and seconds, either singly or in combination. Examples of valid formats are: '5h30m40s', '5h', '5h30m', '60m', and '60s'.",
	"alert":       "The name of the alert rule. Is the identifier and must be unique in the group.",
	"expression":  "The PromQL expression to evaluate. Every evaluation cycle this is evaluated at the current time, and all resultant time series become pending/firing alerts.",
	"for":         "Alerts are considered firing once they have been returned for this long. Alerts which have not yet fired for long enough are considered pending. Default is 0s",
	"labels":      "A map of key:value. Labels to add or overwrite for each alert",
	"annotations": "A map of key:value. Annotations to add or overwrite for each alert",
}

// NewAlertGroupResource is a helper function to simplify the provider implementation.
func NewAlertGroupResource() resource.Resource {
	return &alertGroupResource{}
}

// alertGroupResource is the resource implementation.
type alertGroupResource struct {
	client *observability.APIClient
}

// Metadata returns the resource type name.
func (a *alertGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_alertgroup"
}

// Configure adds the provider configured client to the resource.
func (a *alertGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *observability.APIClient
	var err error
	if providerData.ObservabilityCustomEndpoint != "" {
		apiClient, err = observability.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ObservabilityCustomEndpoint),
		)
	} else {
		apiClient, err = observability.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.GetRegion()),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}
	a.client = apiClient
	tflog.Info(ctx, "Observability alert group client configured")
}

// Schema defines the schema for the resource.
func (a *alertGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Observability alert group resource schema. Used to create alerts based on metrics (Thanos). Must have a `region` specified in the provider configuration.",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
					stringvalidator.LengthBetween(1, 200),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9-]+$`),
						"must match expression",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"interval": schema.StringAttribute{
				Description: descriptions["interval"],
				Optional:    true,
				Validators: []validator.String{
					validate.ValidDurationString(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rules": schema.ListNestedAttribute{
				Description: "Rules for the alert group",
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"alert": schema.StringAttribute{
							Description: descriptions["alert"],
							Required:    true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[a-zA-Z0-9-]+$`),
									"must match expression",
								),
								stringvalidator.LengthBetween(1, 200),
							},
						},
						"expression": schema.StringAttribute{
							Description: descriptions["expression"],
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 600),
								// The API currently accepts expressions with trailing newlines but does not return them,
								// leading to inconsistent Terraform results. This issue has been reported to the Obs team.
								// Until it is resolved, we proactively notify users if their input contains a trailing newline.
								validate.ValidNoTrailingNewline(),
							},
						},
						"for": schema.StringAttribute{
							Description: descriptions["for"],
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(2, 8),
								validate.ValidDurationString(),
							},
						},
						"labels": schema.MapAttribute{
							Description: descriptions["labels"],
							Optional:    true,
							ElementType: types.StringType,
							Validators: []validator.Map{
								mapvalidator.KeysAre(stringvalidator.LengthAtMost(200)),
								mapvalidator.ValueStringsAre(stringvalidator.LengthAtMost(200)),
								mapvalidator.SizeAtMost(10),
							},
						},
						"annotations": schema.MapAttribute{
							Description: descriptions["annotations"],
							Optional:    true,
							ElementType: types.StringType,
							Validators: []validator.Map{
								mapvalidator.KeysAre(stringvalidator.LengthAtMost(200)),
								mapvalidator.ValueStringsAre(stringvalidator.LengthAtMost(200)),
								mapvalidator.SizeAtMost(5),
							},
						},
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (a *alertGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	alertGroupName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "alert_group_name", alertGroupName)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating alertgroup", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createAlertGroupResp, err := a.client.CreateAlertgroups(ctx, instanceId, projectId).CreateAlertgroupsPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating alertgroup", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// all alert groups are returned. We have to search the map for the one corresponding to our name
	for _, alertGroup := range *createAlertGroupResp.Data {
		if model.Name.ValueString() != *alertGroup.Name {
			continue
		}

		err = mapFields(ctx, &alertGroup, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating alert group", fmt.Sprintf("Processing API payload: %v", err))
			return
		}
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "alert group created")
}

// Read refreshes the Terraform state with the latest data.
func (a *alertGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	alertGroupName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "alert_group_name", alertGroupName)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	readAlertGroupResp, err := a.client.GetAlertgroup(ctx, alertGroupName, instanceId, projectId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading alert group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, readAlertGroupResp.Data, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading alert group", fmt.Sprintf("Error processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

// Update attempts to update the resource. In this case, alertgroups cannot be updated.
// The Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (a *alertGroupResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating alert group", "Observability alert groups can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (a *alertGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	alertGroupName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "alert_group_name", alertGroupName)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	_, err := a.client.DeleteAlertgroup(ctx, alertGroupName, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting alert group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Alert group deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,name
func (a *alertGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing scrape config",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id],[name]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[2])...)
	tflog.Info(ctx, "Observability alert group state imported")
}

// toCreatePayload generates the payload to create a new alert group.
func toCreatePayload(ctx context.Context, model *Model) (*observability.CreateAlertgroupsPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payload := observability.CreateAlertgroupsPayload{}

	if !utils.IsUndefined(model.Name) {
		payload.Name = model.Name.ValueStringPointer()
	}

	if !utils.IsUndefined(model.Interval) {
		payload.Interval = model.Interval.ValueStringPointer()
	}

	if !utils.IsUndefined(model.Rules) {
		rules, err := toRulesPayload(ctx, model)
		if err != nil {
			return nil, err
		}
		payload.Rules = &rules
	}

	return &payload, nil
}

// toRulesPayload generates rules for create payload.
func toRulesPayload(ctx context.Context, model *Model) ([]observability.UpdateAlertgroupsRequestInnerRulesInner, error) {
	if model.Rules.Elements() == nil || len(model.Rules.Elements()) == 0 {
		return []observability.UpdateAlertgroupsRequestInnerRulesInner{}, nil
	}

	var rules []rule
	diags := model.Rules.ElementsAs(ctx, &rules, false)
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	var oarrs []observability.UpdateAlertgroupsRequestInnerRulesInner
	for i := range rules {
		rule := &rules[i]
		oarr := observability.UpdateAlertgroupsRequestInnerRulesInner{}

		if !utils.IsUndefined(rule.Alert) {
			alert := conversion.StringValueToPointer(rule.Alert)
			if alert == nil {
				return nil, fmt.Errorf("found nil alert for rule[%d]", i)
			}
			oarr.Alert = alert
		}

		if !utils.IsUndefined(rule.Expression) {
			expression := conversion.StringValueToPointer(rule.Expression)
			if expression == nil {
				return nil, fmt.Errorf("found nil expression for rule[%d]", i)
			}
			oarr.Expr = expression
		}

		if !utils.IsUndefined(rule.For) {
			for_ := conversion.StringValueToPointer(rule.For)
			if for_ == nil {
				return nil, fmt.Errorf("found nil expression for for_[%d]", i)
			}
			oarr.For = for_
		}

		if !utils.IsUndefined(rule.Labels) {
			labels, err := conversion.ToStringInterfaceMap(ctx, rule.Labels)
			if err != nil {
				return nil, fmt.Errorf("converting to Go map: %w", err)
			}
			oarr.Labels = &labels
		}

		if !utils.IsUndefined(rule.Annotations) {
			annotations, err := conversion.ToStringInterfaceMap(ctx, rule.Annotations)
			if err != nil {
				return nil, fmt.Errorf("converting to Go map: %w", err)
			}
			oarr.Annotations = &annotations
		}

		oarrs = append(oarrs, oarr)
	}

	return oarrs, nil
}

// mapRules maps alertGroup response to the model.
func mapFields(ctx context.Context, alertGroup *observability.AlertGroup, model *Model) error {
	if alertGroup == nil {
		return fmt.Errorf("nil alertGroup")
	}

	if model == nil {
		return fmt.Errorf("nil model")
	}

	if utils.IsUndefined(model.Name) {
		return fmt.Errorf("empty name")
	}

	if utils.IsUndefined(model.ProjectId) {
		return fmt.Errorf("empty projectId")
	}

	if utils.IsUndefined(model.InstanceId) {
		return fmt.Errorf("empty instanceId")
	}

	var name string
	if !utils.IsUndefined(model.Name) {
		name = model.Name.ValueString()
	} else if alertGroup.Name != nil {
		name = *alertGroup.Name
	} else {
		return fmt.Errorf("found empty name")
	}

	model.Name = types.StringValue(name)
	idParts := []string{model.ProjectId.ValueString(), model.InstanceId.ValueString(), name}
	model.Id = types.StringValue(strings.Join(idParts, core.Separator))

	var interval string
	if !utils.IsUndefined(model.Interval) {
		interval = model.Interval.ValueString()
	} else if alertGroup.Interval != nil {
		interval = *alertGroup.Interval
	} else {
		return fmt.Errorf("found empty interval")
	}
	model.Interval = types.StringValue(interval)

	if alertGroup.Rules != nil {
		err := mapRules(ctx, alertGroup, model)
		if err != nil {
			return fmt.Errorf("map rules: %w", err)
		}
	}

	return nil
}

// mapRules maps alertGroup response rules to the model rules.
func mapRules(_ context.Context, alertGroup *observability.AlertGroup, model *Model) error {
	var newRules []attr.Value

	for i, r := range *alertGroup.Rules {
		ruleMap := map[string]attr.Value{
			"alert":       types.StringPointerValue(r.Alert),
			"expression":  types.StringPointerValue(r.Expr),
			"for":         types.StringPointerValue(r.For),
			"labels":      types.MapNull(types.StringType),
			"annotations": types.MapNull(types.StringType),
		}

		if r.Labels != nil {
			labelElems := map[string]attr.Value{}
			for k, v := range *r.Labels {
				labelElems[k] = types.StringValue(v)
			}
			ruleMap["labels"] = types.MapValueMust(types.StringType, labelElems)
		}

		if r.Annotations != nil {
			annoElems := map[string]attr.Value{}
			for k, v := range *r.Annotations {
				annoElems[k] = types.StringValue(v)
			}
			ruleMap["annotations"] = types.MapValueMust(types.StringType, annoElems)
		}

		ruleTf, diags := types.ObjectValue(ruleTypes, ruleMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}
		newRules = append(newRules, ruleTf)
	}

	rulesTf, diags := types.ListValue(types.ObjectType{AttrTypes: ruleTypes}, newRules)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	model.Rules = rulesTf
	return nil
}
