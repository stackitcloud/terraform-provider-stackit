package objectstorage

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &credentialsGroupDataSource{}
)

// NewCredentialsGroupDataSource is a helper function to simplify the provider implementation.
func NewCredentialsGroupDataSource() datasource.DataSource {
	return &credentialsGroupDataSource{}
}

// credentialsGroupDataSource is the data source implementation.
type credentialsGroupDataSource struct {
	client *objectstorage.APIClient
}

// Metadata returns the data source type name.
func (r *credentialsGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_credentials_group"
}

// Configure adds the provider configured client to the data source.
func (r *credentialsGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	if providerData.ObjectStorageCustomEndpoint != "" {
		apiClient, err = objectstorage.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ObjectStorageCustomEndpoint),
		)
	} else {
		apiClient, err = objectstorage.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "ObjectStorage credentials group client configured")
}

// Schema defines the schema for the data source.
func (r *credentialsGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                 "ObjectStorage credentials group data source schema.",
		"id":                   "Terraform's internal data source identifier. It is structured as \"`project_id`,`credentials_group_id`\".",
		"credentials_group_id": "The credentials group ID",
		"name":                 "The credentials group's display name.",
		"project_id":           "Object Storage Project ID to which the credentials group is associated.",
		"urn":                  "Credentials group uniform resource name (URN)",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"credentials_group_id": schema.StringAttribute{
				Description: descriptions["id"],
				Optional:    true,
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
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Optional:    true,
				Computed:    true,
			},
			"urn": schema.StringAttribute{
				Computed:    true,
				Description: descriptions["urn"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialsGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_group_id", credentialsGroupId)

	err := readCredentialsGroups(ctx, &model, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credentialsGroup", fmt.Sprintf("getting credential group from list of credentials groups: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage credentials group read")
}
