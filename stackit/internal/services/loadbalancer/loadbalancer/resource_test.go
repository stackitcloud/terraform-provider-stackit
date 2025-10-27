package loadbalancer

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
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
						"port":         types.Int64Value(80),
						"protocol":     types.StringValue(string(loadbalancer.LISTENERPROTOCOL_TCP)),
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
						"role":       types.StringValue(string(loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
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
							"healthy_threshold":   types.Int64Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int64Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int64Value(80),
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
				ExternalAddress: utils.Ptr("external_address"),
				Listeners: &[]loadbalancer.Listener{
					{
						DisplayName: utils.Ptr("display_name"),
						Port:        utils.Ptr(int64(80)),
						Protocol:    loadbalancer.LISTENERPROTOCOL_TCP.Ptr(),
						ServerNameIndicators: &[]loadbalancer.ServerNameIndicator{
							{
								Name: utils.Ptr("domain.com"),
							},
						},
						TargetPool: utils.Ptr("target_pool"),
						Tcp: &loadbalancer.OptionsTCP{
							IdleTimeout: utils.Ptr("50s"),
						},
						Udp: &loadbalancer.OptionsUDP{
							IdleTimeout: utils.Ptr("50s"),
						},
					},
				},
				Name: utils.Ptr("name"),
				Networks: &[]loadbalancer.Network{
					{
						NetworkId: utils.Ptr("network_id"),
						Role:      loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS.Ptr(),
					},
					{
						NetworkId: utils.Ptr("network_id_2"),
						Role:      loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS.Ptr(),
					},
				},
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: &[]string{"cidr"},
					},
					PrivateNetworkOnly: utils.Ptr(true),
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: utils.Ptr("logs-credentials_ref"),
							PushUrl:        utils.Ptr("logs-push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: utils.Ptr("metrics-credentials_ref"),
							PushUrl:        utils.Ptr("metrics-push_url"),
						},
					},
				},
				TargetPools: &[]loadbalancer.TargetPool{
					{
						ActiveHealthCheck: &loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   utils.Ptr(int64(1)),
							Interval:           utils.Ptr("2s"),
							IntervalJitter:     utils.Ptr("3s"),
							Timeout:            utils.Ptr("4s"),
							UnhealthyThreshold: utils.Ptr(int64(5)),
						},
						Name:       utils.Ptr("name"),
						TargetPort: utils.Ptr(int64(80)),
						Targets: &[]loadbalancer.Target{
							{
								DisplayName: utils.Ptr("display_name"),
								Ip:          utils.Ptr("ip"),
							},
						},
						SessionPersistence: &loadbalancer.SessionPersistence{
							UseSourceIpAddress: utils.Ptr(true),
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
						"port":         types.Int64Value(80),
						"protocol":     types.StringValue(string(loadbalancer.LISTENERPROTOCOL_TCP)),
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
						"role":       types.StringValue(string(loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
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
							"healthy_threshold":   types.Int64Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int64Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int64Value(80),
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
				PlanId:          utils.Ptr("p10"),
				ExternalAddress: utils.Ptr("external_address"),
				Listeners: &[]loadbalancer.Listener{
					{
						DisplayName: utils.Ptr("display_name"),
						Port:        utils.Ptr(int64(80)),
						Protocol:    loadbalancer.LISTENERPROTOCOL_TCP.Ptr(),
						ServerNameIndicators: &[]loadbalancer.ServerNameIndicator{
							{
								Name: utils.Ptr("domain.com"),
							},
						},
						TargetPool: utils.Ptr("target_pool"),
					},
				},
				Name: utils.Ptr("name"),
				Networks: &[]loadbalancer.Network{
					{
						NetworkId: utils.Ptr("network_id"),
						Role:      loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS.Ptr(),
					},
					{
						NetworkId: utils.Ptr("network_id_2"),
						Role:      loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS.Ptr(),
					},
				},
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: &[]string{"cidr"},
					},
					PrivateNetworkOnly: utils.Ptr(true),
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: utils.Ptr("logs-credentials_ref"),
							PushUrl:        utils.Ptr("logs-push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: utils.Ptr("metrics-credentials_ref"),
							PushUrl:        utils.Ptr("metrics-push_url"),
						},
					},
				},
				TargetPools: &[]loadbalancer.TargetPool{
					{
						ActiveHealthCheck: &loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   utils.Ptr(int64(1)),
							Interval:           utils.Ptr("2s"),
							IntervalJitter:     utils.Ptr("3s"),
							Timeout:            utils.Ptr("4s"),
							UnhealthyThreshold: utils.Ptr(int64(5)),
						},
						Name:       utils.Ptr("name"),
						TargetPort: utils.Ptr(int64(80)),
						Targets: &[]loadbalancer.Target{
							{
								DisplayName: utils.Ptr("display_name"),
								Ip:          utils.Ptr("ip"),
							},
						},
						SessionPersistence: &loadbalancer.SessionPersistence{
							UseSourceIpAddress: utils.Ptr(true),
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

func TestToTargetPoolUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *targetPool
		expected    *loadbalancer.UpdateTargetPoolPayload
		isValid     bool
	}{
		{
			"default_values_ok",
			&targetPool{},
			&loadbalancer.UpdateTargetPoolPayload{},
			true,
		},
		{
			"simple_values_ok",
			&targetPool{
				ActiveHealthCheck: types.ObjectValueMust(activeHealthCheckTypes, map[string]attr.Value{
					"healthy_threshold":   types.Int64Value(1),
					"interval":            types.StringValue("2s"),
					"interval_jitter":     types.StringValue("3s"),
					"timeout":             types.StringValue("4s"),
					"unhealthy_threshold": types.Int64Value(5),
				}),
				Name:       types.StringValue("name"),
				TargetPort: types.Int64Value(80),
				Targets: types.ListValueMust(types.ObjectType{AttrTypes: targetTypes}, []attr.Value{
					types.ObjectValueMust(targetTypes, map[string]attr.Value{
						"display_name": types.StringValue("display_name"),
						"ip":           types.StringValue("ip"),
					}),
				}),
				SessionPersistence: types.ObjectValueMust(sessionPersistenceTypes, map[string]attr.Value{
					"use_source_ip_address": types.BoolValue(false),
				}),
			},
			&loadbalancer.UpdateTargetPoolPayload{
				ActiveHealthCheck: &loadbalancer.ActiveHealthCheck{
					HealthyThreshold:   utils.Ptr(int64(1)),
					Interval:           utils.Ptr("2s"),
					IntervalJitter:     utils.Ptr("3s"),
					Timeout:            utils.Ptr("4s"),
					UnhealthyThreshold: utils.Ptr(int64(5)),
				},
				Name:       utils.Ptr("name"),
				TargetPort: utils.Ptr(int64(80)),
				Targets: &[]loadbalancer.Target{
					{
						DisplayName: utils.Ptr("display_name"),
						Ip:          utils.Ptr("ip"),
					},
				},
				SessionPersistence: &loadbalancer.SessionPersistence{
					UseSourceIpAddress: utils.Ptr(false),
				},
			},
			true,
		},
		{
			"nil_target_pool",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toTargetPoolUpdatePayload(context.Background(), tt.input)
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
				Name:            utils.Ptr("name"),
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
				ExternalAddress: utils.Ptr("external_address"),
				Listeners: utils.Ptr([]loadbalancer.Listener{
					{
						DisplayName: utils.Ptr("display_name"),
						Port:        utils.Ptr(int64(80)),
						Protocol:    loadbalancer.LISTENERPROTOCOL_TCP.Ptr(),
						ServerNameIndicators: &[]loadbalancer.ServerNameIndicator{
							{
								Name: utils.Ptr("domain.com"),
							},
						},
						TargetPool: utils.Ptr("target_pool"),
						Tcp: &loadbalancer.OptionsTCP{
							IdleTimeout: utils.Ptr("50s"),
						},
						Udp: &loadbalancer.OptionsUDP{
							IdleTimeout: utils.Ptr("50s"),
						},
					},
				}),
				Name: utils.Ptr("name"),
				Networks: utils.Ptr([]loadbalancer.Network{
					{
						NetworkId: utils.Ptr("network_id"),
						Role:      loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS.Ptr(),
					},
					{
						NetworkId: utils.Ptr("network_id_2"),
						Role:      loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS.Ptr(),
					},
				}),
				Options: utils.Ptr(loadbalancer.LoadBalancerOptions{
					PrivateNetworkOnly: utils.Ptr(true),
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: utils.Ptr("logs_credentials_ref"),
							PushUrl:        utils.Ptr("logs_push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: utils.Ptr("metrics_credentials_ref"),
							PushUrl:        utils.Ptr("metrics_push_url"),
						},
					},
				}),
				TargetSecurityGroup: loadbalancer.LoadBalancerGetTargetSecurityGroupAttributeType(&loadbalancer.SecurityGroup{
					Id:   utils.Ptr("sg-id-12345"),
					Name: utils.Ptr("sg-name-abcde"),
				}),
				TargetPools: utils.Ptr([]loadbalancer.TargetPool{
					{
						ActiveHealthCheck: utils.Ptr(loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   utils.Ptr(int64(1)),
							Interval:           utils.Ptr("2s"),
							IntervalJitter:     utils.Ptr("3s"),
							Timeout:            utils.Ptr("4s"),
							UnhealthyThreshold: utils.Ptr(int64(5)),
						}),
						Name:       utils.Ptr("name"),
						TargetPort: utils.Ptr(int64(80)),
						Targets: utils.Ptr([]loadbalancer.Target{
							{
								DisplayName: utils.Ptr("display_name"),
								Ip:          utils.Ptr("ip"),
							},
						}),
						SessionPersistence: utils.Ptr(loadbalancer.SessionPersistence{
							UseSourceIpAddress: utils.Ptr(true),
						}),
					},
				}),
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
						"port":         types.Int64Value(80),
						"protocol":     types.StringValue(string(loadbalancer.LISTENERPROTOCOL_TCP)),
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
						"role":       types.StringValue(string(loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
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
							"healthy_threshold":   types.Int64Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int64Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int64Value(80),
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
				ExternalAddress: utils.Ptr("external_address"),
				Listeners: utils.Ptr([]loadbalancer.Listener{
					{
						DisplayName: utils.Ptr("display_name"),
						Port:        utils.Ptr(int64(80)),
						Protocol:    loadbalancer.LISTENERPROTOCOL_TCP.Ptr(),
						ServerNameIndicators: &[]loadbalancer.ServerNameIndicator{
							{
								Name: utils.Ptr("domain.com"),
							},
						},
						TargetPool: utils.Ptr("target_pool"),
					},
				}),
				Name: utils.Ptr("name"),
				Networks: utils.Ptr([]loadbalancer.Network{
					{
						NetworkId: utils.Ptr("network_id"),
						Role:      loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS.Ptr(),
					},
					{
						NetworkId: utils.Ptr("network_id_2"),
						Role:      loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS.Ptr(),
					},
				}),
				Options: utils.Ptr(loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: utils.Ptr([]string{"cidr"}),
					},
					PrivateNetworkOnly: nil, // API sets this to nil if it's false in the request
					Observability: &loadbalancer.LoadbalancerOptionObservability{
						Logs: &loadbalancer.LoadbalancerOptionLogs{
							CredentialsRef: utils.Ptr("logs_credentials_ref"),
							PushUrl:        utils.Ptr("logs_push_url"),
						},
						Metrics: &loadbalancer.LoadbalancerOptionMetrics{
							CredentialsRef: utils.Ptr("metrics_credentials_ref"),
							PushUrl:        utils.Ptr("metrics_push_url"),
						},
					},
				}),
				TargetPools: utils.Ptr([]loadbalancer.TargetPool{
					{
						ActiveHealthCheck: utils.Ptr(loadbalancer.ActiveHealthCheck{
							HealthyThreshold:   utils.Ptr(int64(1)),
							Interval:           utils.Ptr("2s"),
							IntervalJitter:     utils.Ptr("3s"),
							Timeout:            utils.Ptr("4s"),
							UnhealthyThreshold: utils.Ptr(int64(5)),
						}),
						Name:       utils.Ptr("name"),
						TargetPort: utils.Ptr(int64(80)),
						Targets: utils.Ptr([]loadbalancer.Target{
							{
								DisplayName: utils.Ptr("display_name"),
								Ip:          utils.Ptr("ip"),
							},
						}),
						SessionPersistence: utils.Ptr(loadbalancer.SessionPersistence{
							UseSourceIpAddress: utils.Ptr(true),
						}),
					},
				}),
			},
			utils.Ptr(false),
			testRegion,
			&Model{
				Id:              types.StringValue(id),
				ProjectId:       types.StringValue("pid"),
				ExternalAddress: types.StringValue("external_address"),
				Listeners: types.ListValueMust(types.ObjectType{AttrTypes: listenerTypes}, []attr.Value{
					types.ObjectValueMust(listenerTypes, map[string]attr.Value{
						"display_name": types.StringValue("display_name"),
						"port":         types.Int64Value(80),
						"protocol":     types.StringValue(string(loadbalancer.LISTENERPROTOCOL_TCP)),
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
						"role":       types.StringValue(string(loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue(string(loadbalancer.NETWORKROLE_LISTENERS_AND_TARGETS)),
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
							"healthy_threshold":   types.Int64Value(1),
							"interval":            types.StringValue("2s"),
							"interval_jitter":     types.StringValue("3s"),
							"timeout":             types.StringValue("4s"),
							"unhealthy_threshold": types.Int64Value(5),
						}),
						"name":        types.StringValue("name"),
						"target_port": types.Int64Value(80),
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
				ExternalAddress:    utils.Ptr(testExternalAddress),
				PrivateNetworkOnly: nil,
			},
			wantErr: false,
		},
		{
			name: "happy case 2: private_network_only is set to false and external_address is set",
			args: args{
				ExternalAddress:    utils.Ptr(testExternalAddress),
				PrivateNetworkOnly: utils.Ptr(false),
			},
			wantErr: false,
		},
		{
			name: "happy case 3: private_network_only is set to true and external_address is not set",
			args: args{
				ExternalAddress:    nil,
				PrivateNetworkOnly: utils.Ptr(true),
			},
			wantErr: false,
		},
		{
			name: "error case 1: private_network_only and external_address are set",
			args: args{
				ExternalAddress:    utils.Ptr(testExternalAddress),
				PrivateNetworkOnly: utils.Ptr(true),
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
				PrivateNetworkOnly: utils.Ptr(false),
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
