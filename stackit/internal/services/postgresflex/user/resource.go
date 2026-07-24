package postgresflex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"

	postgresflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresflex/utils"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
	_ resource.ResourceWithModifyPlan  = &userResource{}
)

type Model struct {
	Id         types.String `tfsdk:"id"` // needed by TF
	UserId     types.String `tfsdk:"user_id"`
	InstanceId types.String `tfsdk:"instance_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	Username   types.String `tfsdk:"username"`
	Roles      types.Set    `tfsdk:"roles"`
	Password   types.String `tfsdk:"password"`
	// Deprecated: Host is deprecated and will be removed after February 2027
	Host types.String `tfsdk:"host"`
	// Deprecated: Port is deprecated and will be removed after February 2027
	Port types.Int32 `tfsdk:"port"`
	// Deprecated: Uri is deprecated and will be removed after February 2027
	Uri    types.String `tfsdk:"uri"`
	Region types.String `tfsdk:"region"`
	// RotateWhenChanged is a map of arbitrary key/value pairs that will force
	// recreation of the resource when they change, enabling resource rotation based on
	// external conditions such as a rotating timestamp. Changing this forces a new
	// resource to be created.
	RotateWhenChanged types.Map `tfsdk:"rotate_when_changed"`
}

// NewUserResource is a helper function to simplify the provider implementation.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// userResource is the resource implementation.
type userResource struct {
	client       *postgresflex.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *userResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Metadata returns the resource type name.
func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresflex_user"
}

// Configure adds the provider configured client to the resource.
func (r *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema defines the schema for the resource.
func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "Postgres Flex user resource schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`,`user_id`\".",
		"user_id":     "User ID.",
		"instance_id": "ID of the PostgresFlex instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"roles":       "Database access levels for the user.",
		"region":      "The resource region. If not defined, the provider region is used.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: descriptions["user_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"username": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{},
			},
			"roles": schema.SetAttribute{
				Description: descriptions["roles"],
				ElementType: types.StringType,
				Required:    true,
			},
			"password": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"host": schema.StringAttribute{
				DeprecationMessage: "host is deprecated and will be removed after February 2027. The host can be retrieved from the instance in `connection_info.write.host`.",
				Computed:           true,
			},
			"port": schema.Int32Attribute{
				DeprecationMessage: "port is deprecated and will be removed after February 2027. The port can be retrieved from the instance in `connection_info.write.port`.",
				Computed:           true,
			},
			"uri": schema.StringAttribute{
				DeprecationMessage: "uri is deprecated and will be removed after February 2027.",
				Computed:           true,
				Sensitive:          true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rotate_when_changed": schema.MapAttribute{
				Description: "A map of arbitrary key/value pairs that will force " +
					"recreation of the resource when they change, enabling resource rotation " +
					"based on external conditions such as a rotating timestamp. Changing " +
					"this forces a new resource to be created.",
				Optional:    true,
				Required:    false,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	var roles []string
	if !(model.Roles.IsNull() || model.Roles.IsUnknown()) {
		diags = model.Roles.ElementsAs(ctx, &roles, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, roles)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new user
	// Workaround: The user creation will be tried 5 times. In some cases the instance might be
	// in maintenance mode and the user API is temporary unavailable. Usually this is only for 1-2 seconds.
	config := utils.RetryConfig{
		Attempts: 5,
		Backoff: func(attempt int) time.Duration {
			// Wait for every attempt 5 seconds longer. 5s, 10s, 15s and so on
			return time.Duration(attempt*5) * time.Second
		},
		RetryStatusCodes: []int{
			http.StatusLocked,
		},
	}
	userResp, err := utils.RetryRequest(ctx, r.client.DefaultAPI.CreateUser(ctx, projectId, region, instanceId).CreateUserPayload(*payload).Execute, config)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if userResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", "API didn't return response. A user might have been created")
		return
	}

	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceId,
		"user_id":     strconv.FormatInt(userResp.Id, 10),
	})
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := wait.CreateUserWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId, userResp.Id).WaitWithContext(ctx)
	if err != nil {
		return
	}

	// Deprecated: Legacy mode needed during deprecation period to retrieve all v2 values. Can be removed after February 2027
	instanceResp, err := r.client.DefaultAPI.GetInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling get instance API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFieldsCreate(userResp, getResp, instanceResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex user created")
}

// Read refreshes the Terraform state with the latest data.
func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userIdStr := model.UserId.ValueString()
	if userIdStr == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}
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
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Deprecated: Legacy mode needed during deprecation period to retrieve all v2 values. Can be removed after February 2027
	instanceResp, err := r.client.DefaultAPI.GetInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling get instance API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(recordSetResp, instanceResp, &model, region)
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

// Update updates the resource and sets the updated Terraform state on success.
func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userIdStr := model.UserId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userIdStr)
	ctx = tflog.SetField(ctx, "region", region)

	// In v2 the ID was a string. This was changed in the v3 API.
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Parsing user ID: %v", err))
		return
	}

	// Retrieve values from state
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var roles []string
	if !(model.Roles.IsNull() || model.Roles.IsUnknown()) {
		diags = model.Roles.ElementsAs(ctx, &roles, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model, roles)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Updating API payload: %v", err))
		return
	}

	// Update existing instance
	// Workaround: The user update will be tried 5 times. In some cases the instance might be
	// in maintenance mode and the user API is temporary unavailable. Usually this is only for 1-2 seconds.
	config := utils.RetryConfig{
		Attempts: 5,
		Backoff: func(attempt int) time.Duration {
			// Wait for every attempt 5 seconds longer. 5s, 10s, 15s and so on
			return time.Duration(attempt*5) * time.Second
		},
		RetryStatusCodes: []int{
			http.StatusLocked,
		},
	}
	err = utils.RetryRequestWithoutResponse(ctx, r.client.DefaultAPI.PartialUpdateUser(ctx, projectId, region, instanceId, userId).PartialUpdateUserPayload(*payload).Execute, config)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	userResp, err := r.client.DefaultAPI.GetUser(ctx, projectId, region, instanceId, userId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Deprecated: Legacy mode needed during deprecation period to retrieve all v2 values. Can be removed after February 2027
	instanceResp, err := r.client.DefaultAPI.GetInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling get instance API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(userResp, instanceResp, &stateModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex user updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userIdStr := model.UserId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userIdStr)
	ctx = tflog.SetField(ctx, "region", region)

	// In v2 the ID was a string. This was changed in the v3 API.
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting user", fmt.Sprintf("Parsing user ID: %v", err))
		return
	}

	// Delete existing record set
	// Workaround: The user delete will be tried 5 times. In some cases the instance might be
	// in maintenance mode and the user API is temporary unavailable. Usually this is only for 1-2 seconds.
	config := utils.RetryConfig{
		Attempts: 5,
		Backoff: func(attempt int) time.Duration {
			// Wait for every attempt 5 seconds longer. 5s, 10s, 15s and so on
			return time.Duration(attempt*5) * time.Second
		},
		RetryStatusCodes: []int{
			http.StatusLocked,
		},
	}
	err = utils.RetryRequestWithoutResponse(ctx, r.client.DefaultAPI.DeleteUser(ctx, projectId, region, instanceId, userId).Execute, config)
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteUserWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId, userId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting user", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "Postgres Flex user deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,zone_id,record_set_id
func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing user",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[instance_id],[user_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), idParts[3])...)
	core.LogAndAddWarning(ctx, &resp.Diagnostics,
		"Postgresflex user imported with empty password and empty uri",
		"The user password and uri are not imported as they are only available upon creation of a new user. The password and uri fields will be empty.",
	)
	tflog.Info(ctx, "Postgresflex user state imported")
}

func mapFieldsCreate(userResp *postgresflex.CreateUserResponse, getUserResp *postgresflex.GetUserResponse, instanceResp *postgresflex.GetInstanceResponse, model *Model, region string) error {
	if userResp == nil {
		return fmt.Errorf("create response is nil")
	}
	if getUserResp == nil {
		return fmt.Errorf("get response is nil")
	}
	if instanceResp == nil {
		return fmt.Errorf("instance response is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	userId := strconv.FormatInt(userResp.Id, 10)
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), region, model.InstanceId.ValueString(), userId,
	)
	model.UserId = types.StringValue(userId)
	model.Username = types.StringValue(userResp.Name)
	model.Password = types.StringValue(userResp.Password)

	if getUserResp.Roles == nil {
		model.Roles = types.SetNull(types.StringType)
	} else {
		roles := []attr.Value{}
		for _, role := range getUserResp.Roles {
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
	model.Uri = types.StringValue(fmt.Sprintf("postgresql://%s:%s@%s:%d/stackit", userResp.Name, userResp.Password, instanceResp.ConnectionInfo.Write.Host, instanceResp.ConnectionInfo.Write.Port))
	model.Region = types.StringValue(region)
	return nil
}

func mapFields(userResp *postgresflex.GetUserResponse, instanceResp *postgresflex.GetInstanceResponse, model *Model, region string) error {
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

func toCreatePayload(model *Model, roles []string) (*postgresflex.CreateUserPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if roles == nil {
		return nil, fmt.Errorf("nil roles")
	}

	return &postgresflex.CreateUserPayload{
		Roles: roles,
		Name:  model.Username.ValueString(),
	}, nil
}

func toUpdatePayload(model *Model, roles []string) (*postgresflex.PartialUpdateUserPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if roles == nil {
		return nil, fmt.Errorf("nil roles")
	}

	return &postgresflex.PartialUpdateUserPayload{
		Name:  model.Username.ValueStringPointer(),
		Roles: roles,
	}, nil
}
