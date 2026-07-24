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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	waf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"

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
)

type Model struct {
	Id                  types.String `tfsdk:"id"`
	ProjectId           types.String `tfsdk:"project_id"`
	Region              types.String `tfsdk:"region"`
	Name                types.String `tfsdk:"name"`
	Labels              types.Map    `tfsdk:"labels"`
	ManagedRuleSetName  types.String `tfsdk:"managed_rule_set_name"`
	CustomRuleGroupName types.String `tfsdk:"custom_rule_group_name"`
	Usage               types.Object `tfsdk:"usage"`
}

type UsageModel struct {
	Count types.Int32 `tfsdk:"count"`
	Items types.List  `tfsdk:"items"`
}

var usageType = map[string]attr.Type{
	"count": types.Int32Type,
	"items": types.ListType{ElemType: types.ObjectType{AttrTypes: itemsType}},
}

type ItemsModel struct {
	ListenerNames    types.Int32  `tfsdk:"listener_names"`
	LoadBalancerName types.String `tfsdk:"load_balancer_name"`
}

var itemsType = map[string]attr.Type{
	"listener_names":     types.ListType{ElemType: types.StringType},
	"load_balancer_name": types.StringType,
}

type wafResource struct {
	client       *waf.APIClient
	providerData core.ProviderData
}

func (r *wafResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing ALB WAF",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[name]  Got: %q", req.ID),
		)
		return
	}

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"name":       idParts[2],
	})
	tflog.Info(ctx, "ALB WAF state imported")
}

func NewWafResource() resource.Resource {
	return &wafResource{}
}

// Configure implements [resource.ResourceWithConfigure].
func (r *wafResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_alb_waf", core.Resource)
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
	"id":                     "Terraform's internal resource ID. It is structured as \"`project_id`,`region`\".",
	"project_id":             "STACKIT project ID to which the WAF is associated.",
	"region":                 "The resource region (e.g. eu01). If not defined, the provider region is used.",
	"name":                   "The name of the WAF.",
	"labels":                 "User-defined metadata as key-value pairs. Should not exceed 64 entries.",
	"managed_rule_set_name":  "Name of the managed rule set configuration for this WAF.",
	"custom_rule_group_name": "Name of the custom rule group for this WAF.",
	"usage":                  "Object containing usage-information for this WAF.",
	"count":                  "Number of listeners using this WAF.",
	"items":                  "List of Application Load Balancers with their associated listeners that use this WAF.",
	"listener_names":         "List of listener names in this Application Load Balancer using this WAF.",
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
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
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
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9_.-]{0,6}[a-zA-Z0-9]|(?:[a-rt-zA-Z0-9][a-zA-Z0-9_.-]{7}|s[a-su-zA-Z0-9_.-][a-zA-Z0-9_.-]{6}|st[b-zA-Z0-9_.-][a-zA-Z0-9_.-]{5}|sta[a-bd-zA-Z0-9_.-][a-zA-Z0-9_.-]{4}|stac[a-jl-zA-Z0-9_.-][a-zA-Z0-9_.-]{3}|stack[a-hj-zA-Z0-9_.-][a-zA-Z0-9_.-]{2}|stacki[a-su-zA-Z0-9_.-][a-zA-Z0-9_.-]|stackit[a-zA-Z0-9_.])[a-zA-Z0-9_.-]{0,54}[a-zA-Z0-9])$`),
							"must start and end with an alphanumeric character, may contain dashes, underscores and dots, be 1-63 characters long and NOT start with \"stackit-\""),
					),
					mapvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9_.-]{0,61}[a-zA-Z0-9])$`),
							"must start and end with an alphanumeric character, may contain dashes, underscores and dots, be 1-63 characters long and NOT start with \"stackit-\""),
					),
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF", fmt.Sprint("Creating API payload: %w", err))
		return
	}
	createResp, err := r.client.DefaultAPI.CreateWAF(ctx, projectId, region).CreateWAFPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF", fmt.Sprint("Calling API: %w", err))
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating ALB WAF", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF created")
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting ALB WAF", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "ALB WAF deleted")
}

// Metadata implements [resource.Resource].
func (r *wafResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_waf"
}

// Read implements [resource.Resource].
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, response, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading ALB WAF", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF read")
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF", "Name must be defined when updating ALB WAF")
		return
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF", fmt.Sprint("Creating API payload: %w", err))
		return
	}
	updateResp, err := r.client.DefaultAPI.UpdateWAF(ctx, projectId, region, name).UpdateWAFPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF", fmt.Sprint("Calling API: %w", err))
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating ALB WAF", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ALB WAF created")
}

func toUpdatePayload(ctx context.Context, model *Model) (*waf.UpdateWAFPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var labels *map[string]string
	if !(model.Labels.IsNull() || model.Labels.IsUnknown()) {
		diags := model.Labels.ElementsAs(ctx, labels, false)
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}
	}
	return &waf.UpdateWAFPayload{
		Name:                model.Name.ValueStringPointer(),
		CustomRuleGroupName: model.CustomRuleGroupName.ValueStringPointer(),
		ManagedRuleSetName:  model.ManagedRuleSetName.ValueStringPointer(),
		Labels:              labels,
	}, nil
}

func toCreatePayload(ctx context.Context, model *Model) (*waf.CreateWAFPayload, error) {
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
	payload := &waf.CreateWAFPayload{
		Name:                model.Name.ValueStringPointer(),
		CustomRuleGroupName: model.CustomRuleGroupName.ValueStringPointer(),
		Labels:              labels,
		ManagedRuleSetName:  model.ManagedRuleSetName.ValueStringPointer(),
	}
	return payload, nil
}

func mapFields(ctx context.Context, wafResponse *waf.GetWAFResponse, model *Model, region string) error {
	if wafResponse == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var diags diag.Diagnostics

	labels, err := tfutils.MapLabels(ctx, wafResponse.Labels, model.Labels)
	if err != nil {
		return err
	}

	var usage basetypes.ObjectValue

	if usageRes, ok := wafResponse.GetUsageOk(); ok {
		itemsElems := []attr.Value{}
		for _, item := range wafResponse.Usage.Items {
			listenerName, diags := types.ListValueFrom(ctx, types.StringType, item.ListenerNames)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			element := map[string]attr.Value{
				"listener_names":     listenerName,
				"load_balancer_name": types.StringValue(*item.LoadBalancerName),
			}
			itemsElems = append(itemsElems, types.ObjectValueMust(itemsType, element))
		}
		var items basetypes.ListValue
		if len(itemsElems) == 0 {
			items = types.ListNull(types.ObjectType{AttrTypes: itemsType})
		} else {
			items, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: itemsType}, itemsElems)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}
		usageModel := UsageModel{
			Count: types.Int32PointerValue(usageRes.Count),
			Items: items,
		}
		usage, diags = types.ObjectValueFrom(ctx, usageType, usageModel)

		if diags.HasError() {
			return core.DiagsToError(diags)
		}
	} else {
		usage = types.ObjectNull(usageType)
	}

	var name types.String
	if wafResponse.Name != nil {
		name = types.StringValue(*wafResponse.Name)
	} else {
		name = types.StringNull()
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
	model.Name = name
	model.Region = types.StringValue(region)
	model.CustomRuleGroupName = customRuleGroupName
	model.Labels = labels
	model.ManagedRuleSetName = managedRuleSetName
	model.Usage = usage
	return nil
}
