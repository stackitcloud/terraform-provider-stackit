package kms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/stackitcloud/stackit-sdk-go/services/kms/wait"

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
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	kmsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &wrappingKeyResource{}
	_ resource.ResourceWithConfigure   = &wrappingKeyResource{}
	_ resource.ResourceWithImportState = &wrappingKeyResource{}
	_ resource.ResourceWithModifyPlan  = &wrappingKeyResource{}
)

type Model struct {
	AccessScope   types.String `tfsdk:"access_scope"`
	Algorithm     types.String `tfsdk:"algorithm"`
	Description   types.String `tfsdk:"description"`
	DisplayName   types.String `tfsdk:"display_name"`
	Id            types.String `tfsdk:"id"` // needed by TF
	KeyRingId     types.String `tfsdk:"keyring_id"`
	Protection    types.String `tfsdk:"protection"`
	Purpose       types.String `tfsdk:"purpose"`
	ProjectId     types.String `tfsdk:"project_id"`
	Region        types.String `tfsdk:"region"`
	WrappingKeyId types.String `tfsdk:"wrapping_key_id"`
	PublicKey     types.String `tfsdk:"public_key"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func NewWrappingKeyResource() resource.Resource {
	return &wrappingKeyResource{}
}

type wrappingKeyResource struct {
	client       *kms.APIClient
	providerData core.ProviderData
}

func (r *wrappingKeyResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kms_wrapping_key"
}

func (r *wrappingKeyResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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
func (r *wrappingKeyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *wrappingKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "KMS wrapping key resource schema.",
		Attributes: map[string]schema.Attribute{
			"access_scope": schema.StringAttribute{
				Description: fmt.Sprintf("The access scope of the key. Default is `%s`. %s", string(kms.ACCESSSCOPE_PUBLIC), utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedAccessScopeEnumValues)...)),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(string(kms.ACCESSSCOPE_PUBLIC)),
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"algorithm": schema.StringAttribute{
				Description: fmt.Sprintf("The wrapping algorithm used to wrap the key to import. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedWrappingAlgorithmEnumValues)...)),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "A user chosen description to distinguish multiple wrapping keys.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "The display name to distinguish multiple wrapping keys.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`keyring_id`,`wrapping_key_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"keyring_id": schema.StringAttribute{
				Description: "The ID of the associated keyring",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"protection": schema.StringAttribute{
				Description: fmt.Sprintf("The underlying system that is responsible for protecting the key material. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedProtectionEnumValues)...)),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"purpose": schema.StringAttribute{
				Description: fmt.Sprintf("The purpose for which the key will be used. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedWrappingPurposeEnumValues)...)),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
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
			"wrapping_key_id": schema.StringAttribute{
				Description: "The ID of the wrapping key",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"public_key": schema.StringAttribute{
				Description: "The public key of the wrapping key.",
				Computed:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "The date and time the wrapping key will expire.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The date and time the creation of the wrapping key was triggered.",
				Computed:    true,
			},
		},
	}
}

func (r *wrappingKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	keyRingId := model.KeyRingId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "keyring_id", keyRingId)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating wrapping key", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createWrappingKeyResp, err := r.client.CreateWrappingKey(ctx, projectId, region, keyRingId).CreateWrappingKeyPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating wrapping key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if createWrappingKeyResp == nil || createWrappingKeyResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating wrapping key", "API returned empty response")
		return
	}

	wrappingKeyId := *createWrappingKeyResp.Id

	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id":      projectId,
		"region":          region,
		"keyring_id":      keyRingId,
		"wrapping_key_id": wrappingKeyId,
	})

	wrappingKey, err := wait.CreateWrappingKeyWaitHandler(ctx, r.client, projectId, region, keyRingId, wrappingKeyId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error waiting for wrapping key creation", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(wrappingKey, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating wrapping key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key created")
}

func (r *wrappingKeyResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := request.State.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	wrappingKeyId := model.WrappingKeyId.ValueString()

	ctx = tflog.SetField(ctx, "keyring_id", keyRingId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "wrapping_key_id", wrappingKeyId)

	wrappingKeyResponse, err := r.client.GetWrappingKey(ctx, projectId, region, keyRingId, wrappingKeyId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading wrapping key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(wrappingKeyResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading wrapping key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Wrapping key read")
}

func (r *wrappingKeyResource) Update(ctx context.Context, _ resource.UpdateRequest, response *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// wrapping keys cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &response.Diagnostics, "Error updating wrapping key", "Keys can't be updated")
}

func (r *wrappingKeyResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := request.State.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	wrappingKeyId := model.WrappingKeyId.ValueString()

	err := r.client.DeleteWrappingKey(ctx, projectId, region, keyRingId, wrappingKeyId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error deleting wrapping key", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "wrapping key deleted")
}

func (r *wrappingKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing wrapping key",
			fmt.Sprintf("Exptected import identifier with format: [project_id],[region],[keyring_id],[wrapping_key_id], got :%q", req.ID),
		)
		return
	}

	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":      idParts[0],
		"region":          idParts[1],
		"keyring_id":      idParts[2],
		"wrapping_key_id": idParts[3],
	})

	tflog.Info(ctx, "wrapping key state imported")
}

func mapFields(wrappingKey *kms.WrappingKey, model *Model, region string) error {
	if wrappingKey == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var wrappingKeyId string
	if model.WrappingKeyId.ValueString() != "" {
		wrappingKeyId = model.WrappingKeyId.ValueString()
	} else if wrappingKey.Id != nil {
		wrappingKeyId = *wrappingKey.Id
	} else {
		return fmt.Errorf("key id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, model.KeyRingId.ValueString(), wrappingKeyId)
	model.Region = types.StringValue(region)
	model.WrappingKeyId = types.StringValue(wrappingKeyId)
	model.DisplayName = types.StringPointerValue(wrappingKey.DisplayName)
	model.PublicKey = types.StringPointerValue(wrappingKey.PublicKey)
	model.AccessScope = types.StringValue(string(wrappingKey.GetAccessScope()))
	model.Algorithm = types.StringValue(string(wrappingKey.GetAlgorithm()))
	model.Purpose = types.StringValue(string(wrappingKey.GetPurpose()))
	model.Protection = types.StringValue(string(wrappingKey.GetProtection()))

	model.CreatedAt = types.StringNull()
	if wrappingKey.CreatedAt != nil {
		model.CreatedAt = types.StringValue(wrappingKey.CreatedAt.Format(time.RFC3339))
	}

	model.ExpiresAt = types.StringNull()
	if wrappingKey.ExpiresAt != nil {
		model.ExpiresAt = types.StringValue(wrappingKey.ExpiresAt.Format(time.RFC3339))
	}

	// TODO: workaround - remove once STACKITKMS-377 is resolved (just write the return value from the API to the state then)
	if !(model.Description.IsNull() && wrappingKey.Description != nil && *wrappingKey.Description == "") {
		model.Description = types.StringPointerValue(wrappingKey.Description)
	} else {
		model.Description = types.StringNull()
	}

	return nil
}

func toCreatePayload(model *Model) (*kms.CreateWrappingKeyPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	return &kms.CreateWrappingKeyPayload{
		AccessScope: kms.CreateKeyPayloadGetAccessScopeAttributeType(conversion.StringValueToPointer(model.AccessScope)),
		Algorithm:   kms.CreateWrappingKeyPayloadGetAlgorithmAttributeType(conversion.StringValueToPointer(model.Algorithm)),
		Description: conversion.StringValueToPointer(model.Description),
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
		Protection:  kms.CreateKeyPayloadGetProtectionAttributeType(conversion.StringValueToPointer(model.Protection)),
		Purpose:     kms.CreateWrappingKeyPayloadGetPurposeAttributeType(conversion.StringValueToPointer(model.Purpose)),
	}, nil
}
