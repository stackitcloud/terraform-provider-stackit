package roleassignments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	authorizationUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/authorization/utils"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// List of permission assignments targets in form [TF resource name]:[api name]
var roleTargets = []string{
	"project",
	"organization",
}

// This resource is part of the "iam" experiment
var experiment = "iam"

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &roleAssignmentResource{}
	_ resource.ResourceWithConfigure   = &roleAssignmentResource{}
	_ resource.ResourceWithImportState = &roleAssignmentResource{}

	errRoleAssignmentNotFound       = errors.New("response members did not contain expected role assignment")
	errRoleAssignmentDuplicateFound = errors.New("found a duplicate role assignment.")
)

// Provider's internal model
type Model struct {
	Id         types.String `tfsdk:"id"` // needed by TF
	ResourceId types.String `tfsdk:"resource_id"`
	Role       types.String `tfsdk:"role"`
	Subject    types.String `tfsdk:"subject"`
}

// NewProjectRoleAssignmentResource is a helper function to simplify the provider implementation.
func NewRoleAssignmentResources() []func() resource.Resource {
	resources := make([]func() resource.Resource, 0)
	for _, v := range roleTargets {
		resources = append(resources, func() resource.Resource {
			return &roleAssignmentResource{
				apiName: v,
			}
		})
	}
	return resources
}

// roleAssignmentResource is the resource implementation.
type roleAssignmentResource struct {
	authorizationClient *authorization.APIClient
	apiName             string
}

// Metadata returns the resource type name.
func (r *roleAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_authorization_%s_role_assignment", req.ProviderTypeName, r.apiName)
}

// Configure adds the provider configured client to the resource.
func (r *roleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &providerData, experiment, fmt.Sprintf("stackit_authorization_%s_role_assignment", r.apiName), core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := authorizationUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.authorizationClient = apiClient
	tflog.Info(ctx, fmt.Sprintf("Resource Manager %s Role Assignment client configured", r.apiName))
}

// Schema defines the schema for the resource.
func (r *roleAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        features.AddExperimentDescription(fmt.Sprintf("%s Role Assignment resource schema.", r.apiName), experiment, core.Resource),
		"id":          "Terraform's internal resource identifier. It is structured as \"[resource_id],[role],[subject]\".",
		"resource_id": fmt.Sprintf("%s Resource to assign the role to.", r.apiName),
		"role":        "Role to be assigned",
		"subject":     "Identifier of user, service account or client. Usually email address or name in case of clients",
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
			"role": schema.StringAttribute{
				Description: descriptions["role"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subject": schema.StringAttribute{
				Description: descriptions["subject"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *roleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = r.annotateLogger(ctx, &model)

	if err := r.checkDuplicate(ctx, model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error while checking for duplicate role assignments", err.Error())
		return
	}

	// Create new project role assignment
	payload, err := r.toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	createResp, err := r.authorizationClient.AddMembers(ctx, model.ResourceId.ValueString()).AddMembersPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error creating %s role assignment", r.apiName), fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapMembersResponse(createResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error creating %s role assignment", r.apiName), fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s role assignment created", r.apiName))
}

// Read refreshes the Terraform state with the latest data.
func (r *roleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = r.annotateLogger(ctx, &model)

	listResp, err := r.authorizationClient.ListMembers(ctx, r.apiName, model.ResourceId.ValueString()).Subject(model.Subject.ValueString()).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading authorizations", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapListMembersResponse(listResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading authorization", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s role assignment read successful", r.apiName))
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *roleAssignmentResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// does nothing since resource updates should always trigger resource replacement
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *roleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = r.annotateLogger(ctx, &model)

	payload := authorization.RemoveMembersPayload{
		ResourceType: &r.apiName,
		Members: &[]authorization.Member{
			*authorization.NewMember(model.Role.ValueString(), model.Subject.ValueString()),
		},
	}

	// Delete existing project role assignment
	_, err := r.authorizationClient.RemoveMembers(ctx, model.ResourceId.ValueString()).RemoveMembersPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error deleting %s role assignment", r.apiName), fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, fmt.Sprintf("%s role assignment deleted", r.apiName))
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the project role assignment resource import identifier is: resource_id,role,subject
func (r *roleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			fmt.Sprintf("Error importing %s role assignment", r.apiName),
			fmt.Sprintf("Expected import identifier with format [resource_id],[role],[subject], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subject"), idParts[2])...)
	tflog.Info(ctx, fmt.Sprintf("%s role assignment state imported", r.apiName))
}

// Maps project role assignment fields to the provider's internal model.
func mapListMembersResponse(resp *authorization.ListMembersResponse, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if resp.Members == nil {
		return fmt.Errorf("response members are nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ResourceId.ValueString(), model.Role.ValueString(), model.Subject.ValueString())
	model.ResourceId = types.StringPointerValue(resp.ResourceId)

	for _, m := range *resp.Members {
		if *m.Role == model.Role.ValueString() && *m.Subject == model.Subject.ValueString() {
			model.Role = types.StringPointerValue(m.Role)
			model.Subject = types.StringPointerValue(m.Subject)
			return nil
		}
	}
	return errRoleAssignmentNotFound
}

func mapMembersResponse(resp *authorization.MembersResponse, model *Model) error {
	listMembersResponse, err := typeConverter[authorization.ListMembersResponse](resp)
	if err != nil {
		return err
	}
	return mapListMembersResponse(listMembersResponse, model)
}

// Helper to convert objects with equal JSON tags
func typeConverter[R any](data any) (*R, error) {
	var result R
	b, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

// Build Createproject role assignmentPayload from provider's model
func (r *roleAssignmentResource) toCreatePayload(model *Model) (*authorization.AddMembersPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &authorization.AddMembersPayload{
		ResourceType: &r.apiName,
		Members: &[]authorization.Member{
			*authorization.NewMember(model.Role.ValueString(), model.Subject.ValueString()),
		},
	}, nil
}

func (r *roleAssignmentResource) annotateLogger(ctx context.Context, model *Model) context.Context {
	resourceId := model.ResourceId.ValueString()
	ctx = tflog.SetField(ctx, "resource_id", resourceId)
	ctx = tflog.SetField(ctx, "subject", model.Subject.ValueString())
	ctx = tflog.SetField(ctx, "role", model.Role.ValueString())
	ctx = tflog.SetField(ctx, "resource_type", r.apiName)
	return ctx
}

// returns an error if duplicate role assignment exists
func (r *roleAssignmentResource) checkDuplicate(ctx context.Context, model Model) error { //nolint:gocritic // A read only copy is required since an api response is parsed into the model and this check should not affect the model parameter
	listResp, err := r.authorizationClient.ListMembers(ctx, r.apiName, model.ResourceId.ValueString()).Subject(model.Subject.ValueString()).Execute()
	if err != nil {
		return err
	}

	// Map response body to schema
	err = mapListMembersResponse(listResp, &model)

	if err != nil {
		if errors.Is(err, errRoleAssignmentNotFound) {
			return nil
		}
		return err
	}
	return errRoleAssignmentDuplicateFound
}
