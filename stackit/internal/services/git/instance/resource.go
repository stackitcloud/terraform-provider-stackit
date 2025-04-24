package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/git"
	"github.com/stackitcloud/stackit-sdk-go/services/git/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitResource{}
	_ resource.ResourceWithConfigure   = &gitResource{}
	_ resource.ResourceWithImportState = &gitResource{}
)

// Model represents the schema for the git resource.
type Model struct {
	Id         types.String `tfsdk:"id"`          // Required by Terraform
	ProjectId  types.String `tfsdk:"project_id"`  // ProjectId associated with the git instance
	InstanceId types.String `tfsdk:"instance_id"` // InstanceId associated with the git instance
	Name       types.String `tfsdk:"name"`        // Name linked to the git instance
	Url        types.String `tfsdk:"url"`         // Url linked to the git instance
	Version    types.String `tfsdk:"version"`     // Version linked to the git instance
}

// NewGitResource is a helper function to create a new git resource instance.
func NewGitResource() resource.Resource {
	return &gitResource{}
}

// gitResource implements the resource interface for git instances.
type gitResource struct {
	client *git.APIClient
}

// descriptions for the attributes in the Schema
var descriptions = map[string]string{
	"id":          "Terraform's internal resource ID, structured as \"`project_id`,`instance_id`\".",
	"project_id":  "STACKIT project ID to which the git instance is associated.",
	"instance_id": "ID linked to the git instance.",
	"name":        "Unique name linked to the git instance.",
	"url":         "Url linked to the git instance.",
	"version":     "Version linked to the git instance.",
}

// Configure sets up the API client for the git instance resource.
func (g *gitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent potential panics if the provider is not properly configured.
	if req.ProviderData == nil {
		return
	}

	// Validate provider data type before proceeding.
	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_git", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	// Initialize the API client with the appropriate authentication and endpoint settings.
	var apiClient *git.APIClient
	var err error
	if providerData.GitCustomEndpoint != "" {
		apiClient, err = git.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.GitCustomEndpoint),
		)
	} else {
		apiClient, err = git.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	// Handle API client initialization errors.
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	// Store the initialized client.
	g.client = apiClient
	tflog.Info(ctx, "git client configured")
}

// Metadata sets the resource type name for the git instance resource.
func (g *gitResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_git"
}

// Schema defines the schema for the resource.
func (g *gitResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Git Instance resource schema."),
		Description:         "Git Instance resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(5, 32),
				},
			},
			"url": schema.StringAttribute{
				Description: descriptions["url"],
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state for the git instance.
func (g *gitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the planned values for the resource.
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set logging context with the project ID and instance ID.
	projectId := model.ProjectId.ValueString()
	instanceName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_name", instanceName)

	// Create the new git instance via the API client.
	gitInstanceResp, err := g.client.CreateInstance(ctx, projectId).
		CreateInstancePayload(git.CreateInstancePayload{Name: &instanceName}).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	gitInstanceId := *gitInstanceResp.Id
	_, err = wait.CreateGitInstanceWaitHandler(ctx, g.client, projectId, gitInstanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating git instance", fmt.Sprintf("Git instance creation waiting: %v", err))
		return
	}

	err = mapFields(gitInstanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating git instance", fmt.Sprintf("Mapping fields: %v", err))
		return
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Git Instance created")
}

// Read refreshes the Terraform state with the latest git instance data.
func (g *gitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract the project ID and instance id of the model
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()

	// Read the current git instance via id
	gitInstanceResp, err := g.client.GetInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(gitInstanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading git instance", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

// Update attempts to update the resource. In this case, git instances cannot be updated.
// Note: This method is intentionally left without update logic because changes
// to 'project_id' or 'name' require the resource to be entirely replaced.
// As a result, the Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (g *gitResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// git instances cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating git instance", "Git Instance can't be updated")
}

// Delete deletes the git instance and removes it from the Terraform state on success.
func (g *gitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// Call API to delete the existing git instance.
	err := g.client.DeleteInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Git instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (g *gitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Split the import identifier to extract project ID and email.
	idParts := strings.Split(req.ID, core.Separator)

	// Ensure the import identifier format is correct.
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing git instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	instanceId := idParts[1]

	// Set the project ID and instance ID attributes in the state.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceId)...)
	tflog.Info(ctx, "Git instance state imported")
}

// mapFields maps a Git response to the model.
func mapFields(resp *git.Instance, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if resp.Id == nil {
		return fmt.Errorf("git instance id not present")
	}

	// Build the ID by combining the project ID and instance id and assign the model's fields.
	idParts := []string{model.ProjectId.ValueString(), *resp.Id}
	model.Id = types.StringValue(strings.Join(idParts, core.Separator))
	model.Url = types.StringPointerValue(resp.Url)
	model.Name = types.StringPointerValue(resp.Name)
	model.InstanceId = types.StringPointerValue(resp.Id)
	model.Version = types.StringPointerValue(resp.Version)

	return nil
}
