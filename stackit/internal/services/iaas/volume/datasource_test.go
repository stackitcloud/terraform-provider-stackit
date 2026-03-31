package volume

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapDatasourceFields(t *testing.T) {
	type args struct {
		state  DatasourceModel
		input  *iaas.Volume
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    DatasourceModel
		isValid     bool
	}{
		{
			description: "default_values",
			args: args{
				state: DatasourceModel{
					ProjectId: types.StringValue("pid"),
					VolumeId:  types.StringValue("nid"),
				},
				input: &iaas.Volume{
					Id:                   new("nid"),
					EncryptionParameters: nil,
				},
				region: "eu01",
			},
			expected: DatasourceModel{
				Id:               types.StringValue("pid,eu01,nid"),
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
				Region:           types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			args: args{
				state: DatasourceModel{
					ProjectId: types.StringValue("pid"),
					VolumeId:  types.StringValue("nid"),
					Region:    types.StringValue("eu01"),
				},
				input: &iaas.Volume{
					Id:               new("nid"),
					Name:             new("name"),
					AvailabilityZone: new("zone"),
					Labels: &map[string]any{
						"key": "value",
					},
					Description:      new("desc"),
					PerformanceClass: new("class"),
					ServerId:         new("sid"),
					Size:             new(int64(1)),
					Source:           &iaas.VolumeSource{},
					Encrypted:        new(true),
					EncryptionParameters: &iaas.VolumeEncryptionParameter{
						KekKeyId:       new("kek-key-id"),
						KekKeyVersion:  new(int64(1)),
						KekKeyringId:   new("kek-keyring-id"),
						KekProjectId:   new("kek-project-id"),
						KeyPayload:     nil,
						ServiceAccount: new("test-sa@sa.stackit.cloud"),
					},
				},
				region: "eu02",
			},
			expected: DatasourceModel{
				Id:               types.StringValue("pid,eu02,nid"),
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
				Region:    types.StringValue("eu02"),
				Encrypted: types.BoolValue(true),
			},
			isValid: true,
		},
		{
			description: "empty labels and encryption parameters",
			args: args{
				state: DatasourceModel{
					ProjectId: types.StringValue("pid"),
					VolumeId:  types.StringValue("nid"),
					Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
				},
				input: &iaas.Volume{
					Id:                   new("nid"),
					EncryptionParameters: &iaas.VolumeEncryptionParameter{},
				},
				region: "eu01",
			},
			expected: DatasourceModel{
				Id:               types.StringValue("pid,eu01,nid"),
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
				Region:           types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "response_nil_fail",
		},
		{
			description: "no_resource_id",
			args: args{
				state: DatasourceModel{
					ProjectId: types.StringValue("pid"),
				},
				input: &iaas.Volume{},
			},
			expected: DatasourceModel{},
			isValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDatasourceFields(context.Background(), tt.args.input, &tt.args.state, tt.args.region)
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
