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
	tests := []struct {
		description string
		state       Model
		input       *iaas.AffinityGroup
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId:       types.StringValue("pid"),
				AffinityGroupId: types.StringValue("aid"),
			},
			&iaas.AffinityGroup{
				Id: utils.Ptr("aid"),
			},
			Model{
				Id:              types.StringValue("pid,aid"),
				ProjectId:       types.StringValue("pid"),
				AffinityGroupId: types.StringValue("aid"),
				Name:            types.StringNull(),
				Policy:          types.StringNull(),
				Members:         types.ListNull(types.StringType),
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
			"no_affinity_group_id",
			Model{
				ProjectId: types.StringValue("pid"),
			},
			&iaas.AffinityGroup{},
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
				t.Fatalf("Should not have failed")
			}
			if tt.isValid {
				diff := cmp.Diff(tt.state, tt.expected)
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
