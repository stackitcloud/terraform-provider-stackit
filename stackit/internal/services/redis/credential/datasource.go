package redis

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/redis"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &credentialDataSource{}
)

// NewCredentialDataSource is a helper function to simplify the provider implementation.
func NewCredentialDataSource() datasource.DataSource {
	return &credentialDataSource{}
}

// credentialDataSource is the data source implementation.
type credentialDataSource struct {
	client *redis.APIClient
}

// Metadata returns the data source type name.
func (r *credentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_credential"
}

// Configure adds the provider configured client to the data source.
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

	var apiClient *redis.APIClient
	var err error
	if providerData.RedisCustomEndpoint != "" {
		apiClient, err = redis.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.RedisCustomEndpoint),
		)
	} else {
		apiClient, err = redis.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Redis credential client configured")
}

// Schema defines the schema for the data source.
func (r *credentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":          "Redis credential data source schema. Must have a `region` specified in the provider configuration.",
		"id":            "Terraform's internal data source. identifier. It is structured as \"`project_id`,`instance_id`,`credential_id`\".",
		"credential_id": "The credential's ID.",
		"instance_id":   "ID of the Redis instance.",
		"project_id":    "STACKIT project ID to which the instance is associated.",
		"uri":           "Connection URI. Currently the returned uri doesn't support SSL, but you can enable it by using `rediss://` instead of `redis://`. We will provide the SSL connection in a future release",
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
			"load_balanced_host": schema.StringAttribute{
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
				Description: descriptions["uri"],
				Computed:    true,
			},
			"username": schema.StringAttribute{
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
	instanceId := model.InstanceId.ValueString()
	credentialId := model.CredentialId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)

	recordSetResp, err := r.client.GetCredentials(ctx, projectId, instanceId, credentialId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, recordSetResp, &model)
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
	tflog.Info(ctx, "Redis credential read")
}
