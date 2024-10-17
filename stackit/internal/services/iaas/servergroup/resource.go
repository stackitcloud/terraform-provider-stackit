package servergroup

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &serverGroupResource{}
	_ resource.ResourceWithConfigure   = &serverGroupResource{}
	_ resource.ResourceWithImportState = &serverGroupResource{}

	SupportedPolicyTypes = []string{"anti-affinity"}
)

type Model struct {
	Id            types.String `tfsdk:"id"` // needed by TF
	ProjectId     types.String `tfsdk:"project_id"`
	ServerGroupId types.String `tfsdk:"server_group_id"`
	Name          types.String `tfsdk:"name"`
	Policy        types.String `tfsdk:"policy"`
	MemberIds     types.List   `tfsdk:"member_ids"`
}

// NewServerGroupResource is a helper function to simplify the provider implementation.
func NewServerGroupResource() resource.Resource {
	return &serverGroupResource{}
}

// serverGroupResource is the resource implementation.
type serverGroupResource struct {
	client *iaasalpha.APIClient
}

// Metadata returns the resource type name.
func (r *serverGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_group"
}

// Configure adds the provider configured client to the resource.
func (r *serverGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !resourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_server_group", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	var apiClient *iaasalpha.APIClient
	var err error
	if providerData.IaaSCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "iaas_custom_endpoint", providerData.IaaSCustomEndpoint)
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "iaasalpha client configured")
}

// Schema defines the schema for the resource.
func (r *serverGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Server group resource schema. Must have a `region` specified in the provider configuration."),
		Description:         "Server group resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`server_group_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the server group is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"server_group_id": schema.StringAttribute{
				Description: "The server group ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the server group.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
			},
			"policy": schema.StringAttribute{
				Description: "The server group policy. " + utils.SupportedValuesDocumentation(SupportedPolicyTypes),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"member_ids": schema.ListAttribute{
				Description: "The UUIDs of servers that are part of the server group.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *serverGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server group", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new server group

	serverGroup, err := r.client.CreateServerGroup(ctx, projectId).CreateServerGroupPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = tflog.SetField(ctx, "server_group_id", *serverGroup.Id)

	// Map response body to schema
	err = mapFields(ctx, serverGroup, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server group created")
}

// Read refreshes the Terraform state with the latest data.
func (r *serverGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	serverGroupId := model.ServerGroupId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_group_id", serverGroupId)

	serverGroupResp, err := r.client.GetServerGroup(ctx, projectId, serverGroupId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, serverGroupResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "server group read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *serverGroupResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server group", "Server group can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *serverGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	serverGroupId := model.ServerGroupId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_group_id", serverGroupId)

	// Delete existing server group
	err := r.client.DeleteServerGroup(ctx, projectId, serverGroupId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting server group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "server group deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,server_group_id
func (r *serverGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing server group",
			fmt.Sprintf("Expected import identifier with format: [project_id],[server_group_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	serverGroupId := idParts[1]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_group_id", serverGroupId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_group_id"), serverGroupId)...)
	tflog.Info(ctx, "server group state imported")
}

func mapFields(ctx context.Context, serverGroupResp *iaasalpha.ServerGroup, model *Model) error {
	if serverGroupResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var serverGroupId string
	if model.ServerGroupId.ValueString() != "" {
		serverGroupId = model.ServerGroupId.ValueString()
	} else if serverGroupResp.Id != nil {
		serverGroupId = *serverGroupResp.Id
	} else {
		return fmt.Errorf("server group id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		serverGroupId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	if serverGroupResp.Members == nil {
		model.MemberIds = types.ListNull(types.StringType)
	} else {
		respServerGroups := *serverGroupResp.Members
		modelServerGroups, err := utils.ListValuetoStringSlice(model.MemberIds)
		if err != nil {
			return fmt.Errorf("get server group member ids from model: %w", err)
		}

		reconciledServerGroups := utils.ReconcileStringSlices(modelServerGroups, respServerGroups)

		serverGroupsTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledServerGroups)
		if diags.HasError() {
			return fmt.Errorf("map server group members: %w", core.DiagsToError(diags))
		}

		model.MemberIds = serverGroupsTF
	}

	model.ServerGroupId = types.StringValue(serverGroupId)
	model.Name = types.StringPointerValue(serverGroupResp.Name)
	model.Policy = types.StringPointerValue(serverGroupResp.Policy)

	return nil
}

func toCreatePayload(model *Model) (*iaasalpha.CreateServerGroupPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &iaasalpha.CreateServerGroupPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Policy: conversion.StringValueToPointer(model.Policy),
	}, nil
}
