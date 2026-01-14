package customrole

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	authorizationUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/authorization/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// List of resource types which can have custom roles.
var resourceTypes = []string{
	"project",
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &customRoleResource{}
	_ resource.ResourceWithConfigure   = &customRoleResource{}
	_ resource.ResourceWithImportState = &customRoleResource{}
)

// Model represents the schema for the git resource.
type Model struct {
	Id          types.String `tfsdk:"id"` // Required by Terraform
	RoleId      types.String `tfsdk:"role_id"`
	ResourceId  types.String `tfsdk:"resource_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.List   `tfsdk:"permissions"`
}

// customRoleResource is the resource implementation.
type customRoleResource struct {
	resourceType string
	client       *authorization.APIClient
}

// NewProjectRoleAssignmentResources is a helper function generate custom role
// resources for all possible resource types.
func NewCustomRoleResources() []func() resource.Resource {
	resources := make([]func() resource.Resource, 0)
	for _, v := range resourceTypes {
		resources = append(resources, func() resource.Resource {
			return &customRoleResource{
				resourceType: v,
			}
		})
	}

	return resources
}

// descriptions for the attributes in the Schema.
var descriptions = map[string]string{
	"main":        "Custom Role resource schema.",
	"id":          "Terraform's internal resource identifier. It is structured as \"[resource_id],[role_id]\".",
	"role_id":     "The ID of the role.",
	"resource_id": "Resource to add the custom role to.",
	"name":        "Name of the role",
	"description": "A human readable description of the role.",
	"permissions": "Permissions for the role",
	"etag":        "A version identifier for the custom role.",
}

// Configure adds the provider configured client to the resource.
func (r *customRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := authorizationUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	r.client = apiClient

	tflog.Info(ctx, "authorization client configured")
}

// Metadata sets the resource type name.
func (r *customRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_authorization_%s_custom_role", req.ProviderTypeName, r.resourceType)
}

// Schema defines the schema for the resource.
func (r *customRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"role_id": schema.StringAttribute{
				Description: descriptions["role_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: descriptions["resource_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Required:    true,
			},
			"permissions": schema.ListAttribute{
				Description: descriptions["permissions"],
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *customRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	ctx = r.annotateLogger(ctx, &model)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating custom role", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.AddRole(ctx, r.resourceType, model.ResourceId.ValueString()).AddRolePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating custom role", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if err = mapAddCustomRoleResponse(ctx, createResp, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating custom role", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "custom role created")
}

// Read refreshes the Terraform state with the latest data.
func (r *customRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	ctx = r.annotateLogger(ctx, &model)

	roleResp, err := r.client.GetRoleExecute(ctx, r.resourceType, model.ResourceId.ValueString(), model.RoleId.ValueString())
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError

		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading custom role", fmt.Sprintf("Calling API: %v", err))

		return
	}

	ctx = core.LogResponse(ctx)

	if err = mapGetCustomRoleResponse(ctx, roleResp, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading custom role", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set the updated state.
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, fmt.Sprintf("read custom role %s", model.RoleId))
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *customRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { //nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	ctx = r.annotateLogger(ctx, &model)

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating custom role", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Update existing custom role
	roleResp, err := r.client.UpdateRole(ctx, r.resourceType, model.ResourceId.ValueString(), model.RoleId.ValueString()).UpdateRolePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating custom role", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapUpdateCustomRoleResponse(ctx, roleResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating custom role", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "custom role updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *customRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	ctx = r.annotateLogger(ctx, &model)

	_, err := r.client.DeleteRoleExecute(ctx, r.resourceType, model.ResourceId.ValueString(), model.RoleId.ValueString())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting custom role", fmt.Sprintf("Calling API: %v", err))
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "custom role deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the custom role resource import identifier is:
// resource_id,role_id.
func (r *customRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing custom role",
			fmt.Sprintf("Expected import identifier with format [resource_id],[role_id] got %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), idParts[1])...)
	tflog.Info(ctx, "custom role state imported")
}

// mapGetCustomRoleResponse maps custom role response fields to the provider's internal model.
func mapGetCustomRoleResponse(ctx context.Context, resp *authorization.GetRoleResponse, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}

	if resp.Role == nil {
		return fmt.Errorf("response role is nil")
	}

	if resp.Role.Id == nil {
		return fmt.Errorf("response role id is nil")
	}

	if resp.Role.Permissions == nil {
		return fmt.Errorf("response role permissions is nil")
	}

	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	modelPermissions, err := utils.ListValuetoStringSlice(model.Permissions)
	if err != nil {
		return err
	}

	model.Id = utils.BuildInternalTerraformId(*resp.ResourceId, *resp.Role.Id)
	model.ResourceId = types.StringPointerValue(resp.ResourceId)
	model.RoleId = types.StringPointerValue(resp.Role.Id)
	model.Name = types.StringPointerValue(resp.Role.Name)
	model.Description = types.StringPointerValue(resp.Role.Description)

	if len(*resp.Role.Permissions) == 0 {
		model.Permissions = types.ListNull(types.StringType)
		return nil
	}

	var respPermissions []string
	for _, p := range *resp.Role.Permissions {
		if name, ok := p.GetNameOk(); ok {
			respPermissions = append(respPermissions, name)
		}
	}

	reconciledPermissions := utils.ReconcileStringSlices(modelPermissions, respPermissions)

	var diags diag.Diagnostics
	model.Permissions, diags = types.ListValueFrom(ctx, types.StringType, reconciledPermissions)
	if diags.HasError() {
		return fmt.Errorf("mapping permissions: %w", core.DiagsToError(diags))
	}

	return nil
}

func mapAddCustomRoleResponse(ctx context.Context, resp *authorization.AddCustomRoleResponse, model *Model) error {
	getRoleResponse, err := authorizationUtils.TypeConverter[authorization.GetRoleResponse](resp)
	if err != nil {
		return err
	}

	return mapGetCustomRoleResponse(ctx, getRoleResponse, model)
}

func mapUpdateCustomRoleResponse(ctx context.Context, resp *authorization.UpdateRoleResponse, model *Model) error {
	getRoleResponse, err := authorizationUtils.TypeConverter[authorization.GetRoleResponse](resp)
	if err != nil {
		return err
	}

	return mapGetCustomRoleResponse(ctx, getRoleResponse, model)
}

// toCreatePayload builds an addRolePayload from provider's model.
func toCreatePayload(ctx context.Context, model *Model) (*authorization.AddRolePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	permissions := make([]authorization.PermissionRequest, 0)
	for _, permission := range model.Permissions.Elements() {
		if utils.IsUndefined(permission) {
			return nil, errors.New("permission is unknown or null")
		}

		permission, err := conversion.ToString(ctx, permission)
		if err != nil {
			return nil, fmt.Errorf("converting permission list entry to string: %w", err)
		}

		permissions = append(permissions, authorization.PermissionRequest{Name: &permission})
	}

	return &authorization.AddRolePayload{
		Name:        model.Name.ValueStringPointer(),
		Description: model.Description.ValueStringPointer(),
		Permissions: &permissions,
	}, nil
}

// toUpdatePayload builds an updateRolePayload from provider's model.
func toUpdatePayload(ctx context.Context, model *Model) (*authorization.UpdateRolePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	permissions := make([]authorization.PermissionRequest, 0)
	for _, permission := range model.Permissions.Elements() {
		if utils.IsUndefined(permission) {
			return nil, errors.New("permission is unknown or null")
		}

		permission, err := conversion.ToString(ctx, permission)
		if err != nil {
			return nil, fmt.Errorf("converting permission list entry to string: %w", err)
		}

		permissions = append(permissions, authorization.PermissionRequest{Name: &permission})
	}

	return &authorization.UpdateRolePayload{
		Name:        model.Name.ValueStringPointer(),
		Description: model.Description.ValueStringPointer(),
		Permissions: &permissions,
	}, nil
}

func (r *customRoleResource) annotateLogger(ctx context.Context, model *Model) context.Context {
	ctx = tflog.SetField(ctx, "resource_id", model.ResourceId.ValueString())
	ctx = tflog.SetField(ctx, "role_id", model.RoleId.ValueString())
	ctx = tflog.SetField(ctx, "name", model.Name.ValueString())

	return ctx
}
