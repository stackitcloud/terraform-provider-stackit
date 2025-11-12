package kms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"github.com/stackitcloud/stackit-sdk-go/services/kms/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	kmsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

const (
	deletionWarning = "Keyrings will **not** be destroyed by terraform during a `terraform destroy`. They will just be thrown out of the Terraform state and not deleted on API side. **This way we can ensure no keyring setups are deleted by accident and it gives you the option to recover your keys within the grace period.**"
)

var (
	_ resource.Resource                = &keyRingResource{}
	_ resource.ResourceWithConfigure   = &keyRingResource{}
	_ resource.ResourceWithImportState = &keyRingResource{}
	_ resource.ResourceWithModifyPlan  = &keyRingResource{}
)

type Model struct {
	Description types.String `tfsdk:"description"`
	DisplayName types.String `tfsdk:"display_name"`
	KeyRingId   types.String `tfsdk:"keyring_id"`
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	Region      types.String `tfsdk:"region"`
}

func NewKeyRingResource() resource.Resource {
	return &keyRingResource{}
}

type keyRingResource struct {
	client       *kms.APIClient
	providerData core.ProviderData
}

func (r *keyRingResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kms_keyring"
}

func (r *keyRingResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	r.client = kmsUtils.ConfigureClient(ctx, &r.providerData, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "KMS client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *keyRingResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *keyRingResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	description := fmt.Sprintf("KMS Keyring resource schema. %s", core.ResourceRegionFallbackDocstring)

	response.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: fmt.Sprintf("%s\n\n ~> %s", description, deletionWarning),
		Attributes: map[string]schema.Attribute{
			"description": schema.StringAttribute{
				Description: "A user chosen description to distinguish multiple keyrings.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "The display name to distinguish multiple keyrings.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"keyring_id": schema.StringAttribute{
				Description: "An auto generated unique id which identifies the keyring.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`keyring_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the keyring is associated.",
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
				Description: "The resource region. If not defined, the provider region is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *keyRingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating keyring", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	createResponse, err := r.client.CreateKeyRing(ctx, projectId, region).CreateKeyRingPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating keyring", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if createResponse == nil || createResponse.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating keyring", "API returned empty response")
		return
	}

	keyRingId := *createResponse.Id
	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"keyring_id": keyRingId,
	})

	waitResp, err := wait.CreateKeyRingWaitHandler(ctx, r.client, projectId, region, keyRingId).SetSleepBeforeWait(5 * time.Second).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating keyring", fmt.Sprintf("Key Ring creation waiting: %v", err))
		return
	}

	err = mapFields(waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating keyring", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key Ring created")
}

func (r *keyRingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "keyring_id", keyRingId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	keyRingResponse, err := r.client.GetKeyRing(ctx, projectId, region, keyRingId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading keyring", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(keyRingResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading keyring", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key ring read")
}

func (r *keyRingResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// keyrings cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating keyring", "Keyrings can't be updated")
}

func (r *keyRingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The keyring can't be deleted by Terraform because it potentially has still keys inside it.
	// These keys might be *scheduled* for deletion, but aren't deleted completely, so the delete request would fail.
	core.LogAndAddWarning(ctx, &resp.Diagnostics, "Keyring not deleted on API side", deletionWarning)

	tflog.Info(ctx, "keyring deleted")
}

func (r *keyRingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing keyring",
			fmt.Sprintf("Exptected import identifier with format: [project_id],[region],[keyring_id], got :%q", req.ID),
		)
		return
	}

	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"keyring_id": idParts[2],
	})

	tflog.Info(ctx, "keyring state imported")
}

func mapFields(keyRing *kms.KeyRing, model *Model, region string) error {
	if keyRing == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var keyRingId string
	if model.KeyRingId.ValueString() != "" {
		keyRingId = model.KeyRingId.ValueString()
	} else if keyRing.Id != nil {
		keyRingId = *keyRing.Id
	} else {
		return fmt.Errorf("keyring id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, keyRingId)
	model.KeyRingId = types.StringValue(keyRingId)
	model.DisplayName = types.StringPointerValue(keyRing.DisplayName)
	model.Region = types.StringValue(region)

	// TODO: workaround - remove once STACKITKMS-377 is resolved (just write the return value from the API to the state then)
	if !(model.Description.IsNull() && keyRing.Description != nil && *keyRing.Description == "") {
		model.Description = types.StringPointerValue(keyRing.Description)
	} else {
		model.Description = types.StringNull()
	}

	return nil
}

func toCreatePayload(model *Model) (*kms.CreateKeyRingPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &kms.CreateKeyRingPayload{
		Description: conversion.StringValueToPointer(model.Description),
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
	}, nil
}
