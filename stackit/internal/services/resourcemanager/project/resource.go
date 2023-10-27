package project

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
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
	projectOwner = "project.owner"
)

type Model struct {
	Id                types.String `tfsdk:"id"` // needed by TF
	ContainerId       types.String `tfsdk:"container_id"`
	ContainerParentId types.String `tfsdk:"parent_container_id"`
	Name              types.String `tfsdk:"name"`
	Labels            types.Map    `tfsdk:"labels"`
	OwnerEmail        types.String `tfsdk:"owner_email"`
}

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	client *resourcemanager.APIClient
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

	var apiClient *resourcemanager.APIClient
	var err error
	if providerData.ResourceManagerCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "resourcemanager_custom_endpoint", providerData.ResourceManagerCustomEndpoint)
		apiClient, err = resourcemanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithServiceAccountEmail(providerData.ServiceAccountEmail),
			config.WithEndpoint(providerData.ResourceManagerCustomEndpoint),
		)
	} else {
		apiClient, err = resourcemanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithServiceAccountEmail(providerData.ServiceAccountEmail),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Resource Manager project client configured")
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                "Resource Manager project resource schema.",
		"id":                  "Terraform's internal resource ID. It is structured as \"`container_id`\".",
		"container_id":        "Project container ID. Globally unique, user-friendly identifier.",
		"parent_container_id": "Parent container ID",
		"name":                "Project name.",
		"labels":              "Labels are key-value string pairs which can be attached to a resource container. A label key must match the regex [A-ZÄÜÖa-zäüöß0-9_-]{1,64}. A label value must match the regex ^$|[A-ZÄÜÖa-zäüöß0-9_-]{1,64}",
		"owner_email":         "Email address of the owner of the project. This value is only considered during creation. Changing it afterwards will have no effect.",
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
							regexp.MustCompile(`[A-ZÄÜÖa-zäüöß0-9_-]{1,64}`),
							"must match expression"),
					),
				},
			},
			"owner_email": schema.StringAttribute{
				Description: descriptions["owner_email"],
				Required:    true,
			},
		},
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

	serviceAccountEmail := r.client.GetConfig().ServiceAccountEmail
	if serviceAccountEmail == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", "The service account e-mail cannot be empty: set it in the provider configuration or through the STACKIT_SERVICE_ACCOUNT_EMAIL or in your credentials file (default filepath is ~/.stackit/credentials.json)")
		return
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, serviceAccountEmail)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new project
	createResp, err := r.client.CreateProject(ctx).CreateProjectPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Calling API: %v", err))
		return
	}
	respContainerId := *createResp.ContainerId

	// If the request has not been processed yet and the containerId doesnt exist,
	// the waiter will fail with authentication error, so wait some time before checking the creation
	wr, err := wait.CreateProjectWaitHandler(ctx, r.client, respContainerId).SetSleepBeforeWait(1 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}
	got, ok := wr.(*resourcemanager.ProjectResponseWithParents)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Wait result conversion, got %+v", wr))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, got, &model)
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
	diags := req.State.Get(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "container_id", containerId)

	projectResp, err := r.client.GetProject(ctx, containerId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, projectResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Processing API payload: %v", err))
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
	_, err = r.client.UpdateProject(ctx, containerId).UpdateProjectPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Fetch updated zone
	projectResp, err := r.client.GetProject(ctx, containerId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating zone", fmt.Sprintf("Calling API for updated data: %v", err))
		return
	}
	err = mapFields(ctx, projectResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating zone", fmt.Sprintf("Processing API payload: %v", err))
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
	err := r.client.DeleteProject(ctx, containerId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.DeleteProjectWaitHandler(ctx, r.client, containerId).WaitWithContext(ctx)
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

func mapFields(ctx context.Context, projectResp *resourcemanager.ProjectResponseWithParents, model *Model) (err error) {
	if projectResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
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

	model.Id = types.StringValue(containerId)
	model.ContainerId = types.StringValue(containerId)
	if projectResp.Parent != nil {
		model.ContainerParentId = types.StringPointerValue(projectResp.Parent.ContainerId)
	} else {
		model.ContainerParentId = types.StringNull()
	}
	model.Name = types.StringPointerValue(projectResp.Name)
	model.Labels = labels
	return nil
}

func toCreatePayload(model *Model, serviceAccountEmail string) (*resourcemanager.CreateProjectPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	owner := projectOwner
	serviceAccountSubject := serviceAccountEmail
	members := []resourcemanager.ProjectMember{
		{
			Subject: &serviceAccountSubject,
			Role:    &owner,
		},
	}

	ownerSubject := model.OwnerEmail.ValueString()
	if ownerSubject != "" && ownerSubject != serviceAccountSubject {
		members = append(members,
			resourcemanager.ProjectMember{
				Subject: &ownerSubject,
				Role:    &owner,
			})
	}

	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to GO map: %w", err)
	}

	return &resourcemanager.CreateProjectPayload{
		ContainerParentId: model.ContainerParentId.ValueStringPointer(),
		Labels:            labels,
		Members:           &members,
		Name:              model.Name.ValueStringPointer(),
	}, nil
}

func toUpdatePayload(model *Model) (*resourcemanager.UpdateProjectPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to GO map: %w", err)
	}

	return &resourcemanager.UpdateProjectPayload{
		ContainerParentId: model.ContainerParentId.ValueStringPointer(),
		Name:              model.Name.ValueStringPointer(),
		Labels:            labels,
	}, nil
}
