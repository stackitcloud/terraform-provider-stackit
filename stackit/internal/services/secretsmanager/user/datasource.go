package secretsmanager

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/secretsmanager"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &userDataSource{}
)

type DataSourceModel struct {
	Id           types.String `tfsdk:"id"` // needed by TF
	UserId       types.String `tfsdk:"user_id"`
	InstanceId   types.String `tfsdk:"instance_id"`
	ProjectId    types.String `tfsdk:"project_id"`
	Description  types.String `tfsdk:"description"`
	WriteEnabled types.Bool   `tfsdk:"write_enabled"`
	Username     types.String `tfsdk:"username"`
}

// NewUserDataSource is a helper function to simplify the provider implementation.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource is the data source implementation.
type userDataSource struct {
	client *secretsmanager.APIClient
}

// Metadata returns the data source type name.
func (r *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secretsmanager_user"
}

// Configure adds the provider configured client to the data source.
func (r *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *secretsmanager.APIClient
	var err error
	if providerData.SecretsManagerCustomEndpoint != "" {
		apiClient, err = secretsmanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.SecretsManagerCustomEndpoint),
		)
	} else {
		apiClient, err = secretsmanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Secrets Manager user client configured")
}

// Schema defines the schema for the data source.
func (r *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":          "Secrets Manager user data source schema. Must have a `region` specified in the provider configuration.",
		"id":            "Terraform's internal data source identifier. It is structured as \"`project_id`,`instance_id`,`user_id`\".",
		"user_id":       "The user's ID.",
		"instance_id":   "ID of the Secrets Manager instance.",
		"project_id":    "STACKIT Project ID to which the instance is associated.",
		"description":   "A user chosen description to differentiate between multiple users. Can't be changed after creation.",
		"write_enabled": "If true, the user has writeaccess to the secrets engine.",
		"username":      "An auto-generated user name.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"user_id": schema.StringAttribute{
				Description: descriptions["user_id"],
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
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Computed:    true,
			},
			"write_enabled": schema.BoolAttribute{
				Description: descriptions["write_enabled"],
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
func (r *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userId := model.UserId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)

	userResp, err := r.client.GetUser(ctx, projectId, instanceId, userId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema and populate Computed attribute values
	err = mapDataSourceFields(userResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading user", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Secrets Manager user read")
}

func mapDataSourceFields(user *secretsmanager.User, model *DataSourceModel) error {
	if user == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var userId string
	if model.UserId.ValueString() != "" {
		userId = model.UserId.ValueString()
	} else if user.Id != nil {
		userId = *user.Id
	} else {
		return fmt.Errorf("user id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		model.InstanceId.ValueString(),
		userId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.UserId = types.StringValue(userId)
	model.Description = types.StringPointerValue(user.Description)
	model.WriteEnabled = types.BoolPointerValue(user.Write)
	model.Username = types.StringPointerValue(user.Username)
	return nil
}
