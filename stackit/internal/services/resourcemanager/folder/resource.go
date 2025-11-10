package folder

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	resourcemanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &folderResource{}
	_ resource.ResourceWithConfigure   = &folderResource{}
	_ resource.ResourceWithImportState = &folderResource{}
)

const (
	projectOwnerRole = "owner"
)

type Model struct {
	Id                types.String `tfsdk:"id"` // needed by TF
	FolderId          types.String `tfsdk:"folder_id"`
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

// NewFolderResource is a helper function to simplify the provider implementation.
func NewFolderResource() resource.Resource {
	return &folderResource{}
}

// folderResource is the resource implementation.
type folderResource struct {
	client *resourcemanager.APIClient
}

// Metadata returns the resource type name.
func (r *folderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resourcemanager_folder"
}

// Configure adds the provider configured client to the resource.
func (r *folderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := resourcemanagerUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Resource Manager client configured")
}

// Schema defines the schema for the resource.
func (r *folderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                "Resource Manager folder resource schema.",
		"id":                  "Terraform's internal resource ID. It is structured as \"`container_id`\".",
		"container_id":        "Folder container ID. Globally unique, user-friendly identifier.",
		"folder_id":           "Folder UUID identifier. Globally unique folder identifier",
		"parent_container_id": "Parent resource identifier. Both container ID (user-friendly) and UUID are supported.",
		"name":                "The name of the folder.",
		"labels":              "Labels are key-value string pairs which can be attached to a resource container. A label key must match the regex [A-ZÄÜÖa-zäüöß0-9_-]{1,64}. A label value must match the regex ^$|[A-ZÄÜÖa-zäüöß0-9_-]{1,64}.",
		"owner_email":         "Email address of the owner of the folder. This value is only considered during creation. Changing it afterwards will have no effect.",
		"creation_time":       "Date-time at which the folder was created.",
		"update_time":         "Date-time at which the folder was last modified.",
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
			"folder_id": schema.StringAttribute{
				Description: descriptions["folder_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
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
func (r *folderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	tflog.Info(ctx, "creating folder")
	var model ResourceModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	containerParentId := model.ContainerParentId.ValueString()
	folderName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "container_parent_id", containerParentId)
	ctx = tflog.SetField(ctx, "folder_name", folderName)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating folder", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	folderCreateResp, err := r.client.CreateFolder(ctx).CreateFolderPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating folder", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if folderCreateResp.ContainerId == nil || *folderCreateResp.ContainerId == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating folder", "Container ID is missing")
		return
	}

	// This sleep is currently needed due to the IAM Cache.
	time.Sleep(10 * time.Second)

	folderGetResponse, err := r.client.GetFolderDetails(ctx, *folderCreateResp.ContainerId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating folder", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFolderFields(ctx, folderGetResponse, &model.Model, &resp.State)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "API response processing error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Info(ctx, "Folder created")
}

// Read refreshes the Terraform state with the latest data.
func (r *folderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model ResourceModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	containerId := model.ContainerId.ValueString()
	folderName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "folder_name", folderName)
	ctx = tflog.SetField(ctx, "container_id", containerId)

	folderResp, err := r.client.GetFolderDetails(ctx, containerId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusForbidden {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading folder", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFolderFields(ctx, folderResp, &model.Model, &resp.State)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading folder", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set refreshed model
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Resource Manager folder read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *folderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating folder", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing folder
	_, err = r.client.PartialUpdateFolder(ctx, containerId).PartialUpdateFolderPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating folder", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Fetch updated folder
	folderResp, err := r.client.GetFolderDetails(ctx, containerId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating folder", fmt.Sprintf("Calling API for updated data: %v", err))
		return
	}

	err = mapFolderFields(ctx, folderResp, &model.Model, &resp.State)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating folder", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Resource Manager folder updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *folderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model ResourceModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "container_id", containerId)

	// Delete existing folder
	err := r.client.DeleteFolder(ctx, containerId).Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error deleting folder. Deletion may fail because associated projects remain hidden for up to 7 days after user deletion due to technical requirements.",
			fmt.Sprintf("Calling API: %v", err),
		)
		return
	}

	tflog.Info(ctx, "Resource Manager folder deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: container_id
func (r *folderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 1 || idParts[0] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing folder",
			fmt.Sprintf("Expected import identifier with format: [container_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = tflog.SetField(ctx, "container_id", req.ID)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("container_id"), req.ID)...)
	tflog.Info(ctx, "Resource Manager folder state imported")
}

// mapFolderFields maps folder fields from a response into the Terraform model and optionally updates state.
func mapFolderFields(
	ctx context.Context,
	folderGetResponse *resourcemanager.GetFolderDetailsResponse,
	model *Model,
	state *tfsdk.State,
) error {
	if folderGetResponse == nil {
		return fmt.Errorf("folder get response is nil")
	}

	var folderId string
	if model.FolderId.ValueString() != "" {
		folderId = model.FolderId.ValueString()
	} else if folderGetResponse.FolderId != nil {
		folderId = *folderGetResponse.FolderId
	} else {
		return fmt.Errorf("folder id not present")
	}

	var containerId string
	if model.ContainerId.ValueString() != "" {
		containerId = model.ContainerId.ValueString()
	} else if folderGetResponse.ContainerId != nil {
		containerId = *folderGetResponse.ContainerId
	} else {
		return fmt.Errorf("container id not present")
	}

	var err error
	var tfLabels basetypes.MapValue
	if folderGetResponse.Labels != nil && len(*folderGetResponse.Labels) > 0 {
		tfLabels, err = conversion.ToTerraformStringMap(ctx, *folderGetResponse.Labels)
		if err != nil {
			return fmt.Errorf("converting to StringValue map: %w", err)
		}
	} else {
		tfLabels = types.MapNull(types.StringType)
	}

	var containerParentIdTF basetypes.StringValue
	if folderGetResponse.Parent != nil {
		if _, err := uuid.Parse(model.ContainerParentId.ValueString()); err == nil {
			// the provided containerParent is the UUID identifier
			containerParentIdTF = types.StringPointerValue(folderGetResponse.Parent.Id)
		} else {
			// the provided containerParent is the user-friendly container id
			containerParentIdTF = types.StringPointerValue(folderGetResponse.Parent.ContainerId)
		}
	} else {
		containerParentIdTF = types.StringNull()
	}

	model.Id = types.StringValue(containerId)
	model.FolderId = types.StringValue(folderId)
	model.ContainerId = types.StringValue(containerId)
	model.ContainerParentId = containerParentIdTF
	model.Name = types.StringPointerValue(folderGetResponse.Name)
	model.Labels = tfLabels
	model.CreationTime = types.StringValue(folderGetResponse.CreationTime.Format(time.RFC3339))
	model.UpdateTime = types.StringValue(folderGetResponse.UpdateTime.Format(time.RFC3339))

	if state != nil {
		diags := diag.Diagnostics{}
		diags.Append(state.SetAttribute(ctx, path.Root("id"), model.Id)...)
		diags.Append(state.SetAttribute(ctx, path.Root("folder_id"), model.FolderId)...)
		diags.Append(state.SetAttribute(ctx, path.Root("container_id"), model.ContainerId)...)
		diags.Append(state.SetAttribute(ctx, path.Root("parent_container_id"), model.ContainerParentId)...)
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

func toCreatePayload(model *ResourceModel) (*resourcemanager.CreateFolderPayload, error) {
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

	return &resourcemanager.CreateFolderPayload{
		ContainerParentId: conversion.StringValueToPointer(model.ContainerParentId),
		Labels:            labels,
		Members:           members,
		Name:              conversion.StringValueToPointer(model.Name),
	}, nil
}

func toUpdatePayload(model *ResourceModel) (*resourcemanager.PartialUpdateFolderPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelLabels := model.Labels.Elements()
	labels, err := conversion.ToOptStringMap(modelLabels)
	if err != nil {
		return nil, fmt.Errorf("converting to GO map: %w", err)
	}

	return &resourcemanager.PartialUpdateFolderPayload{
		ContainerParentId: conversion.StringValueToPointer(model.ContainerParentId),
		Name:              conversion.StringValueToPointer(model.Name),
		Labels:            labels,
	}, nil
}
