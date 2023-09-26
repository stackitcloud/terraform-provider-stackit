package ske

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func TestMapFields(t *testing.T) {
	cs := ske.ClusterStatusState("OK")
	tests := []struct {
		description string
		input       *ske.ClusterResponse
		expected    Cluster
		isValid     bool
	}{
		{
			"default_values",
			&ske.ClusterResponse{
				Name: utils.Ptr("name"),
			},
			Cluster{
				Id:                        types.StringValue("pid,name"),
				ProjectId:                 types.StringValue("pid"),
				Name:                      types.StringValue("name"),
				KubernetesVersion:         types.StringNull(),
				AllowPrivilegedContainers: types.BoolNull(),
				NodePools:                 []NodePool{},
				Maintenance:               types.ObjectNull(map[string]attr.Type{}),
				Hibernations:              nil,
				Extensions:                nil,
				KubeConfig:                types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&ske.ClusterResponse{
				Extensions: &ske.Extension{
					Acl: &ske.ACL{
						AllowedCidrs: &[]string{"cidr1"},
						Enabled:      utils.Ptr(true),
					},
					Argus: &ske.Argus{
						ArgusInstanceId: utils.Ptr("aid"),
						Enabled:         utils.Ptr(true),
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
					AllowPrivilegedContainers: utils.Ptr(true),
					Version:                   utils.Ptr("1.2.3"),
				},
				Maintenance: &ske.Maintenance{
					AutoUpdate: &ske.MaintenanceAutoUpdate{
						KubernetesVersion:   utils.Ptr(true),
						MachineImageVersion: utils.Ptr(true),
					},
					TimeWindow: &ske.TimeWindow{
						Start: utils.Ptr("0000-01-02T03:04:05+06:00"),
						End:   utils.Ptr("0010-11-12T13:14:15Z"),
					},
				},
				Name: utils.Ptr("name"),
				Nodepools: &[]ske.Nodepool{
					{
						AvailabilityZones: &[]string{"z1", "z2"},
						Cri: &ske.CRI{
							Name: utils.Ptr("cri"),
						},
						Labels: &map[string]string{"k": "v"},
						Machine: &ske.Machine{
							Image: &ske.Image{
								Name:    utils.Ptr("os"),
								Version: utils.Ptr("os-ver"),
							},
							Type: utils.Ptr("B"),
						},
						MaxSurge:       utils.Ptr(int32(3)),
						MaxUnavailable: nil,
						Maximum:        utils.Ptr(int32(5)),
						Minimum:        utils.Ptr(int32(1)),
						Name:           utils.Ptr("node"),
						Taints: &[]ske.Taint{
							{
								Effect: utils.Ptr("effect"),
								Key:    utils.Ptr("key"),
								Value:  utils.Ptr("value"),
							},
						},
						Volume: &ske.Volume{
							Size: utils.Ptr(int32(3)),
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
			Cluster{
				Id:                        types.StringValue("pid,name"),
				ProjectId:                 types.StringValue("pid"),
				Name:                      types.StringValue("name"),
				KubernetesVersion:         types.StringValue("1.2"),
				KubernetesVersionUsed:     types.StringValue("1.2.3"),
				AllowPrivilegedContainers: types.BoolValue(true),

				NodePools: []NodePool{
					{
						Name:           types.StringValue("node"),
						MachineType:    types.StringValue("B"),
						OSName:         types.StringValue("os"),
						OSVersion:      types.StringValue("os-ver"),
						Minimum:        types.Int64Value(1),
						Maximum:        types.Int64Value(5),
						MaxSurge:       types.Int64Value(3),
						MaxUnavailable: types.Int64Null(),
						VolumeType:     types.StringValue("type"),
						VolumeSize:     types.Int64Value(3),
						Labels:         types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")}),
						Taints: []Taint{
							{
								Effect: types.StringValue("effect"),
								Key:    types.StringValue("key"),
								Value:  types.StringValue("value"),
							},
						},
						CRI:               types.StringValue("cri"),
						AvailabilityZones: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("z1"), types.StringValue("z2")}),
					},
				},
				Maintenance: types.ObjectValueMust(maintenanceTypes, map[string]attr.Value{
					"enable_kubernetes_version_updates":    types.BoolValue(true),
					"enable_machine_image_version_updates": types.BoolValue(true),
					"start":                                types.StringValue("03:04:05+06:00"),
					"end":                                  types.StringValue("13:14:15Z"),
				}),
				Hibernations: []Hibernation{
					{
						Start:    types.StringValue("1"),
						End:      types.StringValue("2"),
						Timezone: types.StringValue("CET"),
					},
				},
				Extensions: &Extensions{
					Argus: &ArgusExtension{
						Enabled:         types.BoolValue(true),
						ArgusInstanceId: types.StringValue("aid"),
					},
					ACL: &ACL{
						Enabled: types.BoolValue(true),
						AllowedCIDRs: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("cidr1"),
						}),
					},
				},
				KubeConfig: types.StringNull(),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Cluster{},
			false,
		},
		{
			"no_resource_id",
			&ske.ClusterResponse{},
			Cluster{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Cluster{
				ProjectId: tt.expected.ProjectId,
			}
			err := mapFields(context.Background(), tt.input, state)
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

func TestLatestMatchingVersion(t *testing.T) {
	tests := []struct {
		description                  string
		availableVersions            []ske.KubernetesVersion
		providedVersion              *string
		expectedVersionUsed          *string
		expectedHasDeprecatedVersion bool
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
			utils.Ptr("1.20"),
			utils.Ptr("1.20.2"),
			false,
			true,
		},
		{
			"available_version_no_patch",
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
			utils.Ptr("1.19.0"),
			true,
			true,
		},
		{
			"deprecated_version_not_selected",
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
			utils.Ptr("1.20"),
			utils.Ptr("1.20.0"),
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
			utils.Ptr("1.20"),
			utils.Ptr("1.20.0"),
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
			false,
			false,
		},
		{
			"no_available_version",
			[]ske.KubernetesVersion{},
			utils.Ptr("1.20"),
			nil,
			false,
			false,
		},
		{
			"nil_available_version",
			nil,
			utils.Ptr("1.20"),
			nil,
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
			false,
			false,
		},
		{
			"nil_provided_version",
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
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			versionUsed, hasDeprecatedVersion, err := latestMatchingVersion(tt.availableVersions, tt.providedVersion)
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
		startAPI      string
		startTF       *string
		endAPI        string
		endTF         *string
		isValid       bool
		startExpected string
		endExpected   string
	}{
		{
			description:   "base",
			startAPI:      "0001-02-03T04:05:06+07:08",
			endAPI:        "0011-12-13T14:15:16+17:18",
			isValid:       true,
			startExpected: "04:05:06+07:08",
			endExpected:   "14:15:16+17:18",
		},
		{
			description:   "base_utc",
			startAPI:      "0001-02-03T04:05:06Z",
			endAPI:        "0011-12-13T14:15:16Z",
			isValid:       true,
			startExpected: "04:05:06Z",
			endExpected:   "14:15:16Z",
		},
		{
			description: "api_wrong_format_1",
			startAPI:    "T04:05:06+07:08",
			endAPI:      "0011-12-13T14:15:16+17:18",
			isValid:     false,
		},
		{
			description: "api_wrong_format_2",
			startAPI:    "0001-02-03T04:05:06+07:08",
			endAPI:      "14:15:16+17:18",
			isValid:     false,
		},
		{
			description:   "tf_state_filled_in_1",
			startAPI:      "0001-02-03T04:05:06+07:08",
			startTF:       utils.Ptr("04:05:06+07:08"),
			endAPI:        "0011-12-13T14:15:16+17:18",
			endTF:         utils.Ptr("14:15:16+17:18"),
			isValid:       true,
			startExpected: "04:05:06+07:08",
			endExpected:   "14:15:16+17:18",
		},
		{
			description:   "tf_state_filled_in_2",
			startAPI:      "0001-02-03T04:05:06Z",
			startTF:       utils.Ptr("04:05:06+00:00"),
			endAPI:        "0011-12-13T14:15:16Z",
			endTF:         utils.Ptr("14:15:16+00:00"),
			isValid:       true,
			startExpected: "04:05:06+00:00",
			endExpected:   "14:15:16+00:00",
		},
		{
			description:   "tf_state_filled_in_3",
			startAPI:      "0001-02-03T04:05:06+00:00",
			startTF:       utils.Ptr("04:05:06Z"),
			endAPI:        "0011-12-13T14:15:16+00:00",
			endTF:         utils.Ptr("14:15:16Z"),
			isValid:       true,
			startExpected: "04:05:06Z",
			endExpected:   "14:15:16Z",
		},
		{
			description: "tf_state_doesnt_match_1",
			startAPI:    "0001-02-03T04:05:06+07:08",
			startTF:     utils.Ptr("00:00:00+07:08"),
			endAPI:      "0011-12-13T14:15:16+17:18",
			endTF:       utils.Ptr("14:15:16+17:18"),
			isValid:     false,
		},
		{
			description: "tf_state_doesnt_match_2",
			startAPI:    "0001-02-03T04:05:06+07:08",
			startTF:     utils.Ptr("04:05:06+07:08"),
			endAPI:      "0011-12-13T14:15:16+17:18",
			endTF:       utils.Ptr("00:00:00+17:18"),
			isValid:     false,
		},
		{
			description: "tf_state_doesnt_match_3",
			startAPI:    "0001-02-03T04:05:06+07:08",
			startTF:     utils.Ptr("04:05:06Z"),
			endAPI:      "0011-12-13T14:15:16+17:18",
			endTF:       utils.Ptr("14:15:16+17:18"),
			isValid:     false,
		},
		{
			description: "tf_state_doesnt_match_4",
			startAPI:    "0001-02-03T04:05:06+07:08",
			startTF:     utils.Ptr("04:05:06+07:08"),
			endAPI:      "0011-12-13T14:15:16+17:18",
			endTF:       utils.Ptr("14:15:16Z"),
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			apiResponse := &ske.ClusterResponse{
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
			tfState := &Cluster{
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
				t.Errorf("extected start '%s', got '%s'", tt.startExpected, start)
			}
			if tt.endExpected != end {
				t.Errorf("extected end '%s', got '%s'", tt.endExpected, end)
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
			description:              "null_version_1",
			kubernetesVersion:        nil,
			allowPrivilegeContainers: nil,
			isValid:                  false,
		},
		{
			description:              "null_version_2",
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
