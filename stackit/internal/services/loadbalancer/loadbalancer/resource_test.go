package loadbalancer

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
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
				Listeners: []Listener{
					{
						DisplayName: types.StringValue("display_name"),
						Port:        types.Int64Value(80),
						Protocol:    types.StringValue("protocol"),
						TargetPool:  types.StringValue("target_pool"),
					},
				},
				Name: types.StringValue("name"),
				Networks: []Network{
					{
						NetworkId: types.StringValue("network_id"),
						Role:      types.StringValue("role"),
					},
					{
						NetworkId: types.StringValue("network_id_2"),
						Role:      types.StringValue("role_2"),
					},
				},
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"acl": types.ListValueMust(
							types.StringType,
							[]attr.Value{types.StringValue("cidr")}),
						"private_network_only": types.BoolValue(true),
					},
				),
				TargetPools: []TargetPool{
					{
						ActiveHealthCheck: types.ObjectValueMust(
							activeHealthCheckTypes,
							map[string]attr.Value{
								"healthy_threshold":   types.Int64Value(1),
								"interval":            types.StringValue("2s"),
								"interval_jitter":     types.StringValue("3s"),
								"timeout":             types.StringValue("4s"),
								"unhealthy_threshold": types.Int64Value(5),
							},
						),
						Name:       types.StringValue("name"),
						TargetPort: types.Int64Value(80),
						Targets: []Target{
							{
								DisplayName: types.StringValue("display_name"),
								Ip:          types.StringValue("ip"),
							},
						},
					},
				},
			},
			&loadbalancer.CreateLoadBalancerPayload{
				ExternalAddress: utils.Ptr("external_address"),
				Listeners: utils.Ptr([]loadbalancer.Listener{
					{
						DisplayName: utils.Ptr("display_name"),
						Port:        utils.Ptr(int64(80)),
						Protocol:    utils.Ptr("protocol"),
						TargetPool:  utils.Ptr("target_pool"),
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
					},
				}),
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
						TargetPool:  utils.Ptr("target_pool"),
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
					},
				}),
			},
			&Model{
				Id:              types.StringValue("pid,name"),
				ProjectId:       types.StringValue("pid"),
				Name:            types.StringValue("name"),
				ExternalAddress: types.StringValue("external_address"),
				Listeners: []Listener{
					{
						DisplayName: types.StringValue("display_name"),
						Port:        types.Int64Value(80),
						Protocol:    types.StringValue("protocol"),
						TargetPool:  types.StringValue("target_pool"),
					},
				},
				Networks: []Network{
					{
						NetworkId: types.StringValue("network_id"),
						Role:      types.StringValue("role"),
					},
					{
						NetworkId: types.StringValue("network_id_2"),
						Role:      types.StringValue("role_2"),
					},
				},
				Options: types.ObjectValueMust(
					optionsTypes,
					map[string]attr.Value{
						"acl": types.ListValueMust(
							types.StringType,

							[]attr.Value{types.StringValue("cidr")}),
						"private_network_only": types.BoolValue(true),
					},
				),
				TargetPools: []TargetPool{
					{
						ActiveHealthCheck: types.ObjectValueMust(
							activeHealthCheckTypes,
							map[string]attr.Value{
								"healthy_threshold": types.Int64Value(1),
								"interval":          types.StringValue("2s"),
								"interval_jitter":   types.StringValue("3s"),
								"timeout":           types.StringValue("4s"),

								"unhealthy_threshold": types.Int64Value(5),
							},
						),
						Name:       types.StringValue("name"),
						TargetPort: types.Int64Value(80),
						Targets: []Target{
							{
								DisplayName: types.StringValue("display_name"),
								Ip:          types.StringValue("ip"),
							},
						},
					},
				},
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
			err := mapFields(context.Background(), tt.input, model)
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
