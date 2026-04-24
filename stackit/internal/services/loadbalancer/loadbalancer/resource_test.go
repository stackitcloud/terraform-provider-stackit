package loadbalancer

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	legacyLoadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

const (
	testExternalAddress = "95.46.74.109"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *loadbalancer.CreateLoadBalancerPayload
		isValid     bool
	}{
		{
			"default_values_ok",
			&Model{},
			&loadbalancer.CreateLoadBalancerPayload{
				ExternalAddress: nil,
				Listeners:       nil,
				Name:            nil,
				Networks:        nil,
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: nil,
					},
					PrivateNetworkOnly: nil,
					Observability:      &loadbalancer.LoadbalancerOptionObservability{},
				},
				TargetPools: nil,
			},
			true,
		},
		{
			"simple_values_ok",
			&Model{
				ExternalAddress: types.StringValue("external_address"),
				Listeners: types.ListValueMust(types.ObjectType{AttrTypes: listenerTypes}, []attr.Value{
					types.ObjectValueMust(listenerTypes, map[string]attr.Value{
						"display_name": types.StringValue("display_name"),
						"port":         types.Int32Value(80),
						"protocol":     types.StringValue(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						"server_name_indicators": types.ListValueMust(types.ObjectType{AttrTypes: serverNameIndicatorTypes}, []attr.Value{
							types.ObjectValueMust(
								serverNameIndicatorTypes,
								map[string]attr.Value{
									"name": types.StringValue("domain.com"),
								},
							),
						},
						),
						"target_pool": types.StringValue("target_pool"),
						"tcp": types.ObjectValueMust(tcpTypes, map[string]attr.Value{
							"idle_timeout": types.StringValue("50s"),
						}),
						"udp": types.ObjectValueMust(udpTypes, map[string]attr.Value{
							"idle_timeout": types.StringValue("50s"),
						}),
					}),
				}),
				Name: types.StringValue("name"),
				Networks: types.ListValueMust(types.ObjectType{AttrTypes: networkTypes}, []attr.Value{
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
				}),
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"acl": types.SetValueMust(
							types.StringType,
							[]attr.Value{types.StringValue("cidr")}),
						"private_network_only": types.BoolValue(true),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"logs": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("logs-credentials_ref"),
								"push_url":        types.StringValue("logs-push_url"),
							}),
							"metrics": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("metrics-credentials_ref"),
								"push_url":        types.StringValue("metrics-push_url"),
							}),
						}),
					},
				),
				TargetPools: types.ListValueMust(types.ObjectType{AttrTypes: targetPoolTypes}, []attr.Value{
					types.ObjectValueMust(targetPoolTypes, map[string]attr.Value{
						"active_health_check": types.ObjectValueMust(activeHealthCheckTypes, map[string]attr.Value{
							"healthy_threshold":   types.Int32Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int32Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int32Value(80),
						"targets": types.ListValueMust(types.ObjectType{AttrTypes: targetTypes}, []attr.Value{
							types.ObjectValueMust(targetTypes, map[string]attr.Value{
								"display_name": types.StringValue("display_name"),
								"ip":           types.StringValue("ip"),
							}),
						}),
						"session_persistence": types.ObjectValueMust(sessionPersistenceTypes, map[string]attr.Value{
							"use_source_ip_address": types.BoolValue(true),
						}),
					}),
				}),
			},
			&loadbalancer.CreateLoadBalancerPayload{
				ExternalAddress: new("external_address"),
				Listeners: []loadbalancer.Listener{
					{
						DisplayName: new("display_name"),
						Port:        new(int32(80)),
						Protocol:    new(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						ServerNameIndicators: []loadbalancer.ServerNameIndicator{
							{
								Name: new("domain.com"),
							},
						},
						TargetPool: new("target_pool"),
						Tcp: new(loadbalancer.OptionsTCP{
							IdleTimeout: new("50s"),
						}),
						Udp: new(loadbalancer.OptionsUDP{
							IdleTimeout: new("50s"),
						}),
					},
				},
				Name: new("name"),
				Networks: []loadbalancer.Network{
					{
						NetworkId: new("network_id"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
					{
						NetworkId: new("network_id_2"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
				},
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: []string{"cidr"},
					},
					PrivateNetworkOnly: new(true),
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: new("logs-credentials_ref"),
							PushUrl:        new("logs-push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: new("metrics-credentials_ref"),
							PushUrl:        new("metrics-push_url"),
						},
					},
				},
				TargetPools: []loadbalancer.TargetPool{
					{
						ActiveHealthCheck: &loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   new(int32(1)),
							Interval:           new("2s"),
							IntervalJitter:     new("3s"),
							Timeout:            new("4s"),
							UnhealthyThreshold: new(int32(5)),
						},
						Name:       new("name"),
						TargetPort: new(int32(80)),
						Targets: []loadbalancer.Target{
							{
								DisplayName: new("display_name"),
								Ip:          new("ip"),
							},
						},
						SessionPersistence: &loadbalancer.SessionPersistence{
							UseSourceIpAddress: new(true),
						},
					},
				},
			},
			true,
		},
		{
			"service_plan_ok",
			&Model{
				PlanId:          types.StringValue("p10"),
				ExternalAddress: types.StringValue("external_address"),
				Listeners: types.ListValueMust(types.ObjectType{AttrTypes: listenerTypes}, []attr.Value{
					types.ObjectValueMust(listenerTypes, map[string]attr.Value{
						"display_name": types.StringValue("display_name"),
						"port":         types.Int32Value(80),
						"protocol":     types.StringValue(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						"server_name_indicators": types.ListValueMust(types.ObjectType{AttrTypes: serverNameIndicatorTypes}, []attr.Value{
							types.ObjectValueMust(
								serverNameIndicatorTypes,
								map[string]attr.Value{
									"name": types.StringValue("domain.com"),
								},
							),
						},
						),
						"target_pool": types.StringValue("target_pool"),
						"tcp":         types.ObjectNull(tcpTypes),
						"udp":         types.ObjectNull(udpTypes),
					}),
				}),
				Name: types.StringValue("name"),
				Networks: types.ListValueMust(types.ObjectType{AttrTypes: networkTypes}, []attr.Value{
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
				}),
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"acl": types.SetValueMust(
							types.StringType,
							[]attr.Value{types.StringValue("cidr")}),
						"private_network_only": types.BoolValue(true),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"logs": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("logs-credentials_ref"),
								"push_url":        types.StringValue("logs-push_url"),
							}),
							"metrics": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("metrics-credentials_ref"),
								"push_url":        types.StringValue("metrics-push_url"),
							}),
						}),
					},
				),
				TargetPools: types.ListValueMust(types.ObjectType{AttrTypes: targetPoolTypes}, []attr.Value{
					types.ObjectValueMust(targetPoolTypes, map[string]attr.Value{
						"active_health_check": types.ObjectValueMust(activeHealthCheckTypes, map[string]attr.Value{
							"healthy_threshold":   types.Int32Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int32Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int32Value(80),
						"targets": types.ListValueMust(types.ObjectType{AttrTypes: targetTypes}, []attr.Value{
							types.ObjectValueMust(targetTypes, map[string]attr.Value{
								"display_name": types.StringValue("display_name"),
								"ip":           types.StringValue("ip"),
							}),
						}),
						"session_persistence": types.ObjectValueMust(sessionPersistenceTypes, map[string]attr.Value{
							"use_source_ip_address": types.BoolValue(true),
						}),
					}),
				}),
			},
			&loadbalancer.CreateLoadBalancerPayload{
				PlanId:          new("p10"),
				ExternalAddress: new("external_address"),
				Listeners: []loadbalancer.Listener{
					{
						DisplayName: new("display_name"),
						Port:        new(int32(80)),
						Protocol:    new(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						ServerNameIndicators: []loadbalancer.ServerNameIndicator{
							{
								Name: new("domain.com"),
							},
						},
						TargetPool: new("target_pool"),
					},
				},
				Name: new("name"),
				Networks: []loadbalancer.Network{
					{
						NetworkId: new("network_id"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
					{
						NetworkId: new("network_id_2"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
				},
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: []string{"cidr"},
					},
					PrivateNetworkOnly: new(true),
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: new("logs-credentials_ref"),
							PushUrl:        new("logs-push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: new("metrics-credentials_ref"),
							PushUrl:        new("metrics-push_url"),
						},
					},
				},
				TargetPools: []loadbalancer.TargetPool{
					{
						ActiveHealthCheck: &loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   new(int32(1)),
							Interval:           new("2s"),
							IntervalJitter:     new("3s"),
							Timeout:            new("4s"),
							UnhealthyThreshold: new(int32(5)),
						},
						Name:       new("name"),
						TargetPort: new(int32(80)),
						Targets: []loadbalancer.Target{
							{
								DisplayName: new("display_name"),
								Ip:          new("ip"),
							},
						},
						SessionPersistence: &loadbalancer.SessionPersistence{
							UseSourceIpAddress: new(true),
						},
					},
				},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s,%s", "pid", testRegion, "name")
	tests := []struct {
		description             string
		input                   *loadbalancer.LoadBalancer
		modelPrivateNetworkOnly *bool
		region                  string
		expected                *Model
		isValid                 bool
	}{
		{
			"default_values_ok",
			&loadbalancer.LoadBalancer{
				ExternalAddress: nil,
				Listeners:       nil,
				Name:            new("name"),
				Networks:        nil,
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: nil,
					},
					PrivateNetworkOnly: nil,
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs:    &loadbalancer.LoadbalancerOptionLogs{},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{},
					},
				},
				TargetPools: nil,
			},
			nil,
			testRegion,
			&Model{
				Id:              types.StringValue(id),
				ProjectId:       types.StringValue("pid"),
				ExternalAddress: types.StringNull(),
				Listeners:       types.ListNull(types.ObjectType{AttrTypes: listenerTypes}),
				Name:            types.StringValue("name"),
				Networks:        types.ListNull(types.ObjectType{AttrTypes: networkTypes}),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"acl":                  types.SetNull(types.StringType),
					"private_network_only": types.BoolNull(),
					"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
						"logs": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
							"credentials_ref": types.StringNull(),
							"push_url":        types.StringNull(),
						}),
						"metrics": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
							"credentials_ref": types.StringNull(),
							"push_url":        types.StringNull(),
						}),
					}),
				}),
				PrivateAddress:  types.StringNull(),
				SecurityGroupId: types.StringNull(),
				TargetPools:     types.ListNull(types.ObjectType{AttrTypes: targetPoolTypes}),
				Region:          types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values_ok",
			&loadbalancer.LoadBalancer{
				ExternalAddress: new("external_address"),
				Listeners: []loadbalancer.Listener{
					{
						DisplayName: new("display_name"),
						Port:        new(int32(80)),
						Protocol:    new(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						ServerNameIndicators: []loadbalancer.ServerNameIndicator{
							{
								Name: new("domain.com"),
							},
						},
						TargetPool: new("target_pool"),
						Tcp: &loadbalancer.OptionsTCP{
							IdleTimeout: new("50s"),
						},
						Udp: &loadbalancer.OptionsUDP{
							IdleTimeout: new("50s"),
						},
					},
				},
				Name: new("name"),
				Networks: []loadbalancer.Network{
					{
						NetworkId: new("network_id"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
					{
						NetworkId: new("network_id_2"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
				},
				Options: new(loadbalancer.LoadBalancerOptions{
					PrivateNetworkOnly: new(true),
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: new("logs_credentials_ref"),
							PushUrl:        new("logs_push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: new("metrics_credentials_ref"),
							PushUrl:        new("metrics_push_url"),
						},
					},
				}),
				TargetSecurityGroup: new(loadbalancer.SecurityGroup{
					Id:   new("sg-id-12345"),
					Name: new("sg-name-abcde"),
				}),
				TargetPools: []loadbalancer.TargetPool{
					{
						ActiveHealthCheck: new(loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   new(int32(1)),
							Interval:           new("2s"),
							IntervalJitter:     new("3s"),
							Timeout:            new("4s"),
							UnhealthyThreshold: new(int32(5)),
						}),
						Name:       new("name"),
						TargetPort: new(int32(80)),
						Targets: []loadbalancer.Target{
							{
								DisplayName: new("display_name"),
								Ip:          new("ip"),
							},
						},
						SessionPersistence: new(loadbalancer.SessionPersistence{
							UseSourceIpAddress: new(true),
						}),
					},
				},
			},
			nil,
			testRegion,
			&Model{
				Id:              types.StringValue(id),
				ProjectId:       types.StringValue("pid"),
				ExternalAddress: types.StringValue("external_address"),
				SecurityGroupId: types.StringValue("sg-id-12345"),
				Listeners: types.ListValueMust(types.ObjectType{AttrTypes: listenerTypes}, []attr.Value{
					types.ObjectValueMust(listenerTypes, map[string]attr.Value{
						"display_name": types.StringValue("display_name"),
						"port":         types.Int32Value(80),
						"protocol":     types.StringValue(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						"server_name_indicators": types.ListValueMust(types.ObjectType{AttrTypes: serverNameIndicatorTypes}, []attr.Value{
							types.ObjectValueMust(
								serverNameIndicatorTypes,
								map[string]attr.Value{
									"name": types.StringValue("domain.com"),
								},
							),
						},
						),
						"target_pool": types.StringValue("target_pool"),
						"tcp": types.ObjectValueMust(tcpTypes, map[string]attr.Value{
							"idle_timeout": types.StringValue("50s"),
						}),
						"udp": types.ObjectValueMust(udpTypes, map[string]attr.Value{
							"idle_timeout": types.StringValue("50s"),
						}),
					}),
				}),
				Name: types.StringValue("name"),
				Networks: types.ListValueMust(types.ObjectType{AttrTypes: networkTypes}, []attr.Value{
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
				}),
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"private_network_only": types.BoolValue(true),
						"acl":                  types.SetNull(types.StringType),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"logs": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("logs_credentials_ref"),
								"push_url":        types.StringValue("logs_push_url"),
							}),
							"metrics": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("metrics_credentials_ref"),
								"push_url":        types.StringValue("metrics_push_url"),
							}),
						}),
					},
				),
				TargetPools: types.ListValueMust(types.ObjectType{AttrTypes: targetPoolTypes}, []attr.Value{
					types.ObjectValueMust(targetPoolTypes, map[string]attr.Value{
						"active_health_check": types.ObjectValueMust(activeHealthCheckTypes, map[string]attr.Value{
							"healthy_threshold":   types.Int32Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int32Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int32Value(80),
						"targets": types.ListValueMust(types.ObjectType{AttrTypes: targetTypes}, []attr.Value{
							types.ObjectValueMust(targetTypes, map[string]attr.Value{
								"display_name": types.StringValue("display_name"),
								"ip":           types.StringValue("ip"),
							}),
						}),
						"session_persistence": types.ObjectValueMust(sessionPersistenceTypes, map[string]attr.Value{
							"use_source_ip_address": types.BoolValue(true),
						}),
					}),
				}),
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values_ok_with_null_private_network_only_response",
			&loadbalancer.LoadBalancer{
				ExternalAddress: new("external_address"),
				Listeners: []loadbalancer.Listener{
					{
						DisplayName: new("display_name"),
						Port:        new(int32(80)),
						Protocol:    new(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						ServerNameIndicators: []loadbalancer.ServerNameIndicator{
							{
								Name: new("domain.com"),
							},
						},
						TargetPool: new("target_pool"),
					},
				},
				Name: new("name"),
				Networks: []loadbalancer.Network{
					{
						NetworkId: new("network_id"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
					{
						NetworkId: new("network_id_2"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
				},
				Options: new(loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: []string{"cidr"},
					},
					PrivateNetworkOnly: nil, // API sets this to nil if it's false in the request
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: new("logs_credentials_ref"),
							PushUrl:        new("logs_push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: new("metrics_credentials_ref"),
							PushUrl:        new("metrics_push_url"),
						},
					},
				}),
				TargetPools: []loadbalancer.TargetPool{
					{
						ActiveHealthCheck: new(loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   new(int32(1)),
							Interval:           new("2s"),
							IntervalJitter:     new("3s"),
							Timeout:            new("4s"),
							UnhealthyThreshold: new(int32(5)),
						}),
						Name:       new("name"),
						TargetPort: new(int32(80)),
						Targets: []loadbalancer.Target{
							{
								DisplayName: new("display_name"),
								Ip:          new("ip"),
							},
						},
						SessionPersistence: new(loadbalancer.SessionPersistence{
							UseSourceIpAddress: new(true),
						}),
					},
				},
			},
			new(false),
			testRegion,
			&Model{
				Id:              types.StringValue(id),
				ProjectId:       types.StringValue("pid"),
				ExternalAddress: types.StringValue("external_address"),
				Listeners: types.ListValueMust(types.ObjectType{AttrTypes: listenerTypes}, []attr.Value{
					types.ObjectValueMust(listenerTypes, map[string]attr.Value{
						"display_name": types.StringValue("display_name"),
						"port":         types.Int32Value(80),
						"protocol":     types.StringValue(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						"server_name_indicators": types.ListValueMust(types.ObjectType{AttrTypes: serverNameIndicatorTypes}, []attr.Value{
							types.ObjectValueMust(
								serverNameIndicatorTypes,
								map[string]attr.Value{
									"name": types.StringValue("domain.com"),
								},
							),
						},
						),
						"target_pool": types.StringValue("target_pool"),
						"tcp":         types.ObjectNull(tcpTypes),
						"udp":         types.ObjectNull(udpTypes),
					}),
				}),
				Name: types.StringValue("name"),
				Networks: types.ListValueMust(types.ObjectType{AttrTypes: networkTypes}, []attr.Value{
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
				}),
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"acl": types.SetValueMust(
							types.StringType,
							[]attr.Value{types.StringValue("cidr")}),
						"private_network_only": types.BoolValue(false),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"logs": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("logs_credentials_ref"),
								"push_url":        types.StringValue("logs_push_url"),
							}),
							"metrics": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("metrics_credentials_ref"),
								"push_url":        types.StringValue("metrics_push_url"),
							}),
						}),
					},
				),
				TargetPools: types.ListValueMust(types.ObjectType{AttrTypes: targetPoolTypes}, []attr.Value{
					types.ObjectValueMust(targetPoolTypes, map[string]attr.Value{
						"active_health_check": types.ObjectValueMust(activeHealthCheckTypes, map[string]attr.Value{
							"healthy_threshold":   types.Int32Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int32Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int32Value(80),
						"targets": types.ListValueMust(types.ObjectType{AttrTypes: targetTypes}, []attr.Value{
							types.ObjectValueMust(targetTypes, map[string]attr.Value{
								"display_name": types.StringValue("display_name"),
								"ip":           types.StringValue("ip"),
							}),
						}),
						"session_persistence": types.ObjectValueMust(sessionPersistenceTypes, map[string]attr.Value{
							"use_source_ip_address": types.BoolValue(true),
						}),
					}),
				}),
				Region: types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			nil,
			nil,
			testRegion,
			&Model{},
			false,
		},
		{
			"no_name",
			&loadbalancer.LoadBalancer{},
			nil,
			testRegion,
			&Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId: tt.expected.ProjectId,
			}
			if tt.modelPrivateNetworkOnly != nil {
				model.Options = types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"private_network_only": types.BoolValue(*tt.modelPrivateNetworkOnly),
					"acl":                  types.SetNull(types.StringType),
					"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
						"logs": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
							"credentials_ref": types.StringNull(),
							"push_url":        types.StringNull(),
						}),
						"metrics": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
							"credentials_ref": types.StringNull(),
							"push_url":        types.StringNull(),
						}),
					}),
				})
			}
			err := mapFields(context.Background(), tt.input, model, tt.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(model, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func Test_validateConfig(t *testing.T) {
	type args struct {
		ExternalAddress    *string
		PrivateNetworkOnly *bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy case 1: private_network_only is not set and external_address is set",
			args: args{
				ExternalAddress:    new(testExternalAddress),
				PrivateNetworkOnly: nil,
			},
			wantErr: false,
		},
		{
			name: "happy case 2: private_network_only is set to false and external_address is set",
			args: args{
				ExternalAddress:    new(testExternalAddress),
				PrivateNetworkOnly: new(false),
			},
			wantErr: false,
		},
		{
			name: "happy case 3: private_network_only is set to true and external_address is not set",
			args: args{
				ExternalAddress:    nil,
				PrivateNetworkOnly: new(true),
			},
			wantErr: false,
		},
		{
			name: "error case 1: private_network_only and external_address are set",
			args: args{
				ExternalAddress:    new(testExternalAddress),
				PrivateNetworkOnly: new(true),
			},
			wantErr: true,
		},
		{
			name: "error case 2: private_network_only is not set and external_address is not set",
			args: args{
				ExternalAddress:    nil,
				PrivateNetworkOnly: nil,
			},
			wantErr: true,
		},
		{
			name: "error case 3: private_network_only is set to false and external_address is not set",
			args: args{
				ExternalAddress:    nil,
				PrivateNetworkOnly: new(false),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			diags := diag.Diagnostics{}
			model := &Model{
				ExternalAddress: types.StringPointerValue(tt.args.ExternalAddress),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"acl":                  types.SetNull(types.StringType),
					"observability":        types.ObjectNull(observabilityTypes),
					"private_network_only": types.BoolPointerValue(tt.args.PrivateNetworkOnly),
				}),
			}

			validateConfig(ctx, &diags, model)

			if diags.HasError() != tt.wantErr {
				t.Errorf("validateConfig() = %v, want %v", diags.HasError(), tt.wantErr)
			}
		})
	}
}

func Test_toUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *loadbalancer.UpdateLoadBalancerPayload
		isValid     bool
	}{
		{
			"default_values_ok",
			&Model{},
			&loadbalancer.UpdateLoadBalancerPayload{
				ExternalAddress: nil,
				Listeners:       nil,
				Name:            nil,
				Networks:        nil,
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: nil,
					},
					PrivateNetworkOnly: nil,
					Observability:      &loadbalancer.LoadbalancerOptionObservability{},
				},
				TargetPools: nil,
			},
			true,
		},
		{
			"default_values_with_version_ok",
			&Model{
				Version: types.StringValue("lb-1"),
			},
			&loadbalancer.UpdateLoadBalancerPayload{
				ExternalAddress: nil,
				Listeners:       nil,
				Name:            nil,
				Networks:        nil,
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: nil,
					},
					PrivateNetworkOnly: nil,
					Observability:      &loadbalancer.LoadbalancerOptionObservability{},
				},
				TargetPools: nil,
				Version:     new("lb-1"),
			},
			true,
		},
		{
			"simple_values_ok",
			&Model{
				ExternalAddress: types.StringValue("external_address"),
				Listeners: types.ListValueMust(types.ObjectType{AttrTypes: listenerTypes}, []attr.Value{
					types.ObjectValueMust(listenerTypes, map[string]attr.Value{
						"display_name": types.StringValue("display_name"),
						"port":         types.Int32Value(80),
						"protocol":     types.StringValue(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						"server_name_indicators": types.ListValueMust(types.ObjectType{AttrTypes: serverNameIndicatorTypes}, []attr.Value{
							types.ObjectValueMust(
								serverNameIndicatorTypes,
								map[string]attr.Value{
									"name": types.StringValue("domain.com"),
								},
							),
						},
						),
						"target_pool": types.StringValue("target_pool"),
						"tcp": types.ObjectValueMust(tcpTypes, map[string]attr.Value{
							"idle_timeout": types.StringValue("50s"),
						}),
						"udp": types.ObjectValueMust(udpTypes, map[string]attr.Value{
							"idle_timeout": types.StringValue("50s"),
						}),
					}),
				}),
				Name: types.StringValue("name"),
				Networks: types.ListValueMust(types.ObjectType{AttrTypes: networkTypes}, []attr.Value{
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
				}),
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"acl": types.SetValueMust(
							types.StringType,
							[]attr.Value{types.StringValue("cidr")}),
						"private_network_only": types.BoolValue(true),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"logs": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("logs-credentials_ref"),
								"push_url":        types.StringValue("logs-push_url"),
							}),
							"metrics": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("metrics-credentials_ref"),
								"push_url":        types.StringValue("metrics-push_url"),
							}),
						}),
					},
				),
				TargetPools: types.ListValueMust(types.ObjectType{AttrTypes: targetPoolTypes}, []attr.Value{
					types.ObjectValueMust(targetPoolTypes, map[string]attr.Value{
						"active_health_check": types.ObjectValueMust(activeHealthCheckTypes, map[string]attr.Value{
							"healthy_threshold":   types.Int32Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int32Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int32Value(80),
						"targets": types.ListValueMust(types.ObjectType{AttrTypes: targetTypes}, []attr.Value{
							types.ObjectValueMust(targetTypes, map[string]attr.Value{
								"display_name": types.StringValue("display_name"),
								"ip":           types.StringValue("ip"),
							}),
						}),
						"session_persistence": types.ObjectValueMust(sessionPersistenceTypes, map[string]attr.Value{
							"use_source_ip_address": types.BoolValue(true),
						}),
					}),
				}),
			},
			&loadbalancer.UpdateLoadBalancerPayload{
				ExternalAddress: new("external_address"),
				Listeners: []loadbalancer.Listener{
					{
						DisplayName: new("display_name"),
						Port:        new(int32(80)),
						Protocol:    new(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						ServerNameIndicators: []loadbalancer.ServerNameIndicator{
							{
								Name: new("domain.com"),
							},
						},
						TargetPool: new("target_pool"),
						Tcp: new(loadbalancer.OptionsTCP{
							IdleTimeout: new("50s"),
						}),
						Udp: new(loadbalancer.OptionsUDP{
							IdleTimeout: new("50s"),
						}),
					},
				},
				Name: new("name"),
				Networks: []loadbalancer.Network{
					{
						NetworkId: new("network_id"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
					{
						NetworkId: new("network_id_2"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
				},
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: []string{"cidr"},
					},
					PrivateNetworkOnly: new(true),
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: new("logs-credentials_ref"),
							PushUrl:        new("logs-push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: new("metrics-credentials_ref"),
							PushUrl:        new("metrics-push_url"),
						},
					},
				},
				TargetPools: []loadbalancer.TargetPool{
					{
						ActiveHealthCheck: &loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   new(int32(1)),
							Interval:           new("2s"),
							IntervalJitter:     new("3s"),
							Timeout:            new("4s"),
							UnhealthyThreshold: new(int32(5)),
						},
						Name:       new("name"),
						TargetPort: new(int32(80)),
						Targets: []loadbalancer.Target{
							{
								DisplayName: new("display_name"),
								Ip:          new("ip"),
							},
						},
						SessionPersistence: &loadbalancer.SessionPersistence{
							UseSourceIpAddress: new(true),
						},
					},
				},
			},
			true,
		},
		{
			"service_plan_ok",
			&Model{
				PlanId:          types.StringValue("p10"),
				ExternalAddress: types.StringValue("external_address"),
				Listeners: types.ListValueMust(types.ObjectType{AttrTypes: listenerTypes}, []attr.Value{
					types.ObjectValueMust(listenerTypes, map[string]attr.Value{
						"display_name": types.StringValue("display_name"),
						"port":         types.Int32Value(80),
						"protocol":     types.StringValue(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						"server_name_indicators": types.ListValueMust(types.ObjectType{AttrTypes: serverNameIndicatorTypes}, []attr.Value{
							types.ObjectValueMust(
								serverNameIndicatorTypes,
								map[string]attr.Value{
									"name": types.StringValue("domain.com"),
								},
							),
						},
						),
						"target_pool": types.StringValue("target_pool"),
						"tcp":         types.ObjectNull(tcpTypes),
						"udp":         types.ObjectNull(udpTypes),
					}),
				}),
				Name: types.StringValue("name"),
				Networks: types.ListValueMust(types.ObjectType{AttrTypes: networkTypes}, []attr.Value{
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
				}),
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"acl": types.SetValueMust(
							types.StringType,
							[]attr.Value{types.StringValue("cidr")}),
						"private_network_only": types.BoolValue(true),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"logs": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("logs-credentials_ref"),
								"push_url":        types.StringValue("logs-push_url"),
							}),
							"metrics": types.ObjectValueMust(observabilityOptionTypes, map[string]attr.Value{
								"credentials_ref": types.StringValue("metrics-credentials_ref"),
								"push_url":        types.StringValue("metrics-push_url"),
							}),
						}),
					},
				),
				TargetPools: types.ListValueMust(types.ObjectType{AttrTypes: targetPoolTypes}, []attr.Value{
					types.ObjectValueMust(targetPoolTypes, map[string]attr.Value{
						"active_health_check": types.ObjectValueMust(activeHealthCheckTypes, map[string]attr.Value{
							"healthy_threshold":   types.Int32Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int32Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int32Value(80),
						"targets": types.ListValueMust(types.ObjectType{AttrTypes: targetTypes}, []attr.Value{
							types.ObjectValueMust(targetTypes, map[string]attr.Value{
								"display_name": types.StringValue("display_name"),
								"ip":           types.StringValue("ip"),
							}),
						}),
						"session_persistence": types.ObjectValueMust(sessionPersistenceTypes, map[string]attr.Value{
							"use_source_ip_address": types.BoolValue(true),
						}),
					}),
				}),
			},
			&loadbalancer.UpdateLoadBalancerPayload{
				PlanId:          new("p10"),
				ExternalAddress: new("external_address"),
				Listeners: []loadbalancer.Listener{
					{
						DisplayName: new("display_name"),
						Port:        new(int32(80)),
						Protocol:    new(string(legacyLoadbalancer.LISTENERPROTOCOL_TCP)),
						ServerNameIndicators: []loadbalancer.ServerNameIndicator{
							{
								Name: new("domain.com"),
							},
						},
						TargetPool: new("target_pool"),
					},
				},
				Name: new("name"),
				Networks: []loadbalancer.Network{
					{
						NetworkId: new("network_id"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
					{
						NetworkId: new("network_id_2"),
						Role:      new(string(legacyLoadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					},
				},
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: []string{"cidr"},
					},
					PrivateNetworkOnly: new(true),
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: new("logs-credentials_ref"),
							PushUrl:        new("logs-push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: new("metrics-credentials_ref"),
							PushUrl:        new("metrics-push_url"),
						},
					},
				},
				TargetPools: []loadbalancer.TargetPool{
					{
						ActiveHealthCheck: &loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   new(int32(1)),
							Interval:           new("2s"),
							IntervalJitter:     new("3s"),
							Timeout:            new("4s"),
							UnhealthyThreshold: new(int32(5)),
						},
						Name:       new("name"),
						TargetPort: new(int32(80)),
						Targets: []loadbalancer.Target{
							{
								DisplayName: new("display_name"),
								Ip:          new("ip"),
							},
						},
						SessionPersistence: &loadbalancer.SessionPersistence{
							UseSourceIpAddress: new(true),
						},
					},
				},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func Test_mapSessionPersistence(t *testing.T) {
	tests := []struct {
		name                   string
		sessionPersistenceResp *loadbalancer.SessionPersistence
		wantTp                 map[string]attr.Value
		wantErr                bool
	}{
		{
			name:                   "session persistence is nil",
			sessionPersistenceResp: nil,
			wantTp: map[string]attr.Value{
				"session_persistence": types.ObjectValueMust(sessionPersistenceTypes,
					map[string]attr.Value{
						"use_source_ip_address": types.BoolValue(false),
					},
				),
			},
		},
		{
			name: "use source ip address is false",
			sessionPersistenceResp: &loadbalancer.SessionPersistence{
				UseSourceIpAddress: new(false),
			},
			wantTp: map[string]attr.Value{
				"session_persistence": types.ObjectValueMust(sessionPersistenceTypes,
					map[string]attr.Value{
						"use_source_ip_address": types.BoolValue(false),
					},
				),
			},
		},
		{
			name: "use source ip address is true",
			sessionPersistenceResp: &loadbalancer.SessionPersistence{
				UseSourceIpAddress: new(true),
			},
			wantTp: map[string]attr.Value{
				"session_persistence": types.ObjectValueMust(sessionPersistenceTypes,
					map[string]attr.Value{
						"use_source_ip_address": types.BoolValue(true),
					},
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultTp := map[string]attr.Value{}
			gotErr := mapSessionPersistence(tt.sessionPersistenceResp, resultTp)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("mapSessionPersistence() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("mapSessionPersistence() succeeded unexpectedly")
			}
			if diff := cmp.Diff(tt.wantTp, resultTp); diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}
