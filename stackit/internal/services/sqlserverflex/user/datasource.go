package sqlserverflex

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &userDataSource{}
)

type DataSourceModel struct {
	Id         types.String `tfsdk:"id"` // needed by TF
	UserId     types.String `tfsdk:"user_id"`
	InstanceId types.String `tfsdk:"instance_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	Username   types.String `tfsdk:"username"`
	Roles      types.Set    `tfsdk:"roles"`
	Host       types.String `tfsdk:"host"`
	Port       types.Int64  `tfsdk:"port"`
}

// NewUserDataSource is a helper function to simplify the provider implementation.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource is the data source implementation.
type userDataSource struct {
	client *sqlserverflex.APIClient
}

// Metadata returns the data source type name.
func (r *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflex_user"
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

	var apiClient *sqlserverflex.APIClient
	var err error
	if providerData.SQLServerFlexCustomEndpoint != "" {
		apiClient, err = sqlserverflex.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.SQLServerFlexCustomEndpoint),
		)
	} else {
		apiClient, err = sqlserverflex.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "SQLServer Flex user client configured")
}

// Schema defines the schema for the data source.
func (r *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "SQLServer Flex user data source schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal data source. ID. It is structured as \"`project_id`,`instance_id`,`user_id`\".",
		"user_id":     "User ID.",
		"instance_id": "ID of the SQLServer Flex instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"username":    "Username of the SQLServer Flex instance.",
		"roles":       "Database access levels for the user. Possible values: [\`##STACKIT_LoginManager##\`, \`##STACKIT_DatabaseManager##\`]",
		"password":    "Password of the user account.",
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
			"username": schema.StringAttribute{
				Description: descriptions["username"],
				Computed:    true,
			},
			"roles": schema.SetAttribute{
				Description: descriptions["roles"],
				ElementType: types.StringType,
				Computed:    true,
			},
			"host": schema.StringAttribute{
				Computed: true,
			},
			"port": schema.Int64Attribute{
				Computed: true,
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

	recordSetResp, err := r.client.GetUser(ctx, projectId, instanceId, userId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema and populate Computed attribute values
	err = mapDataSourceFields(recordSetResp, &model)
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
	tflog.Info(ctx, "SQLServer Flex user read")
}

func mapDataSourceFields(userResp *sqlserverflex.GetUserResponse, model *DataSourceModel) error {
	if userResp == nil || userResp.Item == nil {
		return fmt.Errorf("response is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	user := userResp.Item

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
	model.Username = types.StringPointerValue(user.Username)

	if user.Roles == nil {
		model.Roles = types.SetNull(types.StringType)
	} else {
		roles := []attr.Value{}
		for _, role := range *user.Roles {
			roles = append(roles, types.StringValue(role))
		}
		rolesSet, diags := types.SetValue(types.StringType, roles)
		if diags.HasError() {
			return fmt.Errorf("failed to map roles: %w", core.DiagsToError(diags))
		}
		model.Roles = rolesSet
	}
	model.Host = types.StringPointerValue(user.Host)
	model.Port = types.Int64PointerValue(user.Port)
	return nil
}
