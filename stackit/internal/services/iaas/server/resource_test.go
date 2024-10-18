package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
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
		input       *iaas.Server
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId: types.StringValue("pid"),
				ServerId:  types.StringValue("sid"),
			},
			&iaas.Server{
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
				AffinityGroup:    types.StringNull(),
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
			&iaas.Server{
				Id:               utils.Ptr("sid"),
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				ImageId:     utils.Ptr("image_id"),
				KeypairName: utils.Ptr("keypair_name"),
				AffinityGroup: utils.Ptr("group_id"),
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
				ImageId:       types.StringValue("image_id"),
				KeypairName:   types.StringValue("keypair_name"),
				AffinityGroup: types.StringValue("group_id"),
				CreatedAt:     types.StringValue(testTimestampValue),
				UpdatedAt:     types.StringValue(testTimestampValue),
				LaunchedAt:    types.StringValue(testTimestampValue),
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
			&iaas.Server{
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
				AffinityGroup:    types.StringNull(),
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
			&iaas.Server{},
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
		expected    *iaas.CreateServerPayload
		isValid     bool
	}{
		{
			"ok",
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
				ImageId:     types.StringValue("image"),
				KeypairName: types.StringValue("keypair"),
				MachineType: types.StringValue("machine_type"),
				UserData:    types.StringValue(userData),
			},
			&iaas.CreateServerPayload{
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				BootVolume: &iaas.CreateServerPayloadBootVolume{
					PerformanceClass: utils.Ptr("class"),
					Size:             utils.Ptr(int64(1)),
					Source: &iaas.BootVolumeSource{
						Type: utils.Ptr("type"),
						Id:   utils.Ptr("id"),
					},
				},
				ImageId:     utils.Ptr("image"),
				KeypairName: utils.Ptr("keypair"),
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
		expected    *iaas.UpdateServerPayload
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
			&iaas.UpdateServerPayload{
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
