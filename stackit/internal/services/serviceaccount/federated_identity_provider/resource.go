package federated_identity_provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	serviceaccountUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

var (
	_ resource.Resource                = &ServiceAccountFederatedIdentityProviderResource{}
	_ resource.ResourceWithConfigure   = &ServiceAccountFederatedIdentityProviderResource{}
	_ resource.ResourceWithImportState = &ServiceAccountFederatedIdentityProviderResource{}
)

// Model describes the resource data model.
type Model struct {
	Id                  types.String `tfsdk:"id"`
	ProjectId           types.String `tfsdk:"project_id"`
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
	FederationId        types.String `tfsdk:"federation_id"`
	Name                types.String `tfsdk:"name"`
	Issuer              types.String `tfsdk:"issuer"`
	Assertions          types.List   `tfsdk:"assertions"`
}

// AssertionModel describes an assertion in the assertions list.
type AssertionModel struct {
	Item     types.String `tfsdk:"item"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

func NewServiceAccountFederatedIdentityProviderResource() resource.Resource {
	return &ServiceAccountFederatedIdentityProviderResource{}
}

type ServiceAccountFederatedIdentityProviderResource struct {
	client *serviceaccount.APIClient
}

func (r *ServiceAccountFederatedIdentityProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_federated_identity_provider"
}

func (r *ServiceAccountFederatedIdentityProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"id":                    "Terraform's internal resource identifier. It is structured as \"`project_id`,`service_account_email`,`federation_id`\".",
		"main":                  "Service account federated identity provider schema.",
		"project_id":            "The STACKIT project ID associated with the service account.",
		"federation_id":         "The unique identifier for the federated identity provider associated with the service account.",
		"service_account_email": "The email address associated with the service account, used for account identification and communication.",
		"name":                  "The name of the federated identity provider.",
		"issuer":                "The issuer URL.",
		"assertions":            "The assertions for the federated identity provider.",
		"assertions.item":       "The assertion claim. At least one assertion with the claim \"aud\" is required for security reasons.",
		"assertions.operator":   "The assertion operator. Currently, the only supported operator is \"equals\".",
		"assertions.value":      "The assertion value.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("%s%s", descriptions["main"], markdownDescription),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: descriptions["id"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: descriptions["project_id"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"service_account_email": schema.StringAttribute{
				Required:    true,
				Description: descriptions["service_account_email"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"federation_id": schema.StringAttribute{
				Description: descriptions["federation_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: descriptions["name"],
			},
			"issuer": schema.StringAttribute{
				Required:    true,
				Description: descriptions["issuer"],
			},
			"assertions": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"item": schema.StringAttribute{
							Required:    true,
							Description: descriptions["assertions.item"],
						},
						"operator": schema.StringAttribute{
							Required:    true,
							Description: descriptions["assertions.operator"],
							Validators: []validator.String{
								stringvalidator.OneOf("equals"),
							},
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: descriptions["assertions.value"],
						},
					},
				},
				Required: true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(50), // This is the current page limit for assertions.
					requireAssertions(),
				},
				Description: descriptions["assertions"],
			},
		},
	}
}

func (r *ServiceAccountFederatedIdentityProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceAccountFederatedIdentityProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating federated identity provider", fmt.Sprintf("failed to convert model to payload: %v", err))
		return
	}

	apiResp, err := r.client.DefaultAPI.CreateFederatedIdentityProvider(ctx, projectId, serviceAccountEmail).
		CreateFederatedIdentityProviderPayload(*payload).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating federated identity provider", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if err := mapFields(ctx, apiResp, &model, projectId, serviceAccountEmail); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating federated identity provider", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ServiceAccountFederatedIdentityProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	federationId := model.FederationId.ValueString()

	apiResp, err := r.client.DefaultAPI.ListFederatedIdentityProviders(ctx, projectId, serviceAccountEmail).
		Execute()

	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		// due to security purposes, attempting to get access federation for a non-existent Service Account will return 403.
		if ok && oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusForbidden || oapiErr.StatusCode == http.StatusBadRequest {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading federated identity provider", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if len(apiResp.Resources) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	var found *serviceaccount.FederatedIdentityProvider
	for i, provider := range apiResp.Resources {
		if provider.Id != nil && *provider.Id == federationId {
			found = &(apiResp.Resources)[i]
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := mapFields(ctx, found, &model, projectId, serviceAccountEmail); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading federated identity provider", fmt.Sprintf("failed to map response to model: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ServiceAccountFederatedIdentityProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { //nolint:gocritic // function signature required by Terraform
	// Read the plan to get the desired configuration
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// federation_id is a computed field only available in the current state, not the plan
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.FederationId = stateModel.FederationId

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	federationId := model.FederationId.ValueString()

	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating federated identity provider", fmt.Sprintf("failed to convert model to payload: %v", err))
		return
	}

	apiResp, err := r.client.DefaultAPI.PartialUpdateServiceAccountFederatedIdentityProvider(ctx, projectId, serviceAccountEmail, federationId).
		PartialUpdateServiceAccountFederatedIdentityProviderPayload(*payload).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating federated identity provider", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if err := mapFields(ctx, apiResp, &model, projectId, serviceAccountEmail); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating federated identity provider", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ServiceAccountFederatedIdentityProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	federationId := model.FederationId.ValueString()

	err := r.client.DefaultAPI.DeleteServiceFederatedIdentityProvider(ctx, projectId, serviceAccountEmail, federationId).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting federated identity provider", fmt.Sprintf("Calling API: %v", err))
		return
	}
}

func (r *ServiceAccountFederatedIdentityProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapFields(ctx context.Context, apiResp *serviceaccount.FederatedIdentityProvider, model *Model, projectId, serviceAccountEmail string) error {
	if apiResp == nil {
		return fmt.Errorf("apiResp is nil")
	}

	federationId := ""
	if apiResp.Id != nil {
		federationId = *apiResp.Id
	}
	model.Id = utils.BuildInternalTerraformId(projectId, serviceAccountEmail, federationId)
	model.ProjectId = types.StringValue(projectId)
	model.ServiceAccountEmail = types.StringValue(serviceAccountEmail)
	if federationId != "" {
		model.FederationId = types.StringValue(federationId)
	} else {
		model.FederationId = types.StringNull()
	}

	if apiResp.Name != "" {
		model.Name = types.StringValue(apiResp.Name)
	} else {
		model.Name = types.StringNull()
	}

	if apiResp.Issuer != "" {
		model.Issuer = types.StringValue(apiResp.Issuer)
	} else {
		model.Issuer = types.StringNull()
	}

	// Map assertions
	if len(apiResp.Assertions) > 0 {
		assertions := make([]AssertionModel, len(apiResp.Assertions))
		for i, assertion := range apiResp.Assertions {
			assertions[i] = AssertionModel{
				Item:     types.StringValue(assertion.Item),
				Operator: types.StringValue(assertion.Operator),
				Value:    types.StringValue(assertion.Value),
			}
		}

		assertionsValue, _ := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"item":     types.StringType,
				"operator": types.StringType,
				"value":    types.StringType,
			},
		}, assertions)
		model.Assertions = assertionsValue
	} else {
		model.Assertions = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"item":     types.StringType,
				"operator": types.StringType,
				"value":    types.StringType,
			},
		})
	}

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*serviceaccount.CreateFederatedIdentityProviderPayload, error) {
	payload := &serviceaccount.CreateFederatedIdentityProviderPayload{
		Name:   model.Name.ValueString(),
		Issuer: model.Issuer.ValueString(),
	}

	if !model.Assertions.IsNull() {
		var assertions []AssertionModel
		diags := model.Assertions.ElementsAs(ctx, &assertions, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to extract assertions from model")
		}

		assertionsPayload := make([]serviceaccount.CreateFederatedIdentityProviderPayloadAssertionsInner, len(assertions))
		for i, assertion := range assertions {
			assertionsPayload[i] = serviceaccount.CreateFederatedIdentityProviderPayloadAssertionsInner{
				Item:     conversion.StringValueToPointer(assertion.Item),
				Operator: conversion.StringValueToPointer(assertion.Operator),
				Value:    conversion.StringValueToPointer(assertion.Value),
			}
		}
		payload.Assertions = assertionsPayload
	}

	return payload, nil
}

func toUpdatePayload(ctx context.Context, model *Model) (*serviceaccount.PartialUpdateServiceAccountFederatedIdentityProviderPayload, error) {
	payload := &serviceaccount.PartialUpdateServiceAccountFederatedIdentityProviderPayload{}

	if !model.Issuer.IsNull() {
		payload.Issuer = model.Issuer.ValueString()
	}
	if !model.Name.IsNull() {
		payload.Name = model.Name.ValueString()
	}
	if !model.Assertions.IsNull() {
		var assertions []AssertionModel
		diags := model.Assertions.ElementsAs(ctx, &assertions, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to extract assertions from model")
		}

		assertionsPayload := make([]serviceaccount.PartialUpdateServiceAccountFederatedIdentityProviderPayloadAssertionsInner, len(assertions))
		for i, assertion := range assertions {
			assertionsPayload[i] = serviceaccount.PartialUpdateServiceAccountFederatedIdentityProviderPayloadAssertionsInner{
				Item:     conversion.StringValueToPointer(assertion.Item),
				Operator: conversion.StringValueToPointer(assertion.Operator),
				Value:    conversion.StringValueToPointer(assertion.Value),
			}
		}
		payload.Assertions = assertionsPayload
	}

	return payload, nil
}
