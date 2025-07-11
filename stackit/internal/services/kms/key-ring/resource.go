package kms

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	kmsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

type Model struct {
	Description types.String `tfsdk:"description"`
	DisplayName types.String `tfsdk:"display_name"`
	KeyRingId   types.String `tfsdk:"key_ring_id"`
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	RegionId    types.String `tfsdk:"region_id"`
}

func NewKeyRingResource() resource.Resource {
	return &keyRingResource{}
}

type keyRingResource struct {
	client *kms.APIClient
}

func (k *keyRingResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "kms_key_ring"
}

func (k *keyRingResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	apiClient := kmsUtils.ConfigureClient(ctx, &providerData, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	k.client = apiClient
}

func (k *keyRingResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":         "KMS Key Ring resource schema.",
		"description":  "A user chosen description to distinguish multiple key rings.",
		"display_name": "The display name to distinguish multiple key rings.",
		"key_ring_id":  "An auto generated unique id which identifies the key ring.",
		"id":           "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
		"project_id":   "STACKIT project ID to which the key ring is associated.",
		"region_id":    "The STACKIT region name the key ring is located in.",
	}

	response.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
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
				Description: descriptions["description"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"key_ring_id": schema.StringAttribute{
				Description: descriptions["key_ring_id"],
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
		},
	}

}

func (k *keyRingResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var model Model
	diags := request.Plan.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	regionId := model.RegionId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region_id", regionId)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating key ring", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	createResponse, err := k.client.CreateKeyRing(ctx, projectId, regionId).CreateKeyRingPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating key ring", fmt.Sprintf("Calling API: %v", err))
		return
	}

	keyRingId := *createResponse.Id
	ctx = tflog.SetField(ctx, "key_ring_id", keyRingId)

	err = mapFields(createResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating key ring", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key Ring created")
}

func (k *keyRingResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	//TODO implement me
	panic("implement me")
}

func (k *keyRingResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	//TODO implement me
	panic("implement me")
}

func (k *keyRingResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	//TODO implement me
	panic("implement me")
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

func mapFields(keyRing *kms.KeyRing, model *Model) error {
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), keyRingId)
	model.KeyRingId = types.StringValue(keyRingId)
	model.DisplayName = types.StringPointerValue(keyRing.DisplayName)
	model.Description = types.StringPointerValue(keyRing.Description)

	return nil
}
