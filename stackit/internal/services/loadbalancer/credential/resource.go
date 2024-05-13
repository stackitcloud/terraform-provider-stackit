package loadbalancer

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &credentialResource{}
	_ resource.ResourceWithConfigure   = &credentialResource{}
	_ resource.ResourceWithImportState = &credentialResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	DisplayName    types.String `tfsdk:"display_name"`
	Username       types.String `tfsdk:"username"`
	Password       types.String `tfsdk:"password"`
	CredentialsRef types.String `tfsdk:"credentials_ref"`
}

// NewCredentialResource is a helper function to simplify the provider implementation.
func NewCredentialResource() resource.Resource {
	return &credentialResource{}
}

// credentialResource is the resource implementation.
type credentialResource struct {
	client *loadbalancer.APIClient
}

// Metadata returns the resource type name.
func (r *credentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancer_credential"
}

// Configure adds the provider configured client to the resource.
func (r *credentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *loadbalancer.APIClient
	var err error
	if providerData.LoadBalancerCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "loadbalancer_custom_endpoint", providerData.LoadBalancerCustomEndpoint)
		apiClient, err = loadbalancer.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.LoadBalancerCustomEndpoint),
		)
	} else {
		apiClient, err = loadbalancer.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Load Balancer client configured")
}

// Schema defines the schema for the resource.
func (r *credentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":            "Load balancer credential resource schema. Must have a `region` specified in the provider configuration.",
		"id":              "Terraform's internal resource ID. It is structured as \"`project_id`\",\"`credentials_ref`\".",
		"credentials_ref": "The credentials reference can be used for observability of the Load Balancer.",
		"project_id":      "STACKIT project ID to which the load balancer credential is associated.",
		"display_name":    "Credential name.",
		"username":        "The username used for the ARGUS instance.",
		"password":        "The password used for the ARGUS instance.",
		"deprecation_message": "The `stackit_loadbalancer_credential` resource has been deprecated and will be removed after November 13th 2024. " +
			"Please use `stackit_loadbalancer_observability_credential` instead, which offers the exact same functionality.",
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("%s\n%s", descriptions["main"], descriptions["deprecation_message"]),
		// Callout block: https://developer.hashicorp.com/terraform/registry/providers/docs#callouts
		MarkdownDescription: fmt.Sprintf("%s\n\n!> %s", descriptions["main"], descriptions["deprecation_message"]),
		DeprecationMessage:  descriptions["deprecation_message"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"credentials_ref": schema.StringAttribute{
				Description: descriptions["credentials_ref"],
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
				},
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: descriptions["username"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Description: descriptions["password"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *credentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new credentials
	createResp, err := r.client.CreateCredentials(ctx, projectId).CreateCredentialsPayload(*payload).XRequestID(uuid.NewString()).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "credentials_ref", createResp.Credential.CredentialsRef)

	// Map response body to schema
	err = mapFields(createResp.Credential, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Load balancer credential created")
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsRef := model.CredentialsRef.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_ref", credentialsRef)

	// Get credentials
	credResp, err := r.client.GetCredentials(ctx, projectId, credentialsRef).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(credResp.Credential, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Load balancer credential read")
}

func (r *credentialResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating credential", "Credential can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *credentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsRef := model.CredentialsRef.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_ref", credentialsRef)

	// Delete credentials
	_, err := r.client.DeleteCredentials(ctx, projectId, credentialsRef).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting load balancer", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Load balancer credential deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,name
func (r *credentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing credential",
			fmt.Sprintf("Expected import identifier with format: [project_id],[credentials_ref]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credentials_ref"), idParts[1])...)
	tflog.Info(ctx, "Load balancer credential state imported")
}

func toCreatePayload(model *Model) (*loadbalancer.CreateCredentialsPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &loadbalancer.CreateCredentialsPayload{
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
		Username:    conversion.StringValueToPointer(model.Username),
		Password:    conversion.StringValueToPointer(model.Password),
	}, nil
}

func mapFields(cred *loadbalancer.CredentialsResponse, m *Model) error {
	if cred == nil {
		return fmt.Errorf("response input is nil")
	}
	if m == nil {
		return fmt.Errorf("model input is nil")
	}

	var credentialsRef string
	if m.CredentialsRef.ValueString() != "" {
		credentialsRef = m.CredentialsRef.ValueString()
	} else if cred.CredentialsRef != nil {
		credentialsRef = *cred.CredentialsRef
	} else {
		return fmt.Errorf("credentials ref not present")
	}
	m.CredentialsRef = types.StringValue(credentialsRef)
	m.DisplayName = types.StringPointerValue(cred.DisplayName)
	var username string
	if m.Username.ValueString() != "" {
		username = m.Username.ValueString()
	} else if cred.Username != nil {
		username = *cred.Username
	} else {
		return fmt.Errorf("username not present")
	}
	m.Username = types.StringValue(username)

	idParts := []string{
		m.ProjectId.ValueString(),
		m.CredentialsRef.ValueString(),
	}
	m.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	return nil
}
