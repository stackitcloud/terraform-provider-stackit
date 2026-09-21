package volume

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	automation "github.com/stackitcloud/stackit-sdk-go/services/automation/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	automationUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/automation/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &volumeAutomationResource{}
	_ resource.ResourceWithConfigure   = &volumeAutomationResource{}
	_ resource.ResourceWithImportState = &volumeAutomationResource{}
	_ resource.ResourceWithModifyPlan  = &volumeAutomationResource{}
)

// Model represents the schema for the stackit_volume_automation resource and datasource.
type Model struct {
	ID           types.String         `tfsdk:"id"`
	ProjectId    types.String         `tfsdk:"project_id"`
	Region       types.String         `tfsdk:"region"`
	TemplateId   types.String         `tfsdk:"template_id"`
	AutomationId types.String         `tfsdk:"automation_id"`
	Name         types.String         `tfsdk:"name"`
	Description  types.String         `tfsdk:"description"`
	Input        jsontypes.Normalized `tfsdk:"input"`
	Triggers     *triggersModel       `tfsdk:"triggers"`
}

type triggersModel struct {
	Schedule *scheduleTriggerModel `tfsdk:"schedule"`
}

type scheduleTriggerModel struct {
	Rrule types.String `tfsdk:"rrule"`
}

// descriptions for the attributes in the Schema
var descriptions = map[string]string{
	"id":                               "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`automation_id`\".",
	"project_id":                       "STACKIT Project ID to which the volume automation is associated.",
	"region":                           "The resource region. If not defined, the provider region is used.",
	"template_id":                      "ID of the automation template this volume automation is based on.",
	"automation_id":                    "ID of the volume automation.",
	"name":                             "The volume automation name.",
	"description":                      "The volume automation description.",
	"input":                            "Configuration input for the volume automation. Exactly one of the nested attributes must be set.", // TODO: update description
	"volume_recovery_point_management": "Configuration for automated volume recovery point (snapshot) management.",
	"inherit_volume_labels":            "Whether recovery points inherit the labels of the volume they were created from. Defaults to `false`.",
	"recovery_point_labels":            "Labels to attach to created recovery points.",
	"volume_label_selector":            "Label selector used to select the volumes this automation applies to.",
	"snapshot_retention_policy":        "Defines how long created recovery points (snapshots) are retained.",
	"snapshot_retention_policy_kind":   "The retention policy kind. Valid values are: `count`, `indefinitely`.",
	"snapshot_retention_policy_value":  "Number of recovery points to retain. Required if `kind` is `count`, must not be set otherwise.",
	"triggers":                         "Triggers that determine when the automation runs.",
	"schedule":                         "Runs the automation on a recurring schedule.",
	"rrule":                            "An `rrule` (Recurrence Rule) is a standardized string format used in iCalendar (RFC 5545) to define repeating events, and you can generate one by using a dedicated library or by using online generator tools to specify parameters like frequency, interval, and end dates.",
}

// NewVolumeAutomationResource is a helper function to simplify the provider implementation.
func NewVolumeAutomationResource() resource.Resource {
	return &volumeAutomationResource{}
}

// volumeAutomationResource is the resource implementation.
type volumeAutomationResource struct {
	client       *automation.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *volumeAutomationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

// Metadata returns the resource type name.
func (r *volumeAutomationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_automation"
}

// Configure adds the provider configured client to the resource.
func (r *volumeAutomationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_volume_automation", core.Resource)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := automationUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.providerData = providerData
	r.client = apiClient
	tflog.Info(ctx, "Volume automation client configured.")
}

// Schema defines the schema for the resource.
func (r *volumeAutomationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Volume automation resource schema. Must have a `region` specified in the provider configuration.", core.Resource),
		Description:         "Volume automation resource schema. Must have a `region` specified in the provider configuration.",
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
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template_id": schema.StringAttribute{
				Description: descriptions["template_id"],
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
			"automation_id": schema.StringAttribute{
				Description: descriptions["automation_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Optional:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
			},
			"input": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: descriptions["input"],
				Optional:    true,
			},
			"triggers": schema.SingleNestedAttribute{
				Description: descriptions["triggers"],
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"schedule": schema.SingleNestedAttribute{
						Description: descriptions["schedule"],
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"rrule": schema.StringAttribute{
								Description: descriptions["rrule"],
								Required:    true,
								Validators: []validator.String{
									validate.Rrule(),
								},
							},
						},
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *volumeAutomationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating volume automation", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	automationResp, err := r.client.DefaultAPI.CreateVolumeAutomation(ctx, projectId, region).CreateVolumeAutomationPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating volume automation", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	ctx = tflog.SetField(ctx, "automation_id", automationResp.Id)

	err = mapFields(ctx, automationResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating volume automation", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Volume automation created.")
}

// Read refreshes the Terraform state with the latest data.
func (r *volumeAutomationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	automationId := model.AutomationId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "automation_id", automationId)
	ctx = tflog.SetField(ctx, "region", region)

	automationResp, err := r.client.DefaultAPI.GetVolumeAutomation(ctx, projectId, region, automationId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading volume automation", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, automationResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading volume automation", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Volume automation read.")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *volumeAutomationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var plan Model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state Model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := plan.ProjectId.ValueString()
	automationId := state.AutomationId.ValueString()
	region := r.providerData.GetRegionWithOverride(plan.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "automation_id", automationId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(&plan)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume automation", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Workaround: The input field is an open object where we don't know all keys. If some input fields where removed,
	// we can't set them here to null. For this reason we do one update with updateMask "input", to overwrite the whole input object.
	// Setting it to '*' would cause issue when the API gets new fields in the future and is therefore no option.
	automationResp, err := r.client.DefaultAPI.PartialUpdateVolumeAutomation(ctx, projectId, region, automationId).
		PartialUpdateVolumeAutomationPayload(*payload).
		UpdateMask("input").
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume automation", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Workaround: Updates all other fields accordingly, which were not already update with the previous update.
	automationResp, err = r.client.DefaultAPI.PartialUpdateVolumeAutomation(ctx, projectId, region, automationId).
		PartialUpdateVolumeAutomationPayload(*payload).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume automation", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, automationResp, &plan, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume automation", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Volume automation updated.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *volumeAutomationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	automationId := model.AutomationId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "automation_id", automationId)
	ctx = tflog.SetField(ctx, "region", region)

	err := r.client.DefaultAPI.DeleteVolumeAutomation(ctx, projectId, region, automationId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting volume automation", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Volume automation deleted.")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,region,automation_id
func (r *volumeAutomationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing volume automation",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[automation_id], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":    idParts[0],
		"region":        idParts[1],
		"automation_id": idParts[2],
	})

	tflog.Info(ctx, "Volume automation state imported.")
}

// mapFields maps a VolumeAutomation API response to the model.
func mapFields(_ context.Context, apiResp *automation.VolumeAutomation, model *Model, region string) error {
	if apiResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.AutomationId = types.StringValue(apiResp.Id)
	model.ID = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, apiResp.Id)
	model.Region = types.StringValue(region)

	if apiResp.TemplateId != nil {
		model.TemplateId = types.StringValue(*apiResp.TemplateId)
	}

	model.Name = types.StringPointerValue(apiResp.Name)
	model.Description = types.StringPointerValue(apiResp.Description)

	var inputString *string
	if apiResp.Input.Get() != nil {
		inputJson, err := apiResp.Input.MarshalJSON()
		if err != nil {
			return fmt.Errorf("error marshaling input field: %v", err)
		}
		inputString = new(string(inputJson))
	}
	model.Input = jsontypes.NewNormalizedPointerValue(inputString)

	model.Triggers = mapTriggers(apiResp.Triggers)

	return nil
}

func mapTriggers(triggers automation.NullableAutomationTriggers) *triggersModel {
	if triggers.Get() == nil {
		return nil
	}
	if triggers.Get().Schedule.Get() == nil {
		return &triggersModel{
			Schedule: nil,
		}
	}
	return &triggersModel{
		Schedule: &scheduleTriggerModel{
			Rrule: types.StringValue(triggers.Get().Schedule.Get().Rrule),
		},
	}
}

func toCreatePayload(model *Model) (*automation.CreateVolumeAutomationPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	input, err := toInputPayload(model.Input)
	if err != nil {
		return nil, err
	}

	return &automation.CreateVolumeAutomationPayload{
		TemplateId:  model.TemplateId.ValueString(),
		Name:        *automation.NewNullableString(conversion.StringValueToPointer(model.Name)),
		Description: *automation.NewNullableString(conversion.StringValueToPointer(model.Description)),
		Input:       *input,
		Triggers:    *toTriggersPayload(model),
	}, nil
}

func toUpdatePayload(plan *Model) (*automation.PartialUpdateVolumeAutomationPayload, error) {
	if plan == nil {
		return nil, fmt.Errorf("nil plan model")
	}

	input, err := toInputPayload(plan.Input)
	if err != nil {
		return nil, err
	}

	return &automation.PartialUpdateVolumeAutomationPayload{
		Name:        *automation.NewNullableString(conversion.StringValueToPointer(plan.Name)),
		Description: *automation.NewNullableString(conversion.StringValueToPointer(plan.Description)),
		Input:       *input,
		Triggers:    *toTriggersPayload(plan),
	}, nil
}

func toInputPayload(modelInput jsontypes.Normalized) (*automation.NullableVolumeAutomationInput, error) {
	if utils.IsUndefined(modelInput) {
		return automation.NewNullableVolumeAutomationInput(nil), nil
	}

	inputJson := modelInput.ValueString()
	inputPayload := &automation.NullableVolumeAutomationInput{}

	err := json.Unmarshal([]byte(inputJson), inputPayload)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling input payload: %w", err)
	}
	return inputPayload, nil
}

func toTriggersPayload(model *Model) *automation.NullableAutomationTriggers {
	if model == nil || model.Triggers == nil {
		return automation.NewNullableAutomationTriggers(nil)
	}
	if model.Triggers.Schedule == nil || utils.IsUndefined(model.Triggers.Schedule.Rrule) {
		return automation.NewNullableAutomationTriggers(&automation.AutomationTriggers{
			Schedule: *automation.NewNullableAutomationScheduleTrigger(nil),
		})
	}

	return automation.NewNullableAutomationTriggers(&automation.AutomationTriggers{
		Schedule: *automation.NewNullableAutomationScheduleTrigger(&automation.AutomationScheduleTrigger{
			Rrule: model.Triggers.Schedule.Rrule.ValueString(),
		}),
	})
}
