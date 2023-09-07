package postgresql

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresql"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &credentialsDataSource{}
)

// NewCredentialsDataSource is a helper function to simplify the provider implementation.
func NewCredentialsDataSource() datasource.DataSource {
	return &credentialsDataSource{}
}

// credentialsDataSource is the data source implementation.
type credentialsDataSource struct {
	client *postgresql.APIClient
}

// Metadata returns the resource type name.
func (r *credentialsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_credentials"
}

// Configure adds the provider configured client to the resource.
func (r *credentialsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected stackit.ProviderData, got %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}

	var apiClient *postgresql.APIClient
	var err error
	if providerData.PostgreSQLCustomEndpoint != "" {
		apiClient, err = postgresql.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.PostgreSQLCustomEndpoint),
		)
	} else {
		apiClient, err = postgresql.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		resp.Diagnostics.AddError("Could not Configure API Client", err.Error())
		return
	}

	tflog.Info(ctx, "Postgresql zone client configured")
	r.client = apiClient
}

// Schema defines the schema for the resource.
func (r *credentialsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":           "PostgreSQL credentials data source schema.",
		"id":             "Terraform's internal resource identifier.",
		"credentials_id": "The credentials ID.",
		"instance_id":    "ID of the PostgreSQL instance.",
		"project_id":     "STACKIT project ID to which the instance is associated.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"credentials_id": schema.StringAttribute{
				Description: descriptions["credentials_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"host": schema.StringAttribute{
				Computed: true,
			},
			"hosts": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"http_api_uri": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"password": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"port": schema.Int64Attribute{
				Computed: true,
			},
			"uri": schema.StringAttribute{
				Computed: true,
			},
			"username": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	credentialsId := model.CredentialsId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "credentials_id", credentialsId)

	recordSetResp, err := r.client.GetCredentials(ctx, projectId, instanceId, credentialsId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credentials", err.Error())
		return
	}

	// Map response body to schema and populate Computed attribute values
	err = mapFields(recordSetResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error mapping fields", err.Error())
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Postgresql credentials read")
}
