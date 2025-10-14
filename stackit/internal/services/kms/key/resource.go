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
	_ resource.Resource                = &keyResource{}
	_ resource.ResourceWithConfigure   = &keyResource{}
	_ resource.ResourceWithImportState = &keyResource{}
)

type Model struct {
	AccessScope types.String `tfsdk:"access_scope"`
	Algorithm   types.String `tfsdk:"algorithm"`
	Description types.String `tfsdk:"description"`
	DisplayName types.String `tfsdk:"display_name"`
	Id          types.String `tfsdk:"id"` // needed by TF
	ImportOnly  types.Bool   `tfsdk:"import_only"`
	KeyId       types.String `tfsdk:"key_id"`
	KeyRingId   types.String `tfsdk:"key_ring_id"`
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

func (k *keyResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kms_key"
}

func (k *keyResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	var ok bool
	k.providerData, ok = conversion.ParseProviderData(ctx, request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}
	apiClient := kmsUtils.ConfigureClient(ctx, &k.providerData, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	k.client = apiClient
}

func (k *keyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":         "KMS Key resource schema. Must have a `region` specified in the provider configuration.",
		"access_scope": "The access scope of the key. Default is PUBLIC.",
		"algorithm":    "The encryption algorithm that the key will use to encrypt data",
		"description":  "A user chosen description to distinguish multiple keys",
		"display_name": "The display name to distinguish multiple keys",
		"id":           "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
		"import_only":  "Specifies if the the key should be import_only",
		"key_id":       "The ID of the key",
		"key_ring_id":  "The ID of the associated key ring",
		"protection":   "The underlying system that is responsible for protecting the key material. Currently only software is accepted.",
		"purpose":      "The purpose for which the key will be used",
		"project_id":   "STACKIT project ID to which the key ring is associated.",
		"region":       "The STACKIT region name the key ring is located in.",
	}

	response.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"access_scope": schema.StringAttribute{
				Description: descriptions["access_scope"],
				Optional:    true,
				Required:    false,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
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
			"import_only": schema.BoolAttribute{
				Description: descriptions["import_only"],
				Computed:    true,
				Required:    false,
			},
			"key_id": schema.StringAttribute{
				Description: descriptions["key_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
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
			"protection": schema.StringAttribute{
				Description: descriptions["protection"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
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
		},
	}
}

func (k *keyResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := request.Plan.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := k.providerData.GetRegionWithOverride(model.Region)
	keyRingId := model.KeyRingId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "key_ring_id", keyRingId)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating key", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	createResponse, err := k.client.CreateKey(ctx, projectId, region, keyRingId).CreateKeyPayload(*payload).Execute()

	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	keyId := *createResponse.Id
	ctx = tflog.SetField(ctx, "key_id", keyId)

	err = mapFields(createResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key created")
}

func (k *keyResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := request.State.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := k.providerData.GetRegionWithOverride(model.Region)
	keyId := model.KeyId.ValueString()

	ctx = tflog.SetField(ctx, "key_ring_id", keyRingId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "key_id", keyId)

	keyResponse, err := k.client.GetKey(ctx, projectId, region, keyRingId, keyId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(keyResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key read")
}

func (k *keyResource) Update(ctx context.Context, _ resource.UpdateRequest, response *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// keys cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &response.Diagnostics, "Error updating key", "Keys can't be updated")
}

func (k *keyResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := request.State.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := k.providerData.GetRegionWithOverride(model.Region)
	keyId := model.KeyId.ValueString()

	err := k.client.DeleteKey(ctx, projectId, region, keyRingId, keyId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error deleting key", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "key deleted")
}

func (k *keyResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	idParts := strings.Split(request.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &response.Diagnostics,
			"Error importing key",
			fmt.Sprintf("Exptected import identifier with format: [proejct_id],[instance_id], got :%q", request.ID),
		)
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("key_id"), idParts[1])...)
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), keyId)
	model.KeyId = types.StringValue(keyId)
	model.DisplayName = types.StringPointerValue(key.DisplayName)
	model.Description = types.StringPointerValue(key.Description)
	model.Region = types.StringValue(region)

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
