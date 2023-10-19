package secretsmanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/secretsmanager"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

type Model struct {
	Id           types.String `tfsdk:"id"` // needed by TF
	UserId       types.String `tfsdk:"user_id"`
	InstanceId   types.String `tfsdk:"instance_id"`
	ProjectId    types.String `tfsdk:"project_id"`
	Description  types.String `tfsdk:"description"`
	WriteEnabled types.Bool   `tfsdk:"write_enabled"`
	Username     types.String `tfsdk:"name"`
	Password     types.String `tfsdk:"password"`
}

// NewUserResource is a helper function to simplify the provider implementation.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// userResource is the resource implementation.
type userResource struct {
	client *secretsmanager.APIClient
}

// Metadata returns the resource type name.
func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secretsmanager_user"
}

// Configure adds the provider configured client to the resource.
func (r *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Secrets Manager user client configured")
}

// Schema defines the schema for the resource.
func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":          "Secrets Manager user resource schema. Must have a `region` specified in the provider configuration.",
		"id":            "Terraform's internal resource identifier. It is structured as \"`project_id`,`instance_id`,`user_id`\".",
		"user_id":       "The user's ID.",
		"instance_id":   "ID of the Secrets Manager instance.",
		"project_id":    "STACKIT Project ID to which the instance is associated.",
		"description":   "A user chosen description to differentiate between multiple users. Can't be changed after creation.",
		"write_enabled": "If true, the user has writeaccess to the secrets engine.",
		"username":      "An auto-generated user name.",
		"password":      "An auto-generated password.",
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
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"write_enabled": schema.BoolAttribute{
				Description: descriptions["write_enabled"],
				Required:    true,
			},
			"username": schema.StringAttribute{
				Description: descriptions["username"],
				Computed:    true,
			},
			"password": schema.StringAttribute{
				Description: descriptions["password"],
				Computed:    true,
				Sensitive:   true,
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
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new user
	userResp, err := r.client.CreateUser(ctx, projectId, instanceId).CreateUserPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if userResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", "Got empty user id")
		return
	}
	userId := *userResp.Id
	ctx = tflog.SetField(ctx, "user_id", userId)

	// Map response body to schema
	err = mapFields(userResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Secrets Manager user created")
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
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)

	userResp, err := r.client.GetUser(ctx, projectId, instanceId, userId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(userResp, &model)
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
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)

	// Generate API request body from model
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing user
	err = r.client.UpdateUser(ctx, projectId, instanceId, userId).UpdateUserPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", err.Error())
		return
	}
	user, err := r.client.GetUser(ctx, projectId, instanceId, userId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Calling API to get user's current state: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(user, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating user", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Secrets Manager user updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
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

	// Delete existing user
	err := r.client.DeleteUser(ctx, projectId, instanceId, userId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting user", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "Secrets Manager user deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,user_id
func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing credential",
			fmt.Sprintf("Expected import identifier with format [project_id],[instance_id],[user_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), idParts[2])...)
	core.LogAndAddWarning(ctx, &resp.Diagnostics,
		"Secrets Manager user imported with empty password",
		"The user password is not imported as it is only available upon creation of a new user. The password field will be empty.",
	)
	tflog.Info(ctx, "Secrets Manager user state imported")
}

func toCreatePayload(model *Model) (*secretsmanager.CreateUserPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	return &secretsmanager.CreateUserPayload{
		Description: model.Description.ValueStringPointer(),
		Write:       model.WriteEnabled.ValueBoolPointer(),
	}, nil
}

func toUpdatePayload(model *Model) (*secretsmanager.UpdateUserPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	return &secretsmanager.UpdateUserPayload{
		Write: model.WriteEnabled.ValueBoolPointer(),
	}, nil
}

func mapFields(user *secretsmanager.User, model *Model) error {
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
	// Password only sent in creation response, responses after that have it as ""
	if user.Password != nil && *user.Password != "" {
		model.Password = types.StringPointerValue(user.Password)
	}
	return nil
}
