package runner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	coreUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	intakeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/intake/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	intake "github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &runnerResource{}
	_ resource.ResourceWithConfigure   = &runnerResource{}
	_ resource.ResourceWithImportState = &runnerResource{}
	_ resource.ResourceWithModifyPlan  = &runnerResource{}
)

// Model is the internal model of the terraform resource
type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	ProjectId          types.String `tfsdk:"project_id"`
	RunnerId           types.String `tfsdk:"runner_id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Labels             types.Map    `tfsdk:"labels"`
	MaxMessageSizeKiB  types.Int32  `tfsdk:"max_message_size_kib"`
	MaxMessagesPerHour types.Int32  `tfsdk:"max_messages_per_hour"`
	Region             types.String `tfsdk:"region"`
}

// NewRunnerResource is a helper function to simplify the provider implementation.
func NewRunnerResource() resource.Resource {
	return &runnerResource{}
}

// runnerResource is the resource implementation.
type runnerResource struct {
	client       *intake.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *runnerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intake_runner"
}

// Configure adds the provider configured client to the resource.
func (r *runnerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := intakeUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Intake runner client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *runnerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Schema defines the schema for the resource.
func (r *runnerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                  "Manages STACKIT Intake Runner.",
		"id":                    "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`runner_id`\".",
		"project_id":            "STACKIT Project ID to which the runner is associated.",
		"runner_id":             "The runner ID.",
		"name":                  "The name of the runner.",
		"region":                "The resource region. If not defined, the provider region is used.",
		"description":           "The description of the runner.",
		"labels":                "User-defined labels.",
		"max_message_size_kib":  "The maximum message size in KiB.",
		"max_messages_per_hour": "The maximum number of messages per hour.",
	}

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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"runner_id": schema.StringAttribute{
				Description: descriptions["runner_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"max_message_size_kib": schema.Int32Attribute{
				Description: descriptions["max_message_size_kib"],
				Required:    true,
			},
			"max_messages_per_hour": schema.Int32Attribute{
				Description: descriptions["max_messages_per_hour"],
				Required:    true,
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *runnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// prepare the payload struct for the create bar request
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new runner
	runnerResp, err := r.client.DefaultAPI.CreateIntakeRunner(ctx, projectId, region).CreateIntakeRunnerPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating runner", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id": projectId,
		"region":     region,
		"runner_id":  runnerResp.Id,
	})

	if resp.Diagnostics.HasError() {
		return
	}

	// Wait for creation of intake runner
	_, err = wait.CreateOrUpdateIntakeRunnerWaitHandler(ctx, r.client.DefaultAPI, projectId, region, runnerResp.GetId()).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating runner", fmt.Sprintf("Intake runner creation waiting: %v", err))
		return
	}

	err = mapFields(runnerResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating runner", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Intake runner created")
}

// Read refreshes the Terraform state with the latest data.
func (r *runnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	runnerId := model.RunnerId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "runner_id", runnerId)

	runnerResp, err := r.client.DefaultAPI.GetIntakeRunner(ctx, projectId, region, runnerId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading runner", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(runnerResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading runner", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Intake runner read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *runnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model, state Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	runnerId := model.RunnerId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "runner_id", runnerId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(&model, &state)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating runner", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Update runner
	runnerResp, err := r.client.DefaultAPI.UpdateIntakeRunner(ctx, projectId, region, runnerId).UpdateIntakeRunnerPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating runner", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Wait for update
	_, err = wait.CreateOrUpdateIntakeRunnerWaitHandler(ctx, r.client.DefaultAPI, projectId, region, runnerId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating runner", fmt.Sprintf("Runner update waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(runnerResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating runner", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Intake runner updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *runnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	runnerId := model.RunnerId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "runner_id", runnerId)

	// Delete existing runner
	err := r.client.DefaultAPI.DeleteIntakeRunner(ctx, projectId, region, runnerId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			tflog.Info(ctx, "Intake runner already deleted")
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting runner", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Wait for the delete operation to complete
	_, err = wait.DeleteIntakeRunnerWaitHandler(ctx, r.client.DefaultAPI, projectId, region, runnerId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting runner", fmt.Sprintf("Runner deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Intake runner deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the Intake runner resource import identifier is: [project_id],[region],[runner_id]
func (r *runnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing intake runner",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[runner_id], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"runner_id":  idParts[2],
	})

	tflog.Info(ctx, "Intake runner state imported")
}

// Maps runner fields to the provider internal model
func mapFields(runnerResp *intake.IntakeRunnerResponse, model *Model, region string) error {
	if runnerResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		runnerResp.Id,
	)

	if runnerResp.Id == "" {
		model.RunnerId = types.StringNull()
	} else {
		model.RunnerId = types.StringValue(runnerResp.Id)
	}

	if runnerResp.Labels == nil {
		model.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{})
	} else {
		labels, diags := types.MapValueFrom(context.Background(), types.StringType, runnerResp.Labels)
		if diags.HasError() {
			return fmt.Errorf("converting labels: %w", core.DiagsToError(diags))
		}
		model.Labels = labels
	}

	if runnerResp.DisplayName == "" {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(runnerResp.DisplayName)
	}

	model.Description = types.StringPointerValue(runnerResp.Description)
	model.Region = types.StringValue(region)

	if runnerResp.MaxMessageSizeKiB == 0 {
		model.MaxMessageSizeKiB = types.Int32Null()
	} else {
		model.MaxMessageSizeKiB = types.Int32Value(runnerResp.MaxMessageSizeKiB)
	}

	if runnerResp.MaxMessagesPerHour == 0 {
		model.MaxMessagesPerHour = types.Int32Null()
	} else {
		model.MaxMessagesPerHour = types.Int32Value(runnerResp.MaxMessagesPerHour)
	}

	return nil
}

// Build CreateIntakeRunnerPayload from provider's model
func toCreatePayload(model *Model) (*intake.CreateIntakeRunnerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var labels map[string]string
	if !model.Labels.IsNull() && !model.Labels.IsUnknown() {
		diags := model.Labels.ElementsAs(context.Background(), &labels, false)
		if diags.HasError() {
			return nil, fmt.Errorf("converting labels: %w", core.DiagsToError(diags))
		}
	}

	return &intake.CreateIntakeRunnerPayload{
		Description:        conversion.StringValueToPointer(model.Description),
		DisplayName:        model.Name.ValueString(),
		Labels:             labels,
		MaxMessageSizeKiB:  int32(model.MaxMessageSizeKiB.ValueInt32()),
		MaxMessagesPerHour: int32(model.MaxMessagesPerHour.ValueInt32()),
	}, nil
}

// Build UpdateIntakeRunnerPayload from provider's model
func toUpdatePayload(model, state *Model) (*intake.UpdateIntakeRunnerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}
	if state == nil {
		return nil, fmt.Errorf("state is nil")
	}

	payload := &intake.UpdateIntakeRunnerPayload{}
	if !model.MaxMessageSizeKiB.IsNull() && !model.MaxMessageSizeKiB.IsUnknown() {
		payload.MaxMessageSizeKiB = coreUtils.Ptr(model.MaxMessageSizeKiB.ValueInt32())
	}

	if !model.MaxMessagesPerHour.IsNull() && !model.MaxMessagesPerHour.IsUnknown() {
		payload.MaxMessagesPerHour = coreUtils.Ptr(model.MaxMessagesPerHour.ValueInt32())
	}

	// Optional fields
	payload.DisplayName = conversion.StringValueToPointer(model.Name)
	payload.Description = conversion.StringValueToPointer(model.Description)

	var labels map[string]string
	if !model.Labels.IsNull() && !model.Labels.IsUnknown() {
		diags := model.Labels.ElementsAs(context.Background(), &labels, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to convert labels: %w", core.DiagsToError(diags))
		}
		payload.Labels = labels
	}

	return payload, nil
}
