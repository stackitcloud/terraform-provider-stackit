package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	ufw "github.com/stackitcloud/stackit-sdk-go/services/ufw/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/ufw/v1api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
	_ resource.ResourceWithModifyPlan  = &instanceResource{}
)

type Model struct {
	Id         types.String `tfsdk:"id"`
	RuleId     types.String `tfsdk:"rule_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	Region     types.String `tfsdk:"region"`
	InstanceId types.String `tfsdk:"instance_id"`
	Product    types.String `tfsdk:"product"`
	SourceIP   types.String `tfsdk:"source_ip"`
	Type       types.String `tfsdk:"type"`
}

func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

type instanceResource struct {
	client       ufw.DefaultAPI
	providerData core.ProviderData
}

func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ufw_instance"
}

func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.providerData = providerData

	apiClient, err := ufw.NewAPIClient(
		ufwUtilsConfigureOptions(&providerData)...,
	)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v", err))
		return
	}

	r.client = apiClient.DefaultAPI
	tflog.Info(ctx, "UFW instance client configured")
}

func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "UFW Instance (Rule) resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform internal resource identifier in format 'project_id,region,rule_id'.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rule_id": schema.StringAttribute{
				Description: "The rule UUID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT Project ID associated with the rule.",
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
				Description: "The resource region.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "The target service instance ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"product": schema.StringAttribute{
				Description: "The source service product (e.g. 'edge-cloud').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_ip": schema.StringAttribute{
				Description: "The source IP (CIDR) to which the rule applies.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of the rule (e.g., 'ACL').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, core.DefaultOperationTimeout)
	defer cancel()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating UFW instance", fmt.Sprintf("Building payload: %v", err))
		return
	}

	createResp, err := r.client.CreateRule(ctx, projectId, region).CreateRulePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating UFW instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ruleId := createResp.GetRefId()
	if ruleId == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating UFW instance", "API returned an empty rule ID")
		return
	}

	model.RuleId = types.StringValue(ruleId)

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"rule_id":    ruleId,
		"id":         utils.BuildInternalTerraformId(projectId, region, ruleId).ValueString(),
	})
	if resp.Diagnostics.HasError() {
		return
	}

	ruleData, err := wait.CreateRuleWaitHandler(ctx, r.client, projectId, region, ruleId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating UFW instance", fmt.Sprintf("Waiting state: %v", err))
		return
	}

	err = mapFields(ruleData, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating UFW instance", fmt.Sprintf("Mapping fields: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "UFW instance created")
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, core.DefaultOperationTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ruleId := model.RuleId.ValueString()

	if ruleId == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "rule_id", ruleId)

	ruleData, err := r.client.GetRule(ctx, projectId, region, ruleId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading UFW instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ruleData, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading UFW instance", fmt.Sprintf("Mapping fields: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "UFW instance read")
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ruleId := model.RuleId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, core.DefaultOperationTimeout)
	defer cancel()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "rule_id", ruleId)

	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating UFW instance", fmt.Sprintf("Building payload: %v", err))
		return
	}

	_, err = r.client.UpdateRule(ctx, projectId, region, ruleId).UpdateRulePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating UFW instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ruleData, err := wait.UpdateRuleWaitHandler(ctx, r.client, projectId, region, ruleId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating UFW instance", fmt.Sprintf("Waiting state: %v", err))
		return
	}

	err = mapFields(ruleData, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating UFW instance", fmt.Sprintf("Mapping fields: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "UFW instance updated")
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ruleId := model.RuleId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, core.DefaultOperationTimeout)
	defer cancel()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "rule_id", ruleId)

	_, err := r.client.DeleteRule(ctx, projectId, region, ruleId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting UFW instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteRuleWaitHandler(ctx, r.client, projectId, region, ruleId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting UFW instance", fmt.Sprintf("Waiting state: %v", err))
		return
	}

	tflog.Info(ctx, "UFW instance deleted")
}

func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing UFW instance",
			fmt.Sprintf("Expected import identifier format '[project_id],[region],[rule_id]', got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"rule_id":    idParts[2],
	})

	tflog.Info(ctx, "UFW instance imported")
}

func (r *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel, planModel Model
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
}

func mapFields(ruleResp *ufw.RuleResponse, model *Model) error {
	if ruleResp == nil {
		return fmt.Errorf("response payload is nil")
	}
	if model == nil {
		return fmt.Errorf("model pointer is nil")
	}

	if ruleResp.HasRegion() {
		model.Region = types.StringValue(ruleResp.GetRegion())
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		model.Region.ValueString(),
		model.RuleId.ValueString(),
	)

	model.InstanceId = types.StringValue(ruleResp.InstanceId)
	model.Product = types.StringValue(ruleResp.Product)
	model.SourceIP = types.StringValue(ruleResp.SourceIP)
	model.Type = types.StringValue(ruleResp.Type)

	return nil
}

func toCreatePayload(model *Model) (*ufw.CreateRulePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	payload := ufw.NewCreateRulePayload(
		model.InstanceId.ValueString(),
		model.Product.ValueString(),
		model.SourceIP.ValueString(),
		model.Type.ValueString(),
	)

	return payload, nil
}

func toUpdatePayload(model *Model) (*ufw.UpdateRulePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	payload := ufw.NewUpdateRulePayload(model.SourceIP.ValueString())

	return payload, nil
}

func ufwUtilsConfigureOptions(providerData *core.ProviderData) []config.ConfigurationOption {
	options := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}

	if providerData.UfwCustomEndpoint != "" {
		options = append(options, config.WithEndpoint(providerData.UfwCustomEndpoint))
	}

	return options
}
