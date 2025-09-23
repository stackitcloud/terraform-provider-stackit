package kms

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
)

type Model struct {
	Algorithm     types.String `tfsdk:"algorithm"`
	Backend       types.String `tfsdk:"backend"`
	Description   types.String `tfsdk:"description"`
	DisplayName   types.String `tfsdk:"display_name"`
	Id            types.String `tfsdk:"id"` // needed by TF
	KeyRingId     types.String `tfsdk:"key_ring_id"`
	Purpose       types.String `tfsdk:"purpose"`
	ProjectId     types.String `tfsdk:"project_id"`
	Region        types.String `tfsdk:"region"`
	WrappingKeyId types.String `tfsdk:"wrapping_key_id"`
}

func NewWrappingKeyResource() resource.Resource {
	return &wrappingKeyResource{}
}

type wrappingKeyResource struct {
	client       *kms.APIClient
	providerData core.ProviderData
}

func (w *wrappingKeyResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kms_wrapping_key"
}

func (w *wrappingKeyResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	var ok bool
	w.providerData, ok = conversion.ParseProviderData(ctx, request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}
	apiClient := kmsUtils.ConfigureClient(ctx, &w.providerData, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	w.client = apiClient
}

func (w *wrappingKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":            "KMS Key resource schema. Must have a `region` specified in the provider configuration.",
		"algorithm":       "The encryption algorithm that the key will use to encrypt data",
		"backend":         "The backend that is used for KMS. Right now, only software is accepted.",
		"description":     "A user chosen description to distinguish multiple keys",
		"display_name":    "The display name to distinguish multiple keys",
		"id":              "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
		"key_ring_id":     "The ID of the associated key ring",
		"purpose":         "The purpose for which the key will be used",
		"project_id":      "STACKIT project ID to which the key ring is associated.",
		"region":          "The STACKIT region name the key ring is located in.",
		"wrapping_key_id": "The ID of the wrapping key",
	}

	response.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"algorithm": schema.StringAttribute{
				Description: descriptions["algorithm"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"backend": schema.StringAttribute{
				Description: descriptions["backend"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key_ring_id": schema.StringAttribute{
				Description: descriptions["key_ring_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"purpose": schema.StringAttribute{
				Description: descriptions["purpose"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
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
				Optional:    true,
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"wrapping_key_id": schema.StringAttribute{
				Description: descriptions["wrapping_key_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
		},
	}
}

func (w *wrappingKeyResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model

	diags := request.Plan.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := w.providerData.GetRegionWithOverride(model.Region)
	keyRingId := model.KeyRingId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "key_ring_id", keyRingId)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating wrapping key", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResponse, err := w.client.CreateWrappingKey(ctx, projectId, region, keyRingId).CreateWrappingKeyPayload(*payload).Execute()

	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating wrapping key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	wrappingKeyId := *createResponse.Id
	ctx = tflog.SetField(ctx, "wrapping_key_id", wrappingKeyId)

	err = mapFields(createResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating wrapping key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key created")
}

func (w *wrappingKeyResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := request.State.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := w.providerData.GetRegionWithOverride(model.Region)
	wrappingKeyId := model.WrappingKeyId.ValueString()

	ctx = tflog.SetField(ctx, "key_ring_id", keyRingId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "wrapping_key_id", wrappingKeyId)

	wrappingKeyResponse, err := w.client.GetWrappingKey(ctx, projectId, region, keyRingId, wrappingKeyId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
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

func (w *wrappingKeyResource) Update(ctx context.Context, _ resource.UpdateRequest, response *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// wrapping keys cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &response.Diagnostics, "Error updating wrapping key", "Keys can't be updated")
}

func (w *wrappingKeyResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := request.State.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := w.providerData.GetRegionWithOverride(model.Region)
	wrappingKeyId := model.WrappingKeyId.ValueString()

	err := w.client.DeleteWrappingKey(ctx, projectId, region, keyRingId, wrappingKeyId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error deleting wrapping key", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "wrapping key deleted")
}

func (w *wrappingKeyResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	idParts := strings.Split(request.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &response.Diagnostics,
			"Error importing wrapping key",
			fmt.Sprintf("Exptected import identifier with format: [proejct_id],[instance_id], got :%q", request.ID),
		)
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("wrapping_key_id"), idParts[1])...)
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), wrappingKeyId)
	model.WrappingKeyId = types.StringValue(wrappingKeyId)
	model.DisplayName = types.StringPointerValue(wrappingKey.DisplayName)
	model.Description = types.StringPointerValue(wrappingKey.Description)
	model.Region = types.StringValue(region)

	return nil
}

func toCreatePayload(model *Model) (*kms.CreateWrappingKeyPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	return &kms.CreateWrappingKeyPayload{
		Algorithm:   kms.CreateWrappingKeyPayloadGetAlgorithmAttributeType(conversion.StringValueToPointer(model.Algorithm)),
		Backend:     kms.CreateKeyPayloadGetBackendAttributeType(conversion.StringValueToPointer(model.Backend)),
		Description: conversion.StringValueToPointer(model.Description),
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
		Purpose:     kms.CreateWrappingKeyPayloadGetPurposeAttributeType(conversion.StringValueToPointer(model.Purpose)),
	}, nil
}
