package gateway_status

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
)

const (
	testRegion                   = "eu01"
	testDisplayName              = "Gateway"
	testTunnel1InternalNextHopIP = "123.45.67.89"
	testTunnel1PublicIP          = "98.76.54.32"
	testTunnel2InternalNextHopIP = "123.45.67.89"
	testTunnel2PublicIP          = "98.76.54.32"
)

var (
	testProjectId = uuid.NewString()
	testGatewayId = uuid.NewString()
	testId        = testProjectId + "," + testRegion + "," + testGatewayId
)

func fixtureInput(mods ...func(m *vpn.GatewayStatusResponse)) *vpn.GatewayStatusResponse {
	resp := &vpn.GatewayStatusResponse{
		Id: new(testGatewayId),
		Connections: []vpn.ConnectionStatusResponse{
			vpn.ConnectionStatusResponse{
				DisplayName: new("Conn1"),
				Enabled:     new(true),
				Id:          new("foo"),
			},
			vpn.ConnectionStatusResponse{
				DisplayName: new("Conn2"),
				Enabled:     new(false),
				Id:          new("bar"),
			},
		},
		DisplayName: new(testDisplayName),
		Tunnels: []vpn.VPNTunnels{
			{
				InternalNextHopIP: new(testTunnel1InternalNextHopIP),
				Name:              vpn.VPNTUNNELSNAME_TUNNEL1.Ptr(),
				PublicIP:          new(testTunnel1PublicIP),
			},
			{
				InternalNextHopIP: new(testTunnel2InternalNextHopIP),
				Name:              vpn.VPNTUNNELSNAME_TUNNEL2.Ptr(),
				PublicIP:          new(testTunnel2PublicIP),
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
		ProjectId: types.StringValue(testProjectId),
		Region:    types.StringValue(testRegion),
		Id:        types.StringValue(testId),
		GatewayId: types.StringValue(testGatewayId),
		Connections: types.ListValueMust(types.ObjectType{AttrTypes: connectionType}, []attr.Value{
			types.ObjectValueMust(connectionType, map[string]attr.Value{
				"display_name": types.StringValue("Conn1"),
				"enabled":      types.BoolValue(true),
				"id":           types.StringValue("foo"),
			}),
			types.ObjectValueMust(connectionType, map[string]attr.Value{
				"display_name": types.StringValue("Conn2"),
				"enabled":      types.BoolValue(false),
				"id":           types.StringValue("bar"),
			}),
		}),
		DisplayName: types.StringValue(testDisplayName),
		Tunnels: types.ListValueMust(types.ObjectType{AttrTypes: tunnelType}, []attr.Value{
			types.ObjectValueMust(tunnelType, map[string]attr.Value{
				"internal_next_hop_ip": types.StringValue(testTunnel1InternalNextHopIP),
				"name":                 types.StringValue(string(vpn.VPNTUNNELSNAME_TUNNEL1)),
				"public_ip":            types.StringValue(testTunnel1PublicIP),
			}),
			types.ObjectValueMust(tunnelType, map[string]attr.Value{
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
			name:   "default_values",
			region: "eu01",
			state: &Model{
				ProjectId: types.StringValue(testProjectId),
			},
			input:    fixtureInput(),
			expected: fixtureModel(),
			isValid:  true,
		},
		{
			name:   "no_input",
			region: "eu01",
			state: &Model{
				ProjectId: types.StringValue(testProjectId),
				GatewayId: types.StringValue(testGatewayId),
			},
			input:    nil,
			expected: nil,
			isValid:  false,
		},
		{
			name:     "no_model",
			region:   "eu01",
			state:    nil,
			input:    &vpn.GatewayStatusResponse{},
			expected: nil,
			isValid:  false,
		},
		{
			name:   "no_gateway_id",
			region: "eu01",
			state: &Model{
				ProjectId: types.StringValue(testProjectId),
			},
			input:    &vpn.GatewayStatusResponse{},
			expected: nil,
			isValid:  false,
		},
		{
			name:   "empty_input",
			region: "eu01",
			state: &Model{
				ProjectId: types.StringValue(testProjectId),
				GatewayId: types.StringValue(testGatewayId),
			},
			input: &vpn.GatewayStatusResponse{},
			expected: &Model{
				Id:          types.StringValue(testId),
				ProjectId:   types.StringValue(testProjectId),
				GatewayId:   types.StringValue(testGatewayId),
				Region:      types.StringValue(testRegion),
				Connections: types.ListValueMust(types.ObjectType{AttrTypes: connectionType}, []attr.Value{}),
				Tunnels:     types.ListValueMust(types.ObjectType{AttrTypes: tunnelType}, []attr.Value{}),
			},
			isValid: true,
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

func TestMapTunnels(t *testing.T) {
	tests := []struct {
		name     string
		input    []vpn.VPNTunnels
		expected *basetypes.ListValue
		isValid  bool
	}{
		{
			name: "default_values",
			input: []vpn.VPNTunnels{
				{
					InternalNextHopIP: new(testTunnel1InternalNextHopIP),
					Name:              vpn.VPNTUNNELSNAME_TUNNEL1.Ptr(),
					PublicIP:          new(testTunnel1PublicIP),
				},
				{
					InternalNextHopIP: new(testTunnel2InternalNextHopIP),
					Name:              vpn.VPNTUNNELSNAME_TUNNEL2.Ptr(),
					PublicIP:          new(testTunnel2PublicIP),
				},
			},
			expected: new(types.ListValueMust(types.ObjectType{AttrTypes: tunnelType}, []attr.Value{
				types.ObjectValueMust(tunnelType, map[string]attr.Value{
					"internal_next_hop_ip": types.StringValue(testTunnel1InternalNextHopIP),
					"name":                 types.StringValue(string(vpn.VPNTUNNELSNAME_TUNNEL1)),
					"public_ip":            types.StringValue(testTunnel1PublicIP),
				}),
				types.ObjectValueMust(tunnelType, map[string]attr.Value{
					"internal_next_hop_ip": types.StringValue(testTunnel2InternalNextHopIP),
					"name":                 types.StringValue(string(vpn.VPNTUNNELSNAME_TUNNEL2)),
					"public_ip":            types.StringValue(testTunnel2PublicIP),
				}),
			})),
			isValid: true,
		},
		{
			name:     "empty",
			input:    []vpn.VPNTunnels{},
			expected: new(types.ListValueMust(types.ObjectType{AttrTypes: tunnelType}, []attr.Value{})),
			isValid:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tfTunnels, err := mapTunnels(ctx, tt.input)
			if (err == nil) != tt.isValid {
				t.Errorf("unexpected error: %s", err)
			}
			if tt.isValid {
				if !reflect.DeepEqual(tfTunnels, tt.expected) {
					t.Errorf("ParseProviderData() got = %v, want %v", tfTunnels, tt.expected)
				}
			}
		})
	}
}
