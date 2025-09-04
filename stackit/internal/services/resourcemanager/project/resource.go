package project

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	resourcemanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
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
	projectOwnerRole = "owner"
)

type Model struct {
	Id                types.String `tfsdk:"id"` // needed by TF
	ProjectId         types.String `tfsdk:"project_id"`
	ContainerId       types.String `tfsdk:"container_id"`
	ContainerParentId types.String `tfsdk:"parent_container_id"`
	Name              types.String `tfsdk:"name"`
	Labels            types.Map    `tfsdk:"labels"`
	CreationTime      types.String `tfsdk:"creation_time"`
	UpdateTime        types.String `tfsdk:"update_time"`
}

type ResourceModel struct {
	Model
	OwnerEmail types.String `tfsdk:"owner_email"`
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
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := resourcemanagerUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Resource Manager project client configured")
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main": fmt.Sprintf("%s\n\n%s",
			"Resource Manager project resource schema.",
			"-> In case you're getting started with an empty STACKIT organization and want to use this resource to create projects in it, check out [this guide](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs/guides/stackit_org_service_account) for how to create a service account which you can use for authentication in the STACKIT Terraform provider.",
		),
		"id":                  "Terraform's internal resource ID. It is structured as \"`container_id`\".",
		"project_id":          "Project UUID identifier. This is the ID that can be used in most of the other resources to identify the project.",
		"container_id":        "Project container ID. Globally unique, user-friendly identifier.",
		"parent_container_id": "Parent resource identifier. Both container ID (user-friendly) and UUID are supported",
		"name":                "Project name.",
		"labels":              "Labels are key-value string pairs which can be attached to a resource container. A label key must match the regex [A-ZÄÜÖa-zäüöß0-9_-]{1,64}. A label value must match the regex ^$|[A-ZÄÜÖa-zäüöß0-9_-]{1,64}.  \nTo create a project within a STACKIT Network Area, setting the label `networkArea=<networkAreaID>` is required. This can not be changed after project creation.",
		"owner_email":         "Email address of the owner of the project. This value is only considered during creation. Changing it afterwards will have no effect.",
		"creation_time":       "Date-time at which the project was created.",
		"update_time":         "Date-time at which the project was last modified.",
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
			"creation_time": schema.StringAttribute{
				Description: descriptions["creation_time"],
				Computed:    true,
			},
			"update_time": schema.StringAttribute{
				Description: descriptions["update_time"],
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model ResourceModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "project_container_id", containerId)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
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

	// If the request has not been processed yet and the containerId doesn't exist,
	// the waiter will fail with authentication error, so wait some time before checking the creation
	waitResp, err := wait.CreateProjectWaitHandler(ctx, r.client, respContainerId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	err = mapProjectFields(ctx, waitResp, &model.Model, &resp.State)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating project", fmt.Sprintf("Processing API response: %v", err))
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
	var model ResourceModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "container_id", containerId)

	projectResp, err := r.client.GetProject(ctx, containerId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusForbidden {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapProjectFields(ctx, projectResp, &model.Model, &resp.State)
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
	var model ResourceModel
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
	_, err = r.client.PartialUpdateProject(ctx, containerId).PartialUpdateProjectPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Fetch updated project
	projectResp, err := r.client.GetProject(ctx, containerId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating project", fmt.Sprintf("Calling API for updated data: %v", err))
		return
	}

	err = mapProjectFields(ctx, projectResp, &model.Model, &resp.State)
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
	var model ResourceModel
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
	model.CreationTime = types.StringValue(projectResp.CreationTime.Format(time.RFC3339))
	model.UpdateTime = types.StringValue(projectResp.UpdateTime.Format(time.RFC3339))

	if state != nil {
		diags := diag.Diagnostics{}
		diags.Append(state.SetAttribute(ctx, path.Root("id"), model.Id)...)
		diags.Append(state.SetAttribute(ctx, path.Root("project_id"), model.ProjectId)...)
		diags.Append(state.SetAttribute(ctx, path.Root("parent_container_id"), model.ContainerParentId)...)
		diags.Append(state.SetAttribute(ctx, path.Root("container_id"), model.ContainerId)...)
		diags.Append(state.SetAttribute(ctx, path.Root("name"), model.Name)...)
		diags.Append(state.SetAttribute(ctx, path.Root("labels"), model.Labels)...)
		diags.Append(state.SetAttribute(ctx, path.Root("creation_time"), model.CreationTime)...)
		diags.Append(state.SetAttribute(ctx, path.Root("update_time"), model.UpdateTime)...)
		if diags.HasError() {
			return fmt.Errorf("update terraform state: %w", core.DiagsToError(diags))
		}
	}

	return nil
}

func toMembersPayload(model *ResourceModel) (*[]resourcemanager.Member, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if model.OwnerEmail.IsNull() {
		return nil, fmt.Errorf("owner_email is null")
	}

	return &[]resourcemanager.Member{
		{
			Subject: model.OwnerEmail.ValueStringPointer(),
			Role:    sdkUtils.Ptr(projectOwnerRole),
		},
	}, nil
}

func toCreatePayload(model *ResourceModel) (*resourcemanager.CreateProjectPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	members, err := toMembersPayload(model)
	if err != nil {
		return nil, fmt.Errorf("processing members: %w", err)
	}

	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &resourcemanager.CreateProjectPayload{
		ContainerParentId: conversion.StringValueToPointer(model.ContainerParentId),
		Labels:            labels,
		Members:           members,
		Name:              conversion.StringValueToPointer(model.Name),
	}, nil
}

func toUpdatePayload(model *ResourceModel) (*resourcemanager.PartialUpdateProjectPayload, error) {
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
