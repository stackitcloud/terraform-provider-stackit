package server

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapDataSourceFields(t *testing.T) {
	type args struct {
		state  DataSourceModel
		input  *iaas.Server
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    DataSourceModel
		isValid     bool
	}{
		{
			description: "default_values",
			args: args{
				state: DataSourceModel{
					ProjectId: types.StringValue("pid"),
					ServerId:  types.StringValue("sid"),
				},
				input: &iaas.Server{
					Id: new("sid"),
				},
				region: "eu01",
			},
			expected: DataSourceModel{
				Id:                types.StringValue("pid,eu01,sid"),
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
				Region:            types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			args: args{
				state: DataSourceModel{
					ProjectId: types.StringValue("pid"),
					ServerId:  types.StringValue("sid"),
					Region:    types.StringValue("eu01"),
				},
				input: &iaas.Server{
					Id:               new("sid"),
					Name:             new("name"),
					AvailabilityZone: new("zone"),
					Labels: &map[string]any{
						"key": "value",
					},
					ImageId: new("image_id"),
					Nics: &[]iaas.ServerNetwork{
						{
							NicId: new("nic1"),
						},
						{
							NicId: new("nic2"),
						},
					},
					KeypairName:   new("keypair_name"),
					AffinityGroup: new("group_id"),
					CreatedAt:     new(testTimestamp()),
					UpdatedAt:     new(testTimestamp()),
					LaunchedAt:    new(testTimestamp()),
					Status:        new("active"),
				},
				region: "eu02",
			},
			expected: DataSourceModel{
				Id:               types.StringValue("pid,eu02,sid"),
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
				Region:        types.StringValue("eu02"),
			},
			isValid: true,
		},
		{
			description: "empty_labels",
			args: args{
				state: DataSourceModel{
					ProjectId: types.StringValue("pid"),
					ServerId:  types.StringValue("sid"),
					Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
				},
				input: &iaas.Server{
					Id: new("sid"),
				},
				region: "eu01",
			},
			expected: DataSourceModel{
				Id:                types.StringValue("pid,eu01,sid"),
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
				Region:            types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "response_nil_fail",
		},
		{
			description: "no_resource_id",
			args: args{
				state: DataSourceModel{
					ProjectId: types.StringValue("pid"),
				},
				input: &iaas.Server{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceFields(context.Background(), tt.args.input, &tt.args.state, tt.args.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.args.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
