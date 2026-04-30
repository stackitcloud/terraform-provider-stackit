package gateway

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1beta1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
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
	ID                types.String           `tfsdk:"id"`
	GatewayID         types.String           `tfsdk:"gateway_id"`
	ProjectID         types.String           `tfsdk:"project_id"`
	Region            types.String           `tfsdk:"region"`
	DisplayName       types.String           `tfsdk:"display_name"`
	PlanID            types.String           `tfsdk:"plan_id"`
	RoutingType       types.String           `tfsdk:"routing_type"`
	AvailabilityZones AvailabilityZonesModel `tfsdk:"availability_zones"`
	Bgp               *BGPGatewayConfigModel `tfsdk:"bgp"`
	Labels            types.Map              `tfsdk:"labels"`
	State             types.String           `tfsdk:"state"`
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

	// Availability zones
	model.AvailabilityZones = AvailabilityZonesModel{
		Tunnel1: types.StringValue(string(gateway.AvailabilityZones.Tunnel1)),
		Tunnel2: types.StringValue(string(gateway.AvailabilityZones.Tunnel2)),
	}

	// BGP configuration (optional)
	if gateway.Bgp != nil {
		bgpModel := &BGPGatewayConfigModel{}
		if gateway.Bgp.LocalAsn != nil {
			bgpModel.LocalAsn = types.Int64Value(int64(*gateway.Bgp.LocalAsn))
		} else {
			bgpModel.LocalAsn = types.Int64Null()
		}
		if gateway.Bgp.OverrideAdvertisedRoutes != nil && len(gateway.Bgp.OverrideAdvertisedRoutes) > 0 {
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

	// Labels (optional)
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

	// State
	if gateway.State != nil {
		model.State = types.StringValue(string(*gateway.State))
	}

	return nil
}
