package token

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	serviceaccountUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &serviceAccountTokenResource{}
	_ resource.ResourceWithConfigure = &serviceAccountTokenResource{}
)

// Model represents the schema for the service account token resource in Terraform.
type Model struct {
	Id                  types.String `tfsdk:"id"`
	AccessTokenId       types.String `tfsdk:"access_token_id"`
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
	ProjectId           types.String `tfsdk:"project_id"`
	TtlDays             types.Int64  `tfsdk:"ttl_days"`
	RotateWhenChanged   types.Map    `tfsdk:"rotate_when_changed"`
	Token               types.String `tfsdk:"token"`
	Active              types.Bool   `tfsdk:"active"`
	CreatedAt           types.String `tfsdk:"created_at"`
	ValidUntil          types.String `tfsdk:"valid_until"`
}

// NewServiceAccountTokenResource is a helper function to create a new service account access token resource instance.
func NewServiceAccountTokenResource() resource.Resource {
	return &serviceAccountTokenResource{}
}

// serviceAccountResource implements the resource interface for service account access token.
type serviceAccountTokenResource struct {
	client *serviceaccount.APIClient
}

// Configure sets up the API client for the service account resource.
func (r *serviceAccountTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := serviceaccountUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

// Metadata sets the resource type name for the service account resource.
func (r *serviceAccountTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_access_token"
}

// Schema defines the resource schema for the service account access token.
func (r *serviceAccountTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"id":                    "Terraform's internal resource identifier. It is structured as \"`project_id`,`service_account_email`,`access_token_id`\".",
		"main":                  "Service account access token schema.",
		"deprecation_message":   "This resource is scheduled for deprecation and will be removed on December 17, 2025. To ensure a smooth transition, please refer to our migration guide at https://docs.stackit.cloud/stackit/en/deprecation-plan-for-service-account-access-tokens-and-migration-guide-373293307.html for detailed instructions and recommendations.",
		"project_id":            "STACKIT project ID associated with the service account token.",
		"service_account_email": "Email address linked to the service account.",
		"ttl_days":              "Specifies the token's validity duration in days. If unspecified, defaults to 90 days.",
		"rotate_when_changed":   "A map of arbitrary key/value pairs that will force recreation of the token when they change, enabling token rotation based on external conditions such as a rotating timestamp. Changing this forces a new resource to be created.",
		"access_token_id":       "Identifier for the access token linked to the service account.",
		"token":                 "JWT access token for API authentication. Prefixed by 'Bearer' and should be stored securely as it is irretrievable once lost.",
		"active":                "Indicate whether the token is currently active or inactive",
		"created_at":            "Timestamp indicating when the access token was created.",
		"valid_until":           "Estimated expiration timestamp of the access token. For precise validity, check the JWT details.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("%s\n\n!> %s\n%s", descriptions["main"], descriptions["deprecation_message"], markdownDescription),
		Description:         descriptions["main"],
		DeprecationMessage:  descriptions["deprecation_message"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_account_email": schema.StringAttribute{
				Description: descriptions["service_account_email"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl_days": schema.Int64Attribute{
				Description: descriptions["ttl_days"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 180),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Default: int64default.StaticInt64(90),
			},
			"rotate_when_changed": schema.MapAttribute{
				Description: descriptions["rotate_when_changed"],
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"access_token_id": schema.StringAttribute{
				Description: descriptions["access_token_id"],
				Computed:    true,
			},
			"token": schema.StringAttribute{
				Description: descriptions["token"],
				Computed:    true,
				Sensitive:   true,
			},
			"active": schema.BoolAttribute{
				Description: descriptions["active"],
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: descriptions["created_at"],
				Computed:    true,
			},
			"valid_until": schema.StringAttribute{
				Description: descriptions["valid_until"],
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state for service accounts.
func (r *serviceAccountTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddWarning(ctx, &resp.Diagnostics, "stackit_service_account_access_token resource deprecated", "use stackit_service_account_key resource instead")
	// Retrieve the planned values for the resource.
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set logging context with the project ID and service account email.
	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)

	// Generate the API request payload.
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account access token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Initialize the API request with the required parameters.
	serviceAccountAccessTokenResp, err := r.client.CreateAccessToken(ctx, projectId, serviceAccountEmail).CreateAccessTokenPayload(*payload).Execute()

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Failed to create service account access token", fmt.Sprintf("API call error: %v", err))
		return
	}

	// Map the response to the resource schema.
	err = mapCreateResponse(serviceAccountAccessTokenResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account access token", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Service account access token created")
}

// Read refreshes the Terraform state with the latest service account data.
func (r *serviceAccountTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddWarning(ctx, &resp.Diagnostics, "stackit_service_account_access_token resource deprecated", "use stackit_service_account_key resource instead")
	// Retrieve the current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract the project ID and serviceAccountEmail for the service account.
	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()

	// Fetch the list of service account tokens from the API.
	listSaTokensResp, err := r.client.ListAccessTokens(ctx, projectId, serviceAccountEmail).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		// due to security purposes, attempting to list access tokens for a non-existent Service Account will return 403.
		if ok && oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusForbidden {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account tokens", fmt.Sprintf("Error calling API: %v", err))
		return
	}

	// Iterate over the list of service account tokens to find the one that matches the ID from the state.
	saTokens := *listSaTokensResp.Items
	for i := range saTokens {
		if *saTokens[i].Id != model.AccessTokenId.ValueString() {
			continue
		}

		if !*saTokens[i].Active {
			tflog.Info(ctx, fmt.Sprintf("Service account access token with id %s is not active", model.AccessTokenId.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}

		err = mapListResponse(&saTokens[i], &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account", fmt.Sprintf("Error processing API response: %v", err))
			return
		}

		// Set the updated state.
		diags = resp.State.Set(ctx, &model)
		resp.Diagnostics.Append(diags...)
		return
	}
	// If no matching service account access token is found, remove the resource from the state.
	tflog.Info(ctx, fmt.Sprintf("Service account access token with id %s not found", model.AccessTokenId.ValueString()))
	resp.State.RemoveResource(ctx)
}

// Update attempts to update the resource. In this case, service account token cannot be updated.
// Note: This method is intentionally left without update logic because changes
// to 'project_id', 'service_account_email' or 'ttl_days' require the resource to be entirely replaced.
// As a result, the Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (r *serviceAccountTokenResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Service accounts cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating service account access token", "Service accounts can't be updated")
}

// Delete deletes the service account and removes it from the Terraform state on success.
func (r *serviceAccountTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddWarning(ctx, &resp.Diagnostics, "stackit_service_account_access_token resource deprecated", "use stackit_service_account_key resource instead")
	// Retrieve current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	accessTokenId := model.AccessTokenId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)
	ctx = tflog.SetField(ctx, "access_token_id", accessTokenId)

	// Call API to delete the existing service account.
	err := r.client.DeleteAccessToken(ctx, projectId, serviceAccountEmail, accessTokenId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting service account token", fmt.Sprintf("Calling API: %v", err))
		return
	}
	tflog.Info(ctx, "Service account token deleted")
}

func toCreatePayload(model *Model) (*serviceaccount.CreateAccessTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &serviceaccount.CreateAccessTokenPayload{
		TtlDays: conversion.Int64ValueToPointer(model.TtlDays),
	}, nil
}

func mapCreateResponse(resp *serviceaccount.AccessToken, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if resp.Token == nil {
		return fmt.Errorf("service account token not present")
	}

	if resp.Id == nil {
		return fmt.Errorf("service account id not present")
	}

	var createdAt basetypes.StringValue
	if resp.CreatedAt != nil {
		createdAtValue := *resp.CreatedAt
		createdAt = types.StringValue(createdAtValue.Format(time.RFC3339))
	}

	var validUntil basetypes.StringValue
	if resp.ValidUntil != nil {
		validUntilValue := *resp.ValidUntil
		validUntil = types.StringValue(validUntilValue.Format(time.RFC3339))
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.ServiceAccountEmail.ValueString(), *resp.Id)
	model.AccessTokenId = types.StringPointerValue(resp.Id)
	model.Token = types.StringPointerValue(resp.Token)
	model.Active = types.BoolPointerValue(resp.Active)
	model.CreatedAt = createdAt
	model.ValidUntil = validUntil

	return nil
}

func mapListResponse(resp *serviceaccount.AccessTokenMetadata, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if resp.Id == nil {
		return fmt.Errorf("service account id not present")
	}

	var createdAt basetypes.StringValue
	if resp.CreatedAt != nil {
		createdAtValue := *resp.CreatedAt
		createdAt = types.StringValue(createdAtValue.Format(time.RFC3339))
	}

	var validUntil basetypes.StringValue
	if resp.ValidUntil != nil {
		validUntilValue := *resp.ValidUntil
		validUntil = types.StringValue(validUntilValue.Format(time.RFC3339))
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.ServiceAccountEmail.ValueString(), *resp.Id)
	model.AccessTokenId = types.StringPointerValue(resp.Id)
	model.CreatedAt = createdAt
	model.ValidUntil = validUntil

	return nil
}
