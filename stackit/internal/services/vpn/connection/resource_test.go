package connection

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"

	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

var (
	projectId = uuid.NewString()
	gatewayId = uuid.NewString()
	region    = "eu01"
)

func fixtureTunnelResponse(mods ...func(m *vpn.TunnelConfiguration)) vpn.TunnelConfiguration {
	resp := vpn.TunnelConfiguration{
		RemoteAddress: "203.0.113.1",
		Phase1: vpn.TunnelConfigurationPhase1{
			DhGroups:             []vpn.PhaseDhGroupsInner{"modp2048"},
			EncryptionAlgorithms: []vpn.PhaseEncryptionAlgorithmsInner{"aes256"},
			IntegrityAlgorithms:  []vpn.PhaseIntegrityAlgorithmsInner{"sha2_256"},
			RekeyTime:            new(int32(14400)),
		},
		Phase2: vpn.TunnelConfigurationPhase2{
			DhGroups:             []vpn.PhaseDhGroupsInner{"modp2048"},
			EncryptionAlgorithms: []vpn.PhaseEncryptionAlgorithmsInner{"aes256"},
			IntegrityAlgorithms:  []vpn.PhaseIntegrityAlgorithmsInner{"sha2_256"},
			RekeyTime:            new(int32(3600)),
			StartAction:          vpn.TUNNELCONFIGURATIONPHASE2ALLOFSTARTACTION_START.Ptr(),
			DpdAction:            vpn.TUNNELCONFIGURATIONPHASE2ALLOFDPDACTION_RESTART.Ptr(),
		},
	}
	for _, mod := range mods {
		mod(&resp)
	}
	return resp
}

func fixtureConnectionResponse(mods ...func(m *vpn.ConnectionResponse)) *vpn.ConnectionResponse {
	resp := &vpn.ConnectionResponse{
		Id:            new("connection-id"),
		DisplayName:   "test-connection",
		Enabled:       new(true),
		RemoteSubnets: []string{"10.0.0.0/16"},
		LocalSubnets:  []string{"192.168.0.0/24"},
		Tunnel1:       fixtureTunnelResponse(),
		Tunnel2: fixtureTunnelResponse(func(m *vpn.TunnelConfiguration) {
			m.RemoteAddress = "203.0.113.2"
		}),
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func fixtureBasePhaseModel(mods ...func(m *BasePhaseModel)) BasePhaseModel {
	resp := BasePhaseModel{
		DhGroups: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("modp2048"),
		}),
		EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("aes256"),
		}),
		IntegrityAlgorithms: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("sha2_256"),
		}),
		RekeyTime: types.Int32Value(14400),
	}
	for _, mod := range mods {
		mod(&resp)
	}
	return resp
}

func fixtureTunnelModel(mods ...func(m *TunnelModel)) *TunnelModel {
	resp := &TunnelModel{
		PreSharedKeyWoVersion: types.Int64Value(1),
		RemoteAddress:         types.StringValue("203.0.113.1"),
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

func fixtureModel(mods ...func(m *Model)) Model {
	resp := Model{
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
		Tunnel1:      fixtureTunnelModel(),
		Tunnel2: fixtureTunnelModel(func(m *TunnelModel) {
			m.RemoteAddress = types.StringValue("203.0.113.2")
		}),
		Labels: types.MapNull(types.StringType),
	}
	for _, mod := range mods {
		mod(&resp)
	}
	return resp
}

func fixtureTunnelPayload(mods ...func(m *vpn.TunnelConfiguration)) vpn.TunnelConfiguration {
	resp := vpn.TunnelConfiguration{
		PreSharedKey:  new("secret123-at-least-20-chars"),
		RemoteAddress: "203.0.113.1",
		Phase1: vpn.TunnelConfigurationPhase1{
			DhGroups:             []vpn.PhaseDhGroupsInner{"modp2048"},
			EncryptionAlgorithms: []vpn.PhaseEncryptionAlgorithmsInner{"aes256"},
			IntegrityAlgorithms:  []vpn.PhaseIntegrityAlgorithmsInner{"sha2_256"},
			RekeyTime:            new(int32(14400)),
		},
		Phase2: vpn.TunnelConfigurationPhase2{
			DhGroups:             []vpn.PhaseDhGroupsInner{"modp2048"},
			EncryptionAlgorithms: []vpn.PhaseEncryptionAlgorithmsInner{"aes256"},
			IntegrityAlgorithms:  []vpn.PhaseIntegrityAlgorithmsInner{"sha2_256"},
			RekeyTime:            new(int32(3600)),
			StartAction:          vpn.TUNNELCONFIGURATIONPHASE2ALLOFSTARTACTION_START.Ptr(),
			DpdAction:            vpn.TUNNELCONFIGURATIONPHASE2ALLOFDPDACTION_RESTART.Ptr(),
		},
	}
	for _, mod := range mods {
		mod(&resp)
	}
	return resp
}

func fixtureCreatePayload(mods ...func(m *vpn.CreateGatewayConnectionPayload)) *vpn.CreateGatewayConnectionPayload {
	resp := &vpn.CreateGatewayConnectionPayload{
		DisplayName: "test-connection",
		RemoteSubnets: []string{
			"10.0.0.0/16",
		},
		LocalSubnets: []string{
			"192.168.0.0/24",
		},
		Tunnel1: fixtureTunnelPayload(),
		Tunnel2: fixtureTunnelPayload(func(m *vpn.TunnelConfiguration) {
			m.PreSharedKey = new("secret456-at-least-20-chars")
			m.RemoteAddress = "203.0.113.2"
		}),
		Enabled: new(true),
		Labels:  &map[string]string{},
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func fixtureUpdatePayload(mods ...func(m *vpn.UpdateGatewayConnectionPayload)) *vpn.UpdateGatewayConnectionPayload {
	resp := &vpn.UpdateGatewayConnectionPayload{
		DisplayName: "test-connection",
		RemoteSubnets: []string{
			"10.0.0.0/16",
		},
		LocalSubnets: []string{
			"192.168.0.0/24",
		},
		Tunnel1: fixtureTunnelPayload(),
		Tunnel2: fixtureTunnelPayload(func(m *vpn.TunnelConfiguration) {
			m.PreSharedKey = new("secret456-at-least-20-chars")
			m.RemoteAddress = "203.0.113.2"
		}),
		Enabled: new(true),
		Labels:  &map[string]string{},
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *vpn.ConnectionResponse
		expected    Model
		isValid     bool
	}{
		{
			description: "basic_connection",
			input:       fixtureConnectionResponse(),
			expected:    fixtureModel(),
			isValid:     true,
		},
		{
			description: "minimal_connection",
			input: &vpn.ConnectionResponse{
				Id:      new("connection-id"),
				Tunnel1: vpn.TunnelConfiguration{},
				Tunnel2: vpn.TunnelConfiguration{},
			},
			expected: Model{
				ID:           types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, region, gatewayId, "connection-id")),
				ConnectionID: types.StringValue("connection-id"),
				ProjectID:    types.StringValue(projectId),
				Region:       types.StringValue(region),
				GatewayID:    types.StringValue(gatewayId),
				DisplayName:  types.StringValue(""),
				Enabled:      types.BoolValue(true),
				RemoteSubnet: types.ListNull(types.StringType),
				LocalSubnet:  types.ListNull(types.StringType),
				StaticRoutes: types.ListNull(types.StringType),
				Tunnel1: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue(""),
					Phase1: &Phase1Model{
						BasePhaseModel: BasePhaseModel{
							DhGroups:             types.ListNull(types.StringType),
							EncryptionAlgorithms: types.ListNull(types.StringType),
							IntegrityAlgorithms:  types.ListNull(types.StringType),
						},
					},
					Phase2: &Phase2Model{
						BasePhaseModel: BasePhaseModel{
							DhGroups:             types.ListNull(types.StringType),
							EncryptionAlgorithms: types.ListNull(types.StringType),
							IntegrityAlgorithms:  types.ListNull(types.StringType),
						},
					},
				},
				Tunnel2: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue(""),
					Phase1: &Phase1Model{
						BasePhaseModel: BasePhaseModel{
							DhGroups:             types.ListNull(types.StringType),
							EncryptionAlgorithms: types.ListNull(types.StringType),
							IntegrityAlgorithms:  types.ListNull(types.StringType),
						},
					},
					Phase2: &Phase2Model{
						BasePhaseModel: BasePhaseModel{
							DhGroups:             types.ListNull(types.StringType),
							EncryptionAlgorithms: types.ListNull(types.StringType),
							IntegrityAlgorithms:  types.ListNull(types.StringType),
						},
					},
				},
				Labels: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "connection_with_static_routes_and_bgp",
			input: fixtureConnectionResponse(func(m *vpn.ConnectionResponse) {
				m.StaticRoutes = []string{"10.0.0.0/8"}
				m.Tunnel1.Bgp = &vpn.BGPTunnelConfig{
					RemoteAsn: 65000,
				}
			}),
			expected: fixtureModel(func(m *Model) {
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
			expected: fixtureModel(func(m *Model) {
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
			expected: fixtureModel(func(m *Model) {
				m.Labels = types.MapNull(types.StringType)
			}),
			isValid: true,
		},
		{
			description: "nil_response",
			input:       nil,
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "nil_connection_id",
			input: &vpn.ConnectionResponse{
				Id:          nil,
				DisplayName: "test-connection",
			},
			expected: Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectID: types.StringValue(projectId),
				Region:    types.StringValue(region),
				GatewayID: types.StringValue(gatewayId),
				Tunnel1: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
				},
				Tunnel2: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
				},
			}

			err := mapFields(context.Background(), tt.input, state, region)

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

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *vpn.CreateGatewayConnectionPayload
		isValid     bool
	}{
		{
			description: "basic_connection",
			input: new(fixtureModel(func(m *Model) {
				m.Tunnel1.PreSharedKeyWo = types.StringValue("secret123-at-least-20-chars")
				m.Tunnel2.PreSharedKeyWo = types.StringValue("secret456-at-least-20-chars")
			})),
			expected: fixtureCreatePayload(),
			isValid:  true,
		},
		{
			description: "minimal_create",
			input: &Model{
				Tunnel1: &TunnelModel{},
				Tunnel2: &TunnelModel{},
			},
			expected: &vpn.CreateGatewayConnectionPayload{
				Labels: &map[string]string{},
			},
			isValid: true,
		},
		{
			description: "nil_model",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toCreatePayload(context.Background(), tt.input)

			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.expected, payload)
				if diff != "" {
					t.Fatalf("Data does not match (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *vpn.UpdateGatewayConnectionPayload
		isValid     bool
	}{
		{
			description: "basic_update",
			input: new(fixtureModel(func(m *Model) {
				m.Tunnel1.PreSharedKeyWo = types.StringValue("secret123-at-least-20-chars")
				m.Tunnel2.PreSharedKeyWo = types.StringValue("secret456-at-least-20-chars")
			})),
			expected: fixtureUpdatePayload(),
			isValid:  true,
		},
		{
			description: "minimal_update",
			input: &Model{
				Tunnel1: &TunnelModel{},
				Tunnel2: &TunnelModel{},
			},
			expected: &vpn.UpdateGatewayConnectionPayload{
				Labels: &map[string]string{},
			},
			isValid: true,
		},
		{
			description: "nil_model",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toUpdatePayload(context.Background(), tt.input)

			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.expected, payload)
				if diff != "" {
					t.Fatalf("Data does not match (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToTunnelConfiguration(t *testing.T) {
	tests := []struct {
		description string
		input       *TunnelModel
		isValid     bool
	}{
		{
			description: "valid_tunnel",
			input:       fixtureTunnelModel(),
			isValid:     true,
		},
		{
			description: "tunnel_with_bgp",
			input: fixtureTunnelModel(func(m *TunnelModel) {
				m.Bgp = &BGPTunnelConfigModel{
					RemoteAsn: types.Int64Value(65000),
				}
			}),
			isValid: true,
		},
		{
			description: "empty_tunnel",
			input:       &TunnelModel{},
			isValid:     true,
		},
		{
			description: "nil_tunnel",
			input:       nil,
			isValid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			config, err := toTunnelPayload(tt.input)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !tt.isValid {
				return
			}

			if config.RemoteAddress != tt.input.RemoteAddress.ValueString() {
				t.Errorf("RemoteAddress mismatch: got %v, want %v", config.RemoteAddress, tt.input.RemoteAddress.ValueString())
			}
			if !tfutils.IsUndefined(tt.input.PreSharedKeyWo) {
				if config.PreSharedKey == nil || *config.PreSharedKey != tt.input.PreSharedKeyWo.ValueString() {
					t.Errorf("PreSharedKey mismatch")
				}
			} else if config.PreSharedKey != nil {
				t.Errorf("PreSharedKey should be omitted")
			}

			if tt.input.Bgp != nil {
				if config.Bgp == nil {
					t.Errorf("expected BGP config, got nil")
				} else if config.Bgp.RemoteAsn != tt.input.Bgp.RemoteAsn.ValueInt64() {
					t.Errorf("RemoteAsn mismatch: got %v, want %v", config.Bgp.RemoteAsn, tt.input.Bgp.RemoteAsn.ValueInt64())
				}
			}
		})
	}
}
