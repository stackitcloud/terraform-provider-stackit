package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

const (
	userData              = "user_data"
	base64EncodedUserData = "dXNlcl9kYXRh"
	testTimestampValue    = "2006-01-02T15:04:05Z"
)

func testTimestamp() time.Time {
	timestamp, _ := time.Parse(time.RFC3339, testTimestampValue)
	return timestamp
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaasalpha.Server
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId: types.StringValue("pid"),
				ServerId:  types.StringValue("sid"),
			},
			&iaasalpha.Server{
				Id: utils.Ptr("sid"),
			},
			Model{
				Id:               types.StringValue("pid,sid"),
				ProjectId:        types.StringValue("pid"),
				ServerId:         types.StringValue("sid"),
				Name:             types.StringNull(),
				AvailabilityZone: types.StringNull(),
				Labels:           types.MapNull(types.StringType),
				ImageId:          types.StringNull(),
				KeypairName:      types.StringNull(),
				ServerGroup:      types.StringNull(),
				UserData:         types.StringNull(),
				CreatedAt:        types.StringNull(),
				UpdatedAt:        types.StringNull(),
				LaunchedAt:       types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			Model{
				ProjectId: types.StringValue("pid"),
				ServerId:  types.StringValue("sid"),
			},
			&iaasalpha.Server{
				Id:               utils.Ptr("sid"),
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Image:       utils.Ptr("image_id"),
				Keypair:     utils.Ptr("keypair_name"),
				ServerGroup: utils.Ptr("group_id"),
				CreatedAt:   utils.Ptr(testTimestamp()),
				UpdatedAt:   utils.Ptr(testTimestamp()),
				LaunchedAt:  utils.Ptr(testTimestamp()),
			},
			Model{
				Id:               types.StringValue("pid,sid"),
				ProjectId:        types.StringValue("pid"),
				ServerId:         types.StringValue("sid"),
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				ImageId:     types.StringValue("image_id"),
				KeypairName: types.StringValue("keypair_name"),
				ServerGroup: types.StringValue("group_id"),
				CreatedAt:   types.StringValue(testTimestampValue),
				UpdatedAt:   types.StringValue(testTimestampValue),
				LaunchedAt:  types.StringValue(testTimestampValue),
			},
			true,
		},
		{
			"empty_labels",
			Model{
				ProjectId: types.StringValue("pid"),
				ServerId:  types.StringValue("sid"),
				Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			&iaasalpha.Server{
				Id: utils.Ptr("sid"),
			},
			Model{
				Id:               types.StringValue("pid,sid"),
				ProjectId:        types.StringValue("pid"),
				ServerId:         types.StringValue("sid"),
				Name:             types.StringNull(),
				AvailabilityZone: types.StringNull(),
				Labels:           types.MapValueMust(types.StringType, map[string]attr.Value{}),
				ImageId:          types.StringNull(),
				KeypairName:      types.StringNull(),
				ServerGroup:      types.StringNull(),
				UserData:         types.StringNull(),
				CreatedAt:        types.StringNull(),
				UpdatedAt:        types.StringNull(),
				LaunchedAt:       types.StringNull(),
			},
			true,
		},
		{
			"response_nil_fail",
			Model{},
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				ProjectId: types.StringValue("pid"),
			},
			&iaasalpha.Server{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *iaasalpha.CreateServerPayload
		isValid     bool
	}{
		{
			"create_with_network",
			&Model{
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				BootVolume: types.ObjectValueMust(bootVolumeTypes, map[string]attr.Value{
					"performance_class": types.StringValue("class"),
					"size":              types.Int64Value(1),
					"source_type":       types.StringValue("type"),
					"source_id":         types.StringValue("id"),
				}),
				InitialNetworking: types.ObjectValueMust(initialNetworkTypes, map[string]attr.Value{
					"network_id": types.StringValue("nid"),
					"security_groups": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("group1"),
						types.StringValue("group2"),
					}),
					"network_interface_ids": types.ListNull(types.StringType),
				}),
				ImageId:     types.StringValue("image"),
				KeypairName: types.StringValue("keypair"),
				MachineType: types.StringValue("machine_type"),
				UserData:    types.StringValue(userData),
			},
			&iaasalpha.CreateServerPayload{
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Networking: &iaasalpha.CreateServerPayloadNetworking{
					CreateServerNetworking: &iaasalpha.CreateServerNetworking{
						NetworkId: utils.Ptr("nid"),
					},
				},
				SecurityGroups: utils.Ptr([]string{"group1", "group2"}),
				BootVolume: &iaasalpha.CreateServerPayloadBootVolume{
					PerformanceClass: utils.Ptr("class"),
					Size:             utils.Ptr(int64(1)),
					Source: &iaasalpha.BootVolumeSource{
						Type: utils.Ptr("type"),
						Id:   utils.Ptr("id"),
					},
				},
				Image:       utils.Ptr("image"),
				Keypair:     utils.Ptr("keypair"),
				MachineType: utils.Ptr("machine_type"),
				UserData:    utils.Ptr(base64EncodedUserData),
			},
			true,
		},
		{
			"create_with_network_interface_ids",
			&Model{
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				InitialNetworking: types.ObjectValueMust(initialNetworkTypes, map[string]attr.Value{
					"network_id":      types.StringNull(),
					"security_groups": types.ListNull(types.StringType),
					"network_interface_ids": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("nic1"),
						types.StringValue("nic2"),
					}),
				}),
				ImageId:     types.StringValue("image"),
				KeypairName: types.StringValue("keypair"),
				MachineType: types.StringValue("machine_type"),
				UserData:    types.StringValue(userData),
			},
			&iaasalpha.CreateServerPayload{
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Networking: &iaasalpha.CreateServerPayloadNetworking{
					CreateServerNetworkingWithNics: &iaasalpha.CreateServerNetworkingWithNics{
						NicIds: utils.Ptr([]string{"nic1", "nic2"}),
					},
				},
				Image:       utils.Ptr("image"),
				Keypair:     utils.Ptr("keypair"),
				MachineType: utils.Ptr("machine_type"),
				UserData:    utils.Ptr(base64EncodedUserData),
			},
			true,
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

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *iaasalpha.V1alpha1UpdateServerPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaasalpha.V1alpha1UpdateServerPayload{
				Name: utils.Ptr("name"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, types.MapNull(types.StringType))
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
