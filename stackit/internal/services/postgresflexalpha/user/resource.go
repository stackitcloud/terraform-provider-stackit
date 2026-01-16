package postgresflexalpha

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	postgresflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/utils"

	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
	_ resource.ResourceWithModifyPlan  = &userResource{}
)

type Model struct {
	Id               types.String `tfsdk:"id"` // needed by TF
	UserId           types.Int64  `tfsdk:"user_id"`
	InstanceId       types.String `tfsdk:"instance_id"`
	ProjectId        types.String `tfsdk:"project_id"`
	Username         types.String `tfsdk:"username"`
	Roles            types.Set    `tfsdk:"roles"`
	Password         types.String `tfsdk:"password"`
	Host             types.String `tfsdk:"host"`
	Port             types.Int64  `tfsdk:"port"`
	Region           types.String `tfsdk:"region"`
	Status           types.String `tfsdk:"status"`
	ConnectionString types.String `tfsdk:"connection_string"`
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
func (r *userResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) { // nolint:gocritic // function signature required by Terraform
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
	resp.TypeName = req.ProviderTypeName + "_postgresflexalpha_user"
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
	rolesOptions := []string{"login", "createdb", "createrole"}

	descriptions := map[string]string{
		"main":              "Postgres Flex user resource schema. Must have a `region` specified in the provider configuration.",
		"id":                "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`,`user_id`\".",
		"user_id":           "User ID.",
		"instance_id":       "ID of the PostgresFlex instance.",
		"project_id":        "STACKIT project ID to which the instance is associated.",
		"username":          "The name of the user.",
		"roles":             "Database access levels for the user. " + utils.FormatPossibleValues(rolesOptions...),
		"region":            "The resource region. If not defined, the provider region is used.",
		"status":            "The current status of the user.",
		"password":          "The password for the user. This is only set upon creation.",
		"host":              "The host of the Postgres Flex instance.",
		"port":              "The port of the Postgres Flex instance.",
		"connection_string": "The connection string for the user to the instance.",
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
			"user_id": schema.Int64Attribute{
				Description: descriptions["user_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{},
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
				Description:   descriptions["username"],
				Required:      true,
				PlanModifiers: []planmodifier.String{
					// stringplanmodifier.RequiresReplace(),
				},
			},
			"roles": schema.SetAttribute{
				Description: descriptions["roles"],
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf(rolesOptions...),
					),
				},
			},
			"password": schema.StringAttribute{
				Description: descriptions["password"],
				Computed:    true,
				Sensitive:   true,
			},
			"host": schema.StringAttribute{
				Description: descriptions["host"],
				Computed:    true,
			},
			"port": schema.Int64Attribute{
				Description: descriptions["port"],
				Computed:    true,
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
			"status": schema.StringAttribute{
				Description: descriptions["status"],
				Computed:    true,
			},
			"connection_string": schema.StringAttribute{
				Description: descriptions["connection_string"],
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *userResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	ctx = r.setTFLogFields(ctx, &model)
	arg := r.getClientArg(&model)

	var roles = r.expandRoles(ctx, model.Roles, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, &roles)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new user
	userResp, err := r.client.CreateUserRequest(
		ctx,
		arg.projectId,
		arg.region,
		arg.instanceId,
	).CreateUserRequestPayload(*payload).Execute()

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if userResp.Id == nil || *userResp.Id == 0 {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating user",
			"API didn't return user Id. A user might have been created",
		)
		return
	}
	model.UserId = types.Int64PointerValue(userResp.Id)
	model.Password = types.StringPointerValue(userResp.Password)

	ctx = tflog.SetField(ctx, "user_id", *userResp.Id)

	exists, err := r.getUserResource(ctx, &model)

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if !exists {
		core.LogAndAddError(
			ctx, &resp.Diagnostics, "Error creating user",
			fmt.Sprintf("User ID '%v' resource not found after creation", model.UserId.ValueInt64()),
		)
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
func (r *userResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	exists, err := r.getUserResource(ctx, &model)

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex user read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *userResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	ctx = r.setTFLogFields(ctx, &model)
	arg := r.getClientArg(&model)

	// Retrieve values from state
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var roles = r.expandRoles(ctx, model.Roles, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model, &roles)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Updating API payload: %v", err))
		return
	}

	// Update existing instance
	err = r.client.UpdateUserRequest(
		ctx,
		arg.projectId,
		arg.region,
		arg.instanceId,
		arg.userId,
	).UpdateUserRequestPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	exists, err := r.getUserResource(ctx, &stateModel)

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if !exists {
		core.LogAndAddError(
			ctx, &resp.Diagnostics, "Error updating user",
			fmt.Sprintf("User ID '%v' resource not found after update", stateModel.UserId.ValueInt64()),
		)
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
func (r *userResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	ctx = r.setTFLogFields(ctx, &model)
	arg := r.getClientArg(&model)

	// Delete existing record set
	err := r.client.DeleteUserRequest(ctx, arg.projectId, arg.region, arg.instanceId, arg.userId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting user", fmt.Sprintf("Calling API: %v", err))
	}

	ctx = core.LogResponse(ctx)

	exists, err := r.getUserResource(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting user", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if exists {
		core.LogAndAddError(
			ctx, &resp.Diagnostics, "Error deleting user",
			fmt.Sprintf("User ID '%v' resource still exists after deletion", model.UserId.ValueInt64()),
		)
		return
	}

	resp.State.RemoveResource(ctx)

	tflog.Info(ctx, "Postgres Flex user deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,zone_id,record_set_id
func (r *userResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(
			ctx, &resp.Diagnostics,
			"Error importing user",
			fmt.Sprintf(
				"Expected import identifier with format [project_id],[region],[instance_id],[user_id], got %q",
				req.ID,
			),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), idParts[3])...)
	core.LogAndAddWarning(
		ctx,
		&resp.Diagnostics,
		"postgresflexalpha user imported with empty password and empty uri",
		"The user password and uri are not imported as they are only available upon creation of a new user. The password and uri fields will be empty.",
	)
	tflog.Info(ctx, "postgresflexalpha user state imported")
}

func mapFields(userResp *postgresflex.GetUserResponse, model *Model, region string) error {
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

// getUserResource refreshes the resource state by calling the API and mapping the response to the model.
// Returns true if the resource state was successfully refreshed, false if the resource does not exist.
func (r *userResource) getUserResource(ctx context.Context, model *Model) (bool, error) {
	ctx = r.setTFLogFields(ctx, model)
	arg := r.getClientArg(model)

	// API Call
	userResp, err := r.client.GetUserRequest(ctx, arg.projectId, arg.region, arg.instanceId, arg.userId).Execute()

	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("error fetching user resource: %w", err)
	}

	if err := mapFields(userResp, model, arg.region); err != nil {
		return false, fmt.Errorf("error mapping user resource: %w", err)
	}

	return true, nil
}

type clientArg struct {
	projectId  string
	instanceId string
	region     string
	userId     int64
}

// getClientArg constructs client arguments from the model.
func (r *userResource) getClientArg(model *Model) *clientArg {
	return &clientArg{
		projectId:  model.ProjectId.ValueString(),
		instanceId: model.InstanceId.ValueString(),
		region:     r.providerData.GetRegionWithOverride(model.Region),
		userId:     model.UserId.ValueInt64(),
	}
}

// setTFLogFields adds relevant fields to the context for terraform logging purposes.
func (r *userResource) setTFLogFields(ctx context.Context, model *Model) context.Context {
	usrCtx := r.getClientArg(model)

	ctx = tflog.SetField(ctx, "project_id", usrCtx.projectId)
	ctx = tflog.SetField(ctx, "instance_id", usrCtx.instanceId)
	ctx = tflog.SetField(ctx, "user_id", usrCtx.userId)
	ctx = tflog.SetField(ctx, "region", usrCtx.region)

	return ctx
}

func (r *userResource) expandRoles(ctx context.Context, rolesSet types.Set, diags *diag.Diagnostics) []string {
	if rolesSet.IsNull() || rolesSet.IsUnknown() {
		return nil
	}
	var roles []string
	diags.Append(rolesSet.ElementsAs(ctx, &roles, false)...)
	return roles
}

func toCreatePayload(model *Model, roles *[]string) (*postgresflex.CreateUserRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if roles == nil {
		return nil, fmt.Errorf("nil roles")
	}

	return &postgresflex.CreateUserRequestPayload{
		Roles: toPayloadRoles(roles),
		Name:  conversion.StringValueToPointer(model.Username),
	}, nil
}

func toPayloadRoles(roles *[]string) *[]postgresflex.UserRole {
	var userRoles = make([]postgresflex.UserRole, 0, len(*roles))
	for _, role := range *roles {
		userRoles = append(userRoles, postgresflex.UserRole(role))
	}
	return &userRoles
}

func toUpdatePayload(model *Model, roles *[]string) (
	*postgresflex.UpdateUserRequestPayload,
	error,
) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if roles == nil {
		return nil, fmt.Errorf("nil roles")
	}

	return &postgresflex.UpdateUserRequestPayload{
		Name:  conversion.StringValueToPointer(model.Username),
		Roles: toPayloadRoles(roles),
	}, nil
}
