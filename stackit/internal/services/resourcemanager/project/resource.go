package project

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

const (
	projectResourceType = "project"
	projectOwnerRole    = "owner"
)

type Model struct {
	Id                types.String `tfsdk:"id"` // needed by TF
	ProjectId         types.String `tfsdk:"project_id"`
	ContainerId       types.String `tfsdk:"container_id"`
	ContainerParentId types.String `tfsdk:"parent_container_id"`
	Name              types.String `tfsdk:"name"`
	Labels            types.Map    `tfsdk:"labels"`
	OwnerEmail        types.String `tfsdk:"owner_email"`
	Members           types.List   `tfsdk:"members"`
}

// Struct corresponding to Model.Members[i]
type member struct {
	Role    types.String `tfsdk:"role"`
	Subject types.String `tfsdk:"subject"`
}

// Types corresponding to member
var memberTypes = map[string]attr.Type{
	"role":    types.StringType,
	"subject": types.StringType,
}

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	resourceManagerClient *resourcemanager.APIClient
	authorizationClient   *authorization.APIClient
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resourcemanager_project"
}

// Configure adds the provider configured client to the resource.
func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var rmClient *resourcemanager.APIClient
	var err error
	if providerData.ResourceManagerCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "resourcemanager_custom_endpoint", providerData.ResourceManagerCustomEndpoint)
		rmClient, err = resourcemanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithServiceAccountEmail(providerData.ServiceAccountEmail),
			config.WithEndpoint(providerData.ResourceManagerCustomEndpoint),
		)
	} else {
		rmClient, err = resourcemanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithServiceAccountEmail(providerData.ServiceAccountEmail),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring Resource Manager API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	var aClient *authorization.APIClient
	if providerData.AuthorizationCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "authorization_custom_endpoint", providerData.AuthorizationCustomEndpoint)
		aClient, err = authorization.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.AuthorizationCustomEndpoint),
		)
	} else {
		aClient, err = authorization.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring Membership API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.resourceManagerClient = rmClient
	r.authorizationClient = aClient
	tflog.Info(ctx, "Resource Manager project client configured")
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                        "Resource Manager project resource schema. To use this resource, it is required that you set the service account email in the provider configuration.",
		"id":                          "Terraform's internal resource ID. It is structured as \"`container_id`\".",
		"project_id":                  "Project UUID identifier. This is the ID that can be used in most of the other resources to identify the project.",
		"container_id":                "Project container ID. Globally unique, user-friendly identifier.",
		"parent_container_id":         "Parent resource identifier. Both container ID (user-friendly) and UUID are supported",
		"name":                        "Project name.",
		"labels":                      "Labels are key-value string pairs which can be attached to a resource container. A label key must match the regex [A-ZÄÜÖa-zäüöß0-9_-]{1,64}. A label value must match the regex ^$|[A-ZÄÜÖa-zäüöß0-9_-]{1,64}",
		"owner_email":                 "Email address of the owner of the project. This value is only considered during creation. Changing it afterwards will have no effect.",
		"members":                     "The members assigned to the project. At least one subject needs to be a user, and not a client or service account.",
		"members.role":                fmt.Sprintf("The role of the member in the project. Possible values include, but are not limited to: `owner`, `editor`, `reader`. Legacy roles (%s) are not supported.", strings.Join(utils.QuoteValues(utils.LegacyProjectRoles), ", ")),
		"members.subject":             "Unique identifier of the user, service account or client. This is usually the email address for users or service accounts, and the name in case of clients.",
		"members_deprecation_message": "The \"members\" field has been deprecated in favor of the \"owner_email\" field. Please use the \"owner_email\" field to assign the owner role to a user.",
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
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"container_id": schema.StringAttribute{
				Description: descriptions["container_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"parent_container_id": schema.StringAttribute{
				Description: descriptions["parent_container_id"],
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`[A-ZÄÜÖa-zäüöß0-9_-]{1,64}`),
							"must match expression"),
					),
					mapvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^$|[A-ZÄÜÖa-zäüöß0-9_-]{1,64}`),
							"must match expression"),
					),
				},
			},
			"owner_email": schema.StringAttribute{
				Description: descriptions["owner_email"],
				Required:    true,
			},
			"members": schema.ListNestedAttribute{
				Description:         descriptions["members"],
				DeprecationMessage:  descriptions["members_deprecation_message"],
				MarkdownDescription: fmt.Sprintf("%s\n\n!> %s", descriptions["members"], descriptions["members_deprecation_message"]),
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							Description: descriptions["members.role"],
							Required:    true,
							Validators: []validator.String{
								validate.NonLegacyProjectRole(),
							},
						},
						"subject": schema.StringAttribute{
							Description: descriptions["members.subject"],
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// ModifyPlan will be called in the Plan phase and will check if the members field is set
func (r *projectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	if req.Plan.Raw.IsNull() { // Plan is to destroy the resource
		return
	}

	var model Model
	diags := req.Plan.Get(ctx, &model)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if model.ProjectId.IsNull() || model.ProjectId.IsUnknown() || // Project does not exist yet
		model.Members.IsNull() || model.Members.IsUnknown() { // Members field is not set
		return
	}

	membersResp, err := r.authorizationClient.ListMembersExecute(ctx, projectResourceType, model.ProjectId.ValueString())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error preparing the plan", fmt.Sprintf("Reading members: %v", err))
		return
	}
	members := []string{}
	for _, m := range *membersResp.Members {
		if utils.IsLegacyProjectRole(*m.Role) {
			continue
		}
		members = append(members, fmt.Sprintf("  - %s (%s)", *m.Subject, *m.Role))
	}

	core.LogAndAddWarning(ctx, &resp.Diagnostics, "The members set in the \"members\" field will override the current members in your project",
		fmt.Sprintf("%s\n%s\n%s\n\n%s",
			"The current members in your project will be removed and replaced with the members set in the \"members\" field.",
			"This might not be represented in the Terraform plan if you are migrating from the \"owner_email\" field, since the current members are not yet set in the state.",
			"Please make sure that the members in the \"members\" field are correct and complete.",
			fmt.Sprintf("Current members in your project:\n%v", strings.Join(members, "\n"))))
}

// ConfigValidators validates the resource configuration
func (r *projectResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("owner_email"),
			path.MatchRoot("members"),
		),
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "project_container_id", containerId)

	serviceAccountEmail := r.resourceManagerClient.GetConfig().ServiceAccountEmail
	if serviceAccountEmail == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", "The service account e-mail cannot be empty: set it in the provider configuration or through the STACKIT_SERVICE_ACCOUNT_EMAIL or in your credentials file (default filepath is ~/.stackit/credentials.json)")
		return
	}

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new project
	createResp, err := r.resourceManagerClient.CreateProject(ctx).CreateProjectPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Calling API: %v", err))
		return
	}
	respContainerId := *createResp.ContainerId

	// If the request has not been processed yet and the containerId doesnt exist,
	// the waiter will fail with authentication error, so wait some time before checking the creation
	waitResp, err := wait.CreateProjectWaitHandler(ctx, r.resourceManagerClient, respContainerId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	err = mapProjectFields(ctx, waitResp, &model, &resp.State)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	membersResp, err := r.authorizationClient.ListMembersExecute(ctx, projectResourceType, *waitResp.ProjectId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Reading members: %v", err))
		return
	}

	err = mapMembersFields(ctx, membersResp.Members, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Resource Manager project created")
}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "container_id", containerId)

	projectResp, err := r.resourceManagerClient.GetProject(ctx, containerId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusForbidden {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapProjectFields(ctx, projectResp, &model, &resp.State)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set refreshed model
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Resource Manager project read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "container_id", containerId)

	// Generate API request body from model
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating project", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing project
	_, err = r.resourceManagerClient.PartialUpdateProject(ctx, containerId).PartialUpdateProjectPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Fetch updated project
	projectResp, err := r.resourceManagerClient.GetProject(ctx, containerId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating project", fmt.Sprintf("Calling API for updated data: %v", err))
		return
	}

	err = mapProjectFields(ctx, projectResp, &model, &resp.State)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating project", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Resource Manager project updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "container_id", containerId)

	// Delete existing project
	err := r.resourceManagerClient.DeleteProject(ctx, containerId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.DeleteProjectWaitHandler(ctx, r.resourceManagerClient, containerId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting project", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Resource Manager project deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: container_id
func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 1 || idParts[0] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing project",
			fmt.Sprintf("Expected import identifier with format: [container_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = tflog.SetField(ctx, "container_id", req.ID)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("container_id"), req.ID)...)
	tflog.Info(ctx, "Resource Manager Project state imported")
}

// mapProjectFields maps the API response to the Terraform model and update the Terraform state
func mapProjectFields(ctx context.Context, projectResp *resourcemanager.GetProjectResponse, model *Model, state *tfsdk.State) (err error) {
	if projectResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var projectId string
	if model.ProjectId.ValueString() != "" {
		projectId = model.ProjectId.ValueString()
	} else if projectResp.ProjectId != nil {
		projectId = *projectResp.ProjectId
	} else {
		return fmt.Errorf("project id not present")
	}

	var containerId string
	if model.ContainerId.ValueString() != "" {
		containerId = model.ContainerId.ValueString()
	} else if projectResp.ContainerId != nil {
		containerId = *projectResp.ContainerId
	} else {
		return fmt.Errorf("container id not present")
	}

	var labels basetypes.MapValue
	if projectResp.Labels != nil && len(*projectResp.Labels) != 0 {
		labels, err = conversion.ToTerraformStringMap(ctx, *projectResp.Labels)
		if err != nil {
			return fmt.Errorf("converting to StringValue map: %w", err)
		}
	} else {
		labels = types.MapNull(types.StringType)
	}

	var containerParentIdTF basetypes.StringValue
	if projectResp.Parent != nil {
		if _, err := uuid.Parse(model.ContainerParentId.ValueString()); err == nil {
			// the provided containerParentId is the UUID identifier
			containerParentIdTF = types.StringPointerValue(projectResp.Parent.Id)
		} else {
			// the provided containerParentId is the user-friendly container id
			containerParentIdTF = types.StringPointerValue(projectResp.Parent.ContainerId)
		}
	} else {
		containerParentIdTF = types.StringNull()
	}

	model.Id = types.StringValue(containerId)
	model.ProjectId = types.StringValue(projectId)
	model.ContainerParentId = containerParentIdTF
	model.ContainerId = types.StringValue(containerId)
	model.Name = types.StringPointerValue(projectResp.Name)
	model.Labels = labels

	if state != nil {
		diags := diag.Diagnostics{}
		diags.Append(state.SetAttribute(ctx, path.Root("id"), model.Id)...)
		diags.Append(state.SetAttribute(ctx, path.Root("project_id"), model.ProjectId)...)
		diags.Append(state.SetAttribute(ctx, path.Root("parent_container_id"), model.ContainerParentId)...)
		diags.Append(state.SetAttribute(ctx, path.Root("container_id"), model.ContainerId)...)
		diags.Append(state.SetAttribute(ctx, path.Root("name"), model.Name)...)
		diags.Append(state.SetAttribute(ctx, path.Root("labels"), model.Labels)...)
		if diags.HasError() {
			return fmt.Errorf("update terraform state: %w", core.DiagsToError(diags))
		}
	}

	return nil
}

func mapMembersFields(ctx context.Context, members *[]authorization.Member, model *Model) error {
	if members == nil {
		model.Members = types.ListNull(types.ObjectType{AttrTypes: memberTypes})
		return nil
	}

	if (model.Members.IsNull() || model.Members.IsUnknown()) && !model.OwnerEmail.IsNull() {
		// If the new "members" field is not set and the deprecated "owner_email" field is set,
		// we keep the old behavior and do map the members to avoid an inconsistent result after apply error
		model.Members = types.ListNull(types.ObjectType{AttrTypes: memberTypes})
		return nil
	}

	modelMembers := []member{}
	if !(model.Members.IsNull() || model.Members.IsUnknown()) {
		diags := model.Members.ElementsAs(ctx, &modelMembers, false)
		if diags.HasError() {
			return fmt.Errorf("processing members: %w", core.DiagsToError(diags))
		}
	}
	modelMemberIds := make([]string, len(modelMembers))
	for i, m := range modelMembers {
		modelMemberIds[i] = memberId(authorization.Member{
			Role:    m.Role.ValueStringPointer(),
			Subject: m.Subject.ValueStringPointer(),
		})
	}

	apiMemberIds := []string{}
	for _, m := range *members {
		if utils.IsLegacyProjectRole(*m.Role) {
			continue
		}
		apiMemberIds = append(apiMemberIds, memberId(m))
	}

	reconciledMembersIds := utils.ReconcileStringSlices(modelMemberIds, apiMemberIds)

	membersList := []attr.Value{}
	for i, m := range reconciledMembersIds {
		role := roleFromId(m)
		subject := subjectFromId(m)
		if role == "" || subject == "" {
			return fmt.Errorf("reconcile list of members")
		}

		membersMap := map[string]attr.Value{
			"subject": types.StringValue(subject),
			"role":    types.StringValue(role),
		}

		memberTF, diags := types.ObjectValue(memberTypes, membersMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		membersList = append(membersList, memberTF)
	}

	membersTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: memberTypes},
		membersList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	model.Members = membersTF
	return nil
}

func toMembersPayload(ctx context.Context, model *Model) (*[]authorization.Member, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if model.Members.IsNull() || model.Members.IsUnknown() {
		if model.OwnerEmail.IsNull() {
			return nil, fmt.Errorf("members and owner_email are both null or unknown")
		}

		return &[]authorization.Member{
			{
				Subject: model.OwnerEmail.ValueStringPointer(),
				Role:    sdkUtils.Ptr(projectOwnerRole),
			},
		}, nil
	}

	membersModel := []member{}
	diags := model.Members.ElementsAs(ctx, &membersModel, false)
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	// If the new "members" fields is set, it has precedence over the "owner_email" field
	members := []authorization.Member{}
	for _, m := range membersModel {
		members = append(members, authorization.Member{
			Role:    m.Role.ValueStringPointer(),
			Subject: m.Subject.ValueStringPointer(),
		})
	}

	return &members, nil
}

func toCreatePayload(ctx context.Context, model *Model) (*resourcemanager.CreateProjectPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	members, err := toMembersPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("processing members: %w", err)
	}
	var convertedMembers []resourcemanager.Member
	for _, m := range *members {
		convertedMembers = append(convertedMembers,
			resourcemanager.Member{
				Subject: m.Subject,
				Role:    m.Role,
			})
	}
	var membersPayload *[]resourcemanager.Member
	if len(convertedMembers) > 0 {
		membersPayload = &convertedMembers
	}

	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &resourcemanager.CreateProjectPayload{
		ContainerParentId: conversion.StringValueToPointer(model.ContainerParentId),
		Labels:            labels,
		Members:           membersPayload,
		Name:              conversion.StringValueToPointer(model.Name),
	}, nil
}

func toUpdatePayload(model *Model) (*resourcemanager.PartialUpdateProjectPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to GO map: %w", err)
	}

	return &resourcemanager.PartialUpdateProjectPayload{
		ContainerParentId: conversion.StringValueToPointer(model.ContainerParentId),
		Name:              conversion.StringValueToPointer(model.Name),
		Labels:            labels,
	}, nil
}

// Internal representation of a member, which is uniquely identified by the subject and role
func memberId(member authorization.Member) string {
	return fmt.Sprintf("%s,%s", *member.Subject, *member.Role)
}

// Extract the role from the member ID representation
func roleFromId(id string) string {
	parts := strings.Split(id, ",")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// Extract the subject from the member ID representation
func subjectFromId(id string) string {
	parts := strings.Split(id, ",")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}
