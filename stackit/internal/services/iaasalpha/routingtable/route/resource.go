package route

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/shared"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasalphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &routeResource{}
	_ resource.ResourceWithConfigure   = &routeResource{}
	_ resource.ResourceWithImportState = &routeResource{}
	_ resource.ResourceWithModifyPlan  = &routeResource{}
)

// NewRoutingTableRouteResource is a helper function to simplify the provider implementation.
func NewRoutingTableRouteResource() resource.Resource {
	return &routeResource{}
}

// routeResource is the resource implementation.
type routeResource struct {
	client       *iaasalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *routeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_table_route"
}

// Configure adds the provider configured client to the resource.
func (r *routeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &r.providerData, features.RoutingTablesExperiment, "stackit_routing_table_route", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := iaasalphaUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "IaaS alpha client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *routeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}

	var configModel shared.RouteModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var planModel shared.RouteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Schema defines the schema for the resource.
func (r *routeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Routing table route resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.RoutingTablesExperiment, core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`organization_id`,`region`,`network_area_id`,`routing_table_id`,`route_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT organization ID to which the routing table is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"routing_table_id": schema.StringAttribute{
				Description: "The routing tables ID.",
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
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"route_id": schema.StringAttribute{
				Description: "The ID of the route.",
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
			"destination": schema.SingleNestedAttribute{
				Description: "Destination of the route.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: fmt.Sprintf("CIDRV type. %s %s", utils.FormatPossibleValues("cidrv4", "cidrv6"), "Only `cidrv4` is supported during experimental stage."),
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"value": schema.StringAttribute{
						Description: "An CIDR string.",
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"network_area_id": schema.StringAttribute{
				Description: "The network area ID to which the routing table is associated.",
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
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
			"next_hop": schema.SingleNestedAttribute{
				Description: "Next hop destination.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Type of the next hop. " + utils.FormatPossibleValues("blackhole", "internet", "ipv4", "ipv6"),
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"value": schema.StringAttribute{
						Description: "Either IPv4 or IPv6 (not set for blackhole and internet). Only IPv4 supported during experimental stage.",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date-time when the route was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date-time when the route was updated.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *routeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model shared.RouteModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	routingTableId := model.RoutingTableId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)

	// Create new routing table route
	payload, err := toCreatePayload(ctx, &model.RouteReadModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating routing table route", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	routeResp, err := r.client.AddRoutesToRoutingTable(ctx, organizationId, networkAreaId, region, routingTableId).AddRoutesToRoutingTablePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating routing table route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFieldsFromList(ctx, routeResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating routing table route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "route_id", model.RouteId.ValueString())

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table route created")
}

// Read refreshes the Terraform state with the latest data.
func (r *routeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model shared.RouteModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	organizationId := model.OrganizationId.ValueString()
	routingTableId := model.RoutingTableId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	routeId := model.RouteId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "route_id", routeId)

	routeResp, err := r.client.GetRouteOfRoutingTable(ctx, organizationId, networkAreaId, region, routingTableId, routeId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading routing table route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = shared.MapRouteModel(ctx, routeResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading routing table route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table route read.")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *routeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model shared.RouteModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	routingTableId := model.RoutingTableId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	routeId := model.RouteId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "route_id", routeId)

	// Retrieve values from state
	var stateModel shared.RouteModel
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating routing table route", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	route, err := r.client.UpdateRouteOfRoutingTable(ctx, organizationId, networkAreaId, region, routingTableId, routeId).UpdateRouteOfRoutingTablePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating routing table route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = shared.MapRouteModel(ctx, route, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating routing table route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table route updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *routeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model shared.RouteModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	routingTableId := model.RoutingTableId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	routeId := model.RouteId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "route_id", routeId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing routing table route
	err := r.client.DeleteRouteFromRoutingTable(ctx, organizationId, networkAreaId, region, routingTableId, routeId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error routing table route", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "Routing table route deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the routing table route resource import identifier is: organization_id,region,network_area_id,routing_table_id,route_id
func (r *routeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 5 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" || idParts[4] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing routing table",
			fmt.Sprintf("Expected import identifier with format: [organization_id],[region],[network_area_id],[routing_table_id],[route_id]  Got: %q", req.ID),
		)
		return
	}

	organizationId := idParts[0]
	region := idParts[1]
	networkAreaId := idParts[2]
	routingTableId := idParts[3]
	routeId := idParts[4]
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "route_id", routeId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), region)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_area_id"), networkAreaId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("routing_table_id"), routingTableId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("route_id"), routeId)...)
	tflog.Info(ctx, "Routing table route state imported")
}

func mapFieldsFromList(ctx context.Context, routeResp *iaasalpha.RouteListResponse, model *shared.RouteModel, region string) error {
	if routeResp == nil || routeResp.Items == nil {
		return fmt.Errorf("response input is nil")
	} else if len(*routeResp.Items) < 1 {
		return fmt.Errorf("no routes found in response")
	} else if len(*routeResp.Items) > 1 {
		return fmt.Errorf("more than 1 route found in response")
	}

	route := (*routeResp.Items)[0]
	return shared.MapRouteModel(ctx, &route, model, region)
}

func toCreatePayload(ctx context.Context, model *shared.RouteReadModel) (*iaasalpha.AddRoutesToRoutingTablePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	nextHopPayload, err := toNextHopPayload(ctx, model)
	if err != nil {
		return nil, err
	}
	destinationPayload, err := toDestinationPayload(ctx, model)
	if err != nil {
		return nil, err
	}

	return &iaasalpha.AddRoutesToRoutingTablePayload{
		Items: &[]iaasalpha.Route{
			{
				Labels:      &labels,
				Nexthop:     nextHopPayload,
				Destination: destinationPayload,
			},
		},
	}, nil
}

func toUpdatePayload(ctx context.Context, model *shared.RouteModel, currentLabels types.Map) (*iaasalpha.UpdateRouteOfRoutingTablePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaasalpha.UpdateRouteOfRoutingTablePayload{
		Labels: &labels,
	}, nil
}

func toNextHopPayload(ctx context.Context, model *shared.RouteReadModel) (*iaasalpha.RouteNexthop, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if utils.IsUndefined(model.NextHop) {
		return nil, nil
	}

	nexthopModel := shared.RouteNextHop{}
	diags := model.NextHop.As(ctx, &nexthopModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	switch nexthopModel.Type.ValueString() {
	case "blackhole":
		return sdkUtils.Ptr(iaasalpha.NexthopBlackholeAsRouteNexthop(iaasalpha.NewNexthopBlackhole("blackhole"))), nil
	case "internet":
		return sdkUtils.Ptr(iaasalpha.NexthopInternetAsRouteNexthop(iaasalpha.NewNexthopInternet("internet"))), nil
	case "ipv4":
		return sdkUtils.Ptr(iaasalpha.NexthopIPv4AsRouteNexthop(iaasalpha.NewNexthopIPv4("ipv4", nexthopModel.Value.ValueString()))), nil
	case "ipv6":
		return sdkUtils.Ptr(iaasalpha.NexthopIPv6AsRouteNexthop(iaasalpha.NewNexthopIPv6("ipv6", nexthopModel.Value.ValueString()))), nil
	}
	return nil, fmt.Errorf("unknown nexthop type: %s", nexthopModel.Type.ValueString())
}

func toDestinationPayload(ctx context.Context, model *shared.RouteReadModel) (*iaasalpha.RouteDestination, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if utils.IsUndefined(model.Destination) {
		return nil, nil
	}

	destinationModel := shared.RouteDestination{}
	diags := model.Destination.As(ctx, &destinationModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	switch destinationModel.Type.ValueString() {
	case "cidrv4":
		return sdkUtils.Ptr(iaasalpha.DestinationCIDRv4AsRouteDestination(iaasalpha.NewDestinationCIDRv4("cidrv4", destinationModel.Value.ValueString()))), nil
	case "cidrv6":
		return sdkUtils.Ptr(iaasalpha.DestinationCIDRv6AsRouteDestination(iaasalpha.NewDestinationCIDRv6("cidrv6", destinationModel.Value.ValueString()))), nil
	}
	return nil, fmt.Errorf("unknown destination type: %s", destinationModel.Type.ValueString())
}
