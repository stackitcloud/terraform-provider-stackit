package securitygroup

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &securityGroupResource{}
	_ resource.ResourceWithConfigure   = &securityGroupResource{}
	_ resource.ResourceWithImportState = &securityGroupResource{}
)

type Model struct {
	Id              types.String `tfsdk:"id"` // needed by TF
	ProjectId       types.String `tfsdk:"project_id"`
	SecurityGroupId types.String `tfsdk:"security_group_id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Labels          types.Map    `tfsdk:"labels"`
	Stateful        types.Bool   `tfsdk:"stateful"`
}

// NewSecurityGroupResource is a helper function to simplify the provider implementation.
func NewSecurityGroupResource() resource.Resource {
	return &securityGroupResource{}
}

// securityGroupResource is the resource implementation.
type securityGroupResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *securityGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

// Configure adds the provider configured client to the resource.
func (r *securityGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_security_group", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	var apiClient *iaas.APIClient
	var err error
	if providerData.IaaSCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "iaas_custom_endpoint", providerData.IaaSCustomEndpoint)
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.GetRegion()),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *securityGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Security group resource schema. Must have a `region` specified in the provider configuration."),
		Description:         "Security group resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`security_group_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the security group is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"security_group_id": schema.StringAttribute{
				Description: "The security group ID.",
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
				Description: "The name of the security group.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the security group.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(127),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
			"stateful": schema.BoolAttribute{
				Description: "Configures if a security group is stateful or stateless. There can only be one type of security groups per network interface/server.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *securityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating security group", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new security group

	securityGroup, err := r.client.CreateSecurityGroup(ctx, projectId).CreateSecurityGroupPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating security group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	securityGroupId := *securityGroup.Id

	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)

	// Map response body to schema
	err = mapFields(ctx, securityGroup, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating security group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Security group created")
}

// Read refreshes the Terraform state with the latest data.
func (r *securityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	securityGroupId := model.SecurityGroupId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "security_id", securityGroupId)

	securityGroupResp, err := r.client.GetSecurityGroup(ctx, projectId, securityGroupId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading security group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, securityGroupResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading security group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "security group read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *securityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	securityGroupId := model.SecurityGroupId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)

	// Retrieve values from state
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating security group", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing security group
	updatedSecurityGroup, err := r.client.UpdateSecurityGroup(ctx, projectId, securityGroupId).UpdateSecurityGroupPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating security group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, updatedSecurityGroup, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating security group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "security group updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *securityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	securityGroupId := model.SecurityGroupId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)

	// Delete existing security group
	err := r.client.DeleteSecurityGroup(ctx, projectId, securityGroupId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting security group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "security group deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,security_group_id
func (r *securityGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing security group",
			fmt.Sprintf("Expected import identifier with format: [project_id],[security_group_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	securityGroupId := idParts[1]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("security_group_id"), securityGroupId)...)
	tflog.Info(ctx, "security group state imported")
}

func mapFields(ctx context.Context, securityGroupResp *iaas.SecurityGroup, model *Model) error {
	if securityGroupResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var securityGroupId string
	if model.SecurityGroupId.ValueString() != "" {
		securityGroupId = model.SecurityGroupId.ValueString()
	} else if securityGroupResp.Id != nil {
		securityGroupId = *securityGroupResp.Id
	} else {
		return fmt.Errorf("security group id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		securityGroupId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	labels, diags := types.MapValueFrom(ctx, types.StringType, map[string]interface{}{})
	if diags.HasError() {
		return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
	}
	if securityGroupResp.Labels != nil && len(*securityGroupResp.Labels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *securityGroupResp.Labels)
		if diags.HasError() {
			return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else if model.Labels.IsNull() {
		labels = types.MapNull(types.StringType)
	}

	model.SecurityGroupId = types.StringValue(securityGroupId)
	model.Name = types.StringPointerValue(securityGroupResp.Name)
	model.Description = types.StringPointerValue(securityGroupResp.Description)
	model.Stateful = types.BoolPointerValue(securityGroupResp.Stateful)
	model.Labels = labels

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateSecurityGroupPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.CreateSecurityGroupPayload{
		Stateful:    conversion.BoolValueToPointer(model.Stateful),
		Description: conversion.StringValueToPointer(model.Description),
		Labels:      &labels,
		Name:        conversion.StringValueToPointer(model.Name),
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.UpdateSecurityGroupPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.UpdateSecurityGroupPayload{
		Description: conversion.StringValueToPointer(model.Description),
		Name:        conversion.StringValueToPointer(model.Name),
		Labels:      &labels,
	}, nil
}
