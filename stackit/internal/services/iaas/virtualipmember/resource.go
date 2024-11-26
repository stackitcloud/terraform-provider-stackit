package virtualipmember

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
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
	_ resource.Resource                = &virtualIPResource{}
	_ resource.ResourceWithConfigure   = &virtualIPResource{}
	_ resource.ResourceWithImportState = &virtualIPResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	ProjectId          types.String `tfsdk:"project_id"`
	NetworkId          types.String `tfsdk:"network_id"`
	VirtualIpId        types.String `tfsdk:"virtual_ip_id"`
	NetworkInterfaceId types.String `tfsdk:"network_interface_id"`
}

// NewVirtualIPMemberResource is a helper function to simplify the provider implementation.
func NewVirtualIPMemberResource() resource.Resource {
	return &virtualIPResource{}
}

// networkResource is the resource implementation.
type virtualIPResource struct {
	client *iaasalpha.APIClient
}

// Metadata returns the resource type name.
func (r *virtualIPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_ip_member"
}

// Configure adds the provider configured client to the resource.
func (r *virtualIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_virtual_ip_member", "resource")
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
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the resource.
func (r *virtualIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Virtual IP member resource schema. This resource allows you to add a network interface as a member of a virtual IP. Must have a `region` specified in the provider configuration.",
		MarkdownDescription: features.AddBetaDescription("Virtual IP member resource schema. This resource allows you to add a network interface as a member of a virtual IP. Must have a `region` specified in the provider configuration."),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`network_id`,`virtual_ip_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the virtual IP is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_id": schema.StringAttribute{
				Description: "The network ID to which the virtual IP is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"virtual_ip_id": schema.StringAttribute{
				Description: "The virtual IP ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_interface_id": schema.StringAttribute{
				Description: "The ID of the network interface to add as a member of the virtual IP.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *virtualIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "network_id", networkId)
	virtualIpId := model.VirtualIpId.ValueString()
	ctx = tflog.SetField(ctx, "virtual_ip_id", virtualIpId)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error adding virtual IP member", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Add member to virtual IP
	virtualIp, err := r.client.AddMemberToVirtualIP(ctx, projectId, networkId, virtualIpId).AddMemberToVirtualIPPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error adding virtual IP member", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(virtualIp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating virtual IP.", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Virtual IP member added")
}

// Read refreshes the Terraform state with the latest data.
func (r *virtualIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkAreaId := model.NetworkId.ValueString()
	networkAreaRouteId := model.VirtualIpId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkAreaId)
	ctx = tflog.SetField(ctx, "virtual_ip_id", networkAreaRouteId)

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Virtual IP member read")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *virtualIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	virtualIpId := model.VirtualIpId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "virtual_ip_id", virtualIpId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	// Generate API request body from model
	payload, err := toDeletePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error adding virtual IP member", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Delete existing virtual IP
	_, err = r.client.RemoveMemberFromVirtualIP(ctx, projectId, networkId, virtualIpId).RemoveMemberFromVirtualIPPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error removing virtual IP member", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Virtual IP member removed")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *virtualIPResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update is not supported, all fields require replace
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,network_aread_id,virtual_ip_id
func (r *virtualIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing virtual IP",
			fmt.Sprintf("Expected import identifier with format: [project_id],[network_id],[virtual_ip_id][member]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	networkAreaId := idParts[1]
	networkAreaRouteId := idParts[2]
	networkInterfaceId := idParts[3]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkAreaId)
	ctx = tflog.SetField(ctx, "virtual_ip_id", networkAreaRouteId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_area_id"), networkAreaId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("virtual_ip_id"), networkAreaRouteId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_interface_id"), networkInterfaceId)...)
	tflog.Info(ctx, "Virtual IP member state imported")
}

func mapFields(virtualIp *iaasalpha.VirtualIp, model *Model) error {
	if virtualIp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var virtualIpId string
	if model.VirtualIpId.ValueString() != "" {
		virtualIpId = model.VirtualIpId.ValueString()
	} else if virtualIp.Id != nil {
		virtualIpId = *virtualIp.Id
	} else {
		return fmt.Errorf("virtual IP id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		model.NetworkId.ValueString(),
		virtualIpId,
		model.NetworkInterfaceId.ValueString(),
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	return nil
}

func toCreatePayload(model *Model) (*iaasalpha.AddMemberToVirtualIPPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &iaasalpha.AddMemberToVirtualIPPayload{
		Member: conversion.StringValueToPointer(model.NetworkInterfaceId),
	}, nil
}

func toDeletePayload(model *Model) (*iaasalpha.RemoveMemberFromVirtualIPPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &iaasalpha.RemoveMemberFromVirtualIPPayload{
		Member: conversion.StringValueToPointer(model.NetworkInterfaceId),
	}, nil
}
