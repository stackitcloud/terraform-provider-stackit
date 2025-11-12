package postgresflex

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	postgresflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresflex/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex/wait"
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
	Host       types.String `tfsdk:"host"`
	Port       types.Int64  `tfsdk:"port"`
	Uri        types.String `tfsdk:"uri"`
	Region     types.String `tfsdk:"region"`
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
	rolesOptions := []string{"login", "createdb"}

	descriptions := map[string]string{
		"main":        "Postgres Flex user resource schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`,`user_id`\".",
		"user_id":     "User ID.",
		"instance_id": "ID of the PostgresFlex instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"roles":       "Database access levels for the user. " + utils.FormatPossibleValues(rolesOptions...),
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
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"roles": schema.SetAttribute{
				Description: descriptions["roles"],
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf("login", "createdb"),
					),
				},
			},
			"password": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"host": schema.StringAttribute{
				Computed: true,
			},
			"port": schema.Int64Attribute{
				Computed: true,
			},
			"uri": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
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
	userResp, err := r.client.CreateUser(ctx, projectId, region, instanceId).CreateUserPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if userResp == nil || userResp.Item == nil || userResp.Item.Id == nil || *userResp.Item.Id == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", "API didn't return user Id. A user might have been created")
		return
	}
	userId := *userResp.Item.Id
	ctx = tflog.SetField(ctx, "user_id", userId)

	// Map response body to schema
	err = mapFieldsCreate(userResp, &model, region)
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
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userId := model.UserId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)
	ctx = tflog.SetField(ctx, "region", region)

	recordSetResp, err := r.client.GetUser(ctx, projectId, region, instanceId, userId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(recordSetResp, &model, region)
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
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userId := model.UserId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)
	ctx = tflog.SetField(ctx, "region", region)

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
	err = r.client.UpdateUser(ctx, projectId, region, instanceId, userId).UpdateUserPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", err.Error())
		return
	}

	userResp, err := r.client.GetUser(ctx, projectId, region, instanceId, userId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(userResp, &stateModel, region)
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

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userId := model.UserId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing record set
	err := r.client.DeleteUser(ctx, projectId, region, instanceId, userId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting user", fmt.Sprintf("Calling API: %v", err))
	}
	_, err = wait.DeleteUserWaitHandler(ctx, r.client, projectId, region, instanceId, userId).WaitWithContext(ctx)
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

func mapFieldsCreate(userResp *postgresflex.CreateUserResponse, model *Model, region string) error {
	if userResp == nil || userResp.Item == nil {
		return fmt.Errorf("response is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	user := userResp.Item

	if user.Id == nil {
		return fmt.Errorf("user id not present")
	}
	userId := *user.Id
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), region, model.InstanceId.ValueString(), userId,
	)
	model.UserId = types.StringValue(userId)
	model.Username = types.StringPointerValue(user.Username)

	if user.Password == nil {
		return fmt.Errorf("user password not present")
	}
	model.Password = types.StringValue(*user.Password)

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
	model.Uri = types.StringPointerValue(user.Uri)
	model.Region = types.StringValue(region)
	return nil
}

func mapFields(userResp *postgresflex.GetUserResponse, model *Model, region string) error {
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
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), region, model.InstanceId.ValueString(), userId,
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
		Roles:    &roles,
		Username: conversion.StringValueToPointer(model.Username),
	}, nil
}

func toUpdatePayload(model *Model, roles []string) (*postgresflex.UpdateUserPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if roles == nil {
		return nil, fmt.Errorf("nil roles")
	}

	return &postgresflex.UpdateUserPayload{
		Roles: &roles,
	}, nil
}
