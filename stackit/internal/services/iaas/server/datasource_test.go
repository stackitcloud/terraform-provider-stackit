package server

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		state       DataSourceModel
		input       *iaas.Server
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
				ServerId:  types.StringValue("sid"),
			},
			&iaas.Server{
				Id: utils.Ptr("sid"),
			},
			DataSourceModel{
				Id:                types.StringValue("pid,sid"),
				ProjectId:         types.StringValue("pid"),
				ServerId:          types.StringValue("sid"),
				Name:              types.StringNull(),
				AvailabilityZone:  types.StringNull(),
				Labels:            types.MapNull(types.StringType),
				ImageId:           types.StringNull(),
				NetworkInterfaces: types.ListNull(types.StringType),
				KeypairName:       types.StringNull(),
				AffinityGroup:     types.StringNull(),
				UserData:          types.StringNull(),
				CreatedAt:         types.StringNull(),
				UpdatedAt:         types.StringNull(),
				LaunchedAt:        types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			DataSourceModel{
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
				ImageId: utils.Ptr("image_id"),
				Nics: &[]iaas.ServerNetwork{
					{
						NicId: utils.Ptr("nic1"),
					},
					{
						NicId: utils.Ptr("nic2"),
					},
				},
				KeypairName:   utils.Ptr("keypair_name"),
				AffinityGroup: utils.Ptr("group_id"),
				CreatedAt:     utils.Ptr(testTimestamp()),
				UpdatedAt:     utils.Ptr(testTimestamp()),
				LaunchedAt:    utils.Ptr(testTimestamp()),
				Status:        utils.Ptr("active"),
			},
			DataSourceModel{
				Id:               types.StringValue("pid,sid"),
				ProjectId:        types.StringValue("pid"),
				ServerId:         types.StringValue("sid"),
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				ImageId: types.StringValue("image_id"),
				NetworkInterfaces: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("nic1"),
					types.StringValue("nic2"),
				}),
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
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
				ServerId:  types.StringValue("sid"),
				Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			&iaas.Server{
				Id: utils.Ptr("sid"),
			},
			DataSourceModel{
				Id:                types.StringValue("pid,sid"),
				ProjectId:         types.StringValue("pid"),
				ServerId:          types.StringValue("sid"),
				Name:              types.StringNull(),
				AvailabilityZone:  types.StringNull(),
				Labels:            types.MapValueMust(types.StringType, map[string]attr.Value{}),
				ImageId:           types.StringNull(),
				NetworkInterfaces: types.ListNull(types.StringType),
				KeypairName:       types.StringNull(),
				AffinityGroup:     types.StringNull(),
				UserData:          types.StringNull(),
				CreatedAt:         types.StringNull(),
				UpdatedAt:         types.StringNull(),
				LaunchedAt:        types.StringNull(),
			},
			true,
		},
		{
			"response_nil_fail",
			DataSourceModel{},
			nil,
			DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			DataSourceModel{
				ProjectId: types.StringValue("pid"),
			},
			&iaas.Server{},
			DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceFields(context.Background(), tt.input, &tt.state)
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
