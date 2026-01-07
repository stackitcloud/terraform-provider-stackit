package networkarearoute

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &networkAreaRouteResource{}
	_ resource.ResourceWithConfigure    = &networkAreaRouteResource{}
	_ resource.ResourceWithImportState  = &networkAreaRouteResource{}
	_ resource.ResourceWithModifyPlan   = &networkAreaRouteResource{}
	_ resource.ResourceWithUpgradeState = &networkAreaRouteResource{}
)

// ModelV1 is the currently used model
type ModelV1 struct {
	Id                 types.String        `tfsdk:"id"` // needed by TF
	OrganizationId     types.String        `tfsdk:"organization_id"`
	Region             types.String        `tfsdk:"region"`
	NetworkAreaId      types.String        `tfsdk:"network_area_id"`
	NetworkAreaRouteId types.String        `tfsdk:"network_area_route_id"`
	NextHop            *NexthopModelV1     `tfsdk:"next_hop"`
	Destination        *DestinationModelV1 `tfsdk:"destination"`
	Labels             types.Map           `tfsdk:"labels"`
}

// ModelV0 is the old model (only needed for state upgrade)
type ModelV0 struct {
	Id                 types.String `tfsdk:"id"`
	OrganizationId     types.String `tfsdk:"organization_id"`
	NetworkAreaId      types.String `tfsdk:"network_area_id"`
	NetworkAreaRouteId types.String `tfsdk:"network_area_route_id"`
	NextHop            types.String `tfsdk:"next_hop"`
	Prefix             types.String `tfsdk:"prefix"`
	Labels             types.Map    `tfsdk:"labels"`
}

// DestinationModelV1 maps the route destination data
type DestinationModelV1 struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

// NexthopModelV1 maps the route nexthop data
type NexthopModelV1 struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

// NewNetworkAreaRouteResource is a helper function to simplify the provider implementation.
func NewNetworkAreaRouteResource() resource.Resource {
	return &networkAreaRouteResource{}
}

// networkResource is the resource implementation.
type networkAreaRouteResource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *networkAreaRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_area_route"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *networkAreaRouteResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel ModelV1
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel ModelV1
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

// Configure adds the provider configured client to the resource.
func (r *networkAreaRouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the resource.
func (r *networkAreaRouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Network area route resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Version:             1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`organization_id`,`network_area_id`,`region`,`network_area_route_id`\".",
				Computed:    true,
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
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
			"next_hop": schema.SingleNestedAttribute{
				Description: "Next hop destination.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: fmt.Sprintf("Type of the next hop. %s %s", utils.FormatPossibleValues("blackhole", "internet", "ipv4", "ipv6"), "Only `ipv4` supported currently."),
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"value": schema.StringAttribute{
						Description: "Either IPv4 or IPv6 (not set for blackhole and internet). Only IPv4 supported currently.",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.IP(false),
						},
					},
				},
			},
			"destination": schema.SingleNestedAttribute{
				Description: "Destination of the route.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: fmt.Sprintf("CIDRV type. %s %s", utils.FormatPossibleValues("cidrv4", "cidrv6"), "Only `cidrv4` is supported currently."),
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
						Validators: []validator.String{
							validate.CIDR(),
						},
					},
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

func (r *networkAreaRouteResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			// This handles moving from version 0 to 1
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"organization_id": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							validate.UUID(),
							validate.NoSeparator(),
						},
					},
					"network_area_id": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							validate.UUID(),
							validate.NoSeparator(),
						},
					},
					"network_area_route_id": schema.StringAttribute{
						Computed: true,
						Validators: []validator.String{
							validate.UUID(),
							validate.NoSeparator(),
						},
					},
					"next_hop": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							validate.IP(false),
						},
					},
					"prefix": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							validate.CIDR(),
						},
					},
					"labels": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorStateData ModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				nexthopValue := priorStateData.NextHop.ValueString()
				prefixValue := priorStateData.Prefix.ValueString()

				newStateData := ModelV1{
					Id:                 priorStateData.Id,
					OrganizationId:     priorStateData.OrganizationId,
					NetworkAreaId:      priorStateData.NetworkAreaId,
					NetworkAreaRouteId: priorStateData.NetworkAreaRouteId,
					Labels:             priorStateData.Labels,

					NextHop: &NexthopModelV1{
						Type:  types.StringValue("ipv4"),
						Value: types.StringValue(nexthopValue),
					},
					Destination: &DestinationModelV1{
						Type:  types.StringValue("cidrv4"),
						Value: types.StringValue(prefixValue),
					},
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, newStateData)...)
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *networkAreaRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model ModelV1
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	organizationId := model.OrganizationId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	networkAreaId := model.NetworkAreaId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area route", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network area route
	routes, err := r.client.CreateNetworkAreaRoute(ctx, organizationId, networkAreaId, region).CreateNetworkAreaRoutePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if routes.Items == nil || len(*routes.Items) == 0 {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area route.", "Empty response from API")
		return
	}

	if len(*routes.Items) != 1 {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area route.", "New static route not found or more than 1 route found in API response.")
		return
	}

	// Gets the route ID from the first element, routes.Items[0]
	routeItems := *routes.Items
	route := routeItems[0]
	routeId := *route.Id

	ctx = tflog.SetField(ctx, "network_area_route_id", routeId)

	// Map response body to schema
	err = mapFields(ctx, &route, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area route.", fmt.Sprintf("Processing API payload: %v", err))
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
	var model ModelV1
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	networkAreaRouteId := model.NetworkAreaRouteId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_route_id", networkAreaRouteId)

	networkAreaRouteResp, err := r.client.GetNetworkAreaRoute(ctx, organizationId, networkAreaId, region, networkAreaRouteId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area route.", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, networkAreaRouteResp, &model, region)
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
	var model ModelV1
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	networkAreaRouteId := model.NetworkAreaRouteId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_route_id", networkAreaRouteId)

	// Delete existing network
	err := r.client.DeleteNetworkAreaRoute(ctx, organizationId, networkAreaId, region, networkAreaRouteId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Network area route deleted")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkAreaRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model ModelV1
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	networkAreaRouteId := model.NetworkAreaRouteId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_route_id", networkAreaRouteId)

	// Retrieve values from state
	var stateModel ModelV1
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area route", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing network area route
	networkAreaRouteResp, err := r.client.UpdateNetworkAreaRoute(ctx, organizationId, networkAreaId, region, networkAreaRouteId).UpdateNetworkAreaRoutePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, networkAreaRouteResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area route updated")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: organization_id,network_aread_id,network_area_route_id
func (r *networkAreaRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network area route",
			fmt.Sprintf("Expected import identifier with format: [organization_id],[network_area_id],[region],[network_area_route_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"organization_id":       idParts[0],
		"network_area_id":       idParts[1],
		"region":                idParts[2],
		"network_area_route_id": idParts[3],
	})

	tflog.Info(ctx, "Network area route state imported")
}

func mapFields(ctx context.Context, networkAreaRoute *iaas.Route, model *ModelV1, region string) error {
	if networkAreaRoute == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkAreaRouteId string
	if model.NetworkAreaRouteId.ValueString() != "" {
		networkAreaRouteId = model.NetworkAreaRouteId.ValueString()
	} else if networkAreaRoute.Id != nil {
		networkAreaRouteId = *networkAreaRoute.Id
	} else {
		return fmt.Errorf("network area route id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.OrganizationId.ValueString(), model.NetworkAreaId.ValueString(), region, networkAreaRouteId)
	model.Region = types.StringValue(region)

	labels, err := iaasUtils.MapLabels(ctx, networkAreaRoute.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.NetworkAreaRouteId = types.StringValue(networkAreaRouteId)
	model.Labels = labels

	model.NextHop, err = mapRouteNextHop(networkAreaRoute)
	if err != nil {
		return err
	}

	model.Destination, err = mapRouteDestination(networkAreaRoute)
	if err != nil {
		return err
	}

	return nil
}

func toCreatePayload(ctx context.Context, model *ModelV1) (*iaas.CreateNetworkAreaRoutePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	nextHopPayload, err := toNextHopPayload(model)
	if err != nil {
		return nil, err
	}

	destinationPayload, err := toDestinationPayload(model)
	if err != nil {
		return nil, err
	}

	return &iaas.CreateNetworkAreaRoutePayload{
		Items: &[]iaas.Route{
			{
				Destination: destinationPayload,
				Labels:      &labels,
				Nexthop:     nextHopPayload,
			},
		},
	}, nil
}

func toUpdatePayload(ctx context.Context, model *ModelV1, currentLabels types.Map) (*iaas.UpdateNetworkAreaRoutePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.UpdateNetworkAreaRoutePayload{
		Labels: &labels,
	}, nil
}

func toNextHopPayload(model *ModelV1) (*iaas.RouteNexthop, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	} else if model.NextHop == nil {
		return nil, fmt.Errorf("nexthop is nil in model")
	}

	switch model.NextHop.Type.ValueString() {
	case "blackhole":
		return sdkUtils.Ptr(iaas.NexthopBlackholeAsRouteNexthop(iaas.NewNexthopBlackhole("blackhole"))), nil
	case "internet":
		return sdkUtils.Ptr(iaas.NexthopInternetAsRouteNexthop(iaas.NewNexthopInternet("internet"))), nil
	case "ipv4":
		return sdkUtils.Ptr(iaas.NexthopIPv4AsRouteNexthop(iaas.NewNexthopIPv4("ipv4", model.NextHop.Value.ValueString()))), nil
	case "ipv6":
		return sdkUtils.Ptr(iaas.NexthopIPv6AsRouteNexthop(iaas.NewNexthopIPv6("ipv6", model.NextHop.Value.ValueString()))), nil
	}
	return nil, fmt.Errorf("unknown nexthop type: %s", model.NextHop.Type.ValueString())
}

func toDestinationPayload(model *ModelV1) (*iaas.RouteDestination, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	} else if model.Destination == nil {
		return nil, fmt.Errorf("destination is nil in model")
	}

	switch model.Destination.Type.ValueString() {
	case "cidrv4":
		return sdkUtils.Ptr(iaas.DestinationCIDRv4AsRouteDestination(iaas.NewDestinationCIDRv4("cidrv4", model.Destination.Value.ValueString()))), nil
	case "cidrv6":
		return sdkUtils.Ptr(iaas.DestinationCIDRv6AsRouteDestination(iaas.NewDestinationCIDRv6("cidrv6", model.Destination.Value.ValueString()))), nil
	}
	return nil, fmt.Errorf("unknown destination type: %s", model.Destination.Type.ValueString())
}

func mapRouteNextHop(routeResp *iaas.Route) (*NexthopModelV1, error) {
	if routeResp.Nexthop == nil {
		return &NexthopModelV1{
			Type:  types.StringNull(),
			Value: types.StringNull(),
		}, nil
	}

	switch i := routeResp.Nexthop.GetActualInstance().(type) {
	case *iaas.NexthopIPv4:
		return &NexthopModelV1{
			Type:  types.StringPointerValue(i.Type),
			Value: types.StringPointerValue(i.Value),
		}, nil
	case *iaas.NexthopIPv6:
		return &NexthopModelV1{
			Type:  types.StringPointerValue(i.Type),
			Value: types.StringPointerValue(i.Value),
		}, nil
	case *iaas.NexthopBlackhole:
		return &NexthopModelV1{
			Type:  types.StringPointerValue(i.Type),
			Value: types.StringNull(),
		}, nil
	case *iaas.NexthopInternet:
		return &NexthopModelV1{
			Type:  types.StringPointerValue(i.Type),
			Value: types.StringNull(),
		}, nil
	default:
		return nil, fmt.Errorf("unexpected nexthop type: %T", i)
	}
}

func mapRouteDestination(routeResp *iaas.Route) (*DestinationModelV1, error) {
	if routeResp.Destination == nil {
		return &DestinationModelV1{
			Type:  types.StringNull(),
			Value: types.StringNull(),
		}, nil
	}

	switch i := routeResp.Destination.GetActualInstance().(type) {
	case *iaas.DestinationCIDRv4:
		return &DestinationModelV1{
			Type:  types.StringPointerValue(i.Type),
			Value: types.StringPointerValue(i.Value),
		}, nil
	case *iaas.DestinationCIDRv6:
		return &DestinationModelV1{
			Type:  types.StringPointerValue(i.Type),
			Value: types.StringPointerValue(i.Value),
		}, nil
	default:
		return nil, fmt.Errorf("unexpected Destionation type: %T", i)
	}
}
