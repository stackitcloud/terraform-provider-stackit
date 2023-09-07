package ske

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

type Model struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
}

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	client *ske.APIClient
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_project"
}

// Configure adds the provider configured client to the resource.
func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected stackit.ProviderData, got %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}

	var apiClient *ske.APIClient
	var err error
	if providerData.SKECustomEndpoint != "" {
		apiClient, err = ske.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.SKECustomEndpoint),
		)
	} else {
		apiClient, err = ske.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		resp.Diagnostics.AddError("Could not Configure API Client", err.Error())
		return
	}

	tflog.Info(ctx, "SKE project client configured")
	r.client = apiClient
}

// Schema returns the Terraform schema structure
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT Project ID in which the kubernetes project is enabled.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	_, err := r.client.CreateProject(ctx, projectId).Execute()
	if err != nil {
		resp.Diagnostics.AddError("failed during SKE project creation", err.Error())
		return
	}

	model.Id = types.StringValue(projectId)
	wr, err := ske.CreateProjectWaitHandler(ctx, r.client, projectId).SetTimeout(5 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster", fmt.Sprintf("Project creation waiting: %v", err))
		return
	}
	got, ok := wr.(*ske.ProjectResponse)
	if !ok {
		resp.Diagnostics.AddError("Error creating cluster", fmt.Sprintf("Wait result conversion, got %+v", got))
		return
	}
	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SKE project created or updated")
}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	// read
	_, err := r.client.GetProject(ctx, projectId).Execute()
	if err != nil {
		resp.Diagnostics.AddError("failed during SKE project read", err.Error())
		return
	}
	model.Id = types.StringValue(projectId)
	model.ProjectId = types.StringValue(projectId)
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "SKE project read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	resp.Diagnostics.AddError("Error updating ", "project can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	c := r.client
	_, err := c.DeleteProject(ctx, projectId).Execute()
	if err != nil {
		resp.Diagnostics.AddError("failed deleting project", err.Error())
		return
	}
	_, err = ske.DeleteProjectWaitHandler(ctx, r.client, projectId).SetTimeout(10 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting project", fmt.Sprintf("Project deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "SKE project deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id
func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) { // nolint:gocritic // function signature required by Terraform
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 1 || idParts[0] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: [project_id]  Got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	tflog.Info(ctx, "SKE project state imported")
}
