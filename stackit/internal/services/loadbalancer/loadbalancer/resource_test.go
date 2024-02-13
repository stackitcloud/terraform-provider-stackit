package loadbalancer

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
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
						"protocol":     types.StringValue("protocol"),
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
					}),
				}),
				Name: types.StringValue("name"),
				Networks: types.ListValueMust(types.ObjectType{AttrTypes: networkTypes}, []attr.Value{
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id"),
						"role":       types.StringValue("role"),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue("role_2"),
					}),
				}),
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"acl": types.SetValueMust(
							types.StringType,
							[]attr.Value{types.StringValue("cidr")}),
						"private_network_only": types.BoolValue(true),
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
						Protocol:    utils.Ptr("protocol"),
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
						Role:      utils.Ptr("role"),
					},
					{
						NetworkId: utils.Ptr("network_id_2"),
						Role:      utils.Ptr("role_2"),
					},
				},
				Options: &loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: &[]string{"cidr"},
					},
					PrivateNetworkOnly: utils.Ptr(true),
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
	tests := []struct {
		description string
		input       *loadbalancer.LoadBalancer
		expected    *Model
		isValid     bool
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
				},
				TargetPools: nil,
			},
			&Model{
				Id:        types.StringValue("pid,name"),
				ProjectId: types.StringValue("pid"),
				Name:      types.StringValue("name"),
				Options: types.ObjectValueMust(optionsTypes, map[string]attr.Value{
					"acl":                  types.SetNull(types.StringType),
					"private_network_only": types.BoolNull(),
				}),
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
						Protocol:    utils.Ptr("protocol"),
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
						Role:      utils.Ptr("role"),
					},
					{
						NetworkId: utils.Ptr("network_id_2"),
						Role:      utils.Ptr("role_2"),
					},
				}),
				Options: utils.Ptr(loadbalancer.LoadBalancerOptions{
					AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{
						AllowedSourceRanges: utils.Ptr([]string{"cidr"}),
					},
					PrivateNetworkOnly: utils.Ptr(true),
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
			&Model{
				Id:              types.StringValue("pid,name"),
				ProjectId:       types.StringValue("pid"),
				ExternalAddress: types.StringValue("external_address"),
				Listeners: types.ListValueMust(types.ObjectType{AttrTypes: listenerTypes}, []attr.Value{
					types.ObjectValueMust(listenerTypes, map[string]attr.Value{
						"display_name": types.StringValue("display_name"),
						"port":         types.Int64Value(80),
						"protocol":     types.StringValue("protocol"),
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
					}),
				}),
				Name: types.StringValue("name"),
				Networks: types.ListValueMust(types.ObjectType{AttrTypes: networkTypes}, []attr.Value{
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id"),
						"role":       types.StringValue("role"),
					}),
					types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"network_id": types.StringValue("network_id_2"),
						"role":       types.StringValue("role_2"),
					}),
				}),
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"acl": types.SetValueMust(
							types.StringType,
							[]attr.Value{types.StringValue("cidr")}),
						"private_network_only": types.BoolValue(true),
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
			true,
		},
		{
			"nil_response",
			nil,
			&Model{},
			false,
		},
		{
			"no_name",
			&loadbalancer.LoadBalancer{},
			&Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId: tt.expected.ProjectId,
			}
			err := mapFields(tt.input, model)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(model, tt.expected, cmpopts.IgnoreTypes(types.ListNull(types.StringType)))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
