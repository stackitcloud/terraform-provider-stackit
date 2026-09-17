package volume

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	_ resource.Resource                   = &volumeAutomationResource{}
	_ resource.ResourceWithConfigure      = &volumeAutomationResource{}
	_ resource.ResourceWithImportState    = &volumeAutomationResource{}
	_ resource.ResourceWithModifyPlan     = &volumeAutomationResource{}
	_ resource.ResourceWithValidateConfig = &volumeAutomationResource{}
)

// Model represents the schema for the stackit_volume_automation resource and datasource.
type Model struct {
	ID           types.String   `tfsdk:"id"`
	ProjectId    types.String   `tfsdk:"project_id"`
	Region       types.String   `tfsdk:"region"`
	TemplateId   types.String   `tfsdk:"template_id"`
	AutomationId types.String   `tfsdk:"automation_id"`
	Name         types.String   `tfsdk:"name"`
	Description  types.String   `tfsdk:"description"`
	Input        *inputModel    `tfsdk:"input"`
	Triggers     *triggersModel `tfsdk:"triggers"`
}

type inputModel struct {
	VolumeRecoveryPointManagement *volumeRecoveryPointManagementModel `tfsdk:"volume_recovery_point_management"`
}

type volumeRecoveryPointManagementModel struct {
	InheritVolumeLabels     types.Bool                    `tfsdk:"inherit_volume_labels"`
	RecoveryPointLabels     types.Map                     `tfsdk:"recovery_point_labels"`
	VolumeLabelSelector     types.String                  `tfsdk:"volume_label_selector"`
	SnapshotRetentionPolicy *snapshotRetentionPolicyModel `tfsdk:"snapshot_retention_policy"`
}

type snapshotRetentionPolicyModel struct {
	Kind  types.String `tfsdk:"kind"`
	Value types.Int32  `tfsdk:"value"`
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
	"input":                            "Configuration input for the volume automation. Exactly one of the nested attributes must be set.",
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
			"input": schema.SingleNestedAttribute{
				Description: descriptions["input"],
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"volume_recovery_point_management": schema.SingleNestedAttribute{
						Description: descriptions["volume_recovery_point_management"],
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"inherit_volume_labels": schema.BoolAttribute{
								Description: descriptions["inherit_volume_labels"],
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"recovery_point_labels": schema.MapAttribute{
								Description: descriptions["recovery_point_labels"],
								ElementType: types.StringType,
								Optional:    true,
							},
							"volume_label_selector": schema.StringAttribute{
								Description: descriptions["volume_label_selector"],
								Optional:    true,
							},
							"snapshot_retention_policy": schema.SingleNestedAttribute{
								Description: descriptions["snapshot_retention_policy"],
								Required:    true,
								Attributes: map[string]schema.Attribute{
									"kind": schema.StringAttribute{
										Description: descriptions["snapshot_retention_policy_kind"],
										Required:    true,
										Validators: []validator.String{
											stringvalidator.OneOf(
												string(automation.SNAPSHOTRETENTIONPOLICYCOUNTKIND_COUNT),
												string(automation.SNAPSHOTRETENTIONPOLICYINDEFINITELYKIND_INDEFINITELY),
											),
										},
									},
									"value": schema.Int32Attribute{
										Description: descriptions["snapshot_retention_policy_value"],
										Optional:    true,
										Validators: []validator.Int32{
											int32validator.AtLeast(1),
										},
									},
								},
							},
						},
					},
				},
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

// ValidateConfig validates cross-field constraints that can't be expressed via schema validators alone.
func (r *volumeAutomationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if model.Input == nil || model.Input.VolumeRecoveryPointManagement == nil {
		return
	}
	srp := model.Input.VolumeRecoveryPointManagement.SnapshotRetentionPolicy
	if srp == nil || utils.IsUndefined(srp.Kind) {
		return
	}

	valuePath := path.Root("input").AtName("volume_recovery_point_management").AtName("snapshot_retention_policy").AtName("value")

	switch srp.Kind.ValueString() {
	case string(automation.SNAPSHOTRETENTIONPOLICYCOUNTKIND_COUNT):
		if srp.Value.IsNull() {
			resp.Diagnostics.AddAttributeError(
				valuePath,
				"Missing snapshot_retention_policy.value",
				fmt.Sprintf("value is required when snapshot_retention_policy.kind is %q.", automation.SNAPSHOTRETENTIONPOLICYCOUNTKIND_COUNT),
			)
		}
	case string(automation.SNAPSHOTRETENTIONPOLICYINDEFINITELYKIND_INDEFINITELY):
		if !utils.IsUndefined(srp.Value) {
			resp.Diagnostics.AddAttributeError(
				valuePath,
				"Unexpected snapshot_retention_policy.value",
				fmt.Sprintf("value must not be set when snapshot_retention_policy.kind is %q.", automation.SNAPSHOTRETENTIONPOLICYINDEFINITELYKIND_INDEFINITELY),
			)
		}
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

	payload, err := toCreatePayload(ctx, &model)
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
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
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

	payload, err := toUpdatePayload(ctx, &plan)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating volume automation", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	automationResp, err := r.client.DefaultAPI.PartialUpdateVolumeAutomation(ctx, projectId, region, automationId).
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
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
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
func mapFields(ctx context.Context, apiResp *automation.VolumeAutomation, model *Model, region string) error {
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

	model.Name = conversion.StringPointerValueNullIfEmpty(apiResp.Name)
	model.Description = conversion.StringPointerValueNullIfEmpty(apiResp.Description)

	input, err := mapInput(ctx, apiResp.Input, model.Input)
	if err != nil {
		return fmt.Errorf("mapping input: %w", err)
	}
	model.Input = input

	model.Triggers = mapTriggers(apiResp.Triggers)

	return nil
}

func mapInput(ctx context.Context, apiInput *automation.VolumeAutomationInput, currentInput *inputModel) (*inputModel, error) {
	if apiInput == nil || apiInput.VolumeRecoveryPointManagementInput == nil {
		return nil, nil
	}
	vrpm := apiInput.VolumeRecoveryPointManagementInput

	currentLabels := types.MapNull(types.StringType)
	if currentInput != nil && currentInput.VolumeRecoveryPointManagement != nil {
		currentLabels = currentInput.VolumeRecoveryPointManagement.RecoveryPointLabels
	}

	labels, err := utils.MapLabels(ctx, vrpm.RecoveryPointLabels, currentLabels)
	if err != nil {
		return nil, fmt.Errorf("mapping recovery point labels: %w", err)
	}

	srp, err := mapSnapshotRetentionPolicy(vrpm.SnapshotRetentionPolicy)
	if err != nil {
		return nil, err
	}

	return &inputModel{
		VolumeRecoveryPointManagement: &volumeRecoveryPointManagementModel{
			InheritVolumeLabels:     types.BoolPointerValue(vrpm.InheritVolumeLabels),
			RecoveryPointLabels:     labels,
			VolumeLabelSelector:     conversion.StringPointerValueNullIfEmpty(vrpm.VolumeLabelSelector),
			SnapshotRetentionPolicy: srp,
		},
	}, nil
}

func mapSnapshotRetentionPolicy(srp automation.SnapshotRetentionPolicy) (*snapshotRetentionPolicyModel, error) {
	switch {
	case srp.SnapshotRetentionPolicyCount != nil:
		return &snapshotRetentionPolicyModel{
			Kind:  types.StringValue(string(srp.SnapshotRetentionPolicyCount.Kind)),
			Value: types.Int32Value(srp.SnapshotRetentionPolicyCount.Value),
		}, nil
	case srp.SnapshotRetentionPolicyIndefinitely != nil:
		return &snapshotRetentionPolicyModel{
			Kind:  types.StringValue(string(srp.SnapshotRetentionPolicyIndefinitely.Kind)),
			Value: types.Int32Null(),
		}, nil
	default:
		return nil, fmt.Errorf("response contains an unknown snapshot retention policy variant")
	}
}

func mapTriggers(triggers *automation.AutomationTriggers) *triggersModel {
	if triggers == nil || triggers.Schedule == nil {
		return nil
	}
	return &triggersModel{
		Schedule: &scheduleTriggerModel{
			Rrule: types.StringValue(triggers.Schedule.Rrule),
		},
	}
}

func toCreatePayload(ctx context.Context, model *Model) (*automation.CreateVolumeAutomationPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	input, err := toInputPayload(ctx, model.Input)
	if err != nil {
		return nil, err
	}

	return &automation.CreateVolumeAutomationPayload{
		TemplateId:  model.TemplateId.ValueString(),
		Name:        conversion.StringValueToPointer(model.Name),
		Description: conversion.StringValueToPointer(model.Description),
		Input:       input,
		Triggers:    toTriggersPayload(model.Triggers),
	}, nil
}

func toUpdatePayload(ctx context.Context, plan *Model) (*automation.PartialUpdateVolumeAutomationPayload, error) {
	if plan == nil {
		return nil, fmt.Errorf("nil plan model")
	}

	input, err := toInputPayload(ctx, plan.Input)
	if err != nil {
		return nil, err
	}

	return &automation.PartialUpdateVolumeAutomationPayload{
		// sent as explicit "" instead of omitted so clearing them actually takes effect
		Name:        new(plan.Name.ValueString()),
		Description: new(plan.Description.ValueString()),
		Input:       input,
		Triggers:    toTriggersPayload(plan.Triggers),
	}, nil
}

func toInputPayload(ctx context.Context, model *inputModel) (*automation.VolumeAutomationInput, error) {
	if model == nil || model.VolumeRecoveryPointManagement == nil {
		return nil, nil
	}
	vrpm := model.VolumeRecoveryPointManagement

	srp, err := toSnapshotRetentionPolicyPayload(vrpm.SnapshotRetentionPolicy)
	if err != nil {
		return nil, err
	}

	var recoveryPointLabels *map[string]string
	if !utils.IsUndefined(vrpm.RecoveryPointLabels) {
		labels, err := utils.LabelsToPayload(ctx, vrpm.RecoveryPointLabels)
		if err != nil {
			return nil, fmt.Errorf("converting recovery_point_labels: %w", err)
		}
		recoveryPointLabels = &labels
	}

	input := automation.VolumeRecoveryPointManagementInputAsVolumeAutomationInput(&automation.VolumeRecoveryPointManagementInput{
		Kind:                    string(automation.VOLUMETEMPLATEAUTOMATIONINPUTKIND_VOLUME_RECOVERY_POINT_MANAGEMENT),
		InheritVolumeLabels:     conversion.BoolValueToPointer(vrpm.InheritVolumeLabels),
		RecoveryPointLabels:     recoveryPointLabels,
		VolumeLabelSelector:     conversion.StringValueToPointer(vrpm.VolumeLabelSelector),
		SnapshotRetentionPolicy: srp,
	})
	return &input, nil
}

func toSnapshotRetentionPolicyPayload(model *snapshotRetentionPolicyModel) (automation.SnapshotRetentionPolicy, error) {
	if model == nil {
		return automation.SnapshotRetentionPolicy{}, fmt.Errorf("snapshot_retention_policy is required when volume_recovery_point_management is set")
	}

	switch model.Kind.ValueString() {
	case string(automation.SNAPSHOTRETENTIONPOLICYCOUNTKIND_COUNT):
		if utils.IsUndefined(model.Value) {
			return automation.SnapshotRetentionPolicy{}, fmt.Errorf("value is required when snapshot_retention_policy kind is %q", automation.SNAPSHOTRETENTIONPOLICYCOUNTKIND_COUNT)
		}
		return automation.SnapshotRetentionPolicyCountAsSnapshotRetentionPolicy(&automation.SnapshotRetentionPolicyCount{
			Kind:  automation.SNAPSHOTRETENTIONPOLICYCOUNTKIND_COUNT,
			Value: model.Value.ValueInt32(),
		}), nil
	case string(automation.SNAPSHOTRETENTIONPOLICYINDEFINITELYKIND_INDEFINITELY):
		if !utils.IsUndefined(model.Value) {
			return automation.SnapshotRetentionPolicy{}, fmt.Errorf("value must not be set when snapshot_retention_policy kind is %q", automation.SNAPSHOTRETENTIONPOLICYINDEFINITELYKIND_INDEFINITELY)
		}
		return automation.SnapshotRetentionPolicyIndefinitelyAsSnapshotRetentionPolicy(&automation.SnapshotRetentionPolicyIndefinitely{
			Kind: automation.SNAPSHOTRETENTIONPOLICYINDEFINITELYKIND_INDEFINITELY,
		}), nil
	default:
		return automation.SnapshotRetentionPolicy{}, fmt.Errorf("unsupported snapshot_retention_policy kind %q", model.Kind.ValueString())
	}
}

func toTriggersPayload(model *triggersModel) *automation.AutomationTriggers {
	if model == nil || model.Schedule == nil {
		return nil
	}
	return &automation.AutomationTriggers{
		Schedule: &automation.AutomationScheduleTrigger{
			Rrule: model.Schedule.Rrule.ValueString(),
		},
	}
}
