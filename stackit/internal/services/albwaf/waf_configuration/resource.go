package waf

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	albwafUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/albwaf/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &wafResource{}
	_ resource.ResourceWithConfigure   = &wafResource{}
	_ resource.ResourceWithImportState = &wafResource{}
	_ resource.ResourceWithModifyPlan  = &wafResource{}
)

type Model struct {
	Id                  types.String `tfsdk:"id"`
	ProjectId           types.String `tfsdk:"project_id"`
	Region              types.String `tfsdk:"region"`
	Name                types.String `tfsdk:"name"`
	Labels              types.Map    `tfsdk:"labels"`
	ManagedRuleSetName  types.String `tfsdk:"managed_rule_set_name"`
	CustomRuleGroupName types.String `tfsdk:"custom_rule_group_name"`
}

func NewWafConfigurationResource() resource.Resource {
	return &wafResource{}
}

type ItemsModel struct {
	ListenerNames    types.Int32  `tfsdk:"listener_names"`
	LoadBalancerName types.String `tfsdk:"load_balancer_name"`
}

type wafResource struct {
	client       *albWaf.APIClient
	providerData core.ProviderData
}

func (r *wafResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_waf_configuration"
}

// Use the modifier to set the effective region in the current plan.
func (r *wafResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	// skip initial empty configuration to avoid follow-up errors
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

func (r *wafResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_alb_waf_configuration", core.Resource)
	if resp.Diagnostics.HasError() {
		return
	}
	apiClient := albwafUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "albwaf client configured")
}

var descriptions = map[string]string{
	"main":                   "albwaf resource schema.",
	"id":                     "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`name`\".",
	"project_id":             "STACKIT project ID to which the WAF Configuration is associated.",
	"region":                 "The resource region (e.g. eu01). If not defined, the provider region is used.",
	"name":                   "The name of the WAF Configuration.",
	"labels":                 "User-defined metadata as key-value pairs. Should not exceed 64 entries.",
	"managed_rule_set_name":  "Name of the managed rule set configuration for this WAF Configuration.",
	"custom_rule_group_name": "Name of the custom rule group for this WAF Configuration.",
	"count":                  "Number of listeners using this WAF Configuration.",
	"items":                  "List of Application Load Balancers with their associated listeners that use this WAF Configuration.",
	"listener_names":         "List of listener names in this Application Load Balancer using this WAF Configuration.",
	"load_balancer_name":     "The display name of the Application Load Balancer.",
}

func (r *wafResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: descriptions["main"],
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
				Validators: []validator.String{
					validate.UUID(),
				},
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
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(64),
				},
			},
			"managed_rule_set_name": schema.StringAttribute{
				Description: descriptions["managed_rule_set_name"],
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
						"must start and end with an alphanumeric character, may contain hyphens, and be 1-63 characters long",
					),
				},
			},
			"custom_rule_group_name": schema.StringAttribute{
				Description: descriptions["custom_rule_group_name"],
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
						"must start and end with an alphanumeric character, may contain hyphens, and be 1-63 characters long",
					),
				},
			},
		},
	}
}

func (r *wafResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Configuration", fmt.Sprint("Creating API payload: %w", err))
		return
	}
	createResp, err := r.client.DefaultAPI.CreateWAF(ctx, projectId, region).CreateWAFPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Configuration", fmt.Sprintf("Calling API: %v", err))
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

	err = mapFields(ctx, createResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF Configuration", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Configuration created")
}

func (r *wafResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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

	_, err := r.client.DefaultAPI.DeleteWAF(ctx, projectId, region, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting ALB WAF Configuration", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "ALB WAF Configuration deleted")
}

func (r *wafResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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
	if name == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF Configuration", "Name must be defined when reading ALB WAF Configuration")
		return
	}
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	response, err := r.client.DefaultAPI.GetWAF(ctx, projectId, region, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF Configuration", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, response, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF Configuration", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Configuration read")
}

func (r *wafResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	name := model.Name.ValueString()

	if name == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Configuration", "Name must be defined when updating ALB WAF Configuration")
		return
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Configuration", fmt.Sprint("Creating API payload: %w", err))
		return
	}
	updateResp, err := r.client.DefaultAPI.UpdateWAF(ctx, projectId, region, name).UpdateWAFPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Configuration", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"name":       updateResp.Name,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	err = mapFields(ctx, updateResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF Configuration", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF Configuration created")
}

func (r *wafResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing ALB WAF Configuration",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[name]  Got: %q", req.ID),
		)
		return
	}

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"name":       idParts[2],
	})
	tflog.Info(ctx, "ALB WAF Configuration state imported")
}

func toUpdatePayload(ctx context.Context, model *Model) (*albWaf.UpdateWAFPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var labels *map[string]string
	if !(model.Labels.IsNull() || model.Labels.IsUnknown()) {
		diags := model.Labels.ElementsAs(ctx, &labels, false)
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}
	}
	return &albWaf.UpdateWAFPayload{
		CustomRuleGroupName: model.CustomRuleGroupName.ValueStringPointer(),
		ManagedRuleSetName:  model.ManagedRuleSetName.ValueStringPointer(),
		Labels:              labels,
	}, nil
}

func toCreatePayload(ctx context.Context, model *Model) (*albWaf.CreateWAFPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var labels *map[string]string
	if !(model.Labels.IsNull() || model.Labels.IsUnknown()) {
		diags := model.Labels.ElementsAs(ctx, &labels, false)
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}
	}
	payload := &albWaf.CreateWAFPayload{
		Name:                model.Name.ValueString(),
		CustomRuleGroupName: model.CustomRuleGroupName.ValueStringPointer(),
		Labels:              labels,
		ManagedRuleSetName:  model.ManagedRuleSetName.ValueStringPointer(),
	}
	return payload, nil
}

func mapFields(ctx context.Context, wafResponse *albWaf.GetWAFResponse, model *Model, region string) error {
	if wafResponse == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	labels, err := tfutils.MapLabels(ctx, wafResponse.Labels, model.Labels)
	if err != nil {
		return err
	}

	var customRuleGroupName types.String
	if wafResponse.CustomRuleGroupName != nil {
		customRuleGroupName = types.StringValue(*wafResponse.CustomRuleGroupName)
	} else {
		customRuleGroupName = types.StringNull()
	}

	var managedRuleSetName types.String
	if wafResponse.ManagedRuleSetName != nil {
		managedRuleSetName = types.StringValue(*wafResponse.ManagedRuleSetName)
	} else {
		managedRuleSetName = types.StringNull()
	}

	model.Id = tfutils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, model.Name.ValueString())
	model.Name = types.StringValue(wafResponse.Name)
	model.Region = types.StringValue(region)
	model.CustomRuleGroupName = customRuleGroupName
	model.Labels = labels
	model.ManagedRuleSetName = managedRuleSetName
	return nil
}
