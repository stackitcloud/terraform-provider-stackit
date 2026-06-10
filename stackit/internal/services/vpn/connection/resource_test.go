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
)

var (
	projectId = uuid.NewString()
	gatewayId = uuid.NewString()
	region    = "eu01"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *vpn.ConnectionResponse
		expected    Model
		isValid     bool
	}{
		{
			description: "basic_connection",
			input: &vpn.ConnectionResponse{
				Id:            new("connection-id"),
				DisplayName:   "test-connection",
				Enabled:       new(true),
				RemoteSubnets: []string{"10.0.0.0/16"},
				LocalSubnets:  []string{"192.168.0.0/24"},
				Tunnel1: vpn.TunnelConfiguration{
					RemoteAddress: "203.0.113.1",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             []string{"modp2048"},
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
						RekeyTime:            new(int32(14400)),
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             []string{"modp2048"},
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
						RekeyTime:            new(int32(3600)),
						StartAction:          new("start"),
						DpdAction:            new("restart"),
					},
				},
				Tunnel2: vpn.TunnelConfiguration{
					RemoteAddress: "203.0.113.2",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             []string{"modp2048"},
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
						RekeyTime:            new(int32(14400)),
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             []string{"modp2048"},
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
						RekeyTime:            new(int32(3600)),
						StartAction:          new("start"),
						DpdAction:            new("restart"),
					},
				},
			},
			expected: Model{
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
				Tunnel1: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("203.0.113.1"),
					Phase1: &Phase1Model{
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
					},
					Phase2: &Phase2Model{
						DhGroups: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("modp2048"),
						}),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("aes256"),
						}),
						IntegrityAlgorithms: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("sha2_256"),
						}),
						RekeyTime:   types.Int32Value(3600),
						StartAction: types.StringValue("start"),
						DpdAction:   types.StringValue("restart"),
					},
				},
				Tunnel2: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("203.0.113.2"),
					Phase1: &Phase1Model{
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
					},
					Phase2: &Phase2Model{
						DhGroups: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("modp2048"),
						}),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("aes256"),
						}),
						IntegrityAlgorithms: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("sha2_256"),
						}),
						RekeyTime:   types.Int32Value(3600),
						StartAction: types.StringValue("start"),
						DpdAction:   types.StringValue("restart"),
					},
				},
				Labels: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "connection_with_static_routes_and_bgp",
			input: &vpn.ConnectionResponse{
				Id:           new("conn-id-2"),
				DisplayName:  "bgp-connection",
				Enabled:      new(false),
				StaticRoutes: []string{"10.0.0.0/8"},
				Tunnel1: vpn.TunnelConfiguration{
					RemoteAddress: "203.0.113.10",
					Phase1: vpn.TunnelConfigurationPhase1{
						EncryptionAlgorithms: []string{"aes256gcm16"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						EncryptionAlgorithms: []string{"aes256gcm16"},
						IntegrityAlgorithms:  []string{"sha2_384"},
						DpdAction:            new("clear"),
						StartAction:          new("none"),
					},
					Peering: &vpn.PeeringConfig{
						LocalAddress:  new("169.254.0.1"),
						RemoteAddress: new("169.254.0.2"),
					},
					Bgp: &vpn.BGPTunnelConfig{
						RemoteAsn: 65000,
					},
				},
				Tunnel2: vpn.TunnelConfiguration{
					RemoteAddress: "203.0.113.11",
					Phase1: vpn.TunnelConfigurationPhase1{
						EncryptionAlgorithms: []string{"aes256gcm16"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						EncryptionAlgorithms: []string{"aes256gcm16"},
						IntegrityAlgorithms:  []string{"sha2_384"},
						DpdAction:            new("clear"),
						StartAction:          new("none"),
					},
				},
			},
			expected: Model{
				ID:           types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, region, gatewayId, "conn-id-2")),
				ConnectionID: types.StringValue("conn-id-2"),
				ProjectID:    types.StringValue(projectId),
				Region:       types.StringValue(region),
				GatewayID:    types.StringValue(gatewayId),
				DisplayName:  types.StringValue("bgp-connection"),
				Enabled:      types.BoolValue(false),
				RemoteSubnet: types.ListNull(types.StringType),
				LocalSubnet:  types.ListNull(types.StringType),
				StaticRoutes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("10.0.0.0/8"),
				}),
				Tunnel1: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("203.0.113.10"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256gcm16")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256gcm16")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringValue("none"),
						DpdAction:            types.StringValue("clear"),
					},
					Peering: &PeeringConfigModel{
						LocalAddress:  types.StringValue("169.254.0.1"),
						RemoteAddress: types.StringValue("169.254.0.2"),
					},
					Bgp: &BGPTunnelConfigModel{
						RemoteAsn: types.Int64Value(65000),
					},
				},
				Tunnel2: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("203.0.113.11"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256gcm16")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256gcm16")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringValue("none"),
						DpdAction:            types.StringValue("clear"),
					},
				},
				Labels: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "multiple_static_routes",
			input: &vpn.ConnectionResponse{
				Id:           new("conn-id-3"),
				DisplayName:  "static-routes-connection",
				StaticRoutes: []string{"10.0.0.0/8", "172.16.0.0/12"},
				Tunnel1: vpn.TunnelConfiguration{
					RemoteAddress: "1.2.3.4",
					Phase1: vpn.TunnelConfigurationPhase1{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
				},
				Tunnel2: vpn.TunnelConfiguration{
					RemoteAddress: "5.6.7.8",
					Phase1: vpn.TunnelConfigurationPhase1{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
				},
			},
			expected: Model{
				ID:           types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, region, gatewayId, "conn-id-3")),
				ConnectionID: types.StringValue("conn-id-3"),
				ProjectID:    types.StringValue(projectId),
				Region:       types.StringValue(region),
				GatewayID:    types.StringValue(gatewayId),
				DisplayName:  types.StringValue("static-routes-connection"),
				Enabled:      types.BoolValue(true),
				RemoteSubnet: types.ListNull(types.StringType),
				LocalSubnet:  types.ListNull(types.StringType),
				StaticRoutes: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("10.0.0.0/8"),
					types.StringValue("172.16.0.0/12"),
				}),
				Tunnel1: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("1.2.3.4"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
				Tunnel2: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("5.6.7.8"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
				Labels: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "empty_labels",
			input: &vpn.ConnectionResponse{
				Id:          new("conn-id-4"),
				DisplayName: "empty-labels-connection",
				Labels:      &map[string]string{},
				Tunnel1: vpn.TunnelConfiguration{
					RemoteAddress: "1.2.3.4",
					Phase1: vpn.TunnelConfigurationPhase1{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
				},
				Tunnel2: vpn.TunnelConfiguration{
					RemoteAddress: "5.6.7.8",
					Phase1: vpn.TunnelConfigurationPhase1{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
				},
			},
			expected: Model{
				ID:           types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, region, gatewayId, "conn-id-4")),
				ConnectionID: types.StringValue("conn-id-4"),
				ProjectID:    types.StringValue(projectId),
				Region:       types.StringValue(region),
				GatewayID:    types.StringValue(gatewayId),
				DisplayName:  types.StringValue("empty-labels-connection"),
				Enabled:      types.BoolValue(true),
				RemoteSubnet: types.ListNull(types.StringType),
				LocalSubnet:  types.ListNull(types.StringType),
				StaticRoutes: types.ListNull(types.StringType),
				Tunnel1: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("1.2.3.4"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
				Tunnel2: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("5.6.7.8"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
				Labels: types.MapNull(types.StringType),
			},
			isValid: true,
		},
		{
			description: "asymmetric_phase_fields",
			input: &vpn.ConnectionResponse{
				Id:          new("conn-id-5"),
				DisplayName: "asymmetric-connection",
				Tunnel1: vpn.TunnelConfiguration{
					RemoteAddress: "1.2.3.4",
					Phase1: vpn.TunnelConfigurationPhase1{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
						RekeyTime:            new(int32(7200)),
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
						RekeyTime:            new(int32(1800)),
						StartAction:          new("none"),
						DpdAction:            new("clear"),
					},
				},
				Tunnel2: vpn.TunnelConfiguration{
					RemoteAddress: "5.6.7.8",
					Phase1: vpn.TunnelConfigurationPhase1{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_256"},
					},
				},
			},
			expected: Model{
				ID:           types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, region, gatewayId, "conn-id-5")),
				ConnectionID: types.StringValue("conn-id-5"),
				ProjectID:    types.StringValue(projectId),
				Region:       types.StringValue(region),
				GatewayID:    types.StringValue(gatewayId),
				DisplayName:  types.StringValue("asymmetric-connection"),
				Enabled:      types.BoolValue(true),
				RemoteSubnet: types.ListNull(types.StringType),
				LocalSubnet:  types.ListNull(types.StringType),
				StaticRoutes: types.ListNull(types.StringType),
				Tunnel1: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("1.2.3.4"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Value(7200),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Value(1800),
						StartAction:          types.StringValue("none"),
						DpdAction:            types.StringValue("clear"),
					},
				},
				Tunnel2: &TunnelModel{
					PreSharedKeyWoVersion: types.Int64Value(1),
					RemoteAddress:         types.StringValue("5.6.7.8"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_256")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
				Labels: types.MapNull(types.StringType),
			},
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
			input: &Model{
				DisplayName:  types.StringValue("test-connection"),
				Enabled:      types.BoolValue(true),
				RemoteSubnet: types.ListNull(types.StringType),
				LocalSubnet:  types.ListNull(types.StringType),
				StaticRoutes: types.ListNull(types.StringType),
				Tunnel1: &TunnelModel{
					RemoteAddress:  types.StringValue("203.0.113.1"),
					PreSharedKeyWo: types.StringValue("secret123-at-least-20-chars"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
				Tunnel2: &TunnelModel{
					RemoteAddress:  types.StringValue("203.0.113.2"),
					PreSharedKeyWo: types.StringValue("secret456-at-least-20-chars"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
			},
			expected: &vpn.CreateGatewayConnectionPayload{
				DisplayName: "test-connection",
				Tunnel1: vpn.TunnelConfiguration{
					PreSharedKey:  new("secret123-at-least-20-chars"),
					RemoteAddress: "203.0.113.1",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
				},
				Tunnel2: vpn.TunnelConfiguration{
					PreSharedKey:  new("secret456-at-least-20-chars"),
					RemoteAddress: "203.0.113.2",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
				},
				Enabled: new(true),
			},
			isValid: true,
		},
		{
			description: "with_phase2_fields",
			input: &Model{
				DisplayName:  types.StringValue("test"),
				Enabled:      types.BoolValue(true),
				RemoteSubnet: types.ListNull(types.StringType),
				LocalSubnet:  types.ListNull(types.StringType),
				StaticRoutes: types.ListNull(types.StringType),
				Tunnel1: &TunnelModel{
					RemoteAddress:         types.StringValue("1.2.3.4"),
					PreSharedKeyWo:        types.StringValue("super-secret-key-at-least-20"),
					PreSharedKeyWoVersion: types.Int64Null(),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Value(7200),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Value(1800),
						StartAction:          types.StringValue("none"),
						DpdAction:            types.StringValue("clear"),
					},
				},
				Tunnel2: &TunnelModel{
					RemoteAddress:         types.StringValue("5.6.7.8"),
					PreSharedKeyWo:        types.StringValue("super-secret-key-at-least-20"),
					PreSharedKeyWoVersion: types.Int64Null(),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
			},
			expected: &vpn.CreateGatewayConnectionPayload{
				DisplayName: "test",
				Tunnel1: vpn.TunnelConfiguration{
					PreSharedKey:  new("super-secret-key-at-least-20"),
					RemoteAddress: "1.2.3.4",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
						RekeyTime:            new(int32(7200)),
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
						RekeyTime:            new(int32(1800)),
						StartAction:          new("none"),
						DpdAction:            new("clear"),
					},
				},
				Tunnel2: vpn.TunnelConfiguration{
					PreSharedKey:  new("super-secret-key-at-least-20"),
					RemoteAddress: "5.6.7.8",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
				},
				Enabled: new(true),
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
			input: &Model{
				DisplayName:  types.StringValue("updated-connection"),
				Enabled:      types.BoolValue(false),
				RemoteSubnet: types.ListNull(types.StringType),
				LocalSubnet:  types.ListNull(types.StringType),
				StaticRoutes: types.ListNull(types.StringType),
				Tunnel1: &TunnelModel{
					RemoteAddress:  types.StringValue("203.0.113.1"),
					PreSharedKeyWo: types.StringValue("secret123-at-least-20-chars"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
				Tunnel2: &TunnelModel{
					RemoteAddress:  types.StringValue("203.0.113.2"),
					PreSharedKeyWo: types.StringValue("secret456-at-least-20-chars"),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
			},
			expected: &vpn.UpdateGatewayConnectionPayload{
				DisplayName: "updated-connection",
				Tunnel1: vpn.TunnelConfiguration{
					PreSharedKey:  new("secret123-at-least-20-chars"),
					RemoteAddress: "203.0.113.1",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
				},
				Tunnel2: vpn.TunnelConfiguration{
					PreSharedKey:  new("secret456-at-least-20-chars"),
					RemoteAddress: "203.0.113.2",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
				},
				Enabled: new(false),
			},
			isValid: true,
		},
		{
			description: "update_without_psk",
			input: &Model{
				DisplayName:  types.StringValue("updated-connection"),
				Enabled:      types.BoolValue(false),
				RemoteSubnet: types.ListNull(types.StringType),
				LocalSubnet:  types.ListNull(types.StringType),
				StaticRoutes: types.ListNull(types.StringType),
				Tunnel1: &TunnelModel{
					RemoteAddress:  types.StringValue("203.0.113.1"),
					PreSharedKeyWo: types.StringNull(),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
				Tunnel2: &TunnelModel{
					RemoteAddress:  types.StringValue("203.0.113.2"),
					PreSharedKeyWo: types.StringNull(),
					Phase1: &Phase1Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
					},
					Phase2: &Phase2Model{
						DhGroups:             types.ListNull(types.StringType),
						EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
						IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
						RekeyTime:            types.Int32Null(),
						StartAction:          types.StringNull(),
						DpdAction:            types.StringNull(),
					},
				},
			},
			expected: &vpn.UpdateGatewayConnectionPayload{
				DisplayName: "updated-connection",
				Tunnel1: vpn.TunnelConfiguration{
					PreSharedKey:  nil,
					RemoteAddress: "203.0.113.1",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
				},
				Tunnel2: vpn.TunnelConfiguration{
					PreSharedKey:  nil,
					RemoteAddress: "203.0.113.2",
					Phase1: vpn.TunnelConfigurationPhase1{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
					Phase2: vpn.TunnelConfigurationPhase2{
						DhGroups:             nil,
						EncryptionAlgorithms: []string{"aes256"},
						IntegrityAlgorithms:  []string{"sha2_384"},
					},
				},
				Enabled: new(false),
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
			input: &TunnelModel{
				RemoteAddress:         types.StringValue("203.0.113.1"),
				PreSharedKeyWo:        types.StringValue("secret123-at-least-20-chars"),
				PreSharedKeyWoVersion: types.Int64Null(),
				Phase1: &Phase1Model{
					DhGroups:             types.ListNull(types.StringType),
					EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
					IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
					RekeyTime:            types.Int32Null(),
				},
				Phase2: &Phase2Model{
					DhGroups:             types.ListNull(types.StringType),
					EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
					IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
					RekeyTime:            types.Int32Null(),
					StartAction:          types.StringNull(),
					DpdAction:            types.StringNull(),
				},
			},
			isValid: true,
		},
		{
			description: "tunnel_with_bgp",
			input: &TunnelModel{
				RemoteAddress:         types.StringValue("203.0.113.1"),
				PreSharedKeyWo:        types.StringValue("secret123-at-least-20-chars"),
				PreSharedKeyWoVersion: types.Int64Null(),
				Phase1: &Phase1Model{
					DhGroups:             types.ListNull(types.StringType),
					EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
					IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
					RekeyTime:            types.Int32Null(),
				},
				Phase2: &Phase2Model{
					DhGroups:             types.ListNull(types.StringType),
					EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
					IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
					RekeyTime:            types.Int32Null(),
					StartAction:          types.StringNull(),
					DpdAction:            types.StringNull(),
				},
				Bgp: &BGPTunnelConfigModel{
					RemoteAsn: types.Int64Value(65000),
				},
			},
			isValid: true,
		},
		{
			description: "tunnel_without_psk",
			input: &TunnelModel{
				RemoteAddress:         types.StringValue("203.0.113.1"),
				PreSharedKeyWo:        types.StringNull(),
				PreSharedKeyWoVersion: types.Int64Null(),
				Phase1: &Phase1Model{
					DhGroups:             types.ListNull(types.StringType),
					EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
					IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
					RekeyTime:            types.Int32Null(),
				},
				Phase2: &Phase2Model{
					DhGroups:             types.ListNull(types.StringType),
					EncryptionAlgorithms: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aes256")}),
					IntegrityAlgorithms:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("sha2_384")}),
					RekeyTime:            types.Int32Null(),
					StartAction:          types.StringNull(),
					DpdAction:            types.StringNull(),
				},
			},
			isValid: true,
		},
		{
			description: "nil_tunnel",
			input:       nil,
			isValid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			config, err := toTunnelConfiguration(tt.input)

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
			if !tt.input.PreSharedKeyWo.IsNull() && !tt.input.PreSharedKeyWo.IsUnknown() {
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
