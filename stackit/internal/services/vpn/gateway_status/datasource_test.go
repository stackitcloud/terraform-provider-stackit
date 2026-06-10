package gateway_status

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
)

var (
	testProjectId                = uuid.NewString()
	testGatewayId                = uuid.NewString()
	testRegion                   = "eu01"
	testDisplayName              = "Gateway"
	testId                       = testProjectId + "," + testRegion + "," + testGatewayId
	testTunnel1InternalNextHopIP = "123.45.67.89"
	testTunnel1PublicIP          = "98.76.54.32"
	testTunnel2InternalNextHopIP = "123.45.67.89"
	testTunnel2PublicIP          = "98.76.54.32"
	testErrorMessage             = "foo bar"
)

func fixtureInput(mods ...func(m *vpn.GatewayStatusResponse)) *vpn.GatewayStatusResponse {
	resp := &vpn.GatewayStatusResponse{
		Id:            &testGatewayId,
		Connections:   []vpn.ConnectionStatusResponse{},
		DisplayName:   &testDisplayName,
		GatewayStatus: vpn.GATEWAYSTATUS_READY.Ptr(),
		ErrorMessage:  &testErrorMessage,
		Tunnels: []vpn.VPNTunnels{
			{
				InstanceState:     vpn.GATEWAYSTATUS_READY.Ptr(),
				InternalNextHopIP: &testTunnel1InternalNextHopIP,
				Name:              vpn.VPNTUNNELSNAME_TUNNEL1.Ptr(),
				PublicIP:          &testTunnel1PublicIP,
			},
			{
				InstanceState:     vpn.GATEWAYSTATUS_READY.Ptr(),
				InternalNextHopIP: &testTunnel2InternalNextHopIP,
				Name:              vpn.VPNTUNNELSNAME_TUNNEL2.Ptr(),
				PublicIP:          &testTunnel2PublicIP,
			},
		},
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func fixtureModel(mods ...func(m *Model)) *Model {
	resp := &Model{
		ProjectId:     types.StringValue(testProjectId),
		Region:        types.StringValue(testRegion),
		Id:            types.StringValue(testId),
		GatewayId:     types.StringValue(testGatewayId),
		DisplayName:   types.StringValue(testDisplayName),
		GatewayStatus: types.StringValue(string(vpn.GATEWAYSTATUS_READY)),
		ErrorMessage:  types.StringValue(testErrorMessage),
		Tunnels: types.ListValueMust(types.ObjectType{AttrTypes: tunnelsType}, []attr.Value{
			types.ObjectValueMust(tunnelsType, map[string]attr.Value{
				"instance_state":       types.StringValue(string(vpn.GATEWAYSTATUS_READY)),
				"internal_next_hop_ip": types.StringValue(testTunnel1InternalNextHopIP),
				"name":                 types.StringValue(string(vpn.VPNTUNNELSNAME_TUNNEL1)),
				"public_ip":            types.StringValue(testTunnel1PublicIP),
			}),
			types.ObjectValueMust(tunnelsType, map[string]attr.Value{
				"instance_state":       types.StringValue(string(vpn.GATEWAYSTATUS_READY)),
				"internal_next_hop_ip": types.StringValue(testTunnel2InternalNextHopIP),
				"name":                 types.StringValue(string(vpn.VPNTUNNELSNAME_TUNNEL2)),
				"public_ip":            types.StringValue(testTunnel2PublicIP),
			}),
		}),
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func TestMapDatasourceFields(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		state    *Model
		input    *vpn.GatewayStatusResponse
		expected *Model
		isValid  bool
	}{
		{
			"default_values",
			"eu01",
			&Model{
				ProjectId: types.StringValue(testProjectId),
			},
			fixtureInput(),
			fixtureModel(),
			true,
		},
		{
			"no_input",
			"eu01",
			&Model{
				ProjectId: types.StringValue(testProjectId),
				GatewayId: types.StringValue(testGatewayId),
			},
			nil,
			nil,
			false,
		},
		{
			"no_model",
			"eu01",
			nil,
			&vpn.GatewayStatusResponse{},
			nil,
			false,
		},
		{
			"no_gateway_id",
			"eu01",
			&Model{
				ProjectId: types.StringValue(testProjectId),
			},
			&vpn.GatewayStatusResponse{},
			nil,
			false,
		},
		{
			"empty_input",
			"eu01",
			&Model{
				ProjectId: types.StringValue(testProjectId),
				GatewayId: types.StringValue(testGatewayId),
			},
			&vpn.GatewayStatusResponse{},
			&Model{
				Id:        types.StringValue(testId),
				ProjectId: types.StringValue(testProjectId),
				GatewayId: types.StringValue(testGatewayId),
				Region:    types.StringValue(testRegion),
				Tunnels:   types.ListValueMust(types.ObjectType{AttrTypes: tunnelsType}, []attr.Value{}),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := mapFields(ctx, tt.input, tt.state, tt.region); (err == nil) != tt.isValid {
				t.Errorf("unexpected error: %s", err)
			}
			if tt.isValid {
				if diff := cmp.Diff(tt.state, tt.expected); diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
