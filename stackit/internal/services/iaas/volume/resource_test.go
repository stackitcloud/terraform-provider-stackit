package volume

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaas.Volume
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId: types.StringValue("pid"),
				VolumeId:  types.StringValue("nid"),
			},
			&iaas.Volume{
				Id: utils.Ptr("nid"),
			},
			Model{
				Id:               types.StringValue("pid,nid"),
				ProjectId:        types.StringValue("pid"),
				VolumeId:         types.StringValue("nid"),
				Name:             types.StringNull(),
				AvailabilityZone: types.StringNull(),
				Labels:           types.MapNull(types.StringType),
				Description:      types.StringNull(),
				PerformanceClass: types.StringNull(),
				ServerId:         types.StringNull(),
				Size:             types.Int64Null(),
				Source:           types.ObjectNull(sourceTypes),
			},
			true,
		},
		{
			"simple_values",
			Model{
				ProjectId: types.StringValue("pid"),
				VolumeId:  types.StringValue("nid"),
			},
			&iaas.Volume{
				Id:               utils.Ptr("nid"),
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Description:      utils.Ptr("desc"),
				PerformanceClass: utils.Ptr("class"),
				ServerId:         utils.Ptr("sid"),
				Size:             utils.Ptr(int64(1)),
				Source:           &iaas.VolumeSource{},
			},
			Model{
				Id:               types.StringValue("pid,nid"),
				ProjectId:        types.StringValue("pid"),
				VolumeId:         types.StringValue("nid"),
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description:      types.StringValue("desc"),
				PerformanceClass: types.StringValue("class"),
				ServerId:         types.StringValue("sid"),
				Size:             types.Int64Value(1),
				Source: types.ObjectValueMust(sourceTypes, map[string]attr.Value{
					"type": types.StringNull(),
					"id":   types.StringNull(),
				}),
			},
			true,
		},
		{
			"empty_labels",
			Model{
				ProjectId: types.StringValue("pid"),
				VolumeId:  types.StringValue("nid"),
				Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			&iaas.Volume{
				Id: utils.Ptr("nid"),
			},
			Model{
				Id:               types.StringValue("pid,nid"),
				ProjectId:        types.StringValue("pid"),
				VolumeId:         types.StringValue("nid"),
				Name:             types.StringNull(),
				AvailabilityZone: types.StringNull(),
				Labels:           types.MapValueMust(types.StringType, map[string]attr.Value{}),
				Description:      types.StringNull(),
				PerformanceClass: types.StringNull(),
				ServerId:         types.StringNull(),
				Size:             types.Int64Null(),
				Source:           types.ObjectNull(sourceTypes),
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
			&iaas.Volume{},
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
		source      *sourceModel
		expected    *iaas.CreateVolumePayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description:      types.StringValue("desc"),
				PerformanceClass: types.StringValue("class"),
				Size:             types.Int64Value(1),
				Source: types.ObjectValueMust(sourceTypes, map[string]attr.Value{
					"type": types.StringNull(),
					"id":   types.StringNull(),
				}),
			},
			&sourceModel{
				Type: types.StringValue("volume"),
				Id:   types.StringValue("id"),
			},
			&iaas.CreateVolumePayload{
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Description:      utils.Ptr("desc"),
				PerformanceClass: utils.Ptr("class"),
				Size:             utils.Ptr(int64(1)),
				Source: &iaas.VolumeSource{
					Type: utils.Ptr("volume"),
					Id:   utils.Ptr("id"),
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input, tt.source)
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
		expected    *iaas.UpdateVolumePayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				Description: types.StringValue("desc"),
			},
			&iaas.UpdateVolumePayload{
				Name: utils.Ptr("name"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				Description: utils.Ptr("desc"),
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
