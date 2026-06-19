package connection

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
)

func fixtureDataSourceTunnelModel(mods ...func(m *DataSourceTunnelModel)) *DataSourceTunnelModel {
	resp := &DataSourceTunnelModel{
		RemoteAddress: types.StringValue("203.0.113.1"),
		Phase1: &Phase1Model{
			BasePhaseModel: fixtureBasePhaseModel(),
		},
		Phase2: &Phase2Model{
			BasePhaseModel: fixtureBasePhaseModel(func(m *BasePhaseModel) {
				m.RekeyTime = types.Int32Value(3600)
			}),
			StartAction: types.StringValue("start"),
			DpdAction:   types.StringValue("restart"),
		},
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func fixtureDataSourceModel(mods ...func(m *DataSourceModel)) DataSourceModel {
	resp := DataSourceModel{
		ID:           types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, region, gatewayId, "connection-id")),
		ConnectionID: types.StringValue("connection-id"),
		ProjectID:    types.StringValue(projectId),
		Region:       types.StringValue(region),
		GatewayID:    types.StringValue(gatewayId),
		DisplayName:  types.StringValue("test-connection"),
		Enabled:      types.BoolValue(true),
		RemoteSubnet: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("10.0.0.0/16"),
		}),
		LocalSubnet: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("192.168.0.0/24"),
		}),
		StaticRoutes: types.ListNull(types.StringType),
		Tunnel1:      fixtureDataSourceTunnelModel(),
		Tunnel2: fixtureDataSourceTunnelModel(func(m *DataSourceTunnelModel) {
			m.RemoteAddress = types.StringValue("203.0.113.2")
		}),
		Labels: types.MapNull(types.StringType),
	}
	for _, mod := range mods {
		mod(&resp)
	}
	return resp
}

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *vpn.ConnectionResponse
		expected    DataSourceModel
		isValid     bool
	}{
		{
			description: "basic_connection",
			input:       fixtureConnectionResponse(),
			expected:    fixtureDataSourceModel(),
			isValid:     true,
		},
		{
			description: "connection_with_static_routes_and_bgp",
			input: fixtureConnectionResponse(func(m *vpn.ConnectionResponse) {
				m.StaticRoutes = []string{"10.0.0.0/8"}
				m.Tunnel1.Bgp = &vpn.BGPTunnelConfig{
					RemoteAsn: 65000,
				}
			}),
			expected: fixtureDataSourceModel(func(m *DataSourceModel) {
				m.StaticRoutes = types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("10.0.0.0/8"),
				})
				m.Tunnel1.Bgp = &BGPTunnelConfigModel{
					RemoteAsn: types.Int64Value(65000),
				}
			}),
			isValid: true,
		},
		{
			description: "multiple_static_routes",
			input: fixtureConnectionResponse(func(m *vpn.ConnectionResponse) {
				m.StaticRoutes = []string{"10.0.0.0/8", "172.16.0.0/12"}
			}),
			expected: fixtureDataSourceModel(func(m *DataSourceModel) {
				m.StaticRoutes = types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("10.0.0.0/8"),
					types.StringValue("172.16.0.0/12"),
				})
			}),
			isValid: true,
		},
		{
			description: "empty_labels",
			input: fixtureConnectionResponse(func(m *vpn.ConnectionResponse) {
				m.Labels = &map[string]string{}
			}),
			expected: fixtureDataSourceModel(func(m *DataSourceModel) {
				m.Labels = types.MapNull(types.StringType)
			}),
			isValid: true,
		},
		{
			description: "nil_response",
			input:       nil,
			expected:    DataSourceModel{},
			isValid:     false,
		},
		{
			description: "nil_connection_id",
			input: &vpn.ConnectionResponse{
				Id:          nil,
				DisplayName: "test-connection",
			},
			expected: DataSourceModel{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &DataSourceModel{
				ProjectID: types.StringValue(projectId),
				Region:    types.StringValue(region),
				GatewayID: types.StringValue(gatewayId),
				Tunnel1:   &DataSourceTunnelModel{},
				Tunnel2:   &DataSourceTunnelModel{},
			}

			err := mapDataSourceFields(context.Background(), tt.input, state, region)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.isValid {
				if diff := cmp.Diff(&tt.expected, state); diff != "" {
					t.Fatalf("Data mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
