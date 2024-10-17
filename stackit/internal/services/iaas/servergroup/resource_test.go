package servergroup

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaasalpha.ServerGroup
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId:     types.StringValue("pid"),
				ServerGroupId: types.StringValue("nid"),
			},
			&iaasalpha.ServerGroup{
				Id: utils.Ptr("nid"),
			},
			Model{
				Id:            types.StringValue("pid,nid"),
				ProjectId:     types.StringValue("pid"),
				ServerGroupId: types.StringValue("nid"),
				Name:          types.StringNull(),
				Policy:        types.StringNull(),
				MemberIds:     types.ListNull(types.StringType),
			},
			true,
		},
		{
			"simple_values",
			Model{
				ProjectId:     types.StringValue("pid"),
				ServerGroupId: types.StringValue("nid"),
			},
			&iaasalpha.ServerGroup{
				Id:     utils.Ptr("nid"),
				Name:   utils.Ptr("name"),
				Policy: utils.Ptr("policy"),
				Members: &[]string{
					"member1",
					"member2",
				},
			},
			Model{
				Id:            types.StringValue("pid,nid"),
				ProjectId:     types.StringValue("pid"),
				ServerGroupId: types.StringValue("nid"),
				Name:          types.StringValue("name"),
				Policy:        types.StringValue("policy"),
				MemberIds: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("member1"),
					types.StringValue("member2"),
				}),
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
			&iaasalpha.ServerGroup{},
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
		expected    *iaasalpha.CreateServerGroupPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name:   types.StringValue("name"),
				Policy: types.StringValue("policy"),
			},
			&iaasalpha.CreateServerGroupPayload{
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
