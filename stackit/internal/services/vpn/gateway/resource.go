package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1beta1api"
	"github.com/stackitcloud/stackit-sdk-go/services/vpn/v1beta1api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/vpn/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &vpnGatewayResource{}
	_ resource.ResourceWithConfigure   = &vpnGatewayResource{}
	_ resource.ResourceWithImportState = &vpnGatewayResource{}
	_ resource.ResourceWithModifyPlan  = &vpnGatewayResource{}

	routingTypeOptions = []string{"POLICY_BASED", "ROUTE_BASED", "BGP_ROUTE_BASED"}
)

type AvailabilityZonesModel struct {
	Tunnel1 types.String `tfsdk:"tunnel1"`
	Tunnel2 types.String `tfsdk:"tunnel2"`
}

type BGPGatewayConfigModel struct {
	LocalAsn                 types.Int64 `tfsdk:"local_asn"`
	OverrideAdvertisedRoutes types.List  `tfsdk:"override_advertised_routes"`
}

type Model struct {
	ID                types.String            `tfsdk:"id"`
	GatewayID         types.String            `tfsdk:"gateway_id"`
	ProjectID         types.String            `tfsdk:"project_id"`
	Region            types.String            `tfsdk:"region"`
	DisplayName       types.String            `tfsdk:"display_name"`
	PlanID            types.String            `tfsdk:"plan_id"`
	RoutingType       types.String            `tfsdk:"routing_type"`
	AvailabilityZones *AvailabilityZonesModel `tfsdk:"availability_zones"`
	Bgp               *BGPGatewayConfigModel  `tfsdk:"bgp"`
	Labels            types.Map               `tfsdk:"labels"`
	State             types.String            `tfsdk:"state"`
}

var schemaDescriptions = map[string]string{
	"id":                 "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`gateway_id`\".",
	"gateway_id":         "The server-generated UUID of the VPN gateway.",
	"project_id":         "STACKIT project ID associated with the VPN gateway.",
	"region":             "STACKIT region (e.g. eu01).",
	"display_name":       "A user-friendly name for the VPN gateway.",
	"plan_id":            "The service plan identifier (e.g. p500).",
	"routing_type":       "Routing architecture: POLICY_BASED, ROUTE_BASED, or BGP_ROUTE_BASED.",
	"availability_zones": "Availability zones for the two tunnel endpoints.",
	"bgp":                "BGP configuration. Only applicable when routing_type is BGP_ROUTE_BASED.",
	"labels":             "Map of custom labels (key-value string pairs).",
	"state":              "The current lifecycle state of the gateway (PENDING, READY, ERROR, DELETING).",
}

type vpnGatewayResource struct {
	client       *vpn.APIClient
	providerData core.ProviderData
}

func NewVpnGatewayResource() resource.Resource {
	return &vpnGatewayResource{}
}

func (r *vpnGatewayResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	r.providerData = providerData
	tflog.Info(ctx, "VPN client configured")
}

func (r *vpnGatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_gateway"
}

func (r *vpnGatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("VPN Gateway resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway_id": schema.StringAttribute{
				Description: schemaDescriptions["gateway_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
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
				Description: schemaDescriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`),
						"must start and end with an alphanumeric character, may contain hyphens, and be 1-63 characters long",
					),
				},
			},
			"plan_id": schema.StringAttribute{
				Description: schemaDescriptions["plan_id"],
				Required:    true,
			},
			"routing_type": schema.StringAttribute{
				Description: schemaDescriptions["routing_type"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(routingTypeOptions...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"availability_zones": schema.SingleNestedAttribute{
				Description: schemaDescriptions["availability_zones"],
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"tunnel1": schema.StringAttribute{
						Description: "Availability zone for tunnel 1.",
						Required:    true,
					},
					"tunnel2": schema.StringAttribute{
						Description: "Availability zone for tunnel 2.",
						Required:    true,
					},
				},
			},
			"bgp": schema.SingleNestedAttribute{
				Description: schemaDescriptions["bgp"],
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"local_asn": schema.Int64Attribute{
						Description: "Local ASN for BGP (private ASN range, 64512-4294967294).",
						Optional:    true,
						Validators: []validator.Int64{
							int64validator.Between(64512, 4294967294),
						},
					},
					"override_advertised_routes": schema.ListAttribute{
						Description: "List of IPv4 CIDRs to advertise via BGP. If omitted, SNA network ranges are advertised.",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtMost(100),
							listvalidator.ValueStringsAre(validate.CIDR()),
						},
					},
				},
			},
			"labels": schema.MapAttribute{
				Description: schemaDescriptions["labels"],
				Optional:    true,
				ElementType: types.StringType,
			},
			"state": schema.StringAttribute{
				Description: schemaDescriptions["state"],
				Computed:    true,
			},
		},
	}
}

func (r *vpnGatewayResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tfutils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vpnGatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing VPN gateway",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[gateway_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"gateway_id": idParts[2],
	})
	tflog.Info(ctx, "VPN gateway state imported")
}

func (r *vpnGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN gateway", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.DefaultAPI.CreateVPNGateway(ctx, projectId, vpn.Region(region)).CreateVPNGatewayPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN gateway", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN gateway", "Got empty gateway id")
		return
	}
	gatewayId := *createResp.Id

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"gateway_id": gatewayId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := wait.CreateOrUpdateGatewayWaitHandler(ctx, r.client.DefaultAPI, projectId, vpn.Region(region), gatewayId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN gateway", fmt.Sprintf("Gateway creation waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN gateway", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN gateway created")
}

func (r *vpnGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	gatewayId := model.GatewayID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "region", region)

	gatewayResp, err := r.client.DefaultAPI.GetVPNGateway(ctx, projectId, vpn.Region(region), gatewayId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN gateway", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, gatewayResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN gateway", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN gateway read")
}

func (r *vpnGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	gatewayId := model.GatewayID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN gateway", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	_, err = r.client.DefaultAPI.UpdateVPNGateway(ctx, projectId, vpn.Region(region), gatewayId).UpdateVPNGatewayPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN gateway", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.CreateOrUpdateGatewayWaitHandler(ctx, r.client.DefaultAPI, projectId, vpn.Region(region), gatewayId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN gateway", fmt.Sprintf("Gateway update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN gateway", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN gateway updated")
}

func (r *vpnGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	gatewayId := model.GatewayID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "region", region)

	err := r.client.DefaultAPI.DeleteVPNGateway(ctx, projectId, vpn.Region(region), gatewayId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting VPN gateway", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteGatewayWaitHandler(ctx, r.client.DefaultAPI, projectId, vpn.Region(region), gatewayId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting VPN gateway", fmt.Sprintf("Gateway deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "VPN gateway deleted")
}

func toCreatePayload(ctx context.Context, model *Model) (*vpn.CreateVPNGatewayPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if model.AvailabilityZones == nil {
		return nil, fmt.Errorf("availability_zones is required")
	}

	azTunnel1 := model.AvailabilityZones.Tunnel1.ValueString()
	azTunnel2 := model.AvailabilityZones.Tunnel2.ValueString()

	payload := &vpn.CreateVPNGatewayPayload{
		DisplayName: model.DisplayName.ValueString(),
		PlanId:      model.PlanID.ValueString(),
		RoutingType: vpn.RoutingType(model.RoutingType.ValueString()),
		AvailabilityZones: vpn.CreateVPNGatewayPayloadAvailabilityZones{
			Tunnel1: azTunnel1,
			Tunnel2: azTunnel2,
		},
	}

	if model.Bgp != nil {
		bgpConfig := &vpn.BGPGatewayConfig{}
		if !model.Bgp.LocalAsn.IsNull() && !model.Bgp.LocalAsn.IsUnknown() {
			asn := int32(model.Bgp.LocalAsn.ValueInt64())
			bgpConfig.LocalAsn = &asn
		}
		if !model.Bgp.OverrideAdvertisedRoutes.IsNull() && !model.Bgp.OverrideAdvertisedRoutes.IsUnknown() {
			routes := toStringSlice(ctx, model.Bgp.OverrideAdvertisedRoutes)
			if len(routes) > 0 {
				bgpConfig.OverrideAdvertisedRoutes = routes
			}
		}
		payload.Bgp = bgpConfig
	}

	if !model.Labels.IsNull() && !model.Labels.IsUnknown() {
		labels := make(map[string]string)
		diags := model.Labels.ElementsAs(ctx, &labels, false)
		if diags.HasError() {
			return nil, fmt.Errorf("converting labels: %w", core.DiagsToError(diags))
		}
		if len(labels) > 0 {
			payload.Labels = &labels
		}
	}

	return payload, nil
}

func toUpdatePayload(ctx context.Context, model *Model) (*vpn.UpdateVPNGatewayPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if model.AvailabilityZones == nil {
		return nil, fmt.Errorf("availability_zones is required")
	}

	azTunnel1 := model.AvailabilityZones.Tunnel1.ValueString()
	azTunnel2 := model.AvailabilityZones.Tunnel2.ValueString()

	payload := &vpn.UpdateVPNGatewayPayload{
		DisplayName: model.DisplayName.ValueString(),
		PlanId:      model.PlanID.ValueString(),
		AvailabilityZones: vpn.UpdateVPNGatewayPayloadAvailabilityZones{
			Tunnel1: azTunnel1,
			Tunnel2: azTunnel2,
		},
		RoutingType: vpn.RoutingType(model.RoutingType.ValueString()),
	}

	if model.Bgp != nil {
		bgpConfig := &vpn.BGPGatewayConfig{}
		if !model.Bgp.LocalAsn.IsNull() && !model.Bgp.LocalAsn.IsUnknown() {
			asn := int32(model.Bgp.LocalAsn.ValueInt64())
			bgpConfig.LocalAsn = &asn
		}
		if !model.Bgp.OverrideAdvertisedRoutes.IsNull() && !model.Bgp.OverrideAdvertisedRoutes.IsUnknown() {
			routes := toStringSlice(ctx, model.Bgp.OverrideAdvertisedRoutes)
			if len(routes) > 0 {
				bgpConfig.OverrideAdvertisedRoutes = routes
			}
		}
		payload.Bgp = bgpConfig
	}

	if !model.Labels.IsNull() && !model.Labels.IsUnknown() {
		labels := make(map[string]string)
		diags := model.Labels.ElementsAs(ctx, &labels, false)
		if diags.HasError() {
			return nil, fmt.Errorf("converting labels: %w", core.DiagsToError(diags))
		}
		payload.Labels = &labels
	}

	return payload, nil
}

func mapFields(ctx context.Context, gateway *vpn.GatewayResponse, model *Model, region string) error {
	if gateway == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var gatewayId string
	if model.GatewayID.ValueString() != "" {
		gatewayId = model.GatewayID.ValueString()
	} else if gateway.Id != nil {
		gatewayId = *gateway.Id
	} else {
		return fmt.Errorf("gateway id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), region, gatewayId)
	model.GatewayID = types.StringValue(gatewayId)
	model.DisplayName = types.StringValue(gateway.DisplayName)
	model.PlanID = types.StringValue(gateway.PlanId)
	model.RoutingType = types.StringValue(string(gateway.RoutingType))
	model.Region = types.StringValue(region)

	model.AvailabilityZones = &AvailabilityZonesModel{
		Tunnel1: types.StringValue(string(gateway.AvailabilityZones.Tunnel1)),
		Tunnel2: types.StringValue(string(gateway.AvailabilityZones.Tunnel2)),
	}

	if gateway.Bgp != nil {
		bgpModel := &BGPGatewayConfigModel{}
		if gateway.Bgp.LocalAsn != nil {
			bgpModel.LocalAsn = types.Int64Value(int64(*gateway.Bgp.LocalAsn))
		} else {
			bgpModel.LocalAsn = types.Int64Null()
		}
		if len(gateway.Bgp.OverrideAdvertisedRoutes) > 0 {
			routes := gateway.Bgp.OverrideAdvertisedRoutes
			listVal, diags := types.ListValueFrom(ctx, types.StringType, routes)
			if diags.HasError() {
				return fmt.Errorf("mapping BGP routes: %w", core.DiagsToError(diags))
			}
			bgpModel.OverrideAdvertisedRoutes = listVal
		} else {
			bgpModel.OverrideAdvertisedRoutes = types.ListNull(types.StringType)
		}
		model.Bgp = bgpModel
	}

	if gateway.Labels != nil && len(*gateway.Labels) > 0 {
		labelsMap := make(map[string]attr.Value)
		for k, v := range *gateway.Labels {
			labelsMap[k] = types.StringValue(v)
		}
		mapVal, diags := types.MapValue(types.StringType, labelsMap)
		if diags.HasError() {
			return fmt.Errorf("mapping labels: %w", core.DiagsToError(diags))
		}
		model.Labels = mapVal
	} else {
		model.Labels = types.MapNull(types.StringType)
	}

	if gateway.State != nil {
		model.State = types.StringValue(string(*gateway.State))
	}

	return nil
}

func toStringSlice(ctx context.Context, list types.List) []string {
	var result []string
	list.ElementsAs(ctx, &result, false)
	return result
}
