package objectstorage

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &credentialDataSource{}
)

// NewCredentialDataSource is a helper function to simplify the provider implementation.
func NewCredentialDataSource() datasource.DataSource {
	return &credentialDataSource{}
}

// credentialDataSource is the resource implementation.
type credentialDataSource struct {
	client *objectstorage.APIClient
}

// Metadata returns the resource type name.
func (r *credentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_credential"
}

// Configure adds the provider configured client to the datasource.
func (r *credentialDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *objectstorage.APIClient
	var err error
	if providerData.PostgreSQLCustomEndpoint != "" {
		apiClient, err = objectstorage.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.PostgreSQLCustomEndpoint),
		)
	} else {
		apiClient, err = objectstorage.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "ObjectStorage credential client configured")
}

// Schema defines the schema for the datasource.
func (r *credentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                 "ObjectStorage credential data source schema. Must have a `region` specified in the provider configuration.",
		"id":                   "Terraform's internal resource identifier. It is structured as \"`project_id`,`credentials_group_id`,`credential_id`\".",
		"credential_id":        "The credential ID.",
		"credentials_group_id": "The credential group ID.",
		"project_id":           "STACKIT Project ID to which the credential group is associated.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"credential_id": schema.StringAttribute{
				Description: descriptions["credential_id"],
				Required:    true,
			},
			"credentials_group_id": schema.StringAttribute{
				Description: descriptions["credentials_group_id"],
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"access_key": schema.StringAttribute{
				Computed: true,
			},
			"secret_access_key": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"expiration_timestamp": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	credentialId := model.CredentialId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_group_id", credentialsGroupId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)

	found, err := readCredentials(ctx, &model, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Finding credential: %v", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", "Credential not found")
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage credential read")
}
