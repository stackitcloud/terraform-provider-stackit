package loadbalancer

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &credentialDataSource{}
)

type DataSourceModel struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	DisplayName    types.String `tfsdk:"display_name"`
	Username       types.String `tfsdk:"username"`
	CredentialsRef types.String `tfsdk:"credentials_ref"`
}

// NewCredentialDataSource is a helper function to simplify the provider implementation.
func NewCredentialDataSource() datasource.DataSource {
	return &credentialDataSource{}
}

// credentialDataSource is the data source implementation.
type credentialDataSource struct {
	client *loadbalancer.APIClient
}

// Metadata returns the data source type name.
func (r *credentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancer_credential"
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

	var apiClient *loadbalancer.APIClient
	var err error
	if providerData.LoadBalancerCustomEndpoint != "" {
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Load balancer client configured")
}

// Schema defines the schema for the data source.
func (r *credentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":            "Load balancer credential data source schema. Must have a `region` specified in the provider configuration.",
		"id":              "Terraform's internal resource ID. It is structured as \"`project_id`\",\"`credentials_ref`\".",
		"project_id":      "STACKIT project ID to which the load balancer credential is associated.",
		"display_name":    "Credential name.",
		"username":        "The username used for the ARGUS instance.",
		"password":        "The password used for the ARGUS instance.",
		"credentials_ref": "The credentials reference can be used for observability of the Load Balancer.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
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
				},
			},
			"credentials_ref": schema.StringAttribute{
				Description: descriptions["credentials_ref"],
				Required:    true,
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: descriptions["username"],
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsRef := model.CredentialsRef.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_ref", credentialsRef)

	credResp, err := r.client.GetCredentials(ctx, projectId, credentialsRef).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapDataSourceFields(credResp.Credential, &model)
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

func mapDataSourceFields(cred *loadbalancer.CredentialsResponse, m *DataSourceModel) error {
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
