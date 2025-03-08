package account

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &serviceAccountResource{}
	_ resource.ResourceWithConfigure   = &serviceAccountResource{}
	_ resource.ResourceWithImportState = &serviceAccountResource{}
)

// Model represents the schema for the service account resource.
type Model struct {
	Id        types.String `tfsdk:"id"`         // Required by Terraform
	ProjectId types.String `tfsdk:"project_id"` // ProjectId associated with the service account
	Name      types.String `tfsdk:"name"`       // Name of the service account
	Email     types.String `tfsdk:"email"`      // Email linked to the service account
}

// NewServiceAccountResource is a helper function to create a new service account resource instance.
func NewServiceAccountResource() resource.Resource {
	return &serviceAccountResource{}
}

// serviceAccountResource implements the resource interface for service accounts.
type serviceAccountResource struct {
	client *serviceaccount.APIClient
}

// Configure sets up the API client for the service account resource.
func (r *serviceAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent potential panics if the provider is not properly configured.
	if req.ProviderData == nil {
		return
	}

	// Validate provider data type before proceeding.
	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !resourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_service_account", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	// Initialize the API client with the appropriate authentication and endpoint settings.
	var apiClient *serviceaccount.APIClient
	var err error
	if providerData.ServiceAccountCustomEndpoint != "" {
		apiClient, err = serviceaccount.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ServiceAccountCustomEndpoint),
		)
	} else {
		apiClient, err = serviceaccount.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	// Handle API client initialization errors.
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	// Store the initialized client.
	r.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

// Metadata sets the resource type name for the service account resource.
func (r *serviceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

// Schema defines the schema for the resource.
func (r *serviceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"id":         "Terraform's internal resource ID, structured as \"project_id,email\".",
		"project_id": "STACKIT project ID to which the service account is associated.",
		"name":       "Name of the service account.",
		"email":      "Email of the service account.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Schema for a STACKIT service account resource."),
		Description:         "Schema for a STACKIT service account resource.",
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
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(20),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)*$`), "must start with a lowercase letter, can contain lowercase letters, numbers, and dashes, but cannot start or end with a dash, and dashes cannot be consecutive"),
				},
			},
			"email": schema.StringAttribute{
				Description: descriptions["email"],
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state for service accounts.
func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the planned values for the resource.
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set logging context with the project ID and service account name.
	projectId := model.ProjectId.ValueString()
	serviceAccountName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_name", serviceAccountName)

	// Generate the API request payload.
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create the new service account via the API client.
	serviceAccountResp, err := r.client.CreateServiceAccount(ctx, projectId).CreateServiceAccountPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Set the service account name and map the response to the resource schema.
	model.Name = types.StringValue(serviceAccountName)
	err = mapCreateOrListResponse(serviceAccountResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// This sleep is currently needed due to the IAM Cache.
	time.Sleep(5 * time.Second)

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Service account created")
}

// Read refreshes the Terraform state with the latest service account data.
func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract the project ID for the service account.
	projectId := model.ProjectId.ValueString()

	// Fetch the list of service accounts from the API.
	listSaResp, err := r.client.ListServiceAccounts(ctx, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account", fmt.Sprintf("Error calling API: %v", err))
		return
	}

	// Iterate over the list of service accounts to find the one that matches the email from the state.
	serviceAccounts := *listSaResp.Items
	for i := range serviceAccounts {
		if *serviceAccounts[i].Email != model.Email.ValueString() {
			continue
		}

		// Map the response data to the resource schema and update the state.
		err = mapCreateOrListResponse(&serviceAccounts[i], &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account", fmt.Sprintf("Error processing API response: %v", err))
			return
		}

		// Set the updated state.
		diags = resp.State.Set(ctx, &model)
		resp.Diagnostics.Append(diags...)
		return
	}

	// If no matching service account is found, remove the resource from the state.
	resp.State.RemoveResource(ctx)
}

// Update attempts to update the resource. In this case, service accounts cannot be updated.
// Note: This method is intentionally left without update logic because changes
// to 'project_id' or 'name' require the resource to be entirely replaced.
// As a result, the Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (r *serviceAccountResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Service accounts cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating service account", "Service accounts can't be updated")
}

// Delete deletes the service account and removes it from the Terraform state on success.
func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	serviceAccountName := model.Name.ValueString()
	serviceAccountEmail := model.Email.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_name", serviceAccountName)

	// Call API to delete the existing service account.
	err := r.client.DeleteServiceAccount(ctx, projectId, serviceAccountEmail).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting service account", fmt.Sprintf("Calling API: %v", err))
		return
	}
	tflog.Info(ctx, "Service account deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,email
func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Split the import identifier to extract project ID and email.
	idParts := strings.Split(req.ID, core.Separator)

	// Ensure the import identifier format is correct.
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing service account",
			fmt.Sprintf("Expected import identifier with format: [project_id],[email]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	email := idParts[1]

	// Attempt to parse the name from the email if valid.
	name, err := parseNameFromEmail(email)
	if name != "" && err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	}

	// Set the project ID and email attributes in the state.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), email)...)
	tflog.Info(ctx, "Service account state imported")
}

// toCreatePayload generates the payload to create a new service account.
func toCreatePayload(model *Model) (*serviceaccount.CreateServiceAccountPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &serviceaccount.CreateServiceAccountPayload{
		Name: conversion.StringValueToPointer(model.Name),
	}, nil
}

// mapCreateOrListResponse maps a ServiceAccount response to the model.
func mapCreateOrListResponse(resp *serviceaccount.ServiceAccount, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if resp.Email == nil {
		return fmt.Errorf("service account email not present")
	}

	// Build the ID by combining the project ID and email and assign the model's fields.
	idParts := []string{model.ProjectId.ValueString(), *resp.Email}
	model.Id = types.StringValue(strings.Join(idParts, core.Separator))
	model.Email = types.StringPointerValue(resp.Email)
	model.ProjectId = types.StringPointerValue(resp.ProjectId)

	return nil
}

// parseNameFromEmail extracts the name component from an email address.
// The email format must be `name-<random7characters>@sa.stackit.cloud`.
func parseNameFromEmail(email string) (string, error) {
	namePattern := `^([a-z][a-z0-9]*(?:-[a-z0-9]+)*)-\w{7}@sa\.stackit\.cloud$`
	re := regexp.MustCompile(namePattern)
	match := re.FindStringSubmatch(email)

	// If a match is found, return the name component
	if len(match) > 1 {
		return match[1], nil
	}

	// If no match is found, return an error
	return "", fmt.Errorf("unable to parse name from email")
}
