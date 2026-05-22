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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api/wait"

	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/vpn/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                   = &gatewayResource{}
	_ resource.ResourceWithConfigure      = &gatewayResource{}
	_ resource.ResourceWithImportState    = &gatewayResource{}
	_ resource.ResourceWithModifyPlan     = &gatewayResource{}
	_ resource.ResourceWithValidateConfig = &gatewayResource{}

	gatewayStates      = sdkUtils.EnumSliceToStringSlice(vpn.AllowedGatewayStatusEnumValues)
	routingTypeOptions = sdkUtils.EnumSliceToStringSlice(vpn.AllowedRoutingTypeEnumValues)
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
	Id                types.String            `tfsdk:"id"` // needed by TF
	GatewayId         types.String            `tfsdk:"gateway_id"`
	ProjectId         types.String            `tfsdk:"project_id"`
	Region            types.String            `tfsdk:"region"`
	DisplayName       types.String            `tfsdk:"display_name"`
	PlanId            types.String            `tfsdk:"plan_id"`
	RoutingType       types.String            `tfsdk:"routing_type"`
	AvailabilityZones *AvailabilityZonesModel `tfsdk:"availability_zones"`
	Bgp               *BGPGatewayConfigModel  `tfsdk:"bgp"`
	Labels            types.Map               `tfsdk:"labels"`
	State             types.String            `tfsdk:"state"`
}

var schemaDescriptions = map[string]string{
	"id":                             "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`gateway_id`\".",
	"gateway_id":                     "The server-generated UUID of the VPN gateway.",
	"project_id":                     "STACKIT project ID associated with the VPN gateway.",
	"region":                         "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"display_name":                   "A user-friendly name for the VPN gateway.",
	"plan_id":                        "The service plan identifier (e.g. `p500`). For guidance on finding available plans, see [List available service plans](https://docs.stackit.cloud/products/network/connectivity-hybrid-multi-cloud/vpn/getting-started/gateway-create/#list-available-service-plans).",
	"routing_type":                   fmt.Sprintf("Routing architecture. %s", tfutils.FormatPossibleValues(routingTypeOptions...)),
	"availability_zones":             "Availability zones for the two tunnel endpoints.",
	"availability_zones_tunnel_1":    "Availability zone for tunnel 1.",
	"availability_zones_tunnel_2":    "Availability zone for tunnel 2.",
	"bgp":                            fmt.Sprintf("BGP configuration. Only applicable when routing_type is %s.", vpn.ROUTINGTYPE_BGP_ROUTE_BASED),
	"bgp_local_asn":                  "Local ASN for BGP (private ASN range, 64512-4294967294).",
	"bgp_override_advertised_routes": "List of IPv4 CIDRs to advertise via BGP. If omitted, SNA network ranges are advertised.",
	"labels":                         "Map of custom labels (key-value string pairs).",
	"state":                          fmt.Sprintf("The current lifecycle state of the gateway. %s", tfutils.FormatPossibleValues(gatewayStates...)),
}

type gatewayResource struct {
	apiClient    *vpn.APIClient
	providerData core.ProviderData
}

func NewGatewayResource() resource.Resource {
	return &gatewayResource{}
}

func (r *gatewayResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.apiClient = utils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN client configured")
}

func (r *gatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_gateway"
}

func (r *gatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
						Description: schemaDescriptions["availability_zones_tunnel_1"],
						Required:    true,
					},
					"tunnel2": schema.StringAttribute{
						Description: schemaDescriptions["availability_zones_tunnel_2"],
						Required:    true,
					},
				},
			},
			"bgp": schema.SingleNestedAttribute{
				Description: schemaDescriptions["bgp"],
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"local_asn": schema.Int64Attribute{
						Description: schemaDescriptions["bgp_local_asn"],
						Required:    true,
						Validators: []validator.Int64{
							int64validator.Between(64512, 4294967294),
						},
					},
					"override_advertised_routes": schema.ListAttribute{
						Description: schemaDescriptions["bgp_override_advertised_routes"],
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
				Validators:  validate.LabelValidators(),
			},
			"state": schema.StringAttribute{
				Description: schemaDescriptions["state"],
				Computed:    true,
			},
		},
	}
}

func (r *gatewayResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model Model

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if model.RoutingType.IsNull() || model.RoutingType.IsUnknown() {
		return
	}

	if model.RoutingType.ValueString() != string(vpn.ROUTINGTYPE_BGP_ROUTE_BASED) {
		return
	}

	var bgp types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("bgp"), &bgp)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if bgp.IsNull() || bgp.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("bgp"),
			"Missing required attribute",
			fmt.Sprintf("`bgp` must be set when `routing_type` is set to `%s`", vpn.ROUTINGTYPE_BGP_ROUTE_BASED),
		)
	}
}

func (r *gatewayResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *gatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *gatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN gateway", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.apiClient.DefaultAPI.CreateGateway(ctx, projectId, vpn.Region(region)).CreateGatewayPayload(*payload).Execute()
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

	waitResp, err := wait.CreateGatewayWaitHandler(ctx, r.apiClient.DefaultAPI, projectId, vpn.Region(region), gatewayId).WaitWithContext(ctx)
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

func (r *gatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	gatewayId := model.GatewayId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "region", region)

	gatewayResp, err := r.apiClient.DefaultAPI.GetGateway(ctx, projectId, vpn.Region(region), gatewayId).Execute()
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

func (r *gatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	gatewayId := model.GatewayId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN gateway", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	_, err = r.apiClient.DefaultAPI.UpdateGateway(ctx, projectId, vpn.Region(region), gatewayId).UpdateGatewayPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN gateway", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.UpdateGatewayWaitHandler(ctx, r.apiClient.DefaultAPI, projectId, vpn.Region(region), gatewayId).WaitWithContext(ctx)
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

func (r *gatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	gatewayId := model.GatewayId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "region", region)

	err := r.apiClient.DefaultAPI.DeleteGateway(ctx, projectId, vpn.Region(region), gatewayId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting VPN gateway", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteGatewayWaitHandler(ctx, r.apiClient.DefaultAPI, projectId, vpn.Region(region), gatewayId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting VPN gateway", fmt.Sprintf("Gateway deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "VPN gateway deleted")
}

func toCreatePayload(ctx context.Context, model *Model) (*vpn.CreateGatewayPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payload := &vpn.CreateGatewayPayload{
		DisplayName: model.DisplayName.ValueString(),
		PlanId:      model.PlanId.ValueString(),
		RoutingType: vpn.RoutingType(model.RoutingType.ValueString()),
		AvailabilityZones: vpn.CreateGatewayPayloadAvailabilityZones{
			Tunnel1: model.AvailabilityZones.Tunnel1.ValueString(),
			Tunnel2: model.AvailabilityZones.Tunnel2.ValueString(),
		},
	}

	if model.Bgp != nil {
		bgpConfig := &vpn.BGPGatewayConfig{}
		if !model.Bgp.LocalAsn.IsNull() {
			bgpConfig.LocalAsn = new(model.Bgp.LocalAsn.ValueInt64())
		}
		if !model.Bgp.OverrideAdvertisedRoutes.IsNull() {
			routes, err := tfutils.ListValueToStringSlice(model.Bgp.OverrideAdvertisedRoutes)
			if err != nil {
				return nil, err
			}
			if len(routes) > 0 {
				bgpConfig.OverrideAdvertisedRoutes = routes
			}
		}
		payload.Bgp = bgpConfig
	}

	labels, err := tfutils.LabelsToPayload(ctx, model.Labels)
	if err != nil {
		return nil, err
	}
	payload.Labels = &labels

	return payload, nil
}

func toUpdatePayload(ctx context.Context, model *Model) (*vpn.UpdateGatewayPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payload := &vpn.UpdateGatewayPayload{
		DisplayName: model.DisplayName.ValueString(),
		PlanId:      model.PlanId.ValueString(),
		AvailabilityZones: vpn.UpdateGatewayPayloadAvailabilityZones{
			Tunnel1: model.AvailabilityZones.Tunnel1.ValueString(),
			Tunnel2: model.AvailabilityZones.Tunnel2.ValueString(),
		},
		RoutingType: vpn.RoutingType(model.RoutingType.ValueString()),
	}

	if model.Bgp != nil {
		bgpConfig := &vpn.BGPGatewayConfig{}
		if !model.Bgp.LocalAsn.IsNull() {
			asn := model.Bgp.LocalAsn.ValueInt64()
			bgpConfig.LocalAsn = &asn
		}
		if !model.Bgp.OverrideAdvertisedRoutes.IsNull() {
			routes, err := tfutils.ListValueToStringSlice(model.Bgp.OverrideAdvertisedRoutes)
			if err != nil {
				return nil, err
			}
			if len(routes) > 0 {
				bgpConfig.OverrideAdvertisedRoutes = routes
			}
		}
		payload.Bgp = bgpConfig
	}

	labels, err := tfutils.LabelsToPayload(ctx, model.Labels)
	if err != nil {
		return nil, err
	}
	payload.Labels = &labels

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
	if model.GatewayId.ValueString() != "" {
		gatewayId = model.GatewayId.ValueString()
	} else if gateway.Id != nil {
		gatewayId = *gateway.Id
	} else {
		return fmt.Errorf("gateway id not present")
	}

	model.Id = tfutils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, gatewayId)
	model.GatewayId = types.StringValue(gatewayId)
	model.DisplayName = types.StringValue(gateway.DisplayName)
	model.PlanId = types.StringValue(gateway.PlanId)
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
		} else if model.Bgp != nil && !model.Bgp.OverrideAdvertisedRoutes.IsNull() {
			// preserve empty list from plan/state to avoid inconsistent state
			bgpModel.OverrideAdvertisedRoutes = types.ListValueMust(types.StringType, []attr.Value{})
		} else {
			bgpModel.OverrideAdvertisedRoutes = types.ListNull(types.StringType)
		}
		model.Bgp = bgpModel
	}

	labels, err := tfutils.MapLabels(ctx, gateway.Labels, model.Labels)
	if err != nil {
		return fmt.Errorf("mapping labels: %w", err)
	}
	model.Labels = labels

	if gateway.State != nil {
		model.State = types.StringValue(string(*gateway.State))
	}

	return nil
}
