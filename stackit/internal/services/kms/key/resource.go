package kms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	kmsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

const (
	deletionWarning = "Keys will **not** be instantly destroyed by terraform during a `terraform destroy`. They will just be scheduled for deletion via the API and thrown out of the Terraform state afterwards. **This way we can ensure no key setups are deleted by accident and it gives you the option to recover your keys within the grace period.**"
)

var (
	_ resource.Resource                = &keyResource{}
	_ resource.ResourceWithConfigure   = &keyResource{}
	_ resource.ResourceWithImportState = &keyResource{}
	_ resource.ResourceWithModifyPlan  = &keyResource{}
)

type Model struct {
	AccessScope types.String `tfsdk:"access_scope"`
	Algorithm   types.String `tfsdk:"algorithm"`
	Description types.String `tfsdk:"description"`
	DisplayName types.String `tfsdk:"display_name"`
	Id          types.String `tfsdk:"id"` // needed by TF
	ImportOnly  types.Bool   `tfsdk:"import_only"`
	KeyId       types.String `tfsdk:"key_id"`
	KeyRingId   types.String `tfsdk:"keyring_id"`
	Protection  types.String `tfsdk:"protection"`
	Purpose     types.String `tfsdk:"purpose"`
	ProjectId   types.String `tfsdk:"project_id"`
	Region      types.String `tfsdk:"region"`
}

func NewKeyResource() resource.Resource {
	return &keyResource{}
}

type keyResource struct {
	client       *kms.APIClient
	providerData core.ProviderData
}

func (r *keyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_key"
}

func (r *keyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.client = kmsUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "KMS client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *keyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *keyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := fmt.Sprintf("KMS Key resource schema. %s", core.ResourceRegionFallbackDocstring)
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: fmt.Sprintf("%s\n\n ~> %s", description, deletionWarning),
		Attributes: map[string]schema.Attribute{
			"access_scope": schema.StringAttribute{
				Description: fmt.Sprintf("The access scope of the key. Default is `%s`. %s", string(kms.ACCESSSCOPE_PUBLIC), utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedAccessScopeEnumValues)...)),
				Optional:    true,
				Computed:    true, // must be computed because of default value
				Default:     stringdefault.StaticString(string(kms.ACCESSSCOPE_PUBLIC)),
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"algorithm": schema.StringAttribute{
				Description: fmt.Sprintf("The encryption algorithm that the key will use to encrypt data. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedAlgorithmEnumValues)...)),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "A user chosen description to distinguish multiple keys",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "The display name to distinguish multiple keys",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`keyring_id`,`key_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"import_only": schema.BoolAttribute{
				Description: "States whether versions can be created or only imported.",
				Computed:    true,
				Optional:    true,
			},
			"key_id": schema.StringAttribute{
				Description: "The ID of the key",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
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
				Description: fmt.Sprintf("The purpose for which the key will be used. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedPurposeEnumValues)...)),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the key is associated.",
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

func (r *keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating key", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResponse, err := r.client.CreateKey(ctx, projectId, region, keyRingId).CreateKeyPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if createResponse == nil || createResponse.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating key", "API returned empty response")
		return
	}

	keyId := *createResponse.Id
	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"keyring_id": keyRingId,
		"key_id":     keyId,
	})

	waitHandlerResp, err := wait.CreateOrUpdateKeyWaitHandler(ctx, r.client, projectId, region, keyRingId, keyId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error waiting for key creation", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(waitHandlerResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key created")
}

func (r *keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	keyId := model.KeyId.ValueString()

	ctx = tflog.SetField(ctx, "keyring_id", keyRingId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "key_id", keyId)

	keyResponse, err := r.client.GetKey(ctx, projectId, region, keyRingId, keyId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(keyResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key read")
}

func (r *keyResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// keys cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating key", "Keys can't be updated")
}

func (r *keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	keyId := model.KeyId.ValueString()

	err := r.client.DeleteKey(ctx, projectId, region, keyRingId, keyId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting key", fmt.Sprintf("Calling API: %v", err))
	}

	// The keys can't be deleted instantly by Terraform, they can only be scheduled for deletion via the API.
	core.LogAndAddWarning(ctx, &resp.Diagnostics, "Key scheduled for deletion on API side", deletionWarning)

	tflog.Info(ctx, "key deleted")
}

func (r *keyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing key",
			fmt.Sprintf("Exptected import identifier with format: [project_id],[region],[keyring_id],[key_id], got :%q", req.ID),
		)
		return
	}

	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"keyring_id": idParts[2],
		"key_id":     idParts[3],
	})

	tflog.Info(ctx, "key state imported")
}

func mapFields(key *kms.Key, model *Model, region string) error {
	if key == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var keyId string
	if model.KeyId.ValueString() != "" {
		keyId = model.KeyId.ValueString()
	} else if key.Id != nil {
		keyId = *key.Id
	} else {
		return fmt.Errorf("key id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, model.KeyRingId.ValueString(), keyId)
	model.KeyId = types.StringValue(keyId)
	model.DisplayName = types.StringPointerValue(key.DisplayName)
	model.Region = types.StringValue(region)
	model.ImportOnly = types.BoolPointerValue(key.ImportOnly)
	model.AccessScope = types.StringValue(string(key.GetAccessScope()))
	model.Algorithm = types.StringValue(string(key.GetAlgorithm()))
	model.Purpose = types.StringValue(string(key.GetPurpose()))
	model.Protection = types.StringValue(string(key.GetProtection()))

	// TODO: workaround - remove once STACKITKMS-377 is resolved (just write the return value from the API to the state then)
	if !(model.Description.IsNull() && key.Description != nil && *key.Description == "") {
		model.Description = types.StringPointerValue(key.Description)
	} else {
		model.Description = types.StringNull()
	}

	return nil
}

func toCreatePayload(model *Model) (*kms.CreateKeyPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	return &kms.CreateKeyPayload{
		AccessScope: kms.CreateKeyPayloadGetAccessScopeAttributeType(conversion.StringValueToPointer(model.AccessScope)),
		Algorithm:   kms.CreateKeyPayloadGetAlgorithmAttributeType(conversion.StringValueToPointer(model.Algorithm)),
		Protection:  kms.CreateKeyPayloadGetProtectionAttributeType(conversion.StringValueToPointer(model.Protection)),
		Description: conversion.StringValueToPointer(model.Description),
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
		ImportOnly:  conversion.BoolValueToPointer(model.ImportOnly),
		Purpose:     kms.CreateKeyPayloadGetPurposeAttributeType(conversion.StringValueToPointer(model.Purpose)),
	}, nil
}
