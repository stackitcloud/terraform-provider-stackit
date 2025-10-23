package affinitygroup

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapFields(t *testing.T) {
	type args struct {
		state  Model
		input  *iaas.AffinityGroup
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    Model
		isValid     bool
	}{
		{
			description: "default_values",
			args: args{
				state: Model{
					ProjectId:       types.StringValue("pid"),
					AffinityGroupId: types.StringValue("aid"),
				},
				input: &iaas.AffinityGroup{
					Id: utils.Ptr("aid"),
				},
				region: "eu01",
			},
			expected: Model{
				Id:              types.StringValue("pid,eu01,aid"),
				ProjectId:       types.StringValue("pid"),
				AffinityGroupId: types.StringValue("aid"),
				Name:            types.StringNull(),
				Policy:          types.StringNull(),
				Members:         types.ListNull(types.StringType),
				Region:          types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "response_nil_fail",
		},
		{
			description: "no_affinity_group_id",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
				},
				input: &iaas.AffinityGroup{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.args.input, &tt.args.state, tt.args.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed")
			}
			if tt.isValid {
				diff := cmp.Diff(tt.args.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %v", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *iaas.CreateAffinityGroupPayload
		isValid     bool
	}{
		{
			"default",
			&Model{
				ProjectId: types.StringValue("pid"),
				Name:      types.StringValue("name"),
				Policy:    types.StringValue("policy"),
			},
			&iaas.CreateAffinityGroupPayload{
				Name:   utils.Ptr("name"),
				Policy: utils.Ptr("policy"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
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
