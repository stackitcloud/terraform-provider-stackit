package postgresflexalpha

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	postgresflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &userDataSource{}
)

type DataSourceModel struct {
	Id               types.String `tfsdk:"id"` // needed by TF
	UserId           types.Int64  `tfsdk:"user_id"`
	InstanceId       types.String `tfsdk:"instance_id"`
	ProjectId        types.String `tfsdk:"project_id"`
	Username         types.String `tfsdk:"username"`
	Roles            types.Set    `tfsdk:"roles"`
	Host             types.String `tfsdk:"host"`
	Port             types.Int64  `tfsdk:"port"`
	Region           types.String `tfsdk:"region"`
	Status           types.String `tfsdk:"status"`
	ConnectionString types.String `tfsdk:"connection_string"`
}

// NewUserDataSource is a helper function to simplify the provider implementation.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource is the data source implementation.
type userDataSource struct {
	client       *postgresflexalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *userDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_postgresflexalpha_user"
}

// Configure adds the provider configured client to the data source.
func (r *userDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := postgresflexalphaUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Postgres Flex user client configured")
}

// Schema defines the schema for the data source.
func (r *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":              "Postgres Flex user data source schema. Must have a `region` specified in the provider configuration.",
		"id":                "Terraform's internal data source. ID. It is structured as \"`project_id`,`region`,`instance_id`,`user_id`\".",
		"user_id":           "User ID.",
		"instance_id":       "ID of the PostgresFlex instance.",
		"project_id":        "STACKIT project ID to which the instance is associated.",
		"region":            "The resource region. If not defined, the provider region is used.",
		"status":            "The current status of the user.",
		"connection_string": "The connection string for the user to the instance.",
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
				Computed: true,
			},
			"port": schema.Int64Attribute{
				Computed: true,
			},
			"region": schema.StringAttribute{
				// the region cannot be found automatically, so it has to be passed
				Optional:    true,
				Description: descriptions["region"],
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"connection_string": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *userDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userId := model.UserId.ValueInt64()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)
	ctx = tflog.SetField(ctx, "region", region)

	recordSetResp, err := r.client.GetUserRequest(ctx, projectId, region, instanceId, userId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading user",
			fmt.Sprintf(
				"User with ID %q or instance with ID %q does not exist in project %q.",
				userId,
				instanceId,
				projectId,
			),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema and populate Computed attribute values
	err = mapDataSourceFields(recordSetResp, &model, region)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading user",
			fmt.Sprintf("Processing API payload: %v", err),
		)
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

func mapDataSourceFields(userResp *postgresflexalpha.GetUserResponse, model *DataSourceModel, region string) error {
	if userResp == nil {
		return fmt.Errorf("response is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	user := userResp

	var userId int64
	if model.UserId.ValueInt64() != 0 {
		userId = model.UserId.ValueInt64()
	} else if user.Id != nil {
		userId = *user.Id
	} else {
		return fmt.Errorf("user id not present")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), region, model.InstanceId.ValueString(), strconv.FormatInt(userId, 10),
	)
	model.UserId = types.Int64Value(userId)
	model.Username = types.StringPointerValue(user.Name)

	if user.Roles == nil {
		model.Roles = types.SetNull(types.StringType)
	} else {
		var roles []attr.Value
		for _, role := range *user.Roles {
			roles = append(roles, types.StringValue(string(role)))
		}
		rolesSet, diags := types.SetValue(types.StringType, roles)
		if diags.HasError() {
			return fmt.Errorf("failed to map roles: %w", core.DiagsToError(diags))
		}
		model.Roles = rolesSet
	}
	model.Host = types.StringPointerValue(user.Host)
	model.Port = types.Int64PointerValue(user.Port)
	model.Region = types.StringValue(region)
	model.Status = types.StringPointerValue(user.Status)
	model.ConnectionString = types.StringPointerValue(user.ConnectionString)
	return nil
}
