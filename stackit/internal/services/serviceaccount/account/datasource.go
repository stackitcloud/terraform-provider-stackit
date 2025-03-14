package account

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &serviceAccountDataSource{}
)

// NewServiceAccountDataSource creates a new instance of the serviceAccountDataSource.
func NewServiceAccountDataSource() datasource.DataSource {
	return &serviceAccountDataSource{}
}

// serviceAccountDataSource is the datasource implementation for service accounts.
type serviceAccountDataSource struct {
	client *serviceaccount.APIClient
}

// Configure initializes the serviceAccountDataSource with the provided provider data.
func (r *serviceAccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured correctly.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !resourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_service_account", "datasource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

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

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

// Metadata provides metadata for the service account datasource.
func (r *serviceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

// Schema defines the schema for the service account data source.
func (r *serviceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"id":         "Terraform's internal resource ID, structured as \"project_id,email\".",
		"project_id": "STACKIT project ID to which the service account is associated.",
		"name":       "Name of the service account.",
		"email":      "Email of the service account.",
	}

	// Define the schema with validation rules and descriptions for each attribute.
	// The datasource schema differs slightly from the resource schema.
	// In this case, the email attribute is required to read the service account data from the API.
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
			},
			"email": schema.StringAttribute{
				Description: descriptions["email"],
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
		},
	}
}

// Read reads all service accounts from the API and updates the state with the latest information.
func (r *serviceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract the project ID from the model configuration
	projectId := model.ProjectId.ValueString()

	// Call the API to list service accounts in the specified project
	listSaResp, err := r.client.ListServiceAccounts(ctx, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account", fmt.Sprintf("Error calling API: %v", err))
		return
	}

	// Iterate over the service accounts returned by the API to find the one matching the email
	serviceAccounts := *listSaResp.Items
	for i := range serviceAccounts {
		// Skip if the service account email does not match
		if *serviceAccounts[i].Email != model.Email.ValueString() {
			continue
		}

		// Map the API response to the model, updating its fields with the service account data
		err = mapCreateOrListResponse(&serviceAccounts[i], &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account", fmt.Sprintf("Error processing API response: %v", err))
			return
		}

		// Try to parse the name from the provided email address
		name, err := parseNameFromEmail(model.Email.ValueString())
		if name != "" && err == nil {
			model.Name = types.StringValue(name)
		}

		// Update the state with the service account model
		diags = resp.State.Set(ctx, &model)
		resp.Diagnostics.Append(diags...)
		return
	}

	// If no matching service account is found, remove the resource from the state
	core.LogAndAddError(ctx, &resp.Diagnostics, "Service account not found", "")
	resp.State.RemoveResource(ctx)
}
