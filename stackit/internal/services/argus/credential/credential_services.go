package argus

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

func (r *credentialResource) credentialRead(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	diags := req.State.Get(ctx, &r.model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := r.model.ProjectId.ValueString()
	instanceId := r.model.InstanceId.ValueString()
	userName := r.model.Username.ValueString()

	err := getCredentialsAndHandleErrors(ctx, instanceId, projectId, userName, r, resp)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Calling API: %v", err))
	}

	diags = resp.State.Set(ctx, r.model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Argus credential read")
}

func (r *credentialResource) credentialCreate(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if diags := req.Plan.Get(ctx, &r.model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

	var err error
	credential, err := r.client.CreateCredentials(ctx, r.model.InstanceId.ValueString(), r.model.ProjectId.ValueString()).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Calling API: %v", err))
	}

	if err = mapFields(credential.Credentials, &r.model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Processing API payload: %v", err))
	}

	if diags := resp.State.Set(ctx, &r.model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

	tflog.Info(ctx, "Argus credential created", map[string]interface{}{"id": r.model.Id.ValueString()})
}

func (r *credentialResource) credentialDelete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if diags := req.State.Get(ctx, &r.model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", "Failed to get resource state")
	}

	err := deleteCredentialsAndHandleErrors(ctx, r.model.InstanceId.ValueString(), r.model.ProjectId.ValueString(), r.model.Username.ValueString(), r, resp)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "Argus credential deleted", map[string]interface{}{"id": r.model.Id.ValueString()})
}

func (r *credentialResource) credentialUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating credential", "Credential can't be updated")
}

func (r *credentialResource) credentialSchema(resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Argus credential resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`,`username`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the credential is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "The Argus Instance ID the credential belongs to.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "Credential username",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"password": schema.StringAttribute{
				Description: "Credential password",
				Computed:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *credentialResource) configureClient(ctx context.Context, req *resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var apiClient *argus.APIClient
	var err error

	if req.ProviderData == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", "Provider data is nil")
		return
	}
	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	apiClient, err = argus.NewAPIClient(config.WithCustomAuth(providerData.RoundTripper))
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	if providerData.ArgusCustomEndpoint != "" {
		config.WithEndpoint(providerData.ArgusCustomEndpoint)
	} else {
		apiClient, err = argus.NewAPIClient(
			config.WithRegion(providerData.Region),
		)
	}

	r.client = apiClient
	tflog.Info(ctx, "Argus credential client configured")
}
