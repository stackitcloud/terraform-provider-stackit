package postgresflex

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	postgresflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresflex/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api"
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
	// Deprecated: Host is deprecated and will be removed after February 2027
	Host types.String `tfsdk:"host"`
	// Deprecated: Port is deprecated and will be removed after February 2027
	Port   types.Int32  `tfsdk:"port"`
	Region types.String `tfsdk:"region"`
}

// NewUserDataSource is a helper function to simplify the provider implementation.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource is the data source implementation.
type userDataSource struct {
	client       *postgresflex.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresflex_user"
}

// Configure adds the provider configured client to the data source.
func (r *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := postgresflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Postgres Flex user client configured")
}

// Schema defines the schema for the data source.
func (r *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "Postgres Flex user data source schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal data source. ID. It is structured as \"`project_id`,`region`,`instance_id`,`user_id`\".",
		"user_id":     "User ID.",
		"instance_id": "ID of the PostgresFlex instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"region":      "The resource region. If not defined, the provider region is used.",
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
				Computed: true,
			},
			"roles": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"host": schema.StringAttribute{
				DeprecationMessage: "host is deprecated and will be removed after February 2027. The host can be retrieved from the instance `connection_info.write.host`.",
				Computed:           true,
			},
			"port": schema.Int32Attribute{
				DeprecationMessage: "port is deprecated and will be removed after February 2027. The port can be retrieved from the instance `connection_info.write.port`.",
				Computed:           true,
			},
			"region": schema.StringAttribute{
				// the region cannot be found automatically, so it has to be passed
				Optional:    true,
				Description: descriptions["region"],
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

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userIdStr := model.UserId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userIdStr)
	ctx = tflog.SetField(ctx, "region", region)

	// In v2 the ID was a string. This was changed in the v3 API.
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading user", fmt.Sprintf("Parsing user ID: %v", err))
		return
	}

	recordSetResp, err := r.client.DefaultAPI.GetUser(ctx, projectId, region, instanceId, userId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading user",
			fmt.Sprintf("User with ID %q or instance with ID %q does not exist in project %q.", userIdStr, instanceId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Deprecated: Legacy mode needed during deprecation period to retrieve all v2 values. Can be removed after February 2027
	instanceResp, err := r.client.DefaultAPI.GetInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling get instance API: %v", err))
		return
	}

	// Map response body to schema and populate Computed attribute values
	err = mapDataSourceFields(recordSetResp, instanceResp, &model, region)
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
	tflog.Info(ctx, "Postgres Flex user read")
}

func mapDataSourceFields(userResp *postgresflex.GetUserResponse, instanceResp *postgresflex.GetInstanceResponse, model *DataSourceModel, region string) error {
	if userResp == nil {
		return fmt.Errorf("user response is nil")
	}
	if instanceResp == nil {
		return fmt.Errorf("instance response is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var userId string
	if model.UserId.ValueString() != "" {
		userId = model.UserId.ValueString()
	} else {
		userId = strconv.FormatInt(userResp.Id, 10)
	}
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), region, model.InstanceId.ValueString(), userId,
	)
	model.UserId = types.StringValue(userId)
	model.Username = types.StringValue(userResp.Name)

	if userResp.Roles == nil {
		model.Roles = types.SetNull(types.StringType)
	} else {
		roles := []attr.Value{}
		for _, role := range userResp.Roles {
			roles = append(roles, types.StringValue(role))
		}
		rolesSet, diags := types.SetValue(types.StringType, roles)
		if diags.HasError() {
			return fmt.Errorf("failed to map roles: %w", core.DiagsToError(diags))
		}
		model.Roles = rolesSet
	}
	model.Host = types.StringValue(instanceResp.ConnectionInfo.Write.Host)
	model.Port = types.Int32Value(instanceResp.ConnectionInfo.Write.Port)
	model.Region = types.StringValue(region)
	return nil
}
