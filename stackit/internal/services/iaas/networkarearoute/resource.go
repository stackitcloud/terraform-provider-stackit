package networkarearoute

import (
	"context"
	"fmt"
	"net/http"
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
	_ resource.Resource                = &networkAreaRouteResource{}
	_ resource.ResourceWithConfigure   = &networkAreaRouteResource{}
	_ resource.ResourceWithImportState = &networkAreaRouteResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	OrganizationId     types.String `tfsdk:"organization_id"`
	NetworkAreaId      types.String `tfsdk:"network_area_id"`
	NetworkAreaRouteId types.String `tfsdk:"network_area_route_id"`
	NextHop            types.String `tfsdk:"next_hop"`
	Prefix             types.String `tfsdk:"prefix"`
}

// NewNetworkAreaRouteResource is a helper function to simplify the provider implementation.
func NewNetworkAreaRouteResource() resource.Resource {
	return &networkAreaRouteResource{}
}

// networkResource is the resource implementation.
type networkAreaRouteResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *networkAreaRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_area_route"
}

// Configure adds the provider configured client to the resource.
func (r *networkAreaRouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_network_area_route", "resource")
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
func (r *networkAreaRouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network area route resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Terraform's internal resource ID. It is structured as \"`organization_id`,`network_area_id`,`network_area_route_id`\".",
				MarkdownDescription: features.AddBetaDescription("Network area route resource schema. Must have a `region` specified in the provider configuration."),
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT organization ID to which the network area is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_area_id": schema.StringAttribute{
				Description: "The network area ID to which the network area route is associated.",
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
			"network_area_route_id": schema.StringAttribute{
				Description: "The network area route ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"next_hop": schema.StringAttribute{
				Description: "The IP address of the routing system, that will route the prefix configured. Should be a valid IPv4 address.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.IP(),
				},
			},
			"prefix": schema.StringAttribute{
				Description: "The network, that is reachable though the Next Hop. Should use CIDR notation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.CIDR(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *networkAreaRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	networkAreaId := model.NetworkAreaId.ValueString()
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area route", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network area route
	routes, err := r.client.CreateNetworkAreaRoute(ctx, organizationId, networkAreaId).CreateNetworkAreaRoutePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area route", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if routes.Items == nil || len(*routes.Items) == 0 {
		core.LogAndAddError(ctx, &resp.Diagnostics, "empty response from API", fmt.Sprintf("Routes count: %d", len(*routes.Items)))
		return
	}

	var routeId string
	var route iaas.Route
	for _, route = range *routes.Items {
		if !(*route.Prefix == model.Prefix.ValueString() && *route.Nexthop == model.NextHop.ValueString()) {
			core.LogAndAddError(ctx, &resp.Diagnostics, "new static route not found in API response", fmt.Sprintf("Prefix from the route in API: %s", *route.Prefix))
			return
		}

		routeId = *route.RouteId
	}

	ctx = tflog.SetField(ctx, "network_area_route_id", routeId)

	// Map response body to schema
	err = mapFields(&route, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area route created")
}

// Read refreshes the Terraform state with the latest data.
func (r *networkAreaRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	networkAreaRouteId := model.NetworkAreaRouteId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "network_area_route_id", networkAreaRouteId)

	networkAreaRouteResp, err := r.client.GetNetworkAreaRoute(ctx, organizationId, networkAreaId, networkAreaRouteId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(networkAreaRouteResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area route read")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkAreaRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	networkAreaRouteId := model.NetworkAreaRouteId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "network_area_route_id", networkAreaRouteId)

	// Delete existing network
	err := r.client.DeleteNetworkAreaRoute(ctx, organizationId, networkAreaId, networkAreaRouteId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Network area route deleted")
}

func (r *networkAreaRouteResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area route", "Network area route can't be updated")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: organization_id,network_aread_id,network_area_route_id
func (r *networkAreaRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network area route",
			fmt.Sprintf("Expected import identifier with format: [organization_id],[network_area_id],[network_area_route_id]  Got: %q", req.ID),
		)
		return
	}

	organizationId := idParts[0]
	networkAreaId := idParts[1]
	networkAreaRouteId := idParts[2]
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "network_area_route_id", networkAreaRouteId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_area_id"), networkAreaId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_area_route_id"), networkAreaRouteId)...)
	tflog.Info(ctx, "Network area route state imported")
}

func mapFields(networkAreaRoute *iaas.Route, model *Model) error {
	if networkAreaRoute == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkAreaRouteId string
	if model.NetworkAreaRouteId.ValueString() != "" {
		networkAreaRouteId = model.NetworkAreaRouteId.ValueString()
	} else if networkAreaRoute.RouteId != nil {
		networkAreaRouteId = *networkAreaRoute.RouteId
	} else {
		return fmt.Errorf("network area route id not present")
	}

	idParts := []string{
		model.OrganizationId.ValueString(),
		model.NetworkAreaId.ValueString(),
		networkAreaRouteId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	model.NetworkAreaRouteId = types.StringValue(networkAreaRouteId)
	model.NextHop = types.StringPointerValue(networkAreaRoute.Nexthop)
	model.Prefix = types.StringPointerValue(networkAreaRoute.Prefix)
	return nil
}

func toCreatePayload(model *Model) (*iaas.CreateNetworkAreaRoutePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &iaas.CreateNetworkAreaRoutePayload{
		Ipv4: &[]iaas.Route{
			{
				Prefix:  conversion.StringValueToPointer(model.Prefix),
				Nexthop: conversion.StringValueToPointer(model.NextHop),
			},
		},
	}, nil
}
