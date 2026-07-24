package waf

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	waf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"
)

const (
	testRegion = "eu01"
)

func Test_mapFieds(t *testing.T) {
	fixtureModel := func(mods ...func(*Model)) *Model {
		m := Model{
			Id:                  types.StringValue(fmt.Sprintf("pid,%s,name", testRegion)),
			ProjectId:           types.StringValue("pid"),
			Region:              types.StringValue(testRegion),
			Name:                types.StringValue("name"),
			Labels:              types.MapNull(types.StringType),
			ManagedRuleSetName:  types.StringNull(),
			CustomRuleGroupName: types.StringNull(),
			Usage:               types.ObjectNull(usageType),
		}

		for _, mod := range mods {
			mod(&m)
		}
		return &m
	}
	tests := []struct {
		name    string
		input   *waf.GetWAFResponse
		state   *Model
		region  string
		want    *Model
		wantErr bool
	}{
		{
			name: "default_values",
			input: &waf.GetWAFResponse{
				Name: new("name"),
			},
			state:   fixtureModel(),
			region:  testRegion,
			want:    fixtureModel(),
			wantErr: false,
		},
		{
			name: "simple values",
			input: &waf.GetWAFResponse{
				Name:                new("name"),
				Labels:              &map[string]string{"label1": "value1"},
				ManagedRuleSetName:  new("managed_rule_set"),
				CustomRuleGroupName: new("custom_rule_group"),
			},
			state:  fixtureModel(),
			region: testRegion,
			want: fixtureModel(
				func(m *Model) {
					m.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{
						"label1": types.StringValue("value1"),
					})
					m.ManagedRuleSetName = types.StringValue("managed_rule_set")
					m.CustomRuleGroupName = types.StringValue("custom_rule_group")
				},
			),
			wantErr: false,
		},
		{
			name: "usage values",
			input: &waf.GetWAFResponse{
				Name: new("name"),
				Usage: &waf.WAFUsage{
					Count: new(int32(2)),
					Items: []waf.WAFUsageItem{
						{
							ListenerNames:    []string{"listener1", "listener2"},
							LoadBalancerName: new("name_of_load_balancer"),
						},
						{
							ListenerNames:    []string{"listener1"},
							LoadBalancerName: new("name_of_load_balancer2"),
						},
					},
				},
			},
			state:  fixtureModel(),
			region: testRegion,
			want: fixtureModel(
				func(m *Model) {
					itemsElements := []attr.Value{
						types.ObjectValueMust(itemsType, map[string]attr.Value{
							"listener_names":     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("listener1"), types.StringValue("listener2")}),
							"load_balancer_name": types.StringValue("name_of_load_balancer"),
						}),
						types.ObjectValueMust(itemsType, map[string]attr.Value{
							"listener_names":     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("listener1")}),
							"load_balancer_name": types.StringValue("name_of_load_balancer2"),
						}),
					}
					m.Usage = types.ObjectValueMust(usageType, map[string]attr.Value{
						"count": types.Int32Value(2),
						"items": types.ListValueMust(types.ObjectType{AttrTypes: itemsType}, itemsElements),
					})
				},
			),
			wantErr: false,
		},
		{
			name:    "fails when model is nil",
			state:   nil,
			wantErr: true,
		},
		{
			name:    "fails when input is nil",
			input:   nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := mapFields(t.Context(), tt.input, tt.state, tt.region)

			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("mapFieds() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("mapFieds() succeeded unexpectedly")
			}
			if !tt.wantErr {
				diff := cmp.Diff(tt.state, tt.want)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func Test_toCreatePayload(t *testing.T) {
	tests := []struct {
		name    string
		model   *Model
		want    *waf.CreateWAFPayload
		wantErr bool
	}{
		{
			name: "basic values",
			model: &Model{
				ManagedRuleSetName:  types.StringValue("example"),
				CustomRuleGroupName: types.StringValue("example group name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"label1": types.StringValue("value1"),
				}),
			},
			want: &waf.CreateWAFPayload{
				ManagedRuleSetName:  new("example"),
				CustomRuleGroupName: new("example group name"),
				Labels:              &map[string]string{"label1": "value1"},
			},
			wantErr: false,
		},
		{
			name:    "fails when model is nil",
			model:   nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := toCreatePayload(t.Context(), tt.model)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("toCreatePayload() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("toCreatePayload() succeeded unexpectedly")
			}
			diff := cmp.Diff(got, tt.want)
			if diff != "" {
				t.Errorf("Data does not match: %s", diff)
			}
		})
	}
}
