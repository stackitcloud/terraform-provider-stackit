package staticroute

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &staticRouteResource{}
	_ resource.ResourceWithConfigure   = &staticRouteResource{}
	_ resource.ResourceWithImportState = &staticRouteResource{}
	_ resource.ResourceWithModifyPlan  = &staticRouteResource{}
)

type Model struct {
	SharedModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type SharedModel struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	VpcId          types.String `tfsdk:"vpc_id"`
	RoutingTableId types.String `tfsdk:"routing_table_id"`
	RouteId        types.String `tfsdk:"route_id"`
	Region         types.String `tfsdk:"region"`
	Destination    types.Object `tfsdk:"destination"`
	Nexthop        types.Object `tfsdk:"nexthop"`
	Labels         types.Map    `tfsdk:"labels"`
}

type Destination struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

var destinationTypes = map[string]attr.Type{
	"type":  types.StringType,
	"value": types.StringType,
}

type Nexthop struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

var nexthopTypes = map[string]attr.Type{
	"type":  types.StringType,
	"value": types.StringType,
}

func NewStaticRouteResource() resource.Resource {
	return &staticRouteResource{}
}

type staticRouteResource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

func (r *staticRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_routing_table_static_route"
}

func (r *staticRouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &r.providerData, features.VpcExperiment, "stackit_vpc_routing_table_static_route", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := iaasAlphaUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "IaaS v2alpha client configured")
}

func (r *staticRouteResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "VPC Routing table static route resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.VpcExperiment, core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descId,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descProjectId,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: descVpcId,
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
				Description: descRoutingTableId,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"route_id": schema.StringAttribute{
				Description: descRouteId,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Description: descRegion,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"destination": schema.SingleNestedAttribute{
				Description: descDestination,
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: descDestinationType,
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"value": schema.StringAttribute{
						Description: descDestinationValue,
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
			"nexthop": schema.SingleNestedAttribute{
				Description: descNexthop,
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: descNexthopType,
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"value": schema.StringAttribute{
						Description: descNexthopValue,
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"labels": schema.MapAttribute{
				Description: descLabels,
				ElementType: types.StringType,
				Optional:    true,
				Validators:  validate.LabelValidators(),
			},
			"timeouts": timeouts.AttributesAll(ctx),
		},
	}
}

func (r *staticRouteResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // signature required by TF
	timeouts.Attributes(ctx, timeouts.Opts{Read: true})
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

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *staticRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := model.Timeouts.Create(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()

	payload, err := toCreatePayload(ctx, &model.SharedModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating static route", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	route, err := r.client.DefaultAPI.AddVPCStaticRoute(ctx, projectId, vpcId, region, routingTableId).AddVPCStaticRoutePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating static route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, route, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating static route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC routing table static route created")
}

func (r *staticRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()
	routeId := model.RouteId.ValueString()

	if routeId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "route_id", routeId)

	route, err := r.client.DefaultAPI.GetVPCStaticRoute(ctx, projectId, vpcId, region, routingTableId, routeId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading vpc static route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, route, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading static route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC static route read")
}

func (r *staticRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := model.Timeouts.Update(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()
	routeId := model.RouteId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "route_id", routeId)

	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := toUpdatePayload(ctx, &model.SharedModel, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating vpc static route", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	route, err := r.client.DefaultAPI.UpdateVPCStaticRoute(ctx, projectId, vpcId, region, routingTableId, routeId).UpdateVPCStaticRoutePayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating vpc static route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, route, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating static route", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC static route updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *staticRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := model.Timeouts.Delete(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()
	routeId := model.RouteId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "route_id", routeId)

	err := r.client.DefaultAPI.DeleteVPCStaticRoute(ctx, projectId, vpcId, region, routingTableId, routeId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting vpc static route", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "VPC static route deleted")
}

// ImportState imports a resource into the Terraform state on success.
func (r *staticRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 5 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" || idParts[4] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing vpc static route",
			fmt.Sprintf("Expected import identifier with format: [project_id],[vpc_id],[region],[routing_table_id],[route_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":       idParts[0],
		"vpc_id":           idParts[1],
		"region":           idParts[2],
		"routing_table_id": idParts[3],
		"route_id":         idParts[4],
	})

	tflog.Info(ctx, "VPC Routing Table static routes imported")
}

func mapFields(ctx context.Context, route *iaas.Route, model *SharedModel, region string) error {
	if route == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var routeId string
	if model.RouteId.ValueString() != "" {
		routeId = model.RouteId.ValueString()
	} else if route.Id != nil {
		routeId = *route.Id
	} else {
		return fmt.Errorf("route id not present")
	}

	labels, err := iaasUtils.MapLabels(ctx, route.Labels, model.Labels)
	if err != nil {
		return err
	}

	// destination
	model.Destination, err = mapRouteDestination(route)
	if err != nil {
		return fmt.Errorf("error mapping route destination: %w", err)
	}

	// nexthop
	model.Nexthop, err = mapRouteNextHop(route)
	if err != nil {
		return fmt.Errorf("error mapping route nexthop: %w", err)
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		model.VpcId.ValueString(),
		region,
		model.RoutingTableId.ValueString(),
		routeId,
	)
	model.Region = types.StringValue(region)
	model.RouteId = types.StringValue(routeId)
	model.Labels = labels
	return nil
}

func mapRouteNextHop(routeResp *iaas.Route) (types.Object, error) {
	nextHopMap := map[string]attr.Value{}
	switch i := routeResp.Nexthop.GetActualInstance().(type) {
	case *iaas.NexthopIPv4:
		nextHopMap["type"] = types.StringValue(i.Type)
		nextHopMap["value"] = types.StringValue(i.Value)
	case *iaas.NexthopIPv6:
		nextHopMap["type"] = types.StringValue(i.Type)
		nextHopMap["value"] = types.StringValue(i.Value)
	case *iaas.NexthopBlackhole:
		nextHopMap["type"] = types.StringValue(i.Type)
		nextHopMap["value"] = types.StringNull()
	case *iaas.NexthopInternet:
		nextHopMap["type"] = types.StringValue(i.Type)
		nextHopMap["value"] = types.StringNull()
	default:
		return types.ObjectNull(nexthopTypes), fmt.Errorf("unexpected Nexthop type: %T", i)
	}

	nextHopTF, diags := types.ObjectValue(nexthopTypes, nextHopMap)
	if diags.HasError() {
		return types.ObjectNull(nexthopTypes), core.DiagsToError(diags)
	}

	return nextHopTF, nil
}

func mapRouteDestination(routeResp *iaas.Route) (types.Object, error) {
	destinationMap := map[string]attr.Value{}
	switch i := routeResp.Destination.GetActualInstance().(type) {
	case *iaas.DestinationCIDRv4:
		destinationMap["type"] = types.StringValue(i.Type)
		destinationMap["value"] = types.StringValue(i.Value)
	case *iaas.DestinationCIDRv6:
		destinationMap["type"] = types.StringValue(i.Type)
		destinationMap["value"] = types.StringValue(i.Value)
	default:
		return types.ObjectNull(destinationTypes), fmt.Errorf("unexpected Destination type: %T", i)
	}

	destinationTF, diags := types.ObjectValue(destinationTypes, destinationMap)
	if diags.HasError() {
		return types.ObjectNull(destinationTypes), core.DiagsToError(diags)
	}

	return destinationTF, nil
}

func toCreatePayload(ctx context.Context, model *SharedModel) (*iaas.AddVPCStaticRoutePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting labels: %w", err)
	}

	destination, err := toRouteDestination(ctx, model.Destination)
	if err != nil {
		return nil, err
	}

	nexthop, err := toRouteNextHop(ctx, model.Nexthop)
	if err != nil {
		return nil, err
	}

	return &iaas.AddVPCStaticRoutePayload{
		Labels:      labels,
		Destination: destination,
		Nexthop:     nexthop,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *SharedModel, currentLabels types.Map) (iaas.UpdateVPCStaticRoutePayload, error) {
	var result iaas.UpdateVPCStaticRoutePayload
	if model == nil {
		return result, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return result, fmt.Errorf("converting labels: %w", err)
	}

	result.Labels = labels
	return result, nil
}

func toRouteDestination(ctx context.Context, destinationTF types.Object) (iaas.AddVPCStaticRoutePayloadDestination, error) {
	var result iaas.AddVPCStaticRoutePayloadDestination
	if utils.IsUndefined(destinationTF) {
		return result, nil
	}

	model := Destination{}
	diags := destinationTF.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return result, core.DiagsToError(diags)
	}

	switch model.Type.ValueString() {
	case "cidrv4":
		result.DestinationCIDRv4 = &iaas.DestinationCIDRv4{Type: model.Type.ValueString(), Value: model.Value.ValueString()}
	case "cidrv6":
		result.DestinationCIDRv6 = &iaas.DestinationCIDRv6{Type: model.Type.ValueString(), Value: model.Value.ValueString()}
	default:
		return result, fmt.Errorf("unsupported destination type: %s", model.Type.ValueString())
	}
	return result, nil
}

func toRouteNextHop(ctx context.Context, nextHopTF types.Object) (iaas.AddVPCStaticRoutePayloadNexthop, error) {
	var result iaas.AddVPCStaticRoutePayloadNexthop
	if utils.IsUndefined(nextHopTF) {
		return result, nil
	}

	model := Nexthop{}
	diags := nextHopTF.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return result, core.DiagsToError(diags)
	}

	switch model.Type.ValueString() {
	case "ipv4":
		result.NexthopIPv4 = &iaas.NexthopIPv4{Type: model.Type.ValueString(), Value: model.Value.ValueString()}
	case "ipv6":
		result.NexthopIPv6 = &iaas.NexthopIPv6{Type: model.Type.ValueString(), Value: model.Value.ValueString()}
	case "blackhole":
		result.NexthopBlackhole = &iaas.NexthopBlackhole{Type: model.Type.ValueString()}
	case "internet":
		result.NexthopInternet = &iaas.NexthopInternet{Type: model.Type.ValueString()}
	default:
		return result, fmt.Errorf("unsupported nexthop type: %s", model.Type.ValueString())
	}
	return result, nil
}
