package ske

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

type skeClientMocked struct {
	returnError    bool
	getClusterResp *ske.Cluster
}

const testRegion = "region"

func (c *skeClientMocked) GetClusterExecute(_ context.Context, _, _, _ string) (*ske.Cluster, error) {
	if c.returnError {
		return nil, fmt.Errorf("get cluster failed")
	}

	return c.getClusterResp, nil
}

func TestMapFields(t *testing.T) {
	cs := ske.ClusterStatusState("OK")
	tests := []struct {
		description     string
		stateExtensions types.Object
		stateNodePools  types.List
		input           *ske.Cluster
		region          string
		expected        Model
		isValid         bool
	}{
		{
			"default_values",
			types.ObjectNull(extensionsTypes),
			types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
			&ske.Cluster{
				Name: utils.Ptr("name"),
			},
			testRegion,
			Model{
				Id:                        types.StringValue("pid,region,name"),
				ProjectId:                 types.StringValue("pid"),
				Name:                      types.StringValue("name"),
				KubernetesVersion:         types.StringNull(),
				AllowPrivilegedContainers: types.BoolNull(),
				NodePools:                 types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
				Maintenance:               types.ObjectNull(maintenanceTypes),
				Network:                   types.ObjectNull(networkTypes),
				Hibernations:              types.ListNull(types.ObjectType{AttrTypes: hibernationTypes}),
				Extensions:                types.ObjectNull(extensionsTypes),
				EgressAddressRanges:       types.ListNull(types.StringType),
				PodAddressRanges:          types.ListNull(types.StringType),
				Region:                    types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			types.ObjectNull(extensionsTypes),
			types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
			&ske.Cluster{
				Extensions: &ske.Extension{
					Acl: &ske.ACL{
						AllowedCidrs: &[]string{"cidr1"},
						Enabled:      utils.Ptr(true),
					},
					Observability: &ske.Observability{
						InstanceId: utils.Ptr("aid"),
						Enabled:    utils.Ptr(true),
					},
					Dns: &ske.DNS{
						Zones:   &[]string{"foo.onstackit.cloud"},
						Enabled: utils.Ptr(true),
					},
				},
				Hibernation: &ske.Hibernation{
					Schedules: &[]ske.HibernationSchedule{
						{
							End:      utils.Ptr("2"),
							Start:    utils.Ptr("1"),
							Timezone: utils.Ptr("CET"),
						},
					},
				},
				Kubernetes: &ske.Kubernetes{
					Version: utils.Ptr("1.2.3"),
				},
				Maintenance: &ske.Maintenance{
					AutoUpdate: &ske.MaintenanceAutoUpdate{
						KubernetesVersion:   utils.Ptr(true),
						MachineImageVersion: utils.Ptr(true),
					},
					TimeWindow: &ske.TimeWindow{
						Start: utils.Ptr(time.Date(0, 1, 2, 3, 4, 5, 6, time.FixedZone("UTC+6:00", 6*60*60))),
						End:   utils.Ptr(time.Date(10, 11, 12, 13, 14, 15, 0, time.UTC)),
					},
				},
				Network: &ske.Network{
					Id: utils.Ptr("nid"),
				},
				Name: utils.Ptr("name"),
				Nodepools: &[]ske.Nodepool{
					{
						AllowSystemComponents: utils.Ptr(true),
						AvailabilityZones:     &[]string{"z1", "z2"},
						Cri: &ske.CRI{
							Name: ske.CRINAME_DOCKER.Ptr(),
						},
						Labels: &map[string]string{"k": "v"},
						Machine: &ske.Machine{
							Image: &ske.Image{
								Name:    utils.Ptr("os"),
								Version: utils.Ptr("os-ver"),
							},
							Type: utils.Ptr("B"),
						},
						MaxSurge:       utils.Ptr(int64(3)),
						MaxUnavailable: nil,
						Maximum:        utils.Ptr(int64(5)),
						Minimum:        utils.Ptr(int64(1)),
						Name:           utils.Ptr("node"),
						Taints: &[]ske.Taint{
							{
								Effect: ske.TAINTEFFECT_NO_EXECUTE.Ptr(),
								Key:    utils.Ptr("key"),
								Value:  utils.Ptr("value"),
							},
						},
						Volume: &ske.Volume{
							Size: utils.Ptr(int64(3)),
							Type: utils.Ptr("type"),
						},
					},
				},
				Status: &ske.ClusterStatus{
					Aggregated:          &cs,
					Error:               nil,
					Hibernated:          nil,
					EgressAddressRanges: &[]string{"0.0.0.0/32", "1.1.1.1/32"},
					PodAddressRanges:    &[]string{"0.0.0.0/32", "1.1.1.1/32"},
				},
			},
			testRegion,
			Model{
				Id:                        types.StringValue("pid,region,name"),
				ProjectId:                 types.StringValue("pid"),
				Name:                      types.StringValue("name"),
				KubernetesVersion:         types.StringNull(),
				KubernetesVersionUsed:     types.StringValue("1.2.3"),
				AllowPrivilegedContainers: types.BoolValue(true),
				EgressAddressRanges: types.ListValueMust(
					types.StringType,
					[]attr.Value{
						types.StringValue("0.0.0.0/32"),
						types.StringValue("1.1.1.1/32"),
					},
				),
				PodAddressRanges: types.ListValueMust(
					types.StringType,
					[]attr.Value{
						types.StringValue("0.0.0.0/32"),
						types.StringValue("1.1.1.1/32"),
					},
				),
				NodePools: types.ListValueMust(
					types.ObjectType{AttrTypes: nodePoolTypes},
					[]attr.Value{
						types.ObjectValueMust(
							nodePoolTypes,
							map[string]attr.Value{
								"name":            types.StringValue("node"),
								"machine_type":    types.StringValue("B"),
								"os_name":         types.StringValue("os"),
								"os_version":      types.StringNull(),
								"os_version_min":  types.StringNull(),
								"os_version_used": types.StringValue("os-ver"),
								"minimum":         types.Int64Value(1),
								"maximum":         types.Int64Value(5),
								"max_surge":       types.Int64Value(3),
								"max_unavailable": types.Int64Null(),
								"volume_type":     types.StringValue("type"),
								"volume_size":     types.Int64Value(3),
								"labels": types.MapValueMust(
									types.StringType,
									map[string]attr.Value{
										"k": types.StringValue("v"),
									},
								),
								"taints": types.ListValueMust(
									types.ObjectType{AttrTypes: taintTypes},
									[]attr.Value{
										types.ObjectValueMust(
											taintTypes,
											map[string]attr.Value{
												"effect": types.StringValue(string(ske.TAINTEFFECT_NO_EXECUTE)),
												"key":    types.StringValue("key"),
												"value":  types.StringValue("value"),
											},
										),
									},
								),
								"cri": types.StringValue(string(ske.CRINAME_DOCKER)),
								"availability_zones": types.ListValueMust(
									types.StringType,
									[]attr.Value{
										types.StringValue("z1"),
										types.StringValue("z2"),
									},
								),
								"allow_system_components": types.BoolValue(true),
							},
						),
					},
				),
				Maintenance: types.ObjectValueMust(maintenanceTypes, map[string]attr.Value{
					"enable_kubernetes_version_updates":    types.BoolValue(true),
					"enable_machine_image_version_updates": types.BoolValue(true),
					"start":                                types.StringValue("03:04:05+06:00"),
					"end":                                  types.StringValue("13:14:15Z"),
				}),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"id": types.StringValue("nid"),
				}),
				Hibernations: types.ListValueMust(
					types.ObjectType{AttrTypes: hibernationTypes},
					[]attr.Value{
						types.ObjectValueMust(
							hibernationTypes,
							map[string]attr.Value{
								"start":    types.StringValue("1"),
								"end":      types.StringValue("2"),
								"timezone": types.StringValue("CET"),
							},
						),
					},
				),
				Extensions: types.ObjectValueMust(extensionsTypes, map[string]attr.Value{
					"acl": types.ObjectValueMust(aclTypes, map[string]attr.Value{
						"enabled": types.BoolValue(true),
						"allowed_cidrs": types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("cidr1"),
						}),
					}),
					"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
						"enabled":     types.BoolValue(true),
						"instance_id": types.StringValue("aid"),
					}),
					"dns": types.ObjectValueMust(dnsTypes, map[string]attr.Value{
						"enabled": types.BoolValue(true),
						"zones": types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("foo.onstackit.cloud"),
						}),
					}),
				}),
				Region: types.StringValue(testRegion),
			},
			true,
		}, /*
			{
				"empty_network",
				types.ObjectNull(extensionsTypes),
				types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
				&ske.Cluster{
					Name:    utils.Ptr("name"),
					Network: &ske.Network{},
				},
				testRegion,
				Model{
					Id:                        types.StringValue("pid,region,name"),
					ProjectId:                 types.StringValue("pid"),
					Name:                      types.StringValue("name"),
					KubernetesVersion:         types.StringNull(),
					AllowPrivilegedContainers: types.BoolNull(),
					NodePools:                 types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
					Maintenance:               types.ObjectNull(maintenanceTypes),
					Network:                   types.ObjectNull(networkTypes),
					Hibernations:              types.ListNull(types.ObjectType{AttrTypes: hibernationTypes}),
					Extensions:                types.ObjectNull(extensionsTypes),
					EgressAddressRanges:       types.ListNull(types.StringType),
					PodAddressRanges:          types.ListNull(types.StringType),
					Region:                    types.StringValue(testRegion),
				},
				true,
			},
			{
				"extensions_mixed_values",
				types.ObjectNull(extensionsTypes),
				types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
				&ske.Cluster{
					Extensions: &ske.Extension{
						Acl: &ske.ACL{
							AllowedCidrs: nil,
							Enabled:      utils.Ptr(true),
						},
						Observability: &ske.Observability{
							InstanceId: nil,
							Enabled:    utils.Ptr(true),
						},
						Dns: &ske.DNS{
							Zones:   nil,
							Enabled: utils.Ptr(true),
						},
					},
					Name: utils.Ptr("name"),
				},
				testRegion,
				Model{
					Id:                        types.StringValue("pid,region,name"),
					ProjectId:                 types.StringValue("pid"),
					Name:                      types.StringValue("name"),
					KubernetesVersion:         types.StringNull(),
					AllowPrivilegedContainers: types.BoolNull(),
					NodePools:                 types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
					Maintenance:               types.ObjectNull(maintenanceTypes),
					Hibernations:              types.ListNull(types.ObjectType{AttrTypes: hibernationTypes}),
					EgressAddressRanges:       types.ListNull(types.StringType),
					PodAddressRanges:          types.ListNull(types.StringType),
					Extensions: types.ObjectValueMust(extensionsTypes, map[string]attr.Value{
						"acl": types.ObjectValueMust(aclTypes, map[string]attr.Value{
							"enabled":       types.BoolValue(true),
							"allowed_cidrs": types.ListNull(types.StringType),
						}),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"enabled":     types.BoolValue(true),
							"instance_id": types.StringNull(),
						}),
						"dns": types.ObjectValueMust(dnsTypes, map[string]attr.Value{
							"enabled": types.BoolValue(true),
							"zones":   types.ListNull(types.StringType),
						}),
					}),
					Region: types.StringValue(testRegion),
				},
				true,
			},
			{
				"extensions_disabled",
				types.ObjectValueMust(extensionsTypes, map[string]attr.Value{
					"acl": types.ObjectValueMust(aclTypes, map[string]attr.Value{
						"enabled":       types.BoolValue(false),
						"allowed_cidrs": types.ListNull(types.StringType),
					}),
					"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
						"enabled":     types.BoolValue(false),
						"instance_id": types.StringNull(),
					}),
					"dns": types.ObjectValueMust(dnsTypes, map[string]attr.Value{
						"enabled": types.BoolValue(false),
						"zones":   types.ListNull(types.StringType),
					}),
				}),
				types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
				&ske.Cluster{
					Extensions: &ske.Extension{},
					Name:       utils.Ptr("name"),
				},
				testRegion,
				Model{
					Id:                        types.StringValue("pid,region,name"),
					ProjectId:                 types.StringValue("pid"),
					Name:                      types.StringValue("name"),
					KubernetesVersion:         types.StringNull(),
					AllowPrivilegedContainers: types.BoolNull(),
					NodePools:                 types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
					Maintenance:               types.ObjectNull(maintenanceTypes),
					Hibernations:              types.ListNull(types.ObjectType{AttrTypes: hibernationTypes}),
					EgressAddressRanges:       types.ListNull(types.StringType),
					PodAddressRanges:          types.ListNull(types.StringType),
					Extensions: types.ObjectValueMust(extensionsTypes, map[string]attr.Value{
						"acl": types.ObjectValueMust(aclTypes, map[string]attr.Value{
							"enabled":       types.BoolValue(false),
							"allowed_cidrs": types.ListNull(types.StringType),
						}),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"enabled":     types.BoolValue(false),
							"instance_id": types.StringNull(),
						}),
						"dns": types.ObjectValueMust(dnsTypes, map[string]attr.Value{
							"enabled": types.BoolValue(false),
							"zones":   types.ListNull(types.StringType),
						}),
					}),
					Region: types.StringValue(testRegion),
				},
				true,
			},
			{
				"extensions_only_observability_disabled",
				types.ObjectValueMust(extensionsTypes, map[string]attr.Value{
					"acl": types.ObjectValueMust(aclTypes, map[string]attr.Value{
						"enabled": types.BoolValue(true),
						"allowed_cidrs": types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("cidr1"),
						}),
					}),
					"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
						"enabled":     types.BoolValue(false),
						"instance_id": types.StringValue("id"),
					}),
					"dns": types.ObjectValueMust(dnsTypes, map[string]attr.Value{
						"enabled": types.BoolValue(true),
						"zones":   types.ListNull(types.StringType),
					}),
				}),
				types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
				&ske.Cluster{
					Extensions: &ske.Extension{
						Acl: &ske.ACL{
							AllowedCidrs: &[]string{"cidr1"},
							Enabled:      utils.Ptr(true),
						},
						Dns: &ske.DNS{
							Zones:   nil,
							Enabled: utils.Ptr(true),
						},
					},
					Name: utils.Ptr("name"),
				},
				testRegion,
				Model{
					Id:                        types.StringValue("pid,region,name"),
					ProjectId:                 types.StringValue("pid"),
					Name:                      types.StringValue("name"),
					KubernetesVersion:         types.StringNull(),
					AllowPrivilegedContainers: types.BoolNull(),
					NodePools:                 types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
					Maintenance:               types.ObjectNull(maintenanceTypes),
					Hibernations:              types.ListNull(types.ObjectType{AttrTypes: hibernationTypes}),
					EgressAddressRanges:       types.ListNull(types.StringType),
					PodAddressRanges:          types.ListNull(types.StringType),
					Extensions: types.ObjectValueMust(extensionsTypes, map[string]attr.Value{
						"acl": types.ObjectValueMust(aclTypes, map[string]attr.Value{
							"enabled": types.BoolValue(true),
							"allowed_cidrs": types.ListValueMust(types.StringType, []attr.Value{
								types.StringValue("cidr1"),
							}),
						}),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"enabled":     types.BoolValue(false),
							"instance_id": types.StringValue("id"),
						}),
						"dns": types.ObjectValueMust(dnsTypes, map[string]attr.Value{
							"enabled": types.BoolValue(true),
							"zones":   types.ListNull(types.StringType),
						}),
					}),
					Region: types.StringValue(testRegion),
				},
				true,
			},
			{
				"extensions_not_set",
				types.ObjectNull(extensionsTypes),
				types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
				&ske.Cluster{
					Extensions: &ske.Extension{},
					Name:       utils.Ptr("name"),
				},
				testRegion,
				Model{
					Id:                        types.StringValue("pid,region,name"),
					ProjectId:                 types.StringValue("pid"),
					Name:                      types.StringValue("name"),
					KubernetesVersion:         types.StringNull(),
					AllowPrivilegedContainers: types.BoolNull(),
					NodePools:                 types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
					Maintenance:               types.ObjectNull(maintenanceTypes),
					Hibernations:              types.ListNull(types.ObjectType{AttrTypes: hibernationTypes}),
					Extensions:                types.ObjectNull(extensionsTypes),
					EgressAddressRanges:       types.ListNull(types.StringType),
					PodAddressRanges:          types.ListNull(types.StringType),
					Region:                    types.StringValue(testRegion),
				},
				true,
			},
			{
				"nil_taints_when_empty_list_on_state",
				types.ObjectNull(extensionsTypes),
				types.ListValueMust(
					types.ObjectType{AttrTypes: nodePoolTypes},
					[]attr.Value{
						types.ObjectValueMust(
							nodePoolTypes,
							map[string]attr.Value{
								"name":            types.StringValue("node"),
								"machine_type":    types.StringValue("B"),
								"os_name":         types.StringValue("os"),
								"os_version":      types.StringNull(),
								"os_version_min":  types.StringNull(),
								"os_version_used": types.StringValue("os-ver"),
								"minimum":         types.Int64Value(1),
								"maximum":         types.Int64Value(5),
								"max_surge":       types.Int64Value(3),
								"max_unavailable": types.Int64Null(),
								"volume_type":     types.StringValue("type"),
								"volume_size":     types.Int64Value(3),
								"labels": types.MapValueMust(
									types.StringType,
									map[string]attr.Value{
										"k": types.StringValue("v"),
									},
								),
								"taints": types.ListValueMust(types.ObjectType{AttrTypes: taintTypes}, []attr.Value{}),
								"cri":    types.StringValue(string(ske.CRINAME_DOCKER)),
								"availability_zones": types.ListValueMust(
									types.StringType,
									[]attr.Value{
										types.StringValue("z1"),
										types.StringValue("z2"),
									},
								),
								"allow_system_components": types.BoolValue(true),
							},
						),
					},
				),
				&ske.Cluster{
					Extensions: &ske.Extension{
						Acl: &ske.ACL{
							AllowedCidrs: &[]string{"cidr1"},
							Enabled:      utils.Ptr(true),
						},
						Observability: &ske.Observability{
							InstanceId: utils.Ptr("aid"),
							Enabled:    utils.Ptr(true),
						},
						Dns: &ske.DNS{
							Zones:   &[]string{"zone1"},
							Enabled: utils.Ptr(true),
						},
					},
					Hibernation: &ske.Hibernation{
						Schedules: &[]ske.HibernationSchedule{
							{
								End:      utils.Ptr("2"),
								Start:    utils.Ptr("1"),
								Timezone: utils.Ptr("CET"),
							},
						},
					},
					Kubernetes: &ske.Kubernetes{
						Version: utils.Ptr("1.2.3"),
					},
					Maintenance: &ske.Maintenance{
						AutoUpdate: &ske.MaintenanceAutoUpdate{
							KubernetesVersion:   utils.Ptr(true),
							MachineImageVersion: utils.Ptr(true),
						},
						TimeWindow: &ske.TimeWindow{
							Start: utils.Ptr(time.Date(0, 1, 2, 3, 4, 5, 6, time.FixedZone("UTC+6:00", 6*60*60))),
							End:   utils.Ptr(time.Date(10, 11, 12, 13, 14, 15, 0, time.UTC)),
						},
					},
					Network: &ske.Network{
						Id: utils.Ptr("nid"),
					},
					Name: utils.Ptr("name"),
					Nodepools: &[]ske.Nodepool{
						{
							AvailabilityZones: &[]string{"z1", "z2"},
							Cri: &ske.CRI{
								Name: ske.CRINAME_DOCKER.Ptr(),
							},
							Labels: &map[string]string{"k": "v"},
							Machine: &ske.Machine{
								Image: &ske.Image{
									Name:    utils.Ptr("os"),
									Version: utils.Ptr("os-ver"),
								},
								Type: utils.Ptr("B"),
							},
							MaxSurge:       utils.Ptr(int64(3)),
							MaxUnavailable: nil,
							Maximum:        utils.Ptr(int64(5)),
							Minimum:        utils.Ptr(int64(1)),
							Name:           utils.Ptr("node"),
							Taints:         nil,
							Volume: &ske.Volume{
								Size: utils.Ptr(int64(3)),
								Type: utils.Ptr("type"),
							},
						},
					},
					Status: &ske.ClusterStatus{
						Aggregated: &cs,
						Error:      nil,
						Hibernated: nil,
					},
				},
				testRegion,
				Model{
					Id:                        types.StringValue("pid,region,name"),
					ProjectId:                 types.StringValue("pid"),
					Name:                      types.StringValue("name"),
					KubernetesVersion:         types.StringNull(),
					KubernetesVersionUsed:     types.StringValue("1.2.3"),
					AllowPrivilegedContainers: types.BoolValue(true),
					EgressAddressRanges:       types.ListNull(types.StringType),
					PodAddressRanges:          types.ListNull(types.StringType),
					NodePools: types.ListValueMust(
						types.ObjectType{AttrTypes: nodePoolTypes},
						[]attr.Value{
							types.ObjectValueMust(
								nodePoolTypes,
								map[string]attr.Value{
									"name":            types.StringValue("node"),
									"machine_type":    types.StringValue("B"),
									"os_name":         types.StringValue("os"),
									"os_version":      types.StringNull(),
									"os_version_min":  types.StringNull(),
									"os_version_used": types.StringValue("os-ver"),
									"minimum":         types.Int64Value(1),
									"maximum":         types.Int64Value(5),
									"max_surge":       types.Int64Value(3),
									"max_unavailable": types.Int64Null(),
									"volume_type":     types.StringValue("type"),
									"volume_size":     types.Int64Value(3),
									"labels": types.MapValueMust(
										types.StringType,
										map[string]attr.Value{
											"k": types.StringValue("v"),
										},
									),
									"taints": types.ListValueMust(types.ObjectType{AttrTypes: taintTypes}, []attr.Value{}),
									"cri":    types.StringValue(string(ske.CRINAME_DOCKER)),
									"availability_zones": types.ListValueMust(
										types.StringType,
										[]attr.Value{
											types.StringValue("z1"),
											types.StringValue("z2"),
										},
									),
									"allow_system_components": types.BoolNull(),
								},
							),
						},
					),
					Maintenance: types.ObjectValueMust(maintenanceTypes, map[string]attr.Value{
						"enable_kubernetes_version_updates":    types.BoolValue(true),
						"enable_machine_image_version_updates": types.BoolValue(true),
						"start":                                types.StringValue("03:04:05+06:00"),
						"end":                                  types.StringValue("13:14:15Z"),
					}),
					Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"id": types.StringValue("nid"),
					}),
					Hibernations: types.ListValueMust(
						types.ObjectType{AttrTypes: hibernationTypes},
						[]attr.Value{
							types.ObjectValueMust(
								hibernationTypes,
								map[string]attr.Value{
									"start":    types.StringValue("1"),
									"end":      types.StringValue("2"),
									"timezone": types.StringValue("CET"),
								},
							),
						},
					),
					Extensions: types.ObjectValueMust(extensionsTypes, map[string]attr.Value{
						"acl": types.ObjectValueMust(aclTypes, map[string]attr.Value{
							"enabled": types.BoolValue(true),
							"allowed_cidrs": types.ListValueMust(types.StringType, []attr.Value{
								types.StringValue("cidr1"),
							}),
						}),
						"observability": types.ObjectValueMust(observabilityTypes, map[string]attr.Value{
							"enabled":     types.BoolValue(true),
							"instance_id": types.StringValue("aid"),
						}),
						"dns": types.ObjectValueMust(dnsTypes, map[string]attr.Value{
							"enabled": types.BoolValue(true),
							"zones": types.ListValueMust(types.StringType, []attr.Value{
								types.StringValue("zone1"),
							}),
						}),
					}),
					Region: types.StringValue(testRegion),
				},
				true,
			},
			{
				"nil_response",
				types.ObjectNull(extensionsTypes),
				types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
				nil,
				testRegion,
				Model{},
				false,
			},
			{
				"no_resource_id",
				types.ObjectNull(extensionsTypes),
				types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes}),
				&ske.Cluster{},
				testRegion,
				Model{},
				false,
			},*/
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				Extensions: tt.stateExtensions,
				NodePools:  tt.stateNodePools,
			}
			err := mapFields(context.Background(), tt.input, state, tt.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestLatestMatchingKubernetesVersion(t *testing.T) {
	tests := []struct {
		description                  string
		availableVersions            []ske.KubernetesVersion
		kubernetesVersionMin         *string
		currentKubernetesVersion     *string
		expectedVersionUsed          *string
		expectedHasDeprecatedVersion bool
		expectedWarning              bool
		isValid                      bool
	}{
		{
			"available_version",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.1"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.2"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.20.1"),
			nil,
			utils.Ptr("1.20.1"),
			false,
			false,
			true,
		},
		{
			"available_version_zero_patch",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.1"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.2"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.20.0"),
			nil,
			utils.Ptr("1.20.0"),
			false,
			false,
			true,
		},
		{
			"available_version_with_no_provided_patch",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.1"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.2"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.20"),
			nil,
			utils.Ptr("1.20.2"),
			false,
			false,
			true,
		},
		{
			"available_version_with_higher_preview_patch_not_selected",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.1"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.2"),
					State:   utils.Ptr(VersionStatePreview),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.20"),
			nil,
			utils.Ptr("1.20.1"),
			false,
			false,
			true,
		},
		{
			"available_version_no_provided_patch_2",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.20"),
			nil,
			utils.Ptr("1.20.0"),
			false,
			false,
			true,
		},
		{
			"deprecated_version",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateDeprecated),
				},
			},
			utils.Ptr("1.19"),
			nil,
			utils.Ptr("1.19.0"),
			true,
			false,
			true,
		},
		{
			"preview_version",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStatePreview),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.20.0"),
			nil,
			utils.Ptr("1.20.0"),
			false,
			true,
			true,
		},
		{
			"nil_provided_version_get_latest",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			nil,
			nil,
			utils.Ptr("1.20.0"),
			false,
			false,
			true,
		},
		{
			"nil_provided_version_use_current",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			nil,
			utils.Ptr("1.19.0"),
			utils.Ptr("1.19.0"),
			false,
			false,
			true,
		},
		{
			"update_lower_min_provided",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.19"),
			utils.Ptr("1.20.0"),
			utils.Ptr("1.20.0"),
			false,
			false,
			true,
		},
		{
			"update_lower_min_provided_deprecated_version",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.21.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateDeprecated),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateDeprecated),
				},
			},
			utils.Ptr("1.19"),
			utils.Ptr("1.20.0"),
			utils.Ptr("1.20.0"),
			true,
			false,
			true,
		},
		{
			"update_matching_min_provided",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.20"),
			utils.Ptr("1.20.0"),
			utils.Ptr("1.20.0"),
			false,
			false,
			true,
		},
		{
			"update_higher_min_provided",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.20"),
			utils.Ptr("1.19.0"),
			utils.Ptr("1.20.0"),
			false,
			false,
			true,
		},
		{
			"no_matching_available_versions",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.21"),
			nil,
			nil,
			false,
			false,
			false,
		},
		{
			"no_matching_available_versions_patch",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.21.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.21.1"),
			nil,
			nil,
			false,
			false,
			false,
		},
		{
			"no_matching_available_versions_patch_2",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.21.2"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr("1.21.1"),
			nil,
			nil,
			false,
			false,
			false,
		},
		{
			"no_matching_available_versions_patch_current",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.21.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			nil,
			utils.Ptr("1.21.1"),
			nil,
			false,
			false,
			false,
		},
		{
			"no_matching_available_versions_patch_2_current",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.21.2"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			nil,
			utils.Ptr("1.21.1"),
			nil,
			false,
			false,
			false,
		},
		{
			"no_available_version",
			[]ske.KubernetesVersion{},
			utils.Ptr("1.20"),
			nil,
			nil,
			false,
			false,
			false,
		},
		{
			"nil_available_version",
			nil,
			utils.Ptr("1.20"),
			nil,
			nil,
			false,
			false,
			false,
		},
		{
			"empty_provided_version",
			[]ske.KubernetesVersion{
				{
					Version: utils.Ptr("1.20.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
				{
					Version: utils.Ptr("1.19.0"),
					State:   utils.Ptr(VersionStateSupported),
				},
			},
			utils.Ptr(""),
			nil,
			nil,
			false,
			false,
			false,
		},
		{
			description: "minimum_version_without_patch_version_results_in_latest_supported_version,even_if_preview_is_available",
			availableVersions: []ske.KubernetesVersion{
				{Version: utils.Ptr("1.20.0"), State: utils.Ptr(VersionStateSupported)},
				{Version: utils.Ptr("1.20.1"), State: utils.Ptr(VersionStateSupported)},
				{Version: utils.Ptr("1.20.2"), State: utils.Ptr(VersionStateSupported)},
				{Version: utils.Ptr("1.20.3"), State: utils.Ptr(VersionStateSupported)},
				{Version: utils.Ptr("1.20.4"), State: utils.Ptr(VersionStatePreview)},
			},
			kubernetesVersionMin:         utils.Ptr("1.20"),
			currentKubernetesVersion:     nil,
			expectedVersionUsed:          utils.Ptr("1.20.3"),
			expectedHasDeprecatedVersion: false,
			expectedWarning:              false,
			isValid:                      true,
		},
		{
			description: "use_preview_when_no_supported_release_is_available",
			availableVersions: []ske.KubernetesVersion{
				{Version: utils.Ptr("1.19.5"), State: utils.Ptr(VersionStateSupported)},
				{Version: utils.Ptr("1.19.6"), State: utils.Ptr(VersionStateSupported)},
				{Version: utils.Ptr("1.19.7"), State: utils.Ptr(VersionStateSupported)},
				{Version: utils.Ptr("1.20.0"), State: utils.Ptr(VersionStateDeprecated)},
				{Version: utils.Ptr("1.20.1"), State: utils.Ptr(VersionStatePreview)},
			},
			kubernetesVersionMin:         utils.Ptr("1.20"),
			currentKubernetesVersion:     nil,
			expectedVersionUsed:          utils.Ptr("1.20.1"),
			expectedHasDeprecatedVersion: false,
			expectedWarning:              true,
			isValid:                      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var diags diag.Diagnostics
			versionUsed, hasDeprecatedVersion, err := latestMatchingKubernetesVersion(tt.availableVersions, tt.kubernetesVersionMin, tt.currentKubernetesVersion, &diags)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				if *versionUsed != *tt.expectedVersionUsed {
					t.Fatalf("Used version does not match: expecting %s, got %s", *tt.expectedVersionUsed, *versionUsed)
				}
				if tt.expectedHasDeprecatedVersion != hasDeprecatedVersion {
					t.Fatalf("hasDeprecatedVersion flag is wrong: expecting %t, got %t", tt.expectedHasDeprecatedVersion, hasDeprecatedVersion)
				}
			}
			if hasWarnings := len(diags.Warnings()) > 0; tt.expectedWarning != hasWarnings {
				t.Fatalf("Emitted warnings do not match. Expected %t but got %t", tt.expectedWarning, hasWarnings)
			}
		})
	}
}

func TestLatestMatchingMachineVersion(t *testing.T) {
	tests := []struct {
		description                  string
		availableVersions            []ske.MachineImage
		machineVersionMin            *string
		machineName                  string
		currentMachineImage          *ske.Image
		expectedVersionUsed          *string
		expectedHasDeprecatedVersion bool
		isValid                      bool
	}{
		{
			"available_version",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.20.1"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.20.2"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.20.1"),
			"foo",
			nil,
			utils.Ptr("1.20.1"),
			false,
			true,
		},
		{
			"available_version_zero_patch",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.20.1"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.20.2"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.20.0"),
			"foo",
			nil,
			utils.Ptr("1.20.0"),
			false,
			true,
		},
		{
			"available_version_with_no_provided_patch",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.20.1"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.20.2"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.20"),
			"foo",
			nil,
			utils.Ptr("1.20.2"),
			false,
			true,
		},
		{
			"available_version_with_higher_preview_patch_not_selected",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.20.1"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.20.2"),
							State:   utils.Ptr(VersionStatePreview),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.20"),
			"foo",
			nil,
			utils.Ptr("1.20.1"),
			false,
			true,
		},
		{
			"available_version_with_no_provided_patch_2",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.20"),
			"foo",
			nil,
			utils.Ptr("1.20.0"),
			false,
			true,
		},
		{
			"deprecated_version",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateDeprecated),
						},
					},
				},
			},
			utils.Ptr("1.19"),
			"foo",
			nil,
			utils.Ptr("1.19.0"),
			true,
			true,
		},
		{
			"preview_version_selected",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStatePreview),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateDeprecated),
						},
					},
				},
			},
			utils.Ptr("1.20.0"),
			"foo",
			nil,
			utils.Ptr("1.20.0"),
			false,
			true,
		},
		{
			"nil_provided_version_get_latest",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			nil,
			"foo",
			nil,
			utils.Ptr("1.20.0"),
			false,
			true,
		},
		{
			"nil_provided_version_use_current",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			nil,
			"foo",
			&ske.Image{
				Name:    utils.Ptr("foo"),
				Version: utils.Ptr("1.19.0"),
			},
			utils.Ptr("1.19.0"),
			false,
			true,
		},
		{
			"nil_provided_version_os_image_update_get_latest",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			nil,
			"foo",
			&ske.Image{
				Name:    utils.Ptr("bar"),
				Version: utils.Ptr("1.19.0"),
			},
			utils.Ptr("1.20.0"),
			false,
			true,
		},
		{
			"update_lower_min_provided",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.19"),
			"foo",
			&ske.Image{
				Name:    utils.Ptr("foo"),
				Version: utils.Ptr("1.20.0"),
			},
			utils.Ptr("1.20.0"),
			false,
			true,
		},
		{
			"update_lower_min_provided_deprecated_version",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.21.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateDeprecated),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.19"),
			"foo",
			&ske.Image{
				Name:    utils.Ptr("foo"),
				Version: utils.Ptr("1.20.0"),
			},
			utils.Ptr("1.20.0"),
			true,
			true,
		},
		{
			"update_higher_min_provided",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.20"),
			"foo",
			&ske.Image{
				Name:    utils.Ptr("foo"),
				Version: utils.Ptr("1.19.0"),
			},
			utils.Ptr("1.20.0"),
			false,
			true,
		},
		{
			"no_matching_available_versions",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
						{
							Version: utils.Ptr("1.19.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.21"),
			"foo",
			nil,
			nil,
			false,
			false,
		},
		{
			"no_available_versions",
			[]ske.MachineImage{
				{
					Name:     utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{},
				},
			},
			utils.Ptr("1.20"),
			"foo",
			nil,
			nil,
			false,
			false,
		},
		{
			"nil_available_versions",
			[]ske.MachineImage{
				{
					Name:     utils.Ptr("foo"),
					Versions: nil,
				},
			},
			utils.Ptr("1.20"),
			"foo",
			nil,
			nil,
			false,
			false,
		},
		{
			"nil_name",
			[]ske.MachineImage{
				{
					Name: nil,
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.20"),
			"foo",
			nil,
			nil,
			false,
			false,
		},
		{
			"name_not_available",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("bar"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr("1.20"),
			"foo",
			nil,
			nil,
			false,
			false,
		},
		{
			"empty_provided_version",
			[]ske.MachineImage{
				{
					Name: utils.Ptr("foo"),
					Versions: &[]ske.MachineImageVersion{
						{
							Version: utils.Ptr("1.20.0"),
							State:   utils.Ptr(VersionStateSupported),
						},
					},
				},
			},
			utils.Ptr(""),
			"foo",
			nil,
			nil,
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			versionUsed, hasDeprecatedVersion, err := latestMatchingMachineVersion(tt.availableVersions, tt.machineVersionMin, tt.machineName, tt.currentMachineImage)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				if *versionUsed != *tt.expectedVersionUsed {
					t.Fatalf("Used version does not match: expecting %s, got %s", *tt.expectedVersionUsed, *versionUsed)
				}
				if tt.expectedHasDeprecatedVersion != hasDeprecatedVersion {
					t.Fatalf("hasDeprecatedVersion flag is wrong: expecting %t, got %t", tt.expectedHasDeprecatedVersion, hasDeprecatedVersion)
				}
			}
		})
	}
}

func TestGetMaintenanceTimes(t *testing.T) {
	tests := []struct {
		description   string
		startAPI      time.Time
		startTF       *string
		endAPI        time.Time
		endTF         *string
		isValid       bool
		startExpected string
		endExpected   string
	}{
		{
			description:   "base",
			startAPI:      time.Date(1, 2, 3, 4, 5, 6, 7, time.FixedZone("UTC+7:08", 7*60*60+8*60)),
			endAPI:        time.Date(11, 12, 13, 14, 15, 16, 17, time.FixedZone("UTC+17:18", 17*60*60+18*60)),
			isValid:       true,
			startExpected: "04:05:06+07:08",
			endExpected:   "14:15:16+17:18",
		},
		{
			description:   "base_utc",
			startAPI:      time.Date(1, 2, 3, 4, 5, 6, 0, time.UTC),
			endAPI:        time.Date(11, 12, 13, 14, 15, 16, 0, time.UTC),
			isValid:       true,
			startExpected: "04:05:06Z",
			endExpected:   "14:15:16Z",
		},
		{
			description:   "tf_state_filled_in_1",
			startAPI:      time.Date(1, 2, 3, 4, 5, 6, 7, time.FixedZone("UTC+7:08", 7*60*60+8*60)),
			startTF:       utils.Ptr("04:05:06+07:08"),
			endAPI:        time.Date(11, 12, 13, 14, 15, 16, 17, time.FixedZone("UTC+17:18", 17*60*60+18*60)),
			endTF:         utils.Ptr("14:15:16+17:18"),
			isValid:       true,
			startExpected: "04:05:06+07:08",
			endExpected:   "14:15:16+17:18",
		},
		{
			description:   "tf_state_filled_in_2",
			startAPI:      time.Date(1, 2, 3, 4, 5, 6, 0, time.UTC),
			startTF:       utils.Ptr("04:05:06+00:00"),
			endAPI:        time.Date(11, 12, 13, 14, 15, 16, 0, time.UTC),
			endTF:         utils.Ptr("14:15:16+00:00"),
			isValid:       true,
			startExpected: "04:05:06+00:00",
			endExpected:   "14:15:16+00:00",
		},
		{
			description:   "tf_state_filled_in_3",
			startAPI:      time.Date(1, 2, 3, 4, 5, 6, 0, time.UTC),
			startTF:       utils.Ptr("04:05:06Z"),
			endAPI:        time.Date(11, 12, 13, 14, 15, 16, 0, time.UTC),
			endTF:         utils.Ptr("14:15:16Z"),
			isValid:       true,
			startExpected: "04:05:06Z",
			endExpected:   "14:15:16Z",
		},
		{
			description:   "api_takes_precedence_if_different_1",
			startAPI:      time.Date(1, 2, 3, 4, 5, 6, 7, time.FixedZone("UTC+7:08", 7*60*60+8*60)),
			startTF:       utils.Ptr("00:00:00+07:08"),
			endAPI:        time.Date(11, 12, 13, 14, 15, 16, 17, time.FixedZone("UTC+17:18", 17*60*60+18*60)),
			endTF:         utils.Ptr("14:15:16+17:18"),
			isValid:       true,
			startExpected: "04:05:06+07:08",
			endExpected:   "14:15:16+17:18",
		},
		{
			description:   "api_takes_precedence_if_different_2",
			startAPI:      time.Date(1, 2, 3, 4, 5, 6, 7, time.FixedZone("UTC+7:08", 7*60*60+8*60)),
			startTF:       utils.Ptr("04:05:06+07:08"),
			endAPI:        time.Date(11, 12, 13, 14, 15, 16, 17, time.FixedZone("UTC+17:18", 17*60*60+18*60)),
			endTF:         utils.Ptr("00:00:00+17:18"),
			isValid:       true,
			startExpected: "04:05:06+07:08",
			endExpected:   "14:15:16+17:18",
		},
		{
			description:   "api_takes_precedence_if_different_3",
			startAPI:      time.Date(1, 2, 3, 4, 5, 6, 7, time.FixedZone("UTC+7:08", 7*60*60+8*60)),
			startTF:       utils.Ptr("04:05:06Z"),
			endAPI:        time.Date(11, 12, 13, 14, 15, 16, 17, time.FixedZone("UTC+17:18", 17*60*60+18*60)),
			endTF:         utils.Ptr("14:15:16+17:18"),
			isValid:       true,
			startExpected: "04:05:06+07:08",
			endExpected:   "14:15:16+17:18",
		},
		{
			description:   "api_takes_precedence_if_different_3",
			startAPI:      time.Date(1, 2, 3, 4, 5, 6, 7, time.FixedZone("UTC+7:08", 7*60*60+8*60)),
			startTF:       utils.Ptr("04:05:06+07:08"),
			endAPI:        time.Date(11, 12, 13, 14, 15, 16, 17, time.FixedZone("UTC+17:18", 17*60*60+18*60)),
			endTF:         utils.Ptr("14:15:16Z"),
			isValid:       true,
			startExpected: "04:05:06+07:08",
			endExpected:   "14:15:16+17:18",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			apiResponse := &ske.Cluster{
				Maintenance: &ske.Maintenance{
					TimeWindow: &ske.TimeWindow{
						Start: utils.Ptr(tt.startAPI),
						End:   utils.Ptr(tt.endAPI),
					},
				},
			}

			maintenanceValues := map[string]attr.Value{
				"enable_kubernetes_version_updates":    types.BoolNull(),
				"enable_machine_image_version_updates": types.BoolNull(),
				"start":                                types.StringPointerValue(tt.startTF),
				"end":                                  types.StringPointerValue(tt.endTF),
			}
			maintenanceObject, diags := types.ObjectValue(maintenanceTypes, maintenanceValues)
			if diags.HasError() {
				t.Fatalf("failed to create flavor: %v", core.DiagsToError(diags))
			}
			tfState := &Model{
				Maintenance: maintenanceObject,
			}

			start, end, err := getMaintenanceTimes(context.Background(), apiResponse, tfState)

			if err != nil {
				if tt.isValid {
					t.Errorf("getMaintenanceTimes failed on valid input: %v", err)
				}
				return
			}
			if !tt.isValid {
				t.Fatalf("getMaintenanceTimes didn't fail on invalid input")
			}
			if tt.startExpected != start {
				t.Errorf("expected start '%s', got '%s'", tt.startExpected, start)
			}
			if tt.endExpected != end {
				t.Errorf("expected end '%s', got '%s'", tt.endExpected, end)
			}
		})
	}
}

func TestCheckAllowPrivilegedContainers(t *testing.T) {
	tests := []struct {
		description              string
		kubernetesVersion        *string
		allowPrivilegeContainers *bool
		isValid                  bool
	}{
		{
			description:              "null_version_1_flag_deprecated",
			kubernetesVersion:        nil,
			allowPrivilegeContainers: nil,
			isValid:                  true,
		},
		{
			description:              "null_version_2_flag_deprecated",
			kubernetesVersion:        nil,
			allowPrivilegeContainers: utils.Ptr(false),
			isValid:                  false,
		},
		{
			description:              "flag_required_1",
			kubernetesVersion:        utils.Ptr("0.999.999"),
			allowPrivilegeContainers: nil,
			isValid:                  false,
		},
		{
			description:              "flag_required_2",
			kubernetesVersion:        utils.Ptr("0.999.999"),
			allowPrivilegeContainers: utils.Ptr(false),
			isValid:                  true,
		},
		{
			description:              "flag_required_3",
			kubernetesVersion:        utils.Ptr("1.24.999"),
			allowPrivilegeContainers: nil,
			isValid:                  false,
		},
		{
			description:              "flag_required_4",
			kubernetesVersion:        utils.Ptr("1.24.999"),
			allowPrivilegeContainers: utils.Ptr(false),
			isValid:                  true,
		},
		{
			description:              "flag_deprecated_1",
			kubernetesVersion:        utils.Ptr("1.25"),
			allowPrivilegeContainers: nil,
			isValid:                  true,
		},
		{
			description:              "flag_deprecated_2",
			kubernetesVersion:        utils.Ptr("1.25"),
			allowPrivilegeContainers: utils.Ptr(false),
			isValid:                  false,
		},
		{
			description:              "flag_deprecated_3",
			kubernetesVersion:        utils.Ptr("2.0.0"),
			allowPrivilegeContainers: nil,
			isValid:                  true,
		},
		{
			description:              "flag_deprecated_4",
			kubernetesVersion:        utils.Ptr("2.0.0"),
			allowPrivilegeContainers: utils.Ptr(false),
			isValid:                  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			diags := checkAllowPrivilegedContainers(
				types.BoolPointerValue(tt.allowPrivilegeContainers),
				types.StringPointerValue(tt.kubernetesVersion),
			)

			if tt.isValid && diags.HasError() {
				t.Errorf("checkAllowPrivilegedContainers failed on valid input: %v", core.DiagsToError(diags))
			}
			if !tt.isValid && !diags.HasError() {
				t.Errorf("checkAllowPrivilegedContainers didn't fail on valid input")
			}
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	tests := []struct {
		description               string
		mockedResp                *ske.Cluster
		expectedKubernetesVersion *string
		expectedMachineImages     map[string]*ske.Image
		getClusterFails           bool
	}{
		{
			"ok",
			&ske.Cluster{
				Kubernetes: &ske.Kubernetes{
					Version: utils.Ptr("v1.0.0"),
				},
				Nodepools: &[]ske.Nodepool{
					{
						Name: utils.Ptr("foo"),
						Machine: &ske.Machine{
							Image: &ske.Image{
								Name:    utils.Ptr("foo"),
								Version: utils.Ptr("v1.0.0"),
							},
						},
					},
					{
						Name: utils.Ptr("bar"),
						Machine: &ske.Machine{
							Image: &ske.Image{
								Name:    utils.Ptr("bar"),
								Version: utils.Ptr("v2.0.0"),
							},
						},
					},
				},
			},
			utils.Ptr("v1.0.0"),
			map[string]*ske.Image{
				"foo": {
					Name:    utils.Ptr("foo"),
					Version: utils.Ptr("v1.0.0"),
				},
				"bar": {
					Name:    utils.Ptr("bar"),
					Version: utils.Ptr("v2.0.0"),
				},
			},
			false,
		},
		{
			"get fails",
			nil,
			nil,
			nil,
			true,
		},
		{
			"nil kubernetes",
			&ske.Cluster{
				Kubernetes: nil,
			},
			nil,
			nil,
			false,
		},
		{
			"nil kubernetes version",
			&ske.Cluster{
				Kubernetes: &ske.Kubernetes{
					Version: nil,
				},
			},
			nil,
			nil,
			false,
		},
		{
			"nil nodepools",
			&ske.Cluster{
				Kubernetes: &ske.Kubernetes{
					Version: utils.Ptr("v1.0.0"),
				},
				Nodepools: nil,
			},
			utils.Ptr("v1.0.0"),
			nil,
			false,
		},
		{
			"nil nodepools machine",
			&ske.Cluster{
				Kubernetes: &ske.Kubernetes{
					Version: utils.Ptr("v1.0.0"),
				},
				Nodepools: &[]ske.Nodepool{
					{
						Name:    utils.Ptr("foo"),
						Machine: nil,
					},
				},
			},
			utils.Ptr("v1.0.0"),
			map[string]*ske.Image{},
			false,
		},
		{
			"nil nodepools machine image",
			&ske.Cluster{
				Kubernetes: &ske.Kubernetes{
					Version: utils.Ptr("v1.0.0"),
				},
				Nodepools: &[]ske.Nodepool{
					{
						Name: utils.Ptr("foo"),
						Machine: &ske.Machine{
							Image: nil,
						},
					},
				},
			},
			utils.Ptr("v1.0.0"),
			map[string]*ske.Image{},
			false,
		},
		{
			"nil nodepools machine image name",
			&ske.Cluster{
				Kubernetes: &ske.Kubernetes{
					Version: utils.Ptr("v1.0.0"),
				},
				Nodepools: &[]ske.Nodepool{
					{
						Name: utils.Ptr("foo"),
						Machine: &ske.Machine{
							Image: &ske.Image{
								Name: nil,
							},
						},
					},
				},
			},
			utils.Ptr("v1.0.0"),
			map[string]*ske.Image{},
			false,
		},
		{
			"nil nodepools machine image version",
			&ske.Cluster{
				Kubernetes: &ske.Kubernetes{
					Version: utils.Ptr("v1.0.0"),
				},
				Nodepools: &[]ske.Nodepool{
					{
						Name: utils.Ptr("foo"),
						Machine: &ske.Machine{
							Image: &ske.Image{
								Name:    utils.Ptr("foo"),
								Version: nil,
							},
						},
					},
				},
			},
			utils.Ptr("v1.0.0"),
			map[string]*ske.Image{
				"foo": {
					Name:    utils.Ptr("foo"),
					Version: nil,
				},
			},
			false,
		},
		{
			"nil response",
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &skeClientMocked{
				returnError:    tt.getClusterFails,
				getClusterResp: tt.mockedResp,
			}
			model := &Model{
				ProjectId: types.StringValue("pid"),
				Name:      types.StringValue("name"),
			}
			kubernetesVersion, machineImageVersions := getCurrentVersions(context.Background(), client, model)
			diff := cmp.Diff(kubernetesVersion, tt.expectedKubernetesVersion)
			if diff != "" {
				t.Errorf("Kubernetes version does not match: %s", diff)
			}

			diff = cmp.Diff(machineImageVersions, tt.expectedMachineImages)
			if diff != "" {
				t.Errorf("Machine images do not match: %s", diff)
			}
		})
	}
}

func TestGetLatestSupportedKubernetesVersion(t *testing.T) {
	tests := []struct {
		description           string
		listKubernetesVersion []ske.KubernetesVersion
		isValid               bool
		expectedVersion       *string
	}{
		{
			description: "base",
			listKubernetesVersion: []ske.KubernetesVersion{
				{
					State:   utils.Ptr("supported"),
					Version: utils.Ptr("1.2.3"),
				},
				{
					State:   utils.Ptr("supported"),
					Version: utils.Ptr("3.2.1"),
				},
				{
					State:   utils.Ptr("not-supported"),
					Version: utils.Ptr("4.4.4"),
				},
			},
			isValid:         true,
			expectedVersion: utils.Ptr("3.2.1"),
		},
		{
			description:           "no Kubernetes versions 1",
			listKubernetesVersion: nil,
			isValid:               false,
		},
		{
			description:           "no Kubernetes versions 2",
			listKubernetesVersion: []ske.KubernetesVersion{},
			isValid:               false,
		},
		{
			description: "no supported Kubernetes versions",
			listKubernetesVersion: []ske.KubernetesVersion{
				{
					State:   utils.Ptr("not-supported"),
					Version: utils.Ptr("1.2.3"),
				},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			version, err := getLatestSupportedKubernetesVersion(tt.listKubernetesVersion)

			if tt.isValid && err != nil {
				t.Errorf("failed on valid input")
			}
			if !tt.isValid && err == nil {
				t.Errorf("did not fail on invalid input")
			}
			if !tt.isValid {
				return
			}
			diff := cmp.Diff(version, tt.expectedVersion)
			if diff != "" {
				t.Fatalf("Output is not as expected: %s", diff)
			}
		})
	}
}

func TestGetLatestSupportedMachineVersion(t *testing.T) {
	tests := []struct {
		description        string
		listMachineVersion []ske.MachineImageVersion
		isValid            bool
		expectedVersion    *string
	}{
		{
			description: "base",
			listMachineVersion: []ske.MachineImageVersion{
				{
					State:   utils.Ptr("supported"),
					Version: utils.Ptr("1.2.3"),
				},
				{
					State:   utils.Ptr("supported"),
					Version: utils.Ptr("3.2.1"),
				},
				{
					State:   utils.Ptr("not-supported"),
					Version: utils.Ptr("4.4.4"),
				},
			},
			isValid:         true,
			expectedVersion: utils.Ptr("3.2.1"),
		},
		{
			description:        "no mchine versions 1",
			listMachineVersion: nil,
			isValid:            false,
		},
		{
			description:        "no machine versions 2",
			listMachineVersion: []ske.MachineImageVersion{},
			isValid:            false,
		},
		{
			description: "no supported machine versions",
			listMachineVersion: []ske.MachineImageVersion{
				{
					State:   utils.Ptr("not-supported"),
					Version: utils.Ptr("1.2.3"),
				},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			version, err := getLatestSupportedMachineVersion(tt.listMachineVersion)

			if tt.isValid && err != nil {
				t.Errorf("failed on valid input")
			}
			if !tt.isValid && err == nil {
				t.Errorf("did not fail on invalid input")
			}
			if !tt.isValid {
				return
			}
			diff := cmp.Diff(version, tt.expectedVersion)
			if diff != "" {
				t.Fatalf("Output is not as expected: %s", diff)
			}
		})
	}
}

func TestToNetworkPayload(t *testing.T) {
	tests := []struct {
		description string
		model       *Model
		expected    *ske.Network
		isValid     bool
	}{
		{
			"base",
			&Model{
				ProjectId: types.StringValue("pid"),
				Name:      types.StringValue("name"),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"id": types.StringValue("nid"),
				}),
			},
			&ske.Network{
				Id: utils.Ptr("nid"),
			},
			true,
		},
		{
			"no_id",
			&Model{
				ProjectId: types.StringValue("pid"),
				Name:      types.StringValue("name"),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"id": types.StringNull(),
				}),
			},
			&ske.Network{},
			true,
		},
		{
			"no_network",
			&Model{
				ProjectId: types.StringValue("pid"),
				Name:      types.StringValue("name"),
				Network:   types.ObjectNull(networkTypes),
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toNetworkPayload(context.Background(), tt.model)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid {
				diff := cmp.Diff(payload, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestVerifySystemComponentNodepools(t *testing.T) {
	tests := []struct {
		description string
		nodePools   []ske.Nodepool
		isValid     bool
	}{
		{
			description: "all pools allow system components",
			nodePools: []ske.Nodepool{
				{
					AllowSystemComponents: conversion.BoolValueToPointer(basetypes.NewBoolValue(true)),
				},
				{
					AllowSystemComponents: conversion.BoolValueToPointer(basetypes.NewBoolValue(true)),
				},
			},
			isValid: true,
		},
		{
			description: "one pool allows system components",
			nodePools: []ske.Nodepool{
				{
					AllowSystemComponents: conversion.BoolValueToPointer(basetypes.NewBoolValue(true)),
				},
				{
					AllowSystemComponents: conversion.BoolValueToPointer(basetypes.NewBoolValue(false)),
				},
			},
			isValid: true,
		},
		{
			description: "no pool allows system components",
			nodePools: []ske.Nodepool{
				{
					AllowSystemComponents: conversion.BoolValueToPointer(basetypes.NewBoolValue(false)),
				},
				{
					AllowSystemComponents: conversion.BoolValueToPointer(basetypes.NewBoolValue(false)),
				},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := verifySystemComponentsInNodePools(tt.nodePools)
			if (err == nil) != tt.isValid {
				t.Errorf("expected validity to be %v, but got error: %v", tt.isValid, err)
			}
		})
	}
}

func TestMaintenanceWindow(t *testing.T) {
	tc := []struct {
		start     string
		end       string
		wantStart string
		wantEnd   string
	}{
		{"01:00:00Z", "02:00:00Z", "01:00:00", "02:00:00"},
		{"01:00:00+00:00", "02:00:00+00:00", "01:00:00", "02:00:00"},
		{"01:00:00+05:00", "02:00:00+05:00", "01:00:00", "02:00:00"},
		{"01:00:00-05:00", "02:00:00-05:00", "01:00:00", "02:00:00"},
	}
	for _, tt := range tc {
		t.Run(fmt.Sprintf("from %s to %s", tt.start, tt.end), func(t *testing.T) {
			attributeTypes := map[string]attr.Type{
				"start":                                types.StringType,
				"end":                                  types.StringType,
				"enable_kubernetes_version_updates":    types.BoolType,
				"enable_machine_image_version_updates": types.BoolType,
			}

			attributeValues := map[string]attr.Value{
				"start":                                basetypes.NewStringValue(tt.start),
				"end":                                  basetypes.NewStringValue(tt.end),
				"enable_kubernetes_version_updates":    basetypes.NewBoolValue(false),
				"enable_machine_image_version_updates": basetypes.NewBoolValue(false),
			}

			val, diags := basetypes.NewObjectValue(attributeTypes, attributeValues)
			if diags.HasError() {
				t.Fatalf("cannot create object value: %v", diags)
			}
			model := Model{
				Maintenance: val,
			}
			maintenance, err := toMaintenancePayload(context.Background(), &model)
			if err != nil {
				t.Fatalf("cannot create payload: %v", err)
			}

			startLocation := maintenance.TimeWindow.Start.Location()
			endLocation := maintenance.TimeWindow.End.Location()
			wantStart, err := time.ParseInLocation(time.TimeOnly, tt.wantStart, startLocation)
			if err != nil {
				t.Fatalf("cannot parse start date %q: %v", tt.wantStart, err)
			}
			wantEnd, err := time.ParseInLocation(time.TimeOnly, tt.wantEnd, endLocation)
			if err != nil {
				t.Fatalf("cannot parse end date %q: %v", tt.wantEnd, err)
			}

			if expected, actual := wantStart.In(startLocation), *maintenance.TimeWindow.Start; expected != actual {
				t.Errorf("invalid start date. expected %s but got %s", expected, actual)
			}
			if expected, actual := wantEnd.In(endLocation), (*maintenance.TimeWindow.End); expected != actual {
				t.Errorf("invalid End date. expected %s but got %s", expected, actual)
			}
		})
	}
}

func TestSortK8sVersion(t *testing.T) {
	testcases := []struct {
		description string
		versions    []ske.KubernetesVersion
		wantSorted  []ske.KubernetesVersion
	}{
		{
			description: "slice with well formed elements",
			versions: []ske.KubernetesVersion{
				{Version: utils.Ptr("v1.2.3")},
				{Version: utils.Ptr("v1.1.10")},
				{Version: utils.Ptr("v1.2.1")},
				{Version: utils.Ptr("v1.2.0")},
				{Version: utils.Ptr("v1.1")},
				{Version: utils.Ptr("v1.2.2")},
			},
			wantSorted: []ske.KubernetesVersion{
				{Version: utils.Ptr("v1.2.3")},
				{Version: utils.Ptr("v1.2.2")},
				{Version: utils.Ptr("v1.2.1")},
				{Version: utils.Ptr("v1.2.0")},
				{Version: utils.Ptr("v1.1.10")},
				{Version: utils.Ptr("v1.1")},
			},
		},
		{
			description: "slice with undefined elements",
			versions: []ske.KubernetesVersion{
				{Version: utils.Ptr("v1.2.3")},
				{Version: utils.Ptr("v1.1.10")},
				{},
				{Version: utils.Ptr("v1.2.0")},
				{Version: utils.Ptr("v1.1")},
				{Version: utils.Ptr("v1.2.2")},
			},
			wantSorted: []ske.KubernetesVersion{
				{Version: utils.Ptr("v1.2.3")},
				{Version: utils.Ptr("v1.2.2")},
				{Version: utils.Ptr("v1.2.0")},
				{Version: utils.Ptr("v1.1.10")},
				{Version: utils.Ptr("v1.1")},
				{Version: nil},
			},
		},
		{
			description: "slice without prefix and minor version change",
			versions: []ske.KubernetesVersion{
				{Version: utils.Ptr("1.20.0")},
				{Version: utils.Ptr("1.19.0")},
				{Version: utils.Ptr("1.20.1")},
				{Version: utils.Ptr("1.20.2")},
			},
			wantSorted: []ske.KubernetesVersion{
				{Version: utils.Ptr("1.20.2")},
				{Version: utils.Ptr("1.20.1")},
				{Version: utils.Ptr("1.20.0")},
				{Version: utils.Ptr("1.19.0")},
			},
		},
		{
			description: "empty slice",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			sortK8sVersions(tc.versions)

			joinK8sVersions := func(in []ske.KubernetesVersion, sep string) string {
				var builder strings.Builder
				for i, l := 0, len(in); i < l; i++ {
					if i > 0 {
						builder.WriteString(sep)
					}
					if v := in[i].Version; v != nil {
						builder.WriteString(*v)
					} else {
						builder.WriteString("undef")
					}
				}
				return builder.String()
			}

			expected := joinK8sVersions(tc.wantSorted, ", ")
			actual := joinK8sVersions(tc.versions, ", ")

			if expected != actual {
				t.Errorf("wrong sort order. wanted %s but got %s", expected, actual)
			}
		})
	}
}
